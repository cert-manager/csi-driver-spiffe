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

package runtimeconfig

import (
	"context"
	"sync"
)

// memory is an implementation of Interface that holds the runtime
// configuration in memory. Accepts an optional channel to update the configuration.
type memory struct {
	// active is the current runtime configuration.
	active Config

	// lock guards access to active.
	lock sync.RWMutex
}

// NewMemory constructs a new memory implementation of Interface. It sets the
// initial configuration to initial and listens to the updates channel (if
// non-nil) for configuration changes.
func NewMemory(ctx context.Context, initial Config, updates <-chan Config) Interface {
	m := &memory{
		active: initial,
	}

	if updates != nil {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return

				case cfg := <-updates:
					m.lock.Lock()
					m.active = cfg
					m.lock.Unlock()
				}
			}
		}()
	}

	return m
}

// Config returns the current runtime configuration in memory.
func (m *memory) Config() Config {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.active
}

