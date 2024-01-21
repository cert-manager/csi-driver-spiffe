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
	"bytes"
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cert-manager/csi-driver-spiffe/test/e2e/framework"
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
				Volumes: []corev1.Volume{corev1.Volume{
					Name: "csi-driver-spiffe",
					VolumeSource: corev1.VolumeSource{
						CSI: &corev1.CSIVolumeSource{
							Driver:   "spiffe.csi.cert-manager.io",
							ReadOnly: pointer.Bool(true),
							VolumeAttributes: map[string]string{
								"spiffe.csi.cert-manager.io/fs-group": "1541",
							},
						},
					},
				}},
				SecurityContext: &corev1.PodSecurityContext{
					RunAsUser:  pointer.Int64(1321),
					RunAsGroup: pointer.Int64(1541),
				},
				Containers: []corev1.Container{
					corev1.Container{
						Name:            "my-container",
						Image:           "docker.io/library/busybox:1.36.1-musl",
						ImagePullPolicy: corev1.PullNever,
						Command:         []string{"sleep", "10000"},
						VolumeMounts: []corev1.VolumeMount{
							corev1.VolumeMount{
								Name:      "csi-driver-spiffe",
								MountPath: "/var/run/secrets/my-pod",
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
		Eventually(func() bool {
			Expect(f.Client().Get(f.Context(), client.ObjectKey{Namespace: f.Namespace.Name, Name: pod.Name}, &pod)).NotTo(HaveOccurred())
			for _, c := range pod.Status.Conditions {
				if c.Type == corev1.PodReady {
					return c.Status == corev1.ConditionTrue
				}
			}
			return false
		}, "180s", "1s").Should(BeTrue(), "expected pod to become ready in time")

		By("Ensuring files can be read from volume")
		for _, filename := range []string{"tls.crt", "tls.key", "ca.crt"} {
			buf := new(bytes.Buffer)
			cmd := exec.Command(f.Config().KubectlBinPath, "exec", "-n"+f.Namespace.Name, pod.Name, "-cmy-container", "--", "cat", "/var/run/secrets/my-pod/"+filename)
			cmd.Stdout = buf
			cmd.Stderr = GinkgoWriter
			cmd.Run()

			Expect(buf.Bytes()).NotTo(HaveLen(0), "expected the file to have a non-zero entry")
		}
		Expect(f.Client().Delete(f.Context(), &pod)).NotTo(HaveOccurred())
	})

	It("should mount but not be able to read the volume if the fs-group is not the same", func() {
		By("Creating another pod which doesn't have the correct fs-group should error when reading file")
		badPod := *podTemplate.DeepCopy()
		badPod.Namespace = f.Namespace.Name
		badPod.Spec.ServiceAccountName = serviceAccount.Name
		badPod.Spec.SecurityContext.RunAsGroup = pointer.Int64(123)
		Expect(f.Client().Create(f.Context(), &badPod)).NotTo(HaveOccurred())

		By("Waiting for pod to become ready")
		Eventually(func() bool {
			Expect(f.Client().Get(f.Context(), client.ObjectKey{Namespace: f.Namespace.Name, Name: badPod.Name}, &badPod)).NotTo(HaveOccurred())
			for _, c := range badPod.Status.Conditions {
				if c.Type == corev1.PodReady {
					return c.Status == corev1.ConditionTrue
				}
			}
			return false
		}, "180s", "1s").Should(BeTrue(), "expected pod to become ready in time")

		By("Ensuring files cannot be read from volume")
		for _, filename := range []string{"tls.crt", "tls.key", "ca.crt"} {
			buf := new(bytes.Buffer)
			cmd := exec.Command(f.Config().KubectlBinPath, "exec", "-n"+f.Namespace.Name, badPod.Name, "-cmy-container", "--", "cat", "/var/run/secrets/my-pod/"+filename)
			cmd.Stdout = buf
			cmd.Stderr = GinkgoWriter
			cmd.Run()

			Expect(buf.Bytes()).To(HaveLen(0), "expected the file to have a zero entry")
		}
		Expect(f.Client().Delete(f.Context(), &badPod)).NotTo(HaveOccurred())
	})
})
