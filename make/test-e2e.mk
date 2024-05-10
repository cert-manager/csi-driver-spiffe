# Copyright 2023 The cert-manager Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

.PHONY: e2e-setup-cert-manager
e2e-setup-cert-manager: | kind-cluster $(NEEDS_HELM) $(NEEDS_KUBECTL)
	$(HELM) upgrade \
		--install \
		--create-namespace \
		--wait \
		--version $(quay.io/jetstack/cert-manager-controller.TAG) \
		--namespace cert-manager \
		--repo https://charts.jetstack.io \
		--set installCRDs=true \
		--set extraArgs={--controllers='*\,-certificaterequests-approver'} \
		--set image.repository=$(quay.io/jetstack/cert-manager-controller.REPO) \
		--set image.tag=$(quay.io/jetstack/cert-manager-controller.TAG) \
		--set image.pullPolicy=Never \
		--set cainjector.image.repository=$(quay.io/jetstack/cert-manager-cainjector.REPO) \
		--set cainjector.image.tag=$(quay.io/jetstack/cert-manager-cainjector.TAG) \
		--set cainjector.image.pullPolicy=Never \
		--set webhook.image.repository=$(quay.io/jetstack/cert-manager-webhook.REPO) \
		--set webhook.image.tag=$(quay.io/jetstack/cert-manager-webhook.TAG) \
		--set webhook.image.pullPolicy=Never \
		--set startupapicheck.image.repository=$(quay.io/jetstack/cert-manager-startupapicheck.REPO) \
		--set startupapicheck.image.tag=$(quay.io/jetstack/cert-manager-startupapicheck.TAG) \
		--set startupapicheck.image.pullPolicy=Never \
		cert-manager cert-manager >/dev/null

.PHONY: e2e-setup-example
e2e-setup-example: | e2e-setup-cert-manager kind-cluster $(NEEDS_KUBECTL) $(NEEDS_CMCTL)
	$(KUBECTL) apply --server-side -f ./deploy/example/clusterissuer.yaml
	sleep 3
	@# We can rely on the CR being called csi-driver-spiffe-ca-1 in cert-manager v1.13+ thanks to
	@# the StableCertificateRequestName feature gate being beta
	$(CMCTL) approve -n cert-manager csi-driver-spiffe-ca-1 || :
	$(KUBECTL) wait --for=condition=ready clusterissuer csi-driver-spiffe-ca

# The "install" target can be run on its own with any currently active cluster,
# we can't use any other cluster then a target containing "test-e2e" is run.
# When a "test-e2e" target is run, the currently active cluster must be the kind
# cluster created by the "kind-cluster" target.
ifeq ($(findstring test-e2e,$(MAKECMDGOALS)),test-e2e)
install: e2e-setup-example kind-cluster oci-load-manager oci-load-approver
endif

test-e2e-deps: INSTALL_OPTIONS :=
test-e2e-deps: INSTALL_OPTIONS += --set image.repository.driver=$(oci_manager_image_name_development)
test-e2e-deps: INSTALL_OPTIONS += --set image.repository.approver=$(oci_approver_image_name_development)
test-e2e-deps: INSTALL_OPTIONS += --set image.pullPolicy=Never
test-e2e-deps: INSTALL_OPTIONS += --set app.trustDomain=foo.bar
test-e2e-deps: INSTALL_OPTIONS += --set app.approver.signerName=clusterissuers.cert-manager.io/csi-driver-spiffe-ca
test-e2e-deps: INSTALL_OPTIONS += --set app.issuer.name=csi-driver-spiffe-ca
test-e2e-deps: INSTALL_OPTIONS += --set app.driver.volumes[0].name=root-cas
test-e2e-deps: INSTALL_OPTIONS += --set app.driver.volumes[0].secret.secretName=csi-driver-spiffe-ca
test-e2e-deps: INSTALL_OPTIONS += --set app.driver.volumeMounts[0].name=root-cas
test-e2e-deps: INSTALL_OPTIONS += --set app.driver.volumeMounts[0].mountPath=/var/run/secrets/cert-manager-csi-driver-spiffe
test-e2e-deps: INSTALL_OPTIONS += --set app.driver.sourceCABundle=/var/run/secrets/cert-manager-csi-driver-spiffe/ca.crt
test-e2e-deps: install

E2E_FOCUS ?=

.PHONY: test-e2e
## e2e end-to-end tests
## @category Testing
test-e2e: test-e2e-deps | kind-cluster $(NEEDS_GINKGO) $(NEEDS_KUBECTL) $(ARTIFACTS)
	$(GINKGO) \
		--output-dir=$(ARTIFACTS) \
		--focus="$(E2E_FOCUS)" \
		--junit-report=junit-go-e2e.xml \
		./test/e2e/ \
		-ldflags $(go_manager_ldflags) \
		-- \
		--kubeconfig-path=$(CURDIR)/$(kind_kubeconfig) \
		--kubectl-path=$(KUBECTL)
