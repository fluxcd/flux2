# GitOps Toolkit

[![e2e](https://github.com/fluxcd/toolkit/workflows/e2e/badge.svg)](https://github.com/fluxcd/toolkit/actions)
[![report](https://goreportcard.com/badge/github.com/fluxcd/toolkit)](https://goreportcard.com/report/github.com/fluxcd/toolkit)
[![license](https://img.shields.io/github/license/fluxcd/toolkit.svg)](https://github.com/fluxcd/toolkit/blob/main/LICENSE)
[![release](https://img.shields.io/github/release/fluxcd/toolkit/all.svg)](https://github.com/fluxcd/toolkit/releases)
![overview](docs/diagrams/gotk-feature.png)

The GitOps Toolkit is a set of composable APIs and specialized tools
that can be used to build a Continuous Delivery platform on top of Kubernetes.

These tools are build with Kubernetes controller-runtime libraries, and they
can be dynamically configured with Kubernetes custom resources either by
cluster admins or by other automated tools.
The GitOps Toolkit components interact with each other via Kubernetes
events and are responsible for the reconciliation of their designated API objects.

## `gotk` installation

With Homebrew:

```sh
brew tap fluxcd/tap
brew install gotk
```

With Bash:

```sh
curl -s https://toolkit.fluxcd.io/install.sh | sudo bash

# enable completions in ~/.bash_profile
. <(gotk completion bash)
```

Binaries for macOS and Linux AMD64/ARM64 are available to download on the
[release page](https://github.com/fluxcd/toolkit/releases).

Verify that your cluster satisfies the prerequisites with:

```sh
gotk check --pre
```

## Get started

To get started with the GitOps Toolkit, start [browsing the documentation](https://toolkit.fluxcd.io)
or get started with one of the following guides:

- [Get started with GitOps Toolkit (deep dive)](https://toolkit.fluxcd.io/get-started/)
- [Installation](https://toolkit.fluxcd.io/guides/installation/)
- [Manage Helm Releases](https://toolkit.fluxcd.io/guides/helmreleases/)
- [Setup Notifications](https://toolkit.fluxcd.io/guides/notifications/)
- [Setup Webhook Receivers](https://toolkit.fluxcd.io/guides/webhook-receivers/)

## Components

- [Toolkit CLI](https://toolkit.fluxcd.io/cmd/gotk/)
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
- Join the [planning discussions](https://github.com/fluxcd/toolkit/discussions)
- And if you are completely new to the GitOps Toolkit, take a look at our [Get Started guide](https://toolkit.fluxcd.io/get-started/) and give us feedback
- To be part of the conversation about Flux's development, [join the flux-dev mailing list](https://lists.cncf.io/g/cncf-flux-dev).
- Check out [how to contribute](CONTRIBUTING.md) to the project

## Featured Talks
- [4 Sep 2020 - KubeCon/CloudNativeCon Europe: Flux Deep Dive: The road to "Flux v2" and Progressive Delivery with Stefan Prodan & Hidde Beydals](https://youtu.be/8v94nUkXsxU)
- [25 June 2020 - Cloud Native Nordics Tech Talk “Introduction to GitOps & GitOps Toolkit” with Alexis Richardson & Stefan Prodan](https://youtu.be/qQBtSkgl7tI)
- [7 May 2020 - GitOps Days - Community Special: GitOps Toolkit Experimentation](https://youtu.be/WHzxunv4DKk?t=6521)

### Upcoming Meetups
- [19 October 2020 - GitOps Toolkit Guide Walk-through](https://www.meetup.com/GitOps-Community/events/273640196/)
Join us 10am PT / 18:00 BST) for to this special walk-through of the GitOps Toolkit! 
Through this talk you'll be able to see how the upcoming Flux v2 and GitOps Toolkit will bring
great improvements to the Flux that you love! Watch or follow along as Leigh Capili shares some
highlights and then goes through Getting Started with GitOps Toolkit.
- 2 November 2020 - GitOps Toolkit Guide Walk-through - Part 2 (TBD)

We are looking forward to seeing you with us!
