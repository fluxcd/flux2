# Using Flux on Azure

## AKS Cluster Options

It's important to follow some guidelines when installing Flux on AKS.

### CNI and Network Policy

Previously, there has been an issue with Flux and Network Policy on AKS.
([Upstream Azure Issue](https://github.com/Azure/AKS/issues/2031)) ([Flux Issue](https://github.com/fluxcd/flux2/issues/703))
If you ensure your AKS cluster is upgraded, and your Nodes have been restarted with the most recent Node images,
this could resolve flux reconciliation failures where source-controller is unreachable.
Using `--network-plugin=azure --network-policy=calico` has been tested to work properly.
This issue only affects you if you are using `--network-policy` on AKS, which is not a default option.

!!! warning
    AKS `--network-policy` is currently in Preview

### AAD Pod-Identity

Depending on the features you are interested in using with Flux, you may want to install AAD Pod Identity.
With [AAD Pod-Identity](https://azure.github.io/aad-pod-identity/docs/), we can create Pods that have their own
cloud credentials for accessing Azure services like Azure Container Registry(ACR) and Azure Key Vault(AKV).

If you do not use AAD Pod-Identity, you'll need to manage and store Service Principal credentials
in K8s Secrets, to integrate Flux with other Azure Services.

As a pre-requisite, your cluster must have `--enable-managed-identity` configured.

This software can be [installed via Helm](https://azure.github.io/aad-pod-identity/docs/getting-started/installation/)
(unmanaged by Azure).
Use Flux's `HelmRepository` and `HelmRelease` object to manage the aad-pod-identity installation
from a bootstrap repository and keep it up to date.

!!! note
    As an alternative to Helm, the `--enable-aad-pod-identity` flag for the `az aks create` is currently in Preview.
    Follow the Azure guide for [Creating an AKS cluster with AAD Pod Identity](https://docs.microsoft.com/en-us/azure/aks/use-azure-ad-pod-identity)
    if you would like to enable this feature with the Azure CLI.

### Cluster Creation

The following creates an AKS cluster with some minimal configuration that will work well with Flux:

```sh
az aks create \
 --network-plugin="azure" \
 --network-policy="calico" \
 --enable-managed-identity \
 --enable-pod-identity \
 --name="my-cluster"
```

!!! info
    When working with the Azure CLI, it can help to set a default `location`, `group`, and `acr`.
    See `az configure --help`, `az configure --list-defaults`, and `az configure --defaults key=value`.

## Flux Installation for Azure DevOps

Ensure you can login to [dev.azure.com](https://dev.azure.com) for your proper organization,
and create a new repository to hold your Flux install and other Kubernetes resources.

Clone the Git repository locally:

```sh
git clone ssh://git@ssh.dev.azure.com/v3/<org>/<project>/<my-repository>
cd my-repository
```

Create a directory inside the repository:

```sh
mkdir -p ./clusters/my-cluster/flux-system
```

Download the [Flux CLI](../guides/installation.md#install-the-flux-cli) and generate the manifests with:

```sh
flux install \
  --export > ./clusters/my-cluster/flux-system/gotk-components.yaml
```

Commit and push the manifest to the master branch:

```sh
git add -A && git commit -m "add components" && git push
```

Apply the manifests on your cluster:

```sh
kubectl apply -f ./clusters/my-cluster/flux-system/gotk-components.yaml
```

Verify that the controllers have started:

```sh
flux check
```

Create a `GitRepository` object on your cluster by specifying the SSH address of your repo:

```sh
flux create source git flux-system \
  --git-implementation=libgit2 \
  --url=ssh://git@ssh.dev.azure.com/v3/<org>/<project>/<repository> \
  --branch=<branch> \
  --ssh-key-algorithm=rsa \
  --ssh-rsa-bits=4096 \
  --interval=1m
```

The above command will prompt you to add a deploy key to your repository, but Azure DevOps
[does not support repository or org-specific deploy keys](https://developercommunity.visualstudio.com/t/allow-the-creation-of-ssh-deploy-keys-for-vsts-hos/365747).
You may add the deploy key to a user's personal SSH keys, but take note that
revoking the user's access to the repository will also revoke Flux's access.
The better alternative is to create a machine-user whose sole purpose is
to store credentials for automation.
Using a machine-user also has the benefit of being able to be read-only or
restricted to specific repositories if this is needed.

!!! note
    Unlike `git`, Flux does not support the
    ["shorter" scp-like syntax for the SSH protocol](https://git-scm.com/book/en/v2/Git-on-the-Server-The-Protocols#_the_ssh_protocol)
    (e.g. `ssh.dev.azure.com:v3`).
    Use the [RFC 3986 compatible syntax](https://tools.ietf.org/html/rfc3986#section-3) instead: `ssh.dev.azure.com/v3`.

If you wish to use Git over HTTPS, then generate a personal access token and supply it as the password:

```sh
flux create source git flux-system \
  --git-implementation=libgit2 \
  --url=https://dev.azure.com/<org>/<project>/_git/<repository> \
  --branch=main \
  --username=git \
  --password=${AZ_PAT_TOKEN} \
  --interval=1m
```

Please consult the [Azure DevOps documentation](https://docs.microsoft.com/en-us/azure/devops/organizations/accounts/use-personal-access-tokens-to-authenticate?view=azure-devops&tabs=preview-page)
on how to generate personal access tokens for Git repositories.
Azure DevOps PAT's always have an expiration date, so be sure to have some process for renewing or updating these tokens.
Similar to the lack of repo-specific deploy keys, a user needs to generate a user-specific PAT.
If you are using a machine-user, you can generate a PAT or simply use the machine-user's password which does not expire.

Create a `Kustomization` object on your cluster:

```sh
flux create kustomization flux-system \
  --source=flux-system \
  --path="./clusters/my-cluster" \
  --prune=true \
  --interval=10m
```

Export both objects, generate a `kustomization.yaml`, commit and push the manifests to Git:

```sh
flux export source git flux-system \
  > ./clusters/my-cluster/flux-system/gotk-sync.yaml

flux export kustomization flux-system \
  >> ./clusters/my-cluster/flux-system/gotk-sync.yaml

cd ./clusters/my-cluster/flux-system && kustomize create --autodetect

git add -A && git commit -m "add sync manifests" && git push
```

Wait for Flux to reconcile your previous commit with:

```sh
watch flux get kustomization flux-system
```

### Flux Upgrade

To upgrade the Flux components to a newer version, download the latest `flux` binary,
run the install command in your repository root, commit and push the changes:

```sh
flux install \
  --export > ./clusters/my-cluster/flux-system/gotk-components.yaml

git add -A && git commit -m "Upgrade to $(flux -v)" && git push
```

The [source-controller](../components/source/controller.md) will pull the changes on the cluster,
then [kustomize-controller](../components/source/controller.md)
will perform a rolling update of all Flux components including itself.

## Helm Repositories on Azure Container Registry

The Flux `HelmRepository` object currently supports
[Chart Repositories](https://helm.sh/docs/topics/chart_repository/)
as well as fetching `HelmCharts` from paths in `GitRepository` sources.

Azure Container Registry has a sub-command ([`az acr helm`](https://docs.microsoft.com/en-us/cli/azure/acr/helm))
for working with ACR-Hosted Chart Repositories, but it is deprecated.
If you are using these deprecated Azure Chart Repositories,
you can use Flux `HelmRepository` objects with them.

[Newer ACR Helm documentation](https://docs.microsoft.com/en-us/azure/container-registry/container-registry-helm-repos)
suggests using ACR as an experimental [Helm OCI Registry](https://helm.sh/docs/topics/registries/).
This will not work with Flux, because using Charts from OCI Registries is not yet supported.

## Secrets Management with SOPS and Azure Key Vault

You will need to create an Azure Key Vault and bind a credential such as a Service Principal or Managed Identity to it.
If you want to use Managed Identities, install or enable [AAD Pod Identity](#aad-pod-identity).

Patch kustomize-controller with the proper Azure credentials, so that it may access your Azure Key Vault, and then begin
committing SOPS encrypted files to the Git repository with the proper Azure Key Vault configuration.

See the [Mozilla SOPS Azure Guide](../../guides/mozilla-sops/#azure) for further detail.

## Image Updates with Azure Container Registry

You will need to create an ACR registry and bind a credential such as a Service Principal or Managed Identity to it.
If you want to use Managed Identities, install or enable [AAD Pod Identity](#aad-pod-identity).

You may need to update your Flux install to include additional components:
```sh
flux install \
  --components-extra="image-reflector-controller,image-automation-controller" \
  --export > ./clusters/my-cluster/flux-system/gotk-components.yaml
```

Follow the [Image Update Automation Guide](../guides/image-update.md) and see the
[ACR specific section](../guides/image-update.md#azure-container-registry) for more details.

Your AKS cluster's configuration can also be updated to
[allow the kubelets to pull images from ACR](https://docs.microsoft.com/en-us/azure/aks/cluster-container-registry-integration)
without ImagePullSecrets as an optional, complimentary step.
