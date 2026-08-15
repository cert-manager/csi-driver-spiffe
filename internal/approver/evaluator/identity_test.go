/*
Copyright 2026 The cert-manager Authors.

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

package evaluator

import (
	"crypto/x509"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"hegel.dev/go/hegel"
)

// parseURIs parses the pool entries; every pool entry is a valid URL, so the
// only failure mode is a test bug.
func parseURIs(ht *hegel.T, uris []string) []*url.URL {
	out := make([]*url.URL, 0, len(uris))
	for _, s := range uris {
		u, err := url.Parse(s)
		if err != nil {
			ht.Fatalf("failed to parse %q: %v", s, err)
		}
		out = append(out, u)
	}
	return out
}

// drawUsername draws either a username from the pool of
// ServiceAccount-shaped names (so the accept path is reachable) or arbitrary
// Unicode text; the oracles recompute acceptance from the result either way.
func drawUsername(ht *hegel.T, pool []string) string {
	if hegel.Draw(ht, hegel.Booleans()) {
		return pool[hegel.Draw(ht, hegel.Integers(0, len(pool)-1))]
	}
	return hegel.Draw(ht, hegel.Text().MaxSize(40))
}

// TestValidateIdentityProperty: validateIdentity accepts iff the username is
// a well-formed ServiceAccount username (system:serviceaccount:<ns>:<sa>) and
// the CSR carries exactly one URI SAN equal to the SPIFFE ID
// spiffe://<trust-domain>/ns/<ns>/sa/<sa> derived from it. The oracle
// recomputes that rule from the drawn parts. Replaces the example table,
// whose rows instantiated the same rule.
//
// URIs are drawn from a pool of canonical URL strings so that the oracle can
// compare strings directly: validateIdentity compares url.URL.String(), which
// normalizes non-canonical spellings.
func TestValidateIdentityProperty(t *testing.T) {
	const trustDomain = "foo.bar"

	usernamePool := []string{
		"system:serviceaccount:sandbox:sleep",
		"system:serviceaccount:sandbox:httpbin",
		"system:serviceaccount:prod:sleep",
		"system:serviceaccount:foo",
		"system:serviceaccount:a:b:c",
		"system:node:sandbox:sleep",
		"kubernetes-admin",
	}
	uriPool := []string{
		"spiffe://foo.bar/ns/sandbox/sa/sleep",
		"spiffe://foo.bar/ns/sandbox/sa/httpbin",
		"spiffe://foo.bar/ns/prod/sa/sleep",
		"spiffe://bar.foo/ns/sandbox/sa/sleep",
		"http://foo.bar/ns/sandbox/sa/sleep",
	}

	hegel.Test(t, func(ht *hegel.T) {
		username := drawUsername(ht, usernamePool)

		var uris []string
		for _, i := range hegel.Draw(ht, hegel.Lists(hegel.Integers(0, len(uriPool)-1)).MaxSize(3)) {
			uris = append(uris, uriPool[i])
		}

		split := strings.Split(username, ":")
		want := len(split) == 4 && split[0] == "system" && split[1] == "serviceaccount" &&
			len(uris) == 1 &&
			uris[0] == fmt.Sprintf("spiffe://%s/ns/%s/sa/%s", trustDomain, split[2], split[3])

		i := &internal{trustDomain: trustDomain}
		err := i.validateIdentity(&x509.CertificateRequest{URIs: parseURIs(ht, uris)}, username)
		if (err == nil) != want {
			ht.Fatalf("uris %v username %q: error = %v, want accept %t", uris, username, err, want)
		}
	}, hegel.WithTestCases(1000))
}

// TestValidateDriverServiceAccountProperty: validateDriverServiceAccount
// accepts iff the CSR carries exactly one spiffe:// URI in the configured
// trust domain and the request was made by the driver's own ServiceAccount.
// The SPIFFE path is deliberately unconstrained.
func TestValidateDriverServiceAccountProperty(t *testing.T) {
	const (
		trustDomain = "foo.bar"
		driverSA    = "system:serviceaccount:cert-manager:csi-driver-spiffe"
	)

	usernamePool := []string{
		driverSA,
		"system:serviceaccount:sandbox:sleep",
		"kubernetes-admin",
	}
	uriPool := []string{
		"spiffe://foo.bar/ns/sandbox/sa/sleep",
		"spiffe://foo.bar/anything/at/all",
		"spiffe://bar.foo/ns/sandbox/sa/sleep",
		"http://foo.bar/ns/sandbox/sa/sleep",
	}

	hegel.Test(t, func(ht *hegel.T) {
		username := drawUsername(ht, usernamePool)

		var uris []string
		for _, i := range hegel.Draw(ht, hegel.Lists(hegel.Integers(0, len(uriPool)-1)).MaxSize(3)) {
			uris = append(uris, uriPool[i])
		}

		parsed := parseURIs(ht, uris)
		want := len(parsed) == 1 && parsed[0].Scheme == "spiffe" && parsed[0].Host == trustDomain &&
			username == driverSA

		i := &internal{trustDomain: trustDomain, driverServiceAccount: driverSA}
		err := i.validateDriverServiceAccount(&x509.CertificateRequest{URIs: parsed}, username)
		if (err == nil) != want {
			ht.Fatalf("uris %v username %q: error = %v, want accept %t", uris, username, err, want)
		}
	}, hegel.WithTestCases(1000))
}
