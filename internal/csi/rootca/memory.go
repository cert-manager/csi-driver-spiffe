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
	"context"
	"sync"
)

// memory is an implementation of RootCAs that holds the CA certificates in
// memory. Accepts a channel to set the certificates PEM. Events are broadcast
// when memory receives a new certificatesPEM.
type memory struct {
	// certificatesPEM is the current root certificates.
	certificatesPEM []byte

	// lock is used as a semaphore for accessing the certificatesPEM data.
	lock sync.RWMutex

	// subscribers is the list of subscribers that will be sent a message when
	// the root certificates changes.
	subscribers []chan<- struct{}
}

// NewMemory constructs a new memory implementation of RootCAs. NewMemory
// listens to the passed channel to set the root CAs.
func NewMemory(ctx context.Context, rootCAs <-chan []byte) Interface {
	m := &memory{}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			case certificatesPEM := <-rootCAs:
				m.lock.Lock()
				m.certificatesPEM = certificatesPEM
				for i := range m.subscribers {
					go func(i int) { m.subscribers[i] <- struct{}{} }(i)
				}
				m.lock.Unlock()
			}
		}
	}()

	return m
}

// CertificatesPEM returns the current root CA certificate in memory.
func (m *memory) CertificatesPEM() []byte {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.certificatesPEM
}

// Subscribe subscribes the consumer to events to when the root CA changes in
// memory.
func (m *memory) Subscribe() <-chan struct{} {
	m.lock.Lock()
	defer m.lock.Unlock()
	sub := make(chan struct{})
	m.subscribers = append(m.subscribers, sub)
	return sub
}
