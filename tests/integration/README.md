# E2E Tests

The goal is to verify that Flux integration with cloud providers are actually working now and in the future.
Currently, we only have tests for Azure.

## General requirements

These CLI tools need to be installed for each of the tests to run successfully.

- Docker CLI for registry login.
- [SOPS CLI](https://github.com/mozilla/sops) for encrypting files
- kubectl for applying certain install manifests.

## Azure

### Architecture

The [azure](./terraform/azure) Terraform creates the AKS cluster and related resources to run the tests. It creates:
- An Azure Container Registry
- An Azure Kubernetes Cluster
- Two Azure DevOps repositories
- Azure EventHub for sending notifications
- An Azure Key Vault

### Requirements

- Azure account with an active subscription to be able to create AKS and ACR,
  and permission to assign roles. Role assignment is required for allowing AKS workloads to access ACR.
- Azure CLI, need to be logged in using `az login` as a User or as a Service Principal
- An Azure DevOps organization, personal access token and ssh keys for accessing repositories
  within the organization. The scope required for the personal access token is:
  - `Project and Team` - read, write and manage access
  - `Code` - Full
  - Please take a look at the [terraform provider](https://registry.terraform.io/providers/microsoft/azuredevops/latest/docs/guides/authenticating_using_the_personal_access_token#create-a-personal-access-token)
    for more explanation.
  - Azure DevOps only supports RSA keys. Please see
    [documentation](https://learn.microsoft.com/en-us/azure/devops/repos/git/use-ssh-keys-to-authenticate?view=azure-devops#set-up-ssh-key-authentication)
    for how to set up SSH key authentication.

**NOTE:** To use Service Principal (for example in CI environment), set the
`ARM-*` variables in `.env`, source it and authenticate Azure CLI with:
```console
$ az login --service-principal -u $ARM_CLIENT_ID -p $ARM_CLIENT_SECRET --tenant $ARM_TENANT_ID
```

### Permissions

Following permissions are needed for provisioning the infrastructure and running
the tests:
- `Microsoft.Kubernetes/*`
- `Microsoft.Resources/*`
- `Microsoft.Authorization/roleAssignments/{Read,Write,Delete}`
- `Microsoft.ContainerRegistry/*`
- `Microsoft.ContainerService/*`
- `Microsoft.KeyVault/*`
- `Microsoft.EventHub/*`


## Tests

Each test run is initiated by running `terraform apply` in the provider's terraform directory e.g terraform apply,
it does this by using the [tftestenv package](https://github.com/fluxcd/test-infra/blob/main/tftestenv/testenv.go)
within the `fluxcd/test-infra` repository. It then reads the output of the Terraform to get information needed
for the tests like the kubernetes client ID, the azure DevOps repository urls, the key vault ID etc. This means that
a lot of the communication with the Azure API is offset to Terraform instead of requiring it to be implemented in the test.

The following tests are currently implemented:

- Flux can be successfully installed on the cluster using the Flux CLI
- source-controller can clone cloud provider repositories (Azure DevOps, Google Cloud Source Repositories) (https+ssh)
- image-reflector-controller can list tags from provider container Registry image repositories
- kustomize-controller can decrypt secrets using SOPS and provider key vault
- image-automation-controller can create branches and push to cloud repositories (https+ssh)
- source-controller can pull charts from cloud provider container registry Helm repositories

The following tests are run only for Azure since it is supported in the notification-controller:

- notification-controller can send commit status to Azure DevOps
- notification-controller can forward events to Azure Event Hub

### Running tests locally

1. Ensure that you have the Flux CLI binary that is to be tested built and ready. You can build it by running
   `make build` at the root of this repository. The binary is located at `./bin` directory at the root and by default
   this is where the Makefile copies the binary for the tests from. If you have it in a different location, you can set it
   with the `FLUX_BINARY` variable
2. Copy `.env.sample` to `.env` and add the values for the different variables for the provider that you are running the tests for. 
3. Run `make test-<provider>`, setting the location of the flux binary with `FLUX_BINARY` variable

```console
$ make test-azure
make test PROVIDER_ARG="-provider azure"
# These two versions of podinfo are pushed to the cloud registry and used in tests for ImageUpdateAutomation
mkdir -p build
cp ../../bin/flux build/flux
docker pull ghcr.io/stefanprodan/podinfo:6.0.0
6.0.0: Pulling from stefanprodan/podinfo
Digest: sha256:e7eeab287181791d36c82c904206a845e30557c3a4a66a8143fa1a15655dae97
Status: Image is up to date for ghcr.io/stefanprodan/podinfo:6.0.0
ghcr.io/stefanprodan/podinfo:6.0.0
docker pull ghcr.io/stefanprodan/podinfo:6.0.1
6.0.1: Pulling from stefanprodan/podinfo
Digest: sha256:1169f220a670cf640e45e1a7ac42dc381a441e9d4b7396432cadb75beb5b5d68
Status: Image is up to date for ghcr.io/stefanprodan/podinfo:6.0.1
ghcr.io/stefanprodan/podinfo:6.0.1
go test -timeout 60m -v ./ -existing -provider azure --tags=integration
2023/03/24 02:32:25 Setting up azure e2e test infrastructure
2023/03/24 02:32:25 Terraform binary:  /usr/local/bin/terraform
2023/03/24 02:32:25 Init Terraform
....[some output has been cut out]
2023/03/24 02:39:33 helm repository condition not ready
--- PASS: TestACRHelmRelease (15.31s)
=== RUN   TestKeyVaultSops
--- PASS: TestKeyVaultSops (15.98s)
PASS
2023/03/24 02:40:12 Destroying environment...
ok      github.com/fluxcd/flux2/tests/integration       947.341s
```

In the above, the test created a build directory build/ and the flux cli binary is copied build/flux. It would be used
to bootstrap Flux on the cluster. You can configure the location of the Flux CLI binary by setting the FLUX_BINARY variable.
We also pull two version of `ghcr.io/stefanprodan/podinfo` image. These images are pushed to the Azure Container Registry
and used to test `ImageRepository` and `ImageUpdateAutomation`. The terraform resources get created and the tests are run.

**IMPORTANT:** In case the terraform infrastructure results in a bad state, maybe due to a crash during the apply,
the whole infrastructure can be destroyed by running terraform destroy in terraform/<provider> directory.

### Debugging the tests

For debugging environment provisioning, enable verbose output with `-verbose` test flag.

```sh
make test-azure GO_TEST_ARGS="-verbose"
```

The test environment is destroyed at the end by default. Run the tests with -retain flag to retain the created test infrastructure.

```sh
make test-azure GO_TEST_ARGS="-retain"
```
The tests require the infrastructure state to be clean. For re-running the tests with a retained infrastructure, set -existing flag.

```sh
make test-azure GO_TEST_ARGS="-retain -existing"
```

To delete an existing infrastructure created with -retain flag:

```sh
make test-azure GO_TEST_ARGS="-existing"
```

To debug issues on the cluster created by the test (provided you passed in the `-retain` flag):

```sh
export KUBECONFIG=./build/kubeconfig
kubectl get pods
```
