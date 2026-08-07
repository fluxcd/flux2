# RFC-NNNN Authenticate in-cluster HTTP traffic across Flux

**Status:** provisional

**Creation date:** 2026-08-07

**Last update:** 2026-08-07

## Summary

This RFC defines authentication and TLS for the notification-controller event server
and the artifact servers of source-controller and source-watcher. Clients use
Kubernetes bound service account tokens, which the servers verify with the TokenReview
API. By default, the Flux CLI creates the certificate authority during bootstrap
and rotates it on a subsequent run after half of its validity period has elapsed.

The HTTP client, server middleware and certificate helpers will live in `fluxcd/pkg`.
Components in the Flux namespace, including the Flux Operator and 3rd-party source
controllers built on the `ExternalArtifact` API, can use the same helpers.

## Motivation

Flux controllers communicate over plain HTTP without authentication. All controllers
post events to notification-controller. The kustomize-controller and helm-controller
download artifacts from source-controller and source-watcher. A workload that can
reach these endpoints can spoof events or download artifacts containing the desired
state.

The NetworkPolicy shipped with the Flux distribution restricts network access to these
endpoints, but not all CNI implementations enforce NetworkPolicy. On clusters without
enforcement, the endpoints are reachable from every pod.

### Goals

Use one authentication mechanism for HTTP traffic between Flux components and expose
its implementation from `fluxcd/pkg`.

Provide a CLI-managed certificate authority by default, without requiring an external
PKI. Allow operators to supply certificates from another PKI.

Do not send tokens over plain HTTP or TLS connections that fail certificate
verification.

Roll out without breaking existing installations, including clusters running mixed
component versions during upgrades.

### Non-Goals

Authenticating human users. A `kubectl port-forward` connection does not bypass
authentication when enforcement is enabled.

Fine-grained authorization within the Flux namespace. The namespace is the trust
boundary: a valid token for any service account in it grants access to every endpoint
protected by this mechanism. Operators must restrict who can create workloads, use
service accounts or request service account tokens in that namespace. Components with
human-facing consumers, such as the Flux Operator Web UI, remain responsible for RBAC
and SSO.

Securing the notification-controller webhook receiver. It serves external traffic and
has its own per-receiver verification.

Authenticating the metrics and health probe endpoints.

Mutual TLS. Tokens identify clients. TLS identifies servers and encrypts traffic.

## Proposal

### Token projection

Every Flux Deployment mounts a projected service account token with a Flux-specific
audience:

```yaml
volumes:
  - name: flux-token
    projected:
      sources:
        - serviceAccountToken:
            path: token
            audience: toolkit.fluxcd.io
            expirationSeconds: 3600
```

This projection instructs the kubelet to request a token for the Pod's service account
with `toolkit.fluxcd.io` as its audience. The API server issues the token bound to the
Pod, and the kubelet mounts and rotates it.

A shared HTTP transport in `fluxcd/pkg` reads the token file for each request and sets
the `Authorization: Bearer` header. It is used by the event recorder in every
controller and by the artifact clients in kustomize-controller and helm-controller.

A single audience is used for all Flux endpoints, so each pod projects exactly one
token regardless of how many Flux services it talks to.

### Token verification

A shared HTTP middleware in `fluxcd/pkg` authenticates requests to the event and
artifact servers. It submits bearer tokens to the TokenReview API with
`toolkit.fluxcd.io` in `spec.audiences` and caches the results. In `enforced` mode,
missing or invalid credentials return `401 Unauthorized`, and an authenticated service
account from another namespace returns `403 Forbidden`.

The authenticated subject must be a service account in the server's namespace.
Namespace membership is the only authorization check. First-party controllers, 3rd
party components and other workloads using service accounts in that namespace receive
the same access. Flux controllers, including shards, run in `flux-system` namespace, so the
initial design has no service account allow-list and no cross-namespace support.

### Transport security

During `flux install` and `flux bootstrap`, the CLI generates a CA key pair in the
`flux-ca` Secret and publishes its certificate in the `flux-ca-bundle` ConfigMap. Both
objects live in the Flux namespace, and the Secret is not written to Git. On a later
run, the CLI rotates a CA that is past half of its validity. It first adds the new CA
certificate to the trust bundle, then replaces the CA key pair in the Secret. It
records the rotation time and removes the previous CA certificate on a subsequent run
after the seven-day serving-certificate lifetime has elapsed. 
Other installers must create the same objects or configure externally managed certificates.

The `--tls-ca-secret` flag names the Secret watched by each server. Each replica signs
an in-memory serving certificate containing its Service DNS names. When the Secret
changes, the server issues a new certificate without restarting.

The `--tls-ca-bundle` flag names the ConfigMap watched by each client. Clients verify
serving certificates against the bundle and attach a token only to requests sent over
verified TLS.

cert-manager and trust-manager can manage the `flux-ca` Secret and
`flux-ca-bundle` ConfigMap. Before reconciling either object, the CLI checks
`metadata.managedFields` and leaves data owned by another field manager unchanged.

### Enforcement

Clients attach tokens to requests made through the verified TLS transport. Servers
control authentication with a per-controller flag:

- `disabled`: authentication is not evaluated.
- `audit`: tokens are verified, requests that would be rejected are logged and
  counted, but all requests are served.
- `enforced`: requests without a valid token and requests over plain HTTP are
  rejected.

During migration, servers multiplex TLS and plain HTTP on the existing port. Updated
clients attempt verified TLS first and fall back to plain HTTP without a token. This
allows release N clients and servers to interoperate with older components.

### User Stories

#### Event Server Authentication

An administrator on a cluster without NetworkPolicy enforcement can require
notification-controller to accept events only from workloads in the Flux namespace.

#### Artifact Server Authentication

An administrator on a cluster without NetworkPolicy enforcement can restrict artifact
downloads to workloads in the Flux namespace.

#### 3rd Party Components

A component in the Flux namespace, such as the Flux Operator or an
`ExternalArtifact` source controller, can use the same client and server packages.

### Alternatives

**Mutual TLS.** Client certificates could identify workloads, but Flux would need to
issue and rotate a certificate for every client and map certificate identities to
workloads. TokenReview delegates client identity validation to Kubernetes.

**TLS from an external PKI.** The Flux distribution cannot require cert-manager or a
service mesh. An external PKI can manage the CA and trust bundle. A PKI that does not
expose a CA key can instead issue a serving certificate for each server.

**Static shared secret.** A token stored in a Secret mounted by all components would
avoid TokenReview calls, but it has no intrinsic expiry or workload binding and must be
rotated separately.

**Per-service audiences.** A separate audience for each server would prevent a token
accepted by one server from being replayed against another. This does not add
authorization under the namespace-wide trust model, but it would require a separate
token projection and client configuration for each audience. Using one audience does
not prevent a later split.

**Local token validation.** Servers could validate signatures against the cluster OIDC
issuer keys, but they would also need to handle discovery, key rotation, clock skew and
the lifetime of the bound Pod. TokenReview keeps those checks in Kubernetes.

**Kubernetes pod certificates.** The `podCertificate` projected volume from KEP-4317
provides kubelet-managed certificate requests and rotation, but Kubernetes has no
in-tree signer for pod-to-pod TLS. KEP-4317 targets stable status in Kubernetes v1.37.
It is not a replacement for the CLI-managed CA until the feature is stable, an
appropriate signer is available, and Flux no longer supports earlier Kubernetes
releases.

## Design Details

The client transport, server middleware and certificate helpers will be implemented in
`github.com/fluxcd/pkg`. All controllers will use the event client.
Kustomize-controller and helm-controller will use the artifact client.
Notification-controller, source-controller and source-watcher will use the server
middleware. The 3rd party source-controller SDK will expose the same helpers.

The certificates will use ECDSA P-256. The CA will be issued for one year and rotated
by the CLI on any run past half of its validity. The serving certificates will be
valid for seven days and reissued in memory at two thirds of their lifetime. These are
internal defaults and are not exposed as flags.

The TokenReview cache will be bounded and keyed by the SHA-256 digest of each token; it
will not store tokens. Positive results are cached for 30 seconds and negative results
for 5 seconds. Malformed authorization headers are rejected before TokenReview, and
the middleware limits concurrent reviews. A cached token may remain accepted for up to
30 seconds after revocation.

If TokenReview fails because the API server is unavailable or throttles the request,
an `enforced` server returns `503 Service Unavailable` on a cache miss. Cached results
remain valid until their TTL expires. In `audit` mode, the server logs and counts the
failure but serves the request.

A dedicated ClusterRole grants `create` on
`authentication.k8s.io/tokenreviews`. It is bound only to the service accounts for
notification-controller, source-controller and source-watcher. Existing RBAC already
allows these controllers to read Secrets and ConfigMaps.

The Secret referenced by `--tls-ca-secret` is of type `kubernetes.io/tls` and may
contain either a CA key pair or a serving certificate. The server distinguishes them
using the certificate's CA basic constraint. A serving certificate must contain the
server's Service DNS names and is served directly; its issuer certificate must be
placed in the client trust bundle. External certificate managers should use a separate
Secret for each server.

Servers export counters for rejected requests and for requests that fail authentication
in `audit` mode, plus a gauge for certificate expiry. `flux check` warns when a
CLI-managed CA is past 80% of its validity.

### Rollout

The rollout spans two minor releases:

1. Release N: the CLI creates the CA and trust bundle, manifests project the token,
   servers accept TLS and plain HTTP, clients prefer verified TLS but can fall back to
   plain HTTP without a token, and servers default to `audit`.
2. Release N+1: servers default to `enforced`. Before upgrading the servers, operators
   must upgrade all clients to release N or later. The flag remains available for
   external event producers that have not adopted authentication.

### Drawbacks

TokenReview adds API server requests for tokens not present in the cache. A stream of
unique invalid tokens can consume the review concurrency limit. In `enforced` mode,
cache misses fail when the API server is unavailable.

If neither the CLI nor an external certificate manager rotates `flux-ca`, connections
fail when the CA expires after one year. The expiry metric and `flux check` warn before
expiry.

A token for any service account in the Flux namespace grants access to every endpoint
protected by this mechanism. The namespace must be reserved for trusted workloads.

## Implementation History

<!-- TBD -->
