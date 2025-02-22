# RFC-0008 Custom Event Metadata from Annotations

**Status:** implemented

<!--
Status represents the current state of the RFC.
Must be one of `provisional`, `implementable`, `implemented`, `deferred`, `rejected`, `withdrawn`, or `replaced`.
-->

**Creation date:** 2024-05-23

**Last update:** 2025-02-22

## Summary

Flux users often run into situations where they wish to send custom, static metadata fields defined
in Flux objects on the events dispatched by the respective Flux controller to Kubernetes and
notification-controller. This proposal offers a solution for supporting those use cases uniformly
across all Flux controllers by sending the annotation keys in Flux objects that are prefixed with
the API Group `event.toolkit.fluxcd.io` followed by a slash, i.e. `event.toolkit.fluxcd.io/`.

After the overall outcome of this RFC is implemented, Flux would have customization options for
notification metadata strong enough to eliminate the need for the `.spec.summary` field from the
Alert API. During the discussion of this RFC the Flux team has decided to deprecate `.spec.summary`
in favor of `.spec.eventMetadata.summary`, and to remove this field in the Flux release of Alert
API v1 GA when it takes place.

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

When sending events about Flux objects, we propose sending annotation keys prefixed with the well-defined
API Group `event.toolkit.fluxcd.io` followed by a slash, i.e. prefixed with `event.toolkit.fluxcd.io/`, in
addition to all the metadata that is already sent in the event.

### User Stories

#### Story 1

> As a user, I want to embed Flux into my GitHub Workflow in a way that it only succeeds if
> the deployment made by Flux is successful.

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
    event.toolkit.fluxcd.io/image: ghcr.io/stefanprodan/podinfo:latest # {"$imagepolicy": "flux-system:podinfo"}
spec:
  chart:
    spec:
      chart: podinfo
      sourceRef:
        kind: HelmRepository
        name: podinfo
  values:
    image:
      tag: latest  # {"$imagepolicy": "flux-system:podinfo:tag"}
```

In this example image-automation-controller would update the image and tag near the markers. If, for example, it
updates the image to `ghcr.io/stefanprodan/podinfo:6.5.0`, then it would cause notification-controller to start
propagating events like the one below (most fields omitted for brevity):

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
    "image": "ghcr.io/stefanprodan/podinfo:6.5.0"
  }
}
```

### Alternatives

#### Alternative 1

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

#### Alternative 2

Instead of introducing a new API Group, i.e. `event.toolkit.fluxcd.io`, we could use the API
Group `notification.toolkit.fluxcd.io` for the same purpose. This alternative is not great
because it emphasizes an exclusive relationship with notification-controller, which is not the
case. The events here are also Kubernetes Events, and an API Group that is more general and
closer to Kubernetes Events is more appropriate.

## Design Details

All the Flux controllers use our implementation of the `EventRecorder` interface from the Go
package `k8s.io/client-go/tools/record`: [`(*github.com/fluxcd/pkg/runtime/event.Recorder).AnnotatedEventf()`](https://github.com/fluxcd/pkg/blob/6f2619522699f1a78e8c7b41583ad9f7b7c9544e/runtime/events/recorder.go#L119).
This implementation sends the events to notification-controller and also calls the same method
from an injected `EventRecorder`, which sends the events to Kubernetes. To support the use cases
discussed here we would modify this implementation to look for annotations prefixed with
`event.toolkit.fluxcd.io/` in the Flux objects and send them alongside the other metadata of
the event. Here we are talking specifically about the object annotations retrieved from the
Flux object itself, i.e. the first argument of the `AnnotatedEventf()` method: `object runtime.Object`.
This implementation would not change the interface `EventRecorder` used by the controllers,
so all we need to do is bump the Go package `github.com/fluxcd/pkg/runtime` across all controllers.

On the notification-controller side we would start accepting metadata keys starting with this
prefix and remove it before sending the metadata key-value pair to the notification providers.
This is an important aspect of the implementation because notification-controller only
accepts metadata keys that are prefixed with the Group of the respective API the involved
Flux object belongs to, so we need to add an exception for the new prefix.

The API Group `event.toolkit.fluxcd.io` would be introduced as a constant in the package
`github.com/fluxcd/pkg/apis/event` with the name `Group`. This constant would be used in the package
`github.com/fluxcd/pkg/runtime/event` and notification-controller for the implementation described above.

### Precedence Order

After this change there would be four sources of metadata being sent on notifications.
They are listed below with the proposed order of precedence, from lowest to highest:

1. User-defined metadata on Flux objects, set with the `event.toolkit.fluxcd.io/` prefix in the keys of the object's `.metadata.annotations`.
2. User-defined metadata on the Alert object, set with `.spec.eventMetadata`.
3. User-defined summary on the Alert object, set with `.spec.summary`.
4. Controller-defined metadata, set with the `<controller group>.toolkit.fluxcd.io/` prefix in the metadata keys of the event payload.

Upon any key conflicts when combining all these metadata, notification-controller would
resolve them according to the precedence order specified above, print an `info` log and emit a
Kubernetes Event containing all the key conflicts to warn the user and prompt them to change
their configuration to remove those conflicts.

#### Reasoning

Controller-defined metadata has the highest precedence because it integrates with external systems,
e.g. commit SHAs, digests, chart versions, etc.

Alert-level metadata, i.e. `.spec.summary` and `.spec.eventMetadata`, are usually cluster-level,
e.g. the cluster name, region, environment, etc. We don't want tenants overriding cluster-level
metadata.

User-defined metadata on Flux objects, whose use cases are described in the [User Stories](#user-stories)
section, would usually be defined by cluster tenants. Hence it should not override cluster-level metadata.

The `.spec.summary` field of the Alert API was the first introduced for supporting user-defined
metadata. The value of this field is appended to the notification metadata with the key `summary`.
Later, `.spec.eventMetadata` was introduced to enhance this capability by supporting structured
user-defined metadata, i.e. the ability to specify keys other than `summary`. The latter is a
generalization of the former, and, given also the overall strong customization options offered
by Flux for notification metadata after the changes introduced in this RFC, the Flux team has
decided to deprecate `.spec.summary` in favor of `.spec.eventMetadata.summary`, and to remove
it in the Flux release of Alert API v1 GA when it takes place. Until then, we decided to keep
the current priority of the field, which is higher than `.spec.eventMetadata.summary`.

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

* RFC implemented and generally available in Flux v2.5.

<!--
Major milestones in the lifecycle of the RFC such as:
- The first Flux release where an initial version of the RFC was available.
- The version of Flux where the RFC graduated to general availability.
- The version of Flux where the RFC was retired or superseded.
-->
