## Azure E2E

E2E tests for Azure are needed to mitigate introduction of new bugs in dependencies like libgit2. The goal is to verify that Flux integration with
Azure services are actually working now and in the future.

## Architecture

The [aks](./terraform/aks) Terraform creates the AKS cluster and related resources to run the tests. It creates the AKS cluster, Azure DevOps
repositories, Key Vault Key for Sops, Azure container registries and Azure EventHub. The resources should be created and destroyed before and after every test run.

## Requirements
- Azure account with an active subscription to be able to create AKS and ACR, and permission to assign roles. Role assignment is required for allowing AKS workloads to access ACR.
- Azure CLI, need to be logged in using az login.
- Docker CLI for registry login.
- kubectl for applying certain install manifests.

## Tests

Each test run is initiated by running `terraform apply` on the aks Terraform, it does this by using the [tftestenv package](https://github.com/fluxcd/test-infra/blob/main/tftestenv/testenv.go) within the `fluxcd/test-infra` repository.
It then reads the output of the Terraform to get credentials and ssh keys, this means that a lot of the communication with the Azure API is offset to
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

## Running these tests locally

1. Copy `.env.sample` to `.env` and add the values for the different variables which includes.  - your Azure DevOps org, personal access tokens and ssh keys for accessing repositories on Azure DevOps org.
Run  `source .env`.
2. Run `make test-azure`

