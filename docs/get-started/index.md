# Get started with Flux v2

## Prerequisites

You will need two Kubernetes clusters version 1.16 or newer and kubectl version 1.18.
For a quick local test, you can use [Kubernetes kind](https://kind.sigs.k8s.io/docs/user/quick-start/).
Any other Kubernetes setup will work as well though.

In order to follow the guide you'll need a GitHub account and a
[personal access token](https://help.github.com/en/github/authenticating-to-github/creating-a-personal-access-token-for-the-command-line)
that can create repositories (check all permissions under `repo`).

Export your GitHub personal access token and username:

```sh
export GITHUB_TOKEN=<your-token>
export GITHUB_USER=<your-username>
```

## Install the Flux CLI

To install the latest `flux` release on MacOS and Linux using
[Homebrew](https://brew.sh/) run:

```sh
brew install fluxcd/tap/flux
```

Or install `flux` by downloading precompiled binaries using a Bash script:

```sh
curl -s https://toolkit.fluxcd.io/install.sh | sudo bash
```

The install script downloads the flux binary to `/usr/local/bin`.

Binaries for **macOS**, **Windows** and **Linux** AMD64/ARM are available for download on the
[release page](https://github.com/fluxcd/flux2/releases).

To configure your shell to load `flux` completions add to your Bash
profile:

```sh
# ~/.bashrc or ~/.bash_profile
. <(flux completion bash)
```

`zsh`, `fish`, and `powershell` are also supported with their own sub-commands.

## GitOps workflow

You'll be using a dedicated Git repository e.g. `fleet-infra` to manage one or more Kubernetes clusters.
This guide assumes that you have two clusters, one for staging and one for production.

Using the Flux CLI you'll do the following:

- configure each cluster to synchronise with a directory inside the fleet repository
- register app sources (git repositories) that contain plain Kubernetes manifests or Kustomize overlays
- configure app deployments on both clusters (pre-releases on staging, semver releases on production)

## Staging bootstrap

Create the staging cluster using Kubernetes kind or set the kubectl context to an existing cluster:

```sh
kind create cluster --name staging
kubectl cluster-info --context kind-staging
```

Verify that your staging cluster satisfies the prerequisites with:

```console
$ flux check --pre
► checking prerequisites
✔ kubectl 1.18.3 >=1.18.0
✔ kubernetes 1.18.2 >=1.16.0
✔ prerequisites checks passed
```

Run the bootstrap command:

```sh
flux bootstrap github \
  --owner=$GITHUB_USER \
  --repository=fleet-infra \
  --branch=main \
  --path=staging-cluster \
  --personal
```

!!! hint "ARM"
    When deploying to a Kubernetes cluster with ARM architecture,
    you can use `--arch=arm` for ARMv7 32-bit container images
    and `--arch=arm64` for ARMv8 64-bit container images.

The bootstrap command creates a repository if one doesn't exist, and
commits the manifests for the Flux components to the default branch at the specified path.
Then it configures the target cluster to synchronize with the specified path inside the repository.

If you wish to create the repository under a GitHub organization:

```sh
flux bootstrap github \
  --owner=<organization> \
  --repository=<repo-name> \
  --branch=<organization default branch> \
  --team=<team1-slug> \
  --team=<team2-slug> \
  --path=staging-cluster
```

Example output:

```text
$ flux bootstrap github --owner=gitopsrun --repository=fleet-infra --path=staging-cluster --team=devs
► connecting to github.com
✔ repository created
✔ devs team access granted
✔ repository cloned
✚ generating manifests
✔ components manifests pushed
► installing components in flux-system namespace
deployment "source-controller" successfully rolled out
deployment "kustomize-controller" successfully rolled out
deployment "notification-controller" successfully rolled out
✔ install completed
► configuring deploy key
✔ deploy key configured
► generating sync manifests
✔ sync manifests pushed
► applying sync manifests
◎ waiting for cluster sync
✔ bootstrap finished
```

If you prefer GitLab, export `GITLAB_TOKEN` env var and use the command [flux bootstrap gitlab](../cmd/flux_bootstrap_gitlab.md).

!!! hint "Idempotency"
    It is safe to run the bootstrap command as many times as you want.
    If the Flux components are present on the cluster,
    the bootstrap command will perform an upgrade if needed.
    You can target a specific Flux [version](https://github.com/fluxcd/flux2/releases)
    with `flux bootstrap --version=<semver>`.

## Staging workflow

Clone the repository with:

```sh
git clone https://github.com/$GITHUB_USER/fleet-infra
cd fleet-infra
```

Create a git source pointing to a public repository master branch:

```sh
flux create source git webapp \
  --url=https://github.com/stefanprodan/podinfo \
  --branch=master \
  --interval=30s \
  --export > ./staging-cluster/webapp-source.yaml
```

Create a kustomization for synchronizing the common manifests on the cluster:

```sh
flux create kustomization webapp-common \
  --source=webapp \
  --path="./deploy/webapp/common" \
  --prune=true \
  --validation=client \
  --interval=1h \
  --export > ./staging-cluster/webapp-common.yaml
```

Create a kustomization for the backend service that depends on common:

```sh
flux create kustomization webapp-backend \
  --depends-on=webapp-common \
  --source=webapp \
  --path="./deploy/webapp/backend" \
  --prune=true \
  --validation=client \
  --interval=10m \
  --health-check="Deployment/backend.webapp" \
  --health-check-timeout=2m \
  --export > ./staging-cluster/webapp-backend.yaml
```

Create a kustomization for the frontend service that depends on backend:

```sh
flux create kustomization webapp-frontend \
  --depends-on=webapp-backend \
  --source=webapp \
  --path="./deploy/webapp/frontend" \
  --prune=true \
  --validation=client \
  --interval=10m \
  --health-check="Deployment/frontend.webapp" \
  --health-check-timeout=2m \
  --export > ./staging-cluster/webapp-frontend.yaml
```

Push changes to origin:

```sh
git add -A && git commit -m "add staging webapp" && git push
```

In about 30s the synchronization should start:

```console
$ watch flux get kustomizations
NAME            REVISION                                        SUSPENDED       READY   MESSAGE
flux-system     main/6eea299fe9997c8561b826b67950afaf9a476cf8   False           True    Applied revision: main/6eea299fe9997c8561b826b67950afaf9a476cf8
webapp-backend                                                  False           False   dependency 'flux-system/webapp-common' is not ready
webapp-common   master/7411da595c25183daba255068814b83843fe3395 False           True    Applied revision: master/7411da595c25183daba255068814b83843fe3395
webapp-frontend                                                 False           False   dependency 'flux-system/webapp-backend' is not ready
```

When the synchronization finishes you can check that the webapp services are running:

```console
$ kubectl -n webapp get deployments,services
NAME                       READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/backend    1/1     1            1           4m1s
deployment.apps/frontend   1/1     1            1           3m31s

NAME               TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)             AGE
service/backend    ClusterIP   10.52.10.22   <none>        9898/TCP,9999/TCP   4m1s
service/frontend   ClusterIP   10.52.9.85    <none>        80/TCP              3m31s
```

!!! tip
    From this moment forward, any changes made to the webapp
    Kubernetes manifests in the master branch will be synchronised with the staging cluster.

If a Kubernetes manifest is removed from the webapp repository, the reconciler will remove it from your cluster.
If you delete a kustomization from the `fleet-infra` repo, the reconciler will remove all Kubernetes objects that
were previously applied from that kustomization.

If you alter the webapp deployment using `kubectl edit`, the changes will be reverted to match
the state described in git. When dealing with an incident, you can pause the reconciliation of a
kustomization with `flux suspend kustomization <name>`. Once the debugging session
is over, you can re-enable the reconciliation with `flux resume kustomization <name>`.

## Production bootstrap 

On production clusters, you may wish to deploy stable releases of an application.
When creating a git source instead of a branch, you can specify a git tag or a semver expression. 

Create the production cluster using Kubernetes kind or set the kubectl context to an existing cluster:

```sh
kind create cluster --name production
kubectl cluster-info --context kind-production
```

Run the bootstrap for the production environment:

```sh
flux bootstrap github \
  --owner=$GITHUB_USER \
  --repository=fleet-infra \
  --path=prod-cluster \
  --personal
```
Pull the changes locally:

```sh
git pull
```

## Production workflow

Create a git source using a semver range to target stable releases:

```sh
flux create source git webapp \
  --url=https://github.com/stefanprodan/podinfo \
  --tag-semver=">=4.0.0 <4.0.2" \
  --interval=30s \
  --export > ./prod-cluster/webapp-source.yaml
```

Create a kustomization for webapp pointing to the production overlay:

```sh
flux create kustomization webapp \
  --source=webapp \
  --path="./deploy/overlays/production" \
  --prune=true \
  --validation=client \
  --interval=10m \
  --health-check="Deployment/frontend.production" \
  --health-check="Deployment/backend.production" \
  --health-check-timeout=2m \
  --export > ./prod-cluster/webapp-production.yaml
``` 

Push changes to origin:

```sh
git add -A && git commit -m "add prod webapp" && git push
```

List git sources:

```console
$ flux get sources git
NAME            REVISION                                        READY   MESSAGE
flux-system     main/5ae055e24b2c8a78f981708b61507a97a30bd7a6   True    Fetched revision: main/113360052b3153e439a0cf8de76b8e3d2a7bdf27
webapp          4.0.1/113360052b3153e439a0cf8de76b8e3d2a7bdf27  True    Fetched revision: 4.0.1/113360052b3153e439a0cf8de76b8e3d2a7bdf27
```

The kubectl equivalent is `kubectl -n flux-system get gitrepositories`.

List kustomization:

```console
$ flux get kustomizations
NAME            REVISION                                        SUSPENDED       READY   MESSAGE
flux-system     main/5ae055e24b2c8a78f981708b61507a97a30bd7a6   False           True    Applied revision: main/5ae055e24b2c8a78f981708b61507a97a30bd7a6
webapp          4.0.1/113360052b3153e439a0cf8de76b8e3d2a7bdf27  False           True    Applied revision: 4.0.1/113360052b3153e439a0cf8de76b8e3d2a7bdf27
```

The kubectl equivalent is `kubectl -n flux-system get kustomizations`.

If you want to upgrade to the latest 4.x version, you can change the semver expression to:

```sh
flux create source git webapp \
  --url=https://github.com/stefanprodan/podinfo \
  --tag-semver=">=4.0.0 <5.0.0" \
  --interval=30s \
  --export > ./prod-cluster/webapp-source.yaml

git add -A && git commit -m "update prod webapp" && git push
```

Trigger a git sync:

```console
$ flux reconcile ks flux-system --with-source 
► annotating source flux-system
✔ source annotated
◎ waiting for reconcilitation
✔ git reconciliation completed
✔ fetched revision main/d751ea264d48bf0db8b588d1d08184834ac8fec9
◎ waiting for kustomization reconcilitation
✔ kustomization reconcilitation completed
✔ applied revision main/d751ea264d48bf0db8b588d1d08184834ac8fec9
```

The kubectl equivalent is `kubectl -n flux-system annotate gitrepository/flux-system fluxcd.io/reconcileAt="$(date +%s)"`.

Wait for the webapp to be upgraded:

```console
$ watch flux get kustomizations
NAME            REVISION                                        SUSPENDED       READY   MESSAGE
flux-system     main/d751ea264d48bf0db8b588d1d08184834ac8fec9   False           True    Applied revision: main/d751ea264d48bf0db8b588d1d08184834ac8fec9
webapp          4.0.6/26a630c0b4b3452833d96c511d93f6f2d2e90a99  False           True    Applied revision: 4.0.6/26a630c0b4b3452833d96c511d93f6f2d2e90a99
```
