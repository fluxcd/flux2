# RFC-XXXX Advanced Dependency Management

**Status:** provisional

<!--
Status represents the current state of the RFC.
Must be one of `provisional`, `implementable`, `implemented`, `deferred`, `rejected`, `withdrawn`, or `replaced`.
-->

**Creation date:** 2026-07-10

**Last update:** 2026-11-12

## Summary

Today a `Kustomization` can only depend on another `Kustomization`, and a
`HelmRelease` can only depend on another `HelmRelease`. This RFC proposes to
let both Flux Applier APIs depend on any Kubernetes resource.

Entries in `.spec.dependsOn` get three new optional fields: `apiVersion` and
`kind` to point at any resource, and `ready` to turn the readiness check on or
off. Readiness is evaluated with the `kstatus` conventions, and the existing
`readyExpr` CEL expression can complete or replace that check. Entries that do
not use the new fields behave exactly as they do today.

A new `.spec.dependents` field lists the objects that must be deleted before
this object, so that a dependency graph can be removed in reverse order.

## Motivation

`.spec.dependsOn` only works between objects of the same kind. A
`Kustomization` cannot wait for a `HelmRelease`, and neither can wait for a
`CustomResourceDefinition`, a `Secret` created by an external system, or a
custom resource managed by a third-party controller.

This limitation was first reported in
[fluxcd/kustomize-controller#242](https://github.com/fluxcd/kustomize-controller/issues/242)
and later in [fluxcd/flux2#3364](https://github.com/fluxcd/flux2/issues/3364).
The proposal is tracked by the umbrella issue
[fluxcd/flux2#5879](https://github.com/fluxcd/flux2/issues/5879).

The documented workaround combines `.spec.dependsOn` with
`.spec.healthChecks`. It has some limits:

- The team that owns the dependency must declare the right health checks. The
  team that depends on it cannot do this when the two objects are managed by
  different teams.
- A health check runs after the manifests are applied. It delays the `Ready`
  condition, not the apply. To wait for a `cert-manager` `HelmRelease`, a
  developer team needs a wrapper `Kustomization` that only carries the health
  check.
- `.spec.healthCheckExprs` matches by API group and kind, not by name, so an
  expression cannot target a single dependency.

The Flux Operator
[`ResourceSet` API](https://fluxcd.control-plane.io/operator/resourceset/#dependency-management)
already supports dependencies on any Kubernetes resource. This RFC uses the
same API fields (`apiVersion`, `kind`, `name`, `namespace`, `ready` and
`readyExpr`), which keeps the two APIs consistent for their users.

### Goals

- Let `Kustomization` and `HelmRelease` resources depend on any Kubernetes
  resource, including on each other.
- Let users choose how a dependency is checked: the resource exists, it is
  ready according to the built-in check, or a custom CEL expression is true.
- Keep existing `.spec.dependsOn` entries working without any change.
- Share one dependency-checking implementation between kustomize-controller
  and helm-controller, so that both controllers behave the same way.
- Delete a dependency graph in reverse order, so that an object is removed
  only after everything that depends on it.

### Non-Goals

- Making the Flux and Flux Operator dependency APIs behave exactly the same.
  Both APIs are GA and have different defaults.
- Deciding the future of the `AdditiveCELDependencyCheck` feature gate. This
  RFC only describes how the new fields work with the gate as it exists today.
- Ordering the deletion of individual resources inside one object's inventory.

## Proposal

Extend the `.spec.dependsOn` entries of the Flux Applier APIs with the
optional fields `apiVersion`, `kind` and `ready`, next to the existing `name`,
`namespace` and `readyExpr` fields:

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: certs
  namespace: apps
spec:
  dependsOn:
    # Same-kind dependency, unchanged behavior.
    - name: infra
    # Cross-kind dependency on a HelmRelease managed by another team.
    - apiVersion: helm.toolkit.fluxcd.io/v2
      kind: HelmRelease
      name: cert-manager
      namespace: cert-manager
    # Existence-only dependency on a cluster-scoped resource.
    - apiVersion: apiextensions.k8s.io/v1
      kind: CustomResourceDefinition
      name: clusterissuers.cert-manager.io
      ready: false
    # Dependency on a custom resource with a CEL readiness expression.
    - apiVersion: cluster.x-k8s.io/v1beta1
      kind: Cluster
      name: staging
      namespace: capi
      readyExpr: >
        dep.status.conditions.filter(e, e.type == 'Ready')
          .all(e, e.status == 'True')
```

### Defaulting rules

The new fields default to values that keep the current behavior:

- `kind` defaults to the kind of the object that contains the entry.
- When the dependency has the same kind as that object, `apiVersion` and
  `namespace` default to the values of that object.
- When the dependency has a different kind, `apiVersion` is required and
  `namespace` has no default. Set it for namespaced resources, or leave it
  empty for cluster-scoped ones.
- `ready` defaults to `true`, so readiness is checked unless the user turns it
  off. Defaulting it to `false` would silently turn every existing `dependsOn`
  entry into an existence-only check.

An entry such as `- name: infra` therefore keeps its current meaning: the
object of the same kind named `infra`, in the same namespace, checked for
readiness.

### Readiness evaluation

For each dependency, the controller runs the following steps in order:

1. **Existence**: the referenced object is read from the cluster. If the
   object does not exist, or the cluster does not serve its kind, the
   dependency is not satisfied.
2. **Readiness toggle**: if `ready` is `false`, the dependency is satisfied by
   existence alone and the next steps are skipped. `readyExpr` is not
   evaluated, so users can turn the check off and keep their expression.
   `ResourceSet` works the same way.
3. **CEL expression**: if `readyExpr` is set, it must return `true`. It can
   use the variables `self` (the object that declares the dependency) and
   `dep` (the referenced object). When the `AdditiveCELDependencyCheck`
   feature gate is disabled (the default), evaluation stops here. When the
   gate is enabled, step 4 must also pass. This is the existing `readyExpr`
   behavior, now available for any kind.
4. **Built-in check**: the status computed by `kstatus` must be `Current`.
   For Flux objects (API group ending in `toolkit.fluxcd.io`), the `Ready`
   condition must also be `True`, as it is checked today.

The built-in check works without configuration for Kubernetes core resources
and for custom resources that follow the `kstatus` conventions
(`observedGeneration`, `Reconciling` and `Stalled` conditions). `kstatus`
considers resources without a status as `Current`. For those cases, users can
set `ready: false` or write a `readyExpr`.

When a dependency is not satisfied, the controller behaves as it does today:
the object reports the reason `DependencyNotReady` and is retried after the
`--requeue-dependency` interval.

### Deletion ordering

When a parent `Kustomization` prunes several `Kustomization` and `HelmRelease`
objects at once, their controllers delete them in parallel. An operator can be
uninstalled before the custom resources it manages. Their finalizers then
never run, and namespaces stay in `Terminating`.

`.spec.dependents` lists the objects that must be gone before this object is
deleted. An entry takes `apiVersion`, `kind`, `name` and `namespace`, with the
same defaults as `.spec.dependsOn`, and can point at any resource, Flux or
not.

```yaml
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: postgres-operator
  namespace: flux-system
spec:
  dependents:
    - apiVersion: kustomize.toolkit.fluxcd.io/v1
      kind: Kustomization
      name: apps
      namespace: flux-system
```

On deletion, the controller checks whether each dependent still exists in the
cluster before it deletes the applied resources or uninstalls the release. If
any dependent remains, the controller keeps its finalizer and checks again
after the `--requeue-dependency` interval. Once all dependents are gone,
deletion continues as it does today.

Entries can be chained to order deeper graphs. If `infra` lists `operators`
and `operators` lists `apps`, Flux deletes `apps`, then `operators`, then
`infra`.

`dependents` and `dependsOn` are independent: an entry in one does not create
an entry in the other. `.spec.dependents` is an advanced field, and users are
responsible for keeping the graph free of cycles.

#### While deletion is blocked

The controller sets `Ready` to `False` with reason `DependentsNotDeleted`,
names the remaining dependents in the message, and emits a warning event with
the same reason.

To unblock deletion, delete the dependent or remove its entry from the object
being deleted. Kubernetes accepts spec changes after the deletion timestamp is
set. Since the object is usually already gone from Git, the change is made
with `kubectl`.

#### Waiting for termination is opt-in

By default, kustomize-controller removes its finalizer as soon as it sends the
delete requests. The `Kustomization` can then be gone while its workloads are
still terminating.

To close that gap, set `.spec.deletionPolicy: WaitForTermination` on the
dependent `Kustomization` objects. A `HelmRelease` already waits, unless
`.spec.uninstall.disableWait` is `true`. `dependents` does not force either
setting, just as `dependsOn` does not force `healthChecks`.

#### Suspended objects

Both controllers handle a suspended object the same way on deletion: they skip
the pruning or the uninstall, and then remove the finalizer. The object is
gone, but its workloads stay in the cluster. A suspended dependent therefore
does not block the deletion of the object that lists it, and the deletion
order is not guaranteed for its workloads.

### User Stories

> As a developer, I want my `Kustomization` to wait for the `cert-manager`
> `HelmRelease` managed by the platform team, so that my certificates are not
> applied before cert-manager is ready.

> As a cluster administrator, I want a `HelmRelease` to wait until a CRD or a
> `Secret` provisioned by an external system exists, so that installs do not
> fail on missing prerequisites.

> As a Flux user, I want to deploy applications only after a Cluster API
> `Cluster` is ready, using either the built-in readiness detection or my own
> CEL expression when the resource does not follow the kstatus conventions.

> As a Flux user, I want my apps deleted before the database operator
> they use, so the operator can process their finalizers and no namespace
> hangs in `Terminating`.

The examples in the proposal cover these four stories. In the first one, the
platform team does not need to change anything on their side.

### Alternatives

The following alternatives are open for discussion. Each one describes what
it would change compared to the proposal.

#### Keep the healthChecks workaround

This alternative keeps the status quo and requires no API change. The limits
described in the motivation remain: teams have to coordinate in multi-tenant
setups, a dependency cannot cross between `Kustomization` and `HelmRelease`,
and a dependency cannot be checked for existence only.

#### Express existence-only checks with a `readyExpr` keyword

Instead of adding the `ready` field, a reserved value in `readyExpr` would
turn the readiness check off. The API change is then limited to `apiVersion`
and `kind`. The constant expression `"true"` cannot be that value: when the
`AdditiveCELDependencyCheck` gate is enabled, expressions are additive and
the built-in check still runs. The value would have to be a keyword that is
not a CEL expression, in a field that otherwise only holds CEL expressions.

#### Require `readyExpr` for kinds not owned by the controller

The controller would require a CEL expression for every dependency on a kind
it does not manage, and would not assume any readiness logic for those kinds.
Users would then also write expressions for Kubernetes core resources, Flux
objects of the other kind and custom resources that `kstatus` can already
check. `ResourceSet` uses `kstatus` as its fallback.

#### Align fully with the ResourceSet API

`ready` would default to `false` and the `AdditiveCELDependencyCheck` feature
gate would be removed, so that the Applier APIs behave exactly like
`ResourceSet`. Both changes are breaking: the first one turns every existing
`dependsOn` entry into an existence-only check, and the second one affects
users who rely on the gate.

#### Derive dependents from `dependsOn`

No new field would be added. The controller would block the deletion of an
object while another object lists it in `dependsOn`. Each controller would
have to look up objects of the other kind, which creates a hard coupling
between the two Applier APIs. The wait would also apply to all existing users
of `dependsOn`, who could not drop a single dependent to unblock a deletion
during an incident.

#### Use owner references with foreground deletion

The Kubernetes garbage collector would be used instead of a new field. With
foreground deletion, the dependents are deleted together with their owner, so
removing `infra` from Git would also delete `apps`. Owner references cannot
cross namespaces, and a namespaced object cannot own a cluster-scoped one.

#### Require `WaitForTermination` with `dependents`

Every `Kustomization` listed in `dependents` would have to set
`.spec.deletionPolicy: WaitForTermination`, and every listed `HelmRelease`
would have to keep the uninstall wait enabled. Reverse ordering would then
also cover the workloads, without extra configuration to think about.

The drawback is that a field on one object would decide what is valid in the
spec of another object. The controller would have to read the spec of each
dependent and choose what to do when the setting is missing, for example
refuse the deletion or report an error. Dependents that are not Flux objects
have no such setting. In the proposal the two settings stay separate, in the
same way that `dependsOn` does not require `healthChecks` on the dependency.

## Design Details

The `DependencyReference` type in `fluxcd/pkg/apis/meta` is extended as
follows:

```go
type DependencyReference struct {
    // APIVersion of the resource to depend on.
    // Defaults to the API version of the Flux Applier API resource
    // containing the reference when the dependency is of the same kind.
    // +optional
    APIVersion string `json:"apiVersion,omitempty"`

    // Kind of the resource to depend on.
    // Defaults to the kind of the Flux Applier API resource
    // containing the reference.
    // +optional
    Kind string `json:"kind,omitempty"`

    // Name of the resource to depend on.
    // +required
    Name string `json:"name"`

    // Namespace of the resource to depend on.
    // Defaults to the namespace of the Flux Applier API resource
    // containing the reference when the dependency is of the same kind.
    // +optional
    Namespace string `json:"namespace,omitempty"`

    // Ready toggles the readiness check for this dependency.
    // When set to false, the dependency is satisfied as soon as it
    // exists and readyExpr is not evaluated. Defaults to true.
    // +optional
    Ready *bool `json:"ready,omitempty"`

    // ReadyExpr is a CEL expression that can be used to assess the
    // readiness of a dependency. When specified, the built-in readiness
    // check is replaced by the logic defined in the CEL expression.
    // To make the CEL expression additive to the built-in readiness check,
    // the feature gate `AdditiveCELDependencyCheck` must be set to `true`.
    // +optional
    ReadyExpr string `json:"readyExpr,omitempty"`
}
```

The dependency evaluation matrix, where "built-in" is the `kstatus` check
described in the proposal:

|     `ready`      | `readyExpr` | `AdditiveCELDependencyCheck` |            Dependency is satisfied when             |
|:----------------:|:-----------:|:----------------------------:|:---------------------------------------------------:|
| `true` (default) |    unset    |             any              |            it exists and built-in passes            |
| `true` (default) |     set     |      disabled (default)      |         it exists and expression is `true`          |
| `true` (default) |     set     |           enabled            | it exists, expression is `true` and built-in passes |
|     `false`      |     any     |             any              |                      it exists                      |

- **Shared code**: the existence checks for `dependsOn` and `dependents`, the
  `kstatus` check, the CEL evaluation and the defaulting move to the
  `fluxcd/pkg/runtime/dependency` package, next to the existing topological
  sort. The sort is updated to include the kind, so that graphs mixing kinds
  are ordered and cycle-checked correctly.
- **Controller-specific checks** stay in the controllers and run after the
  shared ones. One example is kustomize-controller verifying that a same-kind
  dependency with the same source has applied the current revision.
- **CLI**: the `--depends-on` flag of `flux create kustomization` and
  `flux create helmrelease` accepts the format
  `[<apiVersion>/<Kind>/][<namespace>/]<name>[:<ready>][@<readyExpr>]`. An
  `apiVersion` without a group, such as `v1`, refers to the core group. The
  existing `<name>` and `<namespace>/<name>` formats do not change.
- **Upgrade**: manifests using the new fields should only be applied after
  the Flux CRDs are upgraded. With older CRDs, the API server drops the
  unknown fields and the entry is read as a same-kind reference. A CEL
  validation rule in the CRD schema requires `apiVersion` when `kind` differs
  from the kind of the containing object.
- **No watches**: the controller does not watch the referenced objects. It
  checks unsatisfied dependencies again at the requeue interval.
- **Cycles across controllers**: a cycle between a `Kustomization` and a
  `HelmRelease` cannot be detected by either controller. As with same-kind
  cycles today, the objects report `DependencyNotReady` until the user breaks
  the cycle.
- **Permissions**: the controller reads the referenced objects with its own
  service account, as it does for same-kind dependencies today. Because
  `readyExpr` can now read any kind, the `data` of a `Secret` in another
  namespace is not available to the expression, as in `ResourceSet`.

Both Applier APIs gain `.spec.dependents`, which uses a new
`DependentReference` type in `fluxcd/pkg/apis/meta`. It has no `ready` or
`readyExpr` field, because a dependent is only checked for existence:

```go
// DependentReference identifies an object that must be deleted before the
// object that lists it.
type DependentReference struct {
    // APIVersion of the dependent object.
    // Defaults to the API version of the object containing the reference
    // when the dependent is of the same kind.
    // +optional
    APIVersion string `json:"apiVersion,omitempty"`

    // Kind of the dependent object.
    // Defaults to the kind of the object containing the reference.
    // +optional
    Kind string `json:"kind,omitempty"`

    // Name of the dependent object.
    // +required
    Name string `json:"name"`

    // Namespace of the dependent object.
    // Defaults to the namespace of the object containing the reference
    // when the dependent is of the same kind.
    // +optional
    Namespace string `json:"namespace,omitempty"`
}
```

## Implementation History

<!--
Major milestones in the lifecycle of the RFC such as:
- The first Flux release where an initial version of the RFC was available.
- The version of Flux where the RFC graduated to general availability.
- The version of Flux where the RFC was retired or superseded.
-->
