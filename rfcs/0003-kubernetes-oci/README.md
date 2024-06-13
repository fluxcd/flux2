# RFC-0003 Flux OCI support for Kubernetes manifests

**Status:** implemented

**Creation date:** 2022-03-31

**Last update:** 2023-11-07

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

- Enforce a specific OCI media type for artifacts containing Kubernetes manifests or any other configs.

## Proposal

### Push artifacts

Flux users should be able to package a local directory containing Kubernetes configs into a tarball
and push the archive to a container registry as an OCI artifact.

```sh
flux push artifact oci://docker.io/org/app-config:v1.0.0 \
  --source="$(git config --get remote.origin.url)" \
  --revision="sha1:$(git rev-parse HEAD)" \
  --path="./deploy"
```

The Flux CLI will produce OCI artifacts by setting the config layer
media type to `application/vnd.cncf.flux.config.v1+json`.

The directory pointed to by `--path` is archived and compressed in the `tar+gzip` format
and the layer media type is set to `application/vnd.cncf.flux.content.v1.tar+gzip`.

The source and revision are added to the OCI artifact as Open Containers standard annotations:

```json
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "annotations": {
    "org.opencontainers.image.created": "2023-02-10T09:06:09Z",
    "org.opencontainers.image.revision": "sha1:6ea3e5b4da159fcb4a1288f072d34c3315644bcc",
    "org.opencontainers.image.source": "https://github.com/fluxcd/flux2"
  }
}
```

To ease the promotion workflow of a specific version from one environment to another, the CLI
should offer a tagging command.

```sh
flux tag artifact oci://docker.io/org/app-config:v1.0.0 --tag=latest --tag=production
```

To view all the available artifacts in a repository and their metadata, the CLI should
offer a list command.

```sh
flux list artifacts oci://docker.io/org/app-config
```

To help inspect artifacts, the Flux CLI will offer a `build` and a `pull` command for generating
tarballs locally and for downloading the tarballs from remote container registries.

```sh
flux build artifact --path ./deploy --output tmp/artifact.tgz
flux pull artifact oci://docker.io/org/app-config:v1.0.0 --output ./manifests
```

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
  url: oci://docker.io/org/app-config
  ref:
    tag: v1.0.0
```

The `spec.url` field points to the container image repository in the format `oci://<host>:<port>/<org-name>/<repo-name>`. 
Note that specifying a tag or digest is not in accepted for this field. The `spec.url` value is used by the controller
to fetch the list of tags from the remote OCI repository.

An `OCIRepository` can refer to an artifact by tag, digest or semver range:

```yaml
spec:
  ref:
    # one of
    tag: "latest"
    digest: "sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2"
    semver: "6.0.x"
```

### Layer selection

By default, Flux assumes that the first layer of the OCI artifact contains the Kubernetes configuration.
For multi-layer artifacts created by other tools than Flux CLI
(e.g. [oras](https://github.com/oras-project/oras),
[crane](https://github.com/google/go-containerregistry/tree/main/cmd/crane)),
users can specify the [media type](https://github.com/opencontainers/image-spec/blob/v1.0.2/media-types.md) of the layer
which contains the tarball with Kubernetes manifests.

```yaml
spec:
  layerSelector:
    mediaType: "application/vnd.cncf.flux.content.v1.tar+gzip"
```

If the layer selector matches more than one layer,
the first layer matching the specified media type will be used.
Note that Flux requires that the OCI layer is
[compressed](https://github.com/opencontainers/image-spec/blob/v1.0.2/layer.md#gzip-media-types)
in the `tar+gzip` format.

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

For image pull secrets attached to a service account, the account name can be specified with:

```yaml
spec:
  serviceAccountName: regsa
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

```yaml
spec:
  provider: aws
```

The provider accepts the following values: `generic`, `aws`, `azure` and `gcp`. When the provider is
not specified, it defaults to `generic`. When the provider is set to `aws`, `azure` or `gcp`, the 
controller will use a specific cloud SDK for authentication purposes. If both `spec.secretRef` and
a non-generic provider are present in the definition, the controller will use the static credentials
from the referenced secret.

### Verify artifacts

To verify the authenticity of the OCI artifacts, Flux will use the Sigstore Go SDK and implement verification
for artifacts which were either signed with keys generated by Cosign or signed using the Cosign
[keyless method](https://github.com/sigstore/cosign/blob/main/KEYLESS.md).

To enable signature verification, the Cosign public key can be supplied with:

```yaml
spec:
  verify:
    provider: cosign
    secretRef:
      name: cosign-key
```

For verifying public artifacts which are signed using the keyless method,
the `.spec.verify.matchOIDCIdentity` field must be used instead of
 `spec.verify.secretRef`.

```yaml
spec:
  verify:
    provider: cosign
    matchOIDCIdentity:
      - issuer: "^https://token.actions.githubusercontent.com$"
        subject: "^https://github.com/org/app-repository.*$"
```

The `matchOIDCIdentity` entries must contain the following fields:

- `.issuer`, regexp that matches against the OIDC issuer.
- `.subject`, regexp that matches against the subject identity in the certificate.

The entries are evaluated in an OR fashion, i.e. the identity is deemed to be
verified if any one entry successfully matches against the identity.

When using the keyless method, Flux will verify the signatures in the Rekor
transparency log instance hosted at [rekor.sigstore.dev](https://rekor.sigstore.dev/).

### Reconcile artifacts

The `OCIRepository` can be used as a drop-in replacement for `GitRepository` and `Bucket` sources.
For example, a Flux Kustomization can refer to an `OCIRepository` and reconcile the manifests found in the OCI artifact:

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
flux push artifact oci://ghcr.io/org/my-app-config:v1.0.0 \
	--source="$(git config --get remote.origin.url)" \
	--revision="sha1:$(git rev-parse HEAD)"\
	--path="./deploy"
```

Sign the config image with cosign:

```sh
cosign sign --key cosign.key ghcr.io/org/my-app-config:v1.0.0
```

Mark `v1.0.0` as latest:

```sh
flux tag artifact oci://ghcr.io/org/my-app-config:v1.0.0 --tag latest
```

List the artifacts and their metadata with:

```console
$ flux list artifacts oci://ghcr.io/org/my-app-config
ARTIFACT                                DIGEST                                                                 	SOURCE                                          REVISION                                      
ghcr.io/org/my-app-config:latest   	sha256:45b95019d30af335137977a369ad56e9ea9e9c75bb01afb081a629ba789b890c	https://github.com/org/my-app-config.git   	sha1:20b3a674391df53f05e59a33554973d1cbd4d549	
ghcr.io/org/my-app-config:v1.0.0	sha256:45b95019d30af335137977a369ad56e9ea9e9c75bb01afb081a629ba789b890c	https://github.com/org/my-app-config.git	sha1:3f45e72f0d3457e91e3c530c346d86969f9f4034	
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
  url: oci://ghcr.io/org/my-app-config
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
  path: ./deploy
  prune: true
  wait: true
  timeout: 2m
```

## Design Details

The Flux controllers and CLI will use the [fluxcd/pkg/oci](https://github.com/fluxcd/pkg/tree/main/oci)
library for OCI operations such as push, pull, tag, list tags, etc.

For authentication purposes, the `flux <verb> artifact` commands will use the `~/.docker/config.json`
config file and the Docker credential helpers. On Cloud VMs without Docker installed, the CLI will
use context-based authorization for AWS, Azure and GCP.

The Flux CLI will produce OCI artifacts with the following format:

```json
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "config": {
    "mediaType": "application/vnd.cncf.flux.config.v1+json",
    "size": 233,
    "digest": "sha256:1b80ecb1c04d4a9718a6094a00ed17b76ea8ff2bb846695fa38e7492d34f505c"
  },
  "layers": [
    {
      "mediaType": "application/vnd.cncf.flux.content.v1.tar+gzip",
      "size": 19081,
      "digest": "sha256:46c2b334705cd08db1a6fa46f860cd3364fc1a3636eea37a9b35537549086a1c"
    }
  ],
  "annotations": {
    "org.opencontainers.image.created": "2023-02-10T09:06:09Z",
    "org.opencontainers.image.revision": "sha1:6ea3e5b4da159fcb4a1288f072d34c3315644bcc",
    "org.opencontainers.image.source": "https://github.com/fluxcd/flux2"
  }
}
```

The source-controller will extract the first layer from the OCI artifact, and will repackage it
as an internal `sourcev1.Artifact`. The internal artifact revision will be set to the OCI SHA256 digest
and the OpenContainers annotation will be copied to the internal artifact metadata:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: OCIRepository
metadata:
  creationTimestamp: "2022-06-22T09:14:19Z"
  finalizers:
  - finalizers.fluxcd.io
  generation: 1
  name: podinfo
  namespace: oci
  resourceVersion: "6603"
  uid: 42e0b9f0-021c-476d-86c7-2cd20747bfff
spec:
  interval: 10m
  ref:
    tag: 6.1.6
  timeout: 60s
  url: oci://ghcr.io/stefanprodan/manifests/podinfo
status:
  artifact:
    checksum: d7e924b4882e55b97627355c7b3d2e711e9b54303afa2f50c25377f4df66a83b
    lastUpdateTime: "2022-06-22T09:14:21Z"
    metadata:
      org.opencontainers.image.created: "2023-02-10T09:06:09Z"
      org.opencontainers.image.revision: sha1:b3b00fe35424a45d373bf4c7214178bc36fd7872
      org.opencontainers.image.source: https://github.com/stefanprodan/podinfo.git
    path: ocirepository/oci/podinfo/3b6cdcc7adcc9a84d3214ee1c029543789d90b5ae69debe9efa3f66e982875de.tar.gz
    revision: sha256:3b6cdcc7adcc9a84d3214ee1c029543789d90b5ae69debe9efa3f66e982875de
    size: 1105
    url: http://source-controller.flux-system.svc.cluster.local./ocirepository/oci/podinfo/3b6cdcc7adcc9a84d3214ee1c029543789d90b5ae69debe9efa3f66e982875de.tar.gz
  conditions:
  - lastTransitionTime: "2022-06-22T09:14:21Z"
    message: stored artifact for revision 'sha256:3b6cdcc7adcc9a84d3214ee1c029543789d90b5ae69debe9efa3f66e982875de'
    observedGeneration: 1
    reason: Succeeded
    status: "True"
    type: Ready
  - lastTransitionTime: "2022-06-22T09:14:21Z"
    message: stored artifact for revision 'sha256:3b6cdcc7adcc9a84d3214ee1c029543789d90b5ae69debe9efa3f66e982875de'
    observedGeneration: 1
    reason: Succeeded
    status: "True"
    type: ArtifactInStorage
  observedGeneration: 1
  url: http://source-controller.flux-system.svc.cluster.local./ocirepository/oci/podinfo/latest.tar.gz
```

### Enabling the feature

The feature is enabled by default.

## Implementation History

* **2022-08-08** Partially implemented by [source-controller#788](https://github.com/fluxcd/source-controller/pull/788)
* **2022-08-11** First implementation released with [flux2 v0.32.0](https://github.com/fluxcd/flux2/releases/tag/v0.32.0)
* **2022-08-29** Select layer by OCI media type released with [flux2 v0.33.0](https://github.com/fluxcd/flux2/releases/tag/v0.33.0)
* **2022-09-29** Verifying OCI artifacts with Cosign released with [flux2 v0.35.0](https://github.com/fluxcd/flux2/releases/tag/v0.35.0)
* **2023-02-20** Custom OCI media types released with [flux2 v0.40.0](https://github.com/fluxcd/flux2/releases/tag/v0.40.0)
* **2023-10-31** OIDC identity verification implemented in 
  [source-controller#1250](https://github.com/fluxcd/source-controller/pull/1250)
