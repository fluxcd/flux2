# Flux v2

Flux is a tool for keeping Kubernetes clusters in sync with sources of
configuration (like Git repositories), and automating updates to
configuration when there is new code to deploy.

Flux version 2 ("Flux v2") is built from the ground up to use Kubernetes'
API extension system, and to integrate with Prometheus and other core
components of the Kubernetes ecosystem. In version 2, Flux supports
multi-tenancy and support for syncing an arbitrary number of Git
repositories, among other long-requested features.

Flux v2 is constructed with the [GitOps Toolkit](#gitops-toolkit), a
set of composable APIs and specialized tools for building Continuous
Delivery on top of Kubernetes.

## Who is Flux for?

Flux helps

- **cluster operators** who automate provision and configuration of clusters;
- **platform engineers** who build continuous delivery for developer teams;
- **app developers** who rely on continuous delivery to get their code live.

The [GitOps Toolkit](#gitops-toolkit) is for **platform engineers**
who want to make their own continuous delivery system, and have
requirements not covered by Flux.

Features:

- Source management (Git and Helm repositories, S3 compatible buckets)
- Kustomize and Helm support
- Event-based and on-a-schedule reconciliation
- Role-based reconciliation (multi-tenancy)
- Health assessment (clusters and workloads)
- Dependency management (infra and workloads)
- Alerting to external systems (webhook senders)
- External events handling (webhook receivers)
- Source write-back (automated patching)
- Policy driven validation (OPA, admission controllers)
- Seamless integration with Git providers (GitHub, GitLab, Bitbucket)
- Interoperability with workflow providers (GitHub Actions, Tekton, Argo)
- Interoperability with CAPI providers

## GitOps Toolkit

The GitOps Toolkit is the set of APIs and controllers that make up the
runtime for Flux v2. The APIs comprise Kubernetes custom resources,
which can be created and updated by a cluster user, or by other
automation tooling.

![overview](diagrams/gitops-toolkit.png)

You can use the toolkit to extend Flux, or to build your own systems
for continuous delivery -- see [the developer
guides](https://toolkit.fluxcd.io/dev-guides/source-watcher/).

Components:

- [Source Controller](components/source/controller.md)
    - [GitRepository CRD](components/source/gitrepositories.md)
    - [HelmRepository CRD](components/source/helmrepositories.md)
    - [HelmChart CRD](components/source/helmcharts.md)
    - [Bucket CRD](components/source/buckets.md)
- [Kustomize Controller](components/kustomize/controller.md)
    - [Kustomization CRD](components/kustomize/kustomization.md)
- [Helm Controller](components/helm/controller.md)
    - [HelmRelease CRD](components/helm/helmreleases.md)
- [Notification Controller](components/notification/controller.md)
    - [Provider CRD](components/notification/provider.md)
    - [Alert CRD](components/notification/alert.md)
    - [Receiver CRD](components/notification/receiver.md)

## Get Started

!!!hint "Get started with Flux v2!"
    Following this [guide](get-started/index.md) will just take a couple of minutes to complete:
    After installing the `flux` CLI and running a couple of very simple commands,
    you will have a GitOps workflow setup which involves a staging and a production cluster.

## Community

The Flux project is always looking for new contributors and there are a multitude of ways to get involved.
Depending on what you want to do, some of the following bits might be your first steps:

- Join our upcoming dev meetings ([meeting access and agenda](https://docs.google.com/document/d/1l_M0om0qUEN_NNiGgpqJ2tvsF2iioHkaARDeh6b70B0/view))
- Ask questions and add suggestions in our [GitOps Toolkit Discussions](https://github.com/fluxcd/toolkit/discussions)
- Talk to us in the #flux channel on [CNCF Slack](https://slack.cncf.io/)
- Join the [planning discussions](https://github.com/fluxcd/flux2/discussions)
- And if you are completely new to Flux v2 and the GitOps Toolkit, take a look at our [Get Started guide](get-started/index.md) and give us feedback
- Check out [how to contribute](contributing/index.md) to the project

### Featured Talks

- 19 Oct 2020 - [The Power of GitOps with Flux & GitOps Toolkit - Part 2 with Leigh Capili](https://youtu.be/fC2YCxQRUwU)
- 28 Oct 2020 - [The Kubelist Podcast: Flux with Michael Bridgen](https://www.heavybit.com/library/podcasts/the-kubelist-podcast/ep-5-flux-with-michael-bridgen-of-weaveworks/)
- 19 Oct 2020 - [The Power of GitOps with Flux & GitOps Toolkit - Part 1 with Leigh Capili](https://youtu.be/0v5bjysXTL8)
- 12 Oct 2020 - [Rawkode Live: Introduction to GitOps Toolkit with Stefan Prodan](https://youtu.be/HqTzuOBP0eY)
- 4 Sep 2020 - [KubeCon Europe: The road to Flux v2 and Progressive Delivery with Stefan Prodan & Hidde Beydals](https://youtu.be/8v94nUkXsxU)
- 25 June 2020 - [Cloud Native Nordics: Introduction to GitOps & GitOps Toolkit with Alexis Richardson & Stefan Prodan](https://youtu.be/qQBtSkgl7tI)
- 7 May 2020 - [GitOps Days - Community Special: GitOps Toolkit Experimentation with Stefan Prodan](https://youtu.be/WHzxunv4DKk?t=6521)

### Upcoming Events

- 12-13 Nov 2020 - [GitOps Days EMEA](https://www.gitopsdays.com/) with talks and workshops on migrating to Flux v2 and Helm Controller
- 19 Nov 2020 - [KubeCon NA: Progressive Delivery Techniques with Flagger and Flux v2 with Stefan Prodan](https://kccncna20.sched.com/event/1b04f8408b49976b843a5d0019cb8112)

We are looking forward to seeing you with us!
