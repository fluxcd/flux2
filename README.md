# Flux version 2

[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/4782/badge)](https://bestpractices.coreinfrastructure.org/projects/4782)
[![e2e](https://github.com/fluxcd/flux2/workflows/e2e/badge.svg)](https://github.com/fluxcd/flux2/actions)
[![report](https://goreportcard.com/badge/github.com/fluxcd/flux2)](https://goreportcard.com/report/github.com/fluxcd/flux2)
[![license](https://img.shields.io/github/license/fluxcd/flux2.svg)](https://github.com/fluxcd/flux2/blob/main/LICENSE)
[![release](https://img.shields.io/github/release/fluxcd/flux2/all.svg)](https://github.com/fluxcd/flux2/releases)
[![Artifact HUB](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/flux2)](https://artifacthub.io/packages/helm/fluxcd-community/flux2)

Flux is a tool for keeping Kubernetes clusters in sync with sources of
configuration (like Git repositories), and automating updates to
configuration when there is new code to deploy.

Flux version 2 ("v2") is built from the ground up to use Kubernetes'
API extension system, and to integrate with Prometheus and other core
components of the Kubernetes ecosystem. In version 2, Flux supports
multi-tenancy and support for syncing an arbitrary number of Git
repositories, among other long-requested features.

Flux v2 is constructed with the [GitOps Toolkit](#gitops-toolkit), a
set of composable APIs and specialized tools for building Continuous
Delivery on top of Kubernetes.

Flux is a Cloud Native Computing Foundation ([CNCF](https://www.cncf.io/)) project.

## Quickstart and documentation

To get started check out this [guide](https://fluxcd.io/flux/get-started/)
on how to bootstrap Flux on Kubernetes and deploy a sample application in a GitOps manner.

For more comprehensive documentation, see the following guides:
- [Ways of structuring your repositories](https://fluxcd.io/flux/guides/repository-structure/)
- [Manage Helm Releases](https://fluxcd.io/flux/guides/helmreleases/)
- [Automate image updates to Git](https://fluxcd.io/flux/guides/image-update/)  
- [Manage Kubernetes secrets with Mozilla SOPS](https://fluxcd.io/flux/guides/mozilla-sops/)  

If you need help, please refer to our **[Support page](https://fluxcd.io/support/)**.

## GitOps Toolkit

The GitOps Toolkit is the set of APIs and controllers that make up the
runtime for Flux v2. The APIs comprise Kubernetes custom resources,
which can be created and updated by a cluster user, or by other
automation tooling.

![overview](docs/_files/gitops-toolkit.png)

You can use the toolkit to extend Flux, or to build your own systems
for continuous delivery -- see [the developer
guides](https://fluxcd.io/flux/gitops-toolkit/source-watcher/).

### Components

- [Source Controller](https://fluxcd.io/flux/components/source/)
    - [GitRepository CRD](https://fluxcd.io/flux/components/source/gitrepositories/)
    - [OCIRepository CRD](https://fluxcd.io/flux/components/source/ocirepositories/)
    - [HelmRepository CRD](https://fluxcd.io/flux/components/source/helmrepositories/)
    - [HelmChart CRD](https://fluxcd.io/flux/components/source/helmcharts/)
    - [Bucket CRD](https://fluxcd.io/flux/components/source/buckets/)
- [Kustomize Controller](https://fluxcd.io/flux/components/kustomize/)
    - [Kustomization CRD](https://fluxcd.io/flux/components/kustomize/kustomization/)
- [Helm Controller](https://fluxcd.io/flux/components/helm/)
    - [HelmRelease CRD](https://fluxcd.io/flux/components/helm/helmreleases/)
- [Notification Controller](https://fluxcd.io/flux/components/notification/)
    - [Provider CRD](https://fluxcd.io/flux/components/notification/provider/)
    - [Alert CRD](https://fluxcd.io/flux/components/notification/alert/)
    - [Receiver CRD](https://fluxcd.io/flux/components/notification/receiver/)
- [Image Automation Controllers](https://fluxcd.io/flux/components/image/)
  - [ImageRepository CRD](https://fluxcd.io/flux/components/image/imagerepositories/)
  - [ImagePolicy CRD](https://fluxcd.io/flux/components/image/imagepolicies/)
  - [ImageUpdateAutomation CRD](https://fluxcd.io/flux/components/image/imageupdateautomations/)
  
## Community

Need help or want to contribute? Please see the links below. The Flux project is always looking for
new contributors and there are a multitude of ways to get involved.

- Getting Started?
    - Look at our [Get Started guide](https://fluxcd.io/flux/get-started/) and give us feedback
- Need help?
    - First: Ask questions on our [GH Discussions page](https://github.com/fluxcd/flux2/discussions)
    - Second: Talk to us in the #flux channel on [CNCF Slack](https://slack.cncf.io/)
    - Please follow our [Support Guidelines](https://fluxcd.io/support/)
      (in short: be nice, be respectful of volunteers' time, understand that maintainers and
      contributors cannot respond to all DMs, and keep discussions in the public #flux channel as much as possible).
- Have feature proposals or want to contribute?
    - Propose features on our [GH Discussions page](https://github.com/fluxcd/flux2/discussions)
    - Join our upcoming dev meetings ([meeting access and agenda](https://docs.google.com/document/d/1l_M0om0qUEN_NNiGgpqJ2tvsF2iioHkaARDeh6b70B0/view))
    - [Join the flux-dev mailing list](https://lists.cncf.io/g/cncf-flux-dev).
    - Check out [how to contribute](CONTRIBUTING.md) to the project

### Events

Check out our **[events calendar](https://fluxcd.io/#calendar)**,
both with upcoming talks, events and meetings you can attend.
Or view the **[resources section](https://fluxcd.io/resources)**
with past events videos you can watch.

We look forward to seeing you with us!
