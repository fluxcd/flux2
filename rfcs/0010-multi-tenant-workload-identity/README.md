# RFC-0010 Multi-Tenant Workload Identity

**Status:** implementable

<!--
Status represents the current state of the RFC.
Must be one of `provisional`, `implementable`, `implemented`, `deferred`, `rejected`, `withdrawn`, or `replaced`.
-->

**Creation date:** 2025-02-22

**Last update:** 2025-02-27

## Summary

In this RFC we aim to add support for multi-tenant workload identity in Flux,
i.e. the ability to specify at the object-level which set of cloud provider
permissions must be used for interacting with the respective cloud provider
on behalf of the reconciliation of the object. In this process, credentials
must be obtained automatically, i.e. this feature must not involve the use
of secrets. This would be useful in a number of Flux APIs that need to
interact with cloud providers, spanning all the Flux controllers except
for helm-controller.

## Motivation

Flux has strong multi-tenancy features. For example, the `Kustomization` and
`HelmRelease` APIs support the field `spec.serviceAccountName` for specifying
the Kubernetes `ServiceAccount` to impersonate when interacting with the
Kubernetes API on behalf of a tenant, e.g. when applying resources. This
allows tenants to be constrained under the Kubernetes RBAC permissions
granted to this `ServiceAccount`, and therefore have access only to the
specific subset of resources they are allowed to manage.

Besides the Kubernetes API, Flux also interacts with cloud providers, e.g.
container registries, object storage, pub/sub services, etc. In these cases,
Flux currently supports basically two modes of authentication:

- *Secret-based multi-tenant authentication*: Objects have the field
  `spec.secretRef` for specifying the Kubernetes `Secret` containing the
  credentials to use when interacting with the cloud provider. This is
  similar to the `spec.serviceAccountName` field, but for cloud providers.
  The problem with this approach is that secrets are a security risk and
  operational burden, as they must be managed and rotated.
- *Workload identity-based single-tenant authentication*: Flux offers
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

It's not a goal to provide multi-tenant workload identity *federation* support.
The (small) difference between workload identity and workload identity federation
is that the former assumes that the workloads are running inside the cloud
environment, while the latter assumes that the workloads are running outside
the cloud environment. All the major cloud providers support both, as the majority
of the underlying technology is the same, but the configuration is slightly
different. Because the differences are small we may consider workload identity
federation support in the future, but it's not a goal for this RFC.

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
    azure.workload.identity/tenant-id: 72f988bf-86f1-41af-91ab-2d7cd011db47 # azure tenant for the cluster
    azure.workload.identity/client-id: d6e4fc00-c5b2-4a72-9f84-6a92e3f06b08 # client ID for my tenant A
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
    azure.workload.identity/tenant-id: 72f988bf-86f1-41af-91ab-2d7cd011db47 # azure tenant for the cluster
    azure.workload.identity/client-id: 4a7272f9-f186-41af-9f84-6a92e32d7cd0 # client ID for my tenant B
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

### Alternatives

Alternatives to consider in this solution are around Kubernetes RBAC for creating
`ServiceAccount` tokens cluster-wide, as it has a potential impact on the security
posture of Flux.

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
Kuberentes services from the cloud providers.

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
the cloud provider is then able to allow Kuberentes `ServiceAccounts` to access its resources.
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
`ServiceAccount`.

Because the cloud provider needs to support this impersonation permission, some
cloud providers go further and even remove the impersonation requirement, by
allowing permissions to be granted directly to `ServiceAccounts` (if it needs to
support granting the impersonation permission, then it can probably also support
granting any other permissions depending on the implementation). GCP for example
has implemented this feature recently, calling it "direct access". When using
this feature, a GCP IAM Service Account is not required, i.e. GCP IAM
permissions can be granted directly to Kubernetes `ServiceAccounts`. This is
a significant improvement in the user experience, as it significantly reduces
the required configuration steps.

In sight of the technical background presented above, our proposal becomes simple.
The only solution to support multi-tenant workload identity at the object-level for
the Flux APIs is to associate the Flux objects with Kubernetes `ServiceAccounts`.
We propose to build the `ServiceAccount` token creation and exchange logic into
the Flux controllers through a library in the `github.com/fluxcd/pkg` repository.

### API Changes

For every Flux API that needs to interact with cloud providers, we propose adding
the field `spec.serviceAccountName` for specifying the Kubernetes `ServiceAccount`
on the same namespace of the object that must be used for getting access to the
respective cloud resources. This field would be optional, and when not present the
behavior would be the same as before this feature, i.e. the feature only activates
when the field is present.

In addition to this field, we propose adding the field `spec.sts.endpoint` for
specifying the STS endpoint to use for exchanging the Kubernetes token for the
cloud provider access token. This field would be optional, and when not present
the behavior would be using the default STS endpoint for the cloud provider.
At the time of writing this RFC, not all cloud providers support custom STS
endpoints, so this field would be implemented only for the cloud providers
that support it. This field is notably already present in the `Bucket` API,
and is used for both EKS single-tenant workload identity in the `aws` bucket
provider, and for LDAP authentication in the `generic` bucket provider.

Because in the specific case of the `Bucket` API the `spec.sts` field has
additional use cases, it has an additional field besides `spec.sts.endpoint`
which is `spec.sts.provider`, and when `spec.sts`, which is optional, is
specified, then both `spec.sts.provider` and `spec.sts.endpoint` must be
specified. This is not necessary for the other Flux APIs, so we propose
only the `spec.sts.endpoint` field at the moment. So `spec.sts` would be
optional and `spec.sts.endpoint` would be mandatory when `spec.sts` is
specified.

When talking to a Security Token Service for exchanging tokens it's possible
to use an HTTP/S proxy. Most Flux APIs already support the field
`spec.proxySecretRef` for accessing the respective cloud resources, and some
already use it for talking to the STS service in the single-tenant version
of workload identity as well, e.g. `Bucket`. We propose using this field for
specifying the proxy to use when talking to the STS service also in the
multi-tenant version of workload identity. Because we would introduce the
field `spec.sts.endpoint`, we propose introducing also the field
`spec.sts.proxySecretRef` with the effect of overriding the proxy specified
in `spec.proxySecretRef` when present (this field would be optional).

### Workload Identity Library

We propose creating a Go package `github.com/fluxcd/pkg/workloadidentity`
for implementing the workload identity logic in the Flux controllers.

This package should not be under `github.com/fluxcd/pkg/runtime` because
helm-controller does not need this feature, and a significant amount of
dependencies for talking to cloud providers would be pulled in unnecessarily.

The directory structure would look like this:

```shell
.
└── workloadidentity
    ├── aws
    │   └── aws.go
    ├── azure
    │   └── azure.go
    ├── gcp
    │   └── gcp.go
    ├── interfaces
    │   └── interfaces.go
    ├── providers.go
    └── token_store.go
```

The file `workloadidentity/providers.go` would contain the `providers` map:

```go
var providers = map[string]interfaces.Provider{
	aws.ProviderName:   &aws.Provider{},
	azure.ProviderName: &azure.Provider{},
	gcp.ProviderName:   &gcp.Provider{},
}
```

The file `workloadidentity/token_store.go` would contain the following abstractions:

```go
// TokenStore provides tokens retrieved via workload identity.
type TokenStore struct {
	client client.Client
	cache  *cache.TokenCache
}

// Option is a functional option for configuring the TokenStore.
type Option func(*TokenStore)

// WithCache sets the token cache.
func WithCache(cache *cache.TokenCache) Option {
	// ...
}

// NewTokenStore creates a new TokenStore.
func NewTokenStore(client client.Client, opts ...Option) *TokenStore {
	// ...
}

// GetToken takes a provider name and a ServiceAccount reference and returns an
// an access token issued on behalf of the ServiceAccount for accessing resources
// in the given cloud provider.
func (t *TokenStore) GetToken(
	ctx context.Context,
	providerName string,
	saRef client.ObjectKey,
	opts ...interfaces.Option) (interfaces.Token, error) {

	// 1. Get the provider by name from the providers map defined in providers.go.
	// 2. Get the provider audience for creating the Kubernetes OIDC token.
	// 3. Get the ServiceAccount using the configured controller-runtime client.
	// 4. Get the identity for the ServiceAccount from the provider.
	// 5. Build the cache key using the ServiceAccount reference, provider name and identity.
	// 6. Get the token from the cache. If present return it, otherwise continue.
	// 7. Create an OIDC token for the ServiceAccount in the Kubernetes API using the provider audience.
	// 8. Create a provider access token for the ServiceAccount using the Kubernetes OIDC token.
	// 9. Add the provider access token to the cache and return it.
}
```

The file `workloadidentity/interfaces/interfaces.go` would contain the following interfaces:

```go
// IdentityDirectAccess is the identity string that represents direct access
// to the cloud provider. This is used when the ServiceAccount does not have
// an identity configured for impersonation. This is only possible when the
// cloud provider supports granting permissions directly to the ServiceAccount
// (for example, GCP). This is a feature that makes the workload identity
// setup simpler.
const IdentityDirectAccess = "DirectAccess"

// Provider contains the logic to retrieve an access token for a cloud
// provider from a ServiceAccount (OIDC/JWT) token.
type Provider interface {
	// GetAudience returns the audience the OIDC tokens issued representing
	// ServiceAccounts should have. This is usually a string that represents
	// the cloud provider's STS service, or some entity in the provider that
	// represents a domain for which the OIDC tokens are targeted to.
	GetAudience(ctx context.Context) (string, error)

	// GetIdentity takes a ServiceAccount and returns the identity which the
	// ServiceAccount wants to impersonate, usually by looking at annotations.
	// When there is no identity configured for impersonation this method
	// should return IdentityDirectAccess, representing the fact that the
	// ServiceAccount's own name/reference is what access should be evaluated
	// for in the cloud provider. Direct access may not be supported by all
	// providers.
	GetIdentity(sa *corev1.ServiceAccount) (string, error)

	// NewToken takes a ServiceAccount and its OIDC token and returns a token
	// that can be used to authenticate with the cloud provider. The OIDC token is
	// the JWT token that was issued for the ServiceAccount by the Kubernetes API.
	// The implementation should exchange this token for a cloud provider access
	// token.
	NewToken(ctx context.Context, oidcToken string,
		sa *corev1.ServiceAccount, opts ...Option) (Token, error)
}

// Token is an interface that represents an access token that can be used
// to authenticate with a cloud provider. The only common method is to get the
// duration of the token, because different providers may have different ways to
// represent the token. For example, Azure and GCP use an opaque string token,
// while AWS uses the pair of access key id and secret access key. Consumers of
// this token should know what type to cast this interface to.
type Token interface {
	// GetDuration returns the duration for which the token is valid relative to
	// approximately time.Now(). This is used to determine when the token should
	// be refreshed.
	GetDuration() time.Duration
}

// Option is a functional option for configuring the behavior of NewToken.
type Option func(*Options)

// Options contains options for configuring the behavior of NewToken. Supporting each
// option here depends on the Provider, i.e. some providers may not support certain
// options.
type Options struct {
	Endpoint       string
	HTTPClient     *http.Client
	InvolvedObject *cache.InvolvedObject
}

// Apply applies the given slice of Option(s) to the Options struct.
func (o *Options) Apply(opts ...Option) {
	// ...
}

// WithEndpoint sets the endpoint for the STS service. Supported only by AWS.
func WithEndpoint(endpoint string) Option {
	// ...
}

// WithProxyURL sets an *http.Client configured with an HTTP/S proxy for talking to the STS service.
func WithProxyURL(proxyURL *url.URL) Option {
	// ...
}

// WithInvolvedObject sets the object involved in the token operation for
// recording cache events.
func WithInvolvedObject(kind, name, namespace string) Option {
	// ...
}
```

#### Cache Key

The cache key must include three components: the `ServiceAccount` reference, the
provider name, and the identity. The identity is the string that represents the
cloud provider identity which the `ServiceAccount` will impersonate. When there is
no identity configured for impersonation, the identity should be
`interfaces.IdentityDirectAccess`.

The reason for including both the `ServiceAccount` and the identity in the cache
key is to establish the fact that the `ServiceAccount` had permission to impersonate
the identity at the time when the token was issued. This is very important because
otherwise, if including only the identity, a malicious actor could specify any
identity in their `ServiceAccount` and get a token cached for that identity even
if their `ServiceAccount` does not have permission to impersonate that identity.

We also need to include the identity in the cache key because otherwise, if including
only the `ServiceAccount`, changes to the `ServiceAccount` annotations to impersonate
a different identity would not cause a new token impersonating the new identity to be
created.

##### Security Considerations

As mentioned above, a `ServiceAccount` must have permission to impersonate the
identity it is configured to impersonate. Once a token for the impersonated identity
is issued, that token will be valid for a while even if immediately after issuing it
the `ServiceAccount` loses permission to impersonate that identity. In our cache key
design, the token would remain available for the `ServiceAccount` to use until it
expires.

There are a few mitigations for this attack:

1. Users that revoke impersonation permissions for a `ServiceAccount` should also
  restart the Flux controllers which are potentially caching tokens for that
  `ServiceAccount`. This would clear the cache and effectively mitigate the attack.

2. In the Flux controllers users can specify the `--token-cache-max-duration` flag,
  which can be used to limit the maximum duration for which a token can be cached.
  By reducing the default maximum duration of one hour to a smaller value, users can
  limit the time window for which a token would be available for a `ServiceAccount`
  to use after losing permission to impersonate the identity.

3. Disable cache entirely by setting the flag `--token-cache-max-size=0`.

### Library Integration

An instance of `workloadidentity.TokenStore` would be created in the `main.go` file
of the Flux controllers, and used in the reconciliation loop for getting the cloud
provider access tokens.

Because different cloud providers have different ways to represent the access tokens,
e.g. Azure and GCP tokens are a single opaque string and AWS tokens are a pair
(access key id and secret access key), instances of the `interfaces.Token`
interface would need to be casted to concrete instances of `*<provider>.Token`.

#### `GitRepository` API

<!-- TODO: for azure -->

<!-- azure is the only git provider we have that
supports workload identity, github and generic don't support it -->

#### `OCIRepository`, `HelmRepository` and `ImageRepository` APIs

<!-- TODO: for aws, azure and gcp -->

<!-- all these APIs use/should use the same container registry libraries -->
<!-- HelmRepository only supports the spec.provider field whe spec.type is oci -->

#### `Bucket` API

##### Provider `aws`

Consider the following example for the `Bucket` API:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: Bucket
metadata:
  name: tenant-a-bucket
  namespace: tenant-a
spec:
  ...
  provider: aws
  serviceAccountName: tenant-a-gcs-sa
```

This `Bucket` object would cause the `minio.MinioClient` of source-controller
to be created with the option `minio.WithTokenStore()`. The token store would
be used internally by the `minio.MinioClient` for getting the cloud provider
access token. When doing so, the `minio.MinioClient` would cast the token
interface returned by `TokenStore.GetToken()` to `*aws.Token` and feed it
to `credentials.NewStaticV4()` from the package
`github.com/minio/minio-go/v7/pkg/credentials`, similarly to what the function
[`newCredsFromSecret()`](https://github.com/fluxcd/source-controller/blob/8e9e3a7d540db93f8da7655964a680f99cb1db71/pkg/minio/minio.go#L145-L159)
does.

<!-- TODO: for azure and gcp -->

#### `Kustomization` API

<!-- TODO: aws, azure and gcp -->

<!-- for sops decryption using key management services -->

#### `Provider` API

<!-- TODO: for azureeventhub and googlepubsub -->

## Implementation History

<!--
Major milestones in the lifecycle of the RFC such as:
- The first Flux release where an initial version of the RFC was available.
- The version of Flux where the RFC graduated to general availability.
- The version of Flux where the RFC was retired or superseded.
-->
