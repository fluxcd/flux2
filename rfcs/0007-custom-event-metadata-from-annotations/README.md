# RFC-0007 Custom Event Metadata from Annotations

**Status:** provisional

<!--
Status represents the current state of the RFC.
Must be one of `provisional`, `implementable`, `implemented`, `deferred`, `rejected`, `withdrawn`, or `replaced`.
-->

**Creation date:** 2024-05-23

**Last update:** 2024-05-23

## Summary

Flux users often run into situations where they wish to send custom, static metadata fields defined
in Flux objects on the events dispatched by the respective controller to notification-controller.
This proposal offers a solution for supporting those use cases uniformly across all Flux controllers
by sending annotation keys that are prefixed with a well-defined API Group.

## Motivation

The need for sending static metadata in Flux objects events can benefit users in a few ways:

* When sending events about identical objects living in different clusters to the same notification channel,
users often wish to differentiate those objects by the cluster where they live in. Sending custom metadata
fields like `cluster`, `region` and `availabilityZone` would solve this problem.
* When implementing integrations with notification-controller, users often wish to embed pieces of information
in Flux objects that would help them identify entities or resources that are part of their integrations, e.g.
the ID of a Deployment in the GitHub API.

### Goals

Provide a method for Flux users to embed custom/static fields in their Flux objects
and have those fields sent to notification-controller as metadata on the event payload.

### Non-Goals

In this proposal we **do not** aim to provide a method for Flux users to send etcd-indexed custom metadata
fields from Flux objects in events to notification-controller, most specifically labels. By design an event
already contains enough identification information to locate the associated Flux object inside the cluster,
which covers the use case of labels. Flux does not wish to incentivize practices that are impactful to clusters
without a strong reason or benefit.

## Proposal

When sending events about Flux objects, we propose sending annotation keys prefixed with a well-defined
API Group in addition to those already sent by the controller to implement Flux-specific functionality.

### User Stories

* https://github.com/fluxcd/helm-controller/issues/680
* https://github.com/fluxcd/helm-controller/pull/682#issuecomment-2079547110

### Alternatives

An alternative for specifying custom metadata fields in Flux objects for sending on events
is defining `.spec` APIs for such, like `.spec.eventMetadata` available in the Alert API.
This alternative is not great because:

* Such APIs would be fairly redundant with the well-known Kubernetes annotations.
* Technically speaking, it is much easier to implement an alternative where the
field storing the custom metadata is the same and is already available across all the
Flux objects rather than introducing a new API.

In the specific case of the Alert API this field was introduced because the Alert API
is obviously a special one in the context of events and alerting. In particular, the
Alert objects do not generate events themselves, but rather serve as an aggregation
configuration for matching and propagating events from other Flux objects.

## Design Details

For the API Group used to extract the custom metadata fields from object annotations we have a
couple of alternatives:

* `notification.toolkit.fluxcd.io`. This alternative emphasizes the close relationship of the
feature with notification-controller and does not introduce a new Flux API group.
* `event.toolkit.fluxcd.io`. This alternative decouples the feature from notification-controller
and brings the concept closer to Kubernetes Events. In fact, our implementation builds on the
`EventRecorder` interface from the Go package `k8s.io/client-go/tools/record`, hijacking the
`AnnotatedEventf()` logic to send events not only to Kubernetes but also notification-controller.
This would, however, introduce a new Flux API Group.

Chosen alternative: ?

### How can this feature be enabled / disabled?

To enable the feature, use the well-defined API Group as a prefix in Flux object annotations,
for example:

* `?/cluster: prod`
* `?/region: us-west-2`
* `?/deploymentID: 1`

It's important to notice that not all Flux objects emit events, e.g. Alert and Provider objects.
For a list of the Flux objects that emit events, see the kinds allowed on the
`.spec.eventSources[].kind` field of the Alert API.

To disable the feature, do not use the well-defined API Group as a prefix in Flux object annotations.

## Implementation History

<!--
Major milestones in the lifecycle of the RFC such as:
- The first Flux release where an initial version of the RFC was available.
- The version of Flux where the RFC graduated to general availability.
- The version of Flux where the RFC was retired or superseded.
-->
