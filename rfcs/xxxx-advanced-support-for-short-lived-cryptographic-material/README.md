# RFC-XXXX Advanced Support for Short-Lived Cryptographic Material

**Status:** provisional

<!--
Status represents the current state of the RFC.
Must be one of `provisional`, `implementable`, `implemented`, `deferred`, `rejected`, `withdrawn`, or `replaced`.
-->

**Creation date:** 2026-09-07

**Last update:** 2026-09-07

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

- *Authentication*: Secure authentication with external systems, be it
  from a cloud provider or a vendor-neutral infrastructure components such
  as free open-source projects.
- *Private Communication*: Secure communication among the Flux controllers,
  and between Flux controllers and external systems.
- *Centralized Management*: Integration with central infrastructure that
  is responsible for providing short-lived cryptographic material for the
  rest of the stack of the Flux user.

The Secure Production Identity Framework For Everyone (SPIFFE) CNCF project
aims to provide a standardized framework for issuing and managing short-lived
cryptographic identities for widely used standards such as JWT and x509
certificates. The project is graduated and adoption grows steadily, from both
large technology players such as
[Uber](https://www.uber.com/us/en/blog/solving-the-agent-identity-crisis/)
and open-source projects, e.g. several service meshes and policy engines.
Also, Flux users have manifested interest in integrations with SPIFFE from the
Flux side, such as [#3368 (comment)](https://github.com/fluxcd/flux2/pull/3368#discussion_r1040899292)
and [#5679](https://github.com/fluxcd/flux2/discussions/5679) and several
offline discussions in conferences and meetups and lost Slack threads.

This RFC takes RFC-0010 to the next level in two dimensions:

- Flux will be able to support workload identity for more vendor-neutral
  infrastructure components, such as container registries like Harbor and
  Zot (both CNCF projects) that have implemented support for workload
  identity. RFC-0010 introduced workload identity for remote Kubernetes
  clusters and Flux 2.9 introduced workload identity for OpenBao (and Vault).
  By supporting workload identity for container registries Flux will be
  covering workload identity for all the vendor-neutral infrastructure
  components that are core to Flux: Kubernetes, container registries
  and key management systems for decryption of secrets / transit engine.
- Flux will support both an out-of-the-box workload identity backbone,
  which is Kubernetes itself through ServiceAccount tokens, and a more
  advanced solution that covers more security scenarios beyond workload
  identity, like private communication, and aims specifically at centralized
  management of short-lived cryptographic material. By using the JWT PKI
  provided by Kubernetes a Flux user can go a long way, but each cluster
  will have its own key pair and federating several clusters in a consistent
  way can be a challenge. Because SPIFFE's main goal, on the other hand, is
  to solve this specific problem, it has come up with established ways for
  different SPIFFE runtimes to federate and provide a consistent layer of
  trust across clusters and the entire infrastructure of an organization.

### Goals

The main goal of this RFC is defining the shape of the APIs that will support
the use cases described in this RFC in a way that they can be uniformly
implemented across all the Flux components (given enough time and demand).

### Non-Goals

It's not a goal of this RFC to make Flux become a SPIFFE runtime,
like SPIRE (the reference runtime implementation). The whole point of
SPIFFE is providing a simple interface through which applications can
acquire short-lived cryptographic material from the infrastructure,
allowing the ownership of the signing keys to remain with the
infrastructure rather than with the application. From this RFC's
perspective, Flux is the application, and whatever SPIFFE runtime
the user has deployed is the infrastructure. There are both free
(SPIRE) and commercial options available.

## Proposal

TODO

### API fields

TODO

### Controller Flags

TODO

### CLI Support

TODO

### User Stories

TODO

## Design Details

TODO

### Validation Rules for the API fields

TODO

### Caching

The existing token cache from RFC-0010 will be reused. Following
the same principle established there, any new fields that affect
the credential will be part of the key computed during a
reconciliation to look up the credential cache.

## Implementation History

TODO

<!--
Major milestones in the lifecycle of the RFC such as:
- The first Flux release where an initial version of the RFC was available.
- The version of Flux where the RFC graduated to general availability.
- The version of Flux where the RFC was retired or superseded.
-->
