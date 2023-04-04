# cert-manager-csi-driver-spiffe

![Version: v0.3.1](https://img.shields.io/badge/Version-v0.3.1-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: v0.3.1](https://img.shields.io/badge/AppVersion-v0.3.1-informational?style=flat-square)

cert-manager csi-driver-spiffe is a CSI plugin for Kubernetes which transparently delivers X.509 SPIFFE SVIDs to pods which mount it.

**Homepage:** <https://cert-manager.io/docs/projects/csi-driver-spiffe/>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| cert-manager-maintainers | <cert-manager-maintainers@googlegroups.com> | <https://cert-manager.io> |

## Source Code

* <https://github.com/cert-manager/csi-driver-spiffe>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| app.approver.metrics.service | object | `{"enabled":true,"servicemonitor":{"enabled":false,"interval":"10s","labels":{},"prometheusInstance":"default","scrapeTimeout":"5s"},"type":"ClusterIP"}` | Service to expose metrics endpoint. |
| app.approver.metrics.service.enabled | bool | `true` | Create a Service resource to expose metrics endpoint. |
| app.approver.metrics.service.servicemonitor | object | `{"enabled":false,"interval":"10s","labels":{},"prometheusInstance":"default","scrapeTimeout":"5s"}` | ServiceMonitor resource for this Service. |
| app.approver.metrics.service.type | string | `"ClusterIP"` | Service type to expose metrics. |
| app.approver.readinessProbe.port | int | `6060` | Container port to expose csi-driver-spiffe-approver HTTP readiness probe on default network interface. |
| app.approver.replicaCount | int | `1` | Number of replicas of the approver to run. |
| app.approver.signerName | string | `"clusterissuers.cert-manager.io/*"` | The signer name that csi-driver-spiffe approver will be given permission to approve and deny. CertificateRequests referencing this signer name can be processed by the SPIFFE approver. See: https://cert-manager.io/docs/concepts/certificaterequest/#approval |
| app.certificateRequestDuration | string | `"1h"` | Duration requested for requested certificates. |
| app.driver | object | `{"csiDataDir":"/tmp/cert-manager-csi-driver", "livenessProbe":{"port":9809},"livenessProbeImage":{"pullPolicy":"IfNotPresent","repository":"registry.k8s.io/sig-storage/livenessprobe","tag":"v2.9.0"},"nodeDriverRegistrarImage":{"pullPolicy":"IfNotPresent","repository":"registry.k8s.io/sig-storage/csi-node-driver-registrar","tag":"v2.7.0"},"resources":{},"sourceCABundle":null,"volumeFileName":{"ca":"ca.crt","cert":"tls.crt","key":"tls.key"},"volumeMounts":[],"volumes":[]}` | Options for CSI driver |
| app.driver.csiDataDir | string | `"/tmp/cert-manager-csi-driver"` | Configures the hostPath directory that the driver will write and mount volumes from. |
| app.driver.livenessProbe.port | int | `9809` | The port that will expose the liveness of the csi-driver |
| app.driver.livenessProbeImage.pullPolicy | string | `"IfNotPresent"` | Kubernetes imagePullPolicy on liveness probe. |
| app.driver.livenessProbeImage.repository | string | `"registry.k8s.io/sig-storage/livenessprobe"` | Target image repository. |
| app.driver.livenessProbeImage.tag | string | `"v2.9.0"` | Target image version tag. |
| app.driver.nodeDriverRegistrarImage.pullPolicy | string | `"IfNotPresent"` | Kubernetes imagePullPolicy on node-driver. |
| app.driver.nodeDriverRegistrarImage.repository | string | `"registry.k8s.io/sig-storage/csi-node-driver-registrar"` | Target image repository. |
| app.driver.nodeDriverRegistrarImage.tag | string | `"v2.7.0"` | Target image version tag. |
| app.driver.sourceCABundle | string | `nil` | Optional file containing a CA bundle that will be propagated to managed volumes. |
| app.driver.volumeFileName.ca | string | `"ca.crt"` | File name where the CA bundles are written to, if enabled. |
| app.driver.volumeFileName.cert | string | `"tls.crt"` | File name which signed certificates are written to in volumes. |
| app.driver.volumeFileName.key | string | `"tls.key"` | File name which private keys are written to in volumes. |
| app.driver.volumeMounts | list | `[]` | Optional extra volume mounts. Useful for mounting root CAs |
| app.driver.volumes | list | `[]` | Optional extra volumes. Useful for mounting root CAs |
| app.issuer.group | string | `"cert-manager.io"` | Issuer group which is used to serve this Trust Domain. |
| app.issuer.kind | string | `"ClusterIssuer"` | Issuer kind which is used to serve this Trust Domain. |
| app.issuer.name | string | `"spiffe-ca"` | Issuer name which is used to serve this Trust Domain. |
| app.logLevel | int | `1` | Verbosity of cert-manager-csi-driver logging. |
| app.name | string | `"spiffe.csi.cert-manager.io"` | The name for the CSI driver installation. |
| app.trustDomain | string | `"cluster.local"` | The Trust Domain for this driver. |
| image.pullPolicy | string | `"IfNotPresent"` | Kubernetes imagePullPolicy on DaemonSet. |
| image.repository | object | `{"approver":"quay.io/jetstack/cert-manager-csi-driver-spiffe-approver","driver":"quay.io/jetstack/cert-manager-csi-driver-spiffe"}` | Target image repository. |
| image.tag | string | `"v0.3.1"` | Target image version tag. |
| imagePullSecrets | list | `[]` | Optional secrets used for pulling the csi-driver-spiffe and csi-driver-spiffe-approver container images |
| priorityClassName | string | `""` | Optional priority class to be used for the csi-driver pods. |

