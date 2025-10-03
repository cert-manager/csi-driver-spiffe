# cert-manager csi-driver-spiffe

<!-- see https://artifacthub.io/packages/helm/cert-manager/cert-manager-csi-driver-spiffe for the rendered version -->

## Helm Values

<!-- AUTO-GENERATED -->

#### **image.registry** ~ `string`

Target image registry. This value is prepended to the target image repository, if set.  
For example:

```yaml
registry: quay.io
repository:
  driver: jetstack/cert-manager-csi-driver-spiffe
  approver: jetstack/cert-manager-csi-driver-spiffe-approver
```

#### **image.repository.driver** ~ `string`
> Default value:
> ```yaml
> quay.io/jetstack/cert-manager-csi-driver-spiffe
> ```

Target image repository for the csi-driver driver DaemonSet.
#### **image.repository.approver** ~ `string`
> Default value:
> ```yaml
> quay.io/jetstack/cert-manager-csi-driver-spiffe-approver
> ```

Target image repository for the csi-driver approver Deployment.
#### **image.tag** ~ `string`

Override the image tag to deploy by setting this variable. If no value is set, the chart's appVersion is used.

#### **image.digest** ~ `object`
> Default value:
> ```yaml
> {}
> ```
#### **image.digest.driver** ~ `string`

Target csi-driver driver digest. Override any tag, if set.  
For example:

```yaml
driver: sha256:0e072dddd1f7f8fc8909a2ca6f65e76c5f0d2fcfb8be47935ae3457e8bbceb20
```

#### **image.digest.approver** ~ `string`

Target csi-driver approver digest. Override any tag, if set.  
For example:

```yaml
approver: sha256:0e072dddd1f7f8fc8909a2ca6f65e76c5f0d2fcfb8be47935ae3457e8bbceb20
```

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
#### **app.runtimeIssuanceConfigMap** ~ `string`
> Default value:
> ```yaml
> ""
> ```

Name of a ConfigMap in the installation namespace to watch, providing runtime configuration of an issuer to use.  
  
The "issuer-name", "issuer-kind" and "issuer-group" keys must be present in the ConfigMap for it to be used.
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
> v2.15.0@sha256:11f199f6bec47403b03cb49c79a41f445884b213b382582a60710b8c6fdc316a
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
> v2.17.0@sha256:9b75b9ade162136291d5e8f13a1dfc3dec71ee61419b1bfc112e0796ff8a6aa9
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
#### **app.approver.replicaCount** ~ `number`
> Default value:
> ```yaml
> 1
> ```

Number of replicas of the approver to run.
#### **app.approver.signerName** ~ `string`
> Default value:
> ```yaml
> ""
> ```

A signer name that the csi-driver-spiffe approver will be given permission to approve and deny. CertificateRequests referencing this signer name can be processed by the SPIFFE approver. See: https://cert-manager.io/docs/concepts/certificaterequest/#approval. Defaults to empty which allows approval for all signers
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

Create Prometheus ServiceMonitor resource for cert-manager-csi-driver-spiffe approver.
#### **app.approver.metrics.service.servicemonitor.prometheusInstance** ~ `string`
> Default value:
> ```yaml
> default
> ```

The value for the "prometheus" label on the ServiceMonitor. This allows for multiple Prometheus instances selecting difference ServiceMonitors using label selectors.
#### **app.approver.metrics.service.servicemonitor.interval** ~ `string`
> Default value:
> ```yaml
> 10s
> ```

The interval that the Prometheus will scrape for metrics.
#### **app.approver.metrics.service.servicemonitor.scrapeTimeout** ~ `string`
> Default value:
> ```yaml
> 5s
> ```

The timeout on each metric probe request.
#### **app.approver.metrics.service.servicemonitor.labels** ~ `object`
> Default value:
> ```yaml
> {}
> ```

Additional labels to give the ServiceMonitor resource.
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
#### **commonLabels** ~ `object`
> Default value:
> ```yaml
> {}
> ```

Labels to apply to all resources
#### **nodeSelector** ~ `object`
> Default value:
> ```yaml
> kubernetes.io/os: linux
> ```

Kubernetes node selector: node labels for pod assignment.

#### **affinity** ~ `object`
> Default value:
> ```yaml
> {}
> ```

Kubernetes affinity: constraints for pod assignment.  
  
For example:

```yaml
affinity:
  nodeAffinity:
   requiredDuringSchedulingIgnoredDuringExecution:
     nodeSelectorTerms:
     - matchExpressions:
       - key: foo.bar.com/role
         operator: In
         values:
         - master
```
#### **tolerations** ~ `array`
> Default value:
> ```yaml
> []
> ```

Kubernetes pod tolerations for cert-manager-csi-driver-spiffe.  
  
For example:

```yaml
tolerations:
- key: foo.bar.com/role
  operator: Equal
  value: master
  effect: NoSchedule
```
#### **topologySpreadConstraints** ~ `array`
> Default value:
> ```yaml
> []
> ```

List of Kubernetes TopologySpreadConstraints.  
  
For example:

```yaml
topologySpreadConstraints:
- maxSkew: 2
  topologyKey: topology.kubernetes.io/zone
  whenUnsatisfiable: ScheduleAnyway
  labelSelector:
    matchLabels:
      app.kubernetes.io/instance: cert-manager
      app.kubernetes.io/component: controller
```
#### **openshift.securityContextConstraint.enabled** ~ `boolean,string,null`
> Default value:
> ```yaml
> detect
> ```

Include RBAC to allow the DaemonSet to "use" the specified  
SecurityContextConstraints.  
  
This value can either be a boolean true or false, or the string "detect". If set to "detect" then the securityContextConstraint is automatically enabled for openshift installs.

#### **openshift.securityContextConstraint.name** ~ `string`
> Default value:
> ```yaml
> privileged
> ```

Name of the SecurityContextConstraints to create RBAC for.

<!-- /AUTO-GENERATED -->