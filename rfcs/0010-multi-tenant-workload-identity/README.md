# RFC-0010 Multi-Tenant Workload Identity

**Status:** implementable

<!--
Status represents the current state of the RFC.
Must be one of `provisional`, `implementable`, `implemented`, `deferred`, `rejected`, `withdrawn`, or `replaced`.
-->

**Creation date:** 2025-02-22

**Last update:** 2025-04-29

## Summary

In this RFC we aim to add support for multi-tenant workload identity in Flux,
i.e. the ability to specify at the object-level which set of cloud provider
permissions must be used for interacting with the respective cloud provider
on behalf of the reconciliation of the object. In this process, credentials
must be obtained automatically, i.e. this feature must not involve the use
of secrets. This would be useful in a number of Flux APIs that need to
interact with cloud providers, spanning all the Flux controllers.

### Multi-Tenancy Model

In the context of this RFC, multi-tenancy refers to the ability of a single
Flux instance running inside a Kubernetes cluster to manage Flux objects
belonging to all the tenants in the cluster while still ensuring that each
tenant has access only to their own resources according to the Least Privilege
Principle. In this scenario a tenant is often a team inside an organization,
so the reader can consider the
[multi-team tenancy model](https://kubernetes.io/docs/concepts/security/multi-tenancy/#multiple-teams).
Each team has their own namespaces, which are not shared with other teams.

## Motivation

Flux has strong multi-tenancy features. For example, the `Kustomization` and
`HelmRelease` APIs support the field `spec.serviceAccountName` for specifying
the Kubernetes `ServiceAccount` to impersonate when interacting with the
Kubernetes API on behalf of a tenant, e.g. when applying resources. This
allows tenants to be constrained under the Kubernetes RBAC permissions
granted to this `ServiceAccount`, and therefore have access only to the
specific subset of resources they should be allowed to use.

Besides the Kubernetes API, Flux also interacts with cloud providers, e.g.
container registries, object storage, pub/sub services, etc. In these cases,
Flux currently supports basically two modes of authentication:

- *Secret-based multi-tenant authentication*: Objects have the field
  `spec.secretRef` for specifying the Kubernetes `Secret` containing the
  credentials to use when interacting with the cloud provider. This is
  similar to the `spec.serviceAccountName` field, but for cloud providers.
  The problem with this approach is that secrets are a security risk and
  operational burden, as they must be managed and rotated.
- *Workload-identity-based single-tenant authentication*: Flux offers
  single-tenant workload identity support by configuring the `ServiceAccount`
  of the Flux controllers to impersonate a cloud identity. This eliminates
  the need for secrets, as the credentials are obtained automatically by
  the cloud provider Go libraries used by the Flux controllers when they
  are running inside the respective cloud environment. The problem with
  this approach is that it is single-tenant, i.e. all objects are reconciled
  using the same cloud identity, the one associated with the respective controller.

For delivering the high level of security and multi-tenancy support that
Flux aims for, it is necessary to extend the workload identity support to
be multi-tenant. This means that each object must be able to specify which
cloud identity must be impersonated when interacting with the cloud provider
on behalf of the reconciliation of the object. This would allow tenants to
be constrained under the cloud provider permissions granted to this identity,
and therefore have access only to the specific subset of resources they are
allowed to manage.

### Goals

Provide multi-tenant workload identity support in Flux, i.e. the ability to
specify at the object-level which cloud identity must be impersonated to
interact with the respective cloud provider on behalf of the reconciliation
of the object, without the need for secrets.

### Non-Goals

It's not a goal of this RFC to implement an identity provider for Flux.
Instead, the goal is to leverage Kubernetes' built-in identity provider
capabilities, i.e. the Kubernetes `ServiceAccount` token issuer, to
obtain short-lived access tokens for the cloud providers.

## Proposal

For supporting multi-tenant workload identity at the object-level for the Flux APIs
we propose associating the Flux objects with Kubernetes `ServiceAccounts`. The
controller would need to create a token for the `ServiceAccount` associated with
the object in the Kubernetes API, and then exchange it for a short-lived access
token for the cloud provider. This would require the controller `ServiceAccount`
to have RBAC permission to create tokens for any `ServiceAccounts` in the cluster.

### User Stories

#### Story 1

> As a cluster administrator, I want to allow tenant A to pull OCI artifacts
> from the Amazon ECR repository belonging to tenant A, but only from this
> repository. At the same time, I want to allow tenant B to pull OCI artifacts
> from the Amazon ECR repository belonging to tenant B, but only from this
> repository.

For example, I would like to have the following configuration:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: OCIRepository
metadata:
  name: tenant-a-repo
  namespace: tenant-a
spec:
  ...
  provider: aws
  serviceAccountName: tenant-a-ecr-sa
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tenant-a-ecr-sa
  namespace: tenant-a
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789123:role/tenant-a-ecr
---
apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: OCIRepository
metadata:
  name: tenant-b-repo
  namespace: tenant-b
spec:
  ...
  provider: aws
  serviceAccountName: tenant-b-ecr-sa
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tenant-b-ecr-sa
  namespace: tenant-b
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789123:role/tenant-b-ecr
```

#### Story 2

> As a cluster administrator, I want to allow tenant A to pull and push to the Git
> repository in Azure DevOps belonging to tenant A, but only this repository. At
> the same time, I want to allow tenant B to pull and push to the Git repository
> in Azure DevOps belonging to tenant B, but only this repository.

For example, I would like to have the following configuration:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: tenant-a-repo
  namespace: tenant-a
spec:
  ...
  provider: azure
  serviceAccountName: tenant-a-azure-devops-sa
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tenant-a-azure-devops-sa
  namespace: tenant-a
  annotations:
    azure.workload.identity/client-id: d6e4fc00-c5b2-4a72-9f84-6a92e3f06b08 # client ID for my tenant A
    azure.workload.identity/tenant-id: 72f988bf-86f1-41af-91ab-2d7cd011db47 # azure tenant for the cluster (optional, defaults to the env var AZURE_TENANT_ID set in the controller)
---
apiVersion: image.toolkit.fluxcd.io/v1beta2
kind: ImageUpdateAutomation
metadata:
  name: tenant-a-image-update
  namespace: tenant-a
spec:
  ...
  sourceRef:
    kind: GitRepository
    name: tenant-a-repo
---
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: tenant-b-repo
  namespace: tenant-b
spec:
  ...
  provider: azure
  serviceAccountName: tenant-b-azure-devops-sa
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tenant-b-azure-devops-sa
  namespace: tenant-b
  annotations:
    azure.workload.identity/client-id: 4a7272f9-f186-41af-9f84-6a92e32d7cd0 # client ID for my tenant B
    azure.workload.identity/tenant-id: 72f988bf-86f1-41af-91ab-2d7cd011db47 # azure tenant for the cluster (optional, defaults to the env var AZURE_TENANT_ID set in the controller)
---
apiVersion: image.toolkit.fluxcd.io/v1beta2
kind: ImageUpdateAutomation
metadata:
  name: tenant-b-image-update
  namespace: tenant-b
spec:
  ...
  sourceRef:
    kind: GitRepository
    name: tenant-b-repo
```

#### Story 3

> As a cluster administrator, I want to allow tenant A to pull manifests from
> the GCS bucket belonging to tenant A, but only from this bucket. At the same
> time, I want to allow tenant B to pull manifests from the GCS bucket
> belonging to tenant B, but only from this bucket.

For example, I would like to have the following configuration:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: Bucket
metadata:
  name: tenant-a-bucket
  namespace: tenant-a
spec:
  ...
  provider: gcp
  serviceAccountName: tenant-a-gcs-sa
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tenant-a-gcs-sa
  namespace: tenant-a
  annotations:
    iam.gke.io/gcp-service-account: tenant-a-bucket@my-org-project.iam.gserviceaccount.com
---
apiVersion: source.toolkit.fluxcd.io/v1
kind: Bucket
metadata:
  name: tenant-b-bucket
  namespace: tenant-b
spec:
  ...
  provider: gcp
  serviceAccountName: tenant-b-gcs-sa
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tenant-b-gcs-sa
  namespace: tenant-b
  annotations:
    iam.gke.io/gcp-service-account: tenant-b-bucket@my-org-project.iam.gserviceaccount.com
```

#### Story 4

> As a cluster administrator, I want to allow tenant A to decrypt secrets using
> the AWS KMS key belonging to tenant A, but only this key. At the same time,
> I want to allow tenant B to decrypt secrets using the AWS KMS key belonging
> to tenant B, but only this key.

For example, I would like to have the following configuration:

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: tenant-a-aws-kms
  namespace: tenant-a
spec:
  ...
  decryption:
    provider: sops
    serviceAccountName: tenant-a-aws-kms-sa
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tenant-a-aws-kms-sa
  namespace: tenant-a
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789123:role/tenant-a-kms
---
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: tenant-b-aws-kms
  namespace: tenant-b
spec:
  ...
  decryption:
    provider: sops
    serviceAccountName: tenant-b-aws-kms-sa
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tenant-b-aws-kms-sa
  namespace: tenant-b
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789123:role/tenant-b-kms
```

#### Story 5

> As a cluster administrator, I want to allow tenant A to publish notifications
> to the `tenant-a` topic in Google Cloud Pub/Sub, but only to this topic. At
> the same time, I want to allow tenant B to publish notifications to the
> `tenant-b` topic in Google Cloud Pub/Sub, but only to this topic. I want
> to do so without creating any GCP IAM Service Accounts.

For example, I would like to have the following configuration:

```yaml
apiVersion: notification.toolkit.fluxcd.io/v1beta3
kind: Provider
metadata:
  name: tenant-a-google-pubsub
  namespace: tenant-a
spec:
  ...
  type: googlepubsub
  serviceAccountName: tenant-a-google-pubsub-sa
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tenant-a-google-pubsub-sa
  namespace: tenant-a
---
apiVersion: notification.toolkit.fluxcd.io/v1beta3
kind: Provider
metadata:
  name: tenant-b-google-pubsub
  namespace: tenant-b
spec:
  ...
  type: googlepubsub
  serviceAccountName: tenant-b-google-pubsub-sa
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tenant-b-google-pubsub-sa
  namespace: tenant-b
```

#### Story 6

> As a cluster administrator, I want to allow tenant A to use a GCP
> Service Account to apply resources in a remote GKE cluster with
> Kubernetes RBAC permissions granted to this GCP Service Account,
> and tenant B to do the same using a different GCP Service Account.

For example, I would like to have the following configuration:

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: tenant-a-gke
  namespace: tenant-a
spec:
  ...
  kubeConfig:
    provider: gcp
    serviceAccountName: tenant-a-gke-sa
    cluster: projects/<project-id>/locations/<location>/clusters/<cluster-name>
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tenant-a-gke-sa
  namespace: tenant-a
---
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: tenant-b-gke
  namespace: tenant-b
spec:
  ...
  kubeConfig:
    provider: gcp
    serviceAccountName: tenant-b-gke-sa
    cluster: projects/<project-id>/locations/<location>/clusters/<cluster-name>
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tenant-b-gke-sa
  namespace: tenant-b
```

### Alternatives

#### An alternative for identifying Flux resources in cloud providers

Instead of issuing `ServiceAccount` tokens in the Kubernetes API we could
come up with a username naming scheme for Flux resources and issue tokens
for these usernames instead, e.g. `flux:<resource type>:<namespace>:<name>`.
This would make each Flux object have its own identity instead of using
`ServiceAccounts` for this purpose. This choice would then prevent cases
of other Flux objects from malicious actors in the same namespace from
abusing the permissions granted to the `ServiceAccount` of the object.
This choice, however, would provide a worse user experience, as Flux and
Kubernetes users are already used to the `ServiceAccount` resource being
the identity for resources in the cluster, not only in the context of plain
RBAC but also in the context of workload identity.
This choice would also require the introduction of new APIs for configuring
the respective cloud identities in the Flux objects, when such APIs already
exist as defined by the cloud providers themselves as annotations in the
`ServiceAccount` resources. We therefore choose to stick with the well-known
pattern of using `ServiceAccounts` for configuring the identities of the
Flux resources. Furthermore, as mentioned in the
[Multi-Tenancy Model](#multi-tenancy-model) section, the tenant trust domains
are namespaces, so a tenant is expected to control and have access to all
the resources `ServiceAccounts` in their namespaces are allowed to access.

#### Alternatives for modifying controller RBAC to create `ServiceAccount` tokens

In this section we discuss alternatives for changing the RBAC of controllers for
creating `ServiceAccount` tokens cluster-wide, as it has a potential impact on
the security posture of Flux.

1. We grant RBAC permissions to the `ServiceAccounts` of the Flux controllers
  (that would implement multi-tenant workload identity) for creating tokens
  for any other `ServiceAccounts` in the cluster.
2. We require users to grant "self-impersonation" to the `ServiceAccounts` so they
  can create tokens for themselves. The controller would then impersonate the
  `ServiceAccount` when creating a token for it. This operation would then only
  succeed if the `ServiceAccount` has been correctly granted permission to create
  a token for itself.

In both alternatives the controller `ServiceAccount` would require some form
of cluster-wide impersonation permission. Alternative 2 requires impersonation
permission to be granted directly to the controller `ServiceAccount`, while
in alternative 1, impersonation permission would be indirectly granted by the
process of creating a token for another `ServiceAccount`. By creating a token
for another `ServiceAccount`, the controller `ServiceAccount` effectively has
the same permissions as the `ServiceAccount` it is creating the token for, as
it could simply use the token to impersonate the `ServiceAccount`. Therefore
it is reasonable to affirm that both alternatives are equivalent in terms of
security.

To break the tie between the two alternatives we introduce the fact that
alternative 1 eliminates operational burden on users. In fact, native
workload identity for pods does not require users to grant this
self-impersonation permission to the `ServiceAccounts` of the pods.

We therefore choose alternative 1.

## Design Details

For detailing the proposal we need to first introduce the technical
background on how workload identity is implemented by the managed
Kubernetes services from the cloud providers.

### Technical Background

Workload identity in Kubernetes is based on
[OpenID Connect Discovery](https://openid.net/specs/openid-connect-discovery-1_0.html)
(OIDC).
The *Kubernetes `ServiceAccount` token issuer*, included as the `iss` JWT claim in the
issued tokens, and represented by the default URL `https://kubernetes.default.svc.cluster.local`,
implements the OIDC discovery protocol. Essentially, this means that the Kubernetes API
will respond requests to the URL
`https://kubernetes.default.svc.cluster.local/.well-known/openid-configuration`
with a JSON document similar to the one below:

```json
{
  "issuer": "https://kubernetes.default.svc.cluster.local",
  "jwks_uri": "https://172.18.0.2:6443/openid/v1/jwks",
  "response_types_supported": [
    "id_token"
  ],
  "subject_types_supported": [
    "public"
  ],
  "id_token_signing_alg_values_supported": [
    "RS256"
  ]
}
```

And to the URL `https://172.18.0.2:6443/openid/v1/jwks`, *discovered* through the field
`.jwks_uri` in the JSON response above, the Kubernetes API will respond a JSON document
similar to the following:

```json
{
  "keys": [
    {
      "use": "sig",
      "kty": "RSA",
      "kid": "NWm3YKmazJPVP7tttzkmSxUn0w8LGGp7yS2CanEF-A8",
      "alg": "RS256",
      "n": "lV2tbw9hnz1mseah2kMQNe5sRju4mPLlK0F7np97lLNC49G8yc5TMjyciLF3qsDNFCfWyYmsuGlcRg2BIBBX_jkpIUUjlsktdHhuqO2RnOqyRtNuljlT_b0QJgpgxCqq0DHI31EBc0JALOVd6EjjlhsVvVzZOw_b9KBXVS3D3RENuT0_FWauDq5NYbyYnjlvk-vUXCRMNDQSDNwx6X6bktwsmeDRXtM_bP3DokmnMYc4n0asTEg14L6VKky0ByF88Wi1-y0Pm0BHdobDGt1cIeUDeThk4E79JCHxkT5urAyYHcNwcfU4q-tnD6bTpNkFVsk3cqqK2nF7R_7ac5arSQ",
      "e": "AQAB"
    }
  ]
}
```

This JSON document contains the public keys for verifying the signature of the issued tokens.

By querying these two URLs in sequence, cloud providers are able to fetch the information
required for verifying and trusting the tokens issued by the Kubernetes API. Most specifically,
for trusting the `sub` JWT claim, which contains the Kubernetes `ServiceAccount` reference
(name and namespace) for which the token was issued for, i.e. the `ServiceAccount` properly
said.

By allowing permissions to be granted to `ServiceAccounts` in the cloud provider,
the cloud provider is then able to allow Kubernetes `ServiceAccounts` to access its resources.
This is usually done by a *Security Token Service* (STS) that exchanges the Kubernetes token
for a short-lived cloud provider access token, which is then used to access the cloud provider
resources.

It's important to mention that the Kubernetes `ServiceAccount` token issuer URL must be
trusted by the cloud provider, i.e. users must configure this URL as a trusted identity
provider.

This process forms the basis for workload identity in Kubernetes. As long as the issuer
URL can be reached by the cloud provider, this process can take place successfully.

The reachability of the issuer URL by the cloud provider is where the implementation
of workload identity starts to differ between cloud providers. For example, in GCP
one can configure the content of the JWKS document directly in the GCP IAM console,
which eliminates the need for network calls to the Kubernetes API. In AWS, on the
other hand, this is not possible, the process has to be followed strictly, i.e. the
issuer URL must be reachable by the AWS STS service.

Furthermore, GKE automatically
creates the necessary trust relationship between the Kubernetes issuer and the GCP
STS service (i.e. automatically injects the JWKS document of the GKE cluster in the
STS database), while in EKS this must be done manually by users (an OIDC provider
must be created for each EKS cluster).

Another difference is that the issuer URL remains the default/private one in GKE,
while in EKS it is automatically set to a public one. This is done through
the `--service-account-issuer` flag in the `kube-apiserver` command line arguments
([docs](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#service-account-issuer-discovery)). This is a nice feature, as it allows external
systems to federate access for workloads running in EKS clusters, e.g. EKS workloads
can have federated access to GCP resources.

Yet another difference between cloud providers that sheds light in our proposal is
how applications running inside pods from the managed Kubernetes services obtain
the short-lived cloud provider access tokens. In GCP, the GCP libraries used by
the applications attempt to retrieve tokens from the *metadata server*, which is
reachable by all pods running in GKE. This server creates a token for the
`ServiceAccount` of the calling pod in the Kubernetes API, exchanges it for a
short-lived GCP access token, and returns it to the application. In AKS, on the
other hand, pods are mutated to include a
[*token volume projection*](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#serviceaccount-token-volume-projection). The kubelet mounts and automatically
rotates a volume with a token file inside the pod. The Azure libraries used by
the applications then read this file periodically to perform the token exchange
with the Azure STS service.

Another aspect of workload identity that is important for this RFC is how the cloud
identities are associated with the Kubernetes `ServiceAccounts`. In most cases, an
identity from the IAM service of the cloud provider (e.g. a GCP IAM Service Account,
or an AWS IAM Role) is associated with a Kubernetes `ServiceAccount` by the process
of *impersonation*. Permission to impersonate the cloud identity is granted to the
`ServiceAccount` through a configuration that points to the fully qualified name of
the Kubernetes `ServiceAccount`, i.e. the name and namespace of the `ServiceAccount`
and which cluster it belongs to in the name/address system of the cloud provider.

Because the cloud provider needs to support this impersonation permission, some
cloud providers go further and even remove the impersonation requirement, by
allowing permissions to be granted directly to `ServiceAccounts` (if it needs to
support granting the impersonation permission, then it can probably also easily
support granting any other permissions depending on the implementation). GCP for
example has implemented this feature [recently](https://cloud.google.com/blog/products/identity-security/make-iam-for-gke-easier-to-use-with-workload-identity-federation), a GCP IAM
Service Account is no longer required for workload identity, i.e. GCP IAM
permissions can now be granted directly to Kubernetes `ServiceAccounts`. This is
a significant improvement in the user experience, as it significantly reduces
the required configuration steps. AWS implemented a similar feature called *EKS
Pod Identity*, but it still requires an IAM Role to be associated with the
`ServiceAccount`. The minor improvement from the user experience perspective is
that this association is implemented entirely in the AWS EKS/IAM APIs, no
annotations are required in the Kubernetes `ServiceAccount`. Another improvement
from this EKS feature compared to *IAM Roles for Service Accounts* is that users
no longer need to create an *OIDC Provider* for the EKS cluster in the IAM API.

It should be noted, however, that in the feature proposed here we cannot support
*EKS Pod Identity* because it requires the `ServiceAccount` token to be bound to
a pod, and the Kubernetes API will only issue a `ServiceAccount` token bound to a
pod if that pod uses the respective `ServiceAccount`, which is a requirement we
simply cannot meet. The only pod guaranteed to exist from the perspective of a
Flux controller is itself. Therefore, it's impossible to support EKS Pod Identity
for multi-tenant workload identity. (It is already possible to use it for
single-tenant workload identity, though.)

In sight of the technical background presented above, our proposal becomes simpler.
The only solution to support multi-tenant workload identity at the object-level for
the Flux APIs is to associate the Flux objects with Kubernetes `ServiceAccounts`.
We propose building the `ServiceAccount` token creation and exchange logic into
the Flux controllers through a library in the `github.com/fluxcd/pkg` repository.

### API Changes and Feature Gates

For all the Flux APIs interacting with cloud providers (except `Kustomization`,
see the paragraph below), we propose introducing the field `spec.serviceAccountName`
(if not already present) for specifying the Kubernetes `ServiceAccount` on the same
namespace of the object that must be used for getting access to the respective cloud
resources. This field would be optional, and when not present the original behavior
would be observed, i.e. the feature only activates when the field is present and a
cloud provider among `aws`, `azure` or `gcp` is specified in the `spec.provider`
field. So if only the `spec.provider` field is present and set to a cloud provider,
then the controller would use single-tenant workload identity as it would prior to
the implementation of this RFC, i.e. it would use its own identity for the operation.

Note that this RFC does not seek to change the behavior when `spec.provider` is set
to `generic` (or left empty, when it defaults to `generic`), in which case the field
`spec.secretRef` can be used for specifying the Kubernetes `Secret` containing the
credentials (or `spec.serviceAccountName` in the case of the APIs dealing with
container registries, through the `imagePullSecrets` field of the `ServiceAccount`).

The `Kustomization` API uses Key Management Services (KMS) for decrypting
SOPS-encrypted secrets. We propose adding the dedicated optional field
`spec.decryption.serviceAccountName` for multi-tenant workload identity
when intercting with the KMS service. We choose having a dedicated field
for the `Kustomization` API because the field `spec.serviceAccountName`
already exists and is used for a major part of the functionality which
is authenticating with the Kubernetes API when applying resources. If
we used the same field for both purposes users would be forced to use
multi-tenancy for both cloud and Kubernetes API interactions. Furthermore,
the cloud provider in the `Kustomization` API is detected by the SOPS SDK
itself while decrypting the secrets, so we don't need to introduce
`spec.decryption.provider` for this purpose.

The `Kustomization` and `HelmRelease` APIs have the field
`spec.kubeConfig.secretRef` for specifying a Kubernetes `Secret` containing
a static kubeconfig for accessing a remote Kubernetes cluster. We propose
adding `spec.kubeConfig.configMapRef` for specifying a Kubernetes `ConfigMap`
that is mutually exclusive with `spec.kubeConfig.secretRef` for supporting
workload identity for both managed Kubernetes services from the cloud
providers and also a `generic` provider. The fields in the `ConfigMap`
would be the following:
- `data.provider`: The provider to use for obtaining the temporary
  `*rest.Config` for the remote cluster. One of `generic`, `aws`, `azure`
  or `gcp`. Required.
- `data.cluster`: Used only by `aws`, `azure` and `gcp`. The fully qualified
  name of the cluster resource in the respective cloud provider API. Needed
  for obtaining the unspecified fields `data.address` and `data["ca.crt"]`
  (not required if both are specified).
- `data.address`: The HTTPS address of the API server of the remote cluster.
  Required for `generic`, optional for `aws`, `azure` and `gcp`.
- `data.serviceAccountName`: The optional Kubernetes `ServiceAccount` to use
  for obtaining access to the remote cluster, implementing object-level
  workload identity. If not specified, the controller identity will be used.
- `data.audiences`: The audiences Kubernetes `ServiceAccount` tokens must
  be issued for as a list of strings in YAML format. Optional. Defaults to
  `data.address` for `generic`, and has hardcoded default/specific values for
  `aws`, `azure` and `gcp` depending on the provider.
- `data["ca.crt"]`: The optional PEM-encoded CA certificate of the remote
  cluster.

For remote cluster access, the configured identity, be it controller-level
or object-level, must have the necessary permissions to:
- Access the cluster resource in the cloud provider API to get the
  cluster CA certificate and the cluster API server address. This is
  only necessary if one of `data.address` or `data["ca.crt"]` is not
  specified in the `ConfigMap`. In other words, at least two of the
  three fields `data.address`, `data["ca.crt"]` and `data.cluster`
  must be specified. If both `data.address` and `data["ca.crt"]`
  are specified, then the `data.cluster` field *must not* be specified,
  the controller will error out if it is. If only `data.cluster` and
  `data.address` are specified, then `data.address` has to match at
  least one of the addresses of the cluster resource in the cloud
  provider API. If only `data.cluster` and `data["ca.crt"]` are
  specified, then the first address of the cluster resource in the
  cloud provider API will be used as the address of the remote cluster
  and the CA returned by the cloud provider API will be ignored.
  If only `data.cluster` is specified, then the first address
  of the cluster resource in the cloud provider API will be used.
- The relevant permissions for applying and managing the target resources
  in the remote cluster. For cloud providers this means either Kubernetes
  RBAC or the cloud provider API permissions, as managed Kubernetes services
  support authorizing requests through both ways.
- When used with `spec.serviceAccountName`, the authenticated identity must
  have the necessary permissions to impersonate this `ServiceAccount` in the
  remote cluster (related [bug](https://github.com/fluxcd/pkg/issues/959)).

To enable using the new `serviceAccountName` fields, we propose introducing
a feature gate called `ObjectLevelWorkloadIdentity` in the controllers that
would support the feature. In the first release we should make it opt-in so
cluster admins can consciously roll it out. If the feature gate is disabled
and users set the field a terminal error should be returned.

### Workload Identity Library

We propose using the Go package `github.com/fluxcd/pkg/auth`
for implementing a workload identity library that can be
used by all the Flux controllers that need to interact
with cloud providers. This library would be responsible
for creating the `ServiceAccount` tokens in the Kubernetes
API and exchanging them for short-lived access tokens
for the cloud provider. The library would also be responsible
for caching the tokens when configured by users.

The library should support both single-tenant and multi-tenant workload
identity because single-tenant implementations are already supported in
GA APIs and hence they must remain available for backwards compatibility.
Furthermore, it would be easier to support both use cases in a single
library as opposed to mingling a new library into the currently existing
ones, so this new library becomes the definitive unified solution for
workload identity in Flux.

The library should automatically detect whether the workload identity
is single-tenant or multi-tenant by checking if a `ServiceAccount` was
configured for the operation. If a `ServiceAccount` was configured, then
the operation is multi-tenant, otherwise it is single-tenant and the
granted access token must represent the identity associated with the
controller.

The directory structure would look like this:

```shell
.
└── auth
    ├── aws
    │   └── aws.go
    ├── azure
    │   └── azure.go
    ├── gcp
    │   └── gcp.go
    ├── access_token.go
    ├── options.go
    ├── provider.go
    ├── registry.go
    ├── restconfig.go
    └── token.go
```

The file `auth/token.go` would contain the token abstraction:

```go
package auth

// Token is an interface that represents an access token that can be used to
// authenticate with a cloud provider. The only common method is for getting the
// duration of the token, because different providers have different ways of
// representing the token. For example, Azure and GCP use a single string,
// while AWS uses three strings: access key ID, secret access key and token.
// Consumers of this interface should know what type to cast it to.
type Token interface {
	// GetDuration returns the duration for which the token is valid relative to
	// approximately time.Now(). This is used to determine when the token should
	// be refreshed.
	GetDuration() time.Duration
}
```

The file `auth/access_token.go` would contain the main algorithm for getting access tokens:

```go
package auth

// GetAccessToken returns an access token for accessing resources in the given cloud provider.
func GetAccessToken(ctx context.Context, provider Provider, opts ...Option) (Token, error) {
	// 1. Check if a ServiceAccount is configured and return the controller access token if not (single-tenant WI).
	// 2. Get the provider audience for creating the OIDC token for the ServiceAccount in the Kubernetes API.
	// 3. Get the ServiceAccount using the configured controller-runtime client.
	// 4. Get the provider identity from the ServiceAccount annotations and add it to the options.
	// 5. Build the cache key using the configured options.
	// 6. Get the token from the cache. If present, return it, otherwise continue.
	// 7. Create an OIDC token for the ServiceAccount in the Kubernetes API using the provider audience.
	// 8. Exchange the OIDC token for an access token through the Security Token Service of the provider.
	// 9. Add the final token to the cache and return it.
}
```

The file `auth/registry.go` would contain the logic for creating artifact registry credentials:

```go
package auth

// ArtifactRegistryCredentials is a particular type implementing the Token interface
// for credentials that can be used to authenticate against an artifact registry
// from a cloud provider.
type ArtifactRegistryCredentials struct {
	authn.Authenticator
	ExpiresAt time.Time
}

func (r *ArtifactRegistryCredentials) GetDuration() time.Duration {
	return time.Until(r.ExpiresAt)
}

// GetArtifactRegistryCredentials retrieves the registry credentials for the
// specified artifact repository and provider.
func GetArtifactRegistryCredentials(ctx context.Context, provider Provider,
	artifactRepository string, opts ...Option) (*ArtifactRegistryCredentials, error)
```

The file `auth/restconfig.go` would contain the logic for creating a REST config for the Kubernetes API:

```go
package auth

// RESTConfig is a particular type implementing the Token interface
// for Kubernetes REST configurations.
type RESTConfig struct {
	Host        string
	BearerToken string
	CAData      []byte
	ExpiresAt   time.Time
}

// GetDuration implements Token.
func (r *RESTConfig) GetDuration() time.Duration {
	return time.Until(r.ExpiresAt)
}

// GetRESTConfig retrieves the authentication and connection
// details to a remote Kubernetes cluster for the given provider,
// cluster resource name and API server address.
func GetRESTConfig(ctx context.Context, provider Provider,
cluster, address string, opts ...Option) (*RESTConfig, error)
```

The file `auth/provider.go` would contain the `Provider` interface:

```go
package auth

// Provider contains the logic to retrieve security credentials
// for accessing resources in a cloud provider.
type Provider interface {
	// GetName returns the name of the provider.
	GetName() string

	// NewControllerToken returns a token that can be used to authenticate
	// with the cloud provider retrieved from the default source, i.e. from
	// the environment of the controller pod, e.g. files mounted in the pod,
	// environment variables, local metadata services, etc.
	NewControllerToken(ctx context.Context, opts ...Option) (Token, error)

	// GetAudience returns the audience the OIDC tokens issued representing
	// ServiceAccounts should have. This is usually a string that represents
	// the cloud provider's STS service, or some entity in the provider for
	// which the OIDC tokens are targeted to.
	GetAudience(ctx context.Context, serviceAccount corev1.ServiceAccount) (string, error)

	// GetIdentity takes a ServiceAccount and returns the identity which the
	// ServiceAccount wants to impersonate, by looking at annotations.
	GetIdentity(serviceAccount corev1.ServiceAccount) (string, error)

	// NewToken takes a ServiceAccount and its OIDC token and returns a token
	// that can be used to authenticate with the cloud provider. The OIDC token is
	// the JWT token that was issued for the ServiceAccount by the Kubernetes API.
	// The implementation should exchange this token for a cloud provider access
	// token through the provider's STS service.
	NewTokenForServiceAccount(ctx context.Context, oidcToken string,
		serviceAccount corev1.ServiceAccount, opts ...Option) (Token, error)

	// GetAccessTokenOptionsForArtifactRepository returns the options that must be
	// passed to the provider to retrieve access tokens for an artifact repository.
	GetAccessTokenOptionsForArtifactRepository(artifactRepository string) ([]Option, error)

	// ParseArtifactRepository parses the artifact repository to verify
	// it's a valid repository for the provider. As a result, it returns
	// the input required for the provider to issue registry credentials.
	// This input is included in the cache key for the issued credentials.
	ParseArtifactRepository(artifactRepository string) (string, error)

	// NewArtifactRegistryCredentials takes the registry input extracted by
	// ParseArtifactRepository() and an access token and returns credentials
	// that can be used to authenticate with the registry.
	NewArtifactRegistryCredentials(ctx context.Context, registryInput string,
		accessToken Token, opts ...Option) (*ArtifactRegistryCredentials, error)

	// GetAccessTokenOptionsForCluster returns the options that must be
	// passed to the provider to retrieve access tokens for a cluster.
	// More than one access token may be required depending on the
	// provider, with different options (e.g. scope). Hence the return
	// type is a slice of slices.
	GetAccessTokenOptionsForCluster(cluster string) ([][]Option, error)

	// NewRESTConfig returns a RESTConfig that can be used to authenticate
	// with the Kubernetes API server. The access tokens are used for looking
	// up connection details like the API server address and CA certificate
	// data, and for accessing the cluster API server itself via the IAM
	// system of the cloud provider. If it's just a single token or multiple,
	// it depends on the provider.
	NewRESTConfig(ctx context.Context, accessTokens []Token, opts ...Option) (*RESTConfig, error)
}
```

The file `auth/options.go` would contain the following options:

```go
package auth

// Options contains options for configuring the behavior of the provider methods.
// Not all providers/methods support all options.
type Options struct {
	Client          client.Client
	Cache           *cache.TokenCache
	ServiceAccount  *client.ObjectKey
	InvolvedObject  cache.InvolvedObject
	Audiences       []string
	Scopes          []string
	STSRegion       string
	STSEndpoint     string
	ProxyURL        *url.URL
	ClusterResource string
	ClusterAddress  string
	CAData          string
	AllowShellOut   bool
}

// WithServiceAccount sets the ServiceAccount reference for the token
// and a controller-runtime client to fetch the ServiceAccount and
// create an OIDC token for it in the Kubernetes API.
func WithServiceAccount(saRef client.ObjectKey, client client.Client) Option {
	// ...
}

// WithCache sets the token cache and the involved object for recording events.
func WithCache(cache cache.TokenCache, involvedObject cache.InvolvedObject) Option {
	// ...
}

// WithAudiences sets the audiences for the Kubernetes ServiceAccount token.
func WithAudiences(audiences ...string) Option {
	// ...
}

// WithScopes sets the scopes for the token.
func WithScopes(scopes ...string) Option {
	// ...
}

// WithSTSRegion sets the region for the STS service (some cloud providers
// require a region, e.g. AWS).
func WithSTSRegion(stsRegion string) Option {
	// ...
}

// WithSTSEndpoint sets the endpoint for the STS service.
func WithSTSEndpoint(stsEndpoint string) Option {
	// ...
}

// WithProxyURL sets a *url.URL for an HTTP/S proxy for acquiring the token.
func WithProxyURL(proxyURL url.URL) Option {
	// ...
}

// WithCAData sets the CA data for credentials that require a CA,
// e.g. for Kubernetes REST config.
func WithCAData(caData string) Option {
	// ...
}

// WithClusterResource sets the cluster resource for creating a REST config.
// Must be the fully qualified name of the cluster resource in the cloud
// provider API.
func WithClusterResource(clusterResource string) Option {
	// ...
}

// WithClusterAddress sets the cluster address for creating a REST config.
// This address is used to select the correct cluster endpoint and CA data
// when the provider has a list of endpoints to choose from, or to simply
// validate the address against the cluster resource when the provider
// returns a single endpoint. This is optional, providers returning a list
// of endpoints will select the first one if no address is provided.
func WithClusterAddress(clusterAddress string) Option {
	// ...
}

// WithAllowShellOut allows the provider to shell out to binary tools
// for acquiring controller tokens. MUST be used only by the Flux CLI,
// i.e. in the github.com/fluxcd/flux2 Git repository.
func WithAllowShellOut() Option {
	// ...
}
```

The `auth/aws/aws.go`, `auth/azure/azure.go` and
`auth/gcp/gcp.go` files would contain the implementations for
the respective cloud providers:

```go
package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
)

const ProviderName = "aws"

type Provider struct{}

type Token struct{ types.Credentials }

// GetDuration implements auth.Token.
func (t *Token) GetDuration() time.Duration {
	return time.Until(*t.Expiration)
}

type credentialsProvider struct {
	opts []auth.Option
}

// NewCredentialsProvider creates an aws.CredentialsProvider for the aws provider.
func NewCredentialsProvider(opts ...auth.Option) aws.CredentialsProvider {
	return &credentialsProvider{opts}
}

// Retrieve implements aws.CredentialsProvider.
func (c *credentialsProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	// Use auth.GetToken() to get the token.
}
```

```go
package azure

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

const ProviderName = "azure"

type Provider struct{}

type Token struct{ azcore.AccessToken }

// GetDuration implements auth.Token.
func (t *Token) GetDuration() time.Duration {
	return time.Until(t.ExpiresOn)
}

type tokenCredential struct {
	opts []auth.Option
}

// NewTokenCredential creates an azcore.TokenCredential for the azure provider.
func NewTokenCredential(opts ...auth.Option) azcore.TokenCredential {
	return &tokenCredential{opts}
}

// GetToken implements azcore.TokenCredential.
// The options argument is ignored, any options should be
// specified in the constructor.
func (t *tokenCredential) GetToken(ctx context.Context, _ policy.TokenRequestOptions) (azcore.AccessToken, error) {
	// Use auth.GetToken() to get the token.
}
```

```go
package gcp

import (
	"golang.org/x/oauth2"
)

const ProviderName = "gcp"

type Provider struct {}

type Token struct{ oauth2.Token }

// GetDuration implements auth.Token.
func (t *Token) GetDuration() time.Duration {
	return time.Until(t.Expiry)
}

type tokenSource struct {
	ctx context.Context
	opts []auth.Option
}

// NewTokenSource creates an oauth2.TokenSource for the gcp provider.
func NewTokenSource(ctx context.Context, opts ...auth.Option) oauth2.TokenSource {
	return &tokenSource{ctx, opts}
}

// Token implements oauth2.TokenSource.
func (t *tokenSource) Token() (*oauth2.Token, error) {
	// Use auth.GetToken() to get the token.
}

var gkeMetadata struct {
	projectID      string
	location       string
	name           string
	mu             sync.Mutex
	loaded         bool
}
```

As detailed above, each cloud provider implementation defines a simple wrapper
around the cloud provider access token type. This wrapper implements the
`auth.Token` interface, which is essentially the method `GetDuration()`
for the cache library to manage the token lifetime. The wrappers also contain
a helper function to create a token source for the respective cloud provider
SDKs. These methods have different names and signatures because the cloud provider
SDKs are different and have different types, but they all implement the same
concept of a token source.

The `aws` provider needs to read the environment variable `AWS_REGION` for
configuring the STS client. Even though a specific STS endpoint may be
configured, the AWS SDKs require the region to be set regardless. This
variable is usually set automatically in EKS pods, and can be manually set
by users otherwise (e.g. in Fargate pods).

An important detail to take into account in the `azure` provider implementation
is using our custom implementation of `azidentity.NewDefaultAzureCredential()`
found in kustomize-controller for SOPS decryption. This custom implementation
avoids shelling out to the Azure CLI, which is something we strive to avoid in
the Flux codebase. This is important because today we are doing this in a few
APIs but not others, so it will be a significant improvement to implement this
in a single place and use it everywhere.

The `gcp` provider needs to load the cluster metadata from the `gke-metadata-server`
in order to create tokens. This must be done lazily when the first token is
requested, and there's a very important reason for this: if this was done on
the controller startup, the controller would crash when running outside GKE and
enter `CrashLoopBackOff` because the `gke-metadata-server` would never be
available. This is a very important detail that must be taken into account when
implementing the `gcp` provider. The cluster metadata doesn't change during the
lifetime of the controller pod, so we use a `sync.Mutex` and `bool` to load it
only once into a package variable.

When not running in GKE, the `gcp` provider would use the following annotation
in the `ServiceAccount` to identify the Workload Identity Provider resource
for use with Workload Identity Federation:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-service-account
  namespace: my-namespace
  annotations:
    gcp.auth.fluxcd.io/workload-identity-provider: projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/POOL_ID/providers/PROVIDER_ID
```

#### Cache Key

The cache key *MUST* include *ALL* the inputs specified for acquiring the
temporary credentials, as they all obviously influence how the credentials
are created.

##### Format

The cache key would be the SHA256 hash of the following multi-line strings:

Single-tenant/controller-level access token cache key:

```
provider=<cloud-provider-name>
scopes=<comma-separated-scopes>
stsRegion=<sts-region>
stsEndpoint=<sts-endpoint>
proxyURL=<proxy-url>
caData=<ca-data>
```

Multi-tenant/object-level access token cache key:

```
provider=<cloud-provider-name>
providerIdentity=<cloud-provider-identity>
serviceAccountName=<service-account-name>
serviceAccountNamespace=<service-account-namespace>
serviceAccountTokenAudiences=<comma-separated-audiences>
scopes=<comma-separated-scopes>
stsRegion=<sts-region>
stsEndpoint=<sts-endpoint>
proxyURL=<proxy-url>
caData=<ca-data>
```

Artifact registry credentials:

```
accessTokenCacheKey=sha256(<access-token-cache-key>)
artifactRepositoryCacheKey=<'gcp'-for-gcp|registry-region-for-aws|registry-host-for-azure>
```

REST config:

```
accessToken1CacheKey=sha256(<cache-key-for-access-token-1>)
...
accessTokenNCacheKey=sha256(<cache-key-for-access-token-N>)
cluster=<cluster-resource-name>
address=<cluster-api-server-address>
```

##### Security Considerations and Controls

As mentioned previously, a `ServiceAccount` must have permission to impersonate the
identity it is configured to impersonate. Once a token for the impersonated identity
is issued, that token would be valid for a while even if immediately after issuing it
the `ServiceAccount` loses permission to impersonate that identity. In our cache key
design, the token would remain available for the `ServiceAccount` to use until it
expires. If the impersonation permission was revoked to mitigate an attack, the
attacker could still get a valid token from the cache for a while after the
revocation, and hence still exercise the permissions they had prior to the revocation.

There are a few mitigations for this scenario:

* Users that revoke impersonation permissions for a `ServiceAccount` must also
  change the annotations of the `ServiceAccount` to impersonate a different identity,
  or delete the `ServiceAccount` altogether, or restart the Flux controllers so the
  cache is purged. Any of these actions would effectively prevent the attack, but
  they represent an additional step after revoking the impersonation permission.

* In the Flux controllers users can specify the `--token-cache-max-duration` flag,
  which can be used to limit the maximum duration for which a token can be cached.
  By reducing the default maximum duration of one hour to a smaller value, users can
  limit the time window during which a token would be available for a `ServiceAccount`
  to use after losing permission to impersonate the identity.

* Disable cache entirely by setting the flag `--token-cache-max-size=0`, or removing
  this flag altogether since the default is already zero i.e. no tokens are cached
  in the Flux controller. This mitigation is in case your security requirements are
  extreme and you want to avoid any risk of such an attack. This mitigation is the
  most effective, but it comes with the cost of many API calls to issue tokens in
  the cloud provider, which could result in a performance bottleneck and/or
  throttling/rate-limiting, as tokens would have to be issued for every
  reconciliation.

A similar situation could occur in the single-tenant scenario, when the permission
to impersonate the configured identity is revoked from the controller `ServiceAccount`.
In this case, the attacker would have access to the cloud provider resources that
the controller had access to prior to the revocation of the impersonation permission.
Most of the mitigations mentioned above apply to this scenario as well, except for
the one that involves changing the annotations of the `ServiceAccount` to impersonate
a different identity or deleting the `ServiceAccount` altogether, as the controller
`ServiceAccount` should not be deleted. The best mitigation in this case is to restart
the Flux controllers so the cache is purged.

### Library Integration

When reconciling an object, the controller must use the `auth.GetToken()`
function passing a `controller-runtime` client that has permission to create
`ServiceAccount` tokens in the Kubernetes API, the desired cloud provider by name,
and all the remaining options according to the configuration of the controller and
of the object. The provider names match the ones used for `spec.provider` in the Flux
APIs, i.e. `aws`, `azure` and `gcp`.

Because different cloud providers have different ways of representing their access
tokens (e.g. Azure and GCP tokens are a single opaque string while AWS has three
strings: access key ID, secret access key and token), consumers of the
`auth.Token` interface would need to cast it to `*<provider>.Token`.

The following subsections show details of how the integration would look like.

#### `GitRepository` and `ImageUpdateAutomation` APIs

For these APIs the only provider we have so far that supports workload identity
is `azure`. In this case we would simply replace `AzureOpts []azure.OptFunc` in
the `fluxcd/pkg/git.ProviderOptions` struct with `[]fluxcd/pkg/auth.Option`
and would modify `fluxcd/pkg/git.GetCredentials()` to use `auth.GetToken()`.
The token interface would be cast to `*azure.Token` and the token string would be
assigned to `fluxcd/pkg/git.Credentials.BearerToken`. A `GitRepository` object
configured with the `azure` provider and a `ServiceAccount` would then go through
this code path.

#### `OCIRepository`, `ImageRepository`, `ImagePolicy`, `HelmRepository` and `HelmChart` APIs

The `HelmRepository` API only supports a cloud provider for OCI repositories, so
for all these APIs we would only need to support OCI authentication.

All these APIs currently use `*fluxcd/pkg/oci/auth/login.Manager` to get the
container registry credentials. The new library would replace this library
entirely, as it mostly handles single-tenant workload identity. The new library
covers both single-tenant and multi-tenant workload identity, so it would be
a drop-in replacement for the `login.Manager`.

In the case of the source-controller APIs, all of them use the function `OIDCAuth()`
from the internal package `internal/oci`. We would replace the use of `login.Manager`
with `auth.GetToken()` in this function. The token interface would
be cast to `*auth.RegistryCredentials` and then fed to `authn.FromConfig()`
from the package `github.com/google/go-containerregistry/pkg/authn`.

In the case of `ImageRepository` and `ImagePolicy`, we would replace `login.Manager` with
`auth.GetToken()` in the `setAuthOptions()` method of the
`ImageRepositoryReconciler`, cast the token to `*auth.RegistryCredentials`
and then feed it to `authn.FromConfig()`.

The beauty of this particular integration is that here we no longer require
branching code paths for each cloud provider, we would just need to configure
the options for the `auth.GetToken()` function and the library would take
care of the rest.

#### `Bucket` API

##### Provider `aws`

A `Bucket` object configured with the `aws` provider and a `ServiceAccount` would
cause the internal `minio.MinioClient` of source-controller to be created with the
following new options:

* `minio.WithTokenClient(controller-runtime/pkg/client.Client)`
* `minio.WithTokenCache(*fluxcd/pkg/cache.TokenCache)`

The constructor would then use `auth.GetToken()` to get the
cloud provider access token. When doing so, the `minio.MinioClient` would
cast the token interface to `*aws.Token` and feed it to `credentials.NewStatic()`
from the package `github.com/minio/minio-go/v7/pkg/credentials`.

##### Provider `azure`

A `Bucket` object configured with the `azure` provider and a `ServiceAccount`
would cause the internal `azure.BlobClient` of source-controller to be created
with the following new options:

* `azure.WithTokenClient(controller-runtime/pkg/client.Client)`
* `azure.WithTokenCache(*fluxcd/pkg/cache.TokenCache)`
* `azure.WithServiceAccount(controller-runtime/pkg/client.ObjectKey)`
* `azure.WithInvolvedObject(*fluxcd/pkg/cache.InvolvedObject)`

The constructor would then use `azure.NewTokenCredential()` to feed this
token credential to `azblob.NewClient()`.

##### Provider `gcp`

A `Bucket` object configured with the `gcp` provider and a `ServiceAccount`
would cause the internal `gcp.GCSClient` of source-controller to be created
with the following new options:

* `gcp.WithTokenClient(controller-runtime/pkg/client.Client)`
* `gcp.WithTokenCache(*fluxcd/pkg/cache.TokenCache)`
* `gcp.WithServiceAccount(controller-runtime/pkg/client.ObjectKey)`
* `gcp.WithInvolvedObject(*fluxcd/pkg/cache.InvolvedObject)`

The constructor would then use `gcp.NewTokenSource()` to feed this token
source to the `option.WithTokenSource()` and pass it to
`cloud.google.com/go/storage.NewClient()`.

#### `Kustomization` API (SOPS Decryption)

The `Kustomization` API uses Key Management Services (KMS) for decrypting
SOPS secrets. The internal packages `internal/decryptor` and `internal/sops`
of kustomize-controller already use interfaces compatible with the new
library in the case of `aws` and `azure`, i.e. `*awskms.CredentialsProvider`
and `*azkv.TokenCredential` respectively, so we could easily use the helper
functions for creating the respective token sources to configure the KMS
credentials for SOPS. This is thanks to the respective SOPS libraries
`github.com/getsops/sops/v3/kms` and `github.com/getsops/sops/v3/azkv`.
For GCP we can introduce the equivalent interface that was recently added
in [this](https://github.com/getsops/sops/pull/1794/files) pull request.
This new interface introduced in SOPS upstream can also be used for the
current JSON credentials method that we use via
`google.CredentialsFromJSON().TokenSource`. This would allow us to use only
the respective token source interfaces for all three providers when using
either workload identity or secrets.

#### `Kustomization` and `HelmRelease` APIs (Remote Cluster Access)

The kustomize-controller should fetch a `*rest.Config` from the `auth`
package and feed it to `runtime/client.WithKubeConfig()` for creating
a `runtime/client.(*Impersonator)` with the configured authentication.

The helm-controller should fetch a `*rest.Config` from the `auth`
package and feed it to the internal `kube.NewMemoryRESTClientGetter()`,
just like it does for the secret-based alternative.

#### `Provider` API

The constructor of the internal `notifier.Factory` of notification-controller
would now accept the following new options:

* `notifier.WithTokenClient(controller-runtime/pkg/client.Client)`
* `notifier.WithTokenCache(*fluxcd/pkg/cache.TokenCache)`
* `notifier.WithServiceAccount(controller-runtime/pkg/client.ObjectKey)`
* `notifier.WithInvolvedObject(*fluxcd/pkg/cache.InvolvedObject)`

The cloud provider types that support workload identity would then use these
options. See the following subsections for details.

##### Type `azuredevops`

The `notifier.NewAzureDevOps()` constructor would use the existing and new
options to call `auth.GetToken()` and use it to get the cloud
provider access token. When doing so, the `notifier.AzureDevOps` would cast
the token interface to `*azure.Token` and feed the token string to
`NewPatConnection()` from the package
`github.com/microsoft/azure-devops-go-api/azuredevops/v6`.

##### Type `azureeventhub`

The `notifier.NewAzureEventHub()` constructor would use the existing and new
options to call `auth.GetToken()` and use it to get the cloud
provider access token. When doing so, the `notifier.AzureEventHub` would cast
the token interface to `*azure.Token` and feed the token string to `newJWTHub()`.

##### Type `googlepubsub`

The `notifier.NewGooglePubSub()` constructor would use the existing and new
options to call `gcp.NewTokenSource()` and feed this token source to the
`option.WithTokenSource()` and pass it to `cloud.google.com/go/pubsub.NewClient()`.

## Implementation History

* In Flux 2.6 object-level workload identity was introduced for the
  OCI artifact APIs, i.e. `OCIRepository`, `ImageRepository`, `ImagePolicy`,
  `HelmRepository` and `HelmChart`, as well as for SOPS decryption
  in the `Kustomization` API and Azure Event Hubs in the
  `Provider` API.

<!--
Major milestones in the lifecycle of the RFC such as:
- The first Flux release where an initial version of the RFC was available.
- The version of Flux where the RFC graduated to general availability.
- The version of Flux where the RFC was retired or superseded.
-->
