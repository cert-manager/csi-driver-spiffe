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
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"reflect"
	"testing"

	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	utilpki "github.com/cert-manager/cert-manager/pkg/util/pki"
	"github.com/cert-manager/csi-lib/metadata"
	"github.com/cert-manager/csi-lib/storage"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/stretchr/testify/require"

	"github.com/cert-manager/csi-driver-spiffe/internal/annotations"
	"github.com/cert-manager/csi-driver-spiffe/internal/csi/rootca"
)

// Ensure writeKeyPair is compatible with go-spiffe/v2 x509svid.Parse.
func Test_writeKeyPair(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
	})

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
	rootCAs := rootca.NewMemory(ctx, ch)
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
