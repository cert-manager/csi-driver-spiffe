/*
Copyright 2023 The cert-manager Authors.

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
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_calculateNextIssuanceTime(t *testing.T) {
	notBefore := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	notAfter := time.Date(1970, time.January, 4, 0, 0, 0, 0, time.UTC)

	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &pk.PublicKey, pk)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	tests := map[string]struct {
		chain   []byte
		expTime time.Time
		expErr  bool
	}{
		"a valid chain renews 2/3rds through the leaf's lifetime": {
			chain:   certPEM,
			expTime: notBefore.AddDate(0, 0, 2),
		},
		"a nil chain is rejected": {
			chain:  nil,
			expErr: true,
		},
		"an empty chain is rejected": {
			chain:  []byte{},
			expErr: true,
		},
		"a chain that is not PEM is rejected": {
			chain:  []byte("this is not a PEM encoded certificate"),
			expErr: true,
		},
		"a truncated PEM block is rejected": {
			chain:  certPEM[:len(certPEM)/2],
			expErr: true,
		},
		"a PEM block whose contents are not DER is rejected": {
			chain:  pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("not DER")}),
			expErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			renewTime, err := calculateNextIssuanceTime(test.chain)
			assert.Equal(t, test.expErr, err != nil)
			assert.Equal(t, test.expTime, renewTime)
		})
	}
}
