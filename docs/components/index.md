# GitOps Toolkit components

The GitOps Toolkit is the set of APIs and controllers that make up the
runtime for Flux v2. The APIs comprise Kubernetes custom resources,
which can be created and updated by a cluster user, or by other
automation tooling.

You can use the toolkit to extend Flux, and to build your own systems
for continuous delivery. The [the source-watcher
guide](../dev-guides/source-watcher/) is a good place to start.

A reference for each component and API type is linked below.

- [Source Controller](source/controller.md)
    - [GitRepository CRD](source/gitrepositories.md)
    - [HelmRepository CRD](source/helmrepositories.md)
    - [HelmChart CRD](source/helmcharts.md)
    - [Bucket CRD](source/buckets.md)
- [Kustomize Controller](kustomize/controller.md)
    - [Kustomization CRD](kustomize/kustomization.md)
- [Helm Controller](helm/controller.md)
    - [HelmRelease CRD](helm/helmreleases.md)
- [Notification Controller](notification/controller.md)
    - [Provider CRD](notification/provider.md)
    - [Alert CRD](notification/alert.md)
    - [Receiver CRD](notification/receiver.md)
- [Image automation controllers](image/controller.md)
    - [ImageRepository CRD](image/imagerepositories.md)
    - [ImagePolicy CRD](image/imagepolicies.md)
    - [ImageUpdateAutomation CRD](image/imageupdateautomation.md)
