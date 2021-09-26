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

	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	"github.com/spf13/pflag"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/cert-manager/csi-driver-spiffe/internal/flags"
)

// TODO
type Options struct {
	*flags.Flags

	// ReadyzPort if the port used to expose readiness endpoint.
	ReadyzPort int

	// ReadyzPath if the HTTP path used to expose readiness endpoint.
	ReadyzPath string

	// MetricsPort is the port for exposing Prometheus metrics on 0.0.0.0 on the
	// path '/metrics'.
	MetricsPort int

	// TODO
	LeaderElectionNamespace string

	// TODO
	TrustDomain string

	// TODO
	CertificateRequestDuration time.Duration

	// TODO
	IssuerRef cmmeta.ObjectReference
}

func New() *Options {
	o := new(Options)
	o.Flags = flags.New().Add("Approver", o.addApproverFlags)
	return o
}

func (o *Options) addApproverFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.LeaderElectionNamespace,
		"leader-election-namespace", "cert-manager",
		"Namespace to use for controller leader election.")

	fs.StringVar(&o.TrustDomain, "trust-domain", "cluster.local",
		"The trust domain this approver ensures is present on requests.")

	fs.DurationVar(&o.CertificateRequestDuration, "certificate-request-duration", time.Hour,
		"The duration which is enforced for requests to have.")

	fs.StringVar(&o.IssuerRef.Name, "issuer-name", "my-spiffe-ca",
		"Name of issuer which is matched against to evaluate on.")
	fs.StringVar(&o.IssuerRef.Kind, "issuer-kind", "ClusterIssuer",
		"Kind of issuer which is matched against to evaluate on.")
	fs.StringVar(&o.IssuerRef.Group, "issuer-group", "cert-manager.io",
		"Group of issuer which is matched against to evaluate on.")
}
