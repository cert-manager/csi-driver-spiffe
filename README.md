<p align="center">
  <img src="https://raw.githubusercontent.com/cert-manager/cert-manager/d53c0b9270f8cd90d908460d69502694e1838f5f/logo/logo-small.png" height="256" width="256" alt="cert-manager project logo" />
  <br>
  <a href="https://godoc.org/github.com/cert-manager/csi-driver-spiffe"><img src="https://godoc.org/github.com/cert-manager/csi-driver-spiffe?status.svg"></a>
  <a href="https://goreportcard.com/report/github.com/cert-manager/csi-driver-spiffe"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/cert-manager/csi-driver-spiffe" /></a>
</p>

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

## Documentation

Please follow the documentation at
[cert-manager.io](https://cert-manager.io/docs/projects/csi-driver-spiffe/) for
installing and using csi-driver-spiffe.
