# RFC-XXXX Vendor-Agnostic Short-Lived Credentials

**Status:** provisional

<!--
Status represents the current state of the RFC.
Must be one of `provisional`, `implementable`, `implemented`, `deferred`, `rejected`, `withdrawn`, or `replaced`.
-->

**Creation date:** 2026-02-01

**Last update:** 2026-02-01

## Summary

In [RFC-0010](https://github.com/fluxcd/flux2/tree/main/rfcs/0010-multi-tenant-workload-identity)
we implemented object-level workload identity for cloud providers leveraging
`ServiceAccount` tokens. This RFC proposes extending Flux with vendor-agnostic
short-lived credentials based on open standards (OIDC and SPIFFE), enabling
workload identity authentication with third-party services that are not
cloud-provider-managed, such as self-hosted container registries (e.g.
[Zot](https://zotregistry.dev/), [Harbor](https://goharbor.io/)). We propose
introducing a new spec field `.spec.credential` to the `OCIRepository` and
`ImageRepository` APIs, as a structured object with a `type` sub-field
supporting three credential types: `ServiceAccountToken`, `SpiffeJWT` and
`SpiffeCertificate`.
Once demand for other third-party services supporting these credential
standards show up for other Flux APIs, this pattern can be extended further.

## Motivation

RFC-0010 introduced multi-tenant workload identity for cloud providers (AWS,
Azure, GCP) by associating Flux objects with Kubernetes `ServiceAccounts`.
However, the current workload identity support is limited to cloud-provider
token exchange through their respective Security Token Services (STS). There
is a growing need for Flux to support short-lived credentials for third-party
services that implement open standards directly, without depending on any
cloud provider.

Several real-world use cases motivate this work:

- The `Kustomization` and `HelmRelease` APIs already support a vendor-agnostic
  form of workload identity through the `generic` provider inside
  `.spec.kubeConfig.configMapRef` -> `.data.provider`. This generic provider
  issues a `ServiceAccount` token and uses it directly for authentication with
  remote Kubernetes clusters configured with external OIDC authentication.
  However, this pattern has not been extended to other Flux APIs.
- Container registries such as [Zot](https://github.com/project-zot/zot/pull/3711)
  and [Harbor](https://github.com/goharbor/harbor/issues/22027) are implementing
  OIDC workload identity federation, allowing workloads to authenticate using
  Kubernetes `ServiceAccount` tokens directly, without cloud provider intermediaries.
- The [SPIFFE](https://spiffe.io/) standard provides an alternative identity
  framework that allows workloads to be identified independently of
  `ServiceAccounts`, enabling use cases where the identity is the Flux object
  itself (kind, name, namespace) rather than a Kubernetes `ServiceAccount`.

### Goals

- Provide vendor-agnostic short-lived credential support in Flux, starting with
  the `OCIRepository` and `ImageRepository` APIs.
- Support Kubernetes `ServiceAccount` tokens as OIDC credentials for
  authenticating with third-party services that support OIDC federation.
- Support SPIFFE SVIDs (both JWT and x509 certificate) as credentials for
  authenticating with third-party services that support SPIFFE.
- Establish a pattern that can be extended to other Flux APIs in the future.

### Non-Goals

- It's not a goal to replace or modify the existing cloud-provider workload
  identity support introduced in RFC-0010. The `.spec.provider` field and its
  cloud-provider-specific behavior remain unchanged.
- It's not a goal to implement a full SPIFFE runtime (SPIRE agent). Instead,
  Flux controllers will issue short-lived SPIFFE SVIDs directly using a
  private key provided via a Kubernetes `Secret`.
- It's not a goal to support all Flux APIs in the first iteration. The initial
  implementation targets `OCIRepository` and `ImageRepository`, with other APIs
  to follow in subsequent releases.

## Proposal

We propose introducing a new spec field `.spec.credential` to the `OCIRepository`
and `ImageRepository` APIs. This field specifies the type of vendor-agnostic
short-lived credential to use for authentication. The field is mutually exclusive
with `.spec.provider` (when set to a cloud provider) because "provider" conveys
cloud-provider-specific semantics, and because the existing `generic` provider
value already has specific behavior in the OCI APIs when used together with
`.spec.serviceAccountName` that differs from what is proposed here (see details
[here](#why-a-new-field-and-not-specprovider)). The mutual exclusivity is enforced
via a CEL validation rule:

```yaml
x-kubernetes-validations:
- message: spec.credential can only be used with spec.provider 'generic'
  rule: '!has(self.credential) || !has(self.provider) || self.provider == ''generic'''
```

### Credential Types

The `.spec.credential` field is a structured object with the following
sub-fields:

- **`.spec.credential.type`** (string, required): The type of vendor-agnostic
  short-lived credential to use for authentication.
- **`.spec.credential.audiences`** (list of strings, optional): Specifies the
  audiences (`aud` claim) for the issued JWT when `.spec.credential.type` is
  `ServiceAccountToken` or `SpiffeJWT`. This allows the third-party service to
  verify that the token was intended for it. If not specified, defaults to
  `.spec.url` for `OCIRepository` and `.spec.image` for `ImageRepository`.
  This is analogous to the `Kustomization` and `HelmRelease` APIs, where
  the audience defaults to the remote cluster address.

Grouping credential-related fields under `.spec.credential` keeps the main
spec clean and allows extending the credential configuration with additional
options in the future without polluting the top-level spec.

The valid values for `.spec.credential.type` are:

- **`ServiceAccountToken`**: The controller issues a Kubernetes `ServiceAccount`
  token and uses it directly as a bearer token for authentication. The
  `ServiceAccount` is determined by the existing `.spec.serviceAccountName`
  field, which requires the `ObjectLevelWorkloadIdentity` feature gate
  (introduced in RFC-0010) to be enabled. If `.spec.serviceAccountName` is
  not set, the controller's own `ServiceAccount` is used. In multi-tenancy
  lockdown scenarios, the `--default-service-account` controller flag can be
  used to force a default `ServiceAccount` when `.spec.serviceAccountName` is
  not specified, preventing the controller's own `ServiceAccount` from being
  used. The third-party service must be configured with OIDC federation
  trusting the Kubernetes `ServiceAccount` token issuer. This is the same
  mechanism already used by the `generic` provider in the `Kustomization` and
  `HelmRelease` APIs for remote cluster access.

- **`SpiffeJWT`**: The controller issues a short-lived SPIFFE SVID JWT where
  the identity (the `sub` claim) is the Flux object itself, encoded as a
  [SPIFFE ID](https://github.com/spiffe/spiffe/blob/main/standards/SPIFFE-ID.md)
  in the format
  `spiffe://<trust-domain>/<resource>/<namespace>/<name>`, where `<resource>`
  is the lowercase plural form of the Kubernetes resource type, i.e.
  `ocirepositories` for `OCIRepository` and `imagerepositories` for
  `ImageRepository`.
  The trust domain for the SPIFFE ID is configured via the controller flag
  `--spiffe-trust-domain`. The `iss` claim is set to the controller flag
  `--spiffe-issuer`, which third-party services use for OIDC discovery. The
  JWT is signed using a private key provided via `--spiffe-secret-name`. Unlike
  `ServiceAccountToken`, this credential type does **not** depend on a
  Kubernetes `ServiceAccount` for identity. The third-party service must be
  configured to trust the SPIFFE issuer by having access to the corresponding
  JWKS document.

- **`SpiffeCertificate`**: The controller issues a short-lived SPIFFE SVID
  x509 certificate for client authentication via mTLS. The certificate encodes
  the same SPIFFE ID as `SpiffeJWT` in the SAN URI field. The certificate is
  signed using a CA private key and certificate provided via the controller
  flag `--spiffe-secret-name`. Both `OCIRepository` and `ImageRepository`
  already support client certificate authentication via mTLS, so this
  integrates naturally with the existing transport layer. The third-party
  service must be configured with the CA certificate for trust. Note that
  `.spec.certSecretRef` may still be optionally used alongside
  `SpiffeCertificate` for specifying a CA to trust the server's certificate.

### Controller Flags

The SPIFFE issuer and cryptographic material are configured as controller-level
flags rather than per-object spec fields. This is analogous to the
`ServiceAccountToken` credential type, which relies on the cluster-level
Kubernetes `ServiceAccount` token issuer PKI — an infrastructure-level concern,
not an object-level one. If these inputs were instead provided per-object (e.g.
via `.spec.secretRef`), the pattern would degenerate into a secret-based
authentication strategy similar to the `github` provider in the Git APIs, which
requires a GitHub App private key to be set in `.spec.secretRef`. The goal of
this RFC is for Flux objects to have their own identities with short-lived
credentials issued from a shared, controller-level PKI, just as Kubernetes
`ServiceAccount` tokens are issued from the cluster-level token issuer.

- **`--spiffe-trust-domain`**: The SPIFFE trust domain used to construct
  SPIFFE IDs for all SPIFFE credential types. The SPIFFE ID is encoded as
  `spiffe://<trust-domain>/<resource>/<namespace>/<name>` and is included as the
  `sub` claim in `SpiffeJWT` tokens and in the SAN URI field of
  `SpiffeCertificate` x509 certificates. Required when any object uses
  `SpiffeJWT` or `SpiffeCertificate` as its `.spec.credential`.

- **`--spiffe-issuer`**: The OIDC issuer URL for `SpiffeJWT` credentials.
  Used as the `iss` claim in the issued JWTs. Third-party services use this
  URL for OIDC discovery to fetch the JWKS and verify token signatures.
  Required when any object uses `SpiffeJWT` as its `.spec.credential`.

- **`--spiffe-secret-name`**: The name of a Kubernetes TLS `Secret`
  (`type: kubernetes.io/tls`) in the controller's namespace containing the
  cryptographic material for issuing SPIFFE SVIDs. This format is compatible
  with [cert-manager](https://cert-manager.io/), allowing the `Secret` to be
  automatically provisioned and rotated. The `Secret` must contain:
  - `tls.key`: The private key. Used for signing JWTs (`SpiffeJWT`) and for
    signing client certificates (`SpiffeCertificate`).
  - `tls.crt`: The CA certificate. Used for signing client certificates
    (`SpiffeCertificate`). Not required for `SpiffeJWT`.

  Required when any object uses `SpiffeJWT` or `SpiffeCertificate` as its
  `.spec.credential`.

### User Stories

#### Story 1

> As a cluster administrator, I want tenant A to pull OCI artifacts from a
> self-hosted Zot registry repository belonging to tenant A using workload
> identity, without any cloud provider dependency.

For example, I would like to have the following configuration:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: OCIRepository
metadata:
  name: tenant-a-repo
  namespace: tenant-a
spec:
  url: oci://zot.zot.svc.cluster.local:5000/tenant-a
  credential:
    type: ServiceAccountToken
    audiences:
      - zot.zot.svc.cluster.local
  serviceAccountName: tenant-a-sa
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tenant-a-sa
  namespace: tenant-a
```

The Zot registry is configured with OIDC federation trusting the Kubernetes
`ServiceAccount` token issuer, and an authorization policy granting the
`ServiceAccount` `tenant-a/tenant-a-sa` access only to the `tenant-a`
repository.

#### Story 2

> As a cluster administrator, I want to authenticate Flux objects with a
> SPIFFE-aware service using the identity of the Flux object itself, not a
> `ServiceAccount`.

For example, I would like to have the following configuration:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: OCIRepository
metadata:
  name: my-app
  namespace: production
spec:
  url: oci://registry.example.com/my-app
  credential:
    type: SpiffeJWT
    audiences:
      - registry.example.com
```

The controller is started with `--spiffe-trust-domain=example.com`.
The SPIFFE ID for this object would be
`spiffe://example.com/ocirepositories/production/my-app`,
and the registry would authorize access based on this identity.

#### Story 3

> As a cluster administrator, I want to use mTLS with a SPIFFE certificate
> to authenticate with a container registry that supports SPIFFE-based client
> certificate authentication.

For example, I would like to have the following configuration:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: OCIRepository
metadata:
  name: secure-app
  namespace: production
spec:
  url: oci://registry.example.com/secure-app
  credential:
    type: SpiffeCertificate
```

The controller is started with `--spiffe-trust-domain=example.com`.
It issues a short-lived x509 certificate with the SPIFFE ID
`spiffe://example.com/ocirepositories/production/secure-app`
in the SAN URI field, and uses it for mTLS authentication with the registry.

#### Story 4

> As a cluster administrator, I want to use `ServiceAccount` token
> authentication to scan container images from a Harbor registry that supports
> OIDC workload identity federation.

```yaml
apiVersion: image.toolkit.fluxcd.io/v1
kind: ImageRepository
metadata:
  name: my-app
  namespace: apps
spec:
  image: harbor.example.com/apps/my-app
  credential:
    type: ServiceAccountToken
    audiences:
      - harbor.example.com
  serviceAccountName: my-app-sa
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-app-sa
  namespace: apps
```

### Why a New Field and Not `.spec.provider`

- The word "provider" suggests a cloud provider, which is not what this feature targets.
  "Credential" better communicates that we are implementing authentication standards, not
  targeting cloud vendors.
- Using only `.spec.provider: generic`, like in the `Kustomization` and `HelmRelease` APIs,
  is not viable because this value already has specific behavior in the OCI APIs when used
  together with `.spec.serviceAccountName`, it means: "use `.imagePullSecrets` from the
  `ServiceAccount` referenced by `.spec.serviceAccountName`".

The `Kustomization` and `HelmRelease` APIs have a naming inconsistency where
`provider: generic` in `.spec.kubeConfig.configMapRef` is used for
`ServiceAccountToken`-based authentication with remote clusters. Updating this
ConfigMap API is out-of-scope for this RFC; the existing behavior remains
unchanged and is considered an exception to the standard proposed here.

At the end of the day, `generic` is the default provider if none is specified,
and it will be the only accepted provider when using `.spec.credential`. This
is semantically correct because standard-based credentials are vendor-agnostic
by definition, and therefore a "generic" provider conveys the intended meaning.

### How Trust Should Be Established

#### For `ServiceAccountToken`

The Kubernetes `ServiceAccount` token issuer already implements the OIDC
Discovery protocol. Third-party services must be configured to trust this
issuer by having network access to the OIDC discovery and JWKS endpoints, or
by having the JWKS document configured out-of-band. This is the same trust
model used by cloud providers for workload identity (see
[RFC-0010 Technical Background](https://github.com/fluxcd/flux2/tree/main/rfcs/0010-multi-tenant-workload-identity#technical-background)).
For clusters without a built-in public issuer URL, the responsibility of
serving the OIDC discovery and JWKS documents can be taken away from the
kube-apiserver by using the `--service-account-issuer` and
`--service-account-jwks-uri` apiserver flags to point to externally hosted
documents. Signing key rotation is also possible through the
`--service-account-signing-key-file` and `--service-account-key-file`
apiserver flags (see
[Flux cross-cloud integration docs](https://fluxcd.io/flux/integrations/cross-cloud/#for-clusters-without-a-built-in-public-issuer-url)).

#### For `SpiffeJWT`

`SpiffeJWT` is also OIDC-based — it is an extension of the same OIDC trust
model used by `ServiceAccountToken`, but with a separate issuer and key
material managed by the cluster administrator rather than by the Kubernetes API
server. The trust establishment follows the same OIDC Discovery protocol: users
must host the issuer and JWKS documents at the URL specified by
`--spiffe-issuer`, reachable from the third-party services they are integrating
with. The controller signs the JWTs with the private key from
`--spiffe-secret-name`, and the corresponding public key must be available in
the JWKS document at the issuer URL.

#### For `SpiffeCertificate`

Users must configure the third-party service with the CA certificate provided
via `--spiffe-secret-name` so it can verify client certificates issued by the
Flux controller.

### Alternatives

#### Using `.spec.provider` Instead of `.spec.credential`

An alternative would be to add the new credential types as values of
`.spec.provider` directly (e.g. `provider: serviceaccounttoken`,
`provider: spiffejwt`), rather than introducing a separate `.spec.credential`
field. However, `.spec.provider` conveys cloud-provider-specific semantics and
uses lowercase values like `aws`, `azure`, `gcp` — mixing vendor-agnostic
credential types into this field would blur the distinction between targeting a
cloud provider and using an open standard. Additionally, the SPIFFE credential
types do not require a `ServiceAccount` at all, which is fundamentally
different from how `.spec.provider` operates today.

#### Using an External SPIFFE Runtime (SPIRE)

Instead of issuing SPIFFE SVIDs directly, we could require users to deploy a
full SPIRE agent and have the Flux controllers obtain SVIDs through the SPIFFE
Workload API or Delegated Identity API. However, the standard
[Workload API](https://github.com/spiffe/spiffe/blob/main/standards/SPIFFE_Workload_API.md)
only returns the identity assigned to the calling workload — it does not allow
the caller to request a specific SPIFFE ID. SPIRE's
[Delegated Identity API](https://spiffe.io/docs/latest/deploying/spire_agent/#delegated-identity-api)
does allow a trusted delegate (like a Flux controller) to obtain SVIDs on
behalf of other workloads, but those identities must still match pre-registered
entries on the SPIRE Server. This means a cluster administrator would need to
pre-register a SPIRE entry for every Flux object, which would be a significant
operational burden. Beyond this, deploying a full SPIRE infrastructure solely
for Flux authentication is a heavy dependency that many users would not want.
Our approach of issuing SVIDs directly from a provided private key is simpler
and self-contained, while still producing standard SPIFFE SVIDs that any
SPIFFE-aware service can verify.

## Design Details

### API Changes

For the `OCIRepository` API, we introduce the following new fields:

```go
// Credential specifies the configuration for vendor-agnostic short-lived
// credentials.
type Credential struct {
    // Type specifies the type of credential to use for authentication.
    // +required
    Type string `json:"type"`

    // Audiences specifies the audiences for the issued JWT when Type is
    // ServiceAccountToken or SpiffeJWT. Defaults to .spec.url for
    // OCIRepository and .spec.image for ImageRepository.
    // +optional
    Audiences []string `json:"audiences,omitempty"`
}

type OCIRepositorySpec struct {
    // ... existing fields ...

    // Credential specifies the vendor-agnostic short-lived credential
    // configuration for authentication. Mutually exclusive with using
    // .spec.provider for cloud-provider workload identity and with
    // .spec.secretRef.
    // +optional
    Credential *Credential `json:"credential,omitempty"`
}
```

Equivalent fields are introduced for the `ImageRepository` API.

### Validation Rules

- `.spec.credential` is mutually exclusive with `.spec.provider` when the
  provider is set to anything other than `generic`.
- `.spec.credential` is mutually exclusive with `.spec.secretRef`.
- `.spec.credential.type` is required when `.spec.credential` is specified.
- `.spec.credential.audiences` is optional. When not specified and
  `.spec.credential.type` is `ServiceAccountToken` or `SpiffeJWT`, the
  audience defaults to `.spec.url` for `OCIRepository` and `.spec.image`
  for `ImageRepository`.
- When `.spec.credential.type` is `SpiffeJWT`, the controller flags
  `--spiffe-trust-domain`, `--spiffe-issuer` and `--spiffe-secret-name` must
  be set, otherwise a terminal error is returned.
- When `.spec.credential.type` is `SpiffeCertificate`, the controller flags
  `--spiffe-trust-domain` and `--spiffe-secret-name` must be set, otherwise a
  terminal error is returned.
- When `.spec.credential.type` is `ServiceAccountToken`,
  `.spec.serviceAccountName` is optional. If not set, the controller's own
  `ServiceAccount` is used. Using `.spec.serviceAccountName` requires the
  `ObjectLevelWorkloadIdentity` feature gate (introduced in RFC-0010) to be
  enabled. In multi-tenancy lockdown scenarios, the
  `--default-service-account` controller flag can be used to force a default
  `ServiceAccount` when `.spec.serviceAccountName` is not specified, preventing
  the controller's own `ServiceAccount` from being used.

### Credential Issuance

#### `ServiceAccountToken`

The controller uses the Kubernetes `TokenRequest` API to issue a short-lived
token for the configured `ServiceAccount` (or the controller's own
`ServiceAccount` if none is configured). The token is issued with the audiences
specified in `.spec.credential.audiences`. The token is then used as a bearer token in the
`Authorization` header when authenticating with the third-party service (e.g. a
container registry).

This reuses the same `ServiceAccount` token creation logic already present in
the `github.com/fluxcd/pkg/auth` library from RFC-0010, but skips the cloud
provider token exchange step.

#### `SpiffeJWT`

The controller constructs a JWT with the following claims:

- `iss`: The value of `--spiffe-issuer`.
- `sub`: The SPIFFE ID in the format
  `spiffe://<trust-domain>/<resource>/<namespace>/<name>`, where the
  trust domain is the value of `--spiffe-trust-domain`.
- `aud`: The values from `.spec.credential.audiences`.
- `exp`: One hour from the current time.
- `nbf`: The current time.
- `iat`: The current time.
- `jti`: A unique token identifier.

The JWT is signed using the private key from the `Secret` referenced by
`--spiffe-secret-name`. The signing algorithm is determined by the key type
(e.g. RS256 for RSA, ES256 for EC P-256).

#### `SpiffeCertificate`

The controller generates a short-lived x509 certificate with:

- The SPIFFE ID in the SAN URI field.
- A validity period of one hour.
- Signed by the CA from the `Secret` referenced by `--spiffe-secret-name`.

The certificate and a freshly generated private key are used for mTLS
authentication with the third-party service.

### Integration with the `auth` Library

We propose extending the `github.com/fluxcd/pkg/auth` library to support
vendor-agnostic credentials alongside the existing cloud-provider workload
identity. This involves:

1. Renaming the existing `auth/generic` package to `auth/serviceaccounttoken`.
2. Introducing `auth/spiffejwt` for issuing SPIFFE SVID JWTs, using a JWT
   library (e.g. `golang.org/x/oauth2/jws` or `github.com/go-jose/go-jose`)
   and standard Go crypto libraries (`crypto/rsa`, `crypto/ecdsa`) for signing.
3. Introducing `auth/spiffecertificate` for issuing SPIFFE SVID x509
   certificates, using standard Go crypto libraries (`crypto/x509`,
   `crypto/rsa`, `crypto/ecdsa`, `encoding/pem`) for certificate generation
   and signing.

### CLI Support

The `flux push artifact` command (and related commands: `list`, `pull`, `tag`,
`diff`) will support the following credential types via the `--creds` flag:

#### `ServiceAccountToken`

Two modes are supported:

- **Token creation via `TokenRequest` API**: The CLI creates a short-lived
  `ServiceAccount` token using the Kubernetes API. This mode works both inside
  a pod (e.g. in a CI/CD pipeline running in-cluster) and locally with a
  kubeconfig.

  ```shell
  flux push artifact --creds=ServiceAccountToken \
    --audiences=aud1,aud2 \
    --sa-name=my-sa
  ```

  If `--sa-name` is not specified and the CLI is running inside a pod, it
  reads the mounted `ServiceAccount` token file to determine the
  `ServiceAccount` name. The `auth` library already has logic for finding
  the current `ServiceAccount` when running inside a pod (used for issuing
  controller-level tokens), so this can be reused by the CLI. If `--audiences`
  is not specified, it defaults to the artifact URL. Note that this mode
  requires the `create` verb on the `serviceaccounts/token` subresource for
  the target `ServiceAccount`.

- **Pre-existing token file**: The CLI reads a `ServiceAccount` token from a
  file. This mode is useful when the token has already been obtained through
  other means, e.g. a
  [projected volume](https://kubernetes.io/docs/concepts/storage/projected-volumes/#serviceaccounttoken).

  ```shell
  flux push artifact --creds=ServiceAccountToken \
    --sa-token=/path/to/token
  ```

#### `SpiffeJWT` and `SpiffeCertificate`

In the controller, SPIFFE SVIDs are minted directly by the controller with
identities tied to Flux objects. In the CLI, there is no Flux object, and
providing the private signing key to the CLI would undermine the security model.
Instead, the CLI integrates with SPIRE via the
[SPIFFE Workload API](https://github.com/spiffe/spiffe/blob/main/standards/SPIFFE_Workload_API.md)
using the [`go-spiffe`](https://github.com/spiffe/go-spiffe) library. In this
model, SPIRE assigns the identity to the calling workload (e.g. a CI runner
pod) based on its registration entries — the caller does not choose the SPIFFE
ID. These are complementary identity models: the controller mints SVIDs for
Flux objects, while the CLI relies on SPIRE to attest the workload running the
CLI.

For JWT SVIDs, the CLI calls `workloadapi.FetchJWTSVID()` with the specified
audiences:

```shell
flux push artifact --creds=SpiffeJWT \
  --audiences=aud1,aud2
```

If `--audiences` is not specified, it defaults to the artifact URL.

For x509 SVIDs, the CLI calls `workloadapi.FetchX509SVID()` and uses the
returned certificate and private key for mTLS client authentication:

```shell
flux push artifact --creds=SpiffeCertificate
```

Both communicate with the SPIRE Agent via a Unix domain socket configured
through the `SPIFFE_ENDPOINT_SOCKET` environment variable.

Note that adding `github.com/spiffe/go-spiffe/v2` as a dependency to the CLI
binary pulls in gRPC (for the Workload API socket communication), which has a
non-trivial impact on binary size.

#### `GitHubOIDCToken`

As an exception to the "vendor-agnostic" scope of this RFC, the CLI will also
support GitHub Actions OIDC tokens. This is motivated by the fact that GitHub
Actions is a widely used CI/CD platform and its OIDC token API provides a
convenient way to obtain short-lived tokens without managing secrets. The CLI
obtains the token from the
[GitHub Actions OIDC Token API](https://docs.github.com/en/actions/reference/security/oidc#methods-for-requesting-the-oidc-token).

```shell
flux push artifact --creds=GitHubOIDCToken \
  --audiences=aud1,aud2
```

If `--audiences` is not specified, it defaults to the artifact URL.

### Cache Considerations

The existing token cache from RFC-0010 can be reused. Following the same
principle established there, any new fields that interfere with the
creation of the credential will be included in the cache key.

### Security Considerations

- **Private key protection**: The `Secret` referenced by
  `--spiffe-secret-name` contains sensitive cryptographic material. It must be
  protected with appropriate RBAC and ideally managed through a secrets
  management solution. The controller should only have read access to this
  `Secret`.
- **Short-lived credentials**: All credential types produce short-lived tokens
  or certificates (with a validity of one hour), limiting the blast radius if a
  credential is leaked.
- **SPIFFE ID namespacing**: The SPIFFE ID includes the namespace and object
  resource/name, providing natural multi-tenant isolation. A tenant's Flux
  objects will always have SPIFFE IDs scoped to their namespace.
- **SPIFFE ID uniqueness**: The SPIFFE ID encodes the resource type, namespace
  and name of the Flux object, which are unique within a cluster. This
  guarantees that no two objects can have the same SPIFFE identity, preventing
  one object from impersonating another (which is otherwise possible if using
  a `ServiceAccount`).
- **Trust boundary**: When using `ServiceAccountToken`, the trust boundary is
  the same as RFC-0010 (Kubernetes `ServiceAccount` token issuer). When using
  SPIFFE credentials, the trust boundary extends to whoever has access to the
  private key referenced by `--spiffe-secret-name`. This key should be
  treated with the same level of care as the Kubernetes CA key.

## Implementation History

<!--
Major milestones in the lifecycle of the RFC such as:
- The first Flux release where an initial version of the RFC was available.
- The version of Flux where the RFC graduated to general availability.
- The version of Flux where the RFC was retired or superseded.
-->
