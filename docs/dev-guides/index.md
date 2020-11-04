# GitOps Toolkit

The GitOps Toolkit is the set of APIs and controllers that make up the
runtime for Flux v2. The APIs comprise Kubernetes custom resources,
which can be created and updated by a cluster user, or by other
automation tooling.

You can use the toolkit to extend Flux, and to build your own systems
for continuous delivery. The [the source-watcher
guide](https://toolkit.fluxcd.io/dev-guides/source-watcher/) is a good
place to start.

A reference for each component is linked below.

- [Source Controller](../components/source/controller.md)
    - [GitRepository CRD](../components/source/gitrepositories.md)
    - [HelmRepository CRD](../components/source/helmrepositories.md)
    - [HelmChart CRD](../components/source/helmcharts.md)
    - [Bucket CRD](../components/source/buckets.md)
- [Kustomize Controller](../components/kustomize/controller.md)
    - [Kustomization CRD](../components/kustomize/kustomization.md)
- [Helm Controller](../components/helm/controller.md)
    - [HelmRelease CRD](../components/helm/helmreleases.md)
- [Notification Controller](../components/notification/controller.md)
    - [Provider CRD](../components/notification/provider.md)
    - [Alert CRD](../components/notification/alert.md)
    - [Receiver CRD](../components/notification/receiver.md)
