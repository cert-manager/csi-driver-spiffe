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

package driver

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"math/big"
	"net/url"
	"reflect"
	"testing"
	"time"

	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	utilpki "github.com/cert-manager/cert-manager/pkg/util/pki"
	"github.com/cert-manager/csi-lib/metadata"
	"github.com/cert-manager/csi-lib/storage"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/stretchr/testify/require"

	"hegel.dev/go/hegel"

	"github.com/cert-manager/csi-driver-spiffe/internal/annotations"
	"github.com/cert-manager/csi-driver-spiffe/internal/csi/rootca"
)

// TestCalculateNextIssuanceTimeProperty: for any leaf certificate, the next
// issuance time is exactly 2/3 of the way through the certificate's actual
// lifetime, recomputed here from the drawn validity period independently of
// the PEM decode / X.509 parse pipeline. Only the first PEM block (the leaf)
// governs, so appending further chain blocks changes nothing. Previously
// calculateNextIssuanceTime was only executed incidentally via
// Test_writeKeyPair; its result was never asserted.
//
// Only PEM inputs are drawn: for input with no PEM block at all,
// calculateNextIssuanceTime dereferences the nil block returned by
// pem.Decode and panics — a production fix candidate, not exercisable here.
func TestCalculateNextIssuanceTimeProperty(t *testing.T) {
	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// notBefore is drawn from 1970 to 2100, lifetimes from zero to a century.
	// X.509 stores validity at second precision, so times and lifetimes are
	// drawn in whole seconds.
	const (
		maxUnix     = 4102444800
		maxLifetime = 100 * 365 * 24 * 3600
	)

	hegel.Test(t, func(ht *hegel.T) {
		notBefore := time.Unix(int64(hegel.Draw(ht, hegel.Integers(0, maxUnix))), 0).UTC()
		lifetime := time.Duration(hegel.Draw(ht, hegel.Integers(0, maxLifetime))) * time.Second
		notAfter := notBefore.Add(lifetime)

		tmpl := &x509.Certificate{
			SerialNumber: big.NewInt(1),
			NotBefore:    notBefore,
			NotAfter:     notAfter,
		}
		der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, pk.Public(), pk)
		if err != nil {
			ht.Fatalf("failed to create certificate: %v", err)
		}
		chain := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
		if hegel.Draw(ht, hegel.Booleans()) {
			chain = append(chain, chain...)
		}

		got, err := calculateNextIssuanceTime(chain)
		if err != nil {
			ht.Fatalf("notBefore %v lifetime %v: %v", notBefore, lifetime, err)
		}
		want := notAfter.Add(-lifetime / 3)
		if !got.Equal(want) {
			ht.Fatalf("notBefore %v lifetime %v: got %v, want %v", notBefore, lifetime, got, want)
		}
	}, hegel.WithTestCases(250))
}

// Ensure writeKeyPair is compatible with go-spiffe/v2 x509svid.Parse.
func Test_writeKeyPair(t *testing.T) {
	capk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	caTmpl, err := utilpki.CertificateTemplateFromCertificate(&cmapi.Certificate{Spec: cmapi.CertificateSpec{CommonName: "my-ca"}})
	require.NoError(t, err)

	caPEM, ca, err := utilpki.SignCertificate(caTmpl, caTmpl, capk.Public(), capk)
	require.NoError(t, err)

	leafpk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	leafTmpl, err := utilpki.CertificateTemplateFromCertificate(
		&cmapi.Certificate{
			Spec: cmapi.CertificateSpec{URIs: []string{"spiffe://cert-manager.io/ns/sandbox/sa/default"}},
		},
	)
	require.NoError(t, err)

	leafPEM, _, err := utilpki.SignCertificate(leafTmpl, ca, leafpk.Public(), capk)
	require.NoError(t, err)

	ch := make(chan []byte)
	rootCAs := rootca.NewMemory(t.Context(), ch)
	ch <- caPEM

	store := storage.NewMemoryFS()
	d := &Driver{
		certFileName: "crt.pem",
		keyFileName:  "key.pem",
		caFileName:   "ca.pem",
		rootCAs:      rootCAs,
		store:        store,
	}

	meta := metadata.Metadata{VolumeID: "vol-id"}

	_, err = store.RegisterMetadata(meta)
	require.NoError(t, err)

	err = d.writeKeypair(meta, leafpk, leafPEM, nil)
	require.NoError(t, err)

	files, err := store.ReadFiles("vol-id")
	require.NoError(t, err)

	_, err = x509svid.Parse(files["crt.pem"], files["key.pem"])
	require.NoError(t, err)
}

func Test_DriverAnnotationSanitization(t *testing.T) {
	badAnnotation := annotations.Prefix + "/customannotation"

	tests := map[string]struct {
		in          map[string]string
		expectErr   bool
		expectedOut map[string]string
	}{
		"bad annotations are removed": {
			in: map[string]string{
				badAnnotation:              "abc123",
				"example.com/myannotation": "should-not-be-removed",
			},
			expectErr: true,
			expectedOut: map[string]string{
				"example.com/myannotation": "should-not-be-removed",
			},
		},
		"good annotations don't produce an error": {
			in: map[string]string{
				"example.com/myannotation": "should-not-be-removed",
			},
			expectErr: false,
			expectedOut: map[string]string{
				"example.com/myannotation": "should-not-be-removed",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			out, err := sanitizeAnnotations(test.in)

			if err != nil != test.expectErr {
				t.Errorf("expectErr=%v but err=%v", test.expectErr, err)
			}

			if !reflect.DeepEqual(out, test.expectedOut) {
				t.Errorf("wanted out=%v but got %v", test.expectedOut, out)
			}
		})
	}
}

// Test_signRequest_SAN_criticality checks that signRequest marks the SAN
// extension (OID 2.5.29.17) critical only when the subject is empty (the
// SPIFFE SVID case), and leaves it non-critical otherwise — guarding against
// a regression into unconditionally marking SAN critical.
func Test_signRequest_SAN_criticality(t *testing.T) {
	const spiffeID = "spiffe://example.org/ns/default/sa/workload"

	spiffeURI, err := url.Parse(spiffeID)
	require.NoError(t, err)

	tests := map[string]struct {
		subject         pkix.Name
		wantSANCritical bool
	}{
		"empty subject (SPIFFE SVID) marks SAN critical": {
			subject:         pkix.Name{},
			wantSANCritical: true,
		},
		"non-empty subject leaves SAN non-critical": {
			subject:         pkix.Name{CommonName: "example-workload"},
			wantSANCritical: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			require.NoError(t, err)

			csrTemplate := &x509.CertificateRequest{
				Subject: test.subject,
				URIs:    []*url.URL{spiffeURI},
			}

			csrPEM, err := signRequest(metadata.Metadata{}, key, csrTemplate)
			require.NoError(t, err)

			block, _ := pem.Decode(csrPEM)
			require.NotNil(t, block, "PEM decode must succeed")

			csr, err := x509.ParseCertificateRequest(block.Bytes)
			require.NoError(t, err)

			sanOID := asn1.ObjectIdentifier{2, 5, 29, 17}

			var sanExt *pkix.Extension
			for i := range csr.Extensions {
				if csr.Extensions[i].Id.Equal(sanOID) {
					sanExt = &csr.Extensions[i]
					break
				}
			}
			require.NotNil(t, sanExt, "CSR must contain the SAN extension (OID 2.5.29.17)")
			require.Equal(t, test.wantSANCritical, sanExt.Critical)

			require.Len(t, csr.URIs, 1)
			require.Equal(t, spiffeID, csr.URIs[0].String())
		})
	}
}
