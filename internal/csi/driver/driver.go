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
	"crypto"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/url"
	"time"

	"github.com/cert-manager/csi-lib/driver"
	"github.com/cert-manager/csi-lib/manager"
	"github.com/cert-manager/csi-lib/manager/util"
	"github.com/cert-manager/csi-lib/metadata"
	"github.com/cert-manager/csi-lib/storage"
	"github.com/go-logr/logr"
	cmapi "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	cmclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	"gopkg.in/square/go-jose.v2/jwt"
	"k8s.io/client-go/rest"
	"k8s.io/utils/clock"

	"github.com/cert-manager/csi-driver-spiffe/internal/csi/rootca"
)

// Options holds the Options needed for the CSI driver.
type Options struct {
	// DriverName is the driver name as installed in Kubernetes.
	DriverName string

	// NodeID is the name of the node the driver is running on.
	NodeID string

	// DataRoot is the path to the in-memory data directory used to store data.
	DataRoot string

	// Endpoint is the endpoint which is used to listen for gRPC requests.
	Endpoint string

	// TrustDomain is the trust domain of this SPIFFE PKI. The TrustDomain will
	// appear in signed certificate's URI SANs.
	TrustDomain string

	// CertificateRequestDuration is the duration CertificateRequests will be
	// requested with.
	// Defaults to 1 hour if empty.
	CertificateRequestDuration time.Duration

	// IssuerRef is the IssuerRef used when creating CertificateRequests.
	IssuerRef cmmeta.ObjectReference

	// CertificateFileName is the name of the file that the signed certificate
	// will be written to inside the Pod's volume.
	// Default to `tls.crt` if empty.
	CertificateFileName string

	// KeyFileName is the name of the file that the private key will be written
	// to inside the Pod's volume.
	// Default to `tls.key` if empty.
	KeyFileName string

	// CAFileName is the name of the file that the root CA certificates will be
	// written to inside the Pod's volume. Ignored if RootCAs is nil.
	CAFileName string

	// RestConfig is used for interacting with the Kubernetes API server.
	RestConfig *rest.Config

	// RootCAs is optionally used to write root CA certificate data to Pod's
	// volume. If nil, no root CA data is written to Pod's volume. If defined,
	// root CA data will be written to the file with the name defined in
	// CAFileName. If the root CA certificate data changes, all managed volume's
	// file will be updated.
	RootCAs rootca.Interface
}

// Driver is used for running the actual CSI driver. Driver will respond to
// NodePubishVolume events, and attempt to sign SPIFFE certificates for
// mounting pod's identity.
type Driver struct {
	// log is the Driver logger.
	log logr.Logger

	// trustDomain is the trust domain that will form pod identities.
	trustDomain string

	// certificateRequestDuration is the duration which will be set of all
	// created CertificateRequests.
	certificateRequestDuration time.Duration

	// issuerRef is the issuerRef that will be set on all created
	// CertificateRequests.
	issuerRef cmmeta.ObjectReference

	// certFileName, keyFileName, caFileName are the names used when writing file
	// to volumes.
	certFileName, keyFileName, caFileName string

	// rootCAs provides the root CA certificates to write to file. No CA file is
	// written if this is nil.
	rootCAs rootca.Interface

	// driver is the csi-lib implementation of a cert-manager CSI driver.
	driver *driver.Driver

	// store is the csi-lib implementation of a cert-manager CSI storage manager.
	store *storage.Filesystem

	// updateRootCAFiles is a func to update all managed volumes with the current
	// root CA certificates PEM. Used for testing.
	updateRootCAFilesFn func() error
}

// New constructs a new Driver instance.
func New(log logr.Logger, opts Options) (*Driver, error) {
	d := &Driver{
		log:          log.WithName("csi"),
		trustDomain:  opts.TrustDomain,
		certFileName: opts.CertificateFileName,
		keyFileName:  opts.KeyFileName,
		issuerRef:    opts.IssuerRef,
		rootCAs:      opts.RootCAs,

		certificateRequestDuration: opts.CertificateRequestDuration,
	}

	// Set sane defaults.
	if len(d.certFileName) == 0 {
		d.certFileName = "tls.crt"
	}
	if len(d.keyFileName) == 0 {
		d.keyFileName = "tls.key"
	}
	if len(d.caFileName) == 0 {
		d.caFileName = "ca.crt"
	}
	if d.certificateRequestDuration == 0 {
		d.certificateRequestDuration = time.Hour
	}
	d.updateRootCAFilesFn = d.updateRootCAFiles

	var err error
	d.store, err = storage.NewFilesystem(d.log, opts.DataRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to setup filesystem: %w", err)
	}
	// Used by clients to set the stored file's file-system group before
	// mounting.
	d.store.FSGroupVolumeAttributeKey = "spiffe.csi.cert-manager.io/fs-group"

	cmclient, err := cmclient.NewForConfig(opts.RestConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build cert-manager client: %w", err)
	}

	d.driver, err = driver.New(opts.Endpoint, d.log.WithName("driver"), driver.Options{
		DriverName:    opts.DriverName,
		DriverVersion: "v0.2.0",
		NodeID:        opts.NodeID,
		Store:         d.store,
		Manager: manager.NewManagerOrDie(manager.Options{
			Client: cmclient,
			// Use Pod's service account to request CertificateRequests.
			ClientForMetadata:    util.ClientForMetadataTokenRequestEmptyAud(opts.RestConfig),
			MaxRequestsPerVolume: 1,
			MetadataReader:       d.store,
			Clock:                clock.RealClock{},
			Log:                  d.log.WithName("manager"),
			NodeID:               opts.NodeID,
			GeneratePrivateKey:   generatePrivateKey,
			GenerateRequest:      d.generateRequest,
			SignRequest:          signRequest,
			WriteKeypair:         d.writeKeypair,
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to setup csi driver: %w", err)
	}

	return d, nil
}

// Run is a blocking func that run the CSI driver.
func (d *Driver) Run(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		d.driver.Stop()
	}()

	d.manageCAFiles(ctx, time.Second*5)
	return d.driver.Run()
}

// generateRequest will generate a SPIFFE manager.CertificateRequestBundle
// based upon the identity contained in the metadata service account token.
func (d *Driver) generateRequest(meta metadata.Metadata) (*manager.CertificateRequestBundle, error) {
	// Extract the service account token from the volume metadata in order to
	// derive the service account, and thus identity of the pod.
	token, err := util.EmptyAudienceTokenFromMetadata(meta)
	if err != nil {
		return nil, err
	}

	jwttoken, err := jwt.ParseSigned(token)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token request token: %w", err)
	}

	claims := struct {
		KubernetesIO struct {
			Namespace      string `json:"namespace"`
			ServiceAccount struct {
				Name string `json:"name"`
			} `json:"serviceaccount"`
		} `json:"kubernetes.io"`
	}{}

	// We don't need to verify the token since we will be using it against the
	// API server anyway which is the source of trust for auth by definition.
	if err := jwttoken.UnsafeClaimsWithoutVerification(&claims); err != nil {
		return nil, fmt.Errorf("failed to decode token request token: %w", err)
	}

	saName := claims.KubernetesIO.ServiceAccount.Name
	saNamespace := claims.KubernetesIO.Namespace
	if len(saName) == 0 || len(saNamespace) == 0 {
		return nil, fmt.Errorf("missing namespace or serviceaccount name in request token: %v", claims)
	}

	spiffeID := fmt.Sprintf("spiffe://%s/ns/%s/sa/%s", d.trustDomain, saNamespace, saName)
	uri, err := url.Parse(spiffeID)
	if err != nil {
		return nil, fmt.Errorf("internal error crafting X.509 URI, this is a bug, please report on GitHub: %w", err)
	}

	return &manager.CertificateRequestBundle{
		Request: &x509.CertificateRequest{
			URIs: []*url.URL{uri},
		},
		IsCA:      false,
		Namespace: saNamespace,
		Duration:  d.certificateRequestDuration,
		Usages: []cmapi.KeyUsage{
			cmapi.UsageDigitalSignature,
			cmapi.UsageKeyEncipherment,
			cmapi.UsageServerAuth,
			cmapi.UsageClientAuth,
		},
		IssuerRef:   d.issuerRef,
		Annotations: map[string]string{"spiffe.csi.cert-manager.io/identity": spiffeID},
	}, nil
}

// writeKeypair writes the private key and certificate chain to file that will
// be mounted into the pod.
func (d *Driver) writeKeypair(meta metadata.Metadata, key crypto.PrivateKey, chain []byte, _ []byte) error {
	pemBytes, err := x509.MarshalECPrivateKey(key.(*ecdsa.PrivateKey))
	if err != nil {
		return fmt.Errorf("failed to marshal ECDSA private key for PEM encoding: %w", err)
	}

	keyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: pemBytes,
		},
	)

	// Calculate the next issuance time before we write any data to file, so in
	// the cases where this errors, we are not left in a bad state.
	nextIssuanceTime, err := calculateNextIssuanceTime(chain)
	if err != nil {
		return fmt.Errorf("failed to calculate next issuance time: %w", err)
	}

	data := map[string][]byte{
		d.certFileName: chain,
		d.keyFileName:  keyPEM,
	}
	// If configured, write the CA certificates as defined in RootCAs.
	if d.rootCAs != nil {
		data[d.caFileName] = d.rootCAs.CertificatesPEM()
	}

	// Write data to the actual volume that gets mounted.
	if err := d.store.WriteFiles(meta, data); err != nil {
		return fmt.Errorf("writing data: %w", err)
	}

	meta.NextIssuanceTime = &nextIssuanceTime
	if err := d.store.WriteMetadata(meta.VolumeID, meta); err != nil {
		return fmt.Errorf("writing metadata: %w", err)
	}

	return nil
}
