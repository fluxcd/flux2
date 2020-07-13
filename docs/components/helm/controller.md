# Helm Controller

The Helm Controller is a Kubernetes operator, allowing one to declaratively manage Helm chart
releases with Kubernetes manifests.

![](../../_files/helm-controller.png)

The desired state of a Helm release is described through a Kubernetes Custom Resource named `HelmRelease`.
Based on the creation, mutation or removal of a HelmRelease resource in the cluster,
Helm actions are performed by the controller.

Features:

- Watches for `HelmRelease` objects and generates `HelmChart` objects
- Fetches artifacts produced by [source-controller](../source/controller.md) from `HelmChart` objects
- Watches `HelmChart` objects for revision changes (semver ranges)
- Performs Helm v3 actions including Helm tests as configured in the `HelmRelease` objects
- Runs Helm install/upgrade in a specific order, taking into account the depends-on relationship
- Prunes Helm releases removed from cluster (garbage collection) 
- Reports Helm releases status (alerting provided by [notification-controller](../notification/controller.md))

Links:

- Source code [fluxcd/helm-controller](https://github.com/fluxcd/helm-controller)
- Specification [docs](https://github.com/fluxcd/helm-controller/tree/master/docs/spec)
