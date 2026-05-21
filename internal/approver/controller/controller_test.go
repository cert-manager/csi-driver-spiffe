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

package controller

import (
	"bytes"
	"context"
	"encoding/pem"
	"errors"
	"testing"
	"time"

	"github.com/cert-manager/cert-manager/pkg/api"
	apiutil "github.com/cert-manager/cert-manager/pkg/api/util"
	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	utilpki "github.com/cert-manager/cert-manager/pkg/util/pki"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2/ktesting"
	fakeclock "k8s.io/utils/clock/testing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/cert-manager/csi-driver-spiffe/internal/approver/evaluator"
	"github.com/cert-manager/csi-driver-spiffe/internal/approver/evaluator/fake"
	"github.com/cert-manager/csi-driver-spiffe/internal/csi/runtimeconfig"
)

func Test_Reconcile(t *testing.T) {
	var (
		fixedTime     = time.Date(2021, 01, 01, 01, 0, 0, 0, time.UTC)
		fixedmetatime = &metav1.Time{Time: fixedTime}
		fixedclock    = fakeclock.NewFakeClock(fixedTime)
	)

	spiffeAnnotations := map[string]string{
		"spiffe.csi.cert-manager.io/identity": "spiffe://cluster.local/ns/test-ns/sa/test-sa",
	}

	spiffeIssuerRef := cmmeta.IssuerReference{
		Name:  "spiffe-ca",
		Kind:  "ClusterIssuer",
		Group: "cert-manager.io",
	}
	otherIssuerRef := cmmeta.IssuerReference{
		Name:  "other-ca",
		Kind:  "ClusterIssuer",
		Group: "cert-manager.io",
	}
	spiffeRuntimeConfig := runtimeconfig.NewMemory(context.Background(),
		runtimeconfig.Config{IssuerRef: spiffeIssuerRef}, nil)

	pk, err := utilpki.GenerateECPrivateKey(utilpki.ECCurve521)
	assert.NoError(t, err)
	spiffeCSR, err := utilpki.GenerateCSR(&cmapi.Certificate{
		Spec: cmapi.CertificateSpec{
			PrivateKey: &cmapi.CertificatePrivateKey{Algorithm: cmapi.ECDSAKeyAlgorithm},
			URIs:       []string{"spiffe://cluster.local/ns/test-ns/sa/test-sa"},
		},
	})
	assert.NoError(t, err)
	spiffeCSRDER, err := utilpki.EncodeCSR(spiffeCSR, pk)
	assert.NoError(t, err)
	spiffeCSRPEM := bytes.NewBuffer([]byte{})
	assert.NoError(t, pem.Encode(spiffeCSRPEM, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: spiffeCSRDER}))

	tests := map[string]struct {
		existingCRObjects    []client.Object
		evaluator            evaluator.Interface
		runtimeConfig        runtimeconfig.Interface
		autoApproveNonSPIFFE bool
		expResult            ctrl.Result
		expError             bool
		expObjects           []client.Object
	}{
		"if CertificateRequest doesn't exist, ignore": {
			existingCRObjects: []client.Object{},
			expResult:         ctrl.Result{},
			expError:          false,
			evaluator:         nil,
			expObjects:        []client.Object{},
		},
		"if evaluator returns error, update Denied with error": {
			existingCRObjects: []client.Object{
				&cmapi.CertificateRequest{
					TypeMeta:   metav1.TypeMeta{Kind: "CertificateRequest", APIVersion: "cert-manager.io/v1"},
					ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cr", ResourceVersion: "10", Annotations: spiffeAnnotations},
				},
			},
			expResult: ctrl.Result{},
			evaluator: fake.New().WithEvaluate(func(_ *cmapi.CertificateRequest) error {
				return errors.New("this is an error")
			}),
			expError: false,
			expObjects: []client.Object{
				&cmapi.CertificateRequest{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cr", ResourceVersion: "11", Annotations: spiffeAnnotations},
					Status: cmapi.CertificateRequestStatus{
						Conditions: []cmapi.CertificateRequestCondition{
							{
								Type:               cmapi.CertificateRequestConditionDenied,
								Status:             cmmeta.ConditionTrue,
								Reason:             "spiffe.csi.cert-manager.io",
								Message:            "Denied request: this is an error",
								LastTransitionTime: fixedmetatime,
							},
						},
					},
				},
			},
		},
		"if evaluator doesn't return error, update Approved": {
			existingCRObjects: []client.Object{
				&cmapi.CertificateRequest{
					TypeMeta:   metav1.TypeMeta{Kind: "CertificateRequest", APIVersion: "cert-manager.io/v1"},
					ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cr", ResourceVersion: "10", Annotations: spiffeAnnotations},
				},
			},
			expResult: ctrl.Result{},
			evaluator: fake.New().WithEvaluate(func(_ *cmapi.CertificateRequest) error {
				return nil
			}),
			expError: false,
			expObjects: []client.Object{
				&cmapi.CertificateRequest{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cr", ResourceVersion: "11", Annotations: spiffeAnnotations},
					Status: cmapi.CertificateRequestStatus{
						Conditions: []cmapi.CertificateRequestCondition{
							{
								Type:               cmapi.CertificateRequestConditionApproved,
								Status:             cmmeta.ConditionTrue,
								Reason:             "spiffe.csi.cert-manager.io",
								Message:            "Approved request",
								LastTransitionTime: fixedmetatime,
							},
						},
					},
				},
			},
		},
		"auto-approve: unannotated request targeting non-SPIFFE issuer is Approved": {
			existingCRObjects: []client.Object{
				&cmapi.CertificateRequest{
					TypeMeta:   metav1.TypeMeta{Kind: "CertificateRequest", APIVersion: "cert-manager.io/v1"},
					ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cr", ResourceVersion: "10"},
					Spec:       cmapi.CertificateRequestSpec{IssuerRef: otherIssuerRef},
				},
			},
			evaluator:            fake.New(),
			runtimeConfig:        spiffeRuntimeConfig,
			autoApproveNonSPIFFE: true,
			expResult:            ctrl.Result{},
			expError:             false,
			expObjects: []client.Object{
				&cmapi.CertificateRequest{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cr", ResourceVersion: "11"},
					Spec:       cmapi.CertificateRequestSpec{IssuerRef: otherIssuerRef},
					Status: cmapi.CertificateRequestStatus{
						Conditions: []cmapi.CertificateRequestCondition{
							{
								Type:               cmapi.CertificateRequestConditionApproved,
								Status:             cmmeta.ConditionTrue,
								Reason:             "spiffe.csi.cert-manager.io",
								Message:            "Approved request",
								LastTransitionTime: fixedmetatime,
							},
						},
					},
				},
			},
		},
		"auto-approve: unannotated request targeting SPIFFE issuer is Denied": {
			existingCRObjects: []client.Object{
				&cmapi.CertificateRequest{
					TypeMeta:   metav1.TypeMeta{Kind: "CertificateRequest", APIVersion: "cert-manager.io/v1"},
					ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cr", ResourceVersion: "10"},
					Spec:       cmapi.CertificateRequestSpec{IssuerRef: spiffeIssuerRef},
				},
			},
			evaluator:            fake.New(),
			runtimeConfig:        spiffeRuntimeConfig,
			autoApproveNonSPIFFE: true,
			expResult:            ctrl.Result{},
			expError:             false,
			expObjects: []client.Object{
				&cmapi.CertificateRequest{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cr", ResourceVersion: "11"},
					Spec:       cmapi.CertificateRequestSpec{IssuerRef: spiffeIssuerRef},
					Status: cmapi.CertificateRequestStatus{
						Conditions: []cmapi.CertificateRequestCondition{
							{
								Type:               cmapi.CertificateRequestConditionDenied,
								Status:             cmmeta.ConditionTrue,
								Reason:             "spiffe.csi.cert-manager.io",
								Message:            "Denied request: non-SPIFFE certificate targeting configured SPIFFE issuer",
								LastTransitionTime: fixedmetatime,
							},
						},
					},
				},
			},
		},
		"auto-approve: unannotated request with SPIFFE URI SAN is Denied": {
			existingCRObjects: []client.Object{
				&cmapi.CertificateRequest{
					TypeMeta:   metav1.TypeMeta{Kind: "CertificateRequest", APIVersion: "cert-manager.io/v1"},
					ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cr", ResourceVersion: "10"},
					Spec: cmapi.CertificateRequestSpec{
						IssuerRef: otherIssuerRef,
						Request:   spiffeCSRPEM.Bytes(),
					},
				},
			},
			evaluator:            fake.New(),
			runtimeConfig:        spiffeRuntimeConfig,
			autoApproveNonSPIFFE: true,
			expResult:            ctrl.Result{},
			expError:             false,
			expObjects: []client.Object{
				&cmapi.CertificateRequest{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cr", ResourceVersion: "11"},
					Spec: cmapi.CertificateRequestSpec{
						IssuerRef: otherIssuerRef,
						Request:   spiffeCSRPEM.Bytes(),
					},
					Status: cmapi.CertificateRequestStatus{
						Conditions: []cmapi.CertificateRequestCondition{
							{
								Type:               cmapi.CertificateRequestConditionDenied,
								Status:             cmmeta.ConditionTrue,
								Reason:             "spiffe.csi.cert-manager.io",
								Message:            "Denied request: non-SPIFFE certificate request contains SPIFFE URI SAN",
								LastTransitionTime: fixedmetatime,
							},
						},
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			apiutil.Clock = fixedclock

			fakeclient := fakeclient.NewClientBuilder().
				WithScheme(api.Scheme).
				WithObjects(test.existingCRObjects...).
				WithStatusSubresource(test.existingCRObjects...).
				Build()

			a := &approver{
				client:               fakeclient,
				lister:               fakeclient,
				log:                  ktesting.NewLogger(t, ktesting.DefaultConfig),
				evaluator:            test.evaluator,
				runtimeConfig:        test.runtimeConfig,
				autoApproveNonSPIFFE: test.autoApproveNonSPIFFE,
			}

			result, err := a.Reconcile(t.Context(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-ns", Name: "test-cr"}})
			assert.Equalf(t, test.expError, err != nil, "%v", err)
			assert.Equal(t, test.expResult, result)

			for _, expObj := range test.expObjects {
				var actual client.Object
				switch expObj.(type) {
				case *cmapi.CertificateRequest:
					actual = &cmapi.CertificateRequest{}
				default:
					t.Errorf("unexpected object kind in expected: %#+v", expObj)
				}

				err := fakeclient.Get(t.Context(), client.ObjectKeyFromObject(expObj), actual)
				if err != nil {
					t.Errorf("unexpected error getting expected object: %s", err)
				} else if !apiequality.Semantic.DeepEqual(expObj, actual) {
					t.Errorf("unexpected expected object (-want +got):\n%s", cmp.Diff(expObj, actual))
				}
			}
		})
	}
}
