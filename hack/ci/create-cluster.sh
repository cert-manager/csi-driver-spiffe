#!/bin/sh
set -o errexit

REPO_ROOT="${REPO_ROOT:-$(dirname "${BASH_SOURCE}")/../..}"
KUBECTL_BIN="${KUBECTL_BIN:-$REPO_ROOT/bin/kubectl}"
CMCTL_BIN="${CMCTL_BIN:-$REPO_ROOT/bin/cmctl}"
HELM_BIN="${HELM_BIN:-$REPO_ROOT/bin/helm}"
KIND_BIN="${KIND_BIN:-$REPO_ROOT/bin/kind}"
CSI_TAG="${CSI_TAG:-smoke}"
CSI_DRIVER_REPO="${CSI_DRIVER_REPO:-quay.io/jetstack/cert-manager-csi-driver-spiffe}"
CSI_DRIVER_IMAGE="$CSI_DRIVER_REPO:$CSI_TAG"
CSI_APPROVER_REPO="${CSI_APPROVER_REPO:-quay.io/jetstack/cert-manager-csi-driver-spiffe-approver}"
CSI_APPROVER_IMAGE="$CSI_APPROVER_REPO:$CSI_TAG"

echo ">> building docker images..."
docker build -t $CSI_DRIVER_IMAGE -f Dockerfile.driver .
docker build -t $CSI_APPROVER_IMAGE -f Dockerfile.approver .


echo ">> pre-creating 'kind' docker network to avoid networking issues in CI"
# When running in our CI environment the Docker network's subnet choice will cause issues with routing
# This works this around till we have a way to properly patch this.
docker network create --driver=bridge --subnet=192.168.0.0/16 --gateway 192.168.0.1 kind || true
# Sleep for 2s to avoid any races between docker's network subcommand and 'kind create'
sleep 2

echo ">> creating kind cluster..."
$KIND_BIN delete cluster --name csi-driver-spiffe
$KIND_BIN create cluster --name csi-driver-spiffe

echo ">> loading docker image..."
$KIND_BIN load docker-image $CSI_DRIVER_IMAGE --name csi-driver-spiffe
$KIND_BIN load docker-image $CSI_APPROVER_IMAGE --name csi-driver-spiffe

echo ">> installing cert-manager..."
$HELM_BIN repo add jetstack https://charts.jetstack.io --force-update
$HELM_BIN upgrade -i -n cert-manager cert-manager jetstack/cert-manager --set installCRDs=true --wait --create-namespace --set extraArgs={--controllers='*\,-certificaterequests-approver'} --set global.logLevel=2


echo ">> installing issuer from self-signed Trust Domain"
$KUBECTL_BIN apply -f $REPO_ROOT/deploy/example/clusterissuer.yaml
sleep 3
$CMCTL_BIN approve -n cert-manager $($KUBECTL_BIN get cr -n cert-manager --no-headers -o custom-columns=":metadata.name")
$KUBECTL_BIN wait --for=condition=ready clusterissuer csi-driver-spiffe-ca

echo ">> installing csi-drive-spiffe..."
$HELM_BIN upgrade -i -n cert-manager cert-manager-csi-driver-spiffe ./deploy/charts/csi-driver-spiffe --wait \
  --set app.logLevel=2 \
  --set image.repository.driver=$CSI_DRIVER_REPO \
  --set image.repository.approver=$CSI_APPROVER_REPO \
  --set image.tag=$CSI_TAG \
  --set app.trustDomain=foo.bar \
  --set app.approver.signerName=clusterissuers.cert-manager.io/csi-driver-spiffe-ca \
  --set app.issuer.name=csi-driver-spiffe-ca \
  --set app.driver.volumes[0].name=root-cas \
  --set app.driver.volumes[0].secret.secretName=csi-driver-spiffe-ca \
  --set app.driver.volumeMounts[0].name=root-cas \
  --set app.driver.volumeMounts[0].mountPath=/var/run/secrets/cert-manager-csi-driver-spiffe \
  --set app.driver.sourceCABundle=/var/run/secrets/cert-manager-csi-driver-spiffe/ca.crt
