# cert-manager csi-driver-spiffe

<!-- see https://artifacthub.io/packages/helm/cert-manager/cert-manager-csi-driver-spiffe for the rendered version -->

## Helm Values

<!-- AUTO-GENERATED -->

#### **image.repository.driver** ~ `string`
> Default value:
> ```yaml
> quay.io/jetstack/cert-manager-csi-driver-spiffe
> ```
#### **image.repository.approver** ~ `string`
> Default value:
> ```yaml
> quay.io/jetstack/cert-manager-csi-driver-spiffe-approver
> ```
#### **image.tag** ~ `string`
> Default value:
> ```yaml
> v0.0.0
> ```

Target image version tag.
#### **image.pullPolicy** ~ `string`
> Default value:
> ```yaml
> IfNotPresent
> ```

Kubernetes imagePullPolicy on DaemonSet.
#### **imagePullSecrets** ~ `array`
> Default value:
> ```yaml
> []
> ```

Optional secrets used for pulling the csi-driver-spiffe and csi-driver-spiffe-approver container images  
  
For example:

```yaml
imagePullSecrets:
- name: secret-name
```
#### **app.logLevel** ~ `number`
> Default value:
> ```yaml
> 1
> ```

Verbosity of cert-manager-csi-driver logging.
#### **app.certificateRequestDuration** ~ `string`
> Default value:
> ```yaml
> 1h
> ```

Duration requested for requested certificates.
#### **app.extraCertificateRequestAnnotations** ~ `unknown`
> Default value:
> ```yaml
> null
> ```

List of annotations to add to certificate requests  
  
For example:

```yaml
extraCertificateRequestAnnotations: app=csi-driver-spiffe,foo=bar
```
#### **app.trustDomain** ~ `string`
> Default value:
> ```yaml
> cluster.local
> ```

The Trust Domain for this driver.
#### **app.name** ~ `string`
> Default value:
> ```yaml
> spiffe.csi.cert-manager.io
> ```

The name for the CSI driver installation.
#### **app.issuer.name** ~ `string`
> Default value:
> ```yaml
> spiffe-ca
> ```

Issuer name which is used to serve this Trust Domain.
#### **app.issuer.kind** ~ `string`
> Default value:
> ```yaml
> ClusterIssuer
> ```

Issuer kind which is used to serve this Trust Domain.
#### **app.issuer.group** ~ `string`
> Default value:
> ```yaml
> cert-manager.io
> ```

Issuer group which is used to serve this Trust Domain.
#### **app.driver.sourceCABundle** ~ `unknown`
> Default value:
> ```yaml
> null
> ```

Optional file containing a CA bundle that will be propagated to managed volumes.
#### **app.driver.volumeFileName.cert** ~ `string`
> Default value:
> ```yaml
> tls.crt
> ```

File name which signed certificates are written to in volumes.
#### **app.driver.volumeFileName.key** ~ `string`
> Default value:
> ```yaml
> tls.key
> ```

File name which private keys are written to in volumes.
#### **app.driver.volumeFileName.ca** ~ `string`
> Default value:
> ```yaml
> ca.crt
> ```

File name where the CA bundles are written to, if enabled.
#### **app.driver.volumes** ~ `array`
> Default value:
> ```yaml
> []
> ```

Optional extra volumes. Useful for mounting root CAs  
  
For example:

```yaml
volumes:
- name: root-cas
  secret:
    secretName: root-ca-bundle
```
#### **app.driver.volumeMounts** ~ `array`
> Default value:
> ```yaml
> []
> ```

Optional extra volume mounts. Useful for mounting root CAs  
  
For example:

```yaml
volumeMounts:
- name: root-cas
  mountPath: /var/run/secrets/cert-manager-csi-driver-spiffe
```
#### **app.driver.csiDataDir** ~ `string`
> Default value:
> ```yaml
> /tmp/cert-manager-csi-driver
> ```

Configures the hostPath directory that the driver will write and mount volumes from.
#### **app.driver.resources** ~ `object`
> Default value:
> ```yaml
> {}
> ```

Kubernetes pod resource limits for cert-manager-csi-driver-spiffe  
  
For example:

```yaml
resources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 100m
    memory: 128Mi
```
#### **app.driver.nodeDriverRegistrarImage.repository** ~ `string`
> Default value:
> ```yaml
> registry.k8s.io/sig-storage/csi-node-driver-registrar
> ```

Target image repository.
#### **app.driver.nodeDriverRegistrarImage.tag** ~ `string`
> Default value:
> ```yaml
> v2.10.0
> ```

Target image version tag.
#### **app.driver.nodeDriverRegistrarImage.pullPolicy** ~ `string`
> Default value:
> ```yaml
> IfNotPresent
> ```

Kubernetes imagePullPolicy on node-driver.
#### **app.driver.livenessProbeImage.repository** ~ `string`
> Default value:
> ```yaml
> registry.k8s.io/sig-storage/livenessprobe
> ```

Target image repository.
#### **app.driver.livenessProbeImage.tag** ~ `string`
> Default value:
> ```yaml
> v2.12.0
> ```

Target image version tag.
#### **app.driver.livenessProbeImage.pullPolicy** ~ `string`
> Default value:
> ```yaml
> IfNotPresent
> ```

Kubernetes imagePullPolicy on liveness probe.
#### **app.driver.livenessProbe.port** ~ `number`
> Default value:
> ```yaml
> 9809
> ```

The port that will expose the liveness of the csi-driver
#### **app.approver.replicaCount** ~ `number`
> Default value:
> ```yaml
> 1
> ```

Number of replicas of the approver to run.
#### **app.approver.signerName** ~ `string`
> Default value:
> ```yaml
> clusterissuers.cert-manager.io/*
> ```

The signer name that csi-driver-spiffe approver will be given permission to approve and deny. CertificateRequests referencing this signer name can be processed by the SPIFFE approver. See: https://cert-manager.io/docs/concepts/certificaterequest/#approval
#### **app.approver.readinessProbe.port** ~ `number`
> Default value:
> ```yaml
> 6060
> ```

Container port to expose csi-driver-spiffe-approver HTTP readiness probe on default network interface.
#### **app.approver.metrics.port** ~ `number`
> Default value:
> ```yaml
> 9402
> ```

Port for exposing Prometheus metrics on 0.0.0.0 on path '/metrics'.
#### **app.approver.metrics.service.enabled** ~ `bool`
> Default value:
> ```yaml
> true
> ```

Create a Service resource to expose metrics endpoint.
#### **app.approver.metrics.service.type** ~ `string`
> Default value:
> ```yaml
> ClusterIP
> ```

Service type to expose metrics.
#### **app.approver.metrics.service.servicemonitor.enabled** ~ `bool`
> Default value:
> ```yaml
> false
> ```
#### **app.approver.metrics.service.servicemonitor.prometheusInstance** ~ `string`
> Default value:
> ```yaml
> default
> ```
#### **app.approver.metrics.service.servicemonitor.interval** ~ `string`
> Default value:
> ```yaml
> 10s
> ```
#### **app.approver.metrics.service.servicemonitor.scrapeTimeout** ~ `string`
> Default value:
> ```yaml
> 5s
> ```
#### **app.approver.metrics.service.servicemonitor.labels** ~ `object`
> Default value:
> ```yaml
> {}
> ```
#### **app.approver.resources** ~ `object`
> Default value:
> ```yaml
> {}
> ```

Kubernetes pod resource limits for cert-manager-csi-driver-spiffe approver  
  
For example:

```yaml
resources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 100m
    memory: 128Mi
```
#### **priorityClassName** ~ `string`
> Default value:
> ```yaml
> ""
> ```

Optional priority class to be used for the csi-driver pods.

<!-- /AUTO-GENERATED -->