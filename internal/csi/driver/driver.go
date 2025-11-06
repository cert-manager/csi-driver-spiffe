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
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"strings"
	"sync"
	"time"

	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	cmclient "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	"github.com/cert-manager/csi-lib/driver"
	"github.com/cert-manager/csi-lib/manager"
	"github.com/cert-manager/csi-lib/manager/util"
	"github.com/cert-manager/csi-lib/metadata"
	"github.com/cert-manager/csi-lib/storage"
	jose "github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cert-manager/csi-driver-spiffe/internal/annotations"
	"github.com/cert-manager/csi-driver-spiffe/internal/csi/rootca"
	"github.com/cert-manager/csi-driver-spiffe/internal/version"
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

	// CertificateRequestAnnotations are annotations that are to be added to certificate requests created by the driver
	CertificateRequestAnnotations map[string]string

	// CertificateRequestDuration is the duration CertificateRequests will be
	// requested with.
	// Defaults to 1 hour if empty.
	CertificateRequestDuration time.Duration

	// IssuerRef is the IssuerRef used when creating CertificateRequests.
	IssuerRef *cmmeta.ObjectReference

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

	// IssuanceConfigMapName is the name of the ConfigMap to watch for issuance configuration.
	IssuanceConfigMapName string

	// IssuanceConfigMapNamespace is the namespace of the ConfigMap to watch for issuance configuration
	IssuanceConfigMapNamespace string
}

// Driver is used for running the actual CSI driver. Driver will respond to
// NodePubishVolume events, and attempt to sign SPIFFE certificates for
// mounting pod's identity.
type Driver struct {
	// log is the Driver logger.
	log logr.Logger

	// trustDomain is the trust domain that will form pod identities.
	trustDomain string

	// certificateRequestAnnotations are annotations that are to be added to certificate requests created by the driver
	certificateRequestAnnotations map[string]string

	// certificateRequestDuration is the duration which will be set of all
	// created CertificateRequests.
	certificateRequestDuration time.Duration

	// activeIssuerRef is the issuerRef that will be set on all created CertificateRequests.
	// Can be changed at runtime via runtime configuration (i.e. reading from a ConfigMap)
	// Not to be confused with originalIssuerRef, which is an issuerRef optionally passed in
	// via CLI args.
	activeIssuerRef *cmmeta.ObjectReference

	// originalIssuerRef is the issuerRef passed into the driver at startup. This will be used
	// if no runtime configuration (ConfigMap configuration) is found, or if the ConfigMap for
	// runtime configuration is deleted.
	originalIssuerRef *cmmeta.ObjectReference

	// activeIssuerRefMutex is used to control changes to the activeIssuerRef which can happen
	// concurrently with a request to issue a new cert
	activeIssuerRefMutex *sync.RWMutex

	// certFileName, keyFileName, caFileName are the names used when writing file
	// to volumes.
	certFileName, keyFileName, caFileName string

	// rootCAs provides the root CA certificates to write to file. No CA file is
	// written if this is nil.
	rootCAs rootca.Interface

	// driver is the csi-lib implementation of a cert-manager CSI driver.
	driver *driver.Driver

	// store is the csi-lib implementation of a cert-manager CSI storage manager.
	store storage.Interface

	// camanager is used to update all managed volumes with the current root CA
	// certificates PEM.
	camanager *camanager

	// kubernetesClient is used to watch ConfigMaps for issuance configuration
	kubernetesClient client.WithWatch

	// issuanceConfigMapName is the name of a ConfigMap which will be
	// watched for issuance configuration at runtime
	issuanceConfigMapName string

	// issuanceConfigMapNamespace is the name of a ConfigMap which will be
	// watched for issuance configuration at runtime
	issuanceConfigMapNamespace string
}

// New constructs a new Driver instance.
func New(log logr.Logger, opts Options) (*Driver, error) {
	sanitizedAnnotations, err := sanitizeAnnotations(opts.CertificateRequestAnnotations)
	if err != nil {
		log.Error(err, "some custom annotations were removed")
		// don't exit, not a fatal error as sanitizeAnnotations will trim bad annotations
	}

	originalIssuerRef, err := handleOriginalIssuerRef(opts.IssuerRef)
	if err != nil && err != errNoOriginalIssuer {
		return nil, err
	}

	if originalIssuerRef == nil && (opts.IssuanceConfigMapName == "" || opts.IssuanceConfigMapNamespace == "") {
		// if no install-time issuer was configured, runtime issuance details are not optional
		return nil, fmt.Errorf("runtime issuance configuration is required if no issuer is provided at startup")
	}

	d := &Driver{
		log:          log.WithName("csi"),
		trustDomain:  opts.TrustDomain,
		certFileName: opts.CertificateFileName,
		keyFileName:  opts.KeyFileName,

		// we check if we can set activeIssuerRef later
		activeIssuerRef:   nil,
		originalIssuerRef: originalIssuerRef,

		activeIssuerRefMutex: &sync.RWMutex{},

		rootCAs: opts.RootCAs,

		certificateRequestDuration:    opts.CertificateRequestDuration,
		certificateRequestAnnotations: sanitizedAnnotations,

		issuanceConfigMapName:      opts.IssuanceConfigMapName,
		issuanceConfigMapNamespace: opts.IssuanceConfigMapNamespace,
	}

	if d.originalIssuerRef != nil {
		d.activeIssuerRef = d.originalIssuerRef
	}

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

	store, err := storage.NewFilesystem(d.log, opts.DataRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to setup filesystem: %w", err)
	}

	// Used by clients to set the stored file's file-system group before
	// mounting.
	store.FSGroupVolumeAttributeKey = "spiffe.csi.cert-manager.io/fs-group"

	d.store = store
	d.camanager = newCAManager(log, store, opts.RootCAs,
		opts.CertificateFileName, opts.KeyFileName, opts.CAFileName)

	cmclient, err := cmclient.NewForConfig(opts.RestConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build cert-manager client: %w", err)
	}

	k8sClient, err := client.NewWithWatch(opts.RestConfig, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to build kubernetes watcher client: %w", err)
	}

	d.kubernetesClient = k8sClient

	mngrLog := d.log.WithName("manager")
	d.driver, err = driver.New(opts.Endpoint, d.log.WithName("driver"), driver.Options{
		DriverName:    opts.DriverName,
		DriverVersion: version.AppVersion,
		NodeID:        opts.NodeID,
		Store:         d.store,
		Manager: manager.NewManagerOrDie(manager.Options{
			Client: cmclient,
			// Use Pod's service account to request CertificateRequests.
			ClientForMetadata:    util.ClientForMetadataTokenRequestEmptyAud(opts.RestConfig),
			MaxRequestsPerVolume: 1,
			MetadataReader:       d.store,
			Clock:                clock.RealClock{},
			Log:                  &mngrLog,
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

// watchRuntimeConfigurationSource should be called in a goroutine to watch a ConfigMap for runtime configuration
func (d *Driver) watchRuntimeConfigurationSource(ctx context.Context) {
	logger := d.log.WithName("runtime-config-watcher").WithValues("config-map-name", d.issuanceConfigMapName, "config-map-namespace", d.issuanceConfigMapNamespace)

LOOP:
	for {
		logger.Info("Starting / restarting watcher for runtime configuration")
		cmList := &corev1.ConfigMapList{}

		// First create a watcher. This is in a labelled loop in case the watcher dies for some reason
		// while we're running - in that case, we don't want to give up entirely on watching for runtime config
		// but instead we want to recreate the watcher.

		watcher, err := d.kubernetesClient.Watch(ctx, cmList, &client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector("metadata.name", d.issuanceConfigMapName),
			Namespace:     d.issuanceConfigMapNamespace,
		})

		if err != nil {
			logger.Error(err, "Failed to create ConfigMap watcher; will retry in 5s")
			time.Sleep(5 * time.Second)
			continue
		}

		for {
			// Now loop indefinitely until the main context cancels or we get an event to process.
			// If the main context cancels, we break out of the outer loop and this function returns.
			// If we get an event, we first check whether the channel closed. If so, we recreate the watcher by continuing
			// the outer loop.
			select {
			case <-ctx.Done():
				logger.Info("Received context cancellation, shutting down runtime configuration watcher")
				watcher.Stop()
				break LOOP

			case event, open := <-watcher.ResultChan():
				if !open {
					logger.Info("Received closed channel from ConfigMap watcher, will recreate")
					watcher.Stop()
					continue LOOP
				}

				switch event.Type {
				case watch.Deleted:
					d.handleRuntimeConfigIssuerDeletion(logger)

				case watch.Added:
					err := d.handleRuntimeConfigIssuerChange(logger, event)
					if err != nil {
						logger.Error(err, "Failed to handle new runtime configuration for issuerRef")
					}

				case watch.Modified:
					err := d.handleRuntimeConfigIssuerChange(logger, event)
					if err != nil {
						logger.Error(err, "Failed to handle runtime configuration issuerRef change")
					}

				case watch.Bookmark:
					// Ignore

				case watch.Error:
					err, ok := event.Object.(error)
					if !ok {
						logger.Error(nil, "Got an error event when watching runtime configuration but unable to determine further information")
					} else {
						logger.Error(err, "Got an error event when watching runtime configuration")
					}

				default:
					logger.Info("Got unknown event for runtime configuration ConfigMap; ignoring", "event-type", string(event.Type))
				}
			}
		}
	}

	logger.Info("Stopped runtime configuration watcher")
}

const (
	issuerNameKey  = "issuer-name"
	issuerKindKey  = "issuer-kind"
	issuerGroupKey = "issuer-group"
)

func (d *Driver) handleRuntimeConfigIssuerChange(logger logr.Logger, event watch.Event) error {
	d.activeIssuerRefMutex.Lock()
	defer d.activeIssuerRefMutex.Unlock()

	cm, ok := event.Object.(*corev1.ConfigMap)
	if !ok {
		return fmt.Errorf("got unexpected type for runtime configuration source; this is likely a programming error")
	}

	issuerRef := &cmmeta.ObjectReference{}

	var dataErrs []error
	var exists bool

	issuerRef.Name, exists = cm.Data[issuerNameKey]
	if !exists || len(issuerRef.Name) == 0 {
		dataErrs = append(dataErrs, fmt.Errorf("missing key/value in ConfigMap data: %s", issuerNameKey))
	}

	issuerRef.Kind, exists = cm.Data[issuerKindKey]
	if !exists || len(issuerRef.Kind) == 0 {
		dataErrs = append(dataErrs, fmt.Errorf("missing key/value in ConfigMap data: %s", issuerKindKey))
	}

	issuerRef.Group, exists = cm.Data[issuerGroupKey]
	if !exists || len(issuerRef.Group) == 0 {
		dataErrs = append(dataErrs, fmt.Errorf("missing key/value in ConfigMap data; %s", issuerGroupKey))
	}

	if len(dataErrs) > 0 {
		return errors.Join(dataErrs...)
	}

	// we now have a full issuerRef
	// TODO: check if the issuer exists by querying for the CRD?

	d.activeIssuerRef = issuerRef

	logger.Info("Changed active issuerRef in response to runtime configuration ConfigMap", "issuer-name", d.activeIssuerRef.Name, "issuer-kind", d.activeIssuerRef.Kind, "issuer-group", d.activeIssuerRef.Group)

	return nil
}

func (d *Driver) handleRuntimeConfigIssuerDeletion(logger logr.Logger) {
	d.activeIssuerRefMutex.Lock()
	defer d.activeIssuerRefMutex.Unlock()

	if d.originalIssuerRef == nil {
		logger.Info("Runtime issuance configuration was deleted and no issuerRef was configured at install time; issuance will fail until runtime configuration is reinstated")
		d.activeIssuerRef = nil
		return
	}

	logger.Info("Runtime issuance configuration was deleted; issuance will revert to original issuerRef configured at install time")

	d.activeIssuerRef = d.originalIssuerRef
}

// Run is a blocking func that runs the CSI driver.
func (d *Driver) Run(ctx context.Context) error {
	var wg sync.WaitGroup

	go func() {
		<-ctx.Done()
		d.driver.Stop()
	}()

	wg.Go(func() {
		updateRetryPeriod := time.Second * 5
		d.camanager.run(ctx, updateRetryPeriod)
	})

	if d.hasRuntimeConfiguration() {
		wg.Go(func() {
			d.watchRuntimeConfigurationSource(ctx)
		})
	}

	wg.Add(1)
	var err error
	go func() {
		defer wg.Done()
		err = d.driver.Run()
	}()

	wg.Wait()
	return err
}

// validSigningAlgs is a list of algorithms that the Kubernetes API server allows for signing
// service account tokens. It's taken from [1].
// If in the future the upstream list changes, we may start to see issues being raised
// since we'll reject otherwise-valid algorithms, and this list may then need to be updated.
// [1] https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/plugin/pkg/authenticator/token/oidc/oidc.go#L229-L240
var validSigningAlgs = []jose.SignatureAlgorithm{
	jose.RS256,
	jose.RS384,
	jose.RS512,
	jose.ES256,
	jose.ES384,
	jose.ES512,
	jose.PS256,
	jose.PS384,
	jose.PS512,
}

// generateRequest will generate a SPIFFE manager.CertificateRequestBundle
// based upon the identity contained in the metadata service account token.
func (d *Driver) generateRequest(meta metadata.Metadata) (*manager.CertificateRequestBundle, error) {
	d.activeIssuerRefMutex.RLock()
	defer d.activeIssuerRefMutex.RUnlock()

	if d.activeIssuerRef == nil {
		return nil, fmt.Errorf("no issuerRef is currently active for csi-driver-spiffe; configure one using runtime configuration")
	}

	// Extract the service account token from the volume metadata in order to
	// derive the service account, and thus identity of the pod.
	token, err := util.EmptyAudienceTokenFromMetadata(meta)
	if err != nil {
		return nil, err
	}

	// see comment for validSigningAlgs for more details on how the algorithms were chosen
	jwttoken, err := jwt.ParseSigned(token, validSigningAlgs)
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

	crAnnotations := map[string]string{
		annotations.SPIFFEIdentityAnnnotationKey: spiffeID,
	}

	maps.Copy(crAnnotations, d.certificateRequestAnnotations)

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
		IssuerRef:   *d.activeIssuerRef,
		Annotations: crAnnotations,
	}, nil
}

// writeKeypair writes the private key and certificate chain to file that will
// be mounted into the pod.
func (d *Driver) writeKeypair(meta metadata.Metadata, key crypto.PrivateKey, chain []byte, _ []byte) error {
	pemBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return fmt.Errorf("failed to marshal ECDSA private key for PEM encoding: %w", err)
	}

	keyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "PRIVATE KEY",
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

// hasRuntimeConfiguration returns true if runtime configuration has been correctly
// configured with a ConfigMap name and namespace, and false otherwise.
func (d *Driver) hasRuntimeConfiguration() bool {
	return d.issuanceConfigMapName != "" && d.issuanceConfigMapNamespace != ""
}

func sanitizeAnnotations(in map[string]string) (map[string]string, error) {
	out := map[string]string{}

	var errs []error

	for key, value := range in {
		if strings.HasPrefix(key, annotations.Prefix) {
			errs = append(errs, fmt.Errorf("custom annotation %q was not valid; must not begin with %s", key, annotations.Prefix))
			continue
		}

		out[key] = value
	}

	return out, errors.Join(errs...)
}

var errNoOriginalIssuer = fmt.Errorf("no original issuer was provided")

func handleOriginalIssuerRef(in *cmmeta.ObjectReference) (*cmmeta.ObjectReference, error) {
	if in == nil {
		return nil, errNoOriginalIssuer
	}

	if in.Name == "" && in.Kind == "" && in.Group == "" {
		return nil, errNoOriginalIssuer
	}

	if in.Name == "" {
		return nil, fmt.Errorf("issuerRef.Name is a required field if any field is set for issuerRef")
	}

	if in.Kind == "" {
		return nil, fmt.Errorf("issuerRef.Kind is a required field if any field is set for issuerRef")
	}

	if in.Group == "" {
		return nil, fmt.Errorf("issuerRef.Group is a required field if any field is set for issuerRef")
	}

	return in, nil
}
