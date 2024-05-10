/*
Copyright 2024 The cert-manager Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package runtimeconfiguration

import (
	"time"

	"github.com/cert-manager/cert-manager/pkg/util/pki"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/cert-manager/csi-driver-spiffe/test/e2e/framework"
	"github.com/cert-manager/csi-driver-spiffe/test/e2e/util"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	mountPath     = "/var/run/secrets/my-pod"
	containerName = "my-container"
)

var _ = framework.CasesDescribe("RuntimeConfiguration", func() {
	f := framework.NewDefaultFramework("RuntimeConfiguration")

	var (
		serviceAccount corev1.ServiceAccount
		role           rbacv1.Role
		rolebinding    rbacv1.RoleBinding
		podTemplate    corev1.Pod
	)

	JustBeforeEach(func() {
		By("Creating test resources")

		serviceAccount = corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Namespace: f.Namespace.Name, GenerateName: "csi-driver-spiffe-e2e-sa-"},
		}
		Expect(f.Client().Create(f.Context(), &serviceAccount)).NotTo(HaveOccurred())

		role = rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "csi-driver-spiffe-e2e-role-",
				Namespace:    f.Namespace.Name,
			},
			Rules: []rbacv1.PolicyRule{{
				Verbs:     []string{"create"},
				APIGroups: []string{"cert-manager.io"},
				Resources: []string{"certificaterequests"},
			}},
		}
		Expect(f.Client().Create(f.Context(), &role)).NotTo(HaveOccurred())

		rolebinding = rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "csi-driver-spiffe-e2e-rolebinding-",
				Namespace:    f.Namespace.Name,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     role.Name,
			},
			Subjects: []rbacv1.Subject{{
				Kind:      "ServiceAccount",
				Name:      serviceAccount.Name,
				Namespace: f.Namespace.Name,
			}},
		}
		Expect(f.Client().Create(f.Context(), &rolebinding)).NotTo(HaveOccurred())

		podTemplate = corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-pod-",
				Namespace:    f.Namespace.Name,
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: serviceAccount.Name,
				Volumes: []corev1.Volume{{
					Name: "csi-driver-spiffe",
					VolumeSource: corev1.VolumeSource{
						CSI: &corev1.CSIVolumeSource{
							Driver:           "spiffe.csi.cert-manager.io",
							ReadOnly:         ptr.To(true),
							VolumeAttributes: map[string]string{},
						},
					},
				}},
				Containers: []corev1.Container{
					{
						Name:            containerName,
						Image:           "docker.io/library/busybox:1.36.1-musl",
						ImagePullPolicy: corev1.PullNever,
						Command:         []string{"sleep", "10000"},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "csi-driver-spiffe",
								MountPath: mountPath,
							},
						},
					},
				},
			},
		}

	})

	JustAfterEach(func() {
		By("Cleaning up test resources")
		Expect(f.Client().Delete(f.Context(), &rolebinding)).NotTo(HaveOccurred())
		Expect(f.Client().Delete(f.Context(), &role)).NotTo(HaveOccurred())
		Expect(f.Client().Delete(f.Context(), &serviceAccount)).NotTo(HaveOccurred())
	})

	It("should succeed with a simple pod and no runtime configuration", func() {
		pod := *podTemplate.DeepCopy()

		Expect(f.Client().Create(f.Context(), &pod)).NotTo(HaveOccurred())
		defer func() {
			Expect(f.Client().Delete(f.Context(), &pod)).NotTo(HaveOccurred())
		}()

		Expect(util.WaitForPodReady(f, &pod)).NotTo(HaveOccurred())

		bundle, err := util.ReadCertFromMountPath(f, mountPath, pod.Name, containerName)
		Expect(err).NotTo(HaveOccurred())

		Expect(bundle.CheckNotEmpty()).NotTo(HaveOccurred())
	})

	It("should succeed with a new issuer configured at runtime and revert when runtime configuration is deleted", func() {
		// podOne should be created with the old issuer, since no ConfigMap has been created yet
		By("Creating a pod before any runtime configuration")
		podOne := *podTemplate.DeepCopy()

		Expect(f.Client().Create(f.Context(), &podOne)).NotTo(HaveOccurred())
		defer func() {
			Expect(f.Client().Delete(f.Context(), &podOne)).NotTo(HaveOccurred())
		}()

		Expect(util.WaitForPodReady(f, &podOne)).NotTo(HaveOccurred())

		By("Checking the pod used the issuer configured on startup")
		cliArgCertBundle, err := util.ReadCertFromSecret(f, f.Config().IssuerSecretName, f.Config().IssuerSecretNamespace)
		Expect(err).NotTo(HaveOccurred())

		cliArgCert, err := pki.DecodeX509CertificateBytes(cliArgCertBundle.CertificatePEM)
		Expect(err).NotTo(HaveOccurred())

		podOneBundle, err := util.ReadCertFromMountPath(f, mountPath, podOne.Name, containerName)
		Expect(err).NotTo(HaveOccurred())

		podOneChain, err := pki.DecodeX509CertificateChainBytes(podOneBundle.CertificatePEM)
		Expect(err).NotTo(HaveOccurred())

		Expect(podOneChain[0].CheckSignatureFrom(cliArgCert)).NotTo(HaveOccurred())

		By("Creating a new issuer to use at runtime")
		issuerRef, newCABundle, cleanup, err := util.CreateNewCAIssuer(f)
		defer func() {
			err := cleanup()
			Expect(err).NotTo(HaveOccurred())
		}()

		Expect(err).NotTo(HaveOccurred())

		newCACert, err := pki.DecodeX509CertificateBytes(newCABundle.CertificatePEM)
		Expect(err).NotTo(HaveOccurred())

		runtimeConfigMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      f.Config().IssuanceConfigMapName,
				Namespace: f.Config().IssuanceConfigMapNamespace,
			},
			Data: map[string]string{
				"issuer-name":  issuerRef.Name,
				"issuer-kind":  issuerRef.Kind,
				"issuer-group": issuerRef.Group,
			},
		}

		By("Creating runtime configuration to point at the new issuer")
		Expect(f.Client().Create(f.Context(), runtimeConfigMap)).NotTo(HaveOccurred())

		configMapNeedsCleanup := true
		defer func() {
			if configMapNeedsCleanup {
				Expect(f.Client().Delete(f.Context(), runtimeConfigMap)).NotTo(HaveOccurred())
			}
		}()

		By("Waiting a little for runtime configuration to propagate")
		time.Sleep(5 * time.Second)

		// now we've created the runtime configuration configmap, a newly created pod should
		// use the new issuer

		By("Creating a second pod after runtime configuration was created")
		podTwo := *podTemplate.DeepCopy()

		Expect(f.Client().Create(f.Context(), &podTwo)).NotTo(HaveOccurred())
		defer func() {
			Expect(f.Client().Delete(f.Context(), &podTwo)).NotTo(HaveOccurred())
		}()

		Expect(util.WaitForPodReady(f, &podTwo)).NotTo(HaveOccurred())

		By("Checking that the second pod used the new issuer")
		podTwoBundle, err := util.ReadCertFromMountPath(f, mountPath, podTwo.Name, containerName)
		Expect(err).NotTo(HaveOccurred())

		podTwoChain, err := pki.DecodeX509CertificateChainBytes(podTwoBundle.CertificatePEM)
		Expect(err).NotTo(HaveOccurred())

		Expect(podTwoChain[0].CheckSignatureFrom(cliArgCert)).To(HaveOccurred())
		Expect(podTwoChain[0].CheckSignatureFrom(newCACert)).NotTo(HaveOccurred())

		By("Deleting the configuration ConfigMap")
		Expect(f.Client().Delete(f.Context(), runtimeConfigMap)).NotTo(HaveOccurred())
		// we explicitly deleted the ConfigMap as part of the test - no need to clean it up any more
		configMapNeedsCleanup = false

		By("Waiting a little for runtime configuration to revert")
		time.Sleep(5 * time.Second)

		By("Creating a third pod after runtime configuration was deleted")
		podThree := *podTemplate.DeepCopy()

		Expect(f.Client().Create(f.Context(), &podThree)).NotTo(HaveOccurred())
		defer func() {
			Expect(f.Client().Delete(f.Context(), &podThree)).NotTo(HaveOccurred())
		}()

		Expect(util.WaitForPodReady(f, &podThree)).NotTo(HaveOccurred())

		By("Checking that the third pod used the original issuer")
		podThreeBundle, err := util.ReadCertFromMountPath(f, mountPath, podThree.Name, containerName)
		Expect(err).NotTo(HaveOccurred())

		podThreeChain, err := pki.DecodeX509CertificateChainBytes(podThreeBundle.CertificatePEM)
		Expect(err).NotTo(HaveOccurred())

		Expect(podThreeChain[0].CheckSignatureFrom(newCACert)).To(HaveOccurred())
		Expect(podThreeChain[0].CheckSignatureFrom(cliArgCert)).NotTo(HaveOccurred())
	})

	It("should succeed with a new issuer configured at runtime and change issuers when configuration is updated", func() {
		// First, fetch the cert for the CLI arg, to check later that it wasn't used to sign any pod certificates
		cliArgCertBundle, err := util.ReadCertFromSecret(f, f.Config().IssuerSecretName, f.Config().IssuerSecretNamespace)
		Expect(err).NotTo(HaveOccurred())

		cliArgCert, err := pki.DecodeX509CertificateBytes(cliArgCertBundle.CertificatePEM)
		Expect(err).NotTo(HaveOccurred())

		By("Creating a new issuer to use at runtime")
		issuerRefOne, newCABundleOne, cleanupIssuerOne, err := util.CreateNewCAIssuer(f)
		defer func() {
			err := cleanupIssuerOne()
			Expect(err).NotTo(HaveOccurred())
		}()

		Expect(err).NotTo(HaveOccurred())

		newCACertOne, err := pki.DecodeX509CertificateBytes(newCABundleOne.CertificatePEM)
		Expect(err).NotTo(HaveOccurred())

		runtimeConfigMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      f.Config().IssuanceConfigMapName,
				Namespace: f.Config().IssuanceConfigMapNamespace,
			},
			Data: map[string]string{
				"issuer-name":  issuerRefOne.Name,
				"issuer-kind":  issuerRefOne.Kind,
				"issuer-group": issuerRefOne.Group,
			},
		}

		By("Creating runtime configuration to point at the new issuer")
		Expect(f.Client().Create(f.Context(), runtimeConfigMap)).NotTo(HaveOccurred())

		defer func() {
			Expect(f.Client().Delete(f.Context(), runtimeConfigMap)).NotTo(HaveOccurred())
		}()

		By("Waiting a little for runtime configuration to propagate")
		time.Sleep(5 * time.Second)

		By("Creating a pod which uses runtime configuration")
		podOne := *podTemplate.DeepCopy()

		Expect(f.Client().Create(f.Context(), &podOne)).NotTo(HaveOccurred())
		defer func() {
			Expect(f.Client().Delete(f.Context(), &podOne)).NotTo(HaveOccurred())
		}()

		Expect(util.WaitForPodReady(f, &podOne)).NotTo(HaveOccurred())

		By("Checking the pod used the new issuer")
		podOneBundle, err := util.ReadCertFromMountPath(f, mountPath, podOne.Name, containerName)
		Expect(err).NotTo(HaveOccurred())

		podOneChain, err := pki.DecodeX509CertificateChainBytes(podOneBundle.CertificatePEM)
		Expect(err).NotTo(HaveOccurred())

		Expect(podOneChain[0].CheckSignatureFrom(cliArgCert)).To(HaveOccurred())
		Expect(podOneChain[0].CheckSignatureFrom(newCACertOne)).NotTo(HaveOccurred())

		By("Creating a second new issuer to use at runtime")
		issuerRefTwo, newCABundleTwo, cleanupIssuerTwo, err := util.CreateNewCAIssuer(f)
		defer func() {
			err := cleanupIssuerTwo()
			Expect(err).NotTo(HaveOccurred())
		}()

		Expect(err).NotTo(HaveOccurred())

		newCACertTwo, err := pki.DecodeX509CertificateBytes(newCABundleTwo.CertificatePEM)
		Expect(err).NotTo(HaveOccurred())

		runtimeConfigMap.Data["issuer-name"] = issuerRefTwo.Name
		runtimeConfigMap.Data["issuer-kind"] = issuerRefTwo.Kind
		runtimeConfigMap.Data["issuer-group"] = issuerRefTwo.Group

		By("Updating runtime configuration to point at the new issuer")
		Expect(f.Client().Update(f.Context(), runtimeConfigMap)).NotTo(HaveOccurred())

		By("Waiting a little for the runtime configuration update to propagate")
		time.Sleep(5 * time.Second)

		By("Creating a second pod after runtime configuration was updated")
		podTwo := *podTemplate.DeepCopy()

		Expect(f.Client().Create(f.Context(), &podTwo)).NotTo(HaveOccurred())
		defer func() {
			Expect(f.Client().Delete(f.Context(), &podTwo)).NotTo(HaveOccurred())
		}()

		Expect(util.WaitForPodReady(f, &podTwo)).NotTo(HaveOccurred())

		By("Checking that the second pod used the new issuer")
		podTwoBundle, err := util.ReadCertFromMountPath(f, mountPath, podTwo.Name, containerName)
		Expect(err).NotTo(HaveOccurred())

		podTwoChain, err := pki.DecodeX509CertificateChainBytes(podTwoBundle.CertificatePEM)
		Expect(err).NotTo(HaveOccurred())

		Expect(podTwoChain[0].CheckSignatureFrom(cliArgCert)).To(HaveOccurred())
		Expect(podTwoChain[0].CheckSignatureFrom(newCACertOne)).To(HaveOccurred())
		Expect(podTwoChain[0].CheckSignatureFrom(newCACertTwo)).NotTo(HaveOccurred())
	})
})
