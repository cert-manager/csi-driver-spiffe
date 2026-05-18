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
	"errors"
	"fmt"
	"sync"
	"time"

	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	issuerNameKey  = "issuer-name"
	issuerKindKey  = "issuer-kind"
	issuerGroupKey = "issuer-group"
)

// configmap is an implementation of Interface that watches a Kubernetes
// ConfigMap for runtime configuration and broadcasts a message when the
// configuration changes.
type configmap struct {
	// log is the logger.
	log logr.Logger

	// k8sClient is used to watch the ConfigMap.
	k8sClient client.WithWatch

	// configMapName is the name of the ConfigMap to watch.
	configMapName string

	// configMapNamespace is the namespace of the ConfigMap to watch.
	configMapNamespace string

	// active is the currently active configuration.
	active Config

	// static is the fallback configuration used when the ConfigMap is absent.
	static Config

	// lock guards access to active.
	lock sync.RWMutex
}

// NewConfigMap constructs a new configmap implementation of Interface. It sets
// the active configuration to static initially and starts a goroutine to watch
// the named ConfigMap. When the ConfigMap is added or modified the active
// configuration is updated from the three keys issuer-name, issuer-kind, and
// issuer-group. When the ConfigMap is deleted the active configuration reverts
// to static. The logger is extracted from ctx via logr.FromContext.
func NewConfigMap(ctx context.Context, k8sClient client.WithWatch, configMapName, configMapNamespace string, static Config) Interface {
	log := logr.FromContextOrDiscard(ctx).
		WithName("runtime-config-watcher").
		WithValues("config-map-name", configMapName, "config-map-namespace", configMapNamespace)

	c := &configmap{
		log:                log,
		k8sClient:          k8sClient,
		configMapName:      configMapName,
		configMapNamespace: configMapNamespace,
		active:             static,
		static:             static,
	}

	go c.start(ctx)

	return c
}

// start watches the ConfigMap for changes and updates the active configuration.
// It is intended to be run in a goroutine and retries on failure with a 5s
// delay. It returns when ctx is cancelled.
func (c *configmap) start(ctx context.Context) {
LOOP:
	for {
		c.log.Info("Starting / restarting watcher for runtime configuration")
		cmList := &corev1.ConfigMapList{}

		watcher, err := c.k8sClient.Watch(ctx, cmList, &client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector("metadata.name", c.configMapName),
			Namespace:     c.configMapNamespace,
		})
		if err != nil {
			c.log.Error(err, "Failed to create ConfigMap watcher; will retry in 5s")
			time.Sleep(5 * time.Second)
			continue
		}

		for {
			select {
			case <-ctx.Done():
				c.log.Info("Received context cancellation, shutting down runtime configuration watcher")
				watcher.Stop()
				break LOOP

			case event, open := <-watcher.ResultChan():
				if !open {
					c.log.Info("Received closed channel from ConfigMap watcher, will recreate")
					watcher.Stop()
					continue LOOP
				}

				switch event.Type {
				case watch.Deleted:
					c.handleDeletion()

				case watch.Added:
					if err := c.handleChange(event); err != nil {
						c.log.Error(err, "Failed to handle new runtime configuration for issuerRef")
					}

				case watch.Modified:
					if err := c.handleChange(event); err != nil {
						c.log.Error(err, "Failed to handle runtime configuration issuerRef change")
					}

				case watch.Bookmark:
					// Ignore bookmark events.

				case watch.Error:
					err, ok := event.Object.(error)
					if !ok {
						c.log.Error(nil, "Got an error event when watching runtime configuration but unable to determine further information")
					} else {
						c.log.Error(err, "Got an error event when watching runtime configuration")
					}

				default:
					c.log.Info("Got unknown event for runtime configuration ConfigMap; ignoring", "event-type", string(event.Type))
				}
			}
		}
	}

	c.log.Info("Stopped runtime configuration watcher")
}

// handleChange updates the active configuration from the ConfigMap data in the
// event and broadcasts to subscribers.
func (c *configmap) handleChange(event watch.Event) error {
	cm, ok := event.Object.(*corev1.ConfigMap)
	if !ok {
		return fmt.Errorf("got unexpected type for runtime configuration source; this is likely a programming error")
	}

	issuerRef := cmmeta.IssuerReference{}

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

	c.lock.Lock()
	defer c.lock.Unlock()

	c.active = Config{IssuerRef: issuerRef}
	c.log.Info("Changed active issuerRef in response to runtime configuration ConfigMap",
		"issuer-name", c.active.IssuerRef.Name,
		"issuer-kind", c.active.IssuerRef.Kind,
		"issuer-group", c.active.IssuerRef.Group,
	)

	return nil
}

// handleDeletion reverts the active configuration to the static fallback.
func (c *configmap) handleDeletion() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.static.IssuerRef.Name == "" {
		c.log.Info("Runtime issuance configuration was deleted and no issuerRef was configured at install time; issuance will fail until runtime configuration is reinstated")
	} else {
		c.log.Info("Runtime issuance configuration was deleted; issuance will revert to original issuerRef configured at install time")
	}

	c.active = c.static
}

// Config returns the current active runtime configuration.
func (c *configmap) Config() Config {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.active
}
