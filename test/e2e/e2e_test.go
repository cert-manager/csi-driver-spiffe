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

package smoke

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"
)

// Test_Smoke runs the full suite of smoke tests against approver-policy
func Test_e2e(t *testing.T) {
	rootDir := os.Getenv("ROOTDIR")
	if len(rootDir) == 0 {
		t.Skip("Skipping test as ROOTDIR environment variable not defined")
	}

	gomega.RegisterFailHandler(ginkgo.Fail)

	var artifactsDir string
	if path := os.Getenv("ARTIFACTS"); len(path) > 0 {
		artifactsDir = path
	}

	junitReporter := reporters.NewJUnitReporter(filepath.Join(
		artifactsDir,
		"junit-e2e.xml",
	))

	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "e2e", []ginkgo.Reporter{junitReporter})
}
