# RFC-XXXX Advanced Support for Short-Lived Cryptographic Material

**Status:** provisional

<!--
Status represents the current state of the RFC.
Must be one of `provisional`, `implementable`, `implemented`, `deferred`, `rejected`, `withdrawn`, or `replaced`.
-->

**Creation date:** 2026-09-07

**Last update:** 2026-09-13

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

Many Flux users have advanced security requiments and use cases. This RFC
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
from large technology players, such as
[Uber](https://www.uber.com/us/en/blog/solving-the-agent-identity-crisis/)
applying it to solve agentic identity challenges, to
open-source projects such as service meshes and policy
engines.

Flux users have also manifested interest in integrations
with SPIFFE, such as
[#3368 (comment)](https://github.com/fluxcd/flux2/pull/3368#discussion_r1040899292)
and [#5679](https://github.com/fluxcd/flux2/discussions/5679),
plus many offline discussions in conferences.

This RFC takes RFC-0010 to the next level in two dimensions:

- Flux will be able to support workload identity for more vendor-neutral
  infrastructure components, such as container registries like Harbor and
  Zot (both CNCF projects) that have implemented support for workload
  identity. RFC-0010 introduced workload identity for remote Kubernetes
  clusters, and Flux 2.9 introduced workload identity for OpenBao/Vault.
  By supporting workload identity for container registries Flux will be
  covering workload identity for all the vendor-neutral infrastructure
  components that are core to Flux: Kubernetes, container registries
  and key management systems for decryption.
- Flux will support both an out-of-the-box provider for workload identity,
  which is Kubernetes itself through ServiceAccount tokens, and a more
  advanced solution that covers more security use cases beyond workload
  identity, such as private communication, and aims specifically at
  integration with centralized management of short-lived cryptographic
  material. By using the JWT PKI provided by Kubernetes Flux users can
  go a long way, but each cluster will have its own key pair and federating
  several clusters in a consistent way can be a challenge. On the other
  hand, because SPIFFE's main goal is to solve this very problem, it
  came up with now-established ways for different SPIFFE runtimes to
  federate and provide a unified layer of trust across clusters and the
  entire infrastructure of an organization. "SPIFFE is the bottom turtle"
  is the motto of the project, to emphasize their fundemental goal of
  being the cornerstone for PKI across the board.

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

The proposal covers API fields and controller options. The
proposal is heavily inspired by what we already implemented in
[`flux-mirror`](https://fluxcd.io/flux/cli-plugins/flux-mirror/config/#hosts).

The CLI command `flux push artifact` and family can get the same
features as `flux-mirror` (e.g. GitHub/Forgejo Actions OIDC, etc.),
but the flags will likely be different. Because the discussion
around this set of CLI commands applies only to them, we will
leave it out of the scope of this RFC. This RFC is focused
exclusively on the common API fields and controller options
that will be reused as standards across the Flux controllers.

### API fields

The following layout can be applied either at top-level `.spec` fields (e.g. for OCIRepository, Provider, etc.),
or under fields like `.spec.kubeConfig` and `.spec.decryption`, which are the special cases in the applier APIs
for opt-in features like remote clusters and decryption. As of today, `.spec.kubeConfig` and `.spec.decryption`
are the only fields for which the short-lived cryptographic material features apply, but if in the future
we add more API fields that can benefit from these features, we should apply the same layout to them.

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
    type: jwt # Or x509. credential.provider=kubernetes and provider=azure support only jwt.
    audiences: # Applicable only for type=jwt. Often defaults to the URL of the service being accessed.
      - my-audience
    username: my-username # Indicates that the jwt credential should be used as a password in a user/pass pair.
                          # This has implications for specific services, such as container registries.
                          # Container registries often support both username/password and Bearer token
                          # authentication, which warrants the existence of this field.
  tls:
    # Client and server authentication are independent. A user could use spiffe for one and secret for the other.
    clientAuth:
      provider: secret # Or spiffe. secret means spec.certSecretRef/spec.secretRef.
    serverAuth:
      provider: secret # Same as clientAuth.provider.
      spiffe:
        # Exactly one authorization field must be set: serverID, trustDomain, or authorizeAny.
        serverID: spiffe://example.org/registry # An exact SPIFFE ID.
        trustDomain: example.org              # Any SVID in this trust domain.
        trustDomain: self                     # Special value for any SVID in our own trust domain.
        authorizeAny: true                    # Any SVID the bundle validates (discouraged).
```

The annotations we currently support for ServiceAccounts will now also be accepted
directly in Flux Custom Resource objects themselves as well.

```yaml
metadata:
  annotations:
    # The following annotation is used for cross-cloud / self-managed clusters accessing GCP resources.
    # Today, this annotation is accepted only in ServiceAccount objects. But SPIFFE is fully decoupled
    # from Kubernetes ServiceAccounts, so we need to accept this annotation also in the Flux Custom
    # Resource object itself. When using ServiceAccounts, the annotation can appear both in the
    # ServiceAccount object and in the Flux Custom Resource object itself, but the value has to match.
    gcp.auth.fluxcd.io/workload-identity-provider: projects/my-project/locations/global/workloadIdentityPools/my-pool/providers/my-provider

    # All the other annotations supported today only in ServiceAccounts will also need to be accepted in
    # the Flux Custom Resource object itself. These are the annotations defined by the cloud providers
    # themselves, that we chose in RFC-0010 to support also in Flux for a seamless UX.
    eks.amazonaws.com/role-arn: arn:aws:iam::<account-id>:role/<role-name>
    azure.workload.identity/client-id: <client-id>
    azure.workload.identity/tenant-id: <tenant-id>
    iam.gke.io/gcp-service-account: <sa name>@<project>.iam.gserviceaccount.com
```

#### Redundant Specs

First of all, note that the proposal above introduces fields that overlap with existing
behavior. For example, the following two OCIRepository specs would be functionally
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

For the sake of establishing a well designed API that does not break
existing behavior, we need to accept both forms. Another OCIRepository
example:

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

Verbose or not, a consistent specification will be accepted.

We will, on the other hand, implement as-tight-as-possible validations
for the configuration, as always via both CEL expressions in the Flux
CRDs and controller reconciliation logic. Which leads to the next section.

#### Tight Validations

Via both CEL expressions in the Flux CRDs and controller reconciliation
logic, we will implement as-tight-as-possible validations for the
configuration. For example, the following OCIRepository spec is invalid:

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

The incompatibility here is trying to use ECR with SPIFFE mTLS, which
is just as incompatible as any other TLS/mTLS configuration with ECR.
Another example:

```yaml
spec:
  provider: azure
  credential:
    provider: spiffe
    type: x509
```

Here, the incompatibility is trying to exchange an X509-SVID for
an Azure access token. Azure only supports JWTs. But AWS and GCP
support both JWTs and X.509 certificates. Another example:

```yaml
spec:
  provider: gcp
  serviceAccountName: my-service-account
  credential:
    provider: kubernetes
    type: x509
```

Also invalid, Kubernetes does not have an API for requesting an X.509
bundle for a ServiceAccount.

We will not write here an exhaustive list of all the possible configuration
incompatibilities, but we will implement and document all of them.

#### OpenBao/Vault Workload Identity

The `--sops-vault-configmap` flag in kustomize-controller points to a
ConfigMap in the runtime namespace of the controller containing this
content:

```yaml
data:
  config.yaml: |
    instances:
      - address: https://openbao-a.example.com:8200
        loginPath: auth/kubernetes/login
      - address: https://openbao-b.example.com:8200
        loginPath: ns1/ns2/auth/kubernetes/login
```

Besides all the fields that can be added from `.credential` and `.tls`
under `.instances[]`, a new field in particular is only relevant for the
OpenBao/Vault integration: `.roleTemplate`. A choice we needed to make when
implementing workload identity for OpenBao/Vault in Flux 2.9 was the role
format. We chose the format `{{ .Namespace }}_{{ .ServiceAccountName }}`.
SPIFFE will not use any ServiceAccounts, so this format does not work.
The `.roleTemplate` field will allow users to configure the role format
for OpenBao/Vault workload identity when using SPIFFE as the crypto
provider instead of Kubernetes.

```yaml
data:
  config.yaml: |
    instances:
      - address: https://openbao.example.com:8200
        loginPath: auth/kubernetes/login
        roleTemplate: {{ .Namespace }}_{{ .ServiceAccountName }}
```

The API for `.roleTemplate` will be the following:

- `.Name`: The name of the Flux Custom Resource object, e.g. `my-oci-repo`.
- `.Namespace`: The namespace of the Flux Custom Resource object, e.g. `my-namespace`.
- `.UID`: The UID of the Flux Custom Resource object, e.g. `f3e1c2d4-5b6a-7c8d-9e0f-1a2b3c4d5e6f`.
- `.ServiceAccountName`: The name of the ServiceAccount used by the Flux Custom Resource object (same namespace). Empty for SPIFFE.

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
which is aligned with the way how we support many of the environment
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

SPIFFE has introduced the Broker API specifically with Flux's use case in
mind. In fact, we collaborated with the SPIFFE maintainers to define the
SPIFFE Broker API. In particular, we proposed the `KubernetesObjectReference`
reference type as part of the API.

The SPIFFE Broker API allows a SPIFFE-attested workload, i.e. a workload
that has already fetched an X509-SVID for its own identity from the SPIFFE
Workload API, to fetch SVIDs for other workloads or things e.g. Kubernetes
objects. Such an application is called a SPIFFE Broker.

Flux, acting as a SPIFFE Broker, will fetch SVIDs for the Flux
Custom Resource objects, which are Kubernetes objects, and hence
why we proposed to SPIFFE the `KubernetesObjectReference` reference
type. Whenever reconciling a Flux Custom Resource object that is
configured to use SPIFFE, the controller will interact with the
SPIFFE Broker API to fetch the SVID for that object passing a
`KubernetesObjectReference` containing the object's API group,
kind, namespace, name and UID. All of these will need to be part
of the cache key for the credential cache.

Because of SPIFFE Brokers like Flux that will request SVIDs for Kubernetes
objects that do not map to Kubernetes workloads with running process IDs
(PIDs), like Pods, Deployments and so on, whose attestation process depends
on the Linux Kernel running those processes, the SPIFFE Broker API can be
served by the SPIFFE runtime over TCP. Local attestation via UDS is not a
requirement for fetching SVIDs for Flux Kustomizations, they are not workloads.
Attesting those objects is a sequence of calls to the Kubernetes API Server
that are made by the SPIFFE runtime serving the Broker API. This TCP endpoint
is protected by SPIFFE's mTLS PKI through the X509-SVID of the SPIFFE Broker
and the trust bundle that comes along with it. These are acquired through the
SPIFFE Workload API, as explained initially.

We propose the following flag for the gRPC URL of the SPIFFE Broker Endpoint:

```sh
--spiffe-broker-endpoint=dns:///<svc name>.<namespace>.svc.cluster.local:443
```

This flag covers only the features defined through API fields across the
Flux Custom Resources. It does not cover inter-controller communication.

The flag is mandatory for the controller to be able to reconcile Flux
Custom Resource objects that are configured to use SPIFFE. The controller
will not be able to reconcile those objects if the flag is not set, and
will log an error message to that effect.

#### Inter-Controller Communication

For inter-controller communication, the controller's own X509-SVID
is enough for mTLS authentication and authorization, which means
only the `SPIFFE_ENDPOINT_SOCKET` environment variable is needed
for acquiring the controller's own X509-SVID via SPIFFE Workload
API.

However, in addition to acquiring its own X509-SVID, the controller
also needs to authorize SPIFFE IDs of the other controllers. This
warrants a new set of flags in specific controllers.

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
--notification-controller-spiffe-id=spiffe://<trust domain>/notification-controller
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
--kube-apiserver-spiffe-id=spiffe://<trust domain>/kube-apiserver
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

Implementing Kubernetes ServiceAccount tokens for container registries
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

#### SPIFFE Broker API

Only the object-level features will require access to the SPIFFE Broker API.
This means inter-controller private communication does not require access to
the SPIFFE Broker API, only SPIFFE Workload API.

Accessing the SPIFFE Broker API is possible over TCP, so this dependency
is quite simple and does not introduce concerns around privileged kernel
resources and the `Restricted` Pod Security Standard that Flux opts into.

No bootstrap concerns.

### User Stories

The new user stories introduced by this RFC fall under the following categories:

- Using Kubernetes ServiceAccount tokens or SVIDs for `generic` container registries.
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
      spiffe:
        serverID: spiffe://<trust domain>/my-zot-registry
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
    credential:
      provider: spiffe
      type: jwt
---
# kustomize-controller --sops-vault-configmap
data:
  config.yaml: |
    instances:
      - address: https://openbao.example.com:8200
        loginPath: auth/kubernetes/login
        roleTemplate: flux_kustomization_{{ .Namespace }}
```

#### Story 4

> As a user, I want to use SPIFFE TLS to talk to my remote self-managed
> cluster for applying resources, while using the existing method of
> authenticating with ServiceAccount tokens from the local cluster.

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: my-kustomization
  namespace: my-namespace
spec:
  kubeConfig:
    provider: generic
    serviceAccountName: my-service-account
    configMapRef:
      name: my-kubeconfig-configmap
    tls:
      serverAuth:
        provider: spiffe
        spiffe:
          serverID: spiffe://<trust domain>/my-remote-cluster
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-kubeconfig-configmap
  namespace: my-namespace
data:
  address: https://my-remote-cluster.example.com:6443
```

#### Story 5

> As a user, I want my Flux controllers to only talk to each
> other over SPIFFE mTLS.

We omit the detailed configuration for this story, as it is already described in
the [Inter-Controller Communication](#inter-controller-communication) section.

## Implementation Details

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
than the default pin-to-a-pod strategy. If the Kuberntes Service
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

## Implementation History

<!--
Major milestones in the lifecycle of the RFC such as:
- The first Flux release where an initial version of the RFC was available.
- The version of Flux where the RFC graduated to general availability.
- The version of Flux where the RFC was retired or superseded.
-->
