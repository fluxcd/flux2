# RFC-0011: OpenTelemetry Tracing

**Status:** provisional

<!--
Status represents the current state of the RFC.
Must be one of `provisional`, `implementable`, `implemented`, `deferred`, `rejected`, `withdrawn`, or `replaced`.
-->

**Creation date:** 2025-04-24

**Last update:** YYYY-MM-DD

## Summary
The aim is to collect traces via OpenTelemetry across all Flux related objects, such as HelmReleases, Kustomizations and among others. These may be sent towards a tracing provider where are going to be stored and visualized. Thereby, this may involve a new API definition obj called `Trace`, which may be capable of linking all the `EventSources` and send them out to a reusable tracing `Provider`. In this way, it could facilitate the observability and monitoring of Flux related objects.

## Motivation
This RFC was born out of a need for end-to-end visibility into Flux’s multi-controller GitOps workflow. At the time Flux was one monolithic controller; it has since split into several specialized controllers (source-controller, kustomize-controller, helm-controller, notification-controller, etc.), which makes tracing the path of a single “Git change → applied resource → notification” much harder. 

Correlate a Git commit with all downstream actions. You want one single trace that (via multiple spans) shows:
- Source-controller current revision ID.
- Any Kustomize or Helm reconciliations that ran.
- Events emitted and notifications sent by the notification-controller.

On top of this, can be built custom UIs that surface trace timelines alongside Git commit or Docker image tags, so operators can say “what exactly happened when I tagged v1.2.3?” in a single pane of glass. 

By extending Flux’s CRD objects, users can manage tracing settings (sampling rates, backends, object filters). Therefore, users may not have to implement tools/sidecars around to maintain. 

### Goals
- **End-to-end GitOps traceability:** Capture the traces that follows a Git change through all Flux controllers for simply debugging and root-cause analysis.
- **Declarative, CRD-drive configuration:** Reuse the concept of `Provider` and a similar definition as `Alerts` to build a new API/CR called `Trace`. Therefore, users can link `EventSources` and `Provider` where trace will be sent. Additionally, other setting can be set as sampling rates.
- **Notification-Controller as the trace collector:** Leverage the notification-controller's existing event watching pipeline to ingest reconciliation events and turn me into OpenTelemetry spans, being forwarwed to an OLTP-compatible backend - `Provider`.
- **Cross controller span correlation:** Ensure spans are emitted from multiple, stateless controller can be stiched together into a single trace by using Flux "revision" annotation (GitRepository sync to a downstream Kustomization/HelmRelease reconciliations).

### Non-Goals
- **Not a full-tracing backend:** We won't build or bundle a storage/visualization system. Users may have to still rely on a external collector for long-term retention, querying and UI.
- **Not automatic instrumentation of user workloads:** This integration only captures FluxCd controller events (Source, Kustomize, Helm, etc.). It won't auto-inject spans into your application pods or third-party controllers running in the same cluster.
- **Not a replacement for metrics or logs:** Flux's existing Prometheus metrics and structural logging remain the primary way to monitor performance and errors. Tracing is purely for request-flow visibility, not for time-series monitoring or log aggregation.
- **No deep-code lelve spans beyond CRUD events:** Will emit spans around high-level reconciliation steps (e.g. "reconcile GitRepository", "dispatch Notification"), but we're not aiming to instrument every internal function call or library method within each controller.
- **Not a service mesh integration:** It's not plan of the scope tieing this into Istio, Linkerd, or other mesh-sidecar approaches. It's strictly a controller-drive, CRD-based model.
- **No per-span custom enrichment beyond basic metadata:** While Trace CRD will let you filter which Flux object kinds to trace (`EventSources`), it won't support (at least initially) complex span attributes or tag-enrichment rules. You may have to handle those in your downstream collector/processor if needed.
- **Not a replacement for user-driven OpenTelemetry SDKs:** If you already have a Go-based operator that embed OpenTelemetry's SDK directly, this feature won't override or duplicate that. Think about it as a complementary, declrartive layer for flux controllers.

## Proposal

Add a new `Trace` custom definition in Flux's notification-controller. Under this, there is a conjuntion of: `EventSources` Flux's related objects to get the traces on and `Provider` external system where all the traces are going to be sent towards.

Additionally, as part of the `Provider`, we may have to onboard new type(s) to tackle OLTP compliant systems.
<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation.

If the RFC goal is to document best practices,
then this section can be replaced with the actual documentation.
-->

### User Stories

#### Story 1
 > As a cluster administrator, I want to see everything that happened 
 > since a git change occurred in a single trace. All the applied yaml 
 > files in Source Controller, notifications that went out to Notifications 
 > Controller, all the HelmReleases that were applied in Helm-Controller, 
 > Kustomize Controller, etc...

For instance, having the following setup:


```yaml
apiVersion: notification.toolkit.fluxcd.io/v1beta1
kind: Provider
metadata:
  name: jaeger
  namespace: default
spec:
  type: jaeger
  address: http://jaeger-collector.jaeger-system.svc.cluster.local:9411
  secretRef:
    name: jaeger-secret
---
apiVersion: notification.toolkit.fluxcd.io/v1beta1
kind: Trace
metadata:
  name: webapp
  namespace: default
spec:
  providerRef: 
    name: jaeger
  eventSources:
    - kind: Kustomization
      name: webapp-backend
    - kind: HelmRelease
      name: webapp-frontend
```

#### Story 2
 > I want to build a UI using trace data to track release changes and integrate deeply 
 > with a Git commit/tag, a Docker image tag, and the GOTK flow of operations

### Alternatives
- Addition of a new `Provider` type to "onboard" OLTP-compliant systems. Could be adressed in generic way, just adding a new type to tackle them all or adding specific integrations for the most relevant ones: [Jaeger](https://www.jaegertracing.io/) & [Zipkin](https://zipkin.io/).
<!--
List plausible alternatives to the proposal and explain why the proposal is superior.

This is a good place to incorporate suggestions made during discussion of the RFC.
-->

## Design Details

Adding a new API `Trace` on Flux to manage the link between `Provider` (where the traces are going to be sent) and `EventSources` (Flux's related objects part of the "tracing chain").

Example of `Trace` custom resource alongside the `Provider`:
```yaml
apiVersion: notification.toolkit.fluxcd.io/v1
kind: Trace
metadata:
  name: webapp
  namespace: default
spec:
  providerRef: 
    name: jaeger
  eventSources:
    - kind: Kustomization
      name: webapp-backend
    - kind: HelmRelease
      name: webapp-frontend
---
apiVersion: notification.toolkit.fluxcd.io/v1
kind: Provider
metadata:
  name: jaeger
  namespace: default
spec:
  type: jaeger
  address: http://jaeger-collector.jaeger-system.svc.cluster.local:9411
  secretRef:
    name: jaeger-secret
```

The user will be able to define a Flux `Trace` custom resource and deploy it to the cluster. The trace may rely on revision annotation as the correlation key due to every Flux reconciliation writes the same `source.toolkit.fluxcd.io/reconcile-revision` annotation onto the object (and is emitted on its Event). That revision string (e.g. a Git commit SHA) is your trace ID surrogate. Therefore, if Source Controller emits an event for `GitRepository` with `revision = SHA123`, later on Kustomize Controller will emit another event for `Kustomization` with the same `revision = SHA123`.

In this way, rather than trying to carry a live span context through controller A -> API server -> controller B:
1. Watch Flux Events.
2. Group them by ``event.objectRef.kind`` + ``revision``. Hence, based on `EventSources` field:
     - First time you see `GitRepository@SHA123`, spawn a root span with traceID = hash(SHA123).
     - When you see `Kustomization@SHA123`, spawn a child span (parent = root) in that same trace.
     - And so on for `HelmRelease@SHA123`, `Notification@SHA123`, etc.
3. Subsequently, it may generate the following parent/child relationship:
  ```text
  root span:      “reconcile GitRepository SHA123”      (spanID = H1)
    ↳ child span: “reconcile Kustomization foo”        (spanID = H2, parent = H1)
        ↳ child span: “reconcile HelmRelease bar”     (spanID = H3, parent = H2)
  ```

However, in order to make this design work, we need to ensure each controller:
- Emits its normal Kubernetes `Event` with the `revision` annotation (already built-in).
- Optionally tags the Event with `flux.event.type` and timestamp (they already do).

About sending the traces, `Provider` custom resource is going to be reused as the target external system where all the traces are going to be sent towards, based on each `Trace` custom resource definition. Thus, as most of the already existing providers are non-OLTP compliant, there is an open point about either add a new generic type to handle all OLTP's external systems or add a specific ones for the most relevant ones. Anyhow, the user should be completely agnostic about this point, because `Provider` custom resource definition may not differ much from the already existing ones.

## Implementation History

<!--
Major milestones in the lifecycle of the RFC such as:
- The first Flux release where an initial version of the RFC was available.
- The version of Flux where the RFC graduated to general availability.
- The version of Flux where the RFC was retired or superseded.
-->
