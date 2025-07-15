# RFC External Artifacts

**Status:** provisional

**Creation date:** 2025-04-08

**Last update:** 2025-07-15

## Summary

This RFC proposes the introduction of a new API called `ExternalArtifact` that would allow
3rd party controllers to act as a source of truth for the cluster desired state. In effect,
the `ExternalArtifact` API acts as an extension of the existing `source.toolkit.fluxcd.io` APIs
that enables Flux `kustomize-controller` and `helm-controller` to consume artifacts from external
source types that are not natively supported by `source-controller`.

## Motivation

Over the years, we've received requests from users to support other source types besides the
ones natively supported by `source-controller`. For example, users have asked for support of
downloading Kubernetes manifests from GitHub/GitLab releases, Omaha protocol, SFTP protocol,
and other remote storage systems.

Another common request is to run transformations on the artifacts fetched by source-controller.
For example, users want to be able to generate YAML manifests from jsonnet, cue, and other
templating engines before they are consumed by Flux `kustomize-controller`.

In order to support these use cases, we need to define a standard API that allows 3rd party
controllers to expose artifacts in-cluster (in the same way `source-controller` does)
that can be consumed by Flux  `kustomize-controller` and `helm-controller`.

### Goals

Define a standard API for 3rd party controllers to expose artifacts that can be consumed by
Flux controllers in the same way as the existing `source.toolkit.fluxcd.io` APIs.

Allow Flux users to transition from using `source-controller` to using 3rd party source controllers
with minimal changes to their existing `Kustomizations` and `HelmReleases`.

### Non-Goals

Allow arbitrary custom resources to be referenced in Flux `Kustomization` and `HelmRelease` as `sourceRef`.

Extend the Flux controllers permissions to access custom resources that are not part of the
`source.toolkit.fluxcd.io` APIs.

## Proposal

Assuming we have a custom controller called `release-controller` that is responsible for
reconciling `GitHubRelease` custom resources. This controller downloads the Kubernetes
deployment YAML manifests from the GitHub API and stores them in a local file system
as a `tar.gz` file. The `release-controller` then creates an `ExternalArtifact`
custom resource that tells the Flux controllers from where to fetch the artifact.

Every time the `release-controller` reconciles a `GitHubRelease` custom resource,
it updates the `ExternalArtifact` status with the latest artifact information if the
upstream release has changed.

The `release-controller` is responsible for exposing a HTTP endpoint that serves
the artifacts from its own storage. The URL of the `tar.gz` artifact is stored in
the `ExternalArtifact` status and should be accessible from the Flux controllers
running in the cluster.

Example of a generated `ExternalArtifact` custom resource:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: ExternalArtifact
metadata:
  name: podinfo
  namespace: apps
spec:
  # SourceRef points to the Kubernetes custom resource for
  # which the artifact is generated.
  # +optional
  sourceRef:
    apiVersion: source.example.com/v1alpha1
    kind: GitHubRelease
    name: podinfo
    namespace: apps
status:
  artifact:
    # Digest is the digest of the tar.gz file in the form of '<algorithm>:<checksum>'.
    # The digest is used by the Flux controllers to verify the integrity of the artifact.
    # +required
    digest: sha256:35d47c9db0eee6ffe08a404dfb416bee31b2b79eabc3f2eb26749163ce487f52
    # LastUpdateTime is the timestamp corresponding to the last update of the
    # Artifact in storage.
    # +required
    lastUpdateTime: "2025-03-21T13:37:31Z"
    # Path is the relative file path of the Artifact. It can be used to locate
    # the file in the root of the Artifact storage on the local file system of
    # the controller managing the Source.
    # +required
    path: release/apps/podinfo/6.8.0-b3396ad.tar.gz
    # Revision is a human-readable identifier traceable in the origin source system
    # in the form of '<human-readable-identifier>@<algorithm>:<checksum>'.
    # The revision is used by the Flux controllers to determine if the artifact has changed.
    # +required
    revision: 6.8.0@sha256:35d47c9db0eee6ffe08a404dfb416bee31b2b79eabc3f2eb26749163ce487f52
    # Size is the number of bytes of the tar.gz file.
    # +required
    size: 20914
    # URL is the in-cluster HTTP address of the Artifact as exposed by the controller
    # managing the Source. It can be used to retrieve the Artifact for
    # consumption, e.g. by kustomize-controller applying the Artifact contents.
    # +required
    url: http://release-controller.flux-system.svc.cluster.local./release/apps/podinfo/6.8.0-b3396ad.tar.gz
  conditions:
  - lastTransitionTime: "2025-04-08T09:09:49Z"
    message: stored artifact for release 6.8.0
    observedGeneration: 1
    reason: Succeeded
    status: "True"
    type: Ready
```

Note that the `.status.artifact` is identical to how `source-controller` exposes the
artifact information for `Bucket`, `GitRepository`, and `OCIRepository` custom resources.
This allows the Flux controllers to consume external artifacts with minimal changes.

The `ExternalArtifact` custom resource is referenced by a Flux `Kustomization` as follows:

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: podinfo
  namespace: apps
spec:
  interval: 10m
  sourceRef:
    kind: ExternalArtifact
    name: podinfo
  path: "./"
  prune: true
```

Flux `kustomize-controller` will then fetch the artifact from the URL specified in the
`ExternalArtifact` status, verifies the integrity of the artifact using the digest
and applies the contents of the artifact to the cluster.

Like with the existing `source.toolkit.fluxcd.io` APIs, `kustomize-controller` will
watch the `ExternalArtifact` custom resource for changes and will re-apply the
contents of the artifact when the `.status.artifact.revision` changes.

When the `ExternalArtifact` contains a Helm chart, it can be referenced by a Flux `HelmRelease` as follows:

```yaml
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: podinfo
  namespace: apps
spec:
  interval: 10m
  releaseName: podinfo
  chartRef:
    kind: ExternalArtifact
    name: podinfo
  values:
    replicaCount: 2
```

### User Stories

#### 3rd Party Source Controller

As a 3rd party controller developer, I want to expose artifacts in-cluster that are sourced from `flatcar/nebraska`
that can be consumed by Flux `kustomize-controller` and `helm-controller` so that Flux users can use my controller
as a source of truth for their cluster desired state.

#### Custom Source Transform

As a Flux user, I want to use a custom controller that generates Kubernetes manifests from CUE templates 
which can be consumed by Flux `kustomize-controller`.

#### Policy Enforcement

As a cluster administrator, I want to ensure that only trusted 3rd party controllers
can create and manage `ExternalArtifact` resources in the cluster.

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
metadata:
  name: "trusted-external-artifacts"
spec:
  failurePolicy: Fail
  matchConstraints:
    resourceRules:
    - apiGroups:   ["source.toolkit.fluxcd.io"]
      apiVersions: ["v1"]
      operations:  ["CREATE", "UPDATE"]
      resources:   ["externalartifacts"]
  validations:
    # Restrict the sourceRef to only allow trusted APIs
    - expression: >
        object.spec.sourceRef.apiVersion.startsWith('source.example.com')
    # Restrict the sourceRef to only allow trusted Kinds
    - expression: >
        object.spec.sourceRef.kind == 'GitHubRelease' || 
        object.spec.sourceRef.kind == 'GitLabRelease'
    # Restrict the artifacts to be served only by trusted endpoints within the cluster
    - expression: >
        !has(object.status.artifact) ||
        object.status.artifact.url.startsWith('http://release-controller.flux-system.svc.cluster.local./')
    # Restrict the artifact operations to trusted service accounts
    - expression: >
        request.userInfo.username == 'system:serviceaccount:flux-system:release-controller'
```

### Alternatives

An alternative to this proposal would be to deploy an OCI registry in-cluster.
The 3rd party controllers would then push the artifacts to the registry
and Flux `kustomize-controller` and `helm-controller` would consume the artifacts
via the `OCIRepository` custom resource.

While this approach is feasible, it requires additional infrastructure and
configuration, which may not be desirable for all users. The `ExternalArtifact` API
provides a simpler and more flexible way to expose artifacts in-cluster without
the need to self-host an OCI registry. In addition, the `ExternalArtifact` API
offers a better user experience by allowing Flux user to trace the origin of reconciled resources
back to the original source via `ExternalArtifact.spec.sourceRef` and `flux trace` command.

## Design Details

The `ExternalArtifact` API will be added to `source-controller/api` Go package.
3rd party controllers will import `github.com/fluxcd/source-controller/api/v1`
to generate valid `ExternalArtifact` custom resources using the controller-runtime client.
To create `tar.gz` files compatible with Flux, the controller will make use of
the `github.com/fluxcd/pkg/oci` package, calling the `client.Build` function.

The `ExternalArtifact` CRD will be bundled with the `source-controller` CRDs manifests which
are part of the standard Flux distribution. This means that users will not need to install
the `ExternalArtifact` CRD separately, as it will be available out of the box with Flux.

The `ExternalArtifact` API specifications will be published to the Flux documentation website,
under the `source.toolkit.fluxcd.io` API reference section.

The Flux `Kustomization` and `HelmRelease` APIs will be extended to support the `ExternalArtifact` kind
as a valid `sourceRef.kind` and `chartRef.kind`. The `kustomize-controller` and `helm-controller`
will gain the ability to consume artifacts from `ExternalArtifact` and watch for revision changes.

The `flux trace` command will be extended to support the `ExternalArtifact` API, allowing Flux users
to trace any Kubernetes resource in-cluster that originates from an `ExternalArtifact` and see the
`sourceRef` information that points to the original source.

The `flux` CLI will implement the `flux get externalartifact` command for listing and status checking
of `ExternalArtifact` custom resources in the cluster.

## Implementation History

<!--
Major milestones in the lifecycle of the RFC such as:
- The first Flux release where an initial version of the RFC was available.
- The version of Flux where the RFC graduated to general availability.
- The version of Flux where the RFC was retired or superseded.
-->
