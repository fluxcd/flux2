# Frequently asked questions

## General questions

### What does the GitOps Toolkit mean for Flux?

Flux v1 is a monolithic do-it-all operator; the GitOps Toolkit separates the functionalities into specialized controllers.

Flux v2 will be a curated configuration of the GitOps Toolkit, which you can install and operate simply using the `gotk` command. You can easily pick and choose the functionality you need and extend it to serve your own purposes.

The timeline we are looking at right now is:

1. Put Flux v1 into maintenance mode (no new features being added; bugfixes and CVEs patched only).
1. Continue work on [GitOps Toolkit roadmap](https://toolkit.fluxcd.io/roadmap/).
1. We will provide transition guides for specific user groups, e.g. users of Flux v1 in read-only mode, or of Helm Operator v1, etc. once the functionality is integrated in the GitOps Toolkit and it's deemed "ready".
1. Once the use-cases of Flux v1 are covered, we will continue supporting Flux v1 for 6 months. This will be the transition period before it's considered unsupported.

### Why did you rewrite Flux?

The GitOps Toolkit implements its functionality in individual controllers, which allowed us to address long-standing feature requests much more easily.

By basing these controllers on modern Kubernetes tooling (`controller-runtime` libraries), they can be dynamically configured with Kubernetes custom resources either by cluster admins or by other automated tools -- and you get greatly increased observability.

This gave us the opportunity to build the GitOps Toolkit with the top Flux feature requests in mind:

- Supporting multiple source Git repositories
- Operational insight through health checks, events and alerts
- Multi-tenancy capabilities, like applying each source repository with its own set of permissions

On top of that, testing the GitOps Toolkit and understanding the codebase becomes a lot easier.

### What are significant new differences between Flux v1 and the GitOps Toolkit?

#### Reconciliation

Flux v1                            | Toolkit component driven "Flux v2"
---------------------------------- | ----------------------------------
Limited to a single Git repository | Multiple Git repositories
Declarative config via arguments in the Flux deployment | `GitRepository` custom resource, which produces an artifact which can be reconciled by other controllers
Follow `HEAD` of Git branches | Supports Git branches, pinning on commits and tags, follow SemVer tag ranges
Suspending of reconciliation by downscaling Flux deployment | Reconciliation can be paused per resource by suspending the `GitRepository`
Credentials config via Arguments and/or Secret volume mounts in the Flux pod | Credentials config per `GitRepository` resource: SSH private key, HTTP/S username/password/token, OpenPGP public keys

#### `kustomize` support

Flux v1                            | Toolkit component driven "Flux v2"
---------------------------------- | ----------------------------------
Declarative config through `.flux.yaml` files in the Git repository | Declarative config through a `Kustomization` custom resource, consuming the artifact from the GitRepository
Manifests are generated via shell exec and then reconciled by `fluxd` | Generation, server-side validation, and reconciliation is handled by a specialised `kustomize-controller`
Reconciliation using the service account of the Flux deployment | Support for service account impersonation
Garbage collection needs cluster role binding for Flux to query the Kubernetes discovery API | Garbage collection needs no cluster role binding or access to Kubernetes discovery API
Support for custom commands and generators executed by fluxd in a POSIX shell | No support for custom commands

#### Helm integration

Flux v1                            | Toolkit component driven "Flux v2"
---------------------------------- | ----------------------------------
Declarative config in a single Helm custom resource | Declarative config through `HelmRepository`, `GitRepository`, `HelmChart` and `HelmRelease` custom resources
Chart synchronisation embedded in the operator | Extensive release configuration options, and a reconciliation interval per source
Support for fixed SemVer versions from Helm repositories | Support for SemVer ranges for `HelmChart` resources
Git repository synchronisation on a global interval | Planned support for charts from GitRepository sources
Limited observability via the status object of the HelmRelease resource | Better observability via the HelmRelease status object, Kubernetes events, and notifications
Resource heavy, relatively slow | Better performance
Chart changes from Git sources are determined from Git metadata | Chart changes must be accompanied by a version bump in `Chart.yaml` to produce a new artifact
Chart dependencies for charts from Git sources are downloaded by the operator | Chart dependencies must be committed to Git

#### Notifications, webhooks, observability

Flux v1                            | Toolkit component driven "Flux v2"
---------------------------------- | ----------------------------------
Emits "custom Flux events" to a webhook endpoint | Emits Kubernetes events for all custom resources part of the Toolkit
RPC endpoint can be configured to a 3rd party solution like FluxCloud to be forwarded as notifications to e.g. Slack | Toolkit components can be configured to POST the events to a `notification-controller` endpoint. Selective forwarding of POSTed events as notifications using `Provider` and `Alert` custom resources.
Webhook receiver is a side-project | Webhook receiver, handling a wide range of platforms, is included
Unstructured logging | Structured logging for all components
Custom Prometheus metrics | Generic / common `controller-runtime` Prometheus metrics

### How can I get involved?

There are a variety of ways and we look forward to having you on board building the future of GitOps together:

- [Discuss the direction](https://github.com/fluxcd/toolkit/discussions) of the GitOps Toolkit with us
- Join us in #flux-dev on the [CNCF Slack](https://slack.cncf.io)
- Check out our [contributor docs](https://toolkit.fluxcd.io/contributing/)
- Take a look at the [roadmap of the GitOps Toolkit](https://toolkit.fluxcd.io/roadmap/)

### Are there any breaking changes?

- In Flux v1 Kustomize support was implemented through `.flux.yaml` files in the Git repository. As indicated in the comparison table above, while this approach worked, we found it to be error-prone and hard to debug. The new [Kustomization CR](https://github.com/fluxcd/kustomize-controller/blob/master/docs/spec/v1alpha1/kustomization.md) should make troubleshooting much easier. Unfortunately we needed to drop the support for custom commands as running arbitrary shell scripts in-cluster poses serious security concerns.
- Helm users: we redesigned the `HelmRelease` API and the automation will work quite differently, so upgrading to `HelmRelease` v2 will require a little work from you, but you will gain more flexibility, better observability and performance.

### Is the GitOps Toolkit related to the GitOps Engine?

In an announcement in August 2019, the expectation was set that the Flux project would integrate the GitOps Engine, then being factored out of ArgoCD. Since the result would be backward-incompatible, it would require a major version bump: Flux v2.

After experimentation and considerable thought, we (the maintainers) have found a path to Flux v2 that we think better serves our vision of GitOps: the GitOps Toolkit. In consequence, we do not now plan to integrate GitOps Engine into Flux.
