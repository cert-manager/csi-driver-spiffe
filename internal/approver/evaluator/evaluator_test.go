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

package evaluator

import (
	"bytes"
	"encoding/pem"
	"testing"
	"time"

	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	utilpki "github.com/cert-manager/cert-manager/pkg/util/pki"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func Test_Evaluate(t *testing.T) {
	pk, err := utilpki.GenerateECPrivateKey(utilpki.ECCurve521)
	assert.NoError(t, err)

	tests := map[string]struct {
		req    func(t *testing.T) *cmapi.CertificateRequest
		expErr bool
	}{
		"if request contains a badly encoded PEM, expect error": {
			req: func(t *testing.T) *cmapi.CertificateRequest {
				return &cmapi.CertificateRequest{Spec: cmapi.CertificateRequestSpec{
					Request:  []byte("bad-pem"),
					Duration: &metav1.Duration{Duration: time.Hour},
					Username: "system:serviceaccount:sandbox:sleep",
				}}
			},
			expErr: true,
		},
		"if request duration is nil, expect error": {
			req: func(t *testing.T) *cmapi.CertificateRequest {
				csr, err := utilpki.GenerateCSR(&cmapi.Certificate{
					Spec: cmapi.CertificateSpec{
						PrivateKey: &cmapi.CertificatePrivateKey{Algorithm: cmapi.ECDSAKeyAlgorithm},
						URIs:       []string{"spiffe://foo.bar/ns/sandbox/sa/sleep"},
					},
				})
				assert.NoError(t, err)
				csrDER, err := utilpki.EncodeCSR(csr, pk)
				assert.NoError(t, err)
				csrPEM := bytes.NewBuffer([]byte{})
				assert.NoError(t, pem.Encode(csrPEM, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER}))
				return &cmapi.CertificateRequest{Spec: cmapi.CertificateRequestSpec{
					Request:  csrPEM.Bytes(),
					Duration: nil,
					Username: "system:serviceaccount:sandbox:sleep",
					Usages: []cmapi.KeyUsage{
						cmapi.UsageServerAuth, cmapi.UsageClientAuth,
						cmapi.UsageDigitalSignature, cmapi.UsageKeyEncipherment,
					},
				}}
			},
			expErr: true,
		},
		"if request contains DNS names, expect error": {
			req: func(t *testing.T) *cmapi.CertificateRequest {
				csr, err := utilpki.GenerateCSR(&cmapi.Certificate{
					Spec: cmapi.CertificateSpec{
						PrivateKey: &cmapi.CertificatePrivateKey{Algorithm: cmapi.ECDSAKeyAlgorithm},
						URIs:       []string{"spiffe://foo.bar/ns/sandbox/sa/sleep"},
						DNSNames:   []string{"example.com"},
					},
				})
				assert.NoError(t, err)
				csrDER, err := utilpki.EncodeCSR(csr, pk)
				assert.NoError(t, err)
				csrPEM := bytes.NewBuffer([]byte{})
				assert.NoError(t, pem.Encode(csrPEM, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER}))
				return &cmapi.CertificateRequest{Spec: cmapi.CertificateRequestSpec{
					Request:  csrPEM.Bytes(),
					Duration: &metav1.Duration{Duration: time.Hour},
					Username: "system:serviceaccount:sandbox:sleep",
					Usages: []cmapi.KeyUsage{
						cmapi.UsageServerAuth, cmapi.UsageClientAuth,
						cmapi.UsageDigitalSignature, cmapi.UsageKeyEncipherment,
					},
				}}
			},
			expErr: true,
		},
		"if request contains IPs, expect error": {
			req: func(t *testing.T) *cmapi.CertificateRequest {
				csr, err := utilpki.GenerateCSR(&cmapi.Certificate{
					Spec: cmapi.CertificateSpec{
						PrivateKey:  &cmapi.CertificatePrivateKey{Algorithm: cmapi.ECDSAKeyAlgorithm},
						URIs:        []string{"spiffe://foo.bar/ns/sandbox/sa/sleep"},
						IPAddresses: []string{"1.2.3.4"},
					},
				})
				assert.NoError(t, err)
				csrDER, err := utilpki.EncodeCSR(csr, pk)
				assert.NoError(t, err)
				csrPEM := bytes.NewBuffer([]byte{})
				assert.NoError(t, pem.Encode(csrPEM, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER}))
				return &cmapi.CertificateRequest{Spec: cmapi.CertificateRequestSpec{
					Request:  csrPEM.Bytes(),
					Duration: &metav1.Duration{Duration: time.Hour},
					Username: "system:serviceaccount:sandbox:sleep",
					Usages: []cmapi.KeyUsage{
						cmapi.UsageServerAuth, cmapi.UsageClientAuth,
						cmapi.UsageDigitalSignature, cmapi.UsageKeyEncipherment,
					},
				}}
			},
			expErr: true,
		},
		"if request contains common name, expect error": {
			req: func(t *testing.T) *cmapi.CertificateRequest {
				csr, err := utilpki.GenerateCSR(&cmapi.Certificate{
					Spec: cmapi.CertificateSpec{
						PrivateKey: &cmapi.CertificatePrivateKey{Algorithm: cmapi.ECDSAKeyAlgorithm},
						URIs:       []string{"spiffe://foo.bar/ns/sandbox/sa/sleep"},
						CommonName: "example.com",
					},
				})
				assert.NoError(t, err)
				csrDER, err := utilpki.EncodeCSR(csr, pk)
				assert.NoError(t, err)
				csrPEM := bytes.NewBuffer([]byte{})
				assert.NoError(t, pem.Encode(csrPEM, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER}))
				return &cmapi.CertificateRequest{Spec: cmapi.CertificateRequestSpec{
					Request:  csrPEM.Bytes(),
					Duration: &metav1.Duration{Duration: time.Hour},
					Username: "system:serviceaccount:sandbox:sleep",
					Usages: []cmapi.KeyUsage{
						cmapi.UsageServerAuth, cmapi.UsageClientAuth,
						cmapi.UsageDigitalSignature, cmapi.UsageKeyEncipherment,
					},
				}}
			},
			expErr: true,
		},
		"if request contains email addresses, expect error": {
			req: func(t *testing.T) *cmapi.CertificateRequest {
				csr, err := utilpki.GenerateCSR(&cmapi.Certificate{
					Spec: cmapi.CertificateSpec{
						PrivateKey:     &cmapi.CertificatePrivateKey{Algorithm: cmapi.ECDSAKeyAlgorithm},
						URIs:           []string{"spiffe://foo.bar/ns/sandbox/sa/sleep"},
						EmailAddresses: []string{"alice@example.com"},
					},
				})
				assert.NoError(t, err)
				csrDER, err := utilpki.EncodeCSR(csr, pk)
				assert.NoError(t, err)
				csrPEM := bytes.NewBuffer([]byte{})
				assert.NoError(t, pem.Encode(csrPEM, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER}))
				return &cmapi.CertificateRequest{Spec: cmapi.CertificateRequestSpec{
					Request:  csrPEM.Bytes(),
					Duration: &metav1.Duration{Duration: time.Hour},
					Username: "system:serviceaccount:sandbox:sleep",
					Usages: []cmapi.KeyUsage{
						cmapi.UsageServerAuth, cmapi.UsageClientAuth,
						cmapi.UsageDigitalSignature, cmapi.UsageKeyEncipherment,
					},
				}}
			},
			expErr: true,
		},
		"if request is with isCA=true, expect error": {
			req: func(t *testing.T) *cmapi.CertificateRequest {
				csr, err := utilpki.GenerateCSR(&cmapi.Certificate{
					Spec: cmapi.CertificateSpec{
						PrivateKey: &cmapi.CertificatePrivateKey{Algorithm: cmapi.ECDSAKeyAlgorithm},
						URIs:       []string{"spiffe://foo.bar/ns/sandbox/sa/sleep"},
					},
				})
				assert.NoError(t, err)
				csrDER, err := utilpki.EncodeCSR(csr, pk)
				assert.NoError(t, err)
				csrPEM := bytes.NewBuffer([]byte{})
				assert.NoError(t, pem.Encode(csrPEM, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER}))
				return &cmapi.CertificateRequest{Spec: cmapi.CertificateRequestSpec{
					Request:  csrPEM.Bytes(),
					IsCA:     true,
					Duration: &metav1.Duration{Duration: time.Hour},
					Username: "system:serviceaccount:sandbox:sleep",
					Usages: []cmapi.KeyUsage{
						cmapi.UsageServerAuth, cmapi.UsageClientAuth,
						cmapi.UsageDigitalSignature, cmapi.UsageKeyEncipherment,
					},
				}}
			},
			expErr: true,
		},
		"if request has the wrong usages, expect error": {
			req: func(t *testing.T) *cmapi.CertificateRequest {
				csr, err := utilpki.GenerateCSR(&cmapi.Certificate{
					Spec: cmapi.CertificateSpec{
						PrivateKey: &cmapi.CertificatePrivateKey{Algorithm: cmapi.ECDSAKeyAlgorithm},
						URIs:       []string{"spiffe://foo.bar/ns/sandbox/sa/sleep"},
					},
				})
				assert.NoError(t, err)
				csrDER, err := utilpki.EncodeCSR(csr, pk)
				assert.NoError(t, err)
				csrPEM := bytes.NewBuffer([]byte{})
				assert.NoError(t, pem.Encode(csrPEM, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER}))
				return &cmapi.CertificateRequest{Spec: cmapi.CertificateRequestSpec{
					Request:  csrPEM.Bytes(),
					Duration: &metav1.Duration{Duration: time.Hour},
					Username: "system:serviceaccount:sandbox:sleep",
					Usages: []cmapi.KeyUsage{
						cmapi.UsageDigitalSignature, cmapi.UsageKeyEncipherment,
					},
				}}
			},
			expErr: true,
		},
		"if request has the wrong usages encoded in the request, expect error": {
			req: func(t *testing.T) *cmapi.CertificateRequest {
				csr, err := utilpki.GenerateCSR(&cmapi.Certificate{
					Spec: cmapi.CertificateSpec{
						PrivateKey:            &cmapi.CertificatePrivateKey{Algorithm: cmapi.ECDSAKeyAlgorithm},
						EncodeUsagesInRequest: ptr.To(true),
						URIs:                  []string{"spiffe://foo.bar/ns/sandbox/sa/sleep"},
						Usages: []cmapi.KeyUsage{
							cmapi.UsageDigitalSignature, cmapi.UsageKeyEncipherment,
							cmapi.UsageCertSign, cmapi.UsageCodeSigning,
						},
					},
				})
				assert.NoError(t, err)
				csrDER, err := utilpki.EncodeCSR(csr, pk)
				assert.NoError(t, err)
				csrPEM := bytes.NewBuffer([]byte{})
				assert.NoError(t, pem.Encode(csrPEM, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER}))
				return &cmapi.CertificateRequest{Spec: cmapi.CertificateRequestSpec{
					Request:  csrPEM.Bytes(),
					Duration: &metav1.Duration{Duration: time.Hour},
					Username: "system:serviceaccount:sandbox:sleep",
					Usages: []cmapi.KeyUsage{
						cmapi.UsageServerAuth, cmapi.UsageClientAuth,
						cmapi.UsageDigitalSignature, cmapi.UsageKeyEncipherment,
					},
				}}
			},
			expErr: true,
		},
		"if request has the wrong identity, expect error": {
			req: func(t *testing.T) *cmapi.CertificateRequest {
				csr, err := utilpki.GenerateCSR(&cmapi.Certificate{
					Spec: cmapi.CertificateSpec{
						PrivateKey: &cmapi.CertificatePrivateKey{Algorithm: cmapi.ECDSAKeyAlgorithm},
						URIs:       []string{"spiffe://foo.bar/ns/sandbox/sa/httpbin"},
					},
				})
				assert.NoError(t, err)
				csrDER, err := utilpki.EncodeCSR(csr, pk)
				assert.NoError(t, err)
				csrPEM := bytes.NewBuffer([]byte{})
				assert.NoError(t, pem.Encode(csrPEM, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER}))
				return &cmapi.CertificateRequest{Spec: cmapi.CertificateRequestSpec{
					Request:  csrPEM.Bytes(),
					Duration: &metav1.Duration{Duration: time.Hour},
					Username: "system:serviceaccount:sandbox:sleep",
					Usages: []cmapi.KeyUsage{
						cmapi.UsageServerAuth, cmapi.UsageClientAuth,
						cmapi.UsageDigitalSignature, cmapi.UsageKeyEncipherment,
					},
				}}
			},
			expErr: true,
		},
		"if is valid, expect no error": {
			req: func(t *testing.T) *cmapi.CertificateRequest {
				csr, err := utilpki.GenerateCSR(&cmapi.Certificate{
					Spec: cmapi.CertificateSpec{
						PrivateKey: &cmapi.CertificatePrivateKey{Algorithm: cmapi.ECDSAKeyAlgorithm},
						URIs:       []string{"spiffe://foo.bar/ns/sandbox/sa/sleep"},
					},
				})
				assert.NoError(t, err)
				csrDER, err := utilpki.EncodeCSR(csr, pk)
				assert.NoError(t, err)
				csrPEM := bytes.NewBuffer([]byte{})
				assert.NoError(t, pem.Encode(csrPEM, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER}))
				return &cmapi.CertificateRequest{Spec: cmapi.CertificateRequestSpec{
					Request:  csrPEM.Bytes(),
					Duration: &metav1.Duration{Duration: time.Hour},
					Username: "system:serviceaccount:sandbox:sleep",
					Usages: []cmapi.KeyUsage{
						cmapi.UsageServerAuth, cmapi.UsageClientAuth,
						cmapi.UsageDigitalSignature, cmapi.UsageKeyEncipherment,
					},
				}}
			},
			expErr: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			i := &internal{
				trustDomain:                "foo.bar",
				certificateRequestDuration: time.Hour,
			}

			err := i.Evaluate(test.req(t))
			assert.Equal(t, test.expErr, err != nil, "%v", err)
		})
	}
}
