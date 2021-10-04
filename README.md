<p align="center"><img src="https://github.com/jetstack/cert-manager/blob/master/logo/logo.png" width="250x" /></p>
</a>
<a href="https://godoc.org/github.com/cert-manager/csi-driver-spiffe"><img src="https://godoc.org/github.com/cert-manager/csi-driver-spiffe?status.svg"></a>
<a href="https://goreportcard.com/report/github.com/cert-manager/csi-driver-spiffe"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/cert-manager/csi-driver-spiffe" /></a></p>

# csi-driver-spiffe

csi-driver-spiffe is a Container Storage Interface (CSI) driver plugin for
Kubernetes to work along [cert-manager](https://cert-manager.io/). This CSI driver
transparently delivers [SPIFFE](https://spiffe.io/)
[SVIDs](https://spiffe.io/docs/latest/spiffe-about/spiffe-concepts/#spiffe-verifiable-identity-document-svid)
in the form of X.509 certificate key pairs to mounting Kubernetes Pods.

The end result is all and any Pod running in Kubernetes can securely request
their SPIFFE identity document from a Trust Domain with minimal configuration.
These documents are:
- automatically renewed; :heavy_check_mark:
- private key never leaves the node's virtual memory; :heavy_check_mark:
- each Pod's document is unique; :heavy_check_mark:
- the document shares the same life cycle as the Pod and is destroyed on Pod termination. :heavy_check_mark:

```yaml
...
          volumeMounts:
          - mountPath: "/var/run/secrets/spiffe.io"
            name: spiffe
      volumes:
        - name: spiffe
          csi:
            driver: spiffe.csi.cert-manager.io
            readOnly: true
```

SPIFFE documents can be used for mutual TLS (mTLS) or authentication by Pod's
within its Trust Domain.


### Components

The project is split into two components;

##### CSI Driver

The CSI driver runs as DaemonSet on the cluster which is responsible for
generating, requesting, and mounting the certificate key pair to Pods on the
node it manages. The CSI driver creates and manages a
[tmpfs](https://www.kernel.org/doc/html/latest/filesystems/tmpfs.html) directory
which is used to create and mount Pod volumes from.

When a Pod is created with the CSI volume configured, the
driver will locally generate a private key, and create a cert-manager
[CertificateRequest](https://cert-manager.io/docs/concepts/certificaterequest/)
in the same Namespace as the Pod.

The driver uses [CSI Token
Request](https://kubernetes-csi.github.io/docs/token-requests.html) to both
discover the Pod's identity to form the SPIFFE identity contained in the X.509
certificate signing request, as well as securely impersonate its ServiceAccount
when creating the CertificateRequest.

Once signed by the pre-configured target signer, the driver will mount the
private key and signed certificate into the Pod's Volume to be made available as
a Volume Mount. This certificate key pair is regularly renewed based on the
expiry of the signed certificate.

##### Approver

A distinct
[cert-manager approver](https://cert-manager.io/docs/concepts/certificaterequest/#approval)
Deployment is responsible for managing the approval and denial condition of
created CertificateRequests that target the configured SPIFFE Trust Domain
signer. The approver ensures that requests have:

1. the correct key type (ECDSA P-521);
2. acceptable key usages (Key Encipherment, Digital Signature, Client Auth, Server Auth);
3. the requested duration matches the enforced duration (default 1 hour);
4. no [SANs](https://en.wikipedia.org/wiki/Subject_Alternative_Name) or other
   identifiable attributes except a single [URI SANs](https://en.wikipedia.org/wiki/Uniform_Resource_Identifier);
5. the single URI SAN is the SPIFFE identity of the ServiceAccount who created
   the CertificateRequest;
5. the SPIFFE ID Trust Domain is the same as configured.

If any of these checks do not pass, the CertificateRequest will be marked as
Denied, else it will be marked as Approved. The approver will only manage
CertificateRequests who request from the same
[IssuerRef](https://cert-manager.io/docs/concepts/certificaterequest/) that has
been configured.


## Installation

1. [cert-manager](https://cert-manager.io) is required to be installed with
   csi-driver-spiffe. :warning: requires cert-manager v1.3 or higher.

> :warning:
>
> It is important that the
> [default approver is disabled in cert-manager](https://cert-manager.io/docs/concepts/certificaterequest/#approver-controller).
> If the default approver is not disabled in cert-manager, the csi-driver-spiffe approver will
> race with cert-manager and thus its policy enforcement becomes useless.
>
> ```terminal
> $ helm repo add jetstack https://charts.jetstack.io --force-update
> $ helm upgrade -i -n cert-manager cert-manager jetstack/cert-manager --set extraArgs={--controllers='*\,-certificaterequests-approver'} --set installCRDs=true --create-namespace
> ```
>
> :warning:

2. Install or configure a
   [ClusterIssuer](https://cert-manager.io/docs/configuration/) to give
   cert-manager the ability to sign against your Trust Domain. If a namespace
   scoped Issuer is desired, then that Issuer must be created in every namespace
   that Pods will mount volumes from.
   You must use an Issuer type which is compatible with signing URI SAN
   certificates and the private does not need to be available to the signer, for
   example [CA](https://cert-manager.io/docs/configuration/ca/),
   [Vault](https://cert-manager.io/docs/configuration/vault/),
   [Venafi](https://cert-manager.io/docs/configuration/venafi/),
   [AWS PCA](https://github.com/cert-manager/aws-privateca-issuer),
   [Google CAS](https://github.com/jetstack/google-cas-issuer),
   [Small Step](https://github.com/smallstep/step-issuer). Issuers such as
   [SelfSigned](https://cert-manager.io/docs/configuration/selfsigned/) or
   [ACME](https://cert-manager.io/docs/configuration/acme/) *will not work*.

   An example demo
   [ClusterIssuer](https://cert-manager.io/docs/concepts/issuer/#namespaces) can
   be found [here](deploy/example/clusterissuer.yaml). This Trust Domain's root
   CA is self-signed by cert-manager and *private key is stored in the cluster*.

```terminal
$ kubectl apply -f ./deploy/example/clusterissuer.yaml
# We must also approve the CertificateRequest since we have disabled the default approver
$ kubectl cert-manager approve -n cert-manager $(kubectl get cr -n cert-manager -ojsonpath='{.items[0].metadata.name}')
```

3. Install csi-driver-spiffe into the cluster using the issuer we configured. We
   must also configure the issuer resource type and name of the issuer we
   configured so that the approver has
   [permissions to approve referencing CertificateRequests](https://cert-manager.io/docs/concepts/certificaterequest/#rbac-syntax).

  - Change signer name to match your issuer type.
  - Change name, kind, and group to your issuer.
```terminal
$ helm upgrade -i -n cert-manager cert-manager-csi-driver-spiffe jetstack/cert-manager-csi-driver-spiffe --wait \
  --set "app.logLevel=1" \
  --set "app.trustDomain=my.trust.domain" \
  --set "app.approver.signerName=clusterissuers.cert-manager.io/csi-driver-spiffe-ca" \
  \
  --set "app.issuer.name=csi-driver-spiffe-ca" \
  --set "app.issuer.kind=ClusterIssuer" \
  --set "app.issuer.group=cert-manager.io"
```

## Usage

Once the driver is successfully installed, Pods can begin to request and mount
their key and SPIFFE certificate. Since the Pod's ServiceAccount is impersonated
when creating CertificateRequests, every ServiceAccount must be given that
permission which intends to use the volume.

Example manifest with a dummy Deployment:

```terminal
$ kubectl apply -f ./deploy/example/example-app.yaml

$ kubectl exec -n sandbox $(kubectl get pod -n sandbox -l app=my-csi-app -o jsonpath='{.items[0].metadata.name}') -- cat /var/run/secrets/spiffe.io/tls.crt | openssl x509 --noout --text | grep Issuer:
        Issuer: CN = csi-driver-spiffe-ca
$ kubectl exec -n sandbox $(kubectl get pod -n sandbox -l app=my-csi-app -o jsonpath='{.items[0].metadata.name}') -- cat /var/run/secrets/spiffe.io/tls.crt | openssl x509 --noout --text | grep URI:
                URI:spiffe://foo.bar/ns/sandbox/sa/example-app
```

### FS-Group

When running Pods with a specified user or group, the volume will not be
readable by default due to Unix based file system permissions. The mounting
volumes file group can be specified using the following volume attribute:

```yaml
...
      securityContext:
        runAsUser: 123
        runAsGroup: 456
      volumes:
        - name: spiffe
          csi:
            driver: spiffe.csi.cert-manager.io
            readOnly: true
            volumeAttributes:
              spiffe.csi.cert-manager.io/fs-group: "456"
```

```terminal
$ kubectl apply -f ./deploy/example/fs-group-app.yaml

$ kubectl exec -n sandbox $(kubectl get pod -n sandbox -l app=my-csi-app-fs-group -o jsonpath='{.items[0].metadata.name}') -- cat /var/run/secrets/spiffe.io/tls.crt | openssl x509 --noout --text | grep URI:
                URI:spiffe://foo.bar/ns/sandbox/sa/fs-group-app
```

### Root CA Bundle

By default, the CSI driver will only mount the Pod's private key and signed
certificate. csi-driver-spiffe can be optionally configured to also mount a
statically defined CA bundle from a volume that will be written to all Pod
volumes.

The following example mounts the CA certificate used by the Trust Domain
ClusterIssuer.

```terminal
$ helm upgrade -i -n cert-manager cert-manager-csi-driver-spiffe jetstack/cert-manager-csi-driver-spiffe --wait \
  --set "app.logLevel=1" \
  --set "app.trustDomain=my.trust.domain" \
  --set "app.approver.signerName=clusterissuers.cert-manager.io/csi-driver-spiffe-ca" \
  \
  --set "app.issuer.name=csi-driver-spiffe-ca" \
  --set "app.issuer.kind=ClusterIssuer" \
  --set "app.issuer.group=cert-manager.io" \
  \
  --set "app.driver.volumes[0].name=root-cas" \
  --set "app.driver.volumes[0].secret.secretName=csi-driver-spiffe-ca" \
  --set "app.driver.volumeMounts[0].name=root-cas" \
  --set "app.driver.volumeMounts[0].mountPath=/var/run/secrets/cert-manager-csi-driver-spiffe" \
  --set "app.driver.sourceCABundle=/var/run/secrets/cert-manager-csi-driver-spiffe/ca.crt"
$ kubectl rollout restart deployment -n sandbox my-csi-app
$ kubectl exec -it -n sandbox $(kubectl get pod -n sandbox -l app=my-csi-app -o jsonpath='{.items[0].metadata.name}') -- ls /var/run/secrets/spiffe.io/
ca.crt   tls.crt  tls.key
```
