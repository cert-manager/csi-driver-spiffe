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

package rootca

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/go-logr/logr"
)

// file is an implementation of RootCAs which watches reads the root
// certificates from file, and broadcasts a message when that file has changed.
type file struct {
	// log is the RootCAs file logger.
	log logr.Logger

	// filepath is the file path location to where the root certificates are
	// stored, and will be watched for changes.
	filepath string

	// certificatesPEM is the current root certificates.
	certificatesPEM []byte

	// lock is used as a semaphore for accessing the certificatesPEM data.
	lock sync.RWMutex

	// subscribers is the list of subscribers that will be sent a message when
	// the root certificates changes.
	subscribers []chan<- struct{}
}

// NewFile constructs a new file implementation of RootCAs. NewFile reads and
// sets up a watcher for the root CAs on file.
func NewFile(ctx context.Context, log logr.Logger, filepath string) (Interface, error) {
	log = log.WithName("file").WithValues("filepath", filepath)

	// Read initial certificates from file.
	certificatesPEM, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read root CAs file %q: %s", filepath, err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watch: %w", err)
	}

	if err := watcher.Add(filepath); err != nil {
		return nil, fmt.Errorf("failed to add root CAs file for watching %q: %w", filepath, err)
	}

	f := &file{
		log:             log,
		filepath:        filepath,
		certificatesPEM: certificatesPEM,
	}

	// Start the file watcher.
	go f.start(ctx, watcher)

	return f, nil
}

func (f *file) start(ctx context.Context, watcher *fsnotify.Watcher) {
	defer watcher.Close()

	for {
		select {
		case <-ctx.Done():
			f.log.Info("closing root CAs file watcher")
			return

		case event := <-watcher.Events:
			f.log.V(3).Info("received event from file watcher", "event", event.Op.String())

			// Watch for remove events, since this is actually the syslink being
			// changed in the volume mount.
			if event.Op == fsnotify.Remove {
				if err := watcher.Remove(event.Name); err != nil {
					f.log.Error(err, "failed to remove file watch")
				}
				if err := watcher.Add(f.filepath); err != nil {
					f.log.Error(err, "failed to add new file watch")
				}

				f.reloadCertificatesPEM()
				continue
			}

			// Also allow normal files to be modified and reloaded.
			if event.Op&fsnotify.Write == fsnotify.Write {
				f.reloadCertificatesPEM()
				continue
			}

		case err := <-watcher.Errors:
			f.log.Error(err, "error watching root CAs file")
		}
	}
}

func (f *file) reloadCertificatesPEM() {
	f.lock.Lock()
	defer f.lock.Unlock()

	certificatesPEM, err := os.ReadFile(f.filepath)
	if err != nil {
		f.log.Error(err, "failed to read root CAs file")
		return
	}

	// If the file contents hasn't changed, no need to update store and broadcast
	// event.
	if bytes.Equal(certificatesPEM, f.certificatesPEM) {
		return
	}

	// Update certificatesPEM store and broadcast event to subscribers.
	f.certificatesPEM = certificatesPEM
	for i := range f.subscribers {
		go func(i int) { f.subscribers[i] <- struct{}{} }(i)
	}
}

// CertificatesPEM returns the current root CA certificate on file.
func (f *file) CertificatesPEM() []byte {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return f.certificatesPEM
}

// Subscribe subscribes the consumer to events to when the root CA changes on
// file.
func (f *file) Subscribe() <-chan struct{} {
	f.lock.Lock()
	defer f.lock.Unlock()
	sub := make(chan struct{})
	f.subscribers = append(f.subscribers, sub)
	return sub
}
