# RFC-0006 Alternative Suspend Control

**Status:** provisional

**Creation date:** 2023-09-20

**Last update:** 2023-09-20


## Summary

This RFC proposes an alternative method to indicate the suspended state of suspendable resources to flux controllers through object metadata. It presents an annotation key that can support suspending indefinitely or according to a schedule.  It does not address the deprecation of the the `.spec.suspend` field from the resource apis.


## Motivation

The current implementation of suspending a resource from reconciliation uses the `.spec.suspend` field.  A change to this field results in a generation number increase which can be confusing when diffing.

Teams may want to set windows during which no gitops reconciliations happen for a resource.  Such hold-offs are a natural fit for the existing suspend functionality which could be augmented to support date-time ranges.

### Goals

The flux reconciliation loop will support recognizing a resource's suspend status based on either the `.spec.suspend` field or a specific annotation metadata key.  The flux cli will similarly recognize this with get commands and will manipulate both the resource `.spec.suspend` field and the suspend status under suspend and resume commands.

### Non-Goals

The deprecation plan for the `.spec.suspend` field is out of scope for this RFC.


## Proposal

Register a resource object status for the suspended state. The flux object status `suspended` holds the current state of suspension from reconciliation for the resource.

Register a flux resource metadata key with a temporal suspend semantic with suspend status handling implemented in each of the controllers and in the flux cli.  The annotation `reconcile.fluxcd.io/suspendedDuring` defines when to suspend a resource through a period of time.  Supports a cron format including tags and ranges.  It's presence is used by flux controllers to evaluate and set the suspend status of the resource.  Is removed by a resume command.

### User Stories

#### Suspend/Resume without Generation Roll

Currently when a resource is set to suspended or resumed the `.spec.suspend` field is mutated which results in a roll of the generation number in the `.metadata.generation` and fields `.status.observedGeneration` number.  The community believes that the generation change for this reason is not in alignment with gitops principles.

The flux controllers should recognize that a resource is suspended or unsuspended from the presence of a special metadata key -- this key can be added, removed or changed without patching the object and rolling the generation number.

#### Suspend During Windows

A cluster manager wishes to set periodic windows during which deploys are held back (eg for stability, release control, maintenance, weekends, etc).  Gitops synchronization happens naturally outside of these times.

The cluster manager should be able to define these windows on each resource using a well-known format applied as a metadata key.

#### Seeing Suspend State

Users should be able to see the effective suspend state of the resource with a `flux get` command.  The display should mirror what the controllers interpret the suspend state to be.  This story is included to capture current functionality that should be preserved.

### Alternatives

#### The Gate

The [gate](https://github.com/fluxcd/flux2/pull/3158) proposes a new paired resource and controller to effect a paused reconciliation of selected resources (eg `GitRepository` or `HelmRelease`).

#### More `.spec`

The existing `.spec.suspend` could be expanded with fields for the above semantics.  This would drive more generation number changes and would require a change to the apis.


## Design Details

Implementing this RFC would involve resource object status, controllers, cli and metrics.

This feature would create an alternate path to suspending an object and would not violate the current apis.

### Resource Object Status

Object status could be augmented with a `suspended: true | false` indicating to hold the suspend status of the resource.

```
# github.com/fluxcd/pkg/apis/meta

// SuspendDuringAnnotation is the annotation used to set a suspend timedate expression
// the expression would conform to unix-like cron format to allow for indefinite (eg `@always`)
// or scheduled (eg `* 0-4 * * *`) suspend periods.
const SuspendDuringAnnotation string = "reconcile.fluxcd.io/suspendDuring"

type SuspendedStatus struct {
  // SuspendedStatus represents the state of reconciliation suspension
  Suspended bool `json:"suspended,omitempty"`
}

// GetSuspended returns the current suspended state value from the SuspendedStatus.
func (in SuspendedStatus) GetSuspended() string {
	return in.Suspended
}

// SetSuspended sets the suspended state value in the SuspendedStatus.
func (in *SuspendedStatus) SetSuspended(token bool) {
	in.Suspended = token
}
```

### Controllers

Flux controllers would set this status based on an `OR` of (1) the api `.spec.suspend` and (2) the evalution of the reconcile time against the suspend metadata annotation expression.  The controllers would use this resulting status when deciding to synchronize the resource.

Coupling the `.spec.suspend` and metadata annotations would be optional.  Having the controllers add a suspend metadata annotation, if it did not exist, based on `.spec.suspend: true` would tightly couple the two control points and would provide for a path to deprecating the `.spec.suspend` from the api.  The examples below do not show a coupled implementation.

```
# github.com/fluxcd/kustomize-controller/controller

# kustomization_controller.go

// use a cron expression in the suspendDuring annotation to determine if the resource should be set to suspended
// where and how to implement the notional `TimeInPeriod` function is open for suggestion
if obj.spec.Suspend || TimeInPeriod(reconcileStart, obj.Annotations[meta.SuspendedDuring]) {
    obj.Status.Suspended = true
} else {
    obj.Status.Suspended = false
}

// Skip reconciliation if the object is suspended.
// if obj.Spec.Suspend {  // no longer using `.spec.suspend` directly
if obj.Status.Suspended {
    log.Info("Reconciliation is suspended for this object")
    return ctrl.Result{}, nil
}
```

### cli

The flux cli would recognize the suspend state from the suspend object status.

```
# github.com/fluxcd/flux2/main

# get_source_git.go

func (a *gitRepositoryListAdapter) summariseItem(i int, includeNamespace bool, includeKind bool) []string {
	...
    return append(nameColumns(&item, includeNamespace, includeKind),
    //    revision, strings.Title(strconv.FormatBool(item.Spec.Suspend)), status, msg)
        revision, strings.Title(strconv.FormatBool(item.Status.Suspended)), status, msg)
}
```

The flux cli would manipulate the suspend status in addition to the normal `.spec.suspend` toggling behavior.  Though the controllers would set this status on the next reconciliation, subsequent intersitial `flux get` commands would not report this status change.

Similar to the controllers discussion above, the flux cli could optionally couple the `.spec.suspend` and suspend metadata annotations mutations.  The examples below do not show a coupled implementation.

```
# github.com/fluxcd/flux2/main

# suspend_helmrelease.go

func (obj helmReleaseAdapter) setSuspended() {
    obj.HelmRelease.Spec.Suspend = true
    obj.HelmRelease.Status.SetSuspended(true)
}

# resume_helmrelease.go

func (obj helmReleaseAdapter) setUnsuspended() {
    obj.HelmRelease.Spec.Suspend = false
    obj.HelmRelease.Status.SetSuspended(false)
}
```

### Metrics

Metrics will be reported using the object status.

```
# github.com/fluxcd/kustomize-controller/controller

# imagerepository_controller.go

// Always record suspend, readiness and duration metrics.
// r.Metrics.RecordSuspend(ctx, obj, obj.Spec.Suspend) // no longer used to report suspended
r.Metrics.RecordSuspend(ctx, obj, obj.Status.Suspended)
```

## Implementation History

tbd
