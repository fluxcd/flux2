# Flux release procedures

This document provides an overview of the release procedures for each component
type in the Flux project. It is intended for project maintainers, but may also
be useful for contributors who want to understand the release process.

If you have any questions, please reach out to another maintainer for
clarification.

## Table of contents

- [General rules](#general-rules)
  + [Signing releases](#signing-releases)
- [Component types](#component-types)
  + [Shared packages](#shared-packages)
  + [Controllers](#controllers)
    * [Minor releases](#controller-minor-releases)
    * [Patch releases](#controller-patch-releases)
    * [Release candidates](#controller-release-candidates)
    * [Preview releases](#controller-preview-releases)
  + [Distribution](#distribution)
    * [Minor releases](#distribution-minor-releases)
    * [Patch releases](#distribution-patch-releases)
    * [Release candidates](#distribution-release-candidates)
- [Backport changes for patch releases](#backport-changes-for-patch-releases)

## General rules

### Signing releases

To ensure the integrity and authenticity of releases, all releases must be
signed with a GPG key. The public key must be uploaded to the GitHub account of
the maintainer, and the private key must be kept secure.

In addition, we recommend the following practices:

1. Safeguard your GPG private key, preferably using a hardware security key
   like a YubiKey.
2. Use a subkey dedicated to signing releases, set an expiration date for
   added security, and keep the master key offline. Refer to
   [this guide](https://riseup.net/en/security/message-security/openpgp/best-practices#key-configuration)
   for detailed instructions.

We understand that this may seem overwhelming. If you are not comfortable with
the process, don't hesitate to seek assistance from another maintainer who has
experience with signing releases.

Please note that SSH signatures are not supported at this time due to limited
availability of SSH signature verification outside the `git` CLI.

## Component types

### Shared packages

To release a [package](packages.md) as a project maintainer, follow these steps:

1. Tag the `main` branch using SemVer.
2. Sign the tag according to the [general rules](#general-rules).
3. Push the signed tag to the upstream repository.

Use the following commands as an example:

```shell
git clone https://github.com/fluxcd/pkg.git
git switch main
git tag -s -m "<module>/<semver>" "<module>/<semver>"
git push origin "<module>/<semver>"
```

In the commands above, `<module>` represents the relative path to the directory
containing the `go.mod` file, and `<semver>` refers to the SemVer version of the
release. For instance, `runtime/v1.0.0` for [`fluxcd/pkg/runtime`](https://github.com/fluxcd/pkg/tree/main/runtime),
or `http/fetch/v0.1.0` for [`fluxcd/pkg/http/fetch`](https://github.com/fluxcd/pkg/tree/main/http/fetch).

Before cutting a release candidate, ensure that the package's tests pass
successfully.

Here's an example of releasing a candidate from a feature branch:

```shell
git switch <feature-branch>
git tag -s -m "<module>/<semver>-<feature>.1" "<module>/<semver>-<feature>.1"
git push origin "<module>/<semver>-<feature>.1"
```

### Controllers

To release a [controller](controllers.md) as a project maintainer, follow the
steps below. Note that the release procedure differs depending on the type of
release.

##### Controller minor releases

1. Checkout the `main` branch and pull changes from the remote repository.

2. Create a "release series" branch for the next minor SemVer range, e.g.,
   `release/v1.2.x`, and push it to the remote repository.

   ```shell
   git switch -c release/v1.2.x main
   ```

3. From the release series branch, create a release preparation branch for the
   specific release.

   ```shell
   git switch -c release-v1.2.0 release/v1.2.x
   ```

4. Add an entry to `CHANGELOG.md` for the new release and commit the changes.

   ```shell
   vim CHANGELOG.md
   git add CHANGELOG.md
   git commit -s -m "Add changelog entry for v1.2.0"
   ```

5. Update `github.com/fluxcd/<name>-controller/api` version in `go.mod` and
   `newTag` value in `config/manager/kustomization.yaml` to the target SemVer
   (e.g., `v1.2.0`). Commit and push the changes.

   ```shell
   vim go.mod
   vim config/manager/kustomization.yaml
   git add go.mod config/manager/kustomization.yaml
   git commit -s -m "Release v1.2.0"
   git push origin release-v1.2.0
   ```

6. Create a pull request for the release branch and merge it into the release
   series branch (e.g., `release/v1.2.x`).

7. Create `api/<next semver>` and `<next semver>` tags for the merge commit in
   `release/v1.2.x`. Ensure the tags are signed according to the [general
   rules](#general-rules)., and push them to the remote repository.

   ```shell
   git switch release/v1.2.x
   git pull origin release/v1.2.x
   git tag -s -m "api/v1.2.0" api/v1.2.0
   git push origin api/v1.2.0
   git tag -s -m "v1.2.0" v1.2.0
   git push origin v1.2.0
   ```

8. Confirm that the CI builds and releases the newly tagged version.

9. Create a pull request for the release series branch and merge it into `main`.

##### Controller patch releases

1. Ensure everything to be included in the release is backported to the
   "release series" branch (e.g., `release/v1.2.x`). If not, refer to the
   [backporting](#backport-changes-for-patch-releases) section.

2. From the release series branch, create a release preparation branch for the
   specific patch release.

   ```shell
   git pull origin release/v1.2.x
   git switch -c release-v1.2.1 release/v1.2.1
   ```

3. Add an entry to `CHANGELOG.md` for the new release and commit the changes.

   ```shell
   vim CHANGELOG.md
   git add CHANGELOG.md
   git commit -s -m "Add changelog entry for v1.2.1"
   ```

4. Update `github.com/fluxcd/<name>-controller/api` version in `go.mod` and
   `newTag` value in `config/manager/kustomization.yaml` to the target SemVer
   (e.g., `v1.2.1`). Commit and push the changes.

   ```shell
   vim go.mod
   vim config/manager/kustomization.yaml
   git add go.mod config/manager/kustomization.yaml
   git commit -s -m "Release v1.2.1"
   git push origin release-v1.2.1
   ```

5. Create a pull request for the release branch and merge it into the release
   series branch (e.g., `release/v1.2.x`).

6. Create `api/<next semver>` and `<next semver>` tags for the merge commit in
   `release/v1.2.x`. Ensure the tags are signed according to the [general
   rules](#general-rules)., and push them to the remote repository.

   ```shell
   git switch release/v1.2.x
   git pull origin release/v1.2.x
   git tag -s -m "api/v1.2.1" api/v1.2.1
   git push origin api/v1.2.1
   git tag -s -m "v1.2.1" v1.2.1
   git push origin v1.2.1
   ```

7. Confirm that the CI builds and releases the newly tagged version.

8. Cherry-pick the changelog entry from the release series branch and create a
   pull request to merge it into `main`.

   ```shell
   git pull origin main
   git switch -c pick-changelog-v1.2.1 main
   git cherry-pick -x <commit hash>
   git push origin pick-changelog-v1.2.1
   ```
   
#### Controller release candidates

In some cases, it may be necessary to release a [release
candidate](controllers.md#release-candidates) of a controller.

To create a first release candidate, follow the steps to create a [minor
release](#controller-minor-releases), but use the `rc.X` suffix for SemVer
version to release (e.g., `v1.2.0-rc.1`).

Once the release series branch is created, subsequent release candidates and
the final (non-RC) release should follow the procedure for [patch controller
releases](#controller-patch-releases).

#### Controller preview releases

In some cases, it may be necessary to release a preview of a controller.
A preview release is a release that is not intended for production use, but
rather to allow users to quickly test new features or an intended bug fix, and
provide feedback.

Preview releases are made by triggering the release GitHub Actions workflow on
a specific Git branch. This workflow will build the container image, sign it
using Cosign, and push it to the registry. But does not require a Git tag,
GitHub release, or a changelog entry.

To create a preview release, follow the steps below.

1. Navigate to `https://github.com/fluxcd/<name>-controller/actions/workflows/release.yml`.
2. Click the `Run workflow` button.
3. Select the branch to release from the `Branch` dropdown.
4. The default values for the `image tag` (`preview`) should be correct, but can
   be changed if necessary.
5. Click the green `Run workflow` button.
6. The workflow will now run, and the container image will be pushed to the
   registry. Once the workflow has completed, the image reference will be
   available in the logs, and can be shared in the relevant issue or pull
   request.

### Distribution

To release a [Flux distribution](flux.md) as a project maintainer, follow the
steps below.

Note that the Flux distribution contains multiple components, and you may need
to release [controllers](#controllers) before releasing the distribution.
Automation is in place to automatically create a pull request to update the
version in the `main` branch when a new controller version is released.

#### Distribution minor releases

1. Ensure everything to be included in the release is merged into the `main`
   branch, including any controller releases you wish to include in the
   release.

2. Create a "release series" branch for the next minor SemVer range, e.g.,
   `release/v1.2.x`, and push it to the remote repository.

   ```shell
   git switch -c release/v1.2.x main
   ```
   
3. Prepare the required release notes for this release, see [release
   notes](#distribution-release-notes) for more information.

4. Tag the release series branch with the SemVer version of the release, e.g.,
   `v1.2.0`. Ensure the tag is signed according to the [general
   rules](#general-rules), and push it to the remote repository.

   ```shell
   git tag -s -m "v1.2.0" v1.2.0
   git push origin v1.2.0
   ```

5. Confirm that the CI builds and releases the newly tagged version.

6. Once the release is complete, update the release notes on GitHub with the
   release notes prepared in step 3.

7. Post a message in the [`#flux` CNCF Slack channel](https://cloud-native.slack.com/archives/CLAJ40HV3)
   announcing the release, and pin it.

#### Distribution patch releases

1. Ensure everything to be included in the release is backported to the
   "release series" branch (e.g., `release/v1.2.x`). If not, refer to the
   [backporting](#backport-changes-for-patch-releases) section.

2. Prepare the required release notes for this release, see [release
   notes](#distribution-release-notes) for more information.

3. Tag the release series branch with the SemVer version of the release, e.g.,
   `v1.2.1`. Ensure the tag is signed according to the [general
   rules](#general-rules), and push it to the remote repository.

   ```shell
   git tag -s -m "v1.2.1" v1.2.1
   git push origin v1.2.1
   ```
   
4. Confirm that the CI builds and releases the newly tagged version.

5. Once the release is complete, update the release notes on GitHub with the
   release notes prepared in step 2.

6. Post a message in the [`#flux` CNCF Slack channel](https://cloud-native.slack.com/archives/CLAJ40HV3)

#### Distribution release candidates

In some cases, it may be necessary to release a [release candidate](flux.md#release-candidates)
of the distribution.

To create a first release candidate, follow the steps to create a [minor
release](#distribution-minor-releases), but use the `rc.X` suffix for SemVer
version to release (e.g., `v1.2.0-rc.1`).

Once the release series branch is created, subsequent release candidates and
the final (non-RC) release should follow the procedure for [patch controller
releases](#controller-patch-releases).

#### Distribution release notes

The release notes template for Flux distributions is available in the
[release-notes-template.md](release-notes-template.md) file.

## Backport changes for patch releases

The Flux projects have a backport bot that automates the process of backporting
changes from `main` to the release series branches. The bot is configured to
backport pull requests that are labeled with `backport:<release series>` (e.g.,
`backport:release/v2.1.x`) and have been merged into `main`.

The label(s) are preferably added to the pull request before it is merged into
`main`. If the pull request is merged into `main` without labeling, they can
still be added to the pull request after it has been merged.

The backport bot will automatically backport the pull request to the release
series branch and create a pull request for the backport. It will comment on
the pull request with a link to the backport pull request.

If the backport bot is unable to backport a pull request automatically (for
example, due to conflicts), it will comment on the pull request with
instructions on how to backport the pull request manually.

### Manual backporting

In some cases, the backport bot may not be suitable for backporting a pull
request. For example, if the pull request contains a series of changes of which
a subset should be backported. In these cases, the pull request should be
backported manually.

To backport a pull request manually, create a new branch from the release
series branch (e.g., `release/v2.1.x`) and cherry-pick the commits from the
pull request into the new branch. Push the new branch to the remote repository
and create a pull request to merge it into the release series branch.

```shell
git pull origin release/v2.1.x
git switch -c backport-<pull request number>-to-v2.1.x release/v2.1.x
git cherry-pick -x <commit hash>
# Repeat the cherry-pick command for each commit to backport
git push origin backport-<pull request number>-to-v2.1.x
```
