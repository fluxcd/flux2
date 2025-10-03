# RFC-0012 Namespace Wildcard Alert Support

<!--
The title must be short and descriptive.
-->

**Status:** provisional

<!--
Status represents the current state of the RFC.
Must be one of `provisional`, `implementable`, `implemented`, `deferred`, `rejected`, `withdrawn`, or `replaced`.
-->

**Creation date:** 2025-06-29

**Last update:** 2025-06-29

## Summary

Enable `Alert.spec.eventSources[].namespace: "*"` in the FluxCD Notification Controller behind a dedicated feature gate (`--enable-namespace-wildcard-events`) while preserving the global kill-switch (`--no-cross-namespace-refs`) for multi-tenant isolation. This provides cluster-wide alert configuration with minimal operator effort and no additional surface area for secret leakage.

## Motivation

Flux’s Notification Controller today requires one `Alert` per namespace or explicit namespace lists, making cluster-wide error monitoring laborious and brittle. A user-land [PR #504](https://github.com/fluxcd/notification-controller/pull/504) to allow wildcard namespaces has been on hold pending a formal RFC, and [issue #71](https://github.com/fluxcd/notification-controller/issues/71) has long requested this enhancement. Meanwhile, multi-tenant operators rely on `--no-cross-namespace-refs=true` to enforce strict per-namespace boundaries.


### Goals

- **Operator Convenience**: Support a single `Alert` watching all namespaces via `namespace: "*"` without templating per-namespace resources.  
- **Opt-in Safety**: Gate wildcard support behind `--enable-namespace-wildcard-events` (default off).  
- **Global Kill-Switch**: Honor `--no-cross-namespace-refs=true` to disable wildcard (and all cross-namespace refs) when set.  
- **Minimal Surface Area**: No new CRD types or cross-namespace secret reads.

### Non-Goals

- Introducing new CRDs (e.g., `ClusterAlert`).  
- Implementing complex cross-namespace authorization (e.g., ReferenceGrant).  
- Allowing `Provider` references across namespaces.

## Proposal

1. **New Feature Flag**  
   - Introduce `--enable-namespace-wildcard-events` (boolean; default `false`).  
   - When `true`, controllers accept literal `"*"` in `Alert.spec.eventSources[].namespace`.  
   - When `false`, any `"*"` is rejected during validation with an explicit error.

2. **Interaction with `--no-cross-namespace-refs`**  
   - `--no-cross-namespace-refs=true` remains the global kill-switch: if set, wildcard is rejected regardless of the new flag.  

3. **CRD Validation**  
   - Update the `Alert` CRD schema (`spec.eventSources[].namespace`) to allow `"*"` only if `--enable-namespace-wildcard-events=true`.  
   - Webhook returns:
     ```
     spec.eventSources[i].namespace: '*' is not allowed; enable via --enable-namespace-wildcard-events
     ```
     or, if `--no-cross-namespace-refs=true`:
     ```
     spec.eventSources[i].namespace: '*' is disallowed by --no-cross-namespace-refs
     ```

4. **RBAC Requirements**  
   - To monitor all namespaces, the controller’s ServiceAccount must have `list,watch` on Flux source CRs (e.g., `GitRepository`, `HelmRelease`) cluster-wide.  
   - In tenant-isolated installs where the SA lacks these permissions, wildcard support is effectively inert.

5. **Secret Access Boundary**  
   - Providers continue to reference secrets in their own namespace; no cross-namespace secret reads are introduced by wildcard alerts.


### User Stories

- **Cluster Operator**  
  > As a cluster operator, I want to define a single Alert in `flux-system` that picks up all HelmRelease failures across every namespace so that I don’t need to manage per-namespace Alert CRDs.

- **Multi-Tenant Admin**  
  > As a multi-tenant platform admin, I want to ensure that no tenant can enable wildcard alerts unless I explicitly allow it, and I want a single flag (`--no-cross-namespace-refs=true`) to disable all cross-namespace features.


### Alternatives

- **ReferenceGrant-Gated Wildcard**: Leverage Kubernetes ReferenceGrant API for explicit per-namespace grants (rejected due to KEP-3766 closure).  
- **Namespace Label Selector**: Use `spec.namespaceSelector` to select labeled namespaces (requires cluster-wide label management).  
- **Namespace Regex Matching**: Permit regex patterns in place of exact namespace names (error-prone and overly broad).  
- **ClusterAlert CRD**: Introduce a cluster-scoped Alert type (adds new API surface).  
- **ResourceSet Templating**: Use Flux ResourceSet to generate per-namespace Alerts (still creates multiple CRs).


## Design Details

- **CLI Changes**: Add `--enable-namespace-wildcard-events` to controller options alongside existing flags like `--no-cross-namespace-refs`.  
- **Validation Webhook**: Enforce schema constraint on `namespace` field based on feature gates.  
- **Controller Logic**:  
  - On reconciliation, if wildcard is enabled and not globally disabled, list/watch across all namespaces.  
  - Otherwise, restrict to the `Alert`’s own namespace.  
- **Documentation**: Update [Flux Notification Controller Options](https://fluxcd.io/flux/components/notification/options/) and the Alerts guide to include wildcard examples and flag semantics.


## Implementation History
- **2025-06-29**: Draft RFC-0012 created.  
- **TBD**: Feature implementation, tests, docs.  
- **TBD**: Community review and merge into `flux2/rfcs`.  