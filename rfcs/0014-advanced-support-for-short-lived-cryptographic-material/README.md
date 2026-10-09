# RFC-0014 Advanced Support for Short-Lived Cryptographic Material

**Status:** implementable

<!--
Status represents the current state of the RFC.
Must be one of `provisional`, `implementable`, `implemented`, `deferred`, `rejected`, `withdrawn`, or `replaced`.
-->

**Creation date:** 2026-02-01

**Last update:** 2026-10-06

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

This RFC takes RFC-0010 to the next level in four dimensions:

- Flux will be able to support workload identity for more vendor-neutral
  infrastructure components, such as OCI registries like Harbor and
  Zot (both CNCF projects) that have implemented support for workload
  identity. RFC-0010 introduced workload identity for remote Kubernetes
  clusters, and Flux 2.9 introduced workload identity for OpenBao/Vault.
  By supporting workload identity for OCI registries like Harbor and Zot,
  Flux will support workload identity for opensource projects covering
  three types of services that are core to Flux: Kubernetes, OCI registries
  and key management systems for decryption.
- Flux will solve four classes of *confused deputy* problems by supporting
  the identity of a Flux Custom Resource object to be the object itself
  i.e. the object's Group-Kind-Namespace-Name-UID quintuple through a SPIFFE
  KubernetesObjectReference, instead of the configurable field
  `.serviceAccountName`:
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
- Flux will no longer need the `create` Kubernetes RBAC verb for the
  `serviceaccounts/token` resource at the cluster scope. A major RBAC
  improvement. Flux Operator, which opts out from this RBAC by default,
  would stay opting out when choosing SPIFFE.

### Goals

1. Defining the common APIs that will support the use cases described in
this RFC in a way that they can be uniformly implemented across all the
Flux components (given enough release cycles).

2. Indulge in the "SPIFFE is the bottom turtle" philosophy. Allow Flux
to participate with first-class support in the SPIFFE landscape of an
organization aiming for centralized management of short-lived
cryptographic material.

### Non-Goals

It's not a goal of this RFC to make Flux become a SPIFFE runtime,
like SPIRE (the reference runtime implementation). The whole point of
SPIFFE is providing a simple interface through which applications can
acquire short-lived cryptographic material from the infrastructure,
allowing the ownership of the signing keys to remain with the
infrastructure rather than with the application. From this RFC's
perspective Flux is the application, and whatever SPIFFE runtime
the user has deployed is the infrastructure. There are both free
(SPIRE) and commercial options available for a SPIFFE runtime.

## Proposal

The proposal covers API fields, controller options, new dependencies and bootstrap.

The CLI command `flux push artifact` and family can get the same
features as `flux-mirror` (e.g. GitHub/Forgejo Actions OIDC, etc.),
but the flags will likely be different. Because the discussion
around this set of CLI commands applies only to them, we will
leave it out of the scope of this RFC. This RFC is focused
exclusively on the common API fields and controller options
that will be reused as standards across the Flux controllers.

### API Fields

The proposal is heavily inspired by what we already implemented in
[`flux-mirror`](https://fluxcd.io/flux/cli-plugins/flux-mirror/config/#hosts).

The following layout can be applied either at top-level `.spec` fields (e.g. for OCIRepository, Provider, etc.),
or under fields like `.spec.kubeConfig` and `.spec.decryption` (remote clusters and decryption in Flux applier
APIs). As of today, these two are the only non-top-level fields for which the short-lived cryptographic material
features apply, but if in the future we add more API fields for more services that can integrate with these
features, we would apply the same layout to them. Particularly for `.spec.decryption`, see
[OpenBao/Vault API Fields](#openbaovault-api-fields).

```yaml
spec:

  # The following are the existing fields that will interact with the new fields proposed here:
  provider: generic # Or aws, azure, gcp.
  serviceAccountName: my-service-account # Used when credential.provider=kubernetes or credential is unset.
  certSecretRef: # Used when tls.clientAuth.provider=secret or tls.serverAuth.provider=secret or tls is unset.
    name: my-cert-secret
  secretRef: # Some Flux CRDs support certificates through this field instead of certSecretRef (e.g. GitRepository).
             # This field could also be used to provide static credentials sent through a SPIFFE mTLS connection.
    name: my-secret

  # Proposed new fields:
  credential:
    provider: kubernetes # Or spiffe.
    type: jwt # Or x509. spec.provider=azure supports only jwt.

    # Supported only for credential.provider=kubernetes. Also requires either the serviceAccountName
    # field to be set, or the respective --default.*-service-account flag to be set in the controller
    # (.* here matches one of "", "-decryption" or "-kubeconfig").
    expirationSeconds: 3600
    audiences: # Applicable only for type=jwt. Often defaults to the URL of the service being accessed.
      - my-audience

    # The annotations we currently support for ServiceAccounts will now also be accepted
    # directly in the credential configuration. If also set in the ServiceAccount object,
    # the values must match. Here, they will have proper fields per cloud provider. Each
    # provider has its own requirements for each field, we will keep the existing behavior.

    # Optional.
    aws:
      # Required. Annotation: eks.amazonaws.com/role-arn
      roleARN: arn:aws:iam::<account-id>:role/<role-name>

    # Optional.
    azure:
      # Required. Annotation: azure.workload.identity/client-id
      clientID: <client-id>
      # Required. Annotation: azure.workload.identity/tenant-id
      tenantID: <tenant-id>

    # Optional.
    gcp:
      # Optional. Annotation: iam.gke.io/gcp-service-account
      serviceAccountEmail: <sa name>@<project>.iam.gserviceaccount.com

      # The following annotation is used for cross-cloud/self-managed clusters accessing GCP resources.
      # Today, this annotation is accepted only in ServiceAccount objects. But SPIFFE is fully decoupled
      # from Kubernetes ServiceAccounts, so we need to accept this annotation also in the Flux Custom
      # Resource object itself. When using ServiceAccounts, the annotation can appear both in the
      # ServiceAccount object and in the Flux Custom Resource object itself, but the value has to match.
      # Optional. Annotation: gcp.auth.fluxcd.io/workload-identity-provider
      workloadIdentityProvider: projects/my-project/locations/global/workloadIdentityPools/my-pool/providers/my-provider

    # This optional field is only for OCI Apis and will stay out of the apis/crypto package.
    # Indicates that a JWT credential should be used as the password in a username/password pair.
    # OCI registries often support both username/password and Bearer token authentication for OIDC
    # JWTs, and sometimes only one of the two.
    username: my-username

    # This required field is only for Kustomization.spec.decryption and will stay out
    # of the apis/crypto package.
    # OpenBao/Vault uses "role" to represent the identity for which a login request
    # is being requested. It has rules to match the assertions present inside the
    # cryptographic material and has permissions associated with that role granting
    # access to resources exposed by the OpenBao/Vault server.
    role: <role-name>

  tls:
    # Client and server authentication are independent. A user could use spiffe for one and secret for the other.
    clientAuth:
      provider: secret # Or kubernetes, or spiffe. secret means spec.certSecretRef/spec.secretRef.
    serverAuth:
      provider: secret # Or spiffe. kubernetes is not supported.
      spiffeID: spiffe://example.org/registry    # An exact SPIFFE ID for the server.
      # spiffeID: spiffe://example.org/registry/ # A SPIFFE ID prefix, implied by the trailing slash (`/`).
      # spiffeID: spiffe://example.org           # Any SVID in this trust domain.
      # spiffeID: spiffe://self                  # Any SVID in our own trust domain (`self` is a special value).
```

#### Redundant and Inconsistent Configurations

First of all, note that the proposal above introduces field combinations that
extend only the expressivity of some features i.e. they don't change the
behavior. In other words, some features will become expressible in multiple
ways. For example, the following two OCIRepository specs would be functionally
equivalent:

```yaml
spec:
  provider: azure
  serviceAccountName: my-service-account
---
spec:
  provider: azure
  serviceAccountName: my-service-account
  # The following is redundant, but accurately expresses the intent
  # among the alternatives (e.g. provider=spiffe).
  credential:
    provider: kubernetes
    type: jwt
```

For the sake of establishing a non-breaking-but-well-designed API,
we need to accept both forms. Another OCIRepository example:

```yaml
spec:
  provider: generic
  certSecretRef:
    name: my-cert-secret
---
spec:
  provider: generic
  certSecretRef:
    name: my-cert-secret
  # The following is redundant, but accurately expresses the intent
  # among the alternatives (e.g. provider=spiffe).
  tls:
    clientAuth:
      provider: secret
    serverAuth:
      provider: secret
```

Another one, this time for a Flux Kustomization:

```yaml
spec:
  kubeConfig:
    configMapRef:
      name: my-kubeconfig
    # New fields to make all the APIs consistent:
    provider: aws
    cluster: arn:<partition>:eks:<region>:<account-id>:cluster/<cluster-name>
    serviceAccountName: my-service-account
    credential:
      provider: kubernetes
      type: jwt
      audiences: # Proper list of strings.
        - sts.amazonaws.com
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-kubeconfig
  namespace: my-namespace
data:
  # Address and CA are the only fields that should remain specified via ConfigMap, as they
  # are often converted from kubeconfig Secrets via ResourceSet convertKubeConfigFrom, or
  # generated in other dynamic ways.
  address: https://<cluster-endpoint>
  ca.crt: |
    -----BEGIN CERTIFICATE-----
    ...
    -----END CERTIFICATE-----
  # The following are redundant, as we already specified them in the Flux Kustomization itself.
  provider: aws
  cluster: arn:<partition>:eks:<region>:<account-id>:cluster/<cluster-name>
  serviceAccountName: my-service-account
  audiences: | # Line-break-separated list of strings.
    sts.amazonaws.com
```

All redundant-but-consistent configurations will be accepted.

Inconsistent configurations will be rejected via CEL expressions
in the Flux CRDs when the inconsistency can be fully detected
having only the Flux CR at hand, e.g. in the OCIRepository examples
above only the OCIRepository is needed. Otherwise, the
controller reconciliation logic will do it when another API object
is involved, e.g. in the Kustomization example above both the
Kustomization and the ConfigMap are needed.

We leave the exhaustive list of all possible configuration
inconsistencies omitted from this text, but all of them will
be covered in the implementation.

#### Incompatible Configurations

Some configurations, despite consistent-looking, may still have
incompatibilities. For example, the following OCIRepository spec
contains an incompatibility:

```yaml
spec:
  provider: aws
  serviceAccountName: my-service-account
  credential:
    provider: kubernetes
    type: jwt
  tls:
    clientAuth:
      provider: spiffe
    serverAuth:
      provider: secret
```

The incompatibility here is trying to use TLS features with ECR.
ECR only serves publicly-trusted TLS certificates and does not
support TLS client certificates. Another example:

```yaml
spec:
  provider: azure
  credential:
    provider: spiffe
    type: x509
```

Here, the incompatibility is trying to exchange an X.509 certificate
for an Azure credential. Azure only supports JWTs. Only AWS and GCP
support both JWTs and X.509 certificates. AWS accepts the certificate
as a request input, while GCP accepts it literally as a TLS client
certificate for a connection with the GCP API that returns the GCP
credential.

Just like we will reject configuration inconsistencies, we will also
reject configuration incompatibilities, and in the same way: via CEL
expressions in CRDs when applicable, and via controller logic when
otherwise needed.

We leave the exhaustive list of all the possible configuration
incompatibilities omitted from this text, but all of them will
be covered in the implementation.

### Controller Options

The new controller options cover both of the following:

- The features defined through the API fields across the Flux Custom Resources.
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
defined through API fields across the Flux Custom Resources,
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
defined through API fields across the Flux Custom Resources. They do not
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
> with my Zot registry using its OIDC workload identity federation feature,
> and protect the connection using SPIFFE TLS.

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
  tls:
    serverAuth:
      provider: spiffe
      spiffeID: spiffe://<trust domain>/my-zot-registry
```

#### Story 2

> As a user, my company requires X.509 PKI for exchanging identities with
> AWS through the feature AWS IAM Roles Anywhere. I want to use X509-SVID
> for authenticating into ECR.

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: OCIRepository
metadata:
  name: my-oci-repo
  namespace: my-namespace
spec:
  url: oci://my-ecr-registry.example.com/my-repo
  provider: aws
  credential:
    provider: spiffe
    type: x509
```

#### Story 3

> As a user, I want to login into OpenBao using JWT-SVIDs for decrypting
> secrets with SOPS. One role per namespace.

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: my-kustomization
  namespace: my-namespace
spec:
  decryption:
    provider: sops
    transitEngines:
      - address: https://openbao.example.com:8200
        loginPath: auth/jwt/login
        credential:
          provider: spiffe
          type: jwt
          role: my-role
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
    provider: generic
    credential:
      provider: spiffe
      type: jwt
      audiences: [my-remote-cluster] # Overrides the default [https://my-remote-cluster.example.com:6443]
    configMapRef:
      name: my-kubeconfig-configmap
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-kubeconfig-configmap
  namespace: my-namespace
data:
  address: https://my-remote-cluster.example.com:6443
  ca.crt: |
    -----BEGIN CERTIFICATE-----
    ...
    -----END CERTIFICATE-----
```

#### Story 5

> As a user, I want my Flux controllers to only talk to each
> other over SPIFFE mTLS.

We omit the detailed configuration for this story, as it is already described in
the [Inter-Controller Communication](#inter-controller-communication) section.

### Alternatives

An obvious alternative to consider here is integrating with cert-manager.
The limitations that eliminate cert-manager are:

- X.509-only. In some cases, only OIDC JWTs are supported, e.g. Azure.
- Pod-scoped, no support for multi-tenancy. We need object-level identities.

## Design Details

### OCI APIs and Pull Secrets Referenced in ServiceAccounts

Today, `OCIRepository` and `ImageRepository` with `.spec.provider` set
to `generic` use `.spec.serviceAccountName` for reading the field
`ServiceAccount.imagePullSecrets` i.e. static credentials. To avoid
a major breaking change, the controllers must look into this field
before deciding to choose the workload identity path. If the field
contains pull Secrets, they must be used instead of workload identity,
unless, however, if the `.spec.credential` field is explicitly stating
that workload identity should be used by setting the `kubernetes`
provider and the `jwt` type. This makes the implementation fully
backwards compatible.

We understand that this is not backwards compatible for someone
currently using the `.serviceAccountName` field pointing to a
ServiceAccount that contains no pull Secrets. This is, however,
a configuration that is not currently supported, so we accept
this breaking change. It will surface the configuration mistake
for these users.

### SPIFFE IDs, Prefixes and Trust Domains

All of the SPIFFE IDs in this proposal, i.e. the API field
`.tls.serverAuth.spiffeID` and all the `...-spiffe-id`
controller flags plus `--spiffe-broker-id`, accept either an
exact SPIFFE ID or a SPIFFE ID prefix. A value that ends with
a slash (`/`) is treated as a prefix and authorizes any SPIFFE
ID that starts with it. Prefixes are what make the flags
practical for scale-out peers. The trailing slash is what
identifies a prefix because SPIFFE IDs with trailing slashes
are not allowed by the SPIFFE standard, therefore an ID with
a trailing slash must necessarily mean a prefix, and not a
fully-specified ID.

A SPIFFE ID without a path means matching any SPIFFE ID in the
trust domain of this ID. The SPIFFE ID `spiffe://self` means
matching any SPIFFE ID in our own trust domain. Note that "our
own trust domain" has different meanings depending on the context
in which it is used. For inter-controller communication, "our own
trust domain" refers to the trust domain of the controller itself,
observed through the controller's own X509-SVID, obtained via
the SPIFFE Workload API. For the field `.tls.serverAuth.spiffeID`
in a Flux Custom Resource, "our own trust domain" refers to the
trust domain of the X509-SVID associated with the Custom Resource,
obtained via the SPIFFE Broker API.

### Libraries in `github.com/fluxcd/pkg`

Structs for the two new API fields `.credential` and `.tls` will
be defined in a new package `github.com/fluxcd/pkg/apis/crypto`. The
controller `api/` packages will import this new package to define the new
API fields across the structs of the Flux Custom Resources.

A new package `github.com/fluxcd/pkg/spiffe` will be created to implement
the private communication features (inter-controller communication and
`.tls`), and also building blocks for `github.com/fluxcd/pkg/auth` to
actually implement the `.credential` features.

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

### OpenBao/Vault Design Details

The Kustomization API field `.spec.decryption` targets a set of
OpenBao/Vault instances, not just one. So we propose the vendor-neutral
term coined initially by Vault and inherited by OpenBao *transit engine*
to represent this set of target OpenBao/Vault instances under the
`.spec.decryption` field:

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata: ...
spec:
  decryption:
    provider: sops
    serviceAccountName: my-service-account # For the Kubernetes JWT.
    transitEngines:
      - address: https://openbao2.example.com:8200
        loginPath: auth/jwt/login
        credential:
          provider: kubernetes
          type: jwt
          role: role-a
      - address: https://openbao.example.com:8200
        loginPath: auth/jwt/login
        credential:
          provider: spiffe
          type: jwt
          role: role-b
```

This supersedes the `--sops-vault-configmap` flag introduced in
kustomize-controller v1.9 (Flux v2.9). The flag will be kept for
backward compatibility but no longer advertised in docs. The new
API fields will not be introduced under the `.instances[]` API
loaded from the YAML file inside the ConfigMap.

### Artifact Server mTLS Migration

The controllers that act as artifact clients, i.e. kustomize-controller,
helm-controller and source-watcher, when configured with authorized
SPIFFE IDs for artifact servers, such as source-controller, source-watcher
and other external source controllers, will always force artifact
URLs to HTTPS in memory. This allows the migration to be fully staleless,
i.e. no need to change `.status.artifact.url` in any source objects.
For compatibility with the inter-controller SPIFFE-backed mTLS feature
in Flux >=2.10, external source controllers must support the same
feature.

### Kubernetes `certv1.CertificateSigningRequest`

For implementing `.credential.provider` set to `kubernetes` together
with `.credential.type` set to `x509`, the controller will use the
Kubernetes `certv1.CertificateSigningRequest` API with the cluster's
built-in signer `kubernetes.io/kube-apiserver-client` to issue an
X.509 client certificate representing the configured
`.serviceAccountName`, or a default ServiceAccount set through one
of the workload identity multi-tenancy lockdown flags from RFC-0010.
This will require the following additional RBAC for the controllers
that support workload identity.

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: flux-csr-issuer
rules:
  # Create the CSR and read it back (poll .status.certificate).
  - apiGroups: ["certificates.k8s.io"]
    resources: ["certificatesigningrequests"]
    verbs: ["create", "get", "list", "watch"]
  # Approve it.
  - apiGroups: ["certificates.k8s.io"]
    resources: ["certificatesigningrequests/approval"]
    verbs: ["update"]
  - apiGroups: ["certificates.k8s.io"]
    resources: ["signers"]
    resourceNames: ["kubernetes.io/kube-apiserver-client"]
    verbs: ["approve"]
```

The RBAC above is functionally equivalent to the `create`
verb on the `serviceaccounts/token` resource at the cluster
scope that was introduced in RFC-0010. Both are also
functionally equivalent to `cluster-admin`. Flux lifecycle
management tools like Flux Operator should add both RBACs
only when enabling the feature gate
`ObjectLevelWorkloadIdentity`.

### SPIFFE TLS with the Kubernetes API Server

It has been noted that people are not using SPIFFE for the Kubernetes
API Server TLS certificate much in the wild, if at all. For this reason,
we will wait until users ask for this feature to implement it, but the
controller options and API fields designed here for this use case must
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
identity of the workload or object. A SPIFFE Workload has simply no opinion on
what SPIFFE ID it will receive from the SPIFFE runtime. It simply trusts the
runtime on the other side of the Workload API UDS socket. Therefore, it's not
possible and it does not make sense to enforce multi-tenancy lockdown on the Flux
side when SPIFFE completely takes over.

## Implementation History

<!--
Major milestones in the lifecycle of the RFC such as:
- The first Flux release where an initial version of the RFC was available.
- The version of Flux where the RFC graduated to general availability.
- The version of Flux where the RFC was retired or superseded.
-->
