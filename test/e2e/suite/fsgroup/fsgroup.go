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

package fsgroup

import (
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

var _ = framework.CasesDescribe("FSGroup", func() {
	f := framework.NewDefaultFramework("FSGroup")

	var (
		serviceAccount corev1.ServiceAccount
		role           rbacv1.Role
		rolebinding    rbacv1.RoleBinding

		podTemplate = corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-pod-",
				Namespace:    f.Namespace.Name,
			},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{{
					Name: "csi-driver-spiffe",
					VolumeSource: corev1.VolumeSource{
						CSI: &corev1.CSIVolumeSource{
							Driver:   "spiffe.csi.cert-manager.io",
							ReadOnly: ptr.To(true),
							VolumeAttributes: map[string]string{
								"spiffe.csi.cert-manager.io/fs-group": "1541",
							},
						},
					},
				}},
				SecurityContext: &corev1.PodSecurityContext{
					RunAsUser:  ptr.To(int64(1321)),
					RunAsGroup: ptr.To(int64(1541)),
				},
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
	)

	JustBeforeEach(func() {
		By("Creating test resources")
		serviceAccount = corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-pod-",
				Namespace:    f.Namespace.Name,
			},
		}
		Expect(f.Client().Create(f.Context(), &serviceAccount)).NotTo(HaveOccurred())

		role = rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-pod-",
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
				GenerateName: "test-pod-",
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
	})

	JustAfterEach(func() {
		By("Cleaning up resources")
		Expect(f.Client().Delete(f.Context(), &rolebinding)).NotTo(HaveOccurred())
		Expect(f.Client().Delete(f.Context(), &role)).NotTo(HaveOccurred())
		Expect(f.Client().Delete(f.Context(), &serviceAccount)).NotTo(HaveOccurred())
	})

	It("should mount and be able to read mounted volume when running different group using spiffe.csi.cert-manager.io/fs-group", func() {
		By("Creating pod in the test namespace with CSI driver with different fs-group")
		pod := *podTemplate.DeepCopy()
		pod.Namespace = f.Namespace.Name
		pod.Spec.ServiceAccountName = serviceAccount.Name
		Expect(f.Client().Create(f.Context(), &pod)).NotTo(HaveOccurred())

		By("Waiting for pod to become ready")
		Expect(util.WaitForPodReady(f, &pod)).NotTo(HaveOccurred())

		By("Ensuring files can be read from volume")
		bundle, err := util.ReadCertFromMountPath(f, mountPath, pod.Name, containerName)
		Expect(err).NotTo(HaveOccurred())

		Expect(bundle.CheckNotEmpty()).NotTo(HaveOccurred())

		Expect(f.Client().Delete(f.Context(), &pod)).NotTo(HaveOccurred())
	})

	It("should mount but not be able to read the volume if the fs-group is not the same", func() {
		By("Creating another pod which doesn't have the correct fs-group should error when reading file")
		badPod := *podTemplate.DeepCopy()
		badPod.Namespace = f.Namespace.Name
		badPod.Spec.ServiceAccountName = serviceAccount.Name
		badPod.Spec.SecurityContext.RunAsGroup = ptr.To(int64(123))
		Expect(f.Client().Create(f.Context(), &badPod)).NotTo(HaveOccurred())

		By("Waiting for bad pod to become ready")
		Expect(util.WaitForPodReady(f, &badPod)).NotTo(HaveOccurred())

		By("Ensuring files cannot be read from volume")
		bundle, err := util.ReadCertFromMountPath(f, mountPath, badPod.Name, containerName)
		Expect(err).To(HaveOccurred())

		Expect(bundle).To(BeNil())

		Expect(f.Client().Delete(f.Context(), &badPod)).NotTo(HaveOccurred())
	})
})
