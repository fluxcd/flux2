# Flux controllers release spec

The Flux controllers are 
[Kubernetes operators](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/),
each controller has its own Git repository and release cycle.

Controller repositories and their interdependencies:

1. [fluxcd/source-controller](https://github.com/fluxcd/source-controller)
2. [fluxcd/kustomize-controller](https://github.com/fluxcd/kustomize-controller) (imports `fluxcd/source-controller/api`)
3. [fluxcd/helm-controller](https://github.com/fluxcd/helm-controller) (imports `fluxcd/source-controller/api`)
4. [fluxcd/notification-controller](https://github.com/fluxcd/notification-controller)
5. [fluxcd/image-reflector-controller](https://github.com/fluxcd/image-reflector-controller)
6. [fluxcd/image-automation-controller](https://github.com/fluxcd/image-automation-controller) (imports `fluxcd/source-controller/api` and `fluxcd/image-reflector-controller/api`)

## API versioning 

The Flux APIs (Kubernetes CRDs) follow the
[Kubernetes API versioning](https://kubernetes.io/docs/reference/using-api/#api-versioning) scheme.

### Alpha version

An alpha version API e.g. `v1alpha1` is considered experiment and should be used on
test environments only.

The schema of objects may change in incompatible ways in a later controller release.
The custom resources may require editing and re-creating after a CRD update.

An alpha version API becomes deprecated once a subsequent alpha or beta API version is released.
A deprecated alpha version is subject to removal after a three months period.

An alpha API is introduced when its proposal reaches the  `implementable` phase in the
[Flux RFC process](https://github.com/fluxcd/flux2/tree/main/rfcs).
We encourage users to try out the alpha APIs and provide feedback which is extremely
valuable during early stages of development.

### Beta version

A beta version API e.g. `v2beta1` is considered well tested and safe to be used.

The schema of objects may change in incompatible ways in a subsequent beta or stable API version.
The custom resources may require editing after a CRD update for which migration instructions will be
provided as part of the controller changelog.

A beta version API becomes deprecated once a subsequent beta or stable API version is released. 
A deprecated beta version is subject to removal after a six months period.

### Stable version

A stable version API e.g. `v2` is considered feature complete.

Any changes to the object schema do not require editing or re-creating of custom resources.
Schema fields can't be removed, only new fields can be added with a default value that
doesn't affect the controller's current behaviour.

A stable version API becomes deprecated once a subsequent stable version is released.
Stable API versions are not subject to removal in any future release of a controller major version.

In effect, this means that for as long as Flux `v2` is being maintained, all the stable API versions 
will be supported.

## Controller versioning

The Flux controllers and their Go API packages are released by following the
[Go module version numbering](https://go.dev/doc/modules/version-numbers) conventions:

- `vX.Y.Z-RC.W` release candidates e.g. `v1.0.0-RC.1`
- `vX.Y.Z` stable releases e.g. `v1.0.0`

The API versioning and controller versioning are indirectly related. For example,
a source-controller minor release `v1.1.0` can introduce a new API version
`v1beta1` for a Kind `XRepository` in the `source.toolkit.fluxcd.io` group.

### Release candidates

Release candidates are intended for testing new features or improvements before a final release.

In most cases, a maintainer will publish a release candidate of a controller for Flux users
to tests it on their staging clusters. Release candidates are not meant to be deployed in production
unless advised to do so by a maintainer.

### Patch releases

Patch releases are intended for critical bug fixes to the latest minor version, such as addressing security
vulnerabilities or fixes to severe problems with no workaround.

Patch releases do not contain breaking changes, feature additions or any type of user-facing changes.
If a CVE fix requires a breaking change, then a minor release will provide the fix.

We expect users to be running the latest patch release of a given minor release as soon as the
controller release is included in a Flux patch release.

### Minor releases

Minor releases are intended for backwards compatible feature additions and improvements.
Note that breaking changes may occur if required by a security vulnerability fix.

Minor releases are used when updating Kubernetes dependencies such as `k8s.io/api` from one minor version to another.

In effect, this means a minor version will be released for all Flux controllers approximately every four months
after each Kubernetes minor version release. To properly validate the controllers against the latest Kubernetes version,
we reserve a time window of at least two weeks for Flux controllers end-to-end testing. 

### Major releases

Major releases are intended for drastic changes in the controller behaviour or security stance.

A controller major release will be announced ahead of time throughout all communication channels,
and a support window of one year will be provided for the previous major version.

## Release cadence

Flux controllers follow Kubernetes three releases per year cadence. After each Kubernetes minor release,
all controllers are tested against the latest Kubernetes version and are released at approximately two
weeks after Kubernetes. The newly released controllers offer support for Kubernetes N-2 minor versions.

A Flux controller may have more than three minor releases per year, if maintainers decide to ship a 
new feature or optimisation ahead of schedule.

## Supported releases

For Flux controllers we support the last three minor releases.

Security fixes, may be backported to those three minor versions as patch releases,
depending on severity and feasibility.

## Release artifacts

Each controller release produces the following artifacts:

- Source code (GitHub Releases page)
- Software Bill of Materials in SPDX format (GitHub Releases page)
- Kubernetes manifests such as CRDs and Deployments (GitHub Releases page)
- Signed checksums of source code, SBOM and manifests (GitHub Releases page)
- Multi-arch container images (GitHub Container Registry and DockerHub)

All the artifacts are cryptographically signed and can be verified with Cosign.

The release artifacts can be accessed based on the controller name and version.

To import or update a controller's API package in a Go project:

```shell
go get github.com/fluxcd/<controller-name>/api@<version>
```

To verify and pull a controller's container image:

```shell
cosign verify ghcr.io/fluxcd/<controller-name>:<version>
docker pull ghcr.io/fluxcd/<controller-name>:<version>
```

To download a controller's Kubernetes Custom resource definitions:

```shell
curl -sL https://github.com/fluxcd/<controller-name>/releases/download/<version>/<controller-name>.crds.yaml
```

## Controller release procedure

As a project maintainer, to release a controller and its API:

1. Checkout the `main` branch and pull changes from remote.
2. Create a `api/<next semver>` tag and push it to remote.
3. Create a new branch from `main` i.e. `release-<next semver>`. This
   will function as your release preparation branch.
4. Update the `github.com/fluxcd/<NAME>-controller/api` version in `go.mod`
5. Add an entry to the `CHANGELOG.md` for the new release and change the
   `newTag` value in ` config/manager/kustomization.yaml` to that of the
   semver release you are going to make. Commit and push your changes.
6. Create a PR for your release branch and get it merged into `main`.
7. Create a `<next semver>` tag for the merge commit in `main` and
   push it to remote.
8. Confirm CI builds and releases the newly tagged version.

**Note** that the Git tags must be cryptographically signed with your private key
and your public key must be uploaded to GitHub.
