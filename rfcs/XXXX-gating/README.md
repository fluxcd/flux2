# RFC-XXXX Gating Flux reconciliation

**Status:** provisional

**Creation date:** 2022-09-28

**Last update:** 2022-09-28

## Summary

Flux should offer a mechanism for cluster admins and other teams involved in the release process
to manually approve the rollout of changes onto clusters. In addition, Flux should offer 
a way to define maintenance time windows and other time-based gates, to allow a better control 
of applications and infrastructure changes to critical system.

## Motivation

Flux watches sources (e.g. GitRepositories, OCIRepositories, HelmRepositories, S3-compatible Buckets) and
automatically reconciles the changes onto clusters as described with Flux Kustomizations and HelmReleases.
The teams involved in the delivery process (e.g. dev, qa, sre) can decide when changes are delivered
to production by reviewing and approving the proposed changes in a collaborative manner with pull request.
Once a pull request is merged onto a branch that defines the desired state of the production system,
Flux kicks off the reconciliation process.

There are situations when users want to have a gating mechanism after the desired state changes are merged in Git:

- Manual approval of container image updates (e.g. https://github.com/fluxcd/flux2/discussions/870)
- Manual approval of infrastructure upgrades (e.g. https://github.com/fluxcd/flux2/issues/959)
- Maintenance window (e.g. https://github.com/fluxcd/flux2/discussions/1004)
- Planned releases
- No Deploy Friday

### Goals

- Offer a dedicated API for defining time-based gates in a declarative manner.
- Introduce a `gating-controller` in the Flux suite that manages the `Gate` objects.
- Extend the current Flux APIs and controllers to support gating.

### Non-Goals

<!--
What is out of scope for this RFC? Listing non-goals helps to focus discussion
and make progress.
-->

## Proposal

In order to support manual gating, Flux could be extended with a dedicated API and controller
that would allow users to define `Gate` objects and perform operations like `open` and `close`.

A `Gate` object could be referenced in sources (Buckets, Git, Helm, OCI Repositories)
and syncs (Kustomizations, HelmReleases, ImageUpdateAutomation)
to block the reconciliation until the gate is opened.

A `Gate` can be opened or closed by annotating the object with a timestamp or by
calling a specific webhook receiver exposed by notification-controller.

A `Gate` can be configured to automatically close or open based on a time window defined in the `Gate` spec.

The `Gate` API would replace Flagger's current
[manual gating mechanism](https://docs.flagger.app/usage/webhooks#manual-gating).

### User Stories

> As a member of the SRE team, I want to allow deployments to happen only
> in a particular time frame of my own choosing.

Define a gate that automatically closes after 1h from the time it has been opened:

```yaml
apiVersion: gating.toolkit.fluxcd.io/v1alpha1
kind: Gate
metadata:
  name: sre-approval
  namespace: flux-system
spec:
  interval: 30s
  default: closed
  window: 1h
```

When the gate is created in-cluster, the `gating-controller` uses `spec.default` to set the `Opened` condition:

```yaml
apiVersion: gating.toolkit.fluxcd.io/v1alpha1
kind: Gate
metadata:
  name: sre-approval
  namespace: flux-system
status:
  conditions:
    - lastTransitionTime: "2021-03-26T10:09:26Z"
      message: "Gate closed by default"
      reason: ReconciliationSucceeded
      status: "False"
      type: Opened
```

While the gate is closed, all the objects that reference it will wait for an approval:

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1beta1
kind: Kustomization
metadata:
  name: my-app
  namespace: flux-system
spec:
  gates:
    - name: sre-approval
    - name: qa-approval
status:
  conditions:
    - lastTransitionTime: "2021-03-26T10:09:26Z"
      message: "Reconciliation is waiting approval, gate 'flux-system/sre-approval' is closed."
      reason: GateClosed
      status: "False"
      type: Approved
```

The SRE team can open the gate either by annotating the gate or by calling the notification-controller webhook:

```sh
kubectl -n flux-system annotate --overwrite gate/sre-approval \
open.gate.fluxcd.io/requestedAt="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
```

The `gating-controller` extracts the ISO8601 date from the `open.gate` annotation value,
sets the `requestedAt` & `resetToDefaultAt`, and opens the gate for the specified window:

```yaml
apiVersion: gating.toolkit.fluxcd.io/v1alpha1
kind: Gate
metadata:
  name: sre-approval
  namespace: flux-system
status:
  requestedAt: "2021-03-26T10:00:00Z"
  resetToDefaultAt: "2021-03-26T11:00:00Z"
  conditions:
    - lastTransitionTime: "2021-03-26T10:00:00Z"
      message: "Gate scheduled for closing at 2021-03-26T11:00:00Z"
      reason: ReconciliationSucceeded
      status: "True"
      type: Opened
```

While the gate is opened, all the objects that reference it are approved to reconcile at their configured interval.

The SRE can decide to close the gate ahead of its schedule with:

```sh
kubectl -n flux-system annotate --overwrite gate/sre-approval \
close.gate.fluxcd.io/requestedAt="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
```

The `gating-controller` extracts the ISO8601 date from the `close.gate` annotation value,
compares it with the `open.gate` & `requestedAt` date and closes the gate:

```yaml
apiVersion: gating.toolkit.fluxcd.io/v1alpha1
kind: Gate
metadata:
  name: sre-approval
  namespace: flux-system
status:
  requestedAt: "2021-03-26T10:10:00Z"
  resetToDefaultAt: "2021-03-26T10:10:00Z"
  conditions:
    - lastTransitionTime: "2021-03-26T10:10:00Z"
      message: "Gate close requested"
      reason: ReconciliationSucceeded
      status: "False"
      type: Opened
```

The objects that are referencing this gate, will finish their ongoing reconciliation (if any) then pause.

> As a member of the SRE team, I want to block deployments in a particular time window.

To enforce a maintenance window of 24 hours, you can define a `Gate` that's opened by default:

```yaml
apiVersion: gating.toolkit.fluxcd.io/v1alpha1
kind: Gate
metadata:
  name: maintenance
  namespace: flux-system
spec:
  interval: 30s
  default: opened
  window: 24h
```

To start the maintenance window you can annotate the gate with:

```sh
kubectl -n flux-system annotate --overwrite gate/maintenance \
close.gate.fluxcd.io/requestedAt="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
```

The `gating-controller` extracts the ISO8601 date from the `close.gate`
annotation value and closes the gate for the specified window:

```yaml
apiVersion: gating.toolkit.fluxcd.io/v1alpha1
kind: Gate
metadata:
  name: maintenance
  namespace: flux-system
status:
  requestedAt: "2021-03-26T10:00:00Z"
  resetToDefaultAt: "2021-03-27T10:00:00Z"
  conditions:
    - lastTransitionTime: "2021-03-26T10:00:00Z"
      message: "Gate scheduled for opening at 2021-03-27T11:00:00Z"
      reason: ReconciliationSucceeded
      status: "False"
      type: Opened
```

You could also schedule "No Deploy Fridays" with a CronJob that closes the `maintenance` gate at `0 0 * * FRI`.

### Alternatives

<!--
List plausible alternatives to the proposal and explain why the proposal is superior.

This is a good place to incorporate suggestions made during discussion of the RFC.
-->

## Design Details

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

## Implementation History

<!--
Major milestones in the lifecycle of the RFC such as:
- The first Flux release where an initial version of the RFC was available.
- The version of Flux where the RFC graduated to general availability.
- The version of Flux where the RFC was retired or superseded.
-->
