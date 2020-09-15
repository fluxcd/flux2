# GitOps Toolkit

[![e2e](https://github.com/fluxcd/toolkit/workflows/e2e/badge.svg)](https://github.com/fluxcd/toolkit/actions)
[![report](https://goreportcard.com/badge/github.com/fluxcd/toolkit)](https://goreportcard.com/report/github.com/fluxcd/toolkit)
[![license](https://img.shields.io/github/license/fluxcd/toolkit.svg)](https://github.com/fluxcd/toolkit/blob/master/LICENSE)
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
. <(gotk completion)
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
- [Kustomize Controller](https://toolkit.fluxcd.io/components/kustomize/controller/)
    - [Kustomization CRD](https://toolkit.fluxcd.io/components/kustomize/kustomization/)
- [Helm Controller](https://toolkit.fluxcd.io/components/helm/controller/)
    - [HelmRelease CRD](https://toolkit.fluxcd.io/components/helm/helmreleases/)
- [Notification Controller](https://toolkit.fluxcd.io/components/notification/controller/)
    - [Provider CRD](https://toolkit.fluxcd.io/components/notification/provider/)
    - [Alert CRD](https://toolkit.fluxcd.io/components/notification/alert/)
    - [Receiver CRD](https://toolkit.fluxcd.io/components/notification/receiver/)
