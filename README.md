# Flux version 2

[![e2e](https://github.com/fluxcd/flux2/workflows/e2e/badge.svg)](https://github.com/fluxcd/flux2/actions)
[![report](https://goreportcard.com/badge/github.com/fluxcd/flux2)](https://goreportcard.com/report/github.com/fluxcd/flux2)
[![license](https://img.shields.io/github/license/fluxcd/flux2.svg)](https://github.com/fluxcd/flux2/blob/main/LICENSE)
[![release](https://img.shields.io/github/release/fluxcd/flux2/all.svg)](https://github.com/fluxcd/flux2/releases)
![overview](docs/diagrams/gitops-toolkit.png)

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

## `flux` installation

With Homebrew:

```sh
brew tap fluxcd/tap
brew install flux
```

With Bash:

```sh
curl -s https://toolkit.fluxcd.io/install.sh | sudo bash

# enable completions in ~/.bash_profile
. <(flux completion bash)
```

Binaries for macOS, Windows and Linux AMD64/ARM are available to download on the
[release page](https://github.com/fluxcd/flux2/releases).

Verify that your cluster satisfies the prerequisites with:

```sh
flux check --pre
```

## Get started

To get started with Flux, start [browsing the
documentation](https://toolkit.fluxcd.io) or get started with one of
the following guides:

- [Get started with GitOps Toolkit (deep dive)](https://toolkit.fluxcd.io/get-started/)
- [Installation](https://toolkit.fluxcd.io/guides/installation/)
- [Manage Helm Releases](https://toolkit.fluxcd.io/guides/helmreleases/)
- [Setup Notifications](https://toolkit.fluxcd.io/guides/notifications/)
- [Setup Webhook Receivers](https://toolkit.fluxcd.io/guides/webhook-receivers/)

## GitOps Toolkit

The GitOps Toolkit is the set of APIs and controllers that make up the
runtime for Flux v2. The APIs comprise Kubernetes custom resources,
which can be created and updated by a cluster user, or by other
automation tooling.

You can use the toolkit to extend Flux, or to build your own systems
for continuous delivery -- see [the developer
guides](https://toolkit.fluxcd.io/dev-guides/).

### Components

- [Toolkit CLI](https://toolkit.fluxcd.io/cmd/flux/)
- [Source Controller](https://toolkit.fluxcd.io/components/source/controller/)
    - [GitRepository CRD](https://toolkit.fluxcd.io/components/source/gitrepositories/)
    - [HelmRepository CRD](https://toolkit.fluxcd.io/components/source/helmrepositories/)
    - [HelmChart CRD](https://toolkit.fluxcd.io/components/source/helmcharts/)
    - [Bucket CRD](https://toolkit.fluxcd.io/components/source/buckets/)
- [Kustomize Controller](https://toolkit.fluxcd.io/components/kustomize/controller/)
    - [Kustomization CRD](https://toolkit.fluxcd.io/components/kustomize/kustomization/)
- [Helm Controller](https://toolkit.fluxcd.io/components/helm/controller/)
    - [HelmRelease CRD](https://toolkit.fluxcd.io/components/helm/helmreleases/)
- [Notification Controller](https://toolkit.fluxcd.io/components/notification/controller/)
    - [Provider CRD](https://toolkit.fluxcd.io/components/notification/provider/)
    - [Alert CRD](https://toolkit.fluxcd.io/components/notification/alert/)
    - [Receiver CRD](https://toolkit.fluxcd.io/components/notification/receiver/)

## Community

The GitOps Toolkit is always looking for new contributors and there are a multitude of ways to get involved. Depending on what you want to do, some of the following bits might be your first steps:

- Join our upcoming dev meetings ([meeting access and agenda](https://docs.google.com/document/d/1l_M0om0qUEN_NNiGgpqJ2tvsF2iioHkaARDeh6b70B0/view))
- Talk to us in the #flux channel on [CNCF Slack](https://slack.cncf.io/)
- Join the [planning discussions](https://github.com/fluxcd/flux2/discussions)
- And if you are completely new to the GitOps Toolkit, take a look at our [Get Started guide](https://toolkit.fluxcd.io/get-started/) and give us feedback
- To be part of the conversation about Flux's development, [join the flux-dev mailing list](https://lists.cncf.io/g/cncf-flux-dev).
- Check out [how to contribute](CONTRIBUTING.md) to the project

## Featured Talks
- 19 Oct 2020 - [The Power of GitOps with Flux & GitOps Toolkit - Part 1](https://youtu.be/0v5bjysXTL8)
- 12 Oct 2020 - [Rawkode Live: Introduction to GitOps Toolkit with Stefan Prodan](https://youtu.be/HqTzuOBP0eY)
- 4 Sep 2020 - [KubeCon/CloudNativeCon Europe: The road to Flux v2 and Progressive Delivery with Stefan Prodan & Hidde Beydals](https://youtu.be/8v94nUkXsxU)
- 25 June 2020 - [Cloud Native Nordics: Introduction to GitOps & GitOps Toolkit with Alexis Richardson & Stefan Prodan](https://youtu.be/qQBtSkgl7tI)
- 7 May 2020 - [GitOps Days - Community Special: GitOps Toolkit Experimentation with Stefan Prodan](https://youtu.be/WHzxunv4DKk?t=6521)

### Upcoming Meetups
- 2 Nov 2020 - [The Power of GitOps with Flux & GitOps Toolkit - Part 2](https://www.meetup.com/GitOps-Community/events/273934676/)
- 12-13 Nov 2020 - [GitOps Days EMEA](https://www.gitopsdays.com/) with talks and workshops on migrating to Flux v2 and Helm Controller

We are looking forward to seeing you with us!
