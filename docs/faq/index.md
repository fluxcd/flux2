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

### How can I migrate from v1 to v2?

The Flux community has created guides and example repositories
to help you migrate to Flux v2:

- [Migrate from Flux v1](https://toolkit.fluxcd.io/guides/flux-v1-migration/)
- [Migrate from `.flux.yaml` and kustomize](https://toolkit.fluxcd.io/guides/flux-v1-migration/#flux-with-kustomize)
- [Migrate from Flux v1 automated container image updates](https://toolkit.fluxcd.io/guides/flux-v1-automation-migration/)
- [How to manage multi-tenant clusters with Flux v2](https://github.com/fluxcd/flux2-multi-tenancy)
- [Migrate from Helm Operator to Flux v2](https://toolkit.fluxcd.io/guides/helm-operator-migration/)
- [How to structure your HelmReleases](https://github.com/fluxcd/flux2-kustomize-helm-example)
