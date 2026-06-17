# Releases

## Schedule

The release schedule for this project is ad-hoc. Given the pre-1.0 status of the project we do not have a fixed release cadence. However if a vulnerability is discovered we will respond in accordance with our [security policy](https://github.com/cert-manager/community/blob/main/SECURITY.md) and this response may include a release.

## Process

There is a semi-automated release process for this project. When you create a Git tag with a tagname that has a `v` prefix and push it to GitHub it will trigger the [release workflow].

### Preparing for a Release

**BEFORE** doing a release, check if the other images in the csi-driver-spiffe Helm
chart need to be updated. These images are copied to `quay.io/jetstack` as part of our
release process.

These are:

- `registry.k8s.io/sig-storage/livenessprobe` copied to `quay.io/jetstack/livenessprobe`
    - find the latest version using crane:  
    `crane ls --omit-digest-tags registry.k8s.io/sig-storage/livenessprobe | sort -V | tail -1`
    - update `livenessprobe_image_tag` in `make/00_mod.mk`
- `registry.k8s.io/sig-storage/csi-node-driver-registrar` copied to `quay.io/jetstack/csi-node-driver-registrar`
    - find the latest version using crane:  
    `crane ls --omit-digest-tags registry.k8s.io/sig-storage/csi-node-driver-registrar | sort -V | tail -1`
    - update `nodedriverregistrar_image_tag` in `make/00_mod.mk`

### Pre-release Checks

1. **Check for known vulnerabilities** using `govulncheck` and `trivy`:
    ```sh
    # Check the govulncheck GitHub Action is green on main:
    # https://github.com/cert-manager/csi-driver-spiffe/actions/workflows/govulncheck.yaml

    # Run trivy locally (ArtifactHub uses trivy and flags MEDIUM+ vulnerabilities):
    make _bin/tools/trivy
    _bin/tools/trivy fs --scanners vuln .
    ```
    If trivy reports vulnerabilities, bump the affected dependencies before tagging,
    even if they are indirect. ArtifactHub displays trivy results on the
    [security report page](https://artifacthub.io/packages/helm/cert-manager/cert-manager-csi-driver-spiffe?modal=security-report),
    and users rely on a clean report.

### Doing a Release

The release process for this repo is documented below:

1. Create a tag for the new release:
    ```sh
   export VERSION=v0.5.0-alpha.0
   git tag --annotate --message="Release ${VERSION}" "${VERSION}"
   git push origin "${VERSION}"
   ```

2. A GitHub action will see the new tag and do the following:
    - Build and publish any container images
    - Build and publish the Helm chart
    - Create a draft GitHub release

3. Wait for the PR to be merged and wait for OCI Helm chart to propagate and become available from https://charts.jetstack.io (this might take a few hours).

4. Visit the [releases page], edit the draft release, click "Generate release notes", then edit the notes to add the following to the top
    ```
    csi-driver-spiffe is a clean and simple way to get SPIFFE IDs for your Kubernetes pods with minimal dependencies and minimal fuss.
    ```

5. Publish the release.

### Post-release: Verify the Helm Chart Reaches ArtifactHub

The release workflow pushes the Helm chart to `quay.io/jetstack`, and a CI job in
the private [jetstack/jetstack-charts](https://github.com/jetstack/jetstack-charts)
repository (accessible only to Palo Alto Networks cert-manager team members) opens
a PR to sync it to `charts.jetstack.io`. That PR requires a maintainer approval
before it is merged. Until it is merged, ArtifactHub will continue to show the
previous version.

```sh
gh pr list --repo jetstack/jetstack-charts --search "oci-sync" --state open
```

Review and merge the sync PR, then verify the new version appears on ArtifactHub:
https://artifacthub.io/packages/helm/cert-manager/cert-manager-csi-driver-spiffe

### Post-release: Check the ArtifactHub Security Report

Once the new version appears on ArtifactHub, check its security report for
vulnerabilities. ArtifactHub runs trivy against every container image referenced
in the Helm chart — not just the csi-driver-spiffe images we build, but also the
sidecar images (livenessprobe, csi-node-driver-registrar). Vulnerabilities in
sidecar images are outside our control but are still displayed on our package page.

The security report is visible in the web UI at:

https://artifacthub.io/packages/helm/cert-manager/cert-manager-csi-driver-spiffe/VERSION?modal=security-report

It can also be fetched programmatically using the [ArtifactHub API]. The package
ID for csi-driver-spiffe is `ab494d3e-f81c-44b3-9897-50bec3c88c78`:

```sh
export VERSION=0.14.0
make _bin/tools/yq
curl -sL "https://artifacthub.io/api/v1/packages/ab494d3e-f81c-44b3-9897-50bec3c88c78/${VERSION}/security-report" \
  | _bin/tools/yq -p json -o tsv '
    ["IMAGE", "SEVERITY", "CVE", "PACKAGE", "INSTALLED", "FIXED"],
    (
      to_entries[] |
      .key as $image |
      .value.Results[]? |
      .Vulnerabilities[]? |
      [$image, .Severity, .VulnerabilityID, .PkgName, .InstalledVersion, .FixedVersion // "n/a"]
    ) | @tsv
  ' | column -t -s$'\t'
```

If the report shows vulnerabilities in our own images, consider a follow-up patch
release. Vulnerabilities in sidecar images are fixed upstream in the
[kubernetes-csi/livenessprobe](https://github.com/kubernetes-csi/livenessprobe) and
[kubernetes-csi/node-driver-registrar](https://github.com/kubernetes-csi/node-driver-registrar)
repositories — check their open PRs and recent releases to see whether a fix has
been merged but not yet released, and update the sidecar image tags in
`make/00_mod.mk` once new versions are available.

## Artifacts

This repo will produce the following artifacts each release. For documentation on how those artifacts are produced see the "Process" section.

- *Container Images* - Container images for the are published to `quay.io/jetstack`.
- *Helm chart* - An official Helm chart is maintained within this repo and published to `quay.io/jetstack` and `charts.jetstack.io` on each release.

[ArtifactHub API]: https://artifacthub.io/docs/api/
[release workflow]: https://github.com/cert-manager/csi-driver-spiffe/actions/workflows/release.yaml
[releases page]: https://github.com/cert-manager/csi-driver-spiffe/releases
