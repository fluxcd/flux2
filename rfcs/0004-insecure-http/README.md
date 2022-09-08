# RFC-0004 Block insecure HTTP connections across Flux

**Status:** provisional

**Creation Date:** 2022-09-08

## Summary
Flux should have a consistent way of disabling insecure HTTP connections.

At the controller level, a flag should be present which would disable all outgoing HTTP connections.
At the object level, a field should be provided which would enable the use of non-TLS endpoints.

If the use of a non-TLS endpoint is not supported, it should be made clear to users through the use of
logs and status conditions.

## Motivation
Today the use of non-TLS based connections is inconsistent across Flux controllers.

Controllers that deal only with `http` and `https` schemes have no way to block use of the `http` scheme at controller-level.
Some Flux objects provide a `.spec.insecure` field to enable the use of non-TLS based endpoints, but they don't clearly notify users when the option is not supported (e.g. Azure/GCP Buckets).

### Goals
* Provide a flag across all Flux controllers which disables all outgoing HTTP connections.
* Add a field which enables the use of non-TLS endpoints to appropriate Flux objects.
* Provide a way for users to be made aware that their use of non-TLS endpoints is not supported if that is the case.

### Non-Goals
* Break Flux's current behavior of allowing HTTP connections.

## Proposal

### Controllers
Flux users should be able to enforce that controllers are using HTTPS connections only.
This shall be enabled by adding a new boolean flag `--allow-insecure-http` to the following controllers:
* source-controller
* notification-controller
* image-automation-controller
* image-reflector-controller

> Note: The flag shall not be added to the following controllers:
> * kustomize-controller: This flag is excluded from this controller, as the upstream `kubenetes-sigs/kustomize` project
> does not support disabling HTTP connections while fetching resources from remote bases. We can revisit this if the
> upstream project adds support for this at a later point in time.
> * helm-controller: This flag does not serve a purpose in this controller, as the controller does not make any HTTP calls.
> Furthermore although both controllers can also do remote applies, serving `kube-apiserver` over plain
> HTTP is disabled by default. While technically this can be enabled, the option for this configuration was also disabled
> quite a while back (ref: https://github.com/kubernetes/kubernetes/pull/65830/).

The default value of this flag shall be `true`. This would ensure that there is no breaking change with controllers
still being able to access non-TLS endpoints. To disable this behavior and enforce the use of HTTPS connections, users would
have to explicitly pass the flag to the controller:

```yaml
spec:
  template:
    spec:
      containers:
      - name: manager
        image: fluxcd/source-controller
        args:
          - --watch-all-namespaces
          - --log-level=info
          - --log-encoding=json
          - --enable-leader-election
          - --storage-path=/data
          - --storage-adv-addr=source-controller.$(RUNTIME_NAMESPACE).svc.cluster.local.
          - --allow-insecure-http=false
```

### Objects
Some Flux objects, like `GitRepository`, provide a field for specifying a URL, and the URL would contain the scheme.
In such cases, the scheme can be used for inferring the transport type of the connection and consequently,
whether to use HTTP or HTTPS connections for that object.
But there are a few objects that don't allow such behavior, for example:

* `ImageRepository`: It provides a field, `.spec.image`, which is used for specifying the URL of the image present on
a container registry. But any URL containing a scheme is considered invalid and HTTPS is the default transport used.
This prevents users from using images present on insecure registries.
* OCI `HelmRepository`: When using an OCI registry as a Helm repository, the `.spec.url` is expected to begin with `oci://`.
Since the scheme part of the URL is used to specify the type of `HelmRepository`, there is no way for users to specify
that the registry is hosted at a non-TLS endpoint.

For such objects, we shall introduce a new boolean field `.spec.insecure`, which shall be `false` by default. Users that
need their object to point to an HTTP endpoint, can set this to `true`.

### CLI
The Flux CLI offers several commands for creating Flux specific resources. Some of these commands may involve specifying
an endpoint such as creating an `OCIRepository`:

```sh
 flux create source oci podinfo \
    --url=oci://ghcr.io/stefanprodan/manifests/podinfo \
    --tag=6.1.6 \
    --interval=10m
```

Since these commands essentially create object definitions, the CLI should offer a boolean flag `--insecure`
for relevant objects, which will be used for specifying the value of `.spec.insecure` of such objects.

### Precedence & Validity
Objects with `.spec.insecure` as `true ` will only be allowed if HTTP connections are allowed at the controller level.
Similarly, an object can have `.spec.insecure` as `true` only if the Saas/Cloud provider allows HTTP connections.
For example, using a `Bucket` with its `.spec.provider` set to `azure` would be invalid since Azure doesn't allow
HTTP connections.


### User Stories

#### Story 1
> As a cluster admin of a multi-tenant cluster, I want to ensure all controllers access endpoints using only HTTPS
> regardless of tenants' object definitions.

Apply a `kustomize` patch which prevents the use of HTTP connections:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - gotk-components.yaml
  - gotk-sync.yaml
patches:
  - patch: |
      - op: add
        path: /spec/template/spec/containers/0/args/-
        value: --allow-insecure-http=false
    target:
      kind: Deployment
      name: "(kustomize-controller|helm-controller|source-controller|notification-controller)"
```

#### Story 2
> As an application developer, I'm trying to debug a new image pushed to my local registry which
> is not served over HTTPS.

Modify the object spec to use HTTP connections explicitly:
```yaml
apiVersion: image.toolkit.fluxcd.io/v1beta1
kind: ImageRepository
metadata:
  name: podinfo
  namespace: flux-system
spec:
  image: kind-registry:5000/stefanprodan/podinfo
  interval: 1m0s
  insecure: true
```

### Alternatives
Instead of adding a flag, we can instruct users to make use of Kyverno policies to enforce that
all objects have `.spec.insecure` as `false` and any URLs present in the definition don't have `http`
as the scheme. This is less attractive, as this would ask users to install another software and prevent
Flux multi-tenancy from being standalone.

## Design Details
If a controller is started with `--insecure-allow-http=false`, any URL in a Flux object which has `http`
as the scheme will result in an error and the following condition will be added to the object's
`.status.conditions`:

```yaml
status:
  conditions:
  - lastTransitionTime: "2022-09-06T09:14:21Z"
    message: "Use of insecure HTTP connections isn't allowed for this controller"
    observedGeneration: 1
    reason: URLInvalid
    status: "True"
    type: FetchFailedCondition
```

Similarly, if an object has `.spec.insecure` as `true` but the Cloud provider doesn't allow HTTP connections,
the reconciler will error out and add the condition below to the object's `.status.conditions`:

```yaml
status:
  conditions:
  - lastTransitionTime: "2022-09-06T09:14:21Z"
    message: "Use of insecure HTTP connections isn't allowed for Azure Storage"
    observedGeneration: 1
    reason: InsecureConnectionsDisallowed
    status: "True"
    type: FetchFailedCondition
```

If an object has `.spec.insecure` as `true`, the registry client or bucket client shall be created with the use
of HTTP connections enabled explicitly.

