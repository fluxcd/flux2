# RFC-xxxx Flux OCI support for Kubernetes manifests

**Status:** provisional

**Creation date:** 2022-03-31

**Last update:** 2022-04-13

## Summary

Flux should be able to distribute and reconcile Kubernetes configuration packaged as OCI artifacts.

On the client-side, the Flux CLI should offer a command for packaging Kubernetes configs into
an OCI artifact and pushing the artifact to a container registry using the Docker config file
and the Docker credential helpers for authentication.

On the server-side, the Flux source-controller should offer a dedicated API Kind for defining
how OCI artifacts are pulled from container registries and how the artifact's authenticity can be verified.
Flux should be able to work with any type of artifact even if it's not created with the Flux CLI.

## Motivation

Given that OCI registries are evolving into a generic artifact storage solution,
we should extend Flux to allow fetching Kubernetes manifests and related configs
from container registries similar to how Flux works with Git and Bucket storage.

With OCI support, Flux users can automate artifact updates to Git in the same way
they do today for container images.

### Goals

- Add support to the Flux CLI for packaging Kubernetes manifests and related configs into OCI artifacts.
- Add support to Flux source-controller for fetching configs stored as OCI artifacts.
- Make it easy for users to switch from Git repositories and Buckets to OCI repositories.

### Non-Goals

- Introduce a new OCI media type for artifacts containing Kubernetes manifests.

## Proposal

### Push artifacts

Flux users should be able to package a local directory containing Kubernetes configs into a tarball
and push the archive to a container registry as an OCI artifact.

```sh
flux push artifact docker.io/org/app-config:v1.0.0 -f ./deploy
```

Flux CLI with produce artifacts of type `application/vnd.oci.image.config.v1+json`.
The directory pointed to by `-f` is archived and compressed in the `tar+gzip` format
and the layer media type is set to `application/vnd.oci.image.layer.v1.tar+gzip`.

> A proof-of-concept CLI implementation for distributing Kubernetes configs as OCI artifacts
> is available at [kustomizer.dev](https://github.com/stefanprodan/kustomizer).

### Pull artifacts

Flux users should be able to define a source for pulling manifests inside the cluster from an OCI repository.

```yaml
apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: OCIRepository
metadata:
  name: app-config
  namespace: flux-system
spec:
  interval: 10m
  url: docker.io/org/app-config
  ref:
    tag: v1.0.0
```

An `OCIRepository` can refer to an artifact by tag, digest or semver range:

```yaml
spec:
  ref:
    # one of
    tag: "latest"
    digest: "sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2"
    semver: "6.0.x"
```

To verify the authenticity of an artifact, the Sigstore cosign public key can be supplied with:

```yaml
spec:
  verify:
    provider: cosign
    secretRef:
      name: cosign-key
```

### Pull artifacts from private repositories

For authentication purposes, Flux users can choose between supplying static credentials with Kubernetes secrets
and cloud-based OIDC using an IAM role binding to the source-controller Kubernetes service account.

#### Basic auth

For private repositories hosted on DockerHub, GitHub, Quay, self-hosted Docker Registry and others,
the credentials can be supplied with:

```yaml
spec:
  secretRef:
    name: regcred
```

The `secretRef` points to a Kubernetes secret in the same namespace as the `OCIRepository`,
the secret type must be `kubernetes.io/dockerconfigjson`:

```shell
kubectl create secret docker-registry regcred \
  --docker-server=<your-registry-server> \
  --docker-username=<your-name> \
  --docker-password=<your-pword>
```

#### Client cert auth

For private repositories which require a certificate to authenticate,
the client certificate, private key and the CA certificate (if self-signed), can be provided with:

```yaml
spec:
  certSecretRef:
    name: regcert
```

The `certSecretRef` points to a Kubernetes secret in the same namespace as the `OCIRepository`:

```shell
kubectl create secret generic regcert \
  --from-file=certFile=client.crt \
  --from-file=keyFile=client.key \
  --from-file=caFile=ca.crt
```

#### OIDC auth

When Flux runs on AKS, EKS or GKE, an IAM role (that grants read-only access to ACR, ECR or GCR)
can be used to bind the `source-controller` to the IAM role.

Similar to image-reflector-controller
[auto-login feature](https://fluxcd.io/docs/guides/image-update/#imagerepository-cloud-providers-authentication),
source-controller will expose dedicated flags for each cloud provider:

```sh
--aws-autologin-for-ecr
--azure-autologin-for-acr
--gcp-autologin-for-gcr
```

We should extract the flags and the AWS, Azure and GCP auth implementations from image-reflector-controller into 
`fluxcd/pkg/oci/auth` to reuses the code in source-controller.

### Reconcile artifacts

The `OCIRepository` can be used as a drop-in replacement for `GitRepository` and `Bucket` sources.
For example a Flux Kustomization can refer to an `OCIRepository` and reconcile the manifests found in the OCI artifact:

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: app
  namespace: flux-system
spec:
  interval: 10m
  sourceRef:
    kind: OCIRepository
    name: app-config
  path: ./
```

### User Stories

#### Story 1

> As a developer I want to publish my app Kubernetes manifests to the same GHCR registry
> where I publish my app containers.

First login to GHCR with Docker:

```sh
docker login ghcr.io -u ${GITHUB_USER} -p ${GITHUB_TOKEN}
```

Build your app container image and push it to GHCR:

```sh
docker build -t ghcr.io/org/my-app:v1.0.0 .
docker push ghcr.io/org/my-app:v1.0.0
```

Edit the app deployment manifest and set the new image tag.
Then push the Kubernetes manifests to GHCR:

```sh
flux push artifact ghcr.io/org/my-app-config:v1.0.0 -f ./deploy
```

Sign the config image with cosign:

```sh
cosign sign --key cosign.key ghcr.io/org/my-app-config:v1.0.0
```

#### Story 2

> As a developer I want to deploy my app using Kubernetes manifests published as OCI artifacts to GHCR.

First create a secret using a GitHub token that allows access to GHCR:

```sh
kubectl create secret docker-registry my-app-regcred \
    --docker-server=ghcr.io \
    --docker-username=$GITHUB_USER \
    --docker-password=$GITHUB_TOKEN
```

Then create a secret with your cosgin public key:

```sh
kubectl create secret generic my-app-cosgin-key \
    --from-file=cosign.pub=cosign/my-key.pub
```

Then define an `OCIRepository` to fetch and verify the latest app config version:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: OCIRepository
metadata:
  name: app-config
  namespace: default
spec:
  interval: 10m
  url: ghcr.io/org/my-app-config
  ref:
    semver: "1.x"
  secretRef:
    name: my-app-regcred
  verify:
    provider: cosign
    secretRef:
      name: my-app-cosgin-key
```

And finally, create a Flux Kustomization to reconcile the app on the cluster:

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: app
  namespace: default
spec:
  interval: 10m
  sourceRef:
    kind: OCIRepository
    name: app-config
  path: ./
  prune: true
  wait: true
  timeout: 2m
```

### Alternatives

TODO

## Design Details

TODO

### Enabling the feature

The feature is enabled by default.
