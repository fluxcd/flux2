# Flux controllers release spec

The Flux controllers are 
[Kubernetes operators](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/),
each controller has its own Git repository and release cycle.

Controller repositories and their interdependencies:

1. [fluxcd/source-controller](https://github.com/fluxcd/source-controller)
    - dependencies: `github.com/fluxcd/pkg/runtime`
2. [fluxcd/kustomize-controller](https://github.com/fluxcd/kustomize-controller)
    - dependencies: `github.com/fluxcd/source-controller/api`, `github.com/fluxcd/pkg/runtime`
3. [fluxcd/helm-controller](https://github.com/fluxcd/helm-controller)
    - dependencies: `github.com/fluxcd/source-controller/api`, `github.com/fluxcd/pkg/runtime`
4. [fluxcd/notification-controller](https://github.com/fluxcd/notification-controller)
   - dependencies: `github.com/fluxcd/pkg/runtime`
5. [fluxcd/image-reflector-controller](https://github.com/fluxcd/image-reflector-controller)
    - dependencies: `github.com/fluxcd/pkg/runtime`
6. [fluxcd/image-automation-controller](https://github.com/fluxcd/image-automation-controller)
    - dependencies: `github.com/fluxcd/source-controller/api`, `github.com/fluxcd/image-reflector-controller/api`, `github.com/fluxcd/pkg/runtime`

## Release versioning

The Flux controllers and their API packages are released by following the
[Go module version numbering](https://go.dev/doc/modules/version-numbers) conventions:

- `vX.Y.Z-RC.W` release candidates e.g. `v1.0.0-RC.1`
- `vX.Y.Z` stable releases e.g. `v1.0.0`

To import or update a controller API package in a Go project:

```shell
go get github.com/fluxcd/source-controller/api@v1.0.0
```

To pull a controller container image:

```shell
docker pull ghcr.io/fluxcd/source-controller:v1.0.0
```

A Flux controller's Kubernetes Custom Resource Definitions follow the
[Kubernetes API versioning](https://kubernetes.io/docs/reference/using-api/#api-versioning) scheme:

- `v1apha1` experimental API
- `v1beta1` release candidate (supported while v1 is not released)
- `v1` stable release (supported)

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

In effect, this means a minor version will be released for all Flux controllers approximately every three months
after each Kubernetes minor version release.

To properly validate the controllers against the latest Kubernetes version, we reserve a time window of at least
two weeks for Flux controllers end-to-end testing. 

### Major releases

Major releases are intended for drastic changes in the controller behaviour or security stance.

A controller major release will be announced ahead of time throughout all communication channels,
and a support window of one year will be provided for the previous major version.

## Release Cadence

Flux controllers follow Kubernetes three releases per year cadence. After each Kubernetes minor release,
all controllers are tested against the latest Kubernetes version and are released at approximately two
weeks after Kubernetes. The newly released controllers offer support for Kubernetes N-2 minor versions.

A Flux controller may have more than three minor releases per year, if maintainers decide to ship a 
new feature or optimisation ahead of schedule.

## Supported releases

For Flux controllers we support the last three minor releases.

Security fixes, may be backported to those three minor versions as patch releases,
depending on severity and feasibility.

## Release procedure

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
