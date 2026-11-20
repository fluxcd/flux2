# RFC-NNNN Root causes for failed health checks

**Status:** provisional

**Creation date:** 2026-11-16

**Last update:** 2026-11-16

## Summary

When a health check of a Kustomization or HelmRelease fails, the controller inspects the
workloads behind the failed objects, plus their pods and warning events, and adds a short,
classified root cause to the `Ready` condition message and the failure event. This is
opt-in via a feature gate, bounded in API calls and output size, and sanitized.

## Motivation

The health check summary tells users *which* object isn't ready (and, with
[fluxcd/pkg#1380](https://github.com/fluxcd/pkg/pull/1380), its kstatus message). It doesn't
tell them *why*: the reasons sit on pods and events that Flux doesn't look at. Users without
cluster access, typically application teams on a shared platform, get no actionable feedback
from Flux notifications. Platform teams build side tooling or answer by hand.

Discussion: [fluxcd/flux2#6186](https://github.com/fluxcd/flux2/discussions/6186).

### Goals

- A one-line, classified root cause for the most common workload failures, in the condition
  message and as event metadata, available to all notification providers.
- Bounded cost: work only happens on failure, with a fixed number of list calls per failed
  check.
- No new CRDs and no new API fields required; opt-in via feature gate.

### Non-Goals

- A general troubleshooting or observability tool. Logs, metrics and traces are out of scope.
- Application-level failures that Kubernetes doesn't surface (wrong config with healthy
  pods).
- Changing how readiness is determined (kstatus stays the source of truth).

## Proposal

When `WaitForSet` fails (timeout, or failed early because of stalled objects), the controller
does the following for up to *N* (default 5) failed objects:

1. **Resolve workloads.** Deployments, StatefulSets, DaemonSets and Jobs directly; for a
   HelmRelease, the workloads of its release via the `meta.helm.sh/release-name` and
   `meta.helm.sh/release-namespace` annotations, plus hook Jobs.
2. **Collect.** The workload conditions, their pods (by selector, grouped by revision, see
   *Current revision first* below) and the namespace's warning
   events since the start of the failed attempt. For objects that are gone (hooks,
   rolled-back pods), events outlive the object for about an hour.
3. **Classify and choose** the most specific finding of the current revision:

   | Priority | Finding | Source |
   |---|---|---|
   | 10 | `ImagePullBackOff`/`ErrImagePull`, `CreateContainerConfigError`, `InvalidImageName` | container `waiting` state, `Failed` events |
   | 20 | `CrashLoopBackOff` with the last exit code or signal (`OOMKilled`), failed init containers | container `lastState.terminated` |
   | 25 | `FailedMount`, `FailedAttachVolume`, `FailedCreatePodSandBox` | pod events |
   | 30 | `Unschedulable` | `PodScheduled=False`, `FailedScheduling` |
   | 35 | `FailedCreate` (quota, admission, Pod Security), `BackoffLimitExceeded` | workload conditions and events |
   | 40 | `Evicted` and other failed pods | pod status |
   | 50 | readiness probe failures (`Unhealthy`) | events, sanitized |
   | 60 | running but not ready | pod status |
   | 70 | `ProgressDeadlineExceeded` | Deployment condition |

4. **Report.** Append `; root cause: <Kind>/<ns>/<name>: <Reason>: <detail>` to the health
   check error. Set event metadata keys (for example `healthcheck.rootcause.reason`,
   `.object` and `.detail`), so notification-controller and CEL expressions can use them
   without parsing.

### User Stories

- *As an application developer without cluster access,* I see on my commit or in Slack that
  the image tag I pushed doesn't exist, without asking the platform team.
- *As a platform engineer,* I stop maintaining a side component that reconstructs this
  information.
- *As a multi-tenant cluster admin,* I can enable this without giving tenants more access,
  because the controller reads with the Kustomization's service account where one is
  configured.

### Alternatives

- **External relay (our current workaround, reference implementation).** A Deployment
  watches Kustomizations and writes the root cause into extra commit statuses. It works (it
  runs on the pre-production and production clusters of our platform), but it's tied to one
  notification channel, needs its own RBAC and a GitHub token, and duplicates controller
  logic.
- **kstatus message only** ([fluxcd/pkg#1380](https://github.com/fluxcd/pkg/pull/1380)). It's
  small and already useful, but it stops at the object level; pods and events aren't covered.
- **Message in the commit status only**
  ([fluxcd/notification-controller#1436](https://github.com/fluxcd/notification-controller/issues/1436)).
  It gets the event message to the commit, but the message itself still lacks the cause.
- **UIs** (Flux Operator UI, Headlamp, ...). They need cluster or UI access, the thing the
  affected users lack, and don't reach notifications.
- **eBPF/runtime tools** (e.g. Inspektor Gadget). They observe runtime behaviour, not
  Kubernetes scheduling and image decisions; it's a heavy dependency for this problem.
- **CI pipelines running `kubectl events`.** That needs cluster credentials in CI, which is
  what platforms avoid.

## Design Details

- **Feature gate.** `--feature-gates=HealthCheckRootCause=true` on kustomize-controller (and
  later helm-controller, for its own `wait`); alpha, default off.
- **Current revision first.** During a rollout, a workload's selector matches the pods of
  both the old and the new revision. The current revision is, for a Deployment, its newest
  ReplicaSet (the one whose `deployment.kubernetes.io/revision` matches the Deployment's,
  matched to pods via `pod-template-hash`); for a StatefulSet, the pods whose
  `controller-revision-hash` equals `status.updateRevision`; for a DaemonSet, its newest
  ControllerRevision. The priority order applies within the current revision. Findings from
  older revisions are used only when the current one has none, and are labelled, for example
  `Deployment/apps/podinfo: ImagePullBackOff (previous revision): …`. Test case: the old
  ReplicaSet has a pod in `ImagePullBackOff` and the new one a pod in `FailedScheduling`;
  the reported cause is the scheduler message of the new revision.
- **RBAC.** `get`/`list` on `pods` and `events` (core and `events.k8s.io`), on workload
  kinds, and on `replicasets` and `controllerrevisions` to find the current revision. Reads
  use the impersonated service account (`spec.serviceAccountName`) where
  configured. On `Forbidden`, the cause is reported as unavailable, and the health check
  error stays as today.
- **Cost.** Per failed check: one list of pods per workload selector, and one list of events
  and of ReplicaSets or ControllerRevisions per namespace, capped at *N* objects. Nothing runs
  on success.
- **Output limits.** Each detail at most 256 runes, shortened in the middle; one root cause
  per object, at most *N*.
- **Sanitization.** Probe output may contain response bodies. Keep only the probe type,
  status code and the error class. Image references stay; registry credentials never appear
  in events.

### Enabling the feature

Set the feature gate on the controller.

### What happens when the feature is enabled

Failed health checks get a root-cause suffix in the `Ready`/`Healthy` message and event
metadata keys. Successful reconciliations are unchanged.

### What happens when the feature is disabled after being enabled

Messages go back to the current format with the next reconciliation; no stored state.

### How can it be determined whether the feature is in use

The controller logs the feature gates at startup; messages contain `root cause:`.

### Drawbacks

- More API reads on failure, and wider read RBAC.
- Longer condition messages; mitigated by the caps.
- Information from the cluster (image names, node pressure) reaches external notification
  systems. That's the purpose; the opt-in and sanitization keep it deliberate.

## Implementation History

- 2026-11: reference implementation as an external relay (about 1,100 lines of
  classification logic plus tests), running on pre-production and production clusters.