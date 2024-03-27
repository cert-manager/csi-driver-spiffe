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
	"bytes"
	"os/exec"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cert-manager/csi-driver-spiffe/test/e2e/framework"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
				Volumes: []corev1.Volume{corev1.Volume{
					Name: "csi-driver-spiffe",
					VolumeSource: corev1.VolumeSource{
						CSI: &corev1.CSIVolumeSource{
							Driver:   "spiffe.csi.cert-manager.io",
							ReadOnly: pointer.Bool(true),
						},
					},
				}},
				ServiceAccountName: "test-pod",
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
		pod2 := *(pod1.DeepCopy())
		pod2.Name = "test-pod-2"

		Expect(f.Client().Create(f.Context(), &pod1)).NotTo(HaveOccurred())
		Expect(f.Client().Create(f.Context(), &pod2)).NotTo(HaveOccurred())

		By("Waiting for pods to become ready")
		for _, podName := range []string{"test-pod-1", "test-pod-2"} {
			Eventually(func() bool {
				var pod corev1.Pod
				Expect(f.Client().Get(f.Context(), client.ObjectKey{Namespace: f.Namespace.Name, Name: podName}, &pod)).NotTo(HaveOccurred())

				for _, c := range pod.Status.Conditions {
					if c.Type == corev1.PodReady {
						return c.Status == corev1.ConditionTrue
					}
				}

				return false
			}, "20s", "1s").Should(BeTrue(), "expected pod to become ready in time")
		}

		By("Comparing the CA stored in secret with CA stored in the Secret")
		var caSecret corev1.Secret
		Expect(f.Client().Get(f.Context(), client.ObjectKey{Namespace: f.Config().IssuerSecretNamespace, Name: f.Config().IssuerSecretName}, &caSecret)).NotTo(HaveOccurred())
		caData, ok := caSecret.Data["ca.crt"]
		Expect(ok).To(BeTrue(), "expected 'ca.crt' to be present in Issuer CA Secret")

		for _, podName := range []string{"test-pod-1", "test-pod-2"} {
			buf := new(bytes.Buffer)
			cmd := exec.Command(f.Config().KubectlBinPath, "exec", "-n"+f.Namespace.Name, podName, "-cmy-container", "--", "cat", "/var/run/secrets/my-pod/ca.crt")
			cmd.Stdout = buf
			cmd.Stderr = GinkgoWriter
			cmd.Run()

			Expect(caData).To(Equal(buf.Bytes()), "expected the Issuer CA bundle to equal the CA mounted to the pod file")
		}

		By("Updating the CA data in Secret")
		newCAData := append([]byte("# This is a comment\n"), caData...)
		caSecret.Data["ca.crt"] = newCAData
		Expect(f.Client().Update(f.Context(), &caSecret)).NotTo(HaveOccurred())

		By("Waiting for the new CA data to be written to pod volumes")
		for _, podName := range []string{"test-pod-1", "test-pod-2"} {
			Eventually(func() bool {
				buf := new(bytes.Buffer)
				cmd := exec.Command(f.Config().KubectlBinPath, "exec", "-n"+f.Namespace.Name, podName, "-cmy-container", "--", "cat", "/var/run/secrets/my-pod/ca.crt")
				cmd.Stdout = buf
				cmd.Stderr = GinkgoWriter
				Expect(cmd.Run()).ToNot(HaveOccurred())

				return bytes.Equal(buf.Bytes(), newCAData)
			}, "100s", "1s").Should(BeTrue(), "expected the CA data to be updated on pod file")
		}

		By("Cleaning up resources")
		Expect(f.Client().Delete(f.Context(), &rolebinding)).NotTo(HaveOccurred())
		Expect(f.Client().Delete(f.Context(), &role)).NotTo(HaveOccurred())
		Expect(f.Client().Delete(f.Context(), &serviceAccount)).NotTo(HaveOccurred())
		Expect(f.Client().Delete(f.Context(), &pod1)).NotTo(HaveOccurred())
		Expect(f.Client().Delete(f.Context(), &pod2)).NotTo(HaveOccurred())
	})
})
