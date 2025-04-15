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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/klog/v2/ktesting"
)

func Test_NewFile(t *testing.T) {
	filepath := filepath.Join(t.TempDir(), "test-file.pem")

	t.Log("if no file exists, expect NewFile to error")
	_, err := NewFile(t.Context(), ktesting.NewLogger(t, ktesting.DefaultConfig), filepath)
	assert.Error(t, err, "expect file to not exist")

	t.Log("should return the contents of the file with CertificatesPEM()")
	assert.NoError(t, os.WriteFile(filepath, []byte("test data"), 0600))

	f, err := NewFile(t.Context(), ktesting.NewLogger(t, ktesting.DefaultConfig), filepath)
	assert.NoError(t, err)

	assert.Equal(t, []byte("test data"), f.CertificatesPEM())

	t.Log("should not fire an event when the file doesn't change")
	sub := f.Subscribe()
	assert.NoError(t, os.WriteFile(filepath, []byte("test data"), 0600))
	select {
	case <-time.After(time.Millisecond * 50):
	case <-sub:
		assert.Fail(t, "expected to not receive an event when the target file hasn't changed")
	}

	t.Log("should fire an event when the file changes")
	assert.NoError(t, os.WriteFile(filepath, []byte("new test data"), 0600))

	select {
	case <-time.After(time.Millisecond * 50):
		assert.Fail(t, "expected to receive an event when the target file has changed")
	case <-sub:
	}

	t.Log("should return new test data now it has been written to file")
	assert.Equal(t, []byte("new test data"), f.CertificatesPEM())
}
