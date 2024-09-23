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

// Options are the CSI Driver flag options.
type Options struct {
	*flags.Flags

	// Driver are options specific to the driver itself.
	Driver OptionsDriver

	// CertManager are options specific to created cert-manager
	// CertificateRequests.
	CertManager OptionsCertManager

	// Volume are options specific to mounted volumes.
	Volume OptionsVolume
}

// OptionsDriver are options specific to the CSI driver itself.
type OptionsDriver struct {
	// NodeID is the name of the node the driver is running on.
	NodeID string

	// DataRoot is the path to the in-memory data directory used to store data.
	DataRoot string

	// Endpoint is the endpoint which is used to listen for gRPC requests.
	Endpoint string
}

// OptionsCertManager is options specific to cert-manager CertificateRequests.
type OptionsCertManager struct {
	// IssuanceConfigMapName is the name of a ConfigMap to watch for configuration options. The ConfigMap is expected to be in the same namespace as the csi-driver-spiffe pod.
	IssuanceConfigMapName string

	// IssuanceConfigMapNamespace is the namespace where the runtime configuration ConfigMap is located
	IssuanceConfigMapNamespace string

	// TrustDomain is the trust domain of this SPIFFE PKI. The TrustDomain will
	// appear in signed certificate's URI SANs.
	TrustDomain string

	// CertificateRequestAnnotations are annotations that are to be added to certificate requests created by the driver
	CertificateRequestAnnotations map[string]string

	// CertificateRequestDuration is the duration CertificateRequests will be
	// requested with.
	CertificateRequestDuration time.Duration

	// IncludeDnsSan is set to true to indicate that the service account name should be included as a DNS SAN
	IncludeDnsSan string

	// IssuerRef is the IssuerRef used when creating CertificateRequests.
	IssuerRef cmmeta.ObjectReference
}

// OptionsVolume is options specific to mounted volumes.
type OptionsVolume struct {
	// CertificateFileName is the name of the file that the signed certificate
	// will be written to inside the Pod's volume.
	CertificateFileName string

	// KeyFileName is the name of the file that the private key will be written
	// to inside the Pod's volume.
	// Default to `tls.key` if empty.
	KeyFileName string

	// FileName is the name of the file that the root CA certificates will be
	// written to inside the Pod's volume. Ignored if SourceCABundleFile is not
	// defined.
	CAFileName string

	// SourceCABundleFile is the file path location containing a bundle of PEM
	// encoded X.509 root CA certificates that will be written to managed volumes
	// at the CSICAFileName path. No CAs will be written if this is empty.
	SourceCABundleFile string
}

func New() *Options {
	o := new(Options)
	o.Flags = flags.New().
		Add("Driver", o.addDriverFlags).
		Add("cert-manager", o.addCertManagerFlags).
		Add("Volume", o.addVolumeFlags)

	return o
}

func (o *Options) addDriverFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Driver.NodeID, "node-id", "",
		"Name of the node the driver is running on.")
	fs.StringVar(&o.Driver.DataRoot, "data-root", "",
		"Path to the in-memory data directory used to store data.")
	fs.StringVar(&o.Driver.Endpoint, "endpoint", "",
		"Path to the unix socket used to listen for gRPC requests.")
}

func (o *Options) addCertManagerFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.CertManager.IssuanceConfigMapName, "runtime-issuance-config-map-name", "", "Name of a ConfigMap to watch at runtime for issuer details. If such a ConfigMap is found, overrides issuer-name, issuer-kind and issuer-group")

	fs.StringVar(&o.CertManager.IssuanceConfigMapNamespace, "runtime-issuance-config-map-namespace", "", "Namespace for ConfigMap to be watched at runtime for issuer details")

	fs.StringVar(&o.CertManager.TrustDomain, "trust-domain", "cluster.local",
		"The trust domain that will be requested for on created CertificateRequests.")
	fs.DurationVar(&o.CertManager.CertificateRequestDuration, "certificate-request-duration", time.Hour,
		"The duration that created CertificateRequests will use.")
	fs.StringVar(&o.CertManager.IncludeDnsSan, "include-dns-san", "false", "include the service account name as a DNS SAN")

	fs.StringToStringVar(&o.CertManager.CertificateRequestAnnotations, "extra-certificate-request-annotations", map[string]string{},
		"Comma-separated list of extra annotations to add to certificate requests e.g '--extra-certificate-request-annotations=hello=world,test=annotation'")

	fs.StringVar(&o.CertManager.IssuerRef.Name, "issuer-name", "my-spiffe-ca",
		"Name of the issuer that CertificateRequests will be created for.")
	fs.StringVar(&o.CertManager.IssuerRef.Kind, "issuer-kind", "ClusterIssuer",
		"Kind of the issuer that CertificateRequests will be created for.")
	fs.StringVar(&o.CertManager.IssuerRef.Group, "issuer-group", "cert-manager.io",
		"Group of the issuer that CertificateRequests will be created for.")
}

func (o *Options) addVolumeFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Volume.CertificateFileName, "file-name-certificate", "tls.crt",
		"The file name that signed certificates will be written to within the pod's volume directory.")
	fs.StringVar(&o.Volume.KeyFileName, "file-name-key", "tls.key",
		"The file name that the certificate's private key will be written to within the pod's volume directory.")
	fs.StringVar(&o.Volume.CAFileName, "file-name-ca", "ca.crt",
		"The file name that the certificate's private key will be written to within the pod's volume directory.")

	fs.StringVar(&o.Volume.SourceCABundleFile, "source-ca-bundle", "",
		"File path that is read by the driver which will be written to all managed "+
			"volumes to the file location inside volumes defined in --file-name-ca. If "+
			"undefined, no CA file is written to volumes.")
}
