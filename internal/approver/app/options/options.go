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
	// IssuanceConfigMapName is the name of a ConfigMap to watch for configuration options. The ConfigMap is expected to be in the same namespace as the csi-driver-spiffe pod.
	IssuanceConfigMapName string

	// IssuanceConfigMapNamespace is the namespace where the runtime configuration ConfigMap is located
	IssuanceConfigMapNamespace string

	// TrustDomain is the Trust Domain the evaluator will enforce requests request for.
	TrustDomain string

	// CertificateRequestDuration is the duration the evaluator will enforce
	// CertificateRequest request for.
	CertificateRequestDuration time.Duration

	// UseOwnServiceAccount, when true, changes the approval validation strategy.
	// Instead of verifying that the SPIFFE identity in the CSR matches the
	// requesting pod's ServiceAccount, the approver verifies that the requester
	// is the driver's own ServiceAccount (DriverServiceAccount).
	UseOwnServiceAccount bool

	// DriverServiceAccount is the full Kubernetes username of the CSI driver's
	// ServiceAccount (e.g. "system:serviceaccount:cert-manager:spiffe.csi.cert-manager.io").
	// Only used when UseOwnServiceAccount is true.
	DriverServiceAccount string

	// IssuerRef is the IssuerRef used when creating CertificateRequests.
	IssuerRef cmmeta.IssuerReference

	// AutoApproveNonSPIFFE enables the auto approval of non csi-driver-spiffe CertificateRequest resources. This allows
	// csi-driver-spiffe to act as a drop in replacement for the cert-manager approval controller.
	AutoApproveNonSPIFFE bool
}

func New() *Options {
	o := new(Options)
	o.Flags = flags.New().
		Add("cert-manager", o.addCertManagerFlags).
		Add("Controller", o.addControllerFlags)
	return o
}

func (o *Options) addCertManagerFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.CertManager.IssuanceConfigMapName, "runtime-issuance-config-map-name", "",
		"Name of a ConfigMap to watch at runtime for issuer details. If such a ConfigMap is found, overrides issuer-name, issuer-kind and issuer-group")

	fs.StringVar(&o.CertManager.IssuanceConfigMapNamespace, "runtime-issuance-config-map-namespace", "",
		"Namespace for ConfigMap to be watched at runtime for issuer details")

	fs.StringVar(&o.CertManager.TrustDomain, "trust-domain", "cluster.local",
		"The trust domain this approver ensures is present on requests.")

	fs.DurationVar(&o.CertManager.CertificateRequestDuration, "certificate-request-duration", time.Hour,
		"The duration which is enforced for requests to have.")

	fs.BoolVar(&o.CertManager.UseOwnServiceAccount, "use-own-service-account", false,
		"When true, the approver validates that CertificateRequests are made by the "+
			"driver's own ServiceAccount (--driver-service-account) rather than by the "+
			"mounting pod's ServiceAccount.")

	fs.StringVar(&o.CertManager.DriverServiceAccount, "driver-service-account", "",
		"Full Kubernetes username of the CSI driver's ServiceAccount "+
			"(e.g. \"system:serviceaccount:cert-manager:spiffe.csi.cert-manager.io\"). "+
			"Required when --use-own-service-account is true.")

	fs.StringVar(&o.CertManager.IssuerRef.Name, "issuer-name", "my-spiffe-ca",
		"Name of the issuer that CertificateRequests will be created for.")

	fs.StringVar(&o.CertManager.IssuerRef.Kind, "issuer-kind", "ClusterIssuer",
		"Kind of the issuer that CertificateRequests will be created for.")

	fs.StringVar(&o.CertManager.IssuerRef.Group, "issuer-group", "cert-manager.io",
		"Group of the issuer that CertificateRequests will be created for.")

	fs.BoolVar(&o.CertManager.AutoApproveNonSPIFFE, "auto-approve-non-spiffe", false,
		"Enables the auto approval of non csi-driver-spiffe CertificateRequest resources. This allows csi-driver-spiffe to act as a drop in replacement for the cert-manager approval controller.")
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
