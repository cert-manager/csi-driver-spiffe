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

package options

import (
	"time"

	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/spf13/pflag"

	"github.com/cert-manager/csi-driver-spiffe/internal/flags"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// Options are the CSI Approver flag options.
type Options struct {
	*flags.Flags

	// CertManager are options specific to created cert-manager
	// CertificateRequests.
	CertManager OptionsCertManager

	// Controller are options specific to the controller.
	Controller OptionsController
}

// OptionsController are options specific to the Kubernetes controller.
type OptionsController struct {
	// ReadyzAddress is the TCP address for exposing the HTTP readiness probe
	// which will be served on the HTTP path '/readyz'.
	ReadyzAddress string

	// MetricsAddress is the TCP address for exposing HTTP Prometheus metrics
	// which will be served on the HTTP path '/metrics'. The value "0" will
	// disable exposing metrics.
	MetricsAddress string

	// LeaderElectionNamespace is the namespace that the approver controller will
	// lease election in.
	LeaderElectionNamespace string
}

// OptionsCertManager are options specific to cert-manager and the evaluator.
type OptionsCertManager struct {
	// TrustDomain is the Trust Domain the evaluator will enforce requests request
	// for.
	TrustDomain string

	// CertificateRequestDuration is the duration the evaluator will enforce
	// CertificateRequest request for.
	CertificateRequestDuration time.Duration

	// IssuerRef is the issuer reference that will be used to match on created
	// CertificateRequests.
	IssuerRef cmmeta.ObjectReference
}

func New() *Options {
	o := new(Options)
	o.Flags = flags.New().
		Add("cert-manager", o.addCertManagerFlags).
		Add("Controller", o.addControllerFlags)
	return o
}

func (o *Options) addCertManagerFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.CertManager.TrustDomain, "trust-domain", "cluster.local",
		"The trust domain this approver ensures is present on requests.")

	fs.DurationVar(&o.CertManager.CertificateRequestDuration, "certificate-request-duration", time.Hour,
		"The duration which is enforced for requests to have.")

	fs.StringVar(&o.CertManager.IssuerRef.Name, "issuer-name", "my-spiffe-ca",
		"Name of issuer which is matched against to evaluate on.")
	fs.StringVar(&o.CertManager.IssuerRef.Kind, "issuer-kind", "ClusterIssuer",
		"Kind of issuer which is matched against to evaluate on.")
	fs.StringVar(&o.CertManager.IssuerRef.Group, "issuer-group", "cert-manager.io",
		"Group of issuer which is matched against to evaluate on.")
}

func (o *Options) addControllerFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Controller.LeaderElectionNamespace,
		"leader-election-namespace", "cert-manager",
		"Namespace to use for controller leader election.")

	fs.StringVar(&o.Controller.ReadyzAddress, "readiness-probe-bind-address", ":6060",
		"TCP address for exposing the HTTP readiness probe which will be served on "+
			"the HTTP path '/readyz'.")

	fs.StringVar(&o.Controller.MetricsAddress, "metrics-bind-address", ":9402",
		"TCP address for exposing HTTP Prometheus metrics which will be served on the "+
			"HTTP path '/metrics'. The value \"0\" will disable exposing metrics.")
}
