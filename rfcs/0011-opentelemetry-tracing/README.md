# RFC-0011: OpenTelemetry Tracing

**Status:** provisional

<!--
Status represents the current state of the RFC.
Must be one of `provisional`, `implementable`, `implemented`, `deferred`, `rejected`, `withdrawn`, or `replaced`.
-->

**Creation date:** 2025-04-24

**Last update:** 2025-05-15

## Summary
The aim is to be able to collect traces via OpenTelemetry (OTel) across all FluxCD related objects, such as HelmReleases, Kustomizations and among others. These may be sent towards a tracing provider where may be potentially stored and visualized. Flux does not have any responsability on storing and visualizing those, it keeps being completely stateless. Thereby, being seamless for the user, the implementation is going to be part of the already existing `Alert` API Type. Therefore, `EventSources` is going to discriminate the events belonging to the specific sources, which are going to be looked up to and send them out towards the `Provider` set. In this way, it could facilitate the observability and monitoring of FluxCD related objects.

## Motivation
This RFC was born out of a need for end-to-end visibility into Flux’s multi-controller GitOps workflow. At the time Flux was one monolithic controller; it has since split into several specialized controllers (source-, kustomize-, helm-, notification-, etc.), which makes tracing the path of a single "Source change → applied resource → notification” much harder. Additionally, 
users may not have to implement tools/sidecars around to maintain.

Correlate a Git commit with all downstream actions. You want one single trace that (via multiple spans) shows:
- Alert reference based on a unique ID (root span).
- Any source pulling new content based on a new Digest Checksum.
- Any subsequent rencolitiation that ran.
- Events emitted and notifications sent by the notification-controller.

On top of this, can be built custom UIs that surface trace timelines alongside Git commit or Docker image tags, so operators can say “what exactly happened when I tagged v1.2.3?” in a single pane of glass. 

### Goals
- **End-to-end GitOps traceability:** Capture the traces that follows "a Git change" (any source) through all FluxCD controllers for simply debugging and root-cause analysis.
- **Declarative, CRD-drive configuration:** Reuse the concept of `Alerts` to be able to populate this feature over, out-of-the-box. Therefore, users can link `EventSources` and `Provider` where trace will be sent.
- **Notification Controller as the trace-collector:** Leverage the notification-controller's existing event watching pipeline to ingest reconciliation events and turn me into OpenTelemetry spans, being forwarwed to an OLTP-compatible backend - `Provider`.
- **Cross-controller span correlation:** Ensure spans are emitted from multiple, stateless controller can be stiched together into a single trace by using Flux "revision" annotation.

### Non-Goals
- **Not a full-tracing backend:** We won't build or bundle a storage/visualization system. Users may have to still rely on a external collector for long-term retention, querying and UI.
- **Not automatic instrumentation of user workloads:** This integration only captures FluxCd controller events (Source, Kustomize, Helm, etc.). It won't auto-inject spans into your application pods or third-party controllers running in the same cluster.
- **Not a replacement for metrics or logs:** Flux's existing Prometheus metrics and structural logging remain the primary way to monitor performance and errors. Tracing is purely for request-flow visibility, not for time-series monitoring or log aggregation.
- **No deep-code lelve spans beyond CRUD events:** Will emit spans around high-level reconciliation steps (e.g. "reconcile GitRepository", "dispatch Notification"), but we're not aiming to instrument every internal function call or library method within each controller.
- **Not a service mesh integration:** It's not plan of the scope tieing this into Istio, Linkerd, or other mesh-sidecar approaches. It's strictly a controller-drive, CRD-based model.
- **No per-span custom enrichment beyond basic metadata:** At least intially, it won't support complex span attributes or tag-enrichment rules. You may have to handle those in your downstream collector/processor if needed.
- **Not a replacement for user-driven OpenTelemetry SDKs:** If you already have a Go-based operator that embed OpenTelemetry's SDK directly, this feature won't override or duplicate that. Think about it as a complementary, declarative layer for flux controllers.

## Proposal
<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation.

If the RFC goal is to document best practices,
then this section can be replaced with the actual documentation.
-->
The implementation will extend the notification-controller with OpenTelemetry tracing capabilities by leveraging the existing Alert API object model. This approach maintains Flux's declarative configuration paradigm while adding powerful distributed tracing functionality.

### Core Implementation Strategy
1. **Extend the notification-controller:** Add OpenTelemetry tracing support to the notification-controller, which already has visibility into events across the Flux ecosystem.
2. **Leverage existing Alert CRD structure:** Use the Alert Kind API object as the configuration entry point, where:
     - `EventSources` define which Flux resources to trace (GitRepositories, Kustomizations, HelmReleases, etc.).
     - `Provider` specifies where to send the trace data (Jaeger, Tempo, or other OpenTelemetry-compatible backends).
3. **Span generation and correlation:** Generate spans for each reconciliation event from watched resources, ensuring proper parent-child relationships and context propagation using Flux's revision annotations as correlation identifiers.
4. **Provider compatibility and fallback mechanism:** The implementation supports any provider that implements the OpenTelemetry Protocol (OTLP). When traces are sent to OTLP-compatible providers (like Jaeger or Tempo), they are transmitted as proper OpenTelemetry spans. For non-OTLP providers, the system gracefully degrades by logging trace information as structured warnings in the notification-controller logs, ensuring no alerting functionality is disrupted. This approach maintains system stability while encouraging the use of proper tracing backends.

This approach allows users to declaratively configure tracing using familiar Flux patterns, without requiring code changes to their applications or additional sidecar deployments. The notification-controller will handle the collection, correlation, and forwarding of spans to the configured tracing backend.

Example Configuration:
```yaml
# Define a tracing provider
apiVersion: notification.toolkit.fluxcd.io/v1beta1
kind: Provider
metadata:
  name: jaeger
  namespace: default
spec:
  type: jaeger
  address: http://jaeger-collector.jaeger-system.svc.cluster.local:9411 # Provider endpoint
  secretRef:
    name: jaeger-secret  # Optional authentication

---
# Configure an alert (includes the tracing out-of-the-box)
apiVersion: notification.toolkit.fluxcd.io/v1beta1
kind: Alert
metadata:
  name: webapp-tracing
  namespace: default
spec:
  providerRef: 
    name: jaeger
  eventSources:
    - kind: GitRepository  # Source controller resources
      name: webapp-source
    - kind: Kustomization  # Kustomize controller resources
      name: webapp-backend
    - kind: HelmRelease   # Helm controller resources
      name: webapp-frontend
```

Based on this configuration, the notification-controller will:
- Watch for events from the specified resources.
- Generate OpenTelemetry spans for each reconciliation event.
- Correlate spans across controllers using Flux's revision annotations.
- Forward the spans to the configured Jaeger endpoint - `Provider`.
- This implementation maintains Flux's stateless design principles while providing powerful distributed tracing capabilities that help users understand and troubleshoot their GitOps workflows.

### Alternatives
<!--
List plausible alternatives to the proposal and explain why the proposal is superior.

This is a good place to incorporate suggestions made during discussion of the RFC.
-->

## Design Details

### Trace Identity and Correlation
A key challenge in distributed tracing is establishing a reliable correlation mechanism that works across multiple controllers in a stateless, potentially unreliable environment. Our solution addresses this with a robust span identification strategy.

The root span ID is generated using a deterministic approach that combines:
- **Alert Object UID** (guaranteed unique by Kubernetes across all clusters).
- **Source's revision ID** (extracted from event payloads).

These values are concatenated and passed through a configurable checksum algorithm (SHA-256 by default). This approach ensures:
- Globally unique trace identifiers across multi-tenant and multi-cluster environments.
- Consistent trace correlation even when events arrive out of order.
- Reliable identification of the originating source event.

Example:
```yaml
# Input values
Alert UID: "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
Source Revision: "sha256:2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae"

# Concatenated value
"a1b2c3d4-e5f6-7890-abcd-ef1234567890(<Alert-UID>):sha256:2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae(<source-revision>)"

# Apply SHA-256 (default algorithm)
Root Span ID: "f7846f55cf23e14eebeab5b4e1550cad5b509e3348fbc4efa3a1413d393cb650"
```
When events occur in the system:
1. GitRepository reconciliation event with revision "sha256:2c26..." is captured by notification controller and creates root span with ID "f7846f55...".
2. Kustomization acts on the previous one, creating another event with same revision creates child span linked to "f7846f55...".
3. HelmRelease event with same revision creates another child span.
4. All spans are collected into a single trace viewable in the tracing backend.

### Resilient Span Management
The design accounts for the distributed nature of Flux controllers and potential delays/downtimes that a distributed system always implies:
- **Asynchronous Event Processing:** Since events may arrive in any order due to the distributed nature of Flux controllers, the system doesn't assume sequential processing. Each event can independently locate its parent span or create a new root span as needed.
- **Fault Tolerance:** If the notification-controller experiences downtime or latency issues, it implements a recovery mechanism:
  - When processing an event, it first attempts to locate an existing root span based on the calculated ID.
  - If found, it attaches the new span as a child to maintain the trace hierarchy.
  - If not found (due to previous failures or out-of-order processing), it automatically creates a new root span
- Span Hierarchy Maintenance: All subsequent spans related to the same revision are properly attached to their parent spans, creating a coherent trace visualization regardless of when events are processed.

This design ensures trace continuity even in challenging distributed environments while maintaining Flux's core principles of statelessness and resilience.

## Implementation History

<!--
Major milestones in the lifecycle of the RFC such as:
- The first Flux release where an initial version of the RFC was available.
- The version of Flux where the RFC graduated to general availability.
- The version of Flux where the RFC was retired or superseded.
-->
