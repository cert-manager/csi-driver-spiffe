<p align="center">
  <img src="https://raw.githubusercontent.com/cert-manager/cert-manager/d53c0b9270f8cd90d908460d69502694e1838f5f/logo/logo-small.png" height="256" width="256" alt="cert-manager project logo" />
  <br>
  <a href="https://godoc.org/github.com/cert-manager/csi-driver-spiffe"><img src="https://godoc.org/github.com/cert-manager/csi-driver-spiffe?status.svg"></a>
  <a href="https://goreportcard.com/report/github.com/cert-manager/csi-driver-spiffe"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/cert-manager/csi-driver-spiffe" /></a>
</p>

# csi-driver-spiffe

csi-driver-spiffe is a Container Storage Interface (CSI) driver plugin for
Kubernetes, designed to work alongside [cert-manager](https://cert-manager.io/).

It transparently delivers [SPIFFE](https://spiffe.io/) [SVIDs](https://spiffe.io/docs/latest/spiffe-about/spiffe-concepts/#spiffe-verifiable-identity-document-svid)
(in the form of X.509 certificate key pairs) to mounting Kubernetes Pods.

The end result is that any and all Pods running in Kubernetes can securely request
a SPIFFE identity document from a Trust Domain with minimal configuration.

These documents in turn have the following properties:

- automatically renewed ✔️
- private key never leaves the node's virtual memory ✔️
- each Pod's document is unique ✔️
- the document shares the same life cycle as the Pod and is destroyed on Pod termination ✔️

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

SPIFFE documents can then be used by Pods for mutual TLS (mTLS) or other authentication within their Trust Domain.

## Documentation

Please follow the documentation at [cert-manager.io](https://cert-manager.io/docs/projects/csi-driver-spiffe/)
for installing and using csi-driver-spiffe.

## Release Process

The release process is documented in [RELEASE.md](RELEASE.md).