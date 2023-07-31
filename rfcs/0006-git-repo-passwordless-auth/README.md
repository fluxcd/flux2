# RFC-0006 Passwordless authentication for Git repositories

**Status:** provisional

**Creation date:** 2023-31-07

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

A new string field `.spec.provider` shall be added to the GitRepository API.
The field will be an enum with the following variants:
* `azure`
* `github`
* `gcp`

> AWS CodeCommit is not supported as it does not support authentication via IAM
Roles without the use of https://github.com/aws/git-remote-codecommit.

By default, it will be blank, which indicates that the user wants to
authenticate via HTTP basic/bearer auth or SSH.

### Azure

Git repositories hosted on Azure Devops can be accessed by Flux using OIDC if
the cluster running Flux is hosted on AKS with [managed identity](https://learn.microsoft.com/en-us/azure/devops/integrate/get-started/authentication/service-principal-managed-identity?view=azure-devops)
enabled. The managed identity associated with the cluster must have sufficient
permissions to be able to access Azure Devops resources. This enables Flux to
access the Git repository without the need for any credentials.

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
The Service Account must have sufficient permissions to be able to access Google
Cloud Source Repositories and its credentials should be specified in the secret
referred to in `.spec.secretRef`.

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
  secretRef:
    name: gcp-sa
---
kind: Secret
metadata:
  name: gcp-sa
stringData:
  gcpServiceAccount: |
    {
      "type": "service_account",
      "project_id": "my-google-project",
      "private_key_id": "REDACTED",
      "private_key": "-----BEGIN PRIVATE KEY-----\nREDACTED\n-----END PRIVATE KEY-----\n",
      "client_email": "<service-account-id>@my-google-project.iam.gserviceaccount.com",
      "client_id": "REDACTED",
      "auth_uri": "https://accounts.google.com/o/oauth2/auth",
      "token_uri": "https://oauth2.googleapis.com/token",
      "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
      "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/<service-account-id>%40my-google-project.iam.gserviceaccount.com"
    }
```

### GitHub

Git repositories hosted on GitHub can be accessed via [GitHub Apps](https://docs.github.com/en/apps/overview).
This allows users to create a single resource from which they can access all
their GitHub repositories. The app must have sufficient permissions to be able
to access repositories. The app's ID, private key and installation ID should
be mentioned in the Secret referred to by `.spec.secretRef`. GitHub Enterprise
users will also need to mention their GitHub API URL in the Secret.

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
  name: gcp-sa
stringData:
  githubAppID: <app-id>
  githubInstallationID: <installation-id>
  githubPrivateKey: |
    <PEM-private-key>
  githubApiURl: <github-enterprise-api-url> #optional, required only for GitHub Enterprise users
```

## Design Details

### Azure

If `.spec.provider` is set to `azure`, Flux controllers will reach out to
[Azure IMDS (Instance Metadata Service)](https://learn.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/how-to-use-vm-token#get-a-token-using-go)
to get an access token. This [access token will be then used as a bearer token](https://learn.microsoft.com/en-us/azure/devops/integrate/get-started/authentication/service-principal-managed-identity?view=azure-devops#q-can-i-use-a-service-principal-to-do-git-operations-like-clone-a-repo)
to perform HTTP bearer authentication.

### GCP

If `.spec.provider` is set to `gcp`, Flux controllers will get the Service
Account credentials from the specified Secret and use
[`google.CredentialsFromJSON`](https://pkg.go.dev/golang.org/x/oauth2/google#CredentialsFromJSON)
to fetch the access token. This access token will be then used as the password
and the `client_email` as the username to perform HTTP basic authentication.

### GitHub

If `.spec.provider` is set to `github`, Flux controllers will get the app
details from the specified Secret and use it to [generate an app installation
token](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-an-installation-access-token-for-a-github-app).
This token is then used as the password and [`x-access-token` as the username](https://docs.github.com/en/apps/creating-github-apps/registering-a-github-app/choosing-permissions-for-a-github-app#choosing-permissions-for-git-access)
to perform HTTP basic authentication.

### Caching

To avoid calling the upstream API for a token during every reconciliation, Flux
controllers shall cache the token after fetching it. Since GitHub and GCP tokens
self-expire, the cache shall automatically evict the token after it has expired,
triggering a fetch of a fresh token.

While Azure's managed identities subsystem caches the token, it is
[recommended for the consumer application to implement their own caching](https://learn.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/how-to-use-vm-token#token-caching)
as well.
