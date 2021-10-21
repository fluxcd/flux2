# Flux release procedure

The Flux Go modules and the GitOps Toolkit controllers are released by following the [semver](https://semver.org) conventions.

Repositories subject to semver releases:

1. [fluxcd/pkg](https://github.com/fluxcd/pkg)
    - modules: `apis/meta`, `runtime`, various utilities
    - dependencies: `k8s.io/*`, `sigs.k8s.io/controller-runtime`
1. [fluxcd/source-controller](https://github.com/fluxcd/source-controller)
    - modules: `api`
    - dependencies: `github.com/fluxcd/pkg/*`
1. [fluxcd/kustomize-controller](https://github.com/fluxcd/kustomize-controller)
    - modules: `api`
    - dependencies: `github.com/fluxcd/source-controller/api`, `github.com/fluxcd/pkg/*`
1. [fluxcd/helm-controller](https://github.com/fluxcd/helm-controller)
    - modules: `api`
    - dependencies: `github.com/fluxcd/source-controller/api`, `github.com/fluxcd/pkg/*`
1. [fluxcd/image-reflector-controller](https://github.com/fluxcd/image-reflector-controller)
   - modules: `api`
   - dependencies: `github.com/fluxcd/pkg/*`
1. [fluxcd/image-automation-controller](https://github.com/fluxcd/image-automation-controller)
   - modules: `api`
   - dependencies: `github.com/fluxcd/source-controller/api`, `github.com/fluxcd/image-reflector-controller/api`, `github.com/fluxcd/pkg/*`
1. [fluxcd/notification-controller](https://github.com/fluxcd/notification-controller)
    - modules: `api`
    - dependencies: `github.com/fluxcd/pkg/*`
1. [fluxcd/flux2](https://github.com/fluxcd/flux2)
    - modules: `manifestgen`
    - dependencies: `github.com/fluxcd/source-controller/api`, `github.com/fluxcd/kustomize-controller/api`, `github.com/fluxcd/helm-controller/api`, `github.com/fluxcd/image-reflector-controller/api`, `github.com/fluxcd/image-automation-controller/api`, `github.com/fluxcd/notification-controller/api`, `github.com/fluxcd/pkg/*`
1. [fluxcd/terraform-provider-flux](https://github.com/fluxcd/terraform-provider-flux)
   - dependencies: `github.com/fluxcd/flux2/pkg/manifestgen`

## Release procedure

### Go packages

The Go packages in [fluxcd/pkg](https://github.com/fluxcd/pkg) are dedicated modules, 
each module has its own set of dependencies and release cycle.

Release procedure for a package:

1. Checkout the `main` branch and pull changes from remote.
1. Run `make release-<package name> VER=<next semver>`.

### Controllers

A toolkit controller has a dedicated module for its API, the API module 
has its own set of dependencies.

Release procedure for a controller and its API:

1. Checkout the `main` branch and pull changes from remote.
1. Create a `api/<next semver>` tag and push it to remote.
1. Create a new branch from `main` i.e. `release-<next semver>`. This
   will function as your release preparation branch.
1. Update the `github.com/fluxcd/<NAME>-controller/api` version in `go.mod`
1. Add an entry to the `CHANGELOG.md` for the new release and change the
   `newTag` value in ` config/manager/kustomization.yaml` to that of the
   semver release you are going to make. Commit and push your changes.
1. Create a PR for your release branch and get it merged into `main`.
1. Create a `<next semver>` tag for the merge commit in `main` and
   push it to remote.
1. Confirm CI builds and releases the newly tagged version.

### Flux

Release procedure for Flux:

1. Checkout the `main` branch and pull changes from remote.
1. Create a `<next semver>` tag form `main` and push it to remote.
1. Confirm CI builds and releases the newly tagged version.

## Upgrade Kubernetes modules

Flux has the following Kubernetes dependencies:

- `k8s.io/api`
- `k8s.io/apiextensions-apiserver`
- `k8s.io/apimachinery`
- `k8s.io/cli-runtime`
- `k8s.io/client-go`
- `sigs.k8s.io/controller-runtime`

**Note** that all `k8s.io/*` packages must have the same version in `go.mod` e.g.:

```
	k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/cli-runtime v0.20.2
	k8s.io/client-go v0.20.2
```

The specialised reconcilers depend on:

- kustomize-controller: `sigs.k8s.io/kustomize/api`
- image-automation-controller: `sigs.k8s.io/kustomize/kyaml`
- helm-controller: `helm.sh/helm/v3`

**Note** that the `k8s.io/*` version must be compatible with both `kustomize/api` and `helm/v3`.
If there is a breaking change in `client-go` we have to wait for Kustomize and Helm to upgrade first.

### Upgrade procedure:

`fluxcd/pkg`:

1. Update the `k8s.io/*` version in `pkg/apis/meta/go.mod`
1. Release the `apis/meta` package
1. Update `apis/meta` version in `pkg/runtime/go.mod`
1. Update the `k8s.io/*` version in `pkg/runtime/go.mod`
1. Update `sigs.k8s.io/controller-runtime` version in `pkg/runtime/go.mod`
1. Release the `runtime` package

`fluxcd/source-controller`:

1. Update the `github.com/fluxcd/pkg/apis/meta` version in `source-controller/api/go.mod` and `source-controller/go.mod`
1. Update the `k8s.io/*` version in `source-controller/api/go.mod` and `source-controller/go.mod`
1. Update the `sigs.k8s.io/controller-runtime` version in `source-controller/api/go.mod` and `source-controller/go.mod`
1. Update the `github.com/fluxcd/pkg/runtime` version in `source-controller/go.mod`
1. Release the `api` package

`fluxcd/<kustomize|helm|notification|image-automation>-controller`:

1. Update the `github.com/fluxcd/source-controller/api` version in `<NAME>-controller/api/go.mod`  and `<NAME>-controller/go.mod`
1. Update the `github.com/fluxcd/pkg/apis/meta` version in `<NAME>-controller/api/go.mod` and `<NAME>-controller/go.mod`
1. Update the `k8s.io/*` version in `<NAME>-controller/api/go.mod`  and `<NAME>-controller/go.mod`
1. Update the `github.com/fluxcd/pkg/runtime` version in `<NAME>-controller/go.mod`
1. Release the `api` package

`fluxcd/flux2`:

1. Update the `github.com/fluxcd/*-controller/api` version in `flux2/go.mod` (automated with [GitHub Actions](../../.github/workflows/update.yaml))
1. Update the `github.com/fluxcd/pkg/*` version in `flux2/go.mod`
1. Update the `k8s.io/*` and `github.com/fluxcd/pkg/runtime` version in `flux2/go.mod`

`fluxcd/terraform-provider-flux`:

1. Update the `github.com/fluxcd/flux2` version in `terraform-provider-flux/go.mod` (automated with [GitHub Actions](https://github.com/fluxcd/terraform-provider-flux/blob/main/.github/workflows/update.yaml))
