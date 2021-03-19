# Using Flux on Azure

## AKS Cluster Options

It's important to follow some guidelines when installing Flux on AKS.

### CNI and Network Policy

Previously, there has been an issue with Flux and Network Policy on AKS. ([Upstream Azure Issue](https://github.com/Azure/AKS/issues/2031)) ([Flux Issue](https://github.com/fluxcd/flux2/issues/703))
If you ensure your AKS cluster is upgraded, and your Nodes have been restarted with the most recent Node images, this could
resolve flux reconciliation failures where source-controller is unreachable.
Using `--network-plugin=azure --network-policy=calico` has been tested to work properly.
This issue only affects you if you are using `--network-policy` on AKS, which is not a default option.

!!! warning
    AKS `--network-policy` is currently in Preview

### AAD Pod-Identity

Depending on the features you are interested in using with Flux, you may want to install AAD Pod Identity.
With [AAD Pod-Identity](https://azure.github.io/aad-pod-identity/docs/), we can create Pods that have their own
cloud credentials for accessing Azure services like Azure Container Registry(ACR) and Azure Key Vault(AKV).

If you do not use AAD Pod-Identity, you'll need to manage and store Service Principal credentials in K8s Secrets, to integrate Flux
with other Azure Services.

As a pre-requisite, your cluster must have `--enable-managed-identity` configured.

This software can be [installed via Helm](https://azure.github.io/aad-pod-identity/docs/getting-started/installation/) (unmanaged by Azure).
Use Flux's `HelmRepository` and `HelmRelease` object to manage the aad-pod-identity installation from a bootstrap repository and keep it up to date.

!!! note
    As an alternative to Helm, the `--enable-aad-pod-identity` flag for the `az aks create` is currently in Preview.
    Follow the Azure guide for [Creating an AKS cluster with AAD Pod Identity](https://docs.microsoft.com/en-us/azure/aks/use-azure-ad-pod-identity) if you would like to enable this feature with the Azure CLI.

### Cluster Creation

!!! info
    When working with the Azure CLI, it can help to set a default `location`, `group`, and `acr`.
    See `az configure --help`, `az configure --list-defaults`, and `az configure --defaults key=value`

The following creates an AKS cluster with some minimal configuration that will work well with Flux:

```sh
az aks create \
 --network-plugin="azure" \
 --network-policy="calico" \
 --enable-managed-identity \
 --enable-pod-identity \
 --name="my-cluster"
```

## Flux Installation with Azure DevOps Repos

Ensure you can login to [https://dev.azure.com](dev.azure.com) for your proper organization, and create a new repo to hold your
flux install and other necessary config.

There is no bootstrap provider currently for Azure DevOps Repos,
but you can clone your Azure Repo, then use the [Generic Git Server](../guides/installation.md#generic-git-server)
guide to manually bootstrap Flux. (It must be a Git repo; TFVC Repos are not supported by source-controller)
Take note of the Azure DevOps specific section within the guide.

If you use the generated SSH deploy key from `flux create source git`, ensure it is an RSA key (not an elliptic curve).
Make sure to use the `libgit2` provider for all `GitRepository` objects fetching from Azure Repos since they use Git Protocol v2.

Whether you're using the generated SSH deploy key or a Personal Access Token, the credentials used by
Flux will need to be owned by an Azure DevOps User with access to the repo.
Consider creating a machine-user and granting it granular permissions to access what's needed.
This allows changing user access without affecting Flux.
Since PAT's expire on Azure DevOps, using a machine-user's login password to authenticate with HTTPS and `libgit2`
can be a good option that avoids the need to renew the credential while also having the benefit of more granular permissions.

## Helm Repositories on Azure Container Registry

The Flux `HelmRepository` object currently supports [Chart Repositories](https://helm.sh/docs/topics/chart_repository/)
as well as fetching `HelmCharts` from paths in `GitRepository` sources.

Azure Container Registry has a sub-command ([`az acr helm`](https://docs.microsoft.com/en-us/cli/azure/acr/helm)) for working with
ACR-Hosted Chart Repositories, but it is deprecated.
If you are using these deprecated Azure Chart Repositories, you can use Flux `HelmRepository` objects with them.

[Newer ACR Helm documentation](https://docs.microsoft.com/en-us/azure/container-registry/container-registry-helm-repos) suggests
using ACR as an experimental [Helm OCI Registry](https://helm.sh/docs/topics/registries/).
This will not work with Flux, because using Charts from OCI Registries is not yet supported.

## Secrets Management with SOPS and Azure Key Vault

You will need to create an Azure Key Vault and bind a credential such as a Service Principal or Managed Identity to it.
If you want to use Managed Identities, install or enable [AAD Pod Identity](#aad-pod-identity).

Patch kustomize-controller with the proper Azure credentials, so that it may access your Azure Key Vault, and then begin
committing SOPS encrypted files to the Git repository with the proper Azure Key Vault configuration.

See the [Mozilla SOPS Azure Guide](../guides/azure.md) for further detail.

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

Your AKS cluster's configuration can also be updated to [allow the kubelets to pull images from ACR](https://docs.microsoft.com/en-us/azure/aks/cluster-container-registry-integration)
without ImagePullSecrets as an optional, complimentary step.
