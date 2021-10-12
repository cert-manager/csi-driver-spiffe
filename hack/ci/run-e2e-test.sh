#!/bin/sh
set -o errexit

REPO_ROOT="${REPO_ROOT:-$(dirname "${BASH_SOURCE}")/../..}"
BINDIR="${BINDIR:-$(pwd)/bin}"

echo ">> running e2e tests"
${BINDIR}/kind get kubeconfig --name csi-driver-spiffe > ${BINDIR}/kubeconfig.yaml
ARTIFACTS=${REPO_ROOT}/_artifacts ${BINDIR}/ginkgo -nodes 1 $REPO_ROOT/test/e2e/. -- --kubeconfig-path ${BINDIR}/kubeconfig.yaml --kubectl-path ${BINDIR}/kubectl
