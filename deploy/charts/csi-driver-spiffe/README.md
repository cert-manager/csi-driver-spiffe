# cert-manager csi-driver-spiffe

<!-- see https://artifacthub.io/packages/helm/cert-manager/cert-manager-csi-driver-spiffe for the rendered version -->

## Helm Values

<!-- AUTO-GENERATED -->

#### **image.registry** ~ `string`

Target image registry. This value is prepended to the target image repository, if set.  
For example:

```yaml
registry: quay.io
repository: jetstack/cert-manager-csi-driver-spiffe
```

#### **image.repository** ~ `string`
> Default value:
> ```yaml
> quay.io/jetstack/cert-manager-csi-driver-spiffe
> ```

Target image repository.
#### **image.tag** ~ `string`

Override the image tag to deploy by setting this variable. If no value is set, the chart's appVersion is used.

#### **image.digest** ~ `string`

Target image digest. Override any tag, if set.  
For example:

```yaml
digest: sha256:0e072dddd1f7f8fc8909a2ca6f65e76c5f0d2fcfb8be47935ae3457e8bbceb20
```

#### **image.pullPolicy** ~ `string`
> Default value:
> ```yaml
> IfNotPresent
> ```

Kubernetes imagePullPolicy on Deployment.
#### **imagePullSecrets** ~ `array`
> Default value:
> ```yaml
> []
> ```

Optional secrets used for pulling the csi-driver-spiffe container image  
  
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
#### **app.driver.nodeDriverRegistrarImage.registry** ~ `string`

Target image registry. This value is prepended to the target image repository, if set.  
For example:

```yaml
registry: registry.k8s.io
repository: sig-storage/csi-node-driver-registrar
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

Override the image tag to deploy by setting this variable. If no value is set, the chart's appVersion is used.

#### **app.driver.nodeDriverRegistrarImage.digest** ~ `string`

Target image digest. Override any tag, if set.  
For example:

```yaml
digest: sha256:0e072dddd1f7f8fc8909a2ca6f65e76c5f0d2fcfb8be47935ae3457e8bbceb20
```

#### **app.driver.nodeDriverRegistrarImage.pullPolicy** ~ `string`
> Default value:
> ```yaml
> IfNotPresent
> ```

Kubernetes imagePullPolicy on node-driver.
#### **app.driver.livenessProbeImage.registry** ~ `string`

Target image registry. This value is prepended to the target image repository, if set.  
For example:

```yaml
registry: registry.k8s.io
repository: sig-storage/livenessprobe
```

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

Override the image tag to deploy by setting this variable. If no value is set, the chart's appVersion is used.

#### **app.driver.livenessProbeImage.digest** ~ `string`

Target image digest. Override any tag, if set.  
For example:

```yaml
digest: sha256:0e072dddd1f7f8fc8909a2ca6f65e76c5f0d2fcfb8be47935ae3457e8bbceb20
```

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
#### **priorityClassName** ~ `string`
> Default value:
> ```yaml
> ""
> ```

Optional priority class to be used for the csi-driver pods.
#### **commonLabels** ~ `object`
> Default value:
> ```yaml
> {}
> ```

Labels to apply to all resources

<!-- /AUTO-GENERATED -->