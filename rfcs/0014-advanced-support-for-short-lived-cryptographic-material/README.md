# RFC-0014 Advanced Support for Short-Lived Cryptographic Material

**Status:** implementable

<!--
Status represents the current state of the RFC.
Must be one of `provisional`, `implementable`, `implemented`, `deferred`, `rejected`, `withdrawn`, or `replaced`.
-->

**Creation date:** 2026-02-01

**Last update:** 2026-10-24

## Summary

In [RFC-0010](https://github.com/fluxcd/flux2/tree/main/rfcs/0010-multi-tenant-workload-identity)
we introduced object-level workload identity for cloud provider integrations
leveraging Kubernetes ServiceAccount tokens. That was our first step towards
advanced workload identity features. In this RFC we propose a new set of
closely-related enhancements through a new set of APIs, where the central
topic is advanced support for short-lived cryptographic material. These
enhancements cover not only advanced workload identity support, but also
other scenarios that benefit from short-lived cryptographic material.
Namely, these scenarios are secure communication between Flux controllers
and third-party services, and between Flux controllers themselves.
The main goal of the RFC is to define the shape of these APIs so that we
can capture all these use cases in a consistent way all across Flux. The
implementation of the APIs across all the Flux components will be split
into multiple releases.

## Motivation

Many Flux users have advanced security requirements and use cases. This RFC
is motivated by the subset of security requirements that can benefit from
a unified solution for short-lived cryptographic material. They can be
split into the following categories:

- *Identification*: Secure identification with external systems, such as
  cloud providers or vendor-neutral infrastructure components such as
  free open-source projects.
- *Private Communication*: Secure communication among the Flux controllers,
  and between Flux controllers and external systems.
- *Centralized Management*: Integration with central infrastructure that
  is responsible for providing short-lived cryptographic material for the
  rest of the stack of the Flux user.

The Secure Production Identity Framework For Everyone (SPIFFE) CNCF project
aims to provide a standardized framework for issuing and providing short-lived
cryptographic identities across the board. It supports widely used standards
such as JWT and X.509 certificates. The standard name defined by SPIFFE for
the short-lived cryptographic material it provides is SPIFFE Verifiable
Identity Document (SVID). The standard defines both JWT-SVID and X509-SVID.
A third standard was introduced recently: Workload Identity Token, WIT-SVID.
It still lacks real-world adoption due to being so new, so we leave it out of
the scope of this RFC but keep it in mind for the future. All the SPIFFE
standards referenced in this RFC are defined
[here](https://github.com/spiffe/spiffe/tree/main/standards).

The SPIFFE project is graduated and adoption grows steadily,
from large technology players applying it to solve agentic
identity challenges, to open-source projects such as service
meshes applying it to provide transparent mTLS between apps
and transparent identities.

Flux users have also manifested interest in integrations
with SPIFFE, such as
[#3368 (comment)](https://github.com/fluxcd/flux2/pull/3368#discussion_r1040899292)
and [#5679](https://github.com/fluxcd/flux2/discussions/5679),
plus many offline discussions in conferences.

This RFC takes RFC-0010 to the next level in multiple dimensions:

- Flux will be able to support workload identity for more vendor-neutral
  infrastructure components, such as OCI registries like Harbor and
  Zot (both CNCF projects) that have implemented support for workload
  identity. RFC-0010 introduced workload identity for remote Kubernetes
  clusters, and Flux 2.9 introduced workload identity for OpenBao/Vault.
  By supporting workload identity for OCI registries like Harbor and Zot,
  Flux will support workload identity for opensource projects covering
  three types of services that are core to Flux: Kubernetes, OCI registries
  and key management systems for decryption.
- Flux will solve four classes of *confused deputy* problems by allowing
  the identity of a Flux Custom Resource object to be the object reference
  itself, i.e. the object's Group-Kind-Namespace-Name-UID quintuple, instead
  of the identity of the spec-configurable ServiceAccount:
  - Principal-collapse (identity-sharing) confusion. The deputy can't
    tell which of N objects is acting because they present one shared
    ServiceAccount identity.
  - Identity-borrowing / substitution. The deputy is induced to act as
    an identity the caller selected rather than is, by picking a
    more-privileged `.serviceAccountName` in the same namespace.
  - Authority-aggregation. The deputy exercises the union of privileges
    accumulated in a shared ServiceAccount on behalf of an object entitled
    to only a subset of those privileges.
  - Lifetime / incarnation confusion. The deputy keeps honoring a
    long-lived shared ServiceAccount across object lifetimes: a new
    object inherits the authority of the old.
- Flux will support both an out-of-the-box provider for workload identity,
  which is Kubernetes itself through ServiceAccount tokens, and SPIFFE: a
  much more advanced workload identity solution that covers more security
  use cases and aims specifically at centralizing management of short-lived
  cryptographic material. By using the JWT PKI provided by Kubernetes, Flux
  users go a long way. But each cluster will usually have its own key pair,
  and federating several clusters in a trust domain can be a challenge. On
  the other hand, because SPIFFE's main goal is to solve this very class of
  problems, it came up with now-established ways for different SPIFFE
  runtimes to federate and provide a unified layer of trust across clusters
  and the entire infrastructure of an organization. "SPIFFE is the bottom
  turtle" is the motto of the project. This motto emphasizes their
  fundamental goal of being the cornerstone for PKI across the board.

### Goals

1. Defining the common APIs that will support the use cases described in
this RFC in a way that they can be uniformly implemented across all the
Flux components (given enough release cycles).

2. Indulge in the "SPIFFE is the bottom turtle" philosophy. Allow Flux
to participate with first-class support in the SPIFFE landscape of an
organization aiming for centralized management of short-lived
cryptographic material.

### Non-Goals

- It's not a goal of this RFC to make Flux become a SPIFFE runtime
  like SPIRE (the reference SPIFFE runtime implementation). The whole
  point of SPIFFE is shifting the responsibility of key management
  from applications to the infrastructure. From this RFC's
  perspective, Flux is an application and whatever SPIFFE runtime
  the user has deployed is the infrastructure. (Note: There are both
  free (SPIRE) and commercial options available for a SPIFFE runtime.)
- It's not a goal to add any new controller-level workload identity
  features that acquire identities for use on behalf of Flux objects
  to communicate with external services. The APIs proposed here are
  mainly around annotations in ServiceAccount objects, which do not
  currently exist in the controller-level authentication flows by
  design. This, other challenges for supporting controller-level for
  the new OCI providers and deliberate choice to encourage users to
  migrate to multi-tenant-safe approaches are enough reasons to keep
  any controller-level workload identity features out of the scope of
  this RFC.

## Proposal

The proposal covers API annotations, controller options, new dependencies and bootstrap.

The CLI command `flux push artifact` and family can get the same
features as `flux-mirror` (e.g. GitHub/Forgejo Actions OIDC, etc.),
but the flags will likely be different. Because the discussion
around this set of CLI commands applies only to them, we will
leave it out of the scope of this RFC. This RFC is focused
exclusively on the common annotations and controller options
that will be reused as standards across the Flux controllers.

### API Annotations

This feature will not introduce new `.spec` fields. Instead, it is driven by
annotations in Kubernetes ServiceAccount objects.
The reason is simple:
we already implemented this in RFC-0010. We made this choice back then because we
wanted to provide the exact same experience as the cloud-provider-native workload
identity features (EKS IAM Roles for Service Accounts, AKS Workload Identity, GKE
Workload Identity). We are going to follow this established pattern once again,
because this is what the relationship between platform and development teams has
come to traditionally expect. Platform teams are responsible for configuring the
ServiceAccount objects, while development teams are responsible for configuring
the Flux objects. These annotations will complement the behavior of the fields
`.spec.provider` and `.spec.serviceAccountName` in most of the APIs,
`.spec.kubeConfig.configMapRef` in Kustomization and HelmRelease, and also
`.spec.decryption.serviceAccountName` in Kustomization. Basically, the existing
fields from RFC-0010.

The new annotations will have the following format:

```yaml
(<subdomain>).(auth|mtls).fluxcd.io/<annotation-key>: <annotation-value>
```

The subdomains `*.auth.fluxcd.io` and `*.mtls.fluxcd.io` will represent
independent sets of features: credential acquisition and mutual TLS.

The format above is a generalization of the first annotation under the
`fluxcd.io` domain that we introduced for authentication purposes, in
RFC-0010:

```yaml
gcp.auth.fluxcd.io/workload-identity-provider: projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/POOL_ID/providers/PROVIDER_ID
```

The kebab-case format for the annotation keys matches both the Flux GCP
annotation above and the EKS/AKS/GKE ones (that we borrowed from them):

```yaml
eks.amazonaws.com/role-arn: arn:aws:iam::123456789123:role/tenant-a-ecr
azure.workload.identity/client-id: d6e4fc00-c5b2-4a72-9f84-6a92e3f06b08
azure.workload.identity/tenant-id: 72f988bf-86f1-41af-91ab-2d7cd011db47
iam.gke.io/gcp-service-account: sa-name@project.iam.gserviceaccount.com
```

Above are the five annotations implemented for RFC-0010: Flux GCP plus
the borrowed cloud provider annotations. In the following subsections
we describe the new ones, which are all Flux-defined. The rules for
using these annotations are described in
[API Validation and Backward Compatibility Guarantees](#api-validation-and-backward-compatibility-guarantees).

#### `credential.auth.fluxcd.io/provider`

Values:

- `kubernetes`: Flux will use the Kubernetes TokenRequest API for acquiring
  the credential. When using this provider, the identity associated with the
  credential will be the Kubernetes Username `system:serviceaccount:<ns>:<sa>`
  i.e. the username representing the Kubernetes ServiceAccount itself where
  the annotation is read from.
- `spiffe`: Flux will use the SPIFFE Broker API for acquiring the credential.
  When using this provider, the identity associated with the credential
  will be a SPIFFE ID i.e. `spiffe://<trust domain>/<path>`. In this case,
  the SPIFFE runtime is responsible for determining this identity, not Flux.
  Flux's only job is to send a SPIFFE Broker API KubernetesObjectReference
  (see [this](#spiffe-broker-api-endpoint) section). It contains a fully
  qualified reference to the Flux object, including API group, kind, name,
  namespace and UID. SPIFFE will make sure the object resolved from the
  namespaced name matches the UID contained in the reference.

#### `credential.auth.fluxcd.io/type`

Values:

- `jwt`: Acquire a JSON Web Token from the credential provider.
- `x509`: Acquire an X.509 certificate from the credential provider.

When using `.spec.provider: azure` in the Flux objects, only the `jwt`
credential type will initially be supported. This is because Azure
does not have a feature for getting access tokens by exchanging them
for X.509 certificates at the time of writing this RFC. Only AWS and
GCP have features for those:

- AWS IAM Roles Anywhere
- GCP Workload Identity Federation

Additionally, the `jwt` credential type will always be the only one
supported for the `kubernetes` credential provider. There are two
reasons for this:

- The Kubernetes APIs for issuing credentials do not support fine-grained
  access control for restricting the target audiences of those credentials.
  The permission required for issuing a JWT whose audience is the API
  server itself is the same permission required for issuing a JWT for
  any other audiences. This means that permission is also an effective
  privilege escalation path to `cluster-admin`, one of the worst kinds
  of confused deputy problems for a Kubernetes cluster administrator.
- Goal #2: "SPIFFE is the bottom turtle". SPIFFE's main goal is to be
  the de facto standard for identity and credential management within
  distributed systems. Support for SPIFFE in Flux means strongly
  adopting that philosophy. SPIFFE is the one thinking about new
  credential types like WIT-SVID, not Kubernetes, and that's okay.
  Kubernetes does not have to provide everything out-of-the-box,
  otherwise the CNCF ecosystem would not exist. Flux will, however,
  consider supporting other standards for the same purpose if they
  show up in the future, but only those attaining widespread adoption
  within the ecosystem comparable to SPIFFE (which today is a
  10-year-old project just like Flux).

Support for `jwt` in `kubernetes` is Generally Available since RFC-0010,
hence support for it will not cease. But we choose not to introduce support
for more credential types backed by Kubernetes, unless they attain the
same level of security provided by SPIFFE.

#### `credential.auth.fluxcd.io/expiration`

An optional Go duration string specifying the expiration time of the
credential. The minimum is `10m`, the lower bound enforced by the
Kubernetes `TokenRequest` API. For example, `15m` for fifteen minutes or
`1h` for one hour.

The only scenario that can support this annotation is the `kubernetes`
credential provider (which, as mentioned above, supports only the `jwt`
credential type). At the time of writing this RFC, the SPIFFE APIs
do not support specifying credential expiration.

#### `credential.auth.fluxcd.io/audiences`

An optional list of audiences for the `jwt` credential type only, since
`x509` does not use audiences. The value is split by line breaks and commas,
each part is trimmed of surrounding whitespace, and only non-empty audiences
are added to the list. Splitting on commas is a deliberate hardening choice:
the JWT specifications allow an audience value to contain commas, but such
ambiguous values are a recurring source of vulnerabilities.

We keep the annotation name neutral to allow for future credential
types beyond `jwt`. WIT-SVID/`wit`, for example, is a candidate.

The service provider field defines a default audience.
Cloud providers have specific defaults (implemented in RFC-0010),
while the `generic` provider will always default to the address
of the service, e.g. an OCI registry host, or the API server
address of a remote Kubernetes cluster (the latter also
implemented in RFC-0010).

[Docker Hub OIDC Connections](#support-for-new-oci-service-providers)
only support the audience `https://identity.docker.com`. Therefore,
Flux will error out when this annotation is used with Docker Hub.

#### `credential.auth.fluxcd.io/username`

An optional username, mandatory for the `dockerhub` service
provider. Username is quite a widespread concept and
will be used for any service/credential providers that need it
in the future. In this RFC, however, the use cases are quite
limited: it will be used exclusively for the OCI APIs
(`OCIRepository` and `ImageRepository`) with the
[new OCI providers](#support-for-new-oci-service-providers),
i.e. `generic` and `dockerhub`.

#### `transit-engine.auth.fluxcd.io/login-path`

The login path for using a transit engine (we will now refer to OpenBao
and Vault collectively as transit engines, since they share the concept).
This is the HTTP path in the transit engine where the request for login
must be sent.

#### `transit-engine.auth.fluxcd.io/role`

The role to be used when authenticating against a transit engine.

A transit engine role is an identifier that defines two things:

- Rules configured in the transit engine for matching credentials.
- The permissions granted to the authenticated entity within the transit engine.

#### `spiffe.mtls.fluxcd.io/server-ids`

An optional list of SPIFFE IDs of servers. When configured, this annotation opts
into an mTLS connection backed by SPIFFE with the third party service
associated with the Flux object. The handshake will authorize the
server using one of these SPIFFE IDs.

The value is split by line breaks and commas, each part is trimmed of
surrounding whitespace, and only non-empty values are added to the
list. Splitting on commas is unambiguous because the SPIFFE ID standard
does not allow commas, whitespace or line breaks in a SPIFFE ID.

When using TLS backed by SPIFFE, we will always make it mTLS. Since Flux
is building native support for SPIFFE, there's no reason not to always
send a client X509-SVID certificate as part of the handshake, it comes
for free when acquiring the SPIFFE CA. Opting out from client
authentication when using SPIFFE is only for applications that do not
support SPIFFE natively and are somehow statically configured with
the SPIFFE CA (which doesn't last long, typically requiring frequent
rotation).

Because for us SPIFFE is always mTLS, which implies client authentication
is covered, this annotation will also be looked up from a ServiceAccount.
This locks ServiceAccount as the de facto construct for identity configuration
in Flux.

Among the service providers, this annotation is supported by `generic` and
by `github` when it's a GitHub Enterprise Server; the public providers
`aws`, `azure`, `gcp`, `dockerhub` and GitHub.com don't support it.

#### `aws.auth.fluxcd.io/sts-endpoint`

We are going to make `Bucket.spec.sts.provider` set to `aws`
a legacy method of expressing AWS STS endpoints. That kind of
configuration is better placed together with all the new
annotations we are introducing in this RFC. The field will
continue working as before, for both STS providers: `aws`
and `ldap`. However, if users want to opt into the new
annotations introduced in this RFC, they will be forced to
use `aws.auth.fluxcd.io/sts-endpoint` instead. This change
automatically introduces STS support for every other AWS
integration.

#### Support for New OCI Service Providers

We will introduce a new allowed value for `.spec.provider` in the
OCIRepository and ImageRepository APIs, `dockerhub`, and add support
for short-lived cryptographic material to the existing `generic`
provider.

The `generic` provider already existed in the OCI APIs, but it did
not have any support for short-lived cryptographic material, unlike
`.spec.kubeConfig.configMapRef` and `.spec.decryption`. Before this
RFC, `generic` only supported static secrets coming from Kubernetes
Secret objects.

The `generic` provider represents opensource projects accepting OIDC
tokens directly, either as the password of a username/password pair,
or as a Bearer token in the `Authorization` header. Examples: Zot, Harbor.

A ServiceAccount for `generic` registries requiring a username can use
[`credential.auth.fluxcd.io/username`](#credentialauthfluxcdiousername).

The default audience for `generic` registries for
[`credential.auth.fluxcd.io/audiences`](#credentialauthfluxcdioaudiences)
will be the registry host e.g. `my-zot-registry.com`.

The `dockerhub` provider represents the Docker Hub registry, which
has recently launched OIDC support. Their login process is a simple
credential exchange along the lines of `aws`, `azure` and `gcp`.
This means it requires inputs like the ones passed via the cloud
provider annotations that we initially introduced in RFC-0010. We
will implement `dockerhub` similarly by introducing the annotation
`dockerhub.auth.fluxcd.io/oidc-connection-id`.

Unlike `aws`, `azure` and `gcp`, the only audience supported by Docker
Hub OIDC Connections, `https://identity.docker.com`, will be hardcoded
as the only option and the `credential.auth.fluxcd.io/audiences`
annotation will be rejected with an error.

Unlike `generic`, Docker Hub requires an explicit username, which
makes `credential.auth.fluxcd.io/username` mandatory for `dockerhub`.

### Controller Options

The new controller options cover both of the following:

- The features defined through annotations across the Flux Custom Resources.
- The inter-controller communication.

Some options apply to both, some apply only to one of the two categories mentioned above.

#### The `go-spiffe` Environment Variables

The first set of controller options that will be added are the environment
variables used by the `go-spiffe` library:

```sh
SPIFFE_ENDPOINT_SOCKET=unix:///spiffe-workload-api/spire-agent.sock
```

This is how applications using `go-spiffe` detect the SPIFFE runtime,
which is aligned with how we support many of the environment
variables used in the cloud provider libraries.

These environment variables will apply to both the features
defined through annotations across the Flux Custom Resources,
and to inter-controller communication.

Here, we are mentioning only the main environment variable used by `go-spiffe`,
`SPIFFE_ENDPOINT_SOCKET`, but all others should work.

The variable `SPIFFE_ENDPOINT_SOCKET` specifies the address of the Unix Domain
Socket serving the SPIFFE Workload API. The SPIFFE Workload API is the standard
interface through which applications can fetch SVIDs from the SPIFFE runtime.
The SPIFFE Workload API is a gRPC API, and the Unix Domain Socket is the transport
through which the gRPC API is exposed. The value
`unix:///spiffe-workload-api/spire-agent.sock`
illustrated above is the typical value used with the SPIRE runtime. Other SPIFFE
runtimes will likely use different values, and they all should work seamlessly
with Flux.

#### SPIFFE Broker API Endpoint

SPIFFE has introduced the Broker API, also as part of the SPIFFE runtime
like Workload API, specifically with Flux's use case in mind. In fact, we
collaborated with the SPIFFE maintainers to define the SPIFFE Broker API.
In particular, we contributed the
[KubernetesObjectReference](https://github.com/spiffe/spiffe/blob/99470b9abc825f14aa364dfa2c3b53b02ba5db5b/standards/brokerapi.proto#L84-L105)
reference type as part of the API.

The SPIFFE Broker API allows a SPIFFE-attested workload, i.e. a workload
that has already fetched an X509-SVID for its own identity from the SPIFFE
Workload API, to fetch SVIDs for other workloads or things e.g. Kubernetes
objects. Such an application is called a SPIFFE Broker. SPIFFE does not
define how a Broker gets authorized for the SVIDs it will fetch, SPIFFE
instead leaves this as an implementation concern for the SPIFFE runtime
that will serve the Broker API. SPIRE, for example, defined the custom
Kubernetes RBAC verb `impersonate-via-spire` for allowing the SPIFFE ID
of a Broker workload (wired as a `User` in Kubernetes RBAC) to be
authorized for KubernetesObjectReferences.

Flux, acting as a SPIFFE Broker, will fetch SVIDs for the Flux
Custom Resource objects, which are Kubernetes objects, and hence
why we proposed to SPIFFE the KubernetesObjectReference.
Whenever reconciling a Flux Custom Resource object that is
configured to use SPIFFE, the controller will call the
SPIFFE Broker API to fetch an SVID for that object. The controller
will pass a KubernetesObjectReference containing the object's API
group, kind, namespace, name and UID. All of these will need to be
part of the cache key for the credential cache.

Because of SPIFFE Brokers like Flux, that will request SVIDs for Kubernetes
objects which do not map to Kubernetes workloads with running process IDs,
like Pods, Deployments and so on, whose attestation process depends
on the Linux Kernel that is running those processes, the SPIFFE Broker API
can also be served by the SPIFFE runtime over TCP. Attestation via kernel
simply does not apply to Kubernetes resources that are not workloads, like
the Flux Custom Resources. Attesting such objects is achieved by a sequence
of calls to the Kubernetes API Server, no Linux Kernel involved.

The SPIFFE Broker Endpoint is protected by SPIFFE's mTLS PKI through the
X509-SVID of the SPIFFE Broker and the trust bundle that comes along with
it. These are acquired through the SPIFFE Workload API, as explained in the
beginning of the section.

We propose the following flag for the gRPC URL of the SPIFFE Broker Endpoint:

```sh
--spiffe-broker-endpoint=dns:///<svc name>.<namespace>.svc.cluster.local:443
```

And the following flag for authorizing the SPIFFE ID of the endpoint:

```sh
--spiffe-broker-id=spiffe://<trust domain>/spire/agent/k8s_psat/<cluster>/pod/
```

In the example above, Broker API is being served by the SPIRE Agent, hence
the SPIRE-specific SPIFFE ID. Note the SPIFFE ID ending with a slash (`/`).
This implies a SPIFFE ID *prefix*, because SPIFFE IDs are not allowed to
end in slash. This is to authorize the SPIFFE ID of any SPIRE Agent in the
`k8s_psat` cluster named `<cluster>`, as they are all unique and end with
the pod UID. Note that this is just a detail of how SPIRE works. Other
SPIFFE runtimes may work differently and may assign a single SPIFFE ID
for the Broker API.

The flags introduced in this section are relevant only for the features
defined through annotations across the Flux Custom Resources. They do not
relate to inter-controller communication.

Both flags are mandatory for the controller to be able to reconcile Flux
Custom Resource objects that are configured to use SPIFFE. The controller
will not be able to reconcile those objects if the flags are not set, and
will yield error messages to that effect.

#### Inter-Controller Communication

For inter-controller communication, the controller's own X509-SVID
is enough for mTLS authentication and authorization. This means
only the `SPIFFE_ENDPOINT_SOCKET` environment variable is needed
in this case, for acquiring the controller's own X509-SVID via
SPIFFE Workload API.

However, in addition to acquiring its own X509-SVID, the controller
also needs to authorize SPIFFE IDs of the other controllers. This
warrants a new set of flags in the controllers.

The source-controller has to authorize the SPIFFE IDs of the other
controllers that can reach it. It needs a repeatable flag, since
there can be multiple controllers that can reach it:

- kustomize-controller
- helm-controller
- source-watcher
- Other external source controllers like source-watcher

The flag can be:

```sh
--authorized-artifact-client-spiffe-id=spiffe://<trust domain>/kustomize-controller
--authorized-artifact-client-spiffe-id=spiffe://<trust domain>/helm-controller
--authorized-artifact-client-spiffe-id=spiffe://<trust domain>/source-watcher
--authorized-artifact-client-spiffe-id=spiffe://<trust domain>/my-source-controller
```

Both source-watcher and other external source controllers can have
the exact same flag for their own artifact HTTP server.

The applier controllers (kustomize-controller and helm-controller) need a flag to
authorize artifact HTTP servers:

```sh
--authorized-artifact-server-spiffe-id=spiffe://<trust domain>/source-controller
--authorized-artifact-server-spiffe-id=spiffe://<trust domain>/source-watcher
--authorized-artifact-server-spiffe-id=spiffe://<trust domain>/my-source-controller
```

The notification-controller needs a flag for the Event Server:

```sh
--authorized-notifier-spiffe-id=spiffe://<trust domain>/source-controller
--authorized-notifier-spiffe-id=spiffe://<trust domain>/kustomize-controller
--authorized-notifier-spiffe-id=spiffe://<trust domain>/helm-controller
--authorized-notifier-spiffe-id=spiffe://<trust domain>/image-reflector-controller
--authorized-notifier-spiffe-id=spiffe://<trust domain>/image-automation-controller
--authorized-notifier-spiffe-id=spiffe://<trust domain>/source-watcher
--authorized-notifier-spiffe-id=spiffe://<trust domain>/my-source-controller
```

For sending events, all controllers (except notification-controller) can
have the following:

```sh
--authorized-event-server-spiffe-id=spiffe://<trust domain>/notification-controller
```

The notification-controller also needs a flag for the Receiver Server
to authenticate incoming connections from Ingress and Gateway controllers:

```sh
--authorized-receiver-spiffe-id=spiffe://<trust domain>/my-ingress-controller
--authorized-receiver-spiffe-id=spiffe://<trust domain>/my-gateway-controller
```

Finally, all the controllers can authorize the Kubernetes API Server
if it serves an X509-SVID in the same trust domain:

```sh
--authorized-kube-apiserver-spiffe-id=spiffe://<trust domain>/kube-apiserver
```

This covers the entire communication matrix between the Flux controllers.
Every peer can authorize the peer on the other side of the TCP connection,
for all types of HTTP traffic existing inside Flux.

The presence of at least one occurrence of these flags gates the respective
feature in the respective controller. Note that not all types of traffic
need to be enabled together. The flags allow protecting each type of traffic
independently, e.g. artifact traffic can be protected while Kubernetes
API Server traffic can remain using the default Kubernetes CA mounted into
pods.

### New Dependencies and Bootstrap

Implementing Kubernetes ServiceAccount tokens for OCI registries
introduces no new dependencies or complexity.

Implementing the various SPIFFE features introduces two very important
new dependencies.

#### SPIFFE Workload API and Bootstrap

The SPIFFE Workload API is accessed through a Unix Domain Socket.
Every single SPIFFE feature we are proposing here requires access
to this UDS socket.

To bind-mount the SPIFFE Workload API UDS socket into the Flux
controllers, there are two options:

- Using a `hostPath` Kubernetes volume.
- Using the SPIFFE CSI [driver](https://github.com/spiffe/spiffe-csi).

The `hostPath` alternative is forbidden by the
[`Restricted`](https://kubernetes.io/docs/concepts/security/pod-security-standards/#restricted)
Pod Security Standard that Flux opts into. To use this alternative
Flux users must either change the PSS restriction in Flux or use the
SPIFFE CSI driver, which is made specifically for bind-mounting the
SPIFFE Workload API UDS socket into workloads without need for the
restricted `hostPath` volume.

In order to use the SPIFFE CSI driver in the Flux controller pods,
it has to be deployed in the cluster before Flux. This brings a
bootstrap concern. Whatever bootstrap mechanism is chosen by the
Flux user must support deploying the SPIFFE CSI driver in the
cluster before deploying Flux itself. We will not introduce support
for this in the `flux bootstrap` command yet, as it also does not
support CNI dependencies, e.g. when deploying Flux in a cluster
where it will manage Cilium (Cilium must land first to establish
networking between the Flux controller pods). The only supported
bootstrap method that already handles both CNI and CSI dependencies
is the
[`flux-operator-bootstrap`](https://github.com/controlplaneio-fluxcd/terraform-kubernetes-flux-operator-bootstrap/blob/main/scripts/e2e-critical-components.sh)
Terraform module.

#### SPIFFE Broker API and Bootstrap

Only the object-level features will require access to the SPIFFE Broker API.
This means inter-controller private communication does not require access to
the SPIFFE Broker API, only SPIFFE Workload API.

Accessing the SPIFFE Broker API is possible over TCP, so this dependency
is quite simple and does not introduce concerns around privileged kernel
resources and the `Restricted` Pod Security Standard that Flux opts into.

No bootstrap concerns.

### User Stories

The new user stories introduced by this RFC fall under the following categories:

- Using Kubernetes ServiceAccount tokens or SVIDs for `generic` OCI registries.
- Exchanging SVIDs for cloud provider credentials when using cloud provider services.
- Exchanging JWT-SVIDs for OpenBao/Vault access tokens when decrypting via SOPS.
- Protecting traffic between Flux controllers and external systems with SPIFFE TLS.
- Protecting internal traffic between Flux controllers with SPIFFE TLS.

The exhaustive list of stories is too long, so below we describe
representative examples of each category.

#### Story 1

> As a user, I want to use Kubernetes ServiceAccount tokens to authenticate
> with my Zot registry using its OIDC workload identity federation feature.

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: OCIRepository
metadata:
  name: my-oci-repo
  namespace: my-namespace
spec:
  url: oci://my-zot-registry.example.com/my-repo
  provider: generic
  serviceAccountName: my-service-account
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-service-account
  namespace: my-namespace
  annotations:
    credential.auth.fluxcd.io/provider: kubernetes
    credential.auth.fluxcd.io/type: jwt
```

#### Story 2

> My company requires X.509 PKI for exchanging identities with AWS
> through the feature AWS IAM Roles Anywhere. As a user, I want to
> use X509-SVID for authenticating into ECR.

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: OCIRepository
metadata:
  name: my-oci-repo
  namespace: my-namespace
spec:
  url: oci://my-ecr-registry.example.com/my-repo
  provider: aws
  serviceAccountName: my-service-account
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-service-account
  namespace: my-namespace
  annotations:
    credential.auth.fluxcd.io/provider: spiffe
    credential.auth.fluxcd.io/type: x509
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/my-iam-role
```

#### Story 3

> As a user, I want to login into OpenBao using JWT-SVIDs for
> decrypting secrets with SOPS.

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: my-kustomization
  namespace: my-namespace
spec:
  decryption:
    provider: sops
    serviceAccountName: my-service-account
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-service-account
  namespace: my-namespace
  annotations:
    credential.auth.fluxcd.io/provider: spiffe
    credential.auth.fluxcd.io/type: jwt
    transit-engine.auth.fluxcd.io/login-path: auth/jwt/login
    transit-engine.auth.fluxcd.io/role: my-role
```

#### Story 4

> As a user, I want to use a JWT-SVID to authenticate with my
> remote self-managed cluster.

```yaml
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: my-helm-release
  namespace: my-namespace
spec:
  kubeConfig:
    configMapRef:
      name: my-kubeconfig-configmap
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-kubeconfig-configmap
  namespace: my-namespace
data:
  provider: generic
  serviceAccountName: my-service-account
  address: https://my-remote-cluster.example.com:6443
  # audiences: ... # Rejected when using new annotations.
  ca.crt: |
    -----BEGIN CERTIFICATE-----
    ...
    -----END CERTIFICATE-----
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-service-account
  namespace: my-namespace
  annotations:
    credential.auth.fluxcd.io/provider: spiffe
    credential.auth.fluxcd.io/type: jwt
    credential.auth.fluxcd.io/audiences: |
      my-audience
```

#### Story 5

> As a user, I want my Flux controllers to only talk to each
> other over SPIFFE mTLS.

We omit the detailed configuration for this story, as it is already described in
the [Inter-Controller Communication](#inter-controller-communication) section.

### Alternatives

An obvious alternative to consider for this RFC is integrating with
cert-manager. The limitation eliminating cert-manager is that it only
supports X.509 credentials. In some cases, only OIDC JWTs are supported,
e.g. Azure.

## Design Details

### Libraries in `github.com/fluxcd/pkg`

All the annotations, old and new, ours and cloud provider ones, will
be defined in a new package `github.com/fluxcd/pkg/apis/crypto` and
consumed from there everywhere else. The API validation rules defined
in the next section will be implemented in that package.

A new package `github.com/fluxcd/pkg/spiffe` will be created to implement
the private communication features (inter-controller communication and
the object-level SPIFFE mTLS feature), and also building blocks for
`github.com/fluxcd/pkg/auth` to actually implement the remaining
features. If in the future non-SPIFFE mTLS mechanisms are introduced,
the `spiffe` package introduced by this RFC will shift to provide building
blocks for a new `github.com/fluxcd/pkg/mtls` package, similar to the
relationship between `spiffe` and `auth`.

### API Validations and Backward Compatibility Guarantees

Flux provides only backward compatibility guarantees.
Forward compatibility is not guaranteed. These principles
help us evolve our APIs maximizing the incentive for users
to migrate to the newer ones, but at their own pace by
avoiding breaking changes.

The rules below force users to fully opt into the
annotation-based configuration API if they want to
use the new features introduced by this RFC, while
existing features remain supported without breaking
changes.

The rules are grouped by concern: opting into object-level
workload identity, opting into mTLS, conflicts with
legacy configuration and provider-specific constraints.

#### Object-level workload identity

- The annotations `credential.auth.fluxcd.io/provider`
  and `credential.auth.fluxcd.io/type` must always be
  specified together.
- Every annotation introduced by this RFC requires both
  of them, with the sole exception of the
  `*.mtls.fluxcd.io/*` annotations (see the next
  subsection).
- When neither is specified, we assume `kubernetes` and
  `jwt` for backward compatibility with RFC-0010, and
  the annotations introduced by RFC-0010, i.e.
  `gcp.auth.fluxcd.io/workload-identity-provider`,
  `eks.amazonaws.com/role-arn`,
  `azure.workload.identity/client-id`,
  `azure.workload.identity/tenant-id` and
  `iam.gke.io/gcp-service-account`, keep working as
  before.

#### mTLS

- The `*.mtls.fluxcd.io/*` annotations opt a Flux object
  into mTLS with the target service, independently of
  credential acquisition.
- On their own they require neither
  `credential.auth.fluxcd.io/provider` nor
  `credential.auth.fluxcd.io/type`, so a ServiceAccount
  can be configured for mTLS without opting into
  object-level workload identity at all.
- If any authentication annotation is present alongside mTLS
  annotations, however, i.e. any annotation that configures
  credential acquisition, including the ones introduced
  by RFC-0010, then `/provider` and `/type` must also be
  present. This keeps object-level workload identity an
  explicit choice whenever mTLS is used.
- They reject `.spec.certSecretRef`, and certificates or
  CAs coming from `.spec.secretRef`.
- They reject the public service providers `aws`, `azure`,
  `gcp` and `dockerhub`. They also reject `github`, but only
  when it's GitHub.com: GitHub Enterprise Server can be
  self-hosted and support SPIFFE mTLS.
- These rules are backward compatible because they only
  take place when mTLS annotations are present.

#### Conflicts with legacy configuration

For using any of the annotations introduced by this RFC,
the Flux object must not use any of the following legacy
configurations (which are kept for backward compatibility).

- A ConfigMap referenced by `.spec.kubeConfig.configMapRef`
  that has the `audiences` key must migrate to
  `credential.auth.fluxcd.io/audiences`.
- A Kustomization using SOPS decryption with the
  `--sops-vault-configmap` flag introduced in Flux 2.9
  must migrate to `transit-engine.auth.fluxcd.io/login-path`
  and `transit-engine.auth.fluxcd.io/role`, which must
  always be specified together.
- A Bucket with `.spec.sts.provider` set to `aws` must
  migrate to `aws.auth.fluxcd.io/sts-endpoint`.

#### Provider-specific constraints

- At object level, the new OCI service providers, `generic`
  and `dockerhub`, require the new annotations.
- In the OCI APIs, a ServiceAccount for the `generic` service
  provider that contains `.imagePullSecrets` authenticates with
  those static credentials instead of object-level workload
  identity, so it cannot use the new authentication annotations;
  mTLS annotations remain allowed.
- `credential.auth.fluxcd.io/username` is only valid for
  the OCI APIs (`OCIRepository` and `ImageRepository`) and
  only for the `generic` and `dockerhub` service providers:
  optional for `generic` and mandatory for `dockerhub`.
  It is rejected everywhere else.
- `dockerhub` requires
  `dockerhub.auth.fluxcd.io/oidc-connection-id`.
- `credential.auth.fluxcd.io/audiences` is only valid
  with the `jwt` credential type. It is rejected for the
  `dockerhub` service provider, which only supports the
  audience `https://identity.docker.com`.
- The `kubernetes` credential provider supports only the
  `jwt` credential type, and the `azure` service provider
  supports only the `jwt` credential type.

### SPIFFE IDs, Prefixes and Trust Domains

All of the SPIFFE IDs in this proposal, i.e. the values of the
`spiffe.mtls.fluxcd.io/server-ids` annotation and all the
`...-spiffe-id` controller flags plus `--spiffe-broker-id`, accept
either an exact SPIFFE ID or a SPIFFE ID prefix. All the
`...-spiffe-id` controller flags are repeatable, so they can be
specified multiple times to authorize multiple IDs or prefixes. A
value that ends with a slash (`/`) is treated as a prefix and
authorizes any SPIFFE ID that starts with it. Prefixes are what make
the flags practical for authorizing scale-out peers with unique
SPIFFE IDs per replica that still share a common prefix (like Broker
API SPIRE agents). The trailing slash is what identifies a
prefix because trailing slashes are not allowed in the SPIFFE
ID standard, therefore an ID with a trailing slash must
necessarily mean a prefix, and not a fully-specified ID.

A SPIFFE ID without a path means matching any SPIFFE ID in the
trust domain of this ID.

The special symbol `self` can be used to indicate that the
trust domain is actually our own trust domain (e.g.
`spiffe://self`, `spiffe://self/<path>` or `spiffe://self/<path>/`).
Note that "our own trust domain" has different meanings depending on
the context in which it is used. For inter-controller communication,
"our own trust domain" refers to the trust domain of the controller
itself, observed through the controller's own X509-SVID, obtained via
the SPIFFE Workload API. For `spiffe.mtls.fluxcd.io/server-ids`, "our
own trust domain" refers to the trust domain of the X509-SVID of the
Flux object, acquired by Flux through the SPIFFE Broker API.

### Client-Side Load Balancing for the SPIFFE Broker Endpoint

We can employ a good default client-side load-balancing strategy,
e.g. round-robin, which is a better load-balancing strategy for a
gRPC Service in Kubernetes (when a service mesh is not available)
than the default pin-to-a-pod strategy. If the Kubernetes Service
is [headless](https://kubernetes.io/docs/concepts/services-networking/service/#headless-services)
i.e. it returns the endpoints of individual pods, then the gRPC
library is able to perform client-side load-balancing.

Because the most commonly available implementations of load-balancing
for Kubernetes Services do not natively understand gRPC traffic (as
they usually work at the TCP level), they are not able to perform
proper load-balancing for a long-lived incoming gRPC connection
multiplexing multiple streams concurrently. This is why client
gRPC libraries implement their own load-balancing mechanisms.

If the Kubernetes Service is a simple `ClusterIP` service that
returns a single virtual endpoint, our setup of the connection
will work the same way (and the Service can change to a headless
one later).

A future improvement we can make based on demand is implementing
an EndpointSlice-aware periodic resolver to fix issues with the
client-side load-balancing degenerating over time due to new pods
coming up and old ones leaving. As a reference implementation, we
can use the retired `github.com/sercand/kuberesolver/v6`.

### Artifact Server mTLS Migration

The controllers that act as artifact clients, i.e. kustomize-controller,
helm-controller and source-watcher, when configured with authorized
SPIFFE IDs for artifact servers, such as source-controller, source-watcher
and other external source controllers, will always force artifact
URLs to HTTPS in memory. This allows the migration to be fully stateless,
i.e. no need to change `.status.artifact.url` in any source objects.
For compatibility with the inter-controller SPIFFE-backed mTLS feature
in Flux >=2.10, external source controllers must support the same
feature.

### SPIFFE TLS with the Kubernetes API Server

It has been noted that people are not using SPIFFE for the Kubernetes
API Server TLS certificate much in the wild, if at all. For this reason,
we will wait until users ask for this feature to implement it, but the
controller options and annotations designed here for this use case must
be honored when this feature is eventually implemented.

### No Feature Gate or Multi-Tenancy Lockdown

No need for a feature gate. We shipped `ObjectLevelWorkloadIdentity` because we were afraid of our
design choice being abusive with Kubernetes, i.e. we were afraid of issuing ServiceAccount tokens
for CRs to achieve workload identity being a non-intended use. Kubernetes maintainers confirmed
it's ok to do this, so this gate doesn't make sense anymore, we should remove it at some point.
SPIFFE is literally designed for workload identity so there's no abuse. The gates are the go-spiffe
environment variable configuring the Workload API UDS socket in the controller, and the controller
flags configuring the Broker API TCP endpoint. If SPIFFE is to remain disabled, don't configure
those settings in the controller.

Multi-tenancy lockdown as a responsibility is shifted to SPIFFE. The Broker API
KubernetesObjectReference is a fully-qualified identifier for a Kubernetes object,
and it contains the namespace, which is the multi-tenancy boundary in Kubernetes.
The SPIFFE runtime will determine the object identity based on this namespaced
object reference. Cluster administrators may or may not configure namespaced
SPIFFE IDs for the Flux objects, and it's this choice that enables or disables
multi-tenancy lockdown. This is a major aspect of SPIFFE: SPIFFE decides the
identity of the workload or object. A SPIFFE workload has no opinion on what
SPIFFE ID it should receive from the SPIFFE runtime. Therefore, it's not possible
and it does not make sense to enforce multi-tenancy lockdown on the Flux side
when SPIFFE completely takes over the identity system.

## Implementation History

### Flux 2.10

In this GA release of the feature, we renamed it from "object-level workload identity" 
to "object identity" to have a better separation: now "workload identity" means
exclusively identities assigned to the Flux controllers, and "object identity"
means identities assigned to Flux objects.

Inter-controller mTLS is introduced as an entirely new feature.

Features in this release:

- Kubernetes ServiceAccount tokens for `generic` OCI registries
- Inter-controller mTLS via SPIFFE Workload API
- Object identity via SPIFFE Broker API
