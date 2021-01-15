# Get started with Flux v2

!!! note "Basic knowledge"
    This guide assumes you have some understanding of the core concepts and have read the introduction to Flux.
    The core concepts used in this guide are [GitOps](../core-concepts/index.md#gitops),
    [Sources](../core-concepts/index.md#sources), [Kustomization](../core-concepts/index.md#kustomization).

In this tutorial, you will deploy an application to a kubernetes cluster with Flux
and manage the cluster in a complete GitOps manner.
You'll be using a dedicated Git repository e.g. `fleet-infra` to manage your Kubernetes clusters.

## Prerequisites

In order to follow the guide, you will need a Kubernetes cluster version 1.16 or newer and kubectl version 1.18.
For a quick local test, you can use [Kubernetes kind](https://kind.sigs.k8s.io/docs/user/quick-start/).
Any other Kubernetes setup will work as well though.

Flux is installed in a GitOps way and its manifest will be pushed to the repository,
so you will also need a GitHub account and a
[personal access token](https://help.github.com/en/github/authenticating-to-github/creating-a-personal-access-token-for-the-command-line)
that can create repositories (check all permissions under `repo`) to enable Flux do this.

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

If using Arch Linux, install the latest stable version from **AUR** using
either [flux-bin](https://aur.archlinux.org/packages/flux-bin) (pre-built
binary) or [flux-go](https://aur.archlinux.org/packages/flux-go) (locally built
binary).

Binaries for **macOS**, **Windows** and **Linux** AMD64/ARM are available for download on the
[release page](https://github.com/fluxcd/flux2/releases).

To configure your shell to load `flux` [bash completions](../cmd/flux_completion_bash.md) add to your profile:

```sh
# ~/.bashrc or ~/.bash_profile
. <(flux completion bash)
```

[`zsh`](../cmd/flux_completion_zsh.md), [`fish`](../cmd/flux_completion_fish.md), and [`powershell`](../cmd/flux_completion_powershell.md) are also supported with their own sub-commands.

## Install Flux components

Create the cluster using Kubernetes kind or set the kubectl context to an existing cluster:

```sh
kind create cluster
kubectl cluster-info
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
  --path=./clusters/my-cluster \
  --personal
```

!!! hint "Multi-arch images"
    The component images are published as [multi-arch container images](https://docs.docker.com/docker-for-mac/multi-arch/)
    with support for Linux `amd64`, `arm64` and `armv7` (e.g. 32bit Raspberry Pi)
    architectures.

The bootstrap command creates a repository if one doesn't exist,
commits the manifests for the Flux components to the default branch at the specified path,
and installs the Flux components.
Then it configures the target cluster to synchronize with the specified path inside the repository.

If you wish to create the repository under a GitHub organization:

```sh
flux bootstrap github \
  --owner=<organization> \
  --repository=<repo-name> \
  --branch=<organization default branch> \
  --team=<team1-slug> \
  --team=<team2-slug> \
  --path=./clusters/my-cluster
```

Example output:

```console
$ flux bootstrap github --owner=gitopsrun --team=devs --repository=fleet-infra --path=./clusters/my-cluster 
► connecting to github.com
✔ repository created
✔ devs team access granted
✔ repository cloned
✚ generating manifests
✔ components manifests pushed
► installing components in flux-system namespace
deployment "source-controller" successfully rolled out
deployment "kustomize-controller" successfully rolled out
deployment "helm-controller" successfully rolled out
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

If you prefer GitLab, export `GITLAB_TOKEN` env var and
use the command [flux bootstrap gitlab](../guides/installation.md#gitlab-and-gitlab-enterprise).

!!! hint "Idempotency"
    It is safe to run the bootstrap command as many times as you want.
    If the Flux components are present on the cluster,
    the bootstrap command will perform an upgrade if needed.
    You can target a specific Flux [version](https://github.com/fluxcd/flux2/releases)
    with `flux bootstrap --version=<semver>`.

## Clone the git repository

We are going to drive app deployments in a GitOps manner,
using the Git repository as the desired state for our cluster.
Instead of applying the manifests directly to the cluster,
Flux will apply it for us instead.

Therefore, we need to clone the repository to our local machine:

```sh
git clone https://github.com/$GITHUB_USER/fleet-infra
cd fleet-infra
```

## Add podinfo repository to Flux

We will be using a public repository [github.com/stefanprodan/podinfo](https://github.com/stefanprodan/podinfo),
podinfo is a tiny web application made with Go.

Create a [GitRepository](../components/source/gitrepositories/)
manifest pointing to podinfo repository's master branch:

```sh
flux create source git podinfo \
  --url=https://github.com/stefanprodan/podinfo \
  --branch=master \
  --interval=30s \
  --export > ./clusters/my-cluster/podinfo-source.yaml
```

The above command generates the following manifest:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: GitRepository
metadata:
  name: podinfo
  namespace: flux-system
spec:
  interval: 30s
  ref:
    branch: master
  url: https://github.com/stefanprodan/podinfo
```

Commit and push it to the `fleet-infra` repository:

```sh
git add -A && git commit -m "Add podinfo GitRepository"
git push
```

## Deploy podinfo application

We will create a Flux [Kustomization](../components/kustomize/kustomization/) manifest for podinfo.
This configures Flux to build and apply the [kustomize](https://github.com/stefanprodan/podinfo/tree/master/kustomize)
directory located in the podinfo repository.

```sh
flux create kustomization podinfo \
  --source=podinfo \
  --path="./kustomize" \
  --prune=true \
  --validation=client \
  --interval=5m \
  --export > ./clusters/my-cluster/podinfo-kustomization.yaml
```

The above command generates the following manifest:

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1beta1
kind: Kustomization
metadata:
  name: podinfo
  namespace: flux-system
spec:
  interval: 5m0s
  path: ./kustomize
  prune: true
  sourceRef:
    kind: GitRepository
    name: podinfo
  validation: client
```

Commit and push the `Kustomization` manifest to the repository:

```sh
git add -A && git commit -m "Add podinfo Kustomization"
git push
```

The structure of your repository should look like this:

```
fleet-infra
└── clusters/
    └── my-cluster/
        ├── flux-system/                        
        │   ├── gotk-components.yaml
        │   ├── gotk-sync.yaml
        │   └── kustomization.yaml
        ├── podinfo-kustomization.yaml
        └── podinfo-source.yaml
```

## Watch Flux sync the application

In about 30s the synchronization should start:

```console
$ watch flux get kustomizations
NAME            READY   MESSAGE
flux-system     True    Applied revision: main/fc07af652d3168be329539b30a4c3943a7d12dd8
podinfo         True    Applied revision: master/855f7724be13f6146f61a893851522837ad5b634
```

When the synchronization finishes you can check that podinfo has been deployed on your cluster:

```console
$ kubectl -n default get deployments,services
NAME                      READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/podinfo   2/2     2            2           108s

NAME                 TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)             AGE
service/podinfo      ClusterIP   10.100.149.126   <none>        9898/TCP,9999/TCP   108s
```

!!! tip
    From this moment forward, any changes made to the podinfo
    Kubernetes manifests in the master branch will be synchronised with your cluster.

If a Kubernetes manifest is removed from the podinfo repository, Flux will remove it from your cluster.
If you delete a `Kustomization` from the fleet-infra repository, Flux will remove all Kubernetes objects that
were previously applied from that `Kustomization`.

If you alter the podinfo deployment using `kubectl edit`, the changes will be reverted to match
the state described in Git. When dealing with an incident, you can pause the reconciliation of a
kustomization with `flux suspend kustomization <name>`. Once the debugging session
is over, you can re-enable the reconciliation with `flux resume kustomization <name>`.

## Multi-cluster Setup

To use Flux to manage more than one cluster or promote deployments from staging to production, take a look at the 
two approaches in the repositories listed below.

1. [https://github.com/fluxcd/flux2-kustomize-helm-example](https://github.com/fluxcd/flux2-kustomize-helm-example)
2. [https://github.com/fluxcd/flux2-multi-tenancy](https://github.com/fluxcd/flux2-multi-tenancy)