# Flux release spec

The Flux project repository [fluxcd/flux2](https://github.com/fluxcd/flux2) contains
the Flux command-line tool source code and the Kubernetes manifests for
bundling the [Flux controllers](controllers.md) into a distributable package.

## Release versioning

Flux is released by following the [semver](https://semver.org/) conventions:

- `vX.Y.Z-RC.W` release candidates e.g. `v2.0.0-rc.1`
- `vX.Y.Z` stable releases e.g. `v2.0.0`

The Flux project maintains release branches for the most recent three minor releases
e.g. `release/2.0.x`, `release/2.1.x` and `release/2.2.x`.

### Release candidates

Release candidates are intended for testing new features or improvements before a final release.

In most cases, a maintainer will publish a release candidate for Flux users to test on their
staging clusters. Release candidates are not meant to be deployed in production unless advised
to do so by a maintainer.

Release candidates can be unstable and they are deprecated by subsequent RC or stable versions.

### Patch releases

Patch releases are intended for critical bug fixes to the latest minor version,
such as addressing security vulnerabilities or fixes to severe problems with no workaround.

Patch releases do not contain breaking changes, feature additions or any type of user-facing changes.
If a CVE fix requires a breaking change, then a minor release will provide the fix.

We expect users to be running the latest patch release of a given minor release.

### Minor releases

Minor releases are intended for backwards compatible feature additions and improvements.
Note that breaking changes may occur if required by a security vulnerability fix.

Minor releases are used when updating the Flux controllers or Kubernetes dependencies
from one minor version to another.

In effect, this means a Flux minor version will be released at least every four months after each
Kubernetes minor version release. To properly validate the Flux CLI and controllers against
the latest Kubernetes version, we reserve a time window of at least two weeks for end-to-end testing. 

### Major releases

Major releases are intended for drastic changes to the Flux behaviour or security stance.

A Flux major release will be announced ahead of time throughout all communication channels,
and a support window of one year will be provided for the previous major version.

## Release cadence

Flux is at least released at the same rate as Kubernetes, following their cadence of three
minor releases per year. After each Kubernetes minor release, the CLI and all controllers are
tested against the latest Kubernetes version and are released at approximately two weeks after Kubernetes.
The newly released Flux version offers support for Kubernetes N-2 minor versions.

Flux may have more than three minor releases per year, if maintainers decide to ship a 
new feature or optimisation ahead of schedule.

## Supported releases

For Flux the CLI and its controllers we support the last three minor releases.
Critical bug fixes such as security fixes, may be back-ported to those three minor
versions as patch releases, depending on severity and feasibility.

Note that back-porting is provided by the community on a best-effort basis.

The Flux controllers are guaranteed to be compatible with each other
within one minor version (older or newer) of Flux.

The `flux` command-line tool is supported within one minor version (older or newer) of Flux.

## Supported upgrades

Users can upgrade from any `v2.x` release to any other `v2.x` release (the latest patch version).

After upgrade, [Flux Custom Resources](controllers.md#api-versioning) may require editing,
for which migration instructions are provided as part of the
[changelog](#release-changelog).

We expect users to keep Flux up-to-date on their clusters using automation tools
such as [Flux GitHub Actions](../../action) and
[Renovatebot](https://docs.renovatebot.com/modules/manager/flux/).

Various vendors such as Microsoft Azure, D2iQ, Weaveworks and others offer a managed Flux service,
and it's their responsibility to keep Flux up-to-date and free of CVEs.
The Flux team communicates security issues to vendors as described in the
[Coordinated Vulnerability Disclosure document](https://github.com/fluxcd/.github/blob/14b735cdb23ec80d528ff4f71e562405a2f00639/CVD_LIST.md).

## Kubernetes supported versions

The Flux CLI and controllers offer support for all Kubernetes versions supported upstream.

Every Flux release undergoes a series of conformance and end-to-end tests for 
the latest Kubernetes minor release. The test suite is run against
[Kubernetes Kind](https://kind.sigs.k8s.io/) for both AMD64 and ARM64 distributions.

We expect users to keep Kubernetes up-to-date with the latest patch version of a
supported minor release. Once a Kubernetes version reaches [end-of-life](https://endoflife.date/kubernetes),
we can't guarantee the next Flux release will work with it,
as we don't run end-to-end testing for EOL Kubernetes versions.

## Release artifacts

Each Flux release produces the following artifacts:

- Source code (GitHub Releases page)
- Software Bill of Materials in SPDX format (GitHub Releases page)
- SLSA provenance attestations (GitHub Releases page)
- Kubernetes manifests of all controllers (GitHub Releases page)
- CLI binaries for Linux, macOS and Windows (GitHub Releases page)
- Signed checksums of source code, SBOM and manifests (GitHub Releases page)
- Multi-arch container images of the CLI (GitHub Container Registry and DockerHub)
- OCI artifacts with the Kubernetes manifests (GitHub Container Registry and DockerHub)
- CLI [Homebrew](https://brew.sh/) formulas for Linux and macOS

All the artifacts are cryptographically signed and can be verified with Cosign.

The release artifacts can be accessed based on the Flux version.

## Release changelog

All released versions of Flux are published on [GitHub Releases page](https://github.com/fluxcd/flux2/releases)
along with a list of changes from the previous release.

The changelog contains the following information:

- Security vulnerabilities fixes (if any)
- Breaking changes and migration instructions (if any)
- A summary of new features and improvements for the Flux APIs and controllers
- Links to the changelog of each controller version included in a Flux release
- A list of new features, improvements and bug fixes for the Flux CLI
- A list of documentation additions

**Note** that the vulnerability disclosure procedure is explained on the [security page](https://fluxcd.io/security/).

