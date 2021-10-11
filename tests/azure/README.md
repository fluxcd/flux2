# Azure E2E

E2E tests for Azure are needed to mitigate introduction of new bugs in dependencies like libgit2. The goal is to verify that Flux integration with
Azure services are actually working now and in the future.

## Architecture

The tests are run with the help of pre configured Azure subscriptions and Azure DevOps organization. Access to thse accounts are currently limited to
Flux maintainers.
* [Azure Subscription](https://portal.azure.com/#@weaveworksendtoend.onmicrosoft.com/resource/subscriptions/71e8dce4-9af6-405a-8e96-425f5d3c302b/overview)
* [Azure DevOps organization](https://dev.azure.com/flux-azure/)

All infrastructure is and should be created with Terraform. There are two separate Terraform states. All state should be configured to use remote
state in Azure. They should all be placed in the [same container](https://portal.azure.com/#@weaveworksendtoend.onmicrosoft.com/resource/subscriptions/71e8dce4-9af6-405a-8e96-425f5d3c302b/resourceGroups/terraform-state/providers/Microsoft.Storage/storageAccounts/terraformstate0419/containersList)
but use different keys.

The [shared](./terraform/shared) Terraform creates long running cheaper infrastructure that is used across all tests. This includes a Key Vault which
contains an ssh key and Azure DevOps Personal Access Token which cant be created automatically. It also includes an Azure Container Registry which the
forked [podinfo](https://dev.azure.com/flux-azure/e2e/_git/podinfo) repository pushes an Helm Chart and Docker image to.

The [aks](./terraform/aks) Terraform creates the AKS cluster and related resources to run the tests. It creates the AKS cluster, Azure DevOps
repositories, Key Vault Key for Sops, and Azure EventHub. The resources should be created and destroyed before and after every test run. Currently
the same state is reused between runs to make sure that resources are left running after each test run.

## Tests

Each test run is intiated by running `terraform apply` on the aks Terraform, it does this by using the library [terraform-exec](github.com/hashicorp/terraform-exec).
It then reads the output of the Terraform to get credentials and ssh keys, this means that a lot of the communication with the Azure API is offest to
Terraform instead of requiring it to be implemented in the test.

The following tests are currently implemented:

- [x] Flux can be successfully installed on AKS using the CLI e.g.:
- [x] source-controller can clone Azure DevOps repositories (https+ssh)
- [x] image-reflector-controller can list tags from Azure Container Registry image repositories
- [x] kustomize-controller can decrypt secrets using SOPS and Azure Key Vault
- [x] image-automation-controller can create branches and push to Azure DevOps repositories (https+ssh)
- [x] notification-controller can send commit status to Azure DevOps
- [x] notification-controller can forward events to Azure Event Hub
- [x] source-controller can pull charts from Azure Container Registry Helm repositories

## Give User Access

There are a couple of steps required when adding a new user to get access to the Azure portal and Azure DevOps. To begin with add the new user to
[Azure AD](https://portal.azure.com/#blade/Microsoft_AAD_IAM/UsersManagementMenuBlade/MsGraphUsers), and add the user to the [Azure AD group
flux-contributors](https://portal.azure.com/#blade/Microsoft_AAD_IAM/GroupDetailsMenuBlade/Overview/groupId/24e0f3f6-6555-4d3d-99ab-414c869cab5d). The
new users should now go through the invite process, and be able to sign in to both the Azure Portal and Azure DevOps.

After the new user has signed into Azure DevOps you will need to modify the users permissions. This cannot be done before the user has signed in a
first time. In the [organization users page](https://dev.azure.com/flux-azure/_settings/users) chose "Manage User" and set the "Access Level" to basic
and make the user "Project Contributor" of the "e2e" Azure DevOps project.
