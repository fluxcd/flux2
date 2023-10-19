# RFC-0006 Alternative Suspend Control

**Status:** provisional

**Creation date:** 2023-09-20

**Last update:** 2023-10-18


## Summary

This RFC proposes an alternative method to indicate the suspended state of
suspendable resources to flux controllers through object metadata. It presents
an annotation key that can be used to suspend a resource from reconciliation as
an alternative to the `.spec.suspend` field.  It does not address the
deprecation of this field from the resource apis.  This annotation can
optionally act as a vehicle for communicating contextual information about the
suspended resource to users.


## Motivation

The current implementation of suspending a resource from reconciliation uses
the `.spec.suspend` field.  A change to this field results in a generation
number increase which can be confusing when diffing.

Teams may wish to communicate information about the suspended resource, such as
the reason for the suspension, in the object itself.

### Goals

The flux reconciliation loop will support recognizing a resource's suspend
status from either the api field or the designated metadata annotation key.
The flux cli will similarly recognize this state with `get` commands and but
will alter only the metadata under the `suspend` command.  The `resume` command
will still alter the api field but additionally the metadata.  The
flux cli will support optionally setting the suspend metadata annotation value
with a user supplied string for a contextual message.

### Non-Goals

The deprecation plan for the `.spec.suspend` field is out of scope for this
RFC.


## Proposal

Register a flux resource metadata key `reconcile.fluxcd.io/suspended` with a
suspend semantic to be interpreted by controllers and manipulated by the cli.
The presence of the annotation key is an alternative to the `.spec.suspend` api
field setting when considering if a resource is suspended or not. The
annotation key is set by a `flux suspend` command and removed by a `flux
resume` command.  The annotation key value is open for communicating a message
or reason for the object's suspension.  The value can be set using a
`--message` flag to the `suspend` command.

### User Stories

#### Suspend/Resume without Generation Roll

Currently when a resource is set to suspended or resumed the `.spec.suspend`
field is mutated which increments the `.metadata.generation` field and after
successful reconciliation the `.status.observedGeneration` number.  The
community believes that the generation change for this reason is not in
alignment with gitops principles.  In more detail, upon suspension the
generation increments but the observed generation lags since reconciliation is
not completed successfully.

The flux controllers should recognize that a resource is suspended or
unsuspended from the presence of a special metadata key -- this key can be
added, removed or changed without patching the object in such a way that the
generation number increments.

#### Seeing Suspend State

Users should be able to see the effective suspend state of the resource with a
`flux get` command.  The display should mirror what the controllers interpret
the suspend state to be.  This story is included to capture current
functionality that should be preserved.

#### Suspend with a Reason

Often there is a purpose behind suspending a resource with the flux cli,
whether it be during incident response, source manifest cutovers, or various
other scenarios. The `flux diff` command provides an illustrative UX for
determining what will change if a suspended resource is resumed, but neither it
nor `flux get` help explain _why_ something is paused or when it would be ok to
resume reconciliation. On distributed teams this can become a point of friction
as it needs to be communicated among group stakeholders.

Flux users should have a way to succinctly signal to other users why a resource
is suspended on the resource itself.

### Alternatives

#### More `.spec`

The existing `.spec.suspend` could be expanded with fields for the above
semantics.  This would drive more generation number changes and would require a
change to the apis.


## Design Details

Implementing this RFC would involve the controllers and the cli.

This feature would create an alternate path to suspending an object and would
not violate the current apis.

### Common

The `reconcile.fluxcd.io/suspended` annotation key string and a getter function
would be made avaiable for controllers and the cli to recognize and manipulate the
suspend object metadata.

### Controllers

Flux controllers would skip reconciling a resource based on an `OR` of (1) the
api `.spec.suspend` and (2) the existence of the suspend metadata annotation
key.  This would be implemented in the controller predicates to completely skip
any reconciliation cycle of suspended objects.

### cli

The `get` command would recognize the suspend state from the union of the
`.spec.suspend` and the presence of the suspended annotation.

The `suspend` command would add the suspend annotation but forgo modifying the
`.spec.suspend` field.

The `resume` command would remove the suspend annotation and modify the
`.spec.suspend` field to `false`.

The suspend annotation would by default be set to a generic value.  An optional
cli flag (eg `--message`) would support setting the suspended annotation value
to a user-specified string.

## Implementation History

tbd
