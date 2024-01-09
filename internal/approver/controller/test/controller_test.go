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

package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func Test_controllers(t *testing.T) {
	rootDir := os.Getenv("ROOTDIR")
	if len(rootDir) == 0 {
		t.Skip("Skipping test as ROOTDIR environment variable not defined")
	}

	env = &envtest.Environment{
		AttachControlPlaneOutput: false,
		CRDDirectoryPaths:        []string{filepath.Join(rootDir, "bin/cert-manager")},
	}

	t.Logf("starting API server...")
	if _, err := env.Start(); err != nil {
		t.Fatalf("failed to start control plane: %v", err)
	}
	t.Logf("running API server at %q", env.Config.Host)

	// Register cleanup func to stop the api-server after the test has finished.
	t.Cleanup(func() {
		t.Log("stopping API server")
		if err := env.Stop(); err != nil {
			t.Fatalf("failed to shut down control plane: %v", err)
		}
	})

	gomega.RegisterFailHandler(ginkgo.Fail)

	suiteConfig, reporterConfig := ginkgo.GinkgoConfiguration()

	// Turn on verbose by default to get spec names
	reporterConfig.Verbose = true
	// Turn on EmitSpecProgress to get spec progress (especially on interrupt)
	suiteConfig.EmitSpecProgress = true
	// Randomize specs as well as suites
	suiteConfig.RandomizeAllSpecs = true

	ginkgo.RunSpecs(t, "unit-controller", suiteConfig, reporterConfig)
}
