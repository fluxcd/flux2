# Get started with Flux v2

!!! note "Basic knowledge"
    This guide assumes you have some understanding of the core concepts and have read the introduction to Flux.
    The core concepts used in this guide are [GitOps](../core-concepts/index.md#gitops), [Sources](../core-concepts/index.md#sources), [Kustomization](../core-concepts/index.md#kustomization).

In this tutorial, you will deploy an application to a kubernetes cluster with Flux and manage the cluster in a complete GitOps manner. You'll be using a dedicated Git repository e.g. `fleet-infra` to manage the Kubernetes clusters. All the manifest will be pushed to this repository and then applied by Flux.

## Prerequisites

In order to follow the guide, you will need one Kubernetes cluster version 1.16 or newer and kubectl version 1.18.
For a quick local test, you can use [Kubernetes kind](https://kind.sigs.k8s.io/docs/user/quick-start/).
Any other Kubernetes setup will work as well though.

Flux is installed in a complete GitOps way and its manifest will be pushed to the repository, so you will also need a GitHub account and a
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

To configure your shell to load `flux` completions add to your Bash
profile:

```sh
# ~/.bashrc or ~/.bash_profile
. <(flux completion bash)
```

`zsh`, `fish`, and `powershell` are also supported with their own sub-commands.

## Install Flux components

Create the cluster using Kubernetes kind or set the kubectl context to an existing cluster:

```sh
kind create cluster
kubectl cluster-info --context kind-kind
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

!!! hint "ARM"
    When deploying to a Kubernetes cluster with ARM architecture,
    you can use `--arch=arm` for ARMv7 32-bit container images
    and `--arch=arm64` for ARMv8 64-bit container images.

The bootstrap command creates a repository if one doesn't exist,
commits the manifests for the Flux components to the default branch at the specified path, and installs the Flux components.
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

If you prefer GitLab, export `GITLAB_TOKEN` env var and use the command [flux bootstrap gitlab](../cmd/flux_bootstrap_gitlab.md).

!!! hint "Idempotency"
    It is safe to run the bootstrap command as many times as you want.
    If the Flux components are present on the cluster,
    the bootstrap command will perform an upgrade if needed.
    You can target a specific Flux [version](https://github.com/fluxcd/flux2/releases)
    with `flux bootstrap --version=<semver>`.

## Clone the git repository

We are going to be managing the application in a GitOps manner with the git repository. The Flux manifests generated by the CLI will be pushed to the git repository. Instead of applying the manifests directly to the cluster, Flux will apply it for us instead :).

Therefore, we need to clone the repository to our local machine.

```sh
git clone https://github.com/$GITHUB_USER/fleet-infra
cd fleet-infra
```


## Add podinfo repository to Flux

We will be using a public repository [github.com/stefanprodan/podinfo](https://github.com/stefanprodan/podinfo), podinfo is a tiny web application made with Go.

Create a GitRepository manifest pointing to the repository's master branch with Flux CLI.

```sh
flux create source git podinfo \
  --url=https://github.com/stefanprodan/podinfo \
  --branch=master \
  --interval=30s \
  --export > ./clusters/my-cluster/podinfo-source.yaml
```

Commit and push it to the `fleet-infra` repository, then Flux applies it to the cluster.

```sh
git add -A && git commit -m "Adds podinfo git source"
git push
```

## Deploy podinfo application

We will create a kustomization manifest for podinfo. This will apply the manifest in the `kustomize` directory in the podinfo repository.

```sh
flux create kustomization podinfo \
  --source=podinfo \
  --path="./kustomize" \
  --prune=true \
  --validation=client \
  --interval=5m \
  --export > ./clusters/my-cluster/podinfo-kustomization.yaml
```

Commit and push the kustomization manifest to the git repository so that Flux applies it to the cluster.

```sh
git add -A && git commit -m "Adds podinfo kustomization"
git push
```

The structure of your repository should look like this:
```
fleet-infra
└── clusters/
    └── my-cluster/
        ├── flux-system/                        
        │   ├── gotk-components.yaml/
        │   ├── gotk-sync.yaml/
        │   └── kustomization.yaml/
        ├── podinfo-kustomization.yaml
        └── podinfo-source.yaml
```


## Watch Flux sync the application

In about 30s the synchronization should start:

```console
$ watch flux get kustomizations
NAME            READY   MESSAGE
flux-system     main/fc07af652d3168be329539b30a4c3943a7d12dd8   False           True    Applied revision: main/fc07af652d3168be329539b30a4c3943a7d12dd8
podinfo         master/855f7724be13f6146f61a893851522837ad5b634 False           True    Applied revision: master/855f7724be13f6146f61a893851522837ad5b634
```

When the synchronization finishes you can check that the podinfo has been deployed on your cluster:

```console
$ kubectl -n default get deployments,services
NAME                      READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/podinfo   2/2     2            2           108s

NAME                 TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)             AGE
service/podinfo      ClusterIP   10.100.149.126   <none>        9898/TCP,9999/TCP   108s
```

!!! tip
    From this moment forward, any changes made to the webapp
    Kubernetes manifests in the master branch will be synchronised with your cluster.

If a Kubernetes manifest is removed from the webapp repository, the reconciler will remove it from your cluster.
If you delete a kustomization from the cluster, the reconciler will remove all Kubernetes objects that
were previously applied from that kustomization.

If you delete a kustomization from the `fleet-infra` repo, the reconciler will remove all Kubernetes objects that
were previously applied from that kustomization.

If you alter the webapp deployment using `kubectl edit`, the changes will be reverted to match
the state described in git. When dealing with an incident, you can pause the reconciliation of a
kustomization with `flux suspend kustomization <name>`. Once the debugging session
is over, you can re-enable the reconciliation with `flux resume kustomization <name>`.

## Multi-cluster Setup

To use Flux to manage more than one cluster or promote deployments from staging to production, take a look at the 
two approaches in the repositories listed below.

1. [https://github.com/fluxcd/flux2-kustomize-helm-example](https://github.com/fluxcd/flux2-kustomize-helm-example)
2. [https://github.com/fluxcd/flux2-multi-tenancy](https://github.com/fluxcd/flux2-multi-tenancy)