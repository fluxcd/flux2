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

### What are the differences between v1 and v2?

Flux v1 is a monolithic do-it-all operator;
Flux v2 separates the functionalities into specialized controllers, collectively called the GitOps Toolkit.

You can find a detailed comparison of Flux v1 and v2 features in the [migration FAQ](../guides/faq-migration.md).

### Why should I upgrade

Flux v2 includes some breaking changes, which means there is some work required to migrate. We hope that Flux users can all be persuaded to upgrade. There are some great reasons to follow the FluxCD organization's latest hard work and consider upgrading to Flux v2:

#### Flux v1 runtime behavior doesn't scale well

While there are many Flux v1 users in production, and some of them are running at very large scales, Flux users with smaller operations or those that haven't needed to scale maybe didn't notice that Flux v1 actually doesn't scale very well at all.

Some architectural issues in the original design of Flux weren't practical to resolve, or weren't known, until after implementations could be attempted at scale.

One can debate the design choices made in Flux v1 vs. Flux v2, but it was judged by the maintainers that the design of Flux importantly required some breaking changes to resolve some key architectural issues.

Flux v1 implementation of image automation has serious performance issues scaling into thousands of images. This came to a head when Docker Hub started rate-limiting image pulls, because of the expense of this operation performed casually and at scale.

That's right, rate limiting undoutedly happened because of abusive clients pulling image metadata from many images (like Flux v1 did,) images that might only be stored for the purpose of retention policies, that might be relegated to cold storage if they were not being periodically retrieved.

Flux v2 resolved this with [sortable image tags](/guides/sortable-image-tags/); (this is a breaking change.)

Flux v1 requires one Flux daemon to be running per git repository/branch that syncs to the cluster. Flux v2 only expects cluster operators to run one source-controller instance, allowing to manage multiple repositories, or multiple clusters (or an entire fleet) with just one Flux installation.

Fundamentally, Flux v1 was one single configuration and reconciliation per daemon process, while Flux v2 is designed to handle many configurations for concurrent resources like git repositories, helm charts, helm releases, tenants, clusters, Kustomizations, git providers, alert providers, (... the list continues to grow.)

#### Flux v2 is more reliable and observable

As many advanced Flux v1 users will know, Flux's manifest generation capabilities come at a heavy cost. If manifest generation takes too long, timeout errors in Flux v1 can pre-empt the main loop and prevent the reconciliation of manifests altogether. The effect of this circumstance is a complete Denial-of-Service for changes to any resources managed by Flux v1 — it goes without saying, this is very bad.

Failures in Flux v2 are handled gracefully, with each controller performing separate reconciliations on the resources in their domain. One Kustomization can fail reconciling, or one GitRepository can fail syncing (for whatever reason including its own configurable timeout period), without interrupting the whole Flux system.

An error is captured as a Kubernetes `Event` CRD, and is reflected in the `Status` of the resource that had an error. When there is a fault, the new design allows that other processes should not be impacted by the fault.

#### Flux v2 covers new use cases

There is an idealized use case of GitOps we might explain as: when an update comes, a pull-request is automatically opened and when it gets merged, it is automatically applied to the cluster. That sounds great, but is not really how things work in Flux v1.

In Flux v2, this can actually be used as a real strategy; it is straight-forward to implement and covered by documenation: [Push updates to a different branch](/guides/image-update/#push-updates-to-a-different-branch).

In Flux v1, it was possible to set up incoming webhooks with [flux-recv](https://github.com/fluxcd/flux-recv) as a sidecar to Flux, which while it worked nicely, it isn't nicely integrated and frankly feels bolted-on, sort of like an after-market part. This may be more than appearance, it isn't mentioned at all in Flux v1 docs!

The notification-controller is a core component in the architecture of Flux v2 and the `Receiver` CRD can be used to configure similar functionality with included support for the multi-repository features of Flux.

Similarly, in Flux v1 it was possible to send notifications to outside webhooks like Slack, MS Teams, and GitHub, but only with the help of third-party tools like [justinbarrick/fluxcloud](https://github.com/justinbarrick/fluxcloud). This functionality has also been subsumed as part of notification-controller and the `Alert` CRD can be used to configure outgoing notifications for a growing list of alerting providers today!

#### Flux v2 takes advantage of Kubernetes Extension API

The addition of CRDs to the design of Flux is another great reason to upgrade. Flux v1 had a very limited API which was served from the Flux daemon, usually controlled by using `fluxctl`, which has limited capabilities of inspection, and limited control over the behavior. By using CRDs, Flux v2 can take advantage of the Kubernetes API's extensibility so Flux itself doesn't need to run any daemon which responds directly to API requests.

Operations through Custom Resources (CRDs) provide great new opportunities for observability and eventing as was explained already, and also provides greater reliability through centralization.

Using one centralized, highly-available API service (the Kubernetes API) not only improves reliability, but is a great move for security as well; this decision reduces the risk that when new components are added, growing the functionality of the API, with each step we take we are potentially growing the attack surfaces.

The Kubernetes API is secured by default with TLS certificates for authentication and mandates RBAC configuration for authorization. It's also available in every namespace on the cluster, with a default service account. This is a highly secure API design, and it is plain to see this implementation has many eyes on it.

#### Flux v1 won't be supported forever

The developers of Flux have committed to maintain Flux v1 to support their production user-base for a reasonable span of time.

It's understood that many companies cannot adopt Flux v2 while it remains in prerelease state. So [Flux v1 is in maintenance mode](https://github.com/fluxcd/flux/issues/3320), which will continue at least until a GA release of Flux v2 is announced, and security updates and critical fixes can be made available for at least 6 months following that time.

System administrators often need to plan their migrations far in advance, and these dates won't remain on the horizon forever. It was announced as early as August 2019 that Flux v2 would be backward-incompatible, and that users would eventually be required to upgrade in order to continue to receive support from the maintainers.

Many users of Flux have already migrated production environments to Flux v2. Consider that the sooner this upgrade is undertaken by your organization, the sooner you can get it over with and have put this all behind you.

### How can I migrate from v1 to v2?

The Flux community has created guides and example repositories
to help you migrate to Flux v2:

- [Migrate from Flux v1](https://toolkit.fluxcd.io/guides/flux-v1-migration/)
- [Migrate from `.flux.yaml` and kustomize](https://toolkit.fluxcd.io/guides/flux-v1-migration/#flux-with-kustomize)
- [Migrate from Flux v1 automated container image updates](https://toolkit.fluxcd.io/guides/flux-v1-automation-migration/)
- [How to manage multi-tenant clusters with Flux v2](https://github.com/fluxcd/flux2-multi-tenancy)
- [Migrate from Helm Operator to Flux v2](https://toolkit.fluxcd.io/guides/helm-operator-migration/)
- [How to structure your HelmReleases](https://github.com/fluxcd/flux2-kustomize-helm-example)
