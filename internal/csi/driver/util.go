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
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/cert-manager/csi-lib/metadata"

	cmpki "github.com/cert-manager/cert-manager/pkg/util/pki"
)

// generatePrivateKey generates an ECDSA private key, which is the only currently supported type
func generatePrivateKey(_ metadata.Metadata) (crypto.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

// signRequest will sign a given X.509 certificate signing request with the given key.
// When URI SANs are present and the subject is empty (as with SPIFFE SVIDs),
// RFC 5280 §4.2.1.6 requires the SAN extension to be marked critical.
// Go's x509.CreateCertificateRequest does not do this, so we marshal the
// SAN extension ourselves via cert-manager's MarshalSANs helper.
func signRequest(_ metadata.Metadata, key crypto.PrivateKey, request *x509.CertificateRequest) ([]byte, error) {
	if len(request.URIs) > 0 {
		uriStrings := make([]string, len(request.URIs))
		for i, u := range request.URIs {
			uriStrings[i] = u.String()
		}
		sans := cmpki.GeneralNames{UniformResourceIdentifiers: uriStrings}
		// hasSubject=false → Critical=true
		sanExt, err := cmpki.MarshalSANs(sans, false)
		if err != nil {
			return nil, err
		}
		request.ExtraExtensions = append(request.ExtraExtensions, sanExt)
		// Clear URIs to avoid a duplicate SAN extension from the stdlib.
		request.URIs = nil
	}

	csrDer, err := x509.CreateCertificateRequest(rand.Reader, request, key)
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrDer,
	}), nil
}

// calculateNextIssuanceTime returns the time when the certificate should be
// renewed. This will be 2/3rds the duration of the leaf certificate's validity period.
func calculateNextIssuanceTime(chain []byte) (time.Time, error) {
	block, _ := pem.Decode(chain)

	crt, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing issued certificate: %w", err)
	}

	// Renew once a certificate is 2/3rds of the way through its actual lifetime.
	actualDuration := crt.NotAfter.Sub(crt.NotBefore)

	renewBeforeNotAfter := actualDuration / 3

	return crt.NotAfter.Add(-renewBeforeNotAfter), nil
}
