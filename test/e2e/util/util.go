/*
Copyright 2024 The cert-manager Authors.

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

package util

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cert-manager/csi-driver-spiffe/test/e2e/framework"

	. "github.com/onsi/ginkgo/v2"
)

const (
	pollInterval  = 1 * time.Second
	pollTimeout   = 60 * time.Second
	pollImmediate = false
)

func WaitForPodReady(f *framework.Framework, pod *corev1.Pod) error {
	return wait.PollUntilContextTimeout(f.Context(), pollInterval, pollTimeout, pollImmediate, func(ctx context.Context) (bool, error) {
		err := f.Client().Get(ctx, client.ObjectKeyFromObject(pod), pod)
		if err != nil {
			return false, err
		}

		for _, cond := range pod.Status.Conditions {
			if cond.Type != corev1.PodReady {
				continue
			}

			return cond.Status == corev1.ConditionTrue, nil
		}

		return false, nil
	})
}

// CertBundle holds PEM data read from a csi-driver-spiffe mounted volume
type CertBundle struct {
	CertificatePEM []byte
	PrivateKeyPEM  []byte
	CAPEM          []byte
}

// CheckNotEmpty returns an error if any of the PEM entries in the CertBundle are empty
func (cb *CertBundle) CheckNotEmpty() error {
	if cb == nil {
		return fmt.Errorf("nil CertBundle is empty")
	}

	var errs []error

	if len(cb.CertificatePEM) == 0 {
		errs = append(errs, fmt.Errorf("tls.crt was empty"))
	}

	if len(cb.PrivateKeyPEM) == 0 {
		errs = append(errs, fmt.Errorf("tls.key was empty"))
	}

	if len(cb.CAPEM) == 0 {
		errs = append(errs, fmt.Errorf("ca.crt was empty"))
	}

	return errors.Join(errs...)
}

// ReadCertFromMountPath uses kubectl exec to retrieve tls.crt, tls.key and ca.crt from a running pod
func ReadCertFromMountPath(f *framework.Framework, mountPath string, podName string, containerName string) (*CertBundle, error) {
	bundle := new(CertBundle)

	type fileWithPtr struct {
		Filename     string
		TargetBuffer *[]byte
	}

	targets := []fileWithPtr{{
		Filename:     "tls.crt",
		TargetBuffer: &bundle.CertificatePEM,
	}, {
		Filename:     "tls.key",
		TargetBuffer: &bundle.PrivateKeyPEM,
	}, {
		Filename:     "ca.crt",
		TargetBuffer: &bundle.CAPEM,
	}}

	var readErrs []error

	for _, target := range targets {
		buf := new(bytes.Buffer)

		fullPath := filepath.Join(mountPath, target.Filename)
		containerArg := fmt.Sprintf("-c%s", containerName)

		// #nosec G204
		cmd := exec.Command(f.Config().KubectlBinPath, "exec", "-n", f.Namespace.Name, podName, containerArg, "--", "cat", fullPath)

		cmd.Stdout = buf
		cmd.Stderr = GinkgoWriter

		err := cmd.Run()
		if err != nil {
			readErrs = append(readErrs, fmt.Errorf("failed to read %q from target pod: %s", fullPath, err))
			continue
		}

		*target.TargetBuffer = buf.Bytes()
	}

	err := errors.Join(readErrs...)
	if err != nil {
		return nil, err
	}

	return bundle, nil
}
