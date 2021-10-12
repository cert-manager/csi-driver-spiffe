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
	"crypto/ecdsa"
	"errors"
	"fmt"
	"time"

	cmapi "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"github.com/jetstack/cert-manager/pkg/util"
	utilpki "github.com/jetstack/cert-manager/pkg/util/pki"
)

var (
	requiedUsages = []cmapi.KeyUsage{
		cmapi.UsageKeyEncipherment,
		cmapi.UsageDigitalSignature,
		cmapi.UsageClientAuth,
		cmapi.UsageServerAuth,
	}
)

// Interface is the Evaluator which is used for determining whether a
// CertificateRequest should be approved or denied.
type Interface interface {
	Evaluate(*cmapi.CertificateRequest) error
}

// Options is the options to configure the evaluator.
type Options struct {
	// TrustDomain is the trust domain that will be asserted when evaluating
	// CertificateRequests URI SANs.
	TrustDomain string

	// CertificateRequestDuration is the duration that users _must_ request for,
	// else the request will be denied.
	CertificateRequestDuration time.Duration
}

// internal is the internal implementation of the evaluator that should be used
// when running the approver controller.
type internal struct {
	// trustDomain is the trust domain that will be asserted when evaluating
	// CertificateRequests URI SANs.
	trustDomain string

	// certificateRequestDuration is the duration that users _must_ request for,
	// else the request will be denied.
	certificateRequestDuration time.Duration
}

// New constructs a new evaluator.
func New(opts Options) Interface {
	return &internal{
		trustDomain:                opts.TrustDomain,
		certificateRequestDuration: opts.CertificateRequestDuration,
	}
}

// Evaluate evaluates whether a CertificateRequest should be approved or
// denied. A CertificateRequest should be denied if this function returns an
// error, should be approved otherwise.
func (i *internal) Evaluate(req *cmapi.CertificateRequest) error {
	csr, err := utilpki.DecodeX509CertificateRequestBytes(req.Spec.Request)
	if err != nil {
		return fmt.Errorf("failed to parse request: %w", err)
	}

	if req.Spec.Duration == nil || req.Spec.Duration.Duration != i.certificateRequestDuration {
		return fmt.Errorf("requested certificate doesn't match required, required=%q got=%v",
			i.certificateRequestDuration.String(), req.Spec.Duration)
	}

	if err := csr.CheckSignature(); err != nil {
		return fmt.Errorf("signature check failed for csr: %w", err)
	}

	// if the csr contains any other options set, error
	if len(csr.DNSNames) > 0 || len(csr.IPAddresses) > 0 ||
		len(csr.Subject.CommonName) > 0 || len(csr.EmailAddresses) > 0 {
		return fmt.Errorf("forbidden extensions, DNS=%q IPs=%q CommonName=%q Emails=%q",
			csr.DNSNames, csr.IPAddresses, csr.Subject.CommonName, csr.EmailAddresses)
	}

	if ecdsapub, ok := csr.PublicKey.(*ecdsa.PublicKey); !ok || ecdsapub.Curve.Params().BitSize != 521 {
		return errors.New("forbidden key used by requestor, expecting ECDSA P521")
	}

	if err := validateCSRExtentions(csr); err != nil {
		return err
	}

	if req.Spec.IsCA {
		return errors.New("request contains spec.isCA=true")
	}

	if !util.EqualKeyUsagesUnsorted(req.Spec.Usages, requiedUsages) {
		return fmt.Errorf("request contains wrong usages, exp=%v got=%v", requiedUsages, req.Spec.Usages)
	}

	if err := i.validateIdentity(csr, req.Spec.Username); err != nil {
		return err
	}

	return nil
}
