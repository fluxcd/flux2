# Flux v2

Flux is a tool for keeping Kubernetes clusters in sync with sources of
configuration (like Git repositories), and automating updates to
configuration when there is new code to deploy.

Flux version 2 ("Flux v2") is built from the ground up to use Kubernetes'
API extension system, and to integrate with Prometheus and other core
components of the Kubernetes ecosystem. In version 2, Flux supports
multi-tenancy and support for syncing an arbitrary number of Git
repositories, among other long-requested features.

Flux v2 is constructed with the [GitOps Toolkit](components/index.md),
a set of composable APIs and specialized tools for building Continuous
Delivery on top of Kubernetes.

## Who is Flux for?

Flux helps

- **cluster operators** who automate provision and configuration of clusters;
- **platform engineers** who build continuous delivery for developer teams;
- **app developers** who rely on continuous delivery to get their code live.

The [GitOps Toolkit](components/index.md) is for **platform
engineers** who want to make their own continuous delivery system, and
have requirements not covered by Flux.

## What can I do with Flux?

Flux is based on a set of Kubernetes API extensions ("custom
resources"), which control how git repositories and other sources of
configuration are applied into the cluster ("synced").
For example, you create a `GitRepository` object to mirror
configuration from a Git repository, then a `Kustomization` object to
sync that configuration.

Flux works with Kubernetes' role-based access control (RBAC), so you
can lock down what any particular sync can change. It can send
notifications to Slack and other like systems when configuration is
synced and ready, and receive webhooks to tell it when to sync.

The `flux` command-line tool is a convenient way to bootstrap the
system in a cluster, and to access the custom resources that make up
the API.

![overview](diagrams/gitops-toolkit.png)

## Where do I start?

!!!hint "Get started with Flux v2!"
    Following this [guide](get-started/index.md) will just take a couple of minutes to complete:
    After installing the `flux` CLI and running a couple of very simple commands,
    you will have a GitOps workflow setup which involves a staging and a production cluster.

## More detail on what's in Flux

Features:

- Source configuration from Git and Helm repositories, and
  S3-compatible buckets (e.g., Minio)
- Kustomize and Helm support
- Event-triggered and periodic reconciliation
- Integration with Kubernetes RBAC
- Health assessment (clusters and workloads)
- Dependency management (infrastructure and workloads)
- Alerting to external systems (webhook senders)
- External events handling (webhook receivers)
- Configuration update automation (automated patching)
- Policy-driven validation (OPA, admission controllers)
- Seamless integration with Git providers (GitHub, GitLab, Bitbucket)
- Interoperability with workflow providers (GitHub Actions, Tekton, Argo)
- Interoperability with Cluster API (CAPI) providers

## Community

The Flux project is always looking for new contributors and there are a multitude of ways to get involved.
Depending on what you want to do, some of the following bits might be your first steps:

- Join our upcoming dev meetings ([meeting access and agenda](https://docs.google.com/document/d/1l_M0om0qUEN_NNiGgpqJ2tvsF2iioHkaARDeh6b70B0/view))
- Ask questions and add suggestions in our [GitOps Toolkit Discussions](https://github.com/fluxcd/toolkit/discussions)
- Talk to us in the #flux channel on [CNCF Slack](https://slack.cncf.io/)
- Join the [planning discussions](https://github.com/fluxcd/flux2/discussions)
- And if you are completely new to Flux v2 and the GitOps Toolkit, take a look at our [Get Started guide](get-started/index.md) and give us feedback
- Check out [how to contribute](contributing/index.md) to the project

### Upcoming Events
- 25 Jan 2021 - [GitOps Core Concepts & How to Teach Your Teams with Leigh Capili](https://www.meetup.com/GitOps-Community/events/275625806/)

### Featured Talks
- 11 Jan 2021 - [Helm + GitOps = ⚡️⚡️⚡️ with Scott Rigby](https://youtu.be/YG8jMFrYQvM)
- 14 Dec 2020 - [The Power of GitOps with Flux and Flagger (GitOps Hands-On) with Leigh Capili](https://youtu.be/cB7iXeNLteE)
- 30 Nov 2020 - [The Power of GItOps with Flux 2 - Part 3 with Leigh Capili](https://youtu.be/N_K5g7o9JKg)
- 24 Nov 2020 - [Flux CD v2 with GitOps Toolkit - Kubernetes Deployment and Sync Mechanism](https://youtu.be/R6OeIgb7lUI)
- 02 Nov 2020 - [The Power of GitOps with Flux & GitOps Toolkit - Part 2 with Leigh Capili](https://youtu.be/fC2YCxQRUwU)
- 28 Oct 2020 - [The Kubelist Podcast: Flux with Michael Bridgen](https://www.heavybit.com/library/podcasts/the-kubelist-podcast/ep-5-flux-with-michael-bridgen-of-weaveworks/)
- 19 Oct 2020 - [The Power of GitOps with Flux & GitOps Toolkit - Part 1 with Leigh Capili](https://youtu.be/0v5bjysXTL8)
- 12 Oct 2020 - [Rawkode Live: Introduction to GitOps Toolkit with Stefan Prodan](https://youtu.be/HqTzuOBP0eY)
- 04 Sep 2020 - [KubeCon Europe: The road to Flux v2 and Progressive Delivery with Stefan Prodan & Hidde Beydals](https://youtu.be/8v94nUkXsxU)
- 25 June 2020 - [Cloud Native Nordics: Introduction to GitOps & GitOps Toolkit with Alexis Richardson & Stefan Prodan](https://youtu.be/qQBtSkgl7tI)
- 07 May 2020 - [GitOps Days - Community Special: GitOps Toolkit Experimentation with Stefan Prodan](https://youtu.be/WHzxunv4DKk?t=6521)

We look forward to seeing you with us!
