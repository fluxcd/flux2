# Migrate from Flux v1 to v2

This guide walks you through migrating from Flux v1 to v2.
Read the [FAQ](faq-migration.md) to find out what differences are between v1 and v2.

!!! info "Automated image updates"
    The image automation feature is under development in Flux v2.
    Please consult the [roadmap](../roadmap/index.md) for more details.

!!! info "Feature parity"
    "Feature parity" does not mean Flux v2 works exactly the same as v1 (or is
    backward-compatible); it means you can accomplish the same results, while
    accounting for the fact that it's a system with a substantially different
    design.
    This may at times mean that you have to make adjustments to the way your
    current cluster configuration is structured. If you are in this situation
    and need help, please refer to the [support page](https://fluxcd.io/support/).

## Prerequisites

You will need a Kubernetes cluster version **1.16** or newer
and kubectl version **1.18** or newer.

### Install Flux v2 CLI

With Homebrew:

```sh
brew install fluxcd/tap/flux
```

With Bash:

```sh
curl -s https://fluxcd.io/install.sh | sudo bash

# enable completions in ~/.bash_profile
. <(flux completion bash)
```

Command-line completion for `zsh`, `fish`, and `powershell`
are also supported with their own sub-commands.

Binaries for macOS, Windows and Linux AMD64/ARM are available for download on the
[release page](https://github.com/fluxcd/flux2/releases).

Verify that your cluster satisfies the prerequisites with:

```sh
flux check --pre
```

## GitOps migration

Flux v2 offers an installation procedure that is declarative first
and disaster resilient.

Using the `flux bootstrap` command you can install Flux on a
Kubernetes cluster and configure it to manage itself from a Git
repository. The Git repository created during bootstrap can be used 
to define the state of your fleet of Kubernetes clusters.

For a detailed walk-through of the bootstrap procedure please see the [installation guide](installation.md).

!!! warning "`flux bootstrap` target"
    `flux bootstrap` should not be run against a Git branch or path
    that is already being synchronized by Flux v1, as this will make
    them fight over the resources. Instead, bootstrap to a **new Git
    repository, branch or path**, and continue with moving the
    manifests.

After you've installed Flux v2 on your cluster using bootstrap,
you can delete the Flux v1 from your clusters and move the manifests from the
Flux v1 repository to the bootstrap one.

## In-place migration

!!! warning
    For production use we recommend using the **bootstrap** procedure (see the [Gitops migration](#gitops-migration) section above),
    but if you wish to install Flux v2 in the 
    same way as Flux v1 then follow along.

### Flux read-only mode

Assuming you've installed Flux v1 to sync a directory with plain YAMLs from a private Git repo:

```sh
# create namespace
kubectl create ns flux

# deploy Flux v1
fluxctl install \
--git-url=git@github.com:org/app \
--git-branch=main \
--git-path=./deploy \
--git-readonly \
--namespace=flux | kubectl apply -f -

# print deploy key
fluxctl identity --k8s-fwd-ns flux

# trigger sync
fluxctl sync --k8s-fwd-ns flux
```

!!! hint "Uninstall Flux v1"
    Before you proceed, scale the Flux v1 deployment to zero
    or delete its namespace and RBAC.

If there are YAML files in your `deploy` dir that are not meant to be
applied on the cluster, you can exclude them by placing a `.sourceignore` in your repo root:

```console
$ cat .sourceignore
# exclude all
/*
# include deploy dir
!/deploy
# exclude files from deploy dir
/deploy/**/eksctl.yaml
/deploy/**/charts
```

Install Flux v2 in the `flux-system` namespace:

```console
$ flux install \
  --network-policy=true \
  --watch-all-namespaces=true \
  --namespace=flux-system
✚ generating manifests
✔ manifests build completed
► installing components in flux-system namespace
✔ install completed
◎ verifying installation
✔ source-controller ready
✔ kustomize-controller ready
✔ helm-controller ready
✔ notification-controller ready
✔ install finished
```

Register your Git repository and add the deploy key with read-only access:

```console
$ flux create source git app \
  --url=ssh://git@github.com/org/app \
  --branch=main \
  --interval=1m
► generating deploy key pair
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCp2x9ghVmv1zD...
Have you added the deploy key to your repository: y
► collecting preferred public key from SSH server
✔ collected public key from SSH server:
github.com ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEAq2A...
► applying secret with keys
✔ authentication configured
✚ generating GitRepository source
► applying GitRepository source
✔ GitRepository source created
◎ waiting for GitRepository source reconciliation
✔ GitRepository source reconciliation completed
✔ fetched revision: main/5302d04c2ab8f0579500747efa0fe7abc72c8f9b
```

Configure the reconciliation of the `deploy` dir on your cluster:

```console
$ flux create kustomization app \
  --source=app \
  --path="./deploy" \
  --prune=true \
  --interval=10m
✚ generating Kustomization
► applying Kustomization
✔ Kustomization created
◎ waiting for Kustomization reconciliation
✔ Kustomization app is ready
✔ applied revision main/5302d04c2ab8f0579500747efa0fe7abc72c8f9b
```

If your repository contains secrets encrypted with Mozilla SOPS, please read this [guide](mozilla-sops.md).

Pull changes from Git and apply them immediately:

```sh
flux reconcile kustomization app --with-source 
``` 

List all Kubernetes objects reconciled by `app`:

```sh
kubectl get all --all-namespaces \
-l=kustomize.toolkit.fluxcd.io/name=app \
-l=kustomize.toolkit.fluxcd.io/namespace=flux-system
```

### Flux with Kustomize

Assuming you've installed Flux v1 to sync a Kustomize overlay from an HTTPS Git repository:

```sh
fluxctl install \
--git-url=https://github.com/org/app \
--git-branch=main \
--manifest-generation \
--namespace=flux | kubectl apply -f -
```

With the following `.flux.yaml` in the root dir:

```yaml
version: 1
patchUpdated:
  generators:
    - command: kustomize build ./overlays/prod
  patchFile: flux-patch.yaml
```

!!! hint "Uninstall Flux v1"
    Before you proceed, delete the Flux v1 namespace
    and remove the `.flux.yaml` from your repo.

Install Flux v2 in the `flux-system` namespace:

```sh
flux install
```

Register the Git repository using a personal access token:

```sh
flux create source git app \
  --url=https://github.com/org/app \
  --branch=main \
  --username=git \
  --password=token \
  --interval=1m
```

Configure the reconciliation of the `prod` overlay on your cluster:

```sh
flux create kustomization app \
  --source=GitRepository/app \
  --path="./overlays/prod" \
  --prune=true \
  --interval=10m
```

Check the status of the Kustomization reconciliation:

```console
$ flux get kustomizations app
NAME	REVISION                                     	SUSPENDED	READY
app 	main/5302d04c2ab8f0579500747efa0fe7abc72c8f9b	False    	True
```

### Flux with Slack notifications

Assuming you've configured Flux v1 to send notifications to Slack with FluxCloud.

With Flux v2, create an alert provider for a Slack channel:

```sh
flux create alert-provider slack \
  --type=slack \
  --channel=general \
  --address=https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK
```

And configure notifications for the `app` reconciliation events:

```sh
flux create alert app \
  --provider-ref=slack \
  --event-severity=info \
  --event-source=GitRepository/app \
  --event-source=Kustomization/app
```

For more details, read the guides on how to configure
[notifications](notifications.md) and [webhooks](webhook-receivers.md).

### Flux debugging

Check the status of Git operations:

```console
$ kubectl -n flux-system get gitrepositories
NAME	READY	MESSAGE                                                                                                                                                                                        
app 	True 	Fetched revision: main/5302d04c2ab8f0579500747efa0fe7abc72c8f9b                                                                                                                               	
test	False	SSH handshake failed: unable to authenticate, attempted methods [none publickey]
```

Check the status of the cluster reconciliation with kubectl:

```console
$ kubectl -n flux-system get kustomizations
NAME   READY   STATUS
app    True    Applied revision: main/5302d04c2ab8f0579500747efa0fe7abc72c8f9
test   False   The Service 'backend' is invalid: spec.type: Unsupported value: 'Ingress'
```

Suspend a reconciliation:

```console
$ flux suspend kustomization app
► suspending kustomization app in flux-system namespace
✔ kustomization suspended
```

Check the status with kubectl:

```console
$ kubectl -n flux-system get kustomization app
NAME   READY   STATUS
app    False   Kustomization is suspended, skipping reconciliation
```

Resume a reconciliation:

```console
$ flux resume kustomization app
► resuming Kustomization app in flux-system namespace
✔ Kustomization resumed
◎ waiting for Kustomization reconciliation
✔ Kustomization reconciliation completed
✔ applied revision main/5302d04c2ab8f0579500747efa0fe7abc72c8f9b
```
