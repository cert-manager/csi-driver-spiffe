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

package approval

import (
	"bytes"
	"crypto"
	"encoding/pem"
	"fmt"
	"time"

	apiutil "github.com/cert-manager/cert-manager/pkg/api/util"
	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	utilpki "github.com/cert-manager/cert-manager/pkg/util/pki"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cert-manager/csi-driver-spiffe/test/e2e/framework"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = framework.CasesDescribe("Approval", func() {
	f := framework.NewDefaultFramework("Approval")

	var (
		pk             crypto.Signer
		serviceAccount corev1.ServiceAccount
		role           rbacv1.Role
		rolebinding    rbacv1.RoleBinding
		cl             client.Client
	)

	JustBeforeEach(func() {
		By("Creating test resources")
		var err error
		pk, err = utilpki.GenerateECPrivateKey(utilpki.ECCurve521)
		Expect(err).NotTo(HaveOccurred())

		serviceAccount = corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Namespace: f.Namespace.Name, GenerateName: "csi-driver-spiffe-approval-test-"},
		}
		Expect(f.Client().Create(f.Context(), &serviceAccount)).NotTo(HaveOccurred())

		role = rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "csi-driver-spiffe-approval-test-",
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
				GenerateName: "csi-driver-spiffe-approval-test-",
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

		impersonateRestConfig := *f.Config().RestConfig
		impersonateRestConfig.Impersonate = rest.ImpersonationConfig{UserName: fmt.Sprintf("system:serviceaccount:%s:%s", f.Namespace.Name, serviceAccount.Name)}
		cl, err = client.New(&impersonateRestConfig, client.Options{Scheme: f.Client().Scheme()})
		Expect(err).NotTo(HaveOccurred())
	})

	JustAfterEach(func() {
		By("Cleaning up test resources")
		Expect(f.Client().Delete(f.Context(), &rolebinding)).NotTo(HaveOccurred())
		Expect(f.Client().Delete(f.Context(), &role)).NotTo(HaveOccurred())
		Expect(f.Client().Delete(f.Context(), &serviceAccount)).NotTo(HaveOccurred())
	})

	It("should approve a valid request", func() {
		By("Creating valid request")
		certificateRequest := cmapi.CertificateRequest{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-request-",
				Namespace:    f.Namespace.Name,
			},
			Spec: cmapi.CertificateRequestSpec{
				Request:   genCSRPEM(pk, cmapi.ECDSAKeyAlgorithm, fmt.Sprintf("spiffe://foo.bar/ns/%s/sa/%s", f.Namespace.Name, serviceAccount.Name)),
				Duration:  &metav1.Duration{Duration: time.Hour},
				IssuerRef: f.Config().IssuerRef,
				IsCA:      false,
				Usages:    []cmapi.KeyUsage{cmapi.UsageDigitalSignature, cmapi.UsageKeyEncipherment, cmapi.UsageClientAuth, cmapi.UsageServerAuth},
			},
		}
		Expect(cl.Create(f.Context(), &certificateRequest)).NotTo(HaveOccurred())

		By("Waiting for valid request to be approved")
		Eventually(func() bool {
			Expect(f.Client().Get(f.Context(), client.ObjectKeyFromObject(&certificateRequest), &certificateRequest)).NotTo(HaveOccurred())
			return apiutil.CertificateRequestIsApproved(&certificateRequest)
		}, "5s", "1s").Should(BeTrue(), "expected request to become approved in time")
	})

	It("should deny a request with the wrong duration", func() {
		By("Creating request with wrong duration")
		certificateRequest := cmapi.CertificateRequest{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-request-",
				Namespace:    f.Namespace.Name,
			},
			Spec: cmapi.CertificateRequestSpec{
				Request:   genCSRPEM(pk, cmapi.ECDSAKeyAlgorithm, fmt.Sprintf("spiffe://foo.bar/ns/%s/sa/%s", f.Namespace.Name, serviceAccount.Name)),
				Duration:  &metav1.Duration{Duration: time.Hour * 3},
				IssuerRef: f.Config().IssuerRef,
				IsCA:      false,
				Usages:    []cmapi.KeyUsage{cmapi.UsageDigitalSignature, cmapi.UsageKeyEncipherment, cmapi.UsageClientAuth, cmapi.UsageServerAuth},
			},
		}
		Expect(cl.Create(f.Context(), &certificateRequest)).NotTo(HaveOccurred())

		By("Waiting for valid request to be denied")
		Eventually(func() bool {
			Expect(f.Client().Get(f.Context(), client.ObjectKeyFromObject(&certificateRequest), &certificateRequest)).NotTo(HaveOccurred())
			return apiutil.CertificateRequestIsDenied(&certificateRequest)
		}, "5s", "1s").Should(BeTrue(), "expected request to be denied in time")
	})

	It("should deny a request with the wrong SPIFFE ID", func() {
		By("Creating request with wrong SPIFFE ID")
		certificateRequest := cmapi.CertificateRequest{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-request-",
				Namespace:    f.Namespace.Name,
			},
			Spec: cmapi.CertificateRequestSpec{
				Request:   genCSRPEM(pk, cmapi.ECDSAKeyAlgorithm, fmt.Sprintf("spiffe://foo.bar/ns/%s/sa/%s", f.Namespace.Name, "not-the-right-sa")),
				Duration:  &metav1.Duration{Duration: time.Hour},
				IssuerRef: f.Config().IssuerRef,
				IsCA:      false,
				Usages:    []cmapi.KeyUsage{cmapi.UsageDigitalSignature, cmapi.UsageKeyEncipherment, cmapi.UsageClientAuth, cmapi.UsageServerAuth},
			},
		}
		Expect(cl.Create(f.Context(), &certificateRequest)).NotTo(HaveOccurred())

		By("Waiting for valid request to be denied")
		Eventually(func() bool {
			Expect(f.Client().Get(f.Context(), client.ObjectKeyFromObject(&certificateRequest), &certificateRequest)).NotTo(HaveOccurred())
			return apiutil.CertificateRequestIsDenied(&certificateRequest)
		}, "5s", "1s").Should(BeTrue(), "expected request to be denied in time")
	})

	It("should deny a request with the wrong key usages", func() {
		By("Creating request with wrong key usages")
		certificateRequest := cmapi.CertificateRequest{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-request-",
				Namespace:    f.Namespace.Name,
			},
			Spec: cmapi.CertificateRequestSpec{
				Request:   genCSRPEM(pk, cmapi.ECDSAKeyAlgorithm, fmt.Sprintf("spiffe://foo.bar/ns/%s/sa/%s", f.Namespace.Name, serviceAccount.Name)),
				Duration:  &metav1.Duration{Duration: time.Hour},
				IssuerRef: f.Config().IssuerRef,
				IsCA:      false,
				Usages:    []cmapi.KeyUsage{cmapi.UsageDigitalSignature, cmapi.UsageKeyEncipherment, cmapi.UsageClientAuth, cmapi.UsageServerAuth, cmapi.UsageCertSign},
			},
		}
		Expect(cl.Create(f.Context(), &certificateRequest)).NotTo(HaveOccurred())

		By("Waiting for valid request to be denied")
		Eventually(func() bool {
			Expect(f.Client().Get(f.Context(), client.ObjectKeyFromObject(&certificateRequest), &certificateRequest)).NotTo(HaveOccurred())
			return apiutil.CertificateRequestIsDenied(&certificateRequest)
		}, "5s", "1s").Should(BeTrue(), "expected request to be denied in time")
	})

	It("should deny a request with the wrong key type", func() {
		pk, err := utilpki.GenerateRSAPrivateKey(2048)
		Expect(err).NotTo(HaveOccurred())

		By("Creating request with wrong key type")
		certificateRequest := cmapi.CertificateRequest{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-request-",
				Namespace:    f.Namespace.Name,
			},
			Spec: cmapi.CertificateRequestSpec{
				Request:   genCSRPEM(pk, cmapi.RSAKeyAlgorithm, fmt.Sprintf("spiffe://foo.bar/ns/%s/sa/%s", f.Namespace.Name, serviceAccount.Name)),
				Duration:  &metav1.Duration{Duration: time.Hour},
				IssuerRef: f.Config().IssuerRef,
				IsCA:      false,
				Usages:    []cmapi.KeyUsage{cmapi.UsageDigitalSignature, cmapi.UsageKeyEncipherment, cmapi.UsageClientAuth, cmapi.UsageServerAuth, cmapi.UsageCertSign},
			},
		}
		Expect(cl.Create(f.Context(), &certificateRequest)).NotTo(HaveOccurred())

		By("Waiting for valid request to be denied")
		Eventually(func() bool {
			Expect(f.Client().Get(f.Context(), client.ObjectKeyFromObject(&certificateRequest), &certificateRequest)).NotTo(HaveOccurred())
			return apiutil.CertificateRequestIsDenied(&certificateRequest)
		}, "5s", "1s").Should(BeTrue(), "expected request to be denied in time")
	})
})

func genCSRPEM(pk crypto.Signer, alg cmapi.PrivateKeyAlgorithm, uri string) []byte {
	csr, err := utilpki.GenerateCSR(&cmapi.Certificate{
		Spec: cmapi.CertificateSpec{
			PrivateKey:            &cmapi.CertificatePrivateKey{Algorithm: alg},
			URIs:                  []string{uri},
			EncodeUsagesInRequest: ptr.To(false),
		},
	})
	Expect(err).NotTo(HaveOccurred())
	csrDER, err := utilpki.EncodeCSR(csr, pk)
	Expect(err).NotTo(HaveOccurred())
	csrPEM := bytes.NewBuffer([]byte{})
	Expect(err).NotTo(HaveOccurred())
	Expect(pem.Encode(csrPEM, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})).ToNot(HaveOccurred())

	return csrPEM.Bytes()
}
