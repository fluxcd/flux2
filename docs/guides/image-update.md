# Automate image updates to Git

This guide walks you through configuring container image scanning and deployment rollouts with Flux.

For a container image you can configure Flux to:

- scan the container registry and fetch the image tags
- select the latest tag based on a semver range
- replace the tag in Kubernetes manifests (YAML format)
- checkout a branch, commit and push the changes to the remote Git repository
- apply the changes in-cluster and rollout the container image

!!! warning "Alpha version"
    Note that the image update feature is currently alpha and, it only supports **semver** filters.
    In the future we plan to add support for other filtering options.
    Please see the [roadmap](../roadmap/index.md) for more details.

For production environments, this feature allows you to automatically deploy application patches
(CVEs and bug fixes), and keep a record of all deployments in Git history.

For staging environments, this features allow you to deploy the latest prerelease of an application,
without having to manually edit its deployment manifests in Git.

Production CI/CD workflow:

* DEV: push a bug fix to the app repository
* DEV: bump the patch version and release e.g. `v1.0.1`
* CI: build and push a container image tagged as `registry.domain/org/app:v1.0.1`
* CD: pull the latest image metadata from the app registry (Flux image scanning)
* CD: update the image tag in the app manifest to `v1.0.1` (Flux cluster to Git reconciliation)
* CD: deploy `v1.0.1` to production clusters (Flux Git to cluster reconciliation)

## Prerequisites

You will need a Kubernetes cluster version 1.16 or newer and kubectl version 1.18.
For a quick local test, you can use [Kubernetes kind](https://kind.sigs.k8s.io/docs/user/quick-start/).
Any other Kubernetes setup will work as well.

In order to follow the guide you'll need a GitHub account and a
[personal access token](https://help.github.com/en/github/authenticating-to-github/creating-a-personal-access-token-for-the-command-line)
that can create repositories (check all permissions under `repo`).

Export your GitHub personal access token and username:

```sh
export GITHUB_TOKEN=<your-token>
export GITHUB_USER=<your-username>
```

## Install Flux

Install Flux with the image automation components:

```sh
flux bootstrap github \
  --components-extra=image-reflector-controller,image-automation-controller \
  --owner=$GITHUB_USER \
  --repository=flux-image-updates \
  --branch=main \
  --path=clusters/my-cluster \
  --token-auth \
  --personal
```

The bootstrap command creates a repository if one doesn't exist, and commits the manifests for the
Flux components to the default branch at the specified path. It then configures the target cluster to
synchronize with the specified path inside the repository.

!!! hint "GitLab and other Git platforms"
    You can install Flux and bootstrap repositories hosted on GitLab, BitBucket, Azure DevOps and
    any other Git provider that support SSH or token-based authentication.
    When using SSH, make sure the deploy key is configured with write access.
    Please see the [installation guide](installation.md) for more details.

## Deploy a demo app

We'll be using a tiny webapp called [podinfo](https://github.com/stefanprodan/podinfo) to
showcase the image update feature.

Clone your repository with:

```sh
git clone https://github.com/$GITHUB_USER/flux-image-updates
cd flux-image-updates
```

Add the podinfo Kubernetes deployment file inside `cluster/my-cluster`:

```sh
curl -sL https://raw.githubusercontent.com/stefanprodan/podinfo/5.0.0/kustomize/deployment.yaml \
> ./clusters/my-cluster/podinfo-deployment.yaml
```

Commit and push changes to main branch:

```sh
git add -A && \
git commit -m "add podinfo deployment" && \
git push origin main
```

Tell Flux to pull and apply the changes or wait one minute for Flux to detect the changes on its own:

```sh
flux reconcile kustomization flux-system --with-source
```

Print the podinfo image deployed on your cluster:

```console
$ kubectl get deployment/podinfo -oyaml | grep 'image:'
image: ghcr.io/stefanprodan/podinfo:5.0.0
```

## Configure image scanning

Create an `ImageRepository` to tell Flux which container registry to scan for new tags:

```sh
flux create image repository podinfo \
--image=ghcr.io/stefanprodan/podinfo \
--interval=1m \
--export > ./clusters/my-cluster/podinfo-registry.yaml
```

The above command generates the following manifest:

```yaml
apiVersion: image.toolkit.fluxcd.io/v1alpha1
kind: ImageRepository
metadata:
  name: podinfo
  namespace: flux-system
spec:
  image: ghcr.io/stefanprodan/podinfo
  interval: 1m0s
```

For private images, you can create a Kubernetes secret
in the same namespace as the `ImageRepository` with
`kubectl create secret docker-registry`. Then you can configure
Flux to use the credentials by referencing the Kubernetes secret
in the `ImageRepository`:

```yaml
kind: ImageRepository
spec:
  secretRef:
    name: regcred
```

!!! hint "Storing secrets in Git"
    Note that if you want to store the image pull secret in Git,  you can encrypt
    the manifest with [Mozilla SOPS](mozilla-sops.md) or [Sealed Secrets](sealed-secrets.md).

Create an `ImagePolicy` to tell Flux which semver range to use when filtering tags:

```sh
flux create image policy podinfo \
--image-ref=podinfo \
--interval=1m \
--semver=5.0.x \
--export > ./clusters/my-cluster/podinfo-policy.yaml
```

The above command generates the following manifest:

```yaml
apiVersion: image.toolkit.fluxcd.io/v1alpha1
kind: ImagePolicy
metadata:
  name: podinfo
  namespace: flux-system
spec:
  imageRepositoryRef:
    name: podinfo
  policy:
    semver:
      range: 5.0.x
```

!!! hint "semver ranges"
    A semver range that includes stable releases can be defined with
    `1.0.x` (patch versions only) or `>=1.0.0 <2.0.0` (minor and patch versions).
    If you want to include pre-release e.g. `1.0.0-rc.1`,
    you can define a range like: `>1.0.0-rc <2.0.0`.

Commit and push changes to main branch:

```sh
git add -A && \
git commit -m "add podinfo image scan" && \
git push origin main
```

Tell Flux to pull and apply changes:

```sh
flux reconcile kustomization flux-system --with-source
```

Wait for Flux to fetch the image tag list from GitHub container registry:

```console
$ flux get image repository podinfo
NAME   	READY	MESSAGE                       	LAST SCAN
podinfo	True 	successful scan, found 13 tags	2020-12-13T17:51:48+02:00
```

Find which image tag matches the policy semver range with:

```console
$ flux get image policy podinfo
NAME   	READY	MESSAGE                   
podinfo	True 	Latest image tag for 'ghcr.io/stefanprodan/podinfo' resolved to: 5.0.3
```

## Configure image updates

Edit the `podinfo-deploy.yaml` and add a maker to tell Flux which policy to use when updating the container image:

```yaml
spec:
  containers:
  - name: podinfod
    image: ghcr.io/stefanprodan/podinfo:5.0.0 # {"$imagepolicy": "flux-system:podinfo"}
```

Create an `ImageUpdateAutomation` to tell Flux which Git repository to write image updates to:

```sh
flux create image update flux-system \
--git-repo-ref=flux-system \
--branch=main \
--author-name=fluxcdbot \
--author-email=fluxcdbot@users.noreply.github.com \
--commit-template="[ci skip] update image" \
--export > ./clusters/my-cluster/flux-system-automation.yaml
```

The above command generates the following manifest:

```yaml
apiVersion: image.toolkit.fluxcd.io/v1alpha1
kind: ImageUpdateAutomation
metadata:
  name: flux-system
  namespace: flux-system
spec:
  checkout:
    branch: main
    gitRepositoryRef:
      name: flux-system
  commit:
    authorEmail: fluxcdbot@users.noreply.github.com
    authorName: fluxcdbot
    messageTemplate: '[ci skip] update image'
  interval: 1m0s
  update:
    setters: {}
```

Commit and push changes to main branch:

```sh
git add -A && \
git commit -m "add image updates automation" && \
git push origin main
```

Note that the `ImageUpdateAutomation` runs all the policies found in its namespace at the specified interval.

Tell Flux to pull and apply changes:

```sh
flux reconcile kustomization flux-system --with-source
```

In a couple of seconds Flux will push a commit to your repository with
the latest image tag that matches the podinfo policy:

```console
$ git pull && cat clusters/my-cluster/podinfo-deployment.yaml | grep "image:"
image: ghcr.io/stefanprodan/podinfo:5.0.3 # {"$imagepolicy": "flux-system:podinfo"}
```

Wait for Flux to apply the latest commit on the cluster and verify that podinfo was updated to `5.0.3`:

```console
$ watch "kubectl get deployment/podinfo -oyaml | grep 'image:'"
image: ghcr.io/stefanprodan/podinfo:5.0.3
```

## Configure image update for custom resources

Besides Kubernetes native kinds (Deployment, StatefulSet, DaemonSet, CronJob),
Flux can be used to patch image tags in any Kubernetes custom resource stored in Git.

The image policy marker format is:

* `{"$imagepolicy": "<policy-namespace>:<policy-name>"}`
* `{"$imagepolicy": "<policy-namespace>:<policy-name>:tag"}`


`HelmRelease` example:

```yaml
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: podinfo
  namespace: default
spec:
  values:
    image:
      repository: ghcr.io/stefanprodan/podinfo
      tag: 5.0.0  # {"$imagepolicy": "flux-system:podinfo:tag"}
```

Tekton `Task` example:

```yaml
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: golang
  namespace: default
spec:
  steps:
    - name: golang
      image: docker.io/golang:1.15.6 # {"$imagepolicy": "flux-system:golang"}
```

Flux `Kustomization` example:

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1beta1
kind: Kustomization
metadata:
  name: podinfo
  namespace: default
spec:
  images:
    - name: ghcr.io/stefanprodan/podinfo
      newName: ghcr.io/stefanprodan/podinfo
      newTag: 5.0.0 # {"$imagepolicy": "flux-system:podinfo:tag"}
```

Kustomize config (`kustomization.yaml`) example:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- deployment.yaml
images:
- name: ghcr.io/stefanprodan/podinfo
  newName: ghcr.io/stefanprodan/podinfo
  newTag: 5.0.0 # {"$imagepolicy": "flux-system:podinfo:tag"}
```
