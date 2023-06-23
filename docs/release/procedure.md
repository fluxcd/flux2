# Flux release procedure

## Shared packages release procedure

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

## Controllers release procedure

As a project maintainer, to release a controller and its API:

1. Checkout the `main` branch and pull changes from remote.
2. Create a new branch from `main` i.e. `release-<next semver>`. This
   will function as your release preparation branch.
3. Update the `github.com/fluxcd/<NAME>-controller/api` version in `go.mod`
4. Add an entry to the `CHANGELOG.md` for the new release and change the
   `newTag` value in ` config/manager/kustomization.yaml` to that of the
   semver release you are going to make. Commit and push your changes.
5. Create a PR for your release branch and get it merged into `main`.
6. Create a `api/<next semver>` tag for the merge commit in `main` and push it to remote.
7. Create a `<next semver>` tag for the merge commit in `main` and push it to remote.
8. Confirm CI builds and releases the newly tagged version.

**Note** that the Git tags must be cryptographically signed with your private key
and your public key must be uploaded to GitHub.

## Distribution release procedure

As a project maintainer, to release a new Flux version and its CLI:

- `v2.X.Y-RC.Z` (Branch: `release-2.X`)
    - When the `main` branch is feature-complete for `v2.X`, we will cherrypick PRs essential to `v2.X` to the `release-2.X` branch.
    - We will cut the first [release candidate](#release-candidates) by tagging the `release-2.X` as `v2.X.0-RC.0`.
    - If we're not satisfied with `v2.X.0-RC.0`, we'll keep releasing RCs until all issues are solved.
- `v2.X.0` (Branch: `release-2.X`)
    - The final release is cut from the `release-2.X` branch and tagged as `v2.X.0`.
- `v2.X.Y, Y > 0` (Branch: `release-2.X`)
    - [Patch releases](#patch-releases) are released as we cherrypick commits from `main` into the `release-2.X` branch.
    - Flux controller updates (patch versions of a controller minor release included in `v2.X.0`) PRs are merged directly into the `release-2.X` branch.
    - A patch release is cut from the `release-2.X` branch and tagged as `v2.X.Y`.
