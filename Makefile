# Copyright 2021 The cert-manager Authors.
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


BINDIR ?= $(CURDIR)/bin
ARCH   ?= $(shell go env GOARCH)
OS     ?= $(shell go env GOOS)

HELM_VERSION ?= 3.11.1
KUBEBUILDER_TOOLS_VERISON ?= 1.26.0
IMAGE_PLATFORMS ?= linux/amd64,linux/arm64,linux/arm/v7,linux/ppc64le

GOMARKDOC_FLAGS=--format github --repository.url "https://github.com/cert-manager/csi-driver-spiffe" --repository.default-branch master --repository.path /

RELEASE_VERSION ?= 0.3.0

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: clean
clean: ## clean up created files
	rm -rf \
		$(BINDIR) \
		_artifacts

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: lint
lint: helm-docs ## Run linters against code.
	./hack/verify-boilerplate.sh

.PHONY: helm-docs
helm-docs: depend # verify helm-docs
	./hack/verify-helm-docs.sh

.PHONY: test
test: depend lint vet ## test csi-driver-spiffe
	ARTIFACTS=$(shell pwd)/_artifacts KUBEBUILDER_ASSETS=$(BINDIR)/kubebuilder/bin ROOTDIR=$(CURDIR) go test -v $(TEST_ARGS) ./cmd/... ./internal/...

.PHONY: build
build: build-driver build-approver ## Build binaries.

.PHONY: build-driver
build-driver: ## Build driver binary.
	CGO_ENABLED=0 GO111MODULE=on go build -o bin/cert-manager-csi-driver-spiffe ./cmd/csi

.PHONY: build-approver
build-approver: ## Build approver binary.
	CGO_ENABLED=0 GO111MODULE=on go build -o bin/cert-manager-csi-driver-spiffe-approver ./cmd/approver

.PHONY: verify
verify: test build ## Verify repo.

.PHONY: image
image: ## build docker image targeting all supported platforms
	docker buildx build --platform=$(IMAGE_PLATFORMS) -t quay.io/jetstack/cert-manager-csi-driver-spiffe:v$(RELEASE_VERSION) --output type=oci,dest=./bin/cert-manager-csi-driver-spiffe-oci -f Dockerfile.driver .
	docker buildx build --platform=$(IMAGE_PLATFORMS) -t quay.io/jetstack/cert-manager-csi-driver-spiffe-approver:v$(RELEASE_VERSION) --output type=oci,dest=./bin/cert-manager-csi-driver-spiffe-approver-oci -f Dockerfile.approver .

# TODO: ideally we should ensure that image and image-push are identical save for the different output location (or we should use ko instead)
# for now, we copy+paste the build steps to avoid the need for a manual edit to the Makefile in order to do a release
# This allows us to release from a non-dirty checkout of a tag.

.PHONY: image-push
image-push: ## build docker images for all supported platforms and push to the remote registry
	docker buildx build --platform=$(IMAGE_PLATFORMS) -t quay.io/jetstack/cert-manager-csi-driver-spiffe:v$(RELEASE_VERSION) --push -f Dockerfile.driver .
	docker buildx build --platform=$(IMAGE_PLATFORMS) -t quay.io/jetstack/cert-manager-csi-driver-spiffe-approver:v$(RELEASE_VERSION) --push -f Dockerfile.approver .

.PHONY: demo
demo: depend ## create cluster and deploy approver-policy
	REPO_ROOT=$(shell pwd) ./hack/ci/create-cluster.sh

.PHONY: e2e
e2e: demo ## create cluster, deploy csi-driver-spiffe, run e2e tests
	REPO_ROOT=$(shell pwd) ./hack/ci/run-e2e-test.sh
	REPO_ROOT=$(shell pwd) ./hack/ci/delete-cluster.sh

.PHONY: chart
chart: | $(BINDIR)/helm $(BINDIR)/chart
	$(BINDIR)/helm package --app-version=$(RELEASE_VERSION) --version=$(RELEASE_VERSION) --destination "$(BINDIR)/chart" ./deploy/charts/csi-driver-spiffe

.PHONY: depend
depend: $(BINDIR) $(BINDIR)/ginkgo $(BINDIR)/kubectl $(BINDIR)/kind $(BINDIR)/helm $(BINDIR)/kubebuilder/bin/kube-apiserver $(BINDIR)/cert-manager/crds.yaml $(BINDIR)/cmctl $(BINDIR)/helm-docs

$(BINDIR) $(BINDIR)/chart:
	mkdir -p $@

$(BINDIR)/ginkgo:
	go build -o $(BINDIR)/ginkgo github.com/onsi/ginkgo/ginkgo

$(BINDIR)/kind:
	go build -o $(BINDIR)/kind sigs.k8s.io/kind

$(BINDIR)/helm: $(BINDIR)/helm-v$(HELM_VERSION)-$(OS)-$(ARCH).tar.gz | $(BINDIR)
	tar xfO $< $(OS)-$(ARCH)/helm > $@
	chmod +x $@

$(BINDIR)/helm-v$(HELM_VERSION)-$(OS)-$(ARCH).tar.gz: | $(BINDIR)
	curl -o $@ -LO "https://get.helm.sh/helm-v$(HELM_VERSION)-$(OS)-$(ARCH).tar.gz"

$(BINDIR)/kubectl:
	curl -o ./bin/kubectl -LO "https://storage.googleapis.com/kubernetes-release/release/$(shell curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/$(OS)/$(ARCH)/kubectl"
	chmod +x ./bin/kubectl

$(BINDIR)/kubebuilder/bin/kube-apiserver:
	curl -SLo $(BINDIR)/envtest-bins.tar.gz "https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-$(KUBEBUILDER_TOOLS_VERISON)-$(OS)-$(ARCH).tar.gz"
	mkdir -p $(BINDIR)/kubebuilder
	tar -C $(BINDIR)/kubebuilder --strip-components=1 -zvxf $(BINDIR)/envtest-bins.tar.gz

$(BINDIR)/cmctl:
	go build -o $(BINDIR)/cmctl github.com/cert-manager/cert-manager/cmd/ctl

$(BINDIR)/cert-manager/crds.yaml:
	mkdir -p $(BINDIR)/cert-manager
	curl -SLo $(BINDIR)/cert-manager/crds.yaml https://github.com/cert-manager/cert-manager/releases/download/$(shell curl --silent "https://api.github.com/repos/cert-manager/cert-manager/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')/cert-manager.crds.yaml

$(BINDIR)/helm-docs:
	go build -o $(BINDIR)/helm-docs github.com/norwoodj/helm-docs/cmd/helm-docs
