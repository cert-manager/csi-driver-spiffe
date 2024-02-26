# Design: Installation Without Issuers

## Summary

This design's aim is to enable installation of all cert-manager components without a requirement
that any cert-manager resources - such as issuers - be configured beforehand.

The only components where this isn't already the case are csi-driver-spiffe and istio-csr.

Being able to install ahead of configuring an issuer is useful in, say, GitOps scenarios where
users want to install everything in a first step, then proceed to configure resources afterwards.

Specifically, a scenario we've seen is the [Venafi Manifest Tool](https://docs.venafi.cloud/vaas/k8s-components/c-vmg-overview/)
which is unable to support csi-driver-spiffe and istio-csr.

### Goals

- Users can install csi-driver-spiffe at the same time as cert-manager with no penalty
- There is no requirement for an issuer to exist at install time

### Non-Goals

- Dramatically increasing the level of control around which issuer should be used
   - csi-driver-spiffe today only allows a single issuer per cluster.
   - We're not looking to expand that in this design.
- Allowing some kind of default issuer
   - It's possible that the components could instantiate their own issuers at startup
   - That would sidestep the need for users to configure issuers, which might be a UX win
   - It would introduce problems around trust, preserving the issuer, etc
   - It might be worth exploring, but not here
- Triggering re-issuance when the issuer changes
   - This doesn't happen today - see background at the end of this document
   - This is maybe desirable at some point, but not to be solved here

## Proposed Design: Cluster Config Map

We add a new variable which specifies the name of a ConfigMap resource to be watched in the
installation namespace for the component. This variable might default to
`csi-driver-spiffe-issuer-binding` but should be configurable with a CLI parameter.

The ConfigMap might be set like this:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: csi-driver-spiffe-issuer-binding
  namespace: cert-manager # assume we're installing to the cert-manager namespace
data:
  issuer-name: my-cluster-issuer
  issuer-kind: ClusterIssuer
  issuer-group: cert-manager.io
```

We watch the ConfigMap for changes, and if it's updated we use the new issuer for any future issuance.

### Determining Which Issuer to Use

When we attempt to issue, we do the following checks:

1. Is there a configured configmap defining a cluster issuer? If so, use it
2. Was an issuer passed in on the CLI at install time (the 'old way')? If so, use it
3. Otherwise, error until at least one issuer is bound

### Potential Issue: signerName

Currently, a person installing csi-driver-spiffe must also specify a signer name to allowlist for
signing via RBAC. This happens at install time, and filters through into the [ClusterRole](https://github.com/cert-manager/csi-driver-spiffe/blob/53c87669ba063b6816c71e77234058f98b0d04f0/deploy/charts/csi-driver-spiffe/templates/clusterrole.yaml#L29):

```yaml
...
- apiGroups: ["cert-manager.io"]
  resources: ["signers"]
  verbs: ["approve"]
  resourceNames: ["{{.Values.app.approver.signerName}}"]
...
```

This might need to be tweaked. It seems OK to have the option to have no allowlist (so that
every signer is allowed) since the csi-driver-spiffe-approver is incredibly strict and also checks
the [identity](https://github.com/cert-manager/csi-driver-spiffe/blob/53c87669ba063b6816c71e77234058f98b0d04f0/internal/approver/evaluator/identity.go)
along with other fields of the CSR when making an approval decision (so it shouldn't be possible
to use the csi-driver-spiffe-approver to approve non-SPIFFE certificate requests).

If users do want to be explicit about which signers are allowed, we'll need to be sure they can
specify multiple signers at install time and not just one.

### Pros and Cons

Pros:

- Obvious parallel with current implementation
   - Closely matching current state reduces number of design choices we need to make now
- Quick to implement
- Very little to "watch" (one ConfigMap)
- No need for code-generation for a CRD

Cons:

- Entirely different method of configuring namespaces would be needed down the road
   - This is because we need to assume that users can edit configmaps in their own namespaces
   - Maybe we don't want users to be able to pick which issuer to use; that's a platform admin's role
   - Approval might help here
- csi-driver-spiffe's approver makes this more complicated
   - See "potential issuer: signerName" above
   - Can be worked around in a reasonable way

## Alternative Design: CRDs

Use CRDs to allow binding a cert-manager issuer at runtime.

An example cluster-scoped resource might be:

```yaml
apiVersion: csi-driver-spiffe.cert-manager.io/v1alpha1
kind: ClusterCSIDriverSPIFFEIssuerBinding
metadata:
  name: my-cluster-binding
spec:
  issuerRef:
    name: my-ca-cluster-issuer
    kind: ClusterIssuer
    group: cert-manager.io
```

When reconciled, a resource will:

1. Check that the referenced issuer exists
2. Check the referenced issuer's scope is cluster-wide
3. Check that the component is able to use the issuer
4. Store the issuer reference locally so we don't have to check all bindings when issuing a cert

Then, when the component wants to issue a cert it will:

2. Check if there is a cluster binding available. If so, use that and finish
3. Check if a issuer was passed as a CLI flag at startup. If so, use that and finish
4. Else, emit an error

### Pros and Cons

Pros:

- CRDs are easier to extend. In the future, we can add all kinds of selectors or add namespaced bindings
- A cluster-scoped CRD representing binding neatly represents the idea of a cluster-scoped default

Cons:

- Not obvious what it means to have multiple cluster bindings / namespace bindings in absense of
  selectors
    - This might be a smell indicating that CRDs aren't the correct choice
- Requires more complex logic to implement
    - Code generation
    - controller-runtime
- If we add selectors down the line, would it be confusing to add a targeted namespace selector
  to the cluster binding only to have it overriden by a namespace-scoped binding?

## Background

### Current Behaviour: Rotation

csi-driver-spiffe does not rotate pods when the CA is changed today; as a PoC I created a cluster with csi-driver-spiffe and then ran the following command to upgrade to a different CA:

```console
$ helm upgrade -i -n cert-manager cert-manager-csi-driver-spiffe jetstack/cert-manager-csi-driver-spiffe --wait \
 --set "app.logLevel=1" \
 --set "app.trustDomain=my.trust.domain" \
 --set "app.approver.signerName=clusterissuers.cert-manager.io/second-ca" \
 --set "app.issuer.name=second-ca" \
 --set "app.issuer.kind=ClusterIssuer" \
 --set "app.issuer.group=cert-manager.io"
...
Release "cert-manager-csi-driver-spiffe" has been upgraded. Happy Helming!
...

## IMMEDIATELY AFTER; the pod hasn't been changed

$ kubectl exec -n sandbox my-csi-app-6f58c7589-cwxbv -- cat /var/run/secrets/spiffe.io/tls.crt | openssl x509 --noout --text
Warning: Reading certificate from stdin since no -in or -new option is given
Certificate:
  Data:
    <snip>
    Issuer: CN=csi-driver-spiffe-ca  # <-- would be "second-ca" if the cert had been rotated
    <snip>
```

I then deleted a pod to confirm that the replacement used the new CA:

```console
$ kubectl get pods -n sandbox
... my-csi-app-6f58c7589-thwxh ...
... my-csi-app-6f58c7589-tldb4 ...
... my-csi-app-6f58c7589-5ln9n ...
... my-csi-app-6f58c7589-wlmrx ...
... my-csi-app-6f58c7589-cwxbv ...

$ kubectl delete -n sandbox pod my-csi-app-6f58c7589-cwxbv

$ kubectl get pods -n sandbox
... my-csi-app-6f58c7589-thwxh ...
... my-csi-app-6f58c7589-tldb4 ...
... my-csi-app-6f58c7589-5ln9n ...
... my-csi-app-6f58c7589-wlmrx ...
... my-csi-app-6f58c7589-mxmqh ... # NEW

$ kubectl exec -n sandbox my-csi-app-6f58c7589-mxmqh -- cat /var/run/secrets/spiffe.io/tls.crt | openssl x509 --noout --text
Certificate:
  Data:
    <snip>
    Issuer: CN=second-ca # Rotated!
    <snip>
```
