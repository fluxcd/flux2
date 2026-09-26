# RFC-XXXX Advanced Dependency Management

**Status:** provisional

<!--
Status represents the current state of the RFC.
Must be one of `provisional`, `implementable`, `implemented`, `deferred`, `rejected`, `withdrawn`, or `replaced`.
-->

**Creation date:** 2026-07-10

**Last update:** 2026-07-14

## Summary

This RFC proposes extending the dependency model of the Flux Applier APIs
(`Kustomization` and `HelmRelease`) from dependencies between resources of the
same kind to dependencies on arbitrary Kubernetes resources.

The `DependencyReference` type used by `.spec.dependsOn` gains optional
`apiVersion` and `kind` fields for referencing any Kubernetes resource, and an
optional `ready` field for toggling the readiness check of a dependency.
Readiness of arbitrary resources is evaluated using the `kstatus` conventions,
and can be customized or replaced with the existing `readyExpr` CEL expression
support. When the new fields are omitted, the behavior of `.spec.dependsOn` is
identical to previous Flux versions.

## Motivation

Today `.spec.dependsOn` only works between resources of the same kind: a
`Kustomization` can depend on another `Kustomization`, and a `HelmRelease` can
depend on another `HelmRelease`. Users cannot express that a `Kustomization`
must wait for a `HelmRelease`, nor that either must wait for an arbitrary
cluster resource such as a `CustomResourceDefinition`, a `Secret` issued by an
external system, or a custom resource managed by a third-party controller.

This limitation was first reported in
[fluxcd/kustomize-controller#242](https://github.com/fluxcd/kustomize-controller/issues/242)
and later in [fluxcd/flux2#3364](https://github.com/fluxcd/flux2/issues/3364),
which became one of the most upvoted feature requests in the Flux project. The
proposal is tracked by the umbrella issue
[fluxcd/flux2#5879](https://github.com/fluxcd/flux2/issues/5879).

The documented workaround is to combine `.spec.dependsOn` with
`.spec.healthChecks` on the dependency: the object being depended upon declares
health checks for the resources it manages, and its `Ready` condition then
gates the dependents. This workaround has real limits:

- It requires the owner of the dependency to declare the appropriate health
  checks. In multi-tenant setups the dependent and the dependency are often
  managed by different teams, and the dependent team can neither control nor
  guarantee the health checks of the dependency.
- It requires both sides to use the same Applier kind, since `dependsOn`
  cannot cross kinds. When a platform team manages `cert-manager` with a
  `HelmRelease`, a developer team using a `Kustomization` cannot depend on it
  without introducing a wrapper `Kustomization` whose sole purpose is to carry
  health checks.
- Health checks gate the readiness of the dependency, not the reconciliation
  of the dependent. There is no way to prevent a resource from being applied
  until an arbitrary object exists in the cluster.

The Flux Operator already solved this problem in the
[`ResourceSet` API](https://fluxcd.control-plane.io/operator/resourceset/#dependency-management),
where a dependency references any Kubernetes resource by `apiVersion`, `kind`,
`name` and `namespace`, with an optional readiness toggle and CEL expression.
Bringing the same capability to the Flux Applier APIs makes it available to
the wider Flux user base, and keeping the two APIs consistent where practical
reduces the cognitive load for users moving between them.

### Goals

- Allow `Kustomization` and `HelmRelease` resources to declare dependencies on
  arbitrary Kubernetes resources, including on each other.
- Support gating on existence only, on built-in readiness detection, or on a
  custom CEL expression, for any referenced kind.
- Preserve the exact behavior of existing `.spec.dependsOn` entries: current
  manifests must continue to work without modification.
- Keep the extended `DependencyReference` type consistent with the Flux
  Operator `ResourceSet` dependency API where this does not require breaking
  changes.
- Share a single dependency-checking implementation between kustomize-controller
  and helm-controller so both controllers behave identically.

### Non-Goals

- Fully converging the Flux and Flux Operator dependency APIs and semantics.
  Both APIs are GA and differ in defaults.
- Deciding the long-term future of the `AdditiveCELDependencyCheck` feature
  gate (retention, deprecation or removal). This RFC defines how the new
  fields interact with the gate as it exists today, and nothing more.

## Proposal

Extend the `DependencyReference` type shared by the Flux Applier APIs with
optional `apiVersion`, `kind` and `ready` fields, alongside the existing
`name`, `namespace` and `readyExpr` fields:

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

To guarantee backwards compatibility, the new fields default to values that
reproduce the current behavior:

- `kind` defaults to the kind of the object containing the reference.
- When the dependency kind equals the kind of the containing object,
  `apiVersion` defaults to the API version of the containing object, and
  `namespace` defaults to the namespace of the containing object.
- When the dependency kind differs from the kind of the containing object,
  `apiVersion` is required, and `namespace` must be set explicitly for
  namespaced resources and omitted for cluster-scoped resources. The
  controller does not guess the scope of kinds it does not own.
- `ready` defaults to `true` for every referenced kind, meaning readiness is
  checked unless explicitly disabled. The default is intentionally uniform to
  ensure identical dependency entries behave consistently across all kinds, and
  changing it for the Applier APIs would be a breaking change for existing
  `dependsOn` users.

An entry such as `- name: infra` therefore keeps meaning "the object of my
own kind and API version named `infra` in my namespace, checked for
readiness", exactly as before.

### Readiness evaluation

For each dependency, after defaults are applied, the controller evaluates the
following steps in order:

1. **Existence**: the referenced object is fetched from the cluster. If it
   does not exist, the dependency is not satisfied.
2. **Readiness toggle**: if `ready` is `false`, the dependency is satisfied by
   existence alone and evaluation stops here. A configured `readyExpr` is
   deliberately not evaluated in this case, so users can toggle readiness
   checks off without having to remove an expression they intend to keep.
   This matches the `ResourceSet` semantics, where the expression is only
   evaluated when the readiness check is enabled.
3. **CEL expression**: if `readyExpr` is set, it is evaluated with the
   variables `self` (the object declaring the dependency) and `dep` (the
   referenced object), and must return `true`. When the
   `AdditiveCELDependencyCheck` feature gate is disabled (the default), a
   satisfied expression completes the evaluation; when the gate is enabled,
   evaluation continues with the built-in check, and both must pass. This is
   the existing `readyExpr` behavior, unchanged, now applicable to any kind.
4. **Built-in check**: the status of the referenced object is computed
   following the `kstatus` conventions and must report `Current`. For every
   dependency whose API group is `toolkit.fluxcd.io`, an additional check 
   verifies the truthiness of the `Ready` condition. This preserves the
   existing same-kind readiness semantics (a `Kustomization` or `HelmRelease`
   is ready only when its `Ready` condition is defined and `True`) and extends
   them to cross-kind Flux dependencies such as a `Kustomization` depending on
   a `HelmRelease`.

The built-in check is available for every kind, not only for Flux objects.
Kubernetes core resources are handled by `kstatus` natively, and custom
resources following the `kstatus` conventions (`observedGeneration`,
`Reconciling` and `Stalled` conditions) are handled without any configuration.
Resources without any status are considered `Current` by `kstatus`, which is
why the existence-only toggle and `readyExpr` exist for cases where built-in
detection is insufficient or too permissive.

When a dependency is not satisfied, the controller behaves as it does today:
reconciliation is deferred, an event with reason `DependencyNotReady` is
emitted, and the object is requeued at the dependency requeue interval.

### User Stories

#### Depend on a HelmRelease from a Kustomization

> As a developer, I want my `Kustomization` to wait for the `cert-manager`
> `HelmRelease` managed by the platform team, so that my certificates are not
> applied before cert-manager is ready.

```yaml
spec:
  dependsOn:
    - apiVersion: helm.toolkit.fluxcd.io/v2
      kind: HelmRelease
      name: cert-manager
      namespace: cert-manager
```

The platform team does not need to change anything on their side, and neither
team needs wrapper objects nor duplicated health checks.

#### Gate on the existence of a resource

> As a cluster administrator, I want a `HelmRelease` to wait until a CRD or a
> `Secret` provisioned by an external system exists, so that installs do not
> fail on missing prerequisites.

```yaml
spec:
  dependsOn:
    - apiVersion: apiextensions.k8s.io/v1
      kind: CustomResourceDefinition
      name: clusterissuers.cert-manager.io
      ready: false
```

The existence-only check works regardless of the `AdditiveCELDependencyCheck`
feature gate setting.

#### Depend on a third-party custom resource

> As a Flux user, I want to deploy applications only after a Cluster API
> `Cluster` is ready, using either the built-in readiness detection or my own
> CEL expression when the resource does not follow the kstatus conventions.

Users can rely on the built-in check first and only reach for `readyExpr` when
the resource status requires it, in the same way `.spec.healthCheckExprs`
complements the built-in health checks (see RFC-0009).

### Alternatives

#### Keep the healthChecks workaround

The status quo. As described in the motivation, it couples teams in
multi-tenant setups, cannot cross Applier kinds, and cannot gate on existence.
The volume of user demand over five years indicates the workaround is not an
acceptable answer.

#### Express existence-only checks with `readyExpr: "true"`

Instead of adding the `ready` field, users could set `readyExpr` to the
constant expression `"true"`. This was the initially preferred design, since
it required no API change beyond `apiVersion` and `kind`; the problem of this
approach is the constant expression only disables the built-in check while the
`AdditiveCELDependencyCheck` feature gate is disabled. With the gate enabled,
there is no way to express an existence-only check (see the matrix below).
Relying on a magic expression value is also poor UX compared to an explicit
boolean that matches the `ResourceSet` API.

#### Require `readyExpr` for kinds not owned by the controller

Requiring a CEL expression for every dependency on a kind the controller does
not manage would spare it from assuming readiness semantics for foreign types.
The drawback is the built-in `kstatus` check already works for Kubernetes
core resources, Flux objects and well-behaved custom resources, and forcing
expressions where the built-in check suffices is unnecessary friction. The
`ResourceSet` API already uses `kstatus` checks as a fallback.

#### Align fully with the ResourceSet API

Defaulting `ready` to `false` and removing the `AdditiveCELDependencyCheck`
feature gate would make the Applier APIs behave identically to `ResourceSet`.
Both changes are breaking: the former silently changes every existing
`dependsOn` entry to an existence-only check, and the latter breaks users who
rely on the gate. With this proposal, the two APIs expose the same fields with
the same meaning, differing only in the `ready` default to avoid a breaking
change.

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
|     `false`      |    unset    |             any              |                      it exists                      |
|     `false`      |     set     |             any              |                      it exists                      |

The existence check, `kstatus` computation, CEL evaluation and defaulting are
consolidated in the `fluxcd/pkg/runtime/dependency` package, so that
kustomize-controller and helm-controller share one implementation instead of
maintaining two. Controller-specific semantics remain in the
controllers as an add-on executed after the shared checks pass, such as
kustomize-controller verifying that a same-kind dependency sharing the same
source has applied the current source revision. The topological sort used to order
objects by their dependencies operates on typed references, so graphs mixing
kinds are ordered and cycle-checked correctly.

The `flux create kustomization` and `flux create helmrelease` commands accept
the extended reference in the `--depends-on` flag using the format
`[<apiVersion>/<Kind>/][<namespace>/]<name>[:<ready>][@<readyExpr>]`, with the
existing `<name>` and `<namespace>/<name>` formats unchanged.

The new fields are opt-in per dependency entry and introduce no new feature
gate. Because the API server prunes fields unknown to the CRD schema,
manifests using the new fields must only be applied after the Flux CRDs have
been upgraded: on a cluster with older CRDs, a typed reference would be
silently reduced to a same-kind reference.

Dependencies remain passive checks: the controllers do not watch the
referenced objects and re-evaluate unsatisfied dependencies at the requeue
interval, as described in the proposal. A dependency cycle spanning both
Applier controllers, such as a `Kustomization` and a `HelmRelease` depending
on each other, cannot be detected by either controller. As with same-kind
cycles today, all objects involved report `DependencyNotReady` until the
cycle is broken by the user.

## Implementation History

<!--
Major milestones in the lifecycle of the RFC such as:
- The first Flux release where an initial version of the RFC was available.
- The version of Flux where the RFC graduated to general availability.
- The version of Flux where the RFC was retired or superseded.
-->
