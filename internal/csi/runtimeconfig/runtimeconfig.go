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

// Package runtimeconfig provides the runtime-configurable settings for the
// SPIFFE CSI driver, including support for watching a Kubernetes ConfigMap for
// live issuer configuration updates.
package runtimeconfig

import (
	"context"
	"fmt"

	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Config holds the runtime-configurable settings for the SPIFFE CSI driver.
type Config struct {
	// IssuerRef is the cert-manager issuer reference to use when creating
	// CertificateRequests.
	IssuerRef cmmeta.IssuerReference
}

// Interface provides the current runtime configuration.
type Interface interface {
	// Config returns the current runtime configuration.
	Config() Config
}

// DynamicConfig holds the configuration for the ConfigMap-based runtime
// configuration source.
type DynamicConfig struct {
	// ConfigMapName is the name of the ConfigMap to watch for runtime
	// configuration.
	ConfigMapName string

	// ConfigMapNamespace is the namespace of the ConfigMap to watch for
	// runtime configuration.
	ConfigMapNamespace string
}

// Options configures the runtime configuration source.
type Options struct {
	// StaticConfig is the static configuration to use when no dynamic
	// configuration source is provided, or as the fallback when the dynamic
	// configuration source is unavailable.
	StaticConfig Config

	// DynamicConfig configures an optional ConfigMap-based runtime
	// configuration source.
	DynamicConfig DynamicConfig
}

// New constructs the appropriate Interface based on the provided options.
// When opts.DynamicConfig.ConfigMapName is set, a ConfigMap watcher is
// constructed and c must not be nil. When only a static IssuerRef is provided,
// an in-memory implementation is returned. Returns an error if neither
// StaticConfig.IssuerRef.Name nor DynamicConfig.ConfigMapName is set, or if
// ConfigMapName is set but c is nil. The logger is extracted from ctx via
// logr.FromContext.
func New(ctx context.Context, c client.WithWatch, opts Options) (Interface, error) {
	if opts.DynamicConfig.ConfigMapName != "" {
		if c == nil {
			return nil, fmt.Errorf("a Kubernetes client is required when ConfigMapName is set")
		}
		return NewConfigMap(ctx, c, opts.DynamicConfig.ConfigMapName, opts.DynamicConfig.ConfigMapNamespace, opts.StaticConfig), nil
	}

	if opts.StaticConfig.IssuerRef.Name == "" {
		return nil, fmt.Errorf("runtime issuance configuration is required if no issuer is provided at startup")
	}

	return NewMemory(ctx, opts.StaticConfig, nil), nil
}
