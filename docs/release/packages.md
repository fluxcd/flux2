# Flux shared packages release spec

The Go packages in [github.com/fluxcd/pkg](https://github.com/fluxcd/pkg) are dedicated Go modules,
each module has its own set of dependencies and release cycle.

These packages are primary meant for internal use in Flux controllers and
for projects which integrate and/or extend Flux.

## Release versioning

The Flux packages are released by following the
[Go module version numbering](https://go.dev/doc/modules/version-numbers) conventions:

- `NAME/vX.Y.Z-RC.W` release candidates e.g. `runtime/v1.0.0-RC.1`
- `NAME/vX.Y.Z` stable releases e.g. `runtime/v1.0.0`

To import or update a Flux package in a Go project:

```shell
go get github.com/fluxcd/pkg/runtime@v1.0.0
```

### Release candidates

Release candidates are intended for testing new features or improvements.

In most cases, a maintainer will cut a release candidate of a package to include it
in a Flux controller release candidate.

Release candidates are not meant to be included in Flux stable releases.
Before cutting a stable release of a controller, all imported Flux packages must be pinned to a stable version.

### Patch releases

Patch releases are intended for critical bug fixes to the latest minor version, such as addressing security
vulnerabilities or fixes to severe problems with no workaround.

Patch releases should not contain breaking changes, feature additions or any type of improvements.

Patch releases should be used when updating dependencies such as `k8s.io/api` from one patch version to another.

### Minor releases

Minor releases are intended for backwards compatible feature additions and improvements.

Minor releases should be used when updating dependencies such as `k8s.io/api` from one minor version to another.
If a [Kubernetes minor version](https://github.com/kubernetes/sig-release/blob/master/release-engineering/versioning.md)
upgrade requires a breaking change (e.g. removal of an API such as `PodSecurityPolicy`) in a Flux package public API,
then a major version release is necessary.

### Major releases

Major releases are intended for backwards incompatible feature additions and improvements.

Any change to a package public API, such as a change to a Go function signature, requires a new major release.

## Supported releases

For Flux Go packages we only support the latest stable release. We expect for projects that depend on
Flux packages to stay up-to-date by automating the Go modules updates with tools like Dependabot.

In effect, this means we'll not backport CVE fixes to an older minor or major version of a package.

## Deprecation policy

A Flux Go package can be deprecated at any time. Usually a deprecated package may be replaced a 
different one, but there are no guarantees to always have a suitable replacement.

A deprecated package is marked as so in its `go.mod` e.g.

```go
// Deprecated: use github.com/fluxcd/pkg/tar instead.
module github.com/fluxcd/pkg/untar
```

## Release procedure

As a project maintainer, to release a package, tag the `main` branch using semver,
and push the signed tag to upstream:

```shell
git clone https://github.com/fluxcd/pkg.git
git switch main
git tag -s -m "runtime/v1.0.0" "runtime/v1.0.0"
git push origin "runtime/v1.0.0"
```

**Note** that the Git tags must be cryptographically signed with your private key
and your public key must be uploaded to GitHub.

Release candidates of a specific package can be cut from the `main` branch or from an `dev-<pkg-name>` branch:

```shell
git switch dev-runtime
git tag -s -m "runtime/v1.1.0-RC.1" "runtime/v1.1.0-RC.1"
git push origin "runtime/v1.1.0-RC.1"
```

Before cutting a release candidate, make sure the tests are passing on the `dev` branch.
