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

repo_name := github.com/cert-manager/csi-driver-spiffe

kind_cluster_name := csi-driver-spiffe
kind_cluster_config := $(bin_dir)/scratch/kind_cluster.yaml

build_names := manager approver

go_manager_main_dir := ./cmd/csi
go_manager_mod_dir := .
go_manager_ldflags := -X $(repo_name)/internal/version.AppVersion=$(VERSION) -X $(repo_name)/internal/version.GitCommit=$(GITCOMMIT)
oci_manager_base_image_flavor := csi-static
oci_manager_image_name := quay.io/jetstack/cert-manager-csi-driver-spiffe
oci_manager_image_tag := $(VERSION)
oci_manager_image_name_development := cert-manager.local/cert-manager-csi-driver-spiffe

go_approver_main_dir := ./cmd/approver
go_approver_mod_dir := .
go_approver_ldflags := -X $(repo_name)/internal/version.AppVersion=$(VERSION) -X $(repo_name)/internal/version.GitCommit=$(GITCOMMIT)
oci_approver_base_image_flavor := static
oci_approver_image_name := quay.io/jetstack/cert-manager-csi-driver-spiffe-approver
oci_approver_image_tag := $(VERSION)
oci_approver_image_name_development := cert-manager.local/cert-manager-csi-driver-spiffe-approver

deploy_name := csi-driver-spiffe
deploy_namespace := cert-manager

api_docs_outfile := docs/api/api.md
api_docs_package := $(repo_name)/pkg/apis/trust/v1alpha1
api_docs_branch := main

helm_chart_source_dir := deploy/charts/csi-driver-spiffe
helm_chart_image_name := quay.io/jetstack/charts/cert-manager-csi-driver-spiffe
helm_chart_version := $(VERSION)
helm_labels_template_name := cert-manager-csi-driver-spiffe.labels

golangci_lint_config := .golangci.yaml

define helm_values_mutation_function
$(YQ) \
	'( .image.repository.driver = "$(oci_manager_image_name)" ) | \
	( .image.repository.approver = "$(oci_approver_image_name)" ) | \
	( .image.tag = "$(oci_manager_image_tag)" )' \
	$1 --inplace
endef

images_amd64 ?=
images_arm64 ?=

images_amd64 += docker.io/library/busybox:1.36.1-musl@sha256:3e4c072f6179b8698630d94422ca32effec45a2e2a7ae1d76cf6b5410ee06d33
images_arm64 += docker.io/library/busybox:1.36.1-musl@sha256:b9d37d21fdb80a3262b4f59a802f00cede89937a63ded366bc4f647a5032a4db
