# RFC-0006 Alternative Suspend Control

**Status:** provisional

**Creation date:** 2023-09-20

**Last update:** 2023-10-10


## Summary

This RFC proposes an alternative method to indicate the suspended state of
suspendable resources to flux controllers through object metadata. It presents
an annotation key that can be used to suspend a resource from reconciliation as
an alternative to the `.spec.suspend` field.  It does not address the
deprecation of the the `.spec.suspend` field from the resource apis.  This
annotation can optionally act as a vehicle for communicating contextual
information about the suspended resource to users.


## Motivation

The current implementation of suspending a resource from reconciliation uses
the `.spec.suspend` field.  A change to this field results in a generation
number increase which can be confusing when diffing.

Teams may wish to communicate information about the suspended resource, such as
the reason for the suspension, in the object itself.

### Goals

The flux reconciliation loop will support recognizing a resource's suspend
status based on either the `.spec.suspend` field or a specific metadata
annotation key.  The flux cli will similarly recognize this state with `get`
commands and but will only manipulate the annotation key under `suspend` and
`resume` commands.  The flux cli will support optionally setting the suspend
metadata annotation value with a user supplied string for a contextual message.

### Non-Goals

The deprecation plan for the `.spec.suspend` field is out of scope for this
RFC.


## Proposal

Register a flux resource metadata key `reconcile.fluxcd.io/suspended` with a
suspend semantic to be interpreted by controllers and manipulated by the cli.
The presence of the annotation key is an alternative to the `.spec.suspend:
true` api field setting when considering if a resource is suspended or not.
The annotation key is set by a `flux suspend` command and removed by a `flux
resume` command.  The annotation key value is open for communicating a message
or reason for the object's suspension.  The value can be set using a
`--message` flag to the `suspend` command.

### User Stories

#### Suspend/Resume without Generation Roll

Currently when a resource is set to suspended or resumed the `.spec.suspend`
field is mutated which results in a roll of the generation number in the
`.metadata.generation` and fields `.status.observedGeneration` number.  The
community believes that the generation change for this reason is not in
alignment with gitops principles.

The flux controllers should recognize that a resource is suspended or
unsuspended from the presence of a special metadata key -- this key can be
added, removed or changed without patching the object and rolling the
generation number.

#### Seeing Suspend State

Users should be able to see the effective suspend state of the resource with a
`flux get` command.  The display should mirror what the controllers interpret
the suspend state to be.  This story is included to capture current
functionality that should be preserved.

#### Suspend with a Reason

Often there is a purpose behind suspending a resource with the flux cli,
whether it be during incident response, source manifest cutovers, or various
other scenarios. The `flux diff` command provides a great UX for determining
what will change if a suspended resource is resumed, but it doesn't help
explain _why_ something is paused or when it would be ok to resume
reconciliation. On distributed teams this can become a point of friction as it
needs to be communicated among group stakeholders.

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
would be made avaiable for controllers the cli to recognize and manipulate the
suspend object metadata.

``` # github.com/fluxcd/pkg/apis/meta


// SuspendAnnotation is the annotation used to indicate an object is suspended
and optionally to store metadata about why a resource // was set to suspended.
const SuspendedAnnotation string = "reconcile.fluxcd.io/suspended"

// SuspendAnnotationValue returns a value for the suspended annotation, which
can be used to detect // changes; and, a boolean indicating whether the
annotation was set. func SuspendedAnnotationValue(annotations
map[string]string) (string, bool) { suspend, ok :=
annotations[SuspendedAnnotation] return suspend, ok }

// SetSuspendedAnnotation sets a suspend key with a supplied string value in an
object's metadata annotations func SetSuspendedAnnotation(annotations
*map[string]string, token string) { if annotations == nil { annotations =
&map[string]string{} } if token != "" { (*annotations)[SuspendedAnnotation] =
token } else { (*annotations)[SuspendedAnnotation] = "true" } }

// UnsetSuspendedAnnotation removes a suspend key from an object's metadata
annotations func UnsetSuspendedAnnotation(annotations *map[string]string) {
delete(*annotations, SuspendedAnnotation) } ```

An event predicate would skip attempting to reconcile a resource with the
suspend annotation set.

``` # github.com/fluxcd/pkg/apis/predicates

# suspended.go

// Update implements the default UpdateEvent filter for filtering by
meta.SuspendedAnnotation existence func (SuspendedPredicate) Update(e
event.UpdateEvent) bool { if e.ObjectOld == nil || e.ObjectNew == nil { return
false }

    if _, ok := metav1.SuspendedAnnotationValue(e.ObjectNew.GetAnnotations());
    !ok { return true } return false } ```

### Controllers

Flux controllers would skip reconciling a resource based on an `OR` of (1) the
api `.spec.suspend` and (2) the existence of the suspend metadata annotation
key.

``` # github.com/fluxcd/kustomize-controller/api/v1 # kustomization_types.go

// IsSuspended returns the effective suspend state of the object. func (in
Kustomization) IsSuspended() bool { if in.Spec.Suspend { return true } if _, ok
:= meta.SuspendedAnnotationValue(in.Annotations); ok { return true } return
false }

# github.com/fluxcd/kustomize-controller/controller #
kustomization_controller.go

// Skip reconciliation if the object is suspended. // if obj.Spec.Suspend {  //
no longer using `.spec.suspend` directly if obj.Status.IsSuspended {
log.Info("Reconciliation is suspended for this object") return ctrl.Result{},
nil } ```

### cli

The flux cli would recognize the suspend state from the suspend object status.

``` # github.com/fluxcd/flux2/main

# get_source_git.go

func (a *gitRepositoryListAdapter) summariseItem(i int, includeNamespace bool,
includeKind bool) []string { ... return append(nameColumns(&item,
includeNamespace, includeKind), //    revision,
strings.Title(strconv.FormatBool(item.Spec.Suspend)), status, msg) revision,
strings.Title(strconv.FormatBool(item.IsSuspended)), status, msg) } ```

The flux cli would manipulate the suspend metadata but forgo toggling the
`.spec.suspend` setting.  An optional `--message|-m` flag would support setting
the suspended annotation value to a user-specified string.

``` # github.com/fluxcd/flux2/main

# suspend.go

type SuspendFlags struct { ... message string }

func init() { ... suspendCmd.PersistentFlags().StringVarP(&suspendArgs.message,
"message", "m", "", "set a message about why the resource is suspended, the
message will show up under the reconcile.fluxcd.io/suspended annotation") ... }

type suspendable interface { ... // setSuspended() setSuspended(message string)
}

type suspendCommand struct { ... // obj.setSuspended()
obj.setSuspended(suspendArgs.message) }

# suspend_helmrelease.go

func (obj helmReleaseAdapter) isSuspended() bool { return
obj.HelmRelease.IsSuspended()                               // use the resource
api spec method }

func (obj helmReleaseAdapter) setSuspended(message string) {
meta.SetSuspendedAnnotation(&obj.HelmRelease.Annotations, message) // use the
common annotation setter }

# resume_helmrelease.go

func (obj helmReleaseAdapter) setUnsuspended() {
meta.UnsetSuspendedAnnotation(&obj.HelmRelease.Annotations)        // use the
common annotation unsetter } ```

### Metrics

Custom metrics can be [configured under
kube-state-metrics](https://fluxcd.io/flux/monitoring/custom-metrics/#adding-custom-metrics)
using the object metadata annotation path.

```
- name: "resource_info" help: "The current state of a GitOps Toolkit resource."
  each: type: Info info: labelsFromPath: name: [metadata, name] labelsFromPath:
  ... suspendedAnnotation: [ metadata, annotations,
  reconcile.fluxcd.io/suspended ] ```

## Implementation History

tbd
