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

This RFC comes as a response to the need for adding custom metadata to events about Flux objects
sent to notification providers. See specific user stories in the [User Stories](#user-stories) section.

### Goals

Provide a method for Flux users to embed custom/static metadata in their Flux objects
and have that metadata propagated to the notification providers.

### Non-Goals

In this proposal we **do not** aim to provide a method for Flux users to send etcd-indexed custom metadata
fields from Flux objects in events to notification-controller, most specifically labels. By design an event
already contains enough identification information to locate the associated Flux object inside the cluster,
which covers the use case of labels. Flux does not wish to incentivize practices that are impactful to clusters
without a strong reason or benefit.

## Proposal

When sending events about Flux objects, we propose sending annotation keys prefixed with a well-defined
API Group in addition to those already sent by the controller.

### User Stories

#### Story 1

> As a user, I want to embed FluxCD into my GitHub Workflow in a way that it only succeeds if
> the deployment made by FluxCD is successful.

For example, embedding a Deployment ID from the GitHub API in a `HelmRelease` object like the one below:

```yaml
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: podinfo
  namespace: flux-system
  annotations:
    event.toolkit.fluxcd.io/deploymentID: e076e315-5a48-41c3-81c8-8d8bdee7d74d
spec:
  chart:
    spec:
      chart: podinfo
      version: 6.5.*
      sourceRef:
        kind: HelmRepository
        name: podinfo
```

Should cause notification-controller to propagate an event like the one below (most fields omitted for brevity):

```json
{
  "involvedObject": {
    "apiVersion": "helm.toolkit.fluxcd.io/v2",
    "kind": "HelmRelease",
    "name": "podinfo",
    "namespace": "flux-system",
    "uid": "7d0cdc51-ddcf-4743-b223-83ca5c699632"
  },
  "metadata": {
    "deploymentID": "e076e315-5a48-41c3-81c8-8d8bdee7d74d"
  }
}
```

#### Story 2

> As a user, I want to embed the new image tag in a `HelmRelease` object when the image is updated by an `ImageUpdateAutomation`
> and have that information propagated to the notification providers.

For example:

```yaml
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: podinfo
  namespace: flux-system
  annotations:
    event.toolkit.fluxcd.io/image: ghcr.io/stefanprodan/podinfo # {"$imagepolicy": "flux-system:podinfo:name"}
    event.toolkit.fluxcd.io/imageTag: 6.5.0 # {"$imagepolicy": "flux-system:podinfo:tag"}
spec:
  chart:
    spec:
      chart: podinfo
      version: 6.5.0 # {"$imagepolicy": "flux-system:podinfo:tag"}
      sourceRef:
        kind: HelmRepository
        name: podinfo
```

Should cause notification-controller to propagate an event like the one below (most fields omitted for brevity):

```json
{
  "involvedObject": {
    "apiVersion": "helm.toolkit.fluxcd.io/v2",
    "kind": "HelmRelease",
    "name": "podinfo",
    "namespace": "flux-system",
    "uid": "7d0cdc51-ddcf-4743-b223-83ca5c699632"
  },
  "metadata": {
    "image": "ghcr.io/stefanprodan/podinfo",
    "imageTag": "6.5.0"
  }
}
```

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

Chosen alternative: `event.toolkit.fluxcd.io`

### How can this feature be enabled / disabled?

To enable the feature, use the `event.toolkit.fluxcd.io/` prefix in Flux object annotations,
for example:

* `event.toolkit.fluxcd.io/image: ghcr.io/stefanprodan/podinfo`
* `event.toolkit.fluxcd.io/deploymentID: e076e315-5a48-41c3-81c8-8d8bdee7d74d`

It's important to notice that not all Flux objects emit events, e.g. Alert and Provider objects.
For a list of the Flux objects that emit events, see the kinds allowed on the
`.spec.eventSources[].kind` field of the Alert API.

To disable the feature, do not use `event.toolkit.fluxcd.io/` as a prefix in Flux object annotations.

## Implementation History

<!--
Major milestones in the lifecycle of the RFC such as:
- The first Flux release where an initial version of the RFC was available.
- The version of Flux where the RFC graduated to general availability.
- The version of Flux where the RFC was retired or superseded.
-->
