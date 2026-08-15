/*
Copyright 2026 The cert-manager Authors.

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

	"github.com/stretchr/testify/require"

	"hegel.dev/go/hegel"
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
