/*
Copyright 2021 The cert-manager Authors.

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

package carotation

import (
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cert-manager/csi-driver-spiffe/test/e2e/framework"
	"github.com/cert-manager/csi-driver-spiffe/test/e2e/util"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	mountPath     = "/var/run/secrets/my-pod"
	containerName = "my-container"

	// pollInterval is set to 5s here because the updateRetryPeriod set when the driver
	// starts the camanager is 5s. Polling every second is wasteful because updates will
	// only be made every 5 seconds.
	pollInterval = 5 * time.Second
	pollTimeout  = 300 * time.Second
)

var _ = framework.CasesDescribe("CA rotation", func() {
	f := framework.NewDefaultFramework("CA rotation")

	It("should propagate a new root when it changes", func() {
		By("Creating 2 pods in the test namespace with CSI driver")

		serviceAccount := corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: f.Namespace.Name,
			},
		}
		Expect(f.Client().Create(f.Context(), &serviceAccount)).NotTo(HaveOccurred())

		role := rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: f.Namespace.Name,
			},
			Rules: []rbacv1.PolicyRule{{
				Verbs:     []string{"create"},
				APIGroups: []string{"cert-manager.io"},
				Resources: []string{"certificaterequests"},
			}},
		}
		Expect(f.Client().Create(f.Context(), &role)).NotTo(HaveOccurred())

		rolebinding := rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: f.Namespace.Name,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     "test-pod",
			},
			Subjects: []rbacv1.Subject{{
				Kind:      "ServiceAccount",
				Name:      "test-pod",
				Namespace: f.Namespace.Name,
			}},
		}
		Expect(f.Client().Create(f.Context(), &rolebinding)).NotTo(HaveOccurred())

		pod1 := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-1",
				Namespace: f.Namespace.Name,
			},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{{
					Name: "csi-driver-spiffe",
					VolumeSource: corev1.VolumeSource{
						CSI: &corev1.CSIVolumeSource{
							Driver:   "spiffe.csi.cert-manager.io",
							ReadOnly: ptr.To(true),
						},
					},
				}},
				ServiceAccountName: "test-pod",
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
		pod2 := *(pod1.DeepCopy())
		pod2.Name = "test-pod-2"

		Expect(f.Client().Create(f.Context(), &pod1)).NotTo(HaveOccurred())
		Expect(f.Client().Create(f.Context(), &pod2)).NotTo(HaveOccurred())

		By("Waiting for pods to become ready")

		Expect(util.WaitForPodReady(f, &pod1)).NotTo(HaveOccurred())
		Expect(util.WaitForPodReady(f, &pod2)).NotTo(HaveOccurred())

		By("Comparing the CA stored in secret with CA stored in the Secret")
		var caSecret corev1.Secret
		Expect(f.Client().Get(f.Context(), client.ObjectKey{Namespace: f.Config().IssuerSecretNamespace, Name: f.Config().IssuerSecretName}, &caSecret)).NotTo(HaveOccurred())
		caData, ok := caSecret.Data["ca.crt"]
		Expect(ok).To(BeTrue(), "expected 'ca.crt' to be present in Issuer CA Secret")

		tlsData, ok := caSecret.Data["tls.crt"]
		Expect(ok).To(BeTrue(), "expected 'tls.crt' to be present in Issuer CA Secret")

		Expect(caData).To(Equal(tlsData), "invalid test; expected 'ca.crt' to equal 'tls.crt' for Issuer CA Secret (this implies a previous test run wasn't cleaned up correctly)")

		pod1Bundle, err := util.ReadCertFromMountPath(f, mountPath, pod1.Name, containerName)
		Expect(err).NotTo(HaveOccurred())

		pod2Bundle, err := util.ReadCertFromMountPath(f, mountPath, pod2.Name, containerName)
		Expect(err).NotTo(HaveOccurred())

		Expect(string(caData)).To(Equal(string(pod1Bundle.CAPEM)), "expected Issuer CA bundle to equal CA mounted in pod1 file")

		Expect(string(caData)).To(Equal(string(pod2Bundle.CAPEM)), "expected Issuer CA bundle to equal CA mounted in pod2 file")

		By("Updating the CA data in Secret")

		newCAData := append([]byte("# This is a comment\n"), tlsData...)

		caSecret.Data["ca.crt"] = newCAData
		Expect(f.Client().Update(f.Context(), &caSecret)).NotTo(HaveOccurred())

		defer func() {
			caSecret.Data["ca.crt"] = tlsData
			Expect(f.Client().Update(f.Context(), &caSecret)).NotTo(HaveOccurred())
		}()

		By("Waiting for the new CA data to be written to pod volumes")
		for _, podName := range []string{pod1.Name, pod2.Name} {
			Eventually(func() bool {
				newBundle, err := util.ReadCertFromMountPath(f, mountPath, podName, containerName)
				Expect(err).ToNot(HaveOccurred())

				return strings.TrimSpace(string(newBundle.CAPEM)) == strings.TrimSpace(string(newCAData))
			}).WithTimeout(pollTimeout).WithPolling(pollInterval).WithContext(f.Context()).Should(BeTrue(), "expected the CA data to be updated on pod file")
		}

		By("Cleaning up resources")
		Expect(f.Client().Delete(f.Context(), &rolebinding)).NotTo(HaveOccurred())
		Expect(f.Client().Delete(f.Context(), &role)).NotTo(HaveOccurred())
		Expect(f.Client().Delete(f.Context(), &serviceAccount)).NotTo(HaveOccurred())
		Expect(f.Client().Delete(f.Context(), &pod1)).NotTo(HaveOccurred())
		Expect(f.Client().Delete(f.Context(), &pod2)).NotTo(HaveOccurred())
	})
})
