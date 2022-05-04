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
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/cert-manager/csi-driver-spiffe/internal/csi/rootca"
	"github.com/cert-manager/csi-lib/storage"
	"github.com/go-logr/logr"
)

// camanager is a process responsible for distributing trust bundles to
// mounting pods.
type camanager struct {
	// log is the logger for camanager.
	log logr.Logger

	// store is the csi-lib file system storage implementation. Must by file
	// system in order to read volumes back from mounted pods.
	store *storage.Filesystem

	// rootCAs exposes the current trust bundle to be propagated, and signals
	// when a new trust bundle is available.
	rootCAs rootca.Interface

	// certFileName, keyFileName, caFileName are the names used when writing file
	// to volumes.
	certFileName, keyFileName, caFileName string

	// updateRootCAFiles is a func to update all managed volumes with the current
	// root CA certificates PEM. Used for testing.
	updateRootCAFilesFn func() error
}

// newCAManager constructs a new camanager which distributes new trust bundles
// to mounted pods, as they are changed.
func newCAManager(log logr.Logger,
	store *storage.Filesystem,
	rootCAs rootca.Interface,
	certFileName, keyFileName, caFileName string,
) *camanager {
	c := &camanager{
		log:          log.WithName("ca-manager"),
		store:        store,
		rootCAs:      rootCAs,
		certFileName: certFileName,
		keyFileName:  keyFileName,
		caFileName:   caFileName,
	}
	c.updateRootCAFilesFn = c.updateRootCAFiles
	return c
}

// run subscribes to events from the Root CAs provider, and updates all managed
// volumes CA files accordingly. Exits early if rootCAs is not configured.
// Blocking function.
func (c *camanager) run(ctx context.Context, updateRetryPeriod time.Duration) {
	// Exit straight away if root CAs haven't been configured.
	if c.rootCAs == nil {
		c.log.Info("not running CA file manager, root CA certificates not configured")
		return
	}

	watcher := c.rootCAs.Subscribe()

	c.log.Info("starting root CA file manager")

	// updateChan is used to trigger an update of CA certificates of file of all
	// managed volumes. Trigged by both RootCAs events, as well as retrying updates on errors.
	updateChan := make(chan struct{}, 1)

	for {
		select {
		case <-ctx.Done():
			c.log.Info("closing root CA file manager")
			return

		case <-watcher:
			updateChan <- struct{}{}

		case <-updateChan:
			c.log.Info("root CA file event received, updating managed volumes")

			if err := c.updateRootCAFilesFn(); err != nil {
				c.log.Error(err, "failed to update root CA files on managed volumes")

				// Retry updating the root CA files.
				go func() {
					select {
					// Wait for 5 seconds before retrying.
					case <-time.After(updateRetryPeriod):
					case <-ctx.Done():
						return
					}
					c.log.Error(err, "retrying CA file update...")
					updateChan <- struct{}{}
				}()

				continue
			}

			c.log.Info("updated root CA files on managed volumes")
		}
	}
}

// updateRootCAFiles will update all managed volumes with the CA certificates
// data returned from rootCAs.
func (c *camanager) updateRootCAFiles() error {
	if c.rootCAs == nil {
		// Exit early if rootCAs is not configured.
		return nil
	}

	log := c.log.WithName("ca-updater")

	volumeIDs, err := c.store.ListVolumes()
	if err != nil {
		return fmt.Errorf("failed to list managed volumes: %w", err)
	}

	for _, volumeID := range volumeIDs {
		meta, err := c.store.ReadMetadata(volumeID)
		if err != nil {
			return fmt.Errorf("%q: failed to read metadata from volume: %w", volumeID, err)
		}

		certData, err := c.store.ReadFile(volumeID, c.certFileName)
		if err != nil {
			return fmt.Errorf("%q: failed to read certificate file from volume to perform write: %w",
				volumeID, err)
		}
		keyData, err := c.store.ReadFile(volumeID, c.keyFileName)
		if err != nil {
			return fmt.Errorf("%q: failed to read key file from volume to perform write: %w",
				volumeID, err)
		}

		// No need to re-write CA data again if it hasn't changed on file.
		caData, err := c.store.ReadFile(volumeID, c.caFileName)
		if err == nil && bytes.Equal(caData, c.rootCAs.CertificatesPEM()) {
			continue
		}

		if err := c.store.WriteFiles(meta, map[string][]byte{
			c.certFileName: certData,
			c.keyFileName:  keyData,
			c.caFileName:   c.rootCAs.CertificatesPEM(),
		}); err != nil {
			return fmt.Errorf("%q: failed to write new ca data to volume: %w",
				volumeID, err)
		}

		log.Info("updated CA file on volume", "volume", volumeID)
	}

	return nil
}
