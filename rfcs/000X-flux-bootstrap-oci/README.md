# [RFC] Flux Bootstrap for OCI-compliant Container Registries

**Status:** provisional

**Creation date:** 2024-04-27

**Last update:** 2024-04-27

## Summary

Flux should allow a Git-less bootstrap procedure where the cluster desired state is stored in OCI artifacts.

On the client-side, the Flux CLI should offer a command for packaging its own Kubernetes manifests into
an OCI artifact and pushing the artifact to a container registry.

On the server-side, the Flux controllers should be configured to self-update from the registry
and reconcile the cluster state from OCI artifacts stored in the same or a different registry.

## Motivation

After the implementation of [RFC-0003](../0003-kubernetes-oci/README.md) in 2022 and the introduction
of the `OCIRepository` source, we had a recurring ask from users about improving the UX of running
Flux fully decoupled from Git.

Given that OCI registries are evolving into a generic artifact storage solution,
we should assist Flux users who don't want to depend on a Git (for any reason,
including auth and SSH key management) in their production infrastructure to
bootstrap and manage their Kubernetes clusters using OCI artifacts.

To decouple the clusters reconciliation from the Git repositories, Flux allows packaging and publishing
the Kubernetes manifests stored in Git to an OCI registry by running the `flux push artifact`
command in CI pipelines.

### Goals

- Add support to the Flux CLI for bootstrapping with a container registry as the source of truth.
- Make it easy for users to switch from Git repositories to OCI repositories.

### Non-Goals

- Automate the migration of Flux manifests from a Git bootstrap repository to OCI.

## Proposal

Implement the `flux bootstrap oci` command with the following specifications:

```shell
flux bootstrap oci \
--url=oci://<registry-url>/<flux-manifests>:<tag> \
--username=<registry-username> \
--password=<registry-password> \
--kustomization=<local/path/to/kustomization.yaml> \
--cluster-url=oci://<registry-url>/<fleet-manifests>:<tag> \
--cluster-path=<path/inside/oci/artifact>
```

The bootstrap shares the following flags with the `flux install` command:

```text
--cluster-domain string      internal cluster domain (default "cluster.local")
--components strings         list of components, accepts comma-separated values (default [source-controller,kustomize-controller,helm-controller,notification-controller])
--components-extra strings   list of components in addition to those supplied or defaulted, accepts values such as 'image-reflector-controller,image-automation-controller'
--image-pull-secret string   Kubernetes secret name used for pulling the toolkit images from a private registry
--log-level logLevel         log level, available options are: (debug, info, error) (default info)
--network-policy             deny ingress access to the toolkit controllers from other namespaces using network policies (default true)
--registry string            container registry where the toolkit images are published (default "ghcr.io/fluxcd")
--toleration-keys strings    list of toleration keys used to schedule the components pods onto nodes with matching taints
--version string             toolkit version, when specified the manifests are downloaded from https://github.com/fluxcd/flux2/releases
--watch-all-namespaces       watch for custom resources in all namespaces, if set to false it will only watch the namespace where the toolkit is installed (default true)
```

The Terraform/OpenTofu counterpart is the `flux_bootstrap_oci` provider that exposes
the same configuration options as the CLI.

The bootstrap operations are split into two phases:

- Install and self-update configuration for the Flux components.
- Cluster state reconciliation configuration.

### Install and self-update configuration

The command performs the following steps based on the `url`, `username`,
`password` and `kustomization` arguments:

1. Logs in to the OCI registry using the provided credentials.
2. Generates an OCI artifact from the Flux components manifests and the `kustomization.yaml` file.
3. Applies the Flux components manifests along with their customisations to the cluster.
4. Pushes the OCI artifact to the container registry using the specified tag.
5. Generates an image pull secret, an OCIRepository that points to the OCI artifact and
   a Flux Kustomization object that reconciles the OCI artifact contents.
6. Applies the image pull secret, OCIRepository and Flux Kustomization to the cluster.

Note that the creation of the image pull secret is skipped when
[Kubernetes Workload Identity](#story-2) is used for authentication to the container registry.

Artifacts pushed to the registry:
- `<registry-url>/<flux-manifests>:<checksum>` (immutable artifact)
- `<registry-url>/<flux-manifests>:<tag>` (tag pointing to the immutable artifact)

The OCI artifact has the following content:

```shell
./flux-system/
├── gotk_components.yaml
└── kustomization.yaml
```

Objects created by the command in the `flux-system` namespace:
- `flux-components` Secret
- `flux-components` OCIRepository
- `flux-components` Kustomization

### Cluster state reconciliation configuration

After the OCIRepository and Flux Kustomization called `flux` become ready, the command
continues with the following steps:

1. Logs in to the OCI registry where the cluster artifacts are stored using the provided credentials.
2. If the cluster OCI artifact is not found, an empty artifact is created
   and pushed to the registry using the provided tag.
3. Generates an image pull secret, an OCIRepository and a Flux Kustomization object
   that reconciles the cluster OCI artifact contents.
4. Applies the image pull secret, OCIRepository and Flux Kustomization to the cluster.

Note that the creation of the image pull secret is skipped when
[Kubernetes Workload Identity](#story-2) is used for authentication to the container registry.

Objects created by the command in the `flux-system` namespace:
- `flux-system` Secret
- `flux-system` OCIRepository
- `flux-system` Kustomization

If the cluster registry is the same as the Flux components registry, the command could reuse the
`flux-components` image pull secret.

If the cluster OCI artifact is not found, the generated one contains the following:

```shell
./clusters/my-cluster/ # taken from --cluster-path
└── kustomization.yaml # empty overlay with no resources
```

### Registry authentication

The `flux bootstrap oci` command supports the following authentication methods:

- Basic authentication with `--username` and `--password`. The credentials are stored in a Kubernetes Secret.
- OIDC authentication with `--provider=<aws|azure|gcp>`. No credentials are stored in the cluster, source-controller
  will use Kubernetes Workload Identity to authenticate to the registry.

To avoid passing the credentials as CLI flags, the password can be read from the standard input, e.g.:
`echo <password> | flux bootstrap oci` or using an environment variable `OCI_PASSWORD`.

If the registry is self-hosted and uses a self-signed TLS certificate,
the root CA certificate can be provided with the `--ca-file` flag.

If the registry is exposed on HTTP and not HTTPS, the `--allow-insecure-http`
flag can be used to force non-TLS connections.

### Signing and verification

The `flux bootstrap oci` command supports the following signing and verification methods:

- Cosign
- Notation

TODO: Add more details about the signing and verification methods, flags and options.

### User Stories

#### Story 1

> As a platform operator I want to bootstrap a Kubernetes cluster with Flux
> using OCI artifacts stored in a container registry.

The following example demonstrates how to bootstrap a Flux instance using GitHub Container Registry
as the OCI registry for Flux components and the cluster state.

```shell
flux bootstrap oci \
--url=oci://ghcr.io/stefanprodan/flux-manifests:production \
--username=<ghcr-username> \
--password=<ghcr-token> \
--kustomization=flux-manifests/kustomization.yaml \
--cluster-url=oci://ghcr.io/stefanprodan/fleet-manifests:production \
--cluster-username=<ghcr-username> \
--cluster-password=<ghcr-token> \
--cluster-path=clusters/production
```

Generated OCI artifacts:

- `ghcr.io/stefanprodan/flux-manifests:88b028f`
- `ghcr.io/stefanprodan/flux-manifests:production`
- `ghcr.io/stefanprodan/fleet-manifests:6f7a258`
- `ghcr.io/stefanprodan/fleet-manifests:production`

Objects created in the `flux-system` namespace:

Flux components reconciliation:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: OCIRepository
metadata:
  name: flux-components
  namespace: flux-system
spec:
  interval: 1m
  url: oci://ghcr.io/stefanprodan/flux-manifests
  ref:
    tag: production
  secretRef:
    name: flux-components
---
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: flux-components
  namespace: flux-system
spec:
    interval: 1h
    retryInterval: 5m
    sourceRef:
        kind: OCIRepository
        name: flux-components
    path: ./
    prune: true
```

Cluster state reconciliation:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: OCIRepository
metadata:
  name: flux-system
  namespace: flux-system
spec:
   interval: 1m
   url: oci://ghcr.io/stefanprodan/fleet-manifests
   ref:
    tag: production
   secretRef:
    name: flux-system
---
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: flux-system
  namespace: flux-system
spec:
    interval: 1h
    retryInterval: 5m
    sourceRef:
        kind: OCIRepository
        name: flux-system
    path: clusters/production
    prune: true
```

#### Story 2

> As a platform operator I want to bootstrap an EKS cluster with Flux
> using OCI artifacts stored in ECR.

The following example demonstrates how to bootstrap a Flux instance using ECR using IAM auth.
Assuming the EKS nodes have read-only access to ECR and the bastion host where
the Flux CLI is running has read and write access to ECR:

```shell
flux bootstrap oci \
--provider=aws \
--url=oci://aws_account_id.dkr.ecr.us-west-2.amazonaws.com/flux-manifests:production \
--kustomization=flux-manifests/kustomization.yaml \
--cluster-url=oci://aws_account_id.dkr.ecr.us-west-2.amazonaws.com/fleet-manifests:production \
--cluster-path=clusters/production
```

Note that when using Kubernetes Workload Identity instead of the worker node IAM role,
the `kustomization.yaml` must contain patches for the source-controller Service Account
as described [here](https://fluxcd.io/flux/installation/configuration/workload-identity/).

#### Story 3

> As a platform operator I want to sync the cluster state with the fleet Git repository.

Push changes from the fleet Git repository to the container registry:

```shell
# clone the fleet Git repository
git clone https://github.com/stefanprodan/fleet.git
cd fleet
git switch main

# push the contents the fleet OCI repository and tag it with the commit short SHA
flux push artifact oci://ghcr.io/stefanprodan/fleet-manifests:$(git rev-parse --short HEAD) \
--path="./" \
--source="$(git config --get remote.origin.url)" \
--revision="$(git branch --show-current)@sha1:$(git rev-parse HEAD)"

# tag the new version for production
flux tag artifact oci://ghcr.io/stefanprodan/fleet-manifests:$(git rev-parse --short HEAD) \
--tag=production
```

This operation can be automated using the Flux GitHub Action.

The Git repository structure would be similar to the
[flux2-kustomize-helm-example](https://github.com/fluxcd/flux2-kustomize-helm-example) with the following changes:

- The `clusters/production/flux-system` directory is no more.
- The Flux Kustomization objects defined in the `clusters/production` directory, such as
  `infrastructure.yaml` and `apps.yaml`, have the `.spec.sourceRef` set to
  `kind: OCIRepository` and  `name: flux-system`.

#### Story 4

> As a platform operator I want to update the Flux controllers on my production cluster
> from CI without access to the Kubernetes API.

Download the latest CLI version and update Flux directly in the registry, without rerunning bootstrap:

```shell
# pull the latest manifests from the registry
flux pull artifact oci://ghcr.io/stefanprodan/flux-manifests:production \
--output=./flux-manifests

# update the Flux components manifests
flux install --export > ./flux-manifests/flux-system/gotk-components.yaml

# calculate the checksum of the manifests
checksum=$(grep -ar -e . ./flux-manifests/ | shasum | cut -c-16)

# extract the Flux version and commit
flux_version=$(flux version --client | awk '{print $2}')
flux_commit=$(go version -m  $(which flux) | grep vcs.revisio | awk -F= '{print $NF}')

# push the updated manifests to the registry using the checksum as tag
flux push artifact oci://ghcr.io/stefanprodan/flux-manifests:${checksum} \
--path="./flux-manifests" \
--source="https://github.com/fluxcd/flux2" \
--revision="${flux_version}@sha1:${flux_commit}"

# tag the new version for production
flux tag artifact oci://ghcr.io/stefanprodan/flux-manifests:${checksum} \
--tag=production
```

This operation could be simplified by implementing a dedicated CLI command and/or GitHub Action.

#### Story 5

> As a platform operator I want to update the registry credentials on my clusters.

To rotate the registry credentials, generate a new GitHub token and overwrite the image pull secret:

```shell
flux create secret oci flux-system \
--url=ghcr.io \
--username=<ghcr-username> \
--password=<ghcr-token>
```

Another option is to rerun the bootstrap command with the new credentials.

#### Story 6

> As a platform operator I want to know the Git repository used to generate the OCI artifact
> where a Kubernetes Deployment running on the cluster belongs to.

To determine the source of a Kubernetes object in-cluster, run:

```shell
flux -n default trace deploy/podinfo
```

The trace command will display the OCI artifact URL, tag and digest along
with the Git repository URL, branch and commit.

## Design Details

The bootstrap feature will be implemented as a Go package under `fluxcd/flux2/pkg/bootstrap/oci`
using the [fluxcd/pkg/oci](https://github.com/fluxcd/pkg/tree/main/oci)
library for OCI operations such as auth, push, pull, tag, etc.

Both the Flux CLI and the Terraform/OpenTofu provider will use the `fluxcd/flux2/pkg/bootstrap/oci` package
and expose the same configuration options.

### Enabling the feature

The feature is enabled by default.

## Implementation History

* NONE