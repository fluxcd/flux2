# RFC-xxxx Flux OCI support for Helm

**Status:** provisional

**Creation date:** 2022-03-30

**Last update:** 2022-03-30

## Summary

Given that Helm v3.8 supports [OCI](https://helm.sh/docs/topics/registries/) for package distribution,
we should extend the Flux Source API to allow fetching Helm charts from container registries.

## Motivation

Helm OCI support is one of the most requested feature in Flux
as seen on this [issue](https://github.com/fluxcd/source-controller/issues/124).

### Goals

- Add support for fetching Helm charts stored as OCI artifacts with minimal API changes to Flux.
- Make it easy for users to switch from HTTP/S Helm repositories to OCI repositories.

### Non-Goals

- Introduce a new API kind for referencing charts stored as OCI artifacts.

## Proposal

Introduce an optional field called `type` to the `HelmRepository` spec.

When not specified, the `spec.type` field defaults to `Default` which preserve the current `HelmRepository` API behaviour.

When the `spec.type` field is set to `OCI`, the `spec.url` field must be prefixed with `oci://` (to follow the Helm conventions).
For `oci://` URLs, source-controller will use the Helm SDK and the `oras` library to connect to the OCI remote storage.
For authentication, the controller will use Kubernetes secrets of `kubernetes.io/dockerconfigjson` type.

### User Stories

#### Story 1

> As a developer I want to use Flux `HelmReleases` that refer to Helm charts stored
> as OCI artifacts in GitHub Container Registry.

First create a secret using a GitHub token that allows access to GHCR:

```sh
kubectl create secret docker-registry ghcr-charts \
    --docker-server=ghcr.io \
    --docker-username=$GITHUB_USER \
    --docker-password=$GITHUB_TOKEN
```

Then define a `HelmRepository` of type `OCI` and reference the `dockerconfig` secret:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: HelmRepository
metadata:
  name: ghcr-charts
  namespace: default
spec:
  type: OCI
  url: oci://ghcr.io/my-org/charts/
  secretRef:
    name: ghcr-charts
```

And finally in Flux `HelmReleases`, refer to the ghcr-charts `HelmRepository`:

```yaml
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: podinfo
  namespace: default
spec:
  interval: 60m
  chart:
    spec:
      chart: my-app
      version: '1.0.x'
      sourceRef:
        kind: HelmRepository
        name: ghcr-charts
      interval: 1m # check for new OCI artifacts every minute
```

### Alternatives

We could use introduce a new API type e.g. `HelmOCIRepository`. That is considered unpractical,
as there is no benefit for users in having a dedicated kind instead of a `type` filed in the current
`HelmRepository` API. Adding a `type` filed to the spec follows the Flux Bucket API design, where 
the same Kind servers different implementations: AWS S3 vs Azure Blob vs Google Storage.

## Design Details

In source-controller we'll add a new predicate for indexing `HelmRepositories` based on the `spec.type` field.

When the `spec.type` field is set to `OCI`, the `HelmRepositoryReconciler`
will set the `HelmRepository` Ready status to `False` if the URL is not prefixed with `oci://`,
otherwise the Ready status will be set to `True`.

The current `HelmChartReconciler` will use the `HelmRepositories` with `type: Default`.
For `type: OCI` we'll introduce a new reconciler `HelmChartOCIReconciler` that uses `oras` to download charts
and their dependencies.

### Enabling the feature

The feature is enabled by default.
