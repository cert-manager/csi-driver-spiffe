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
	"strings"
	"time"

	cmapiutil "github.com/cert-manager/cert-manager/pkg/api/util"
	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8srand "k8s.io/apimachinery/pkg/util/rand"
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
		cmd := exec.CommandContext(f.Context(), f.Config().KubectlBinPath, "exec", "-n", f.Namespace.Name, podName, containerArg, "--", "cat", fullPath)

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

// IssuerCleanupFunc is called to clean up issuer related resources after a test. Any returned
// cleanup function should always be safe to call and should always be called at some point after
// the returning function regardless of whether that function returned an error or not
type IssuerCleanupFunc func() error

// dummyIssuerCleanupFunc should be returned by functions which return an IssuerCleanupFunc where
// there's nothing to clean up (e.g. if a fatal error happened before any resources were created).
func dummyIssuerCleanupFunc() error {
	return nil
}

// CreateSelfSignedIssuer creates a SelfSigned ClusterIssuer which can be used to in-turn create CA
// issuers for tests.
// Returns an issuerRef for the issuer and a cleanup function to remove the issuer after the test
// completes.
// The cleanup function is always safe to call and should always be called after this function
// returns, regardless of whether it returned an error or not
func CreateSelfSignedIssuer(f *framework.Framework) (*cmmeta.ObjectReference, IssuerCleanupFunc, error) {
	iss := &cmapi.ClusterIssuer{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "selfsigned-clusterissuer-",
		},
		Spec: cmapi.IssuerSpec{
			IssuerConfig: cmapi.IssuerConfig{
				SelfSigned: &cmapi.SelfSignedIssuer{},
			},
		},
	}

	err := f.Client().Create(f.Context(), iss)
	if err != nil {
		return nil, dummyIssuerCleanupFunc, err
	}

	cleanupFunc := func() error {
		return f.Client().Delete(f.Context(), iss)
	}

	issuerRef := &cmmeta.ObjectReference{
		Name:  iss.Name,
		Kind:  "ClusterIssuer",
		Group: "cert-manager.io",
	}

	return issuerRef, cleanupFunc, nil
}

// CreateNewCAIssuer creates an issuer which can be used for an end-to-end test and cleaned up
// afterwards.
// Returns an issuerRef for the issuer, a bundle containing the issuer's data and a function to
// clean up all issuer resources.
// The cleanup function is always safe to call and should always be called after this function
// returns, regardless of whether it returned an error or not
func CreateNewCAIssuer(f *framework.Framework) (*cmmeta.ObjectReference, *CertBundle, IssuerCleanupFunc, error) {
	var objectsForCleanup []client.Object

	selfSignedIssuerRef, selfSignedCleanupFunc, err := CreateSelfSignedIssuer(f)
	if err != nil {
		return nil, nil, selfSignedCleanupFunc, fmt.Errorf("failed to create selfsigned ClusterIssuer: %s", err)
	}

	cleanupFunc := func() error {
		var errs []error

		selfSignedCleanupErr := selfSignedCleanupFunc()
		if selfSignedCleanupErr != nil {
			errs = append(errs, selfSignedCleanupErr)
		}

		for _, m := range objectsForCleanup {
			err := f.Client().Delete(f.Context(), m)
			if err != nil {
				errs = append(errs, err)
			}
		}

		return errors.Join(errs...)
	}

	certSecretName := fmt.Sprintf("e2e-root-%s", k8srand.String(6))

	cert := &cmapi.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      certSecretName,
			Namespace: "cert-manager", // TODO: this might not always be the case
		},
		Spec: cmapi.CertificateSpec{
			CommonName: certSecretName,
			IsCA:       true,
			PrivateKey: &cmapi.CertificatePrivateKey{
				Algorithm: cmapi.ECDSAKeyAlgorithm,
				Size:      256,
			},
			SecretName: certSecretName,
			IssuerRef:  *selfSignedIssuerRef,
		},
	}

	err = f.Client().Create(f.Context(), cert)
	if err != nil {
		return nil, nil, cleanupFunc, err
	}

	objectsForCleanup = append(objectsForCleanup, cert)

	err = approveCertificateRequestsForCertificate(f, cert)
	if err != nil {
		return nil, nil, cleanupFunc, fmt.Errorf("failed to approve CertificateRequest: %s", err)
	}

	err = WaitForCertificateReady(f, cert)
	if err != nil {
		return nil, nil, cleanupFunc, fmt.Errorf("failed to wait for cert to become ready: %s", err)
	}

	iss := &cmapi.ClusterIssuer{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("ca-issuer-%s", certSecretName),
		},
		Spec: cmapi.IssuerSpec{
			IssuerConfig: cmapi.IssuerConfig{
				CA: &cmapi.CAIssuer{
					SecretName: cert.Spec.SecretName,
				},
			},
		},
	}

	err = f.Client().Create(f.Context(), iss)
	if err != nil {
		return nil, nil, cleanupFunc, err
	}

	objectsForCleanup = append(objectsForCleanup, iss)

	caBundle, err := ReadCertFromSecret(f, certSecretName, "cert-manager")
	if err != nil {
		return nil, nil, cleanupFunc, err
	}

	newIssuerRef := cmmeta.ObjectReference{
		Name:  iss.Name,
		Kind:  "ClusterIssuer",
		Group: "cert-manager.io",
	}

	return &newIssuerRef, caBundle, cleanupFunc, nil
}

// ReadCertFromSecret loads a certificate bundle from a Secret resource
func ReadCertFromSecret(f *framework.Framework, secretName string, secretNamespace string) (*CertBundle, error) {
	certSecret := &corev1.Secret{}

	key := client.ObjectKey{
		Name:      secretName,
		Namespace: secretNamespace,
	}

	err := f.Client().Get(f.Context(), key, certSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch secret %s/%s: %s", secretNamespace, secretName, err)
	}

	chainBytes, exists := certSecret.Data["tls.crt"]
	if !exists {
		return nil, fmt.Errorf("failed to find certificate chain in secret %s/%s", secretNamespace, secretName)
	}

	privateKeyBytes, exists := certSecret.Data["tls.key"]
	if !exists {
		return nil, fmt.Errorf("failed to find private key in secret %s/%s", secretNamespace, secretName)
	}

	caBytes, exists := certSecret.Data["ca.crt"]
	if !exists {
		return nil, fmt.Errorf("failed to find CA data in secret %s/%s", secretNamespace, secretName)
	}

	return &CertBundle{
		CertificatePEM: chainBytes,
		PrivateKeyPEM:  privateKeyBytes,
		CAPEM:          caBytes,
	}, nil
}

// WaitForCertificateReady waits until the references Certificate resource is marked as ready
func WaitForCertificateReady(f *framework.Framework, cert *cmapi.Certificate) error {
	timeout := 60 * time.Second
	interval := 2 * time.Second
	immediate := false

	return wait.PollUntilContextTimeout(f.Context(), interval, timeout, immediate, func(ctx context.Context) (bool, error) {
		err := f.Client().Get(ctx, client.ObjectKeyFromObject(cert), cert)
		if err != nil {
			return false, err
		}

		for _, cond := range cert.Status.Conditions {
			if cond.Type != cmapi.CertificateConditionReady {
				continue
			}

			return cond.Status == cmmeta.ConditionTrue, nil
		}

		return false, nil
	})
}

func approveCertificateRequestsForCertificate(f *framework.Framework, cert *cmapi.Certificate) error {
	crList := &cmapi.CertificateRequestList{}

	listOpts := &client.ListOptions{
		Namespace: cert.Namespace,
	}

	timeout := 10 * time.Second
	interval := 1 * time.Second
	immediate := false

	err := wait.PollUntilContextTimeout(f.Context(), interval, timeout, immediate, func(ctx context.Context) (bool, error) {
		err := f.Client().List(ctx, crList, listOpts)
		if err != nil {
			return false, err
		}

		for _, cr := range crList.Items {
			if strings.HasPrefix(cr.Name, cert.Name) {
				return true, nil
			}
		}

		return false, nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for CertificateRequest to be created: %s", err)
	}

	var updateErrs []error

	for _, cr := range crList.Items {
		if !strings.HasPrefix(cr.Name, cert.Name) {
			continue
		}

		cmapiutil.SetCertificateRequestCondition(&cr, cmapi.CertificateRequestConditionApproved, cmmeta.ConditionTrue, "csi-driver-spiffe-e2e-test", "Manually approved for csi-driver-spiffe e2e tests")

		err := f.Client().Status().Update(f.Context(), &cr)
		if err != nil {
			updateErrs = append(updateErrs, err)
		}
	}

	if len(updateErrs) > 0 {
		return errors.Join(updateErrs...)
	}

	return nil
}
