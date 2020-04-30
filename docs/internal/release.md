# Release

To release a new version the following steps should be followed:

1. Create a new branch from `master` i.e. `release-<next semver>`. This
   will function as your release preparation branch.
1. Change the `VERSION` value in `cmd/tk/main.go` to that of the
   semver release you are going to make. Commit and push your changes.
1. Create a PR for your release branch and get it merged into `master`.
1. Create a `<next semver>` tag for the merge commit in `master` and
   push it to remote.
1. Confirm CI builds and releases the newly tagged version.
