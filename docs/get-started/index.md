# Get started with GitOps Toolkit

## Prerequisites

* Kubernetes >= 1.14
* kubectl >= 1.18
* git

You will need to have Kubernetes set up.
For a quick local test, you can use `minikube`, `kubeadm` or `kind`.
Any other Kubernetes setup will work as well though.

In order to follow the guide you'll need a GitHub account and a 
[personal access token](https://help.github.com/en/github/authenticating-to-github/creating-a-personal-access-token-for-the-command-line)
that can create repositories.

## Install the toolkit CLI

To install the latest `tk` release run:

```bash
curl -s https:/fluxcd.github.io/toolkit/install.sh | sudo bash
```

The install script downloads the tk binary to `/usr/local/bin`.

Binaries for macOS and Linux AMD64 are available for download on the 
[release page](https://github.com/fluxcd/toolkit/releases).

To configure your shell to load tk completions add to your bash profile:

```sh
# ~/.bashrc or ~/.bash_profile
. <(tk completion)
```

## Bootstrap 

Export your GitHub personal access token with:

```sh
export GITHUB_TOKEN=<your-token>
```

The bootstrap command creates a GitHub repository if one doesn't exist and
commits the toolkit components manifests to the master branch.
Then it configures the target cluster to synchronize with the repository.
If the toolkit components are present on the cluster,
the bootstrap command will perform an upgrade if needed.

```sh
tk bootstrap github \
  --owner=<your-github-username> \
  --repository=<repo-name> \
  --path=dev-cluster \
  --personal
```

If you wish to create the repository under a GitHub organization:

```sh
tk bootstrap github \
  --owner=<organization> \
  --repository=<repo-name> \
  --team=<team1-slug> \
  --team=<team2-slug> \
  --path=dev-cluster
```

Example output:

```text
$ tk bootstrap github --owner=gitopsrun --repository=fleet-infra --path=dev-cluster --team=devs

► connecting to github.com
✔ repository created
✔ devs team access granted
✔ repository cloned
✚ generating manifests
✔ components manifests pushed
► installing components in gitops-system namespace
namespace/gitops-system created
customresourcedefinition.apiextensions.k8s.io/gitrepositories.source.fluxcd.io created
customresourcedefinition.apiextensions.k8s.io/helmcharts.source.fluxcd.io created
customresourcedefinition.apiextensions.k8s.io/helmrepositories.source.fluxcd.io created
customresourcedefinition.apiextensions.k8s.io/kustomizations.kustomize.fluxcd.io created
customresourcedefinition.apiextensions.k8s.io/profiles.kustomize.fluxcd.io created
role.rbac.authorization.k8s.io/crd-controller-gitops-system created
rolebinding.rbac.authorization.k8s.io/crd-controller-gitops-system created
clusterrolebinding.rbac.authorization.k8s.io/cluster-reconciler-gitops-system created
service/source-controller created
deployment.apps/kustomize-controller created
deployment.apps/source-controller created
networkpolicy.networking.k8s.io/deny-ingress created
Waiting for deployment "source-controller" rollout to finish: 0 of 1 updated replicas are available...
deployment "source-controller" successfully rolled out
deployment "kustomize-controller" successfully rolled out
✔ install completed
► configuring deploy key
✔ deploy key configured
► generating sync manifests
✔ sync manifests pushed
► applying sync manifests
◎ waiting for cluster sync
✔ bootstrap finished
```

If you prefer GitLab, export `GITLAB_TOKEN` env var and use the command `tk bootstrap gitlab`.

## Create a GitOps workflow

Clone the repository with:

```sh
git clone https://github.com/gitopsrun/fleet-infra
cd fleet-infra
```

Create a git source pointing to a public repository:

```sh
tk create source git webapp \
  --url=https://github.com/stefanprodan/podinfo \
  --branch=master \
  --interval=30s \
  --export > ./dev-cluster/webapp-source.yaml
```

Create a kustomization for synchronizing the common manifests on the cluster:

```sh
tk create kustomization webapp-common \
  --source=webapp \
  --path="./deploy/webapp/common" \
  --prune=true \
  --validate=client \
  --interval=1h \
  --export > ./dev-cluster/webapp-common.yaml
```

Create a kustomization for the backend service that depends on common: 

```sh
tk create kustomization webapp-backend \
  --depends-on=webapp-common \
  --source=webapp \
  --path="./deploy/webapp/backend" \
  --prune=true \
  --validate=client \
  --interval=10m \
  --health-check="Deployment/backend.webapp" \
  --health-check-timeout=2m \
  --export > ./dev-cluster/webapp-backend.yaml
```

Create a kustomization for the frontend service that depends on backend: 

```sh
tk create kustomization webapp-frontend \
  --depends-on=webapp-backend \
  --source=webapp \
  --path="./deploy/webapp/frontend" \
  --prune=true \
  --validate=client \
  --interval=10m \
  --health-check="Deployment/frontend.webapp" \
  --health-check-timeout=2m \
  --export > ./dev-cluster/webapp-frontend.yaml
```

Push changes to origin:

```sh
git add -A && git commit -m "add webapp" && git push
```

In about 30s the synchronization should start:

```text
$ watch tk get kustomizations

✔ gitops-system last applied revision master/35d5765a1acb9e9ce66cad7274c6fe03eee1e8eb
✔ webapp-backend reconciling
✔ webapp-common last applied revision master/f43f9b2eb6766e07f318d266a99d2ec7c940b0cf
✗ webapp-frontend dependency 'gitops-system/webapp-backend' is not ready
```

When the synchronization finishes you can check that the webapp services are running:

```text
$ kubectl -n webapp get deployments,services

NAME                       READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/backend    1/1     1            1           4m1s
deployment.apps/frontend   1/1     1            1           3m31s

NAME               TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)             AGE
service/backend    ClusterIP   10.52.10.22   <none>        9898/TCP,9999/TCP   4m1s
service/frontend   ClusterIP   10.52.9.85    <none>        80/TCP              3m31s
```

