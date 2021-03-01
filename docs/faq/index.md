# Frequently asked questions

## Kustomize questions

### Are there two Kustomization types?

Yes, the `kustomization.kustomize.toolkit.fluxcd.io` is a Kubernetes
[custom resource](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
while `kustomization.kustomize.config.k8s.io` is the type used to configure a
[Kustomize overlay](https://kubectl.docs.kubernetes.io/references/kustomize/).

The `kustomization.kustomize.toolkit.fluxcd.io` object refers to a `kustomization.yaml`
file path inside a Git repository or Bucket source.

### How do I use them together?

Assuming an app repository with `./deploy/prod/kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - deployment.yaml
  - service.yaml
  - ingress.yaml
```

Define a source of type `gitrepository.source.toolkit.fluxcd.io`
that pulls changes from the app repository every 5 minutes inside the cluster:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: GitRepository
metadata:
  name: my-app
  namespace: default
spec:
  interval: 5m
  url: https://github.com/my-org/my-app
  ref:
    branch: main
```

Then define a `kustomization.kustomize.toolkit.fluxcd.io` that uses the `kustomization.yaml`
from `./deploy/prod` to determine which resources to create, update or delete:

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1beta1
kind: Kustomization
metadata:
  name: my-app
  namespace: default
spec:
  interval: 15m
  path: "./deploy/prod"
  prune: true
  sourceRef:
    kind: GitRepository
    name: my-app
```

### What is a Kustomization reconciliation?

In the above example, we pull changes from Git every 5 minutes,
and a new commit will trigger a reconciliation of
all the `Kustomization` objects using that source.

Depending on your configuration, a reconciliation can mean:

* generating a kustomization.yaml file in the specified path
* building the kustomize overlay
* decrypting secrets
* validating the manifests with client or server-side dry-run
* applying changes on the cluster
* health checking of deployed workloads
* garbage collection of resources removed from Git
* issuing events about the reconciliation result
* recoding metrics about the reconciliation process 

The 15 minutes reconciliation interval, is the interval at which you want to undo manual changes
.e.g. `kubectl set image deployment/my-app` by reapplying the latest commit on the cluster.

Note that a reconciliation will override all fields of a Kubernetes object, that diverge from Git.
For example, you'll have to omit the `spec.replicas` field from your `Deployments` YAMLs if you 
are using a `HorizontalPodAutoscaler` that changes the replicas in-cluster.

### Can I use repositories with plain YAMLs?

Yes, you can specify the path where the Kubernetes manifests are,
and kustomize-controller will generate a `kustomization.yaml` if one doesn't exist.

Assuming an app repository with the following structure:

```
├── deploy
│   └── prod
│       ├── .yamllint.yaml
│       ├── deployment.yaml
│       ├── service.yaml
│       └── ingress.yaml
└── src
```

Create a `GitRepository` definition and exclude all the files that are not Kubernetes manifests:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: GitRepository
metadata:
  name: my-app
  namespace: default
spec:
  interval: 5m
  url: https://github.com/my-org/my-app
  ref:
    branch: main
  ignore: |
    # exclude all
    /*
    # include deploy dir
    !/deploy
    # exclude non-Kubernetes YAMLs
    /deploy/**/.yamllint.yaml
```

Then create a `Kustomization` definition to reconcile the `./deploy/prod` dir:

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1beta1
kind: Kustomization
metadata:
  name: my-app
  namespace: default
spec:
  interval: 15m
  path: "./deploy/prod"
  prune: true
  sourceRef:
    kind: GitRepository
    name: my-app
```

With the above configuration, source-controller will pull the Kubernetes manifests
from the app repository and kustomize-controller will generate a
`kustomization.yaml` including all the resources found with `./deploy/prod/**/*.yaml`.

The kustomize-controller creates `kustomization.yaml` files similar to:

```sh
cd ./deploy/prod && kustomize create --autodetect --recursive
```

### What is the behavior of Kustomize used by Flux

We referred to the Kustomization CLI flags here, so that you can replicate the same behavior using the CLI.
The behavior of Kustomize used by the controller is currently configured as following:

- `--allow_id_changes` is set to false, so it does not change any resource IDs.
- `--enable_kyaml` is disabled by default, so it currently used `k8sdeps` to process YAMLs.
- `--enable_alpha_plugins` is disabled by default, so it uses only the built-in plugins.
- `--load_restrictor` is set to `LoadRestrictionsNone`, so it allows loading files outside the dir containing `kustomization.yaml`.
- `--reorder` resources is done in the `legacy` mode, so the output will have namespaces and cluster roles/role bindings first, CRDs before CRs, and webhooks last.

!!! hint "`kustomization.yaml` validation"
    To validate changes before committing and/or merging, [a validation
    utility script is available](https://github.com/fluxcd/flux2-kustomize-helm-example/blob/main/scripts/validate.sh),
    it runs `kustomize` locally or in CI with the same set of flags as
    the controller and validates the output using `kubeval`.

## Helm questions

### How to debug "not ready" errors?

Misconfiguring the `HelmRelease.spec.chart`, like a typo in the chart name, version or chart source URL
would result in a "HelmChart is not ready" error displayed by:

```console
$ flux get helmreleases --all-namespaces
NAMESPACE	NAME   	READY	MESSAGE
default  	podinfo	False 	HelmChart 'default/default-podinfo' is not ready
```

In order to get to the root cause, first make sure the source e.g. the `HelmRepository`
is configured properly and has access to the remote `index.yaml`:

```console
$ flux get sources helm --all-namespaces 
NAMESPACE  	NAME   	READY	MESSAGE
default   	podinfo	False	failed to fetch https://stefanprodan.github.io/podinfo2/index.yaml : 404 Not Found
```

If the source is `Ready`, then the error must be caused by the chart,
for example due to an invalid chart name or non-existing version:

```console
$ flux get sources chart --all-namespaces 
NAMESPACE  	NAME           	READY	MESSAGE
default  	default-podinfo	False	no chart version found for podinfo-9.0.0
```

### Can I use Flux HelmReleases without GitOps?

Yes, you can install the Flux components directly on a cluster
and manage Helm releases with `kubectl`.

Install the controllers needed for Helm operations with `flux`:

```sh
flux install \
--namespace=flux-system \
--network-policy=false \
--components=source-controller,helm-controller
```

Create a Helm release with `kubectl`:

```sh
cat << EOF | kubectl apply -f -
---
apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: HelmRepository
metadata:
  name: bitnami
  namespace: flux-system
spec:
  interval: 30m
  url: https://charts.bitnami.com/bitnami
---
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: metrics-server
  namespace: kube-system
spec:
  interval: 60m
  releaseName: metrics-server
  chart:
    spec:
      chart: metrics-server
      version: "^5.x"
      sourceRef:
        kind: HelmRepository
        name: bitnami
        namespace: flux-system
  values:
    apiService:
      create: true
EOF
```

Based on the above definition, Flux will upgrade the release automatically
when Bitnami publishes a new version of the metrics-server chart.

## Flux v1 vs v2 questions

### What does Flux v2 mean for Flux?

Flux v1 is a monolithic do-it-all operator; Flux v2 separates the functionalities into specialized controllers, collectively called the GitOps Toolkit.

You can install and operate Flux v2 simply using the `flux` command. You can easily pick and choose the functionality you need and extend it to serve your own purposes.

The timeline we are looking at right now is:

1. Put Flux v1 into maintenance mode (no new features being added; bugfixes and CVEs patched only).
1. Continue work on the [Flux v2 roadmap](https://toolkit.fluxcd.io/roadmap/).
1. We will provide transition guides for specific user groups, e.g. users of Flux v1 in read-only mode, or of Helm Operator v1, etc. once the functionality is integrated into Flux v2 and it's deemed "ready".
1. Once the use-cases of Flux v1 are covered, we will continue supporting Flux v1 for 6 months. This will be the transition period before it's considered unsupported.

### Why did you rewrite Flux?

Flux v2 implements its functionality in individual controllers, which allowed us to address long-standing feature requests much more easily.

By basing these controllers on modern Kubernetes tooling (`controller-runtime` libraries), they can be dynamically configured with Kubernetes custom resources either by cluster admins or by other automated tools -- and you get greatly increased observability.

This gave us the opportunity to build Flux v2 with the top Flux v1 feature requests in mind:

- Supporting multiple source Git repositories
- Operational insight through health checks, events and alerts
- Multi-tenancy capabilities, like applying each source repository with its own set of permissions

On top of that, testing the individual components and understanding the codebase becomes a lot easier.

### What are significant new differences between Flux v1 and Flux v2?

#### Reconciliation

Flux v1                            | Flux v2
---------------------------------- | ----------------------------------
Limited to a single Git repository | Multiple Git repositories
Declarative config via arguments in the Flux deployment | `GitRepository` custom resource, which produces an artifact which can be reconciled by other controllers
Follow `HEAD` of Git branches | Supports Git branches, pinning on commits and tags, follow SemVer tag ranges
Suspending of reconciliation by downscaling Flux deployment | Reconciliation can be paused per resource by suspending the `GitRepository`
Credentials config via Arguments and/or Secret volume mounts in the Flux pod | Credentials config per `GitRepository` resource: SSH private key, HTTP/S username/password/token, OpenPGP public keys

#### `kustomize` support

Flux v1                            | Flux v2
---------------------------------- | ----------------------------------
Declarative config through `.flux.yaml` files in the Git repository | Declarative config through a `Kustomization` custom resource, consuming the artifact from the GitRepository
Manifests are generated via shell exec and then reconciled by `fluxd` | Generation, server-side validation, and reconciliation is handled by a specialised `kustomize-controller`
Reconciliation using the service account of the Flux deployment | Support for service account impersonation
Garbage collection needs cluster role binding for Flux to query the Kubernetes discovery API | Garbage collection needs no cluster role binding or access to Kubernetes discovery API
Support for custom commands and generators executed by fluxd in a POSIX shell | No support for custom commands

#### Helm integration

Flux v1                            | Flux v2
---------------------------------- | ----------------------------------
Declarative config in a single Helm custom resource | Declarative config through `HelmRepository`, `GitRepository`, `Bucket`, `HelmChart` and `HelmRelease` custom resources
Chart synchronisation embedded in the operator | Extensive release configuration options, and a reconciliation interval per source
Support for fixed SemVer versions from Helm repositories | Support for SemVer ranges for `HelmChart` resources
Git repository synchronisation on a global interval | Planned support for charts from GitRepository sources
Limited observability via the status object of the HelmRelease resource | Better observability via the HelmRelease status object, Kubernetes events, and notifications
Resource heavy, relatively slow | Better performance
Chart changes from Git sources are determined from Git metadata | Chart changes must be accompanied by a version bump in `Chart.yaml` to produce a new artifact

#### Notifications, webhooks, observability

Flux v1                            | Flux v2
---------------------------------- | ----------------------------------
Emits "custom Flux events" to a webhook endpoint | Emits Kubernetes events for included custom resources
RPC endpoint can be configured to a 3rd party solution like FluxCloud to be forwarded as notifications to e.g. Slack | Flux v2 components can be configured to POST the events to a `notification-controller` endpoint. Selective forwarding of POSTed events as notifications using `Provider` and `Alert` custom resources.
Webhook receiver is a side-project | Webhook receiver, handling a wide range of platforms, is included
Unstructured logging | Structured logging for all components
Custom Prometheus metrics | Generic / common `controller-runtime` Prometheus metrics

### How can I get involved?

There are a variety of ways and we look forward to having you on board building the future of GitOps together:

- [Discuss the direction](https://github.com/fluxcd/flux2/discussions) of Flux v2 with us
- Join us in #flux-dev on the [CNCF Slack](https://slack.cncf.io)
- Check out our [contributor docs](https://toolkit.fluxcd.io/contributing/)
- Take a look at the [roadmap for Flux v2](https://toolkit.fluxcd.io/roadmap/)

### Are there any breaking changes?

- In Flux v1 Kustomize support was implemented through `.flux.yaml` files in the Git repository. As indicated in the comparison table above, while this approach worked, we found it to be error-prone and hard to debug. The new [Kustomization CR](https://github.com/fluxcd/kustomize-controller/blob/master/docs/spec/v1alpha1/kustomization.md) should make troubleshooting much easier. Unfortunately we needed to drop the support for custom commands as running arbitrary shell scripts in-cluster poses serious security concerns.
- Helm users: we redesigned the `HelmRelease` API and the automation will work quite differently, so upgrading to `HelmRelease` v2 will require a little work from you, but you will gain more flexibility, better observability and performance.

### Is the GitOps Toolkit related to the GitOps Engine?

In an announcement in August 2019, the expectation was set that the Flux project would integrate the GitOps Engine, then being factored out of ArgoCD. Since the result would be backward-incompatible, it would require a major version bump: Flux v2.

After experimentation and considerable thought, we (the maintainers) have found a path to Flux v2 that we think better serves our vision of GitOps: the GitOps Toolkit. In consequence, we do not now plan to integrate GitOps Engine into Flux.
