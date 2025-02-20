# RFC-0007 Passwordless authentication for Git repositories

**Status:** implementable

**Creation date:** 2023-31-07
**Last update:** 2024-06-12

## Summary

Flux should provide a mechanism to authenticate against Git repositories without
the use of passwords. This RFC proposes the use of alternative authentication
methods like OIDC, OAuth2 and IAM to access Git repositories hosted on specific
Git SaaS platforms and cloud providers.

## Motivation

At the moment, Flux supports HTTP basic and bearer authentication. Users are
required to create a Secret containing the username and the password/bearer
token, which is then referred to in the GitRepository using `.spec.secretRef`.

While this works fine, it has a couple of drawbacks:
* Scalability: Each new GitRepository potentially warrants another credentials
pair, which doesn't scale well in big organizations with hundreds of
repositories with different owners, increasing the risk of mismanagement and
leaks.
* Identity: A username is associated with an actual human. But often, the
repository belongs to a team of 2 or more people. This leads to a problem where
teams have to decide whose credentials should Flux use for authentication.

These problems exist not due to flaws in Flux, but because of the inherent
nature of password based authentication.

With support for OIDC, OAuth2 and IAM based authentication, we can eliminate
these problems:
* Scalability: Since OIDC is fully handled by the cloud provider, it eliminates
any user involvement in managing credentials. For OAuth2 and IAM, users do need
to provide certain information like the ID of the resource, private key, etc.
but these are still a better alternative to passwords since the same resource
can be reused by multiple teams with different members.
* Identity: Since all the above authentication methods are associated with a
virtual resource independent of a user, it solves the problem of a single person
being tied to automation that several people are involved in.

### Goals

* Integrate with major cloud providers' OIDC and IAM offerings to provide a
seamless way of Git repository authentication.
* Integrate with major Git SaaS providers to support their app based OAuth2
mechanism.

### Non-Goals

* Replace the existing basic and bearer authentication API.

## Proposal

A new string field `.spec.provider` shall be added to the `GitRepository` API.
The field will be an enum with the following variants:

* `generic`
* `aws`
* `azure`
* `gcp`
* `github`
* `gitlab`

`.spec.provider` will be an optional field which defaults to `generic` indicating
that the user wants to authenticate via HTTP basic/bearer auth or SSH by providing
the existing `.spec.secretRef` field. The sections below define the behavior when
`.spec.provider` is set to one of the other providers.

### AWS

Git repositories hosted on AWS CodeCommit can be accessed by Flux via [IAM roles
for service accounts
(IRSA)](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html)
and
[git-remote-codecommit (GRC)](https://docs.aws.amazon.com/codecommit/latest/userguide/setting-up-git-remote-codecommit.html)
signed URLs.

The IAM role associated with service account used in Flux can be granted access
to the CodeCommit repository. The Flux service account can be patched with the
name of the IAM role to be assumed as an annotation. The CodeCommit HTTPS (GRC)
repository URL is of the format `codecommit::<region>://<repo-name>`. This can
be converted to a signed URL before performing a go-git Git operation.

The following patch can be used to add the IAM role name to Flux service accounts:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - gotk-components.yaml
  - gotk-sync.yaml
patches:
  - patch: |
      apiVersion: v1
      kind: ServiceAccount
      metadata:
        name: source-controller
        annotations:
          eks.amazonaws.com/role-arn: <role arn>
    target:
      kind: ServiceAccount
      name: source-controller
```

Example of using AWS CodeCommit with `aws` provider:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: aws-repo
spec:
  interval: 1m
  url: codecommit::<region>://<repository>
  ref:
    branch: master
  provider: aws
```

### Azure

Git repositories hosted on Azure Devops can be accessed using [managed
identity](https://learn.microsoft.com/en-us/azure/devops/integrate/get-started/authentication/service-principal-managed-identity?view=azure-devops).
Seamless access from Flux to Azure devops repository can be achieved through
[Workload
Identity](https://learn.microsoft.com/en-us/azure/aks/workload-identity-overview?tabs=dotnet).
The user creates a managed identity and establishes a federated identity between
Flux service account and the managed identity. Flux service account is patched
to add an annotation specifying the client id of the managed identity. Flux
service account and deployments are patched with labels to use workload
identity.  The managed identity must have sufficient permissions to be able to
access Azure Devops resources. This enables Flux pod to access the Git
repository without the need for any credentials.

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - gotk-components.yaml
  - gotk-sync.yaml
patches:
  - patch: |-
      apiVersion: v1
      kind: ServiceAccount
      metadata:
        name: source-controller
        namespace: flux-system
        annotations:
          azure.workload.identity/client-id: <AZURE_CLIENT_ID>
        labels:
          azure.workload.identity/use: "true"      
  - patch: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: source-controller
        namespace: flux-system
        labels:
          azure.workload.identity/use: "true"
      spec:
        template:
          metadata:
            labels:
              azure.workload.identity/use: "true" 
```

Example of using an Azure Devops repository with `azure` provider:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: azure-devops
spec:
  interval: 1m
  url: https://dev.azure.com/<org>/<project>/_git/<repository>
  ref:
    branch: master
  # notice the lack of secretRef
  provider: azure
```

### GCP

Git repositories hosted on Google Cloud Source Repositories can be accessed by
Flux via a [GCP Service Account](https://cloud.google.com/iam/docs/service-account-overview).

Workload Identity Federation for GKE is [unsupported](https://cloud.google.com/iam/docs/federated-identity-supported-services)
for Cloud Source Repositories. The user must instead create the GCP Service Account and 
link it to the Flux service account in order to enable workload identity.

In order to link the GCP Service Account to the Flux service
account, the following patch must be applied:
  
```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - gotk-components.yaml
  - gotk-sync.yaml
patches:
  - patch: |
      apiVersion: v1
      kind: ServiceAccount
      metadata:
        name: source-controller
        annotations:
          iam.gke.io/gcp-service-account: <identity-name>      
    target:
      kind: ServiceAccount
      name: source-controller
```

The Service Account must have sufficient permissions to be able to access Google
Cloud Source Repositories. The Cloud Source Repositories uses the `source.repos.get`
permission to access the repository, which is under the `roles/source.reader` role.
Take a look at [this guide](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity)
for more information about setting up GKE Workload Identity.

Example of using a Google Cloud Source Repository with `gcp` provider:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: gcp-repo
spec:
  interval: 1m
  url: https://source.developers.google.com/p/<project>/r/<repository>
  ref:
    branch: master
  provider: gcp
```

### GitHub

Git repositories hosted on GitHub can be accessed via [GitHub Apps](https://docs.github.com/en/apps/overview).
This allows users to create a single resource from which they can access all
their GitHub repositories. The app must have sufficient permissions to be able
to access repositories. The app's ID, private key and installation ID should
be mentioned in the Secret referred to by `.spec.secretRef`. GitHub Enterprise
users will also need to mention their GitHub API URL in the Secret.

Example of using a Github repository with `github` provider:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: github-repo
spec:
  interval: 1m
  url: https://github.com/<org>/<repository>
  ref:
    branch: master
  provider: github
  secretRef:
    name: github-app
---
kind: Secret
metadata:
  name: github-app
stringData:
  githubAppID: <app-id>
  githubInstallationID: <installation-id>
  githubPrivateKey: |
    <PEM-private-key>
  githubApiURl: <github-enterprise-api-url> #optional, required only for GitHub Enterprise users
```

### Gitlab

Git repositories hosted on Gitlab can be accessed via OAuth2 Gitlab Applications
created from the
[UI](https://docs.gitlab.com/ee/integration/oauth_provider.html) or using
[API](https://docs.gitlab.com/ee/api/applications.html). The Gitlab Oauth2
application must be created with the required scope to access gitlab
repositories. The application's `application_id`, `secret` and `redirect_uri`
are used to request an [access
token](https://docs.gitlab.com/ee/api/oauth2.html#authorization-code-flow).
These parameters are configured in the secret referred to by `.spec.secretRef`.

Example of using gitlab repository with `gitlab` provider:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: gitlab-repo
spec:
  interval: 1m
  url: https://gitlab.com/<org>/<repository>
  ref:
    branch: main
  provider: gitlab
  secretRef:
    name: gitlab-app
---
kind: Secret
metadata:
  name: gitlab-app
stringData:
  gitlabAppID: <app-id>
  gitlabAppSecret: <app-secret>
  gitlabAppRedirectUrl: <app-redirect-url>
```

### User Stories 

#### User Story 1

> As a user running flux controllers, deployed from a private repository in
> a cloud provider that supports context-based authentication, I want to securely
> authenticate to the repository without setting up secrets and having to manage
> authentication tokens (refreshing, rotating, etc.).

To enable this scenario, the user would enable context-based authentication in
their cloud provider and integrate it with their kubernetes cluster. For example,
in Azure, using AKS and Azure Devops, the user would create a managed identity and
establish a federated identity between Flux service account and the managed identity.
Flux would then be able to access the Git repository by requesting a token from the
Azure service. The user would not need to create a secret or manage any tokens.

#### User Story 2

> As a user running flux controllers, deployed from a private repository, I want
> to configure authentication to the repository that is not associated to a
> personal account and does not expire. 

To enable this scenario, the user would either enable context-based authentication
in their cloud provider and integrate it with their kubernetes cluster, or set 
up an OAuth2 application in their Git SaaS provider and provide the OAuth2 application
details (application ID, secret, redirect URL) in a kubernetes secret. 
Flux would then be able to access the Git repository by requesting a token from the
cloud provider or Git SaaS provider. The user would not need to create any credentials
tied to a personal account.

## Design Details

Flux source controller uses `GitRepository` API to define a source to produce an
Artifact for a Git repository revision. Flux image automation controller updates
YAML files when new images are available and commits changes to a given Git
repository. The `ImageUpdateAutomation` API defines an automation process that
updates the Git repository referenced in it's `.spec.sourceRef`. If the new
optional string field `.spec.provider` is specified in the `GitRepository` API,
the respective provider is used to configure the authentication to check out the
source for flux controllers.

### AWS

If `.spec.provider` is set to `aws`, Flux controllers will use the aws-sdk-go-v2
to assume the role of the IAM role associated with the pod service account and
obtain a short-lived [Security Token Service
(STS)](https://docs.aws.amazon.com/STS/latest/APIReference/welcome.html)
credential. This credential will then be used to create a signed HTTP URL to the
CodeCommit repository, similar to what git-remote-codecommit (GRC) does in
python using the boto library, see
[here](https://github.com/aws/git-remote-codecommit/blob/1.17/git_remote_codecommit/__init__.py#L176-L194).
For example, the GRC URL `codecommit::us-east-1://test-repo-1` results in a
typical Git HTTP repository address `https://AKIAYKF23ZCZFAVYGOEX:20240607T151729Zf17c9b36ba154efc81adf3df9dc3253de52e0a1ab6c81c00a5f9a26b06a103df@git-codecommit.us-east-1.amazonaws.com/v1/repos/test-repo-1`.
This URL contains a basic auth credential. This can be passed to go-git to
perform HTTP Git operations.

### Azure

If `.spec.provider` is set to `azure`, Flux controllers will use
[DefaultAzureCredential](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#DefaultAzureCredential)
to build the workload identity credential. This credential type uses the
environment variables injected by the Azure Workload Identity mutating webhook.
The [access token from the credential will be then used as a bearer
token](https://learn.microsoft.com/en-us/azure/devops/integrate/get-started/authentication/service-principal-managed-identity?view=azure-devops#q-can-i-use-a-service-principal-to-do-git-operations-like-clone-a-repo)
to perform HTTP bearer authentication.

### GCP

If `.spec.provider` is set to `gcp`, Flux source controller will fetch the access token
from the [GKE metadata server](https://cloud.google.com/kubernetes-engine/docs/concepts/workload-identity#metadata_server).
The GKE metadata server runs as a DaemonSet, with one Pod on every Linux node or
a native Windows service on every Windows node in the cluster. The metadata server
intercepts HTTP requests to `http://metadata.google.internal`. 

The source controller will use the url `http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token`
to retrieve a token for the IAM service account that the Pod is configured to impersonate. 
This access token will be then used to perform HTTP basic authentication.

### GitHub

If `.spec.provider` is set to `github`, Flux controllers will get the app
details from the specified Secret and use it to [generate an app installation
token](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-an-installation-access-token-for-a-github-app).
This token is then used as the password and [`x-access-token` as the username](https://docs.github.com/en/apps/creating-github-apps/registering-a-github-app/choosing-permissions-for-a-github-app#choosing-permissions-for-git-access)
to perform HTTP basic authentication.

### Gitlab

If `.spec.provider` is set to `gitlab`, Flux controllers will use the
application_id, secret and redirect_url specified in `.spec.secret` to generate
an access token. The git repository can then be accessed by specifying [oauth2
as the username and the access token as the
password](https://docs.gitlab.com/ee/api/oauth2.html#access-git-over-https-with-access-token)
to perform HTTP basic authentication.
