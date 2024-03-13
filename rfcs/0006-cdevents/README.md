# RFC-0006 Flux CDEvents Receiver

<!--
The title must be short and descriptive.
-->

**Status:** provisional

<!--
Status represents the current state of the RFC.
Must be one of `provisional`, `implementable`, `implemented`, `deferred`, `rejected`, `withdrawn`, or `replaced`.
-->

**Creation date:** 2023-12-08

**Last update:** 2024-02-09

## Summary

This RFC proposes to add a `receiver` type to Flux that can handle CDEvents.
The `receiver` will receive `CDEvent` events, sent to the receiver's webhook URL. It will verify the received events, check their `event type`, and eventually trigger reconciliation of the configured resources.
<!--
One paragraph explanation of the proposed feature or enhancement.
-->

## Motivation

CDEvents enables interoperability between supported tools in a workflow, and flux is a very popular continuous delivery tool, and consequently we have received many questions about implementing CDEvents into the tool. 
<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this RFC. Describe why the change is important and the benefits to users.
-->

### Goals

Integrate CDEvents into Flux with a CDEvents Receiver.
<!--
List the specific goals of this RFC. What is it trying to achieve? How will we
know that this has succeeded?
-->

### Non-Goals

A CDEvent provider will be handled as a separate RFC in the future.

<!--
What is out of scope for this RFC? Listing non-goals helps to focus discussion
and make progress.
-->

## Proposal

Add CDEvents to the list of available receivers in Flux Notification controller. Similar to other receivers such as the github, the user will be able to use `spec.events` in order to specify which event types the receiver will allow. The receiver will also verify using the [CDEvents Go SDK](https://github.com/cdevents/sdk-go) that the payload sent to the webhook URL is a valid CDEvent.

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation.

If the RFC goal is to document best practices,
then this section can be replaced with the actual documentation.
-->

### User Stories

<!--
Optional if existing discussions and/or issues are linked in the motivation section.
-->

Users of multiple different CI/CD tools such as Tekton and Flux could use CDEvents as a potential way to enable interoperability. For example, a user may want a Flux resource to reconcile as part of a Tekton `pipeline`.
The Tekton `pipeline` will fire off a CDEvent to the CloudEvents Broker. A subscription that the user will have set up externally, e.g. with the [knative broker](https://knative.dev/docs/eventing/brokers/) will then send a relevant CDEvent to the CDEvent Receiver within Flux. 

![usecase](cdevents-flux-tekton.png)

### Alternatives

Certain use cases for CDEvents could be done alternatively using available receivers such as the generic webhook.

<!--
List plausible alternatives to the proposal and explain why the proposal is superior.

This is a good place to incorporate suggestions made during discussion of the RFC.
-->

## Design Details

Adding a receiver for CDEvents that works much like the other event-based receivers already implemented. The user will be able to write a yaml file for the receiver and deploy it to their cluster. The receiver takes the payload sent to the webhook URL by an external events broker, checks the headers for the event type, and filters out events based on the user-defined list of events in spec.Events. If left empty, it will act on all valid CDEvents. It then validates the payload body using the [CDEvents Go SDK](https://github.com/cdevents/sdk-go). Valid events will then trigger reconciliation of the resources. The events broker is not a part of this design and is left to the user to set up however they wish.

Example Receiver YAML:

```yaml
apiVersion: notification.toolkit.fluxcd.io/v1
kind: Receiver
metadata:
  name: cdevents-receiver
  namespace: flux-system
spec:
  type: cdevents
  events:
    - "dev.cdevents.change.merged"
  secretRef:
    name: receiver-token
  resources:
    - kind: GitRespository 
      apiVersion: source.toolkit.fluxcd.io/v1
      name: webapp
      namespace: flux-system
```

![User Flowchart](Flux-CDEvents-RFC.png)

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs and code snippets.

The design details should address at least the following questions:
- How can this feature be enabled / disabled?
- Does enabling the feature change any default behavior?
- Can the feature be disabled once it has been enabled?
- How can an operator determine if the feature is in use?
- Are there any drawbacks when enabling this feature?
-->

![Adapter](CDEvents-Flux-RFC-Adapter.png)


## Implementation History

<!--
Major milestones in the lifecycle of the RFC such as:
- The first Flux release where an initial version of the RFC was available.
- The version of Flux where the RFC graduated to general availability.
- The version of Flux where the RFC was retired or superseded.
-->
