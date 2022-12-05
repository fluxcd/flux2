# RFC-NNNN Built-in TLS support for Source and Notification Controllers

**Status:** provisional

**Creation date:** 2022-12-05

**Last update:** 2022-12-05

## Summary

Create built-in TLS support for both Source and Notification controllers,
so that in-cluster cross-component communications are always encrypted.

## Motivation

Due to internal company policies, Data Protection laws and regulations,
Flux users may be required to enforce encryption of data whilst in-transit.
As of `v0.37.0`, there is no built-in way to provide TLS for Flux' endpoints.

The existing endpoints being:

1. Source Controller: artifacts storage, used by Kustomize and HelmRelease
controllers.
2. Notification Controller: events receivers, used by all other Flux components
to submit events.
3. Notification Controller: webhooks for external services to contact Flux.

The webhooks endpoing would be typically used for external use only, and in
that use case the recommended approach is to enable TLS at ingress.
However, this proposal covers the first two endpoints to allow always encrypted
in-cluster communications.

### Goals

- Add TLS support for Notification and Source controllers.
- Establish a migration process when enabling TLS on existing clusters.
- Define best practices on enabling the feature.

### Non-Goals

- Change how CA trust is established across Flux containers.
- Define TLS rotation mechanisms.

## Proposal

Controllers hosting web servers will introduce a new flag to configure the
[TLS secret]: `--tls-secret`. The Secret type must be `kubernetes.io/tls`.

Invalid secret types, or content, will cause the controller to fail to start.

### Source Controller

When TLS is enabled, the File Storage endpoint will serve both HTTP and HTTPS.
Existing source objects previously reconciled won't be patched to a new
`.status.artifact.url`, but all new reconciliations will cause that field to
start with `https://`.

The HTTP endpoint will only serve a permanent redirect (301) to the HTTPS endpoint,
enabling a smooth transition when users enable this feature in existing clusters.

The HTTPS endpoint will be served on port `9443` by default, which will be
configurable via new flag `--storage-https-addr` or environment variable
`STORAGE_HTTPS_ADDR`. The existing `source-controller` service will bind port
`443` to `9443`.

### Notification Controller

When TLS is enabled, the HTTP endpoints will be disabled.

#### Events endpoint

The events HTTPS endpoint will be served on port `9443` by default, which will be
configurable via new flag `--events-https-addr`. The existing `notification-controller`
service will bind port `443` to `9443`.

All flux components will need to be started with the (existing) flag below, for the
TLS only endpoint to be used:
`--events-addr=https://notification-controller.flux-system.svc.cluster.local./`

### User Stories

<!--
Optional if existing discussions and/or issues are linked in the motivation section.
-->

### Alternatives

#### Delegate in-trasit data encryption to CNI or Service Mesh

Instead of built-in support, it was considered the option of delegating this to
external components that support the enablement of end-to-end encryption via TLS
or mTLS. One of the issues with this approach is that the data would still come
out of Flux unencrypted, and therefore privileged co-located components in the
same worker node could be privy to the exchanged plain-text.

Another point considered for not pursuing this alternative was that, being one of
the first workloads inside a cluster, Flux should be self-reliant in assuring the
confidentiality and integrity of its communications.

#### Auto-detection of TLS events endpoint

Instead of having to change the `--event-addr` on each Flux component, an auto
detection mechanism could be implemented to prefer the TLS endpoint whenever the
notification controller's service has one working.

The challenge with such approach is that the security of cross component
communication would be non deterministic from a client perspective, and would depend
on the Notification controller having an active TLS endpoint at the time in which each
Flux component started. The current approach requires additional configuration,
but fails safe.

## Design Details

### TLS Versions and Ciphers Supported

For reduced complexity and increased security, only TLS 1.3 is supported.

### Detect TLS Certificate changes

Controllers will watch the TLS secret for changes, once detected it will trigger a
pod restart to refresh the certificates being used.

### Expand TLS support for the webhooks endpoint

The TLS support could be expanded to the Notification controller webhooks endpoint
to enable secure in-cluster communications.

## Implementation History

<!--
Major milestones in the lifecycle of the RFC such as:
- The first Flux release where an initial version of the RFC was available.
- The version of Flux where the RFC graduated to general availability.
- The version of Flux where the RFC was retired or superseded.
-->

[TLS secret]: https://kubernetes.io/docs/concepts/configuration/secret/#tls-secrets
