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
)

// manageCAFiles subscribes to events from the Root CAs provider, and updates
// all managed volumes CA files accordingly. Exits early if rootCAs is not
// configured.
func (d *Driver) manageCAFiles(ctx context.Context, updateRetryPeriod time.Duration) {
	log := d.log.WithName("ca-manager")

	// Exit straight away if root CAs haven't been configured.
	if d.rootCAs == nil {
		log.Info("not running CA file manager, root CA certificates not configured")
		return
	}

	watcher := d.rootCAs.Subscribe()

	log.Info("starting root CA file manager")

	// updateChan is used to trigger an update of CA certificates of file of all
	// managed volumes. Trigged by both RootCAs events, as well as retrying updates on errors.
	updateChan := make(chan struct{}, 1)

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Info("closing root CA file manager")
				return

			case <-watcher:
				updateChan <- struct{}{}

			case <-updateChan:
				log.Info("root CA file event received, updating managed volumes")

				if err := d.updateRootCAFilesFn(); err != nil {
					log.Error(err, "failed to update root CA files on managed volumes")

					// Retry updating the root CA files.
					go func() {
						select {
						// Wait for 5 seconds before retrying.
						case <-time.After(updateRetryPeriod):
						case <-ctx.Done():
							return
						}
						log.Error(err, "retrying CA file update...")
						updateChan <- struct{}{}
					}()

					continue
				}

				log.Info("updated root CA files on managed volumes")
			}
		}
	}()
}

// updateRootCAFiles will update all managed volumes with the CA certificates
// data returned from rootCAs.
func (d *Driver) updateRootCAFiles() error {
	if d.rootCAs == nil {
		// Exit early if rootCAs is not configured.
		return nil
	}

	volumeIDs, err := d.store.ListVolumes()
	if err != nil {
		return fmt.Errorf("failed to list managed volumes: %w", err)
	}

	for _, volumeID := range volumeIDs {
		meta, err := d.store.ReadMetadata(volumeID)
		if err != nil {
			return fmt.Errorf("%q: failed to read metadata from volume: %w", volumeID, err)
		}

		certData, err := d.store.ReadFile(volumeID, d.certFileName)
		if err != nil {
			return fmt.Errorf("%q: failed to read certificate file from volume to perform write: %w",
				volumeID, err)
		}
		keyData, err := d.store.ReadFile(volumeID, d.keyFileName)
		if err != nil {
			return fmt.Errorf("%q: failed to read key file from volume to perform write: %w",
				volumeID, err)
		}

		// No need to re-write CA data again if it hasn't changed on file.
		caData, err := d.store.ReadFile(volumeID, d.caFileName)
		if err == nil && bytes.Equal(caData, d.rootCAs.CertificatesPEM()) {
			continue
		}

		if err := d.store.WriteFiles(meta, map[string][]byte{
			d.certFileName: certData,
			d.keyFileName:  keyData,
			d.caFileName:   d.rootCAs.CertificatesPEM(),
		}); err != nil {
			return fmt.Errorf("%q: failed to write new ca data to volume: %w",
				volumeID, err)
		}

		d.log.WithName("ca-updater").Info("updated CA file on volume", "volume", volumeID)
	}

	return nil
}
