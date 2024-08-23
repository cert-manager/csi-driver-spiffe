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

package test

import (
	"context"
	"errors"

	apiutil "github.com/cert-manager/cert-manager/pkg/api/util"
	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/cert-manager/csi-driver-spiffe/internal/annotations"
	"github.com/cert-manager/csi-driver-spiffe/internal/approver/controller"
	evaluatorfake "github.com/cert-manager/csi-driver-spiffe/internal/approver/evaluator/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context("Approval", func() {
	var (
		ctx    context.Context
		cancel func()

		cl        client.Client
		namespace corev1.Namespace

		evaluator = evaluatorfake.New()
		issuerRef = cmmeta.ObjectReference{
			Name:  "spiffe-ca",
			Kind:  "ClusterIssuer",
			Group: "cert-manager.io",
		}
	)

	JustBeforeEach(func() {
		ctx, cancel = context.WithCancel(context.TODO())

		scheme := runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).NotTo(HaveOccurred())
		Expect(cmapi.AddToScheme(scheme)).NotTo(HaveOccurred())

		var err error
		cl, err = client.New(env.Config, client.Options{Scheme: scheme})
		Expect(err).NotTo(HaveOccurred())

		namespace = corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-csi-driver-spiffe-",
			},
		}
		Expect(cl.Create(ctx, &namespace)).NotTo(HaveOccurred())

		log := GinkgoLogr
		mgr, err := ctrl.NewManager(env.Config, ctrl.Options{
			Scheme:         scheme,
			LeaderElection: true,
			Metrics: server.Options{
				BindAddress: "0",
			},
			LeaderElectionNamespace:       namespace.Name,
			LeaderElectionID:              "cert-manager-csi-driver-spiffe-approver",
			LeaderElectionReleaseOnCancel: true,
			Logger:                        log,
			Controller: config.Controller{
				// need to skip unique controller name validation
				// since all tests need a dedicated controller
				SkipNameValidation: ptr.To(true),
			},
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(controller.AddApprover(ctx, log, controller.Options{
			Manager:   mgr,
			Evaluator: evaluator,
		})).NotTo(HaveOccurred())

		By("Running Approver controller")
		go func() {
			Expect(mgr.Start(ctx)).NotTo(HaveOccurred())
		}()

		By("Waiting for Leader Election")
		<-mgr.Elected()

		By("Waiting for Informers to Sync")
		Expect(mgr.GetCache().WaitForCacheSync(ctx)).Should(BeTrue())
	})

	JustAfterEach(func() {
		Expect(cl.Delete(ctx, &namespace)).NotTo(HaveOccurred())
		cancel()
	})

	It("should ignore CertificateRequests that are missing the identity annotation", func() {
		cr := cmapi.CertificateRequest{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "cert-manager-csi-driver-spiffe-",
				Namespace:    namespace.Name,
				Annotations:  map[string]string{
					// intentionally left blank
				},
			},
			Spec: cmapi.CertificateRequestSpec{
				Request:   []byte("request"),
				IssuerRef: issuerRef,
			},
		}
		Expect(cl.Create(ctx, &cr)).NotTo(HaveOccurred())

		Consistently(func() bool {
			Eventually(func() error {
				return cl.Get(ctx, client.ObjectKeyFromObject(&cr), &cr)
			}).Should(BeNil())
			return apiutil.CertificateRequestIsApproved(&cr) || apiutil.CertificateRequestIsDenied(&cr)
		}, "3s").Should(BeFalse(), "expected neither approved not denied")
	})

	It("should deny CertificateRequest when the evaluator returns error", func() {
		evaluator.WithEvaluate(func(_ *cmapi.CertificateRequest) error {
			return errors.New("this is an error")
		})

		cr := cmapi.CertificateRequest{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "cert-manager-csi-driver-spiffe-",
				Namespace:    namespace.Name,
				Annotations: map[string]string{
					annotations.SPIFFEIdentityAnnnotationKey: "sentinel",
				},
			},
			Spec: cmapi.CertificateRequestSpec{
				Request:   []byte("request"),
				IssuerRef: issuerRef,
			},
		}
		Expect(cl.Create(ctx, &cr)).NotTo(HaveOccurred())

		Eventually(func() bool {
			Eventually(func() error {
				return cl.Get(ctx, client.ObjectKeyFromObject(&cr), &cr)
			}).Should(BeNil())
			return apiutil.CertificateRequestIsDenied(&cr)
		}).Should(BeTrue(), "expected denial")
	})

	It("should approve CertificateRequest when the evaluator returns nil", func() {
		evaluator.WithEvaluate(func(_ *cmapi.CertificateRequest) error {
			return nil
		})

		cr := cmapi.CertificateRequest{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "cert-manager-csi-driver-spiffe-",
				Namespace:    namespace.Name,
				Annotations: map[string]string{
					annotations.SPIFFEIdentityAnnnotationKey: "sentinel",
				},
			},
			Spec: cmapi.CertificateRequestSpec{
				Request:   []byte("request"),
				IssuerRef: issuerRef,
			},
		}
		Expect(cl.Create(ctx, &cr)).NotTo(HaveOccurred())

		Eventually(func() bool {
			Eventually(func() error {
				return cl.Get(ctx, client.ObjectKeyFromObject(&cr), &cr)
			}).Should(BeNil())
			return apiutil.CertificateRequestIsApproved(&cr)
		}).Should(BeTrue(), "expected approval")
	})
})
