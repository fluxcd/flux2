# Migrate to the Helm Controller

This guide will learn you everything you need to know to be able to migrate from the [Helm Operator](https://github.com/fluxcd/helm-operator) to the [Helm Controller](https://github.com/fluxcd/helm-controller).

## Overview of changes

### Support for Helm v2 dropped
   
The Helm Operator offered support for both Helm v2 and v3, due to Kubernetes client incompatibility issues between the versions. This has blocked the Helm Operator from being able to upgrade to a newer v3 version since the release of `3.2.0`.

In combination with the fact that [Helm v2 reaches end of life after November 13, 2020](https://helm.sh/blog/helm-v2-deprecation-timeline/), support for Helm v2 has been dropped.
   
### Helm and Git repositories, and even Helm charts are now Custom Resources

When working with the Helm Operator, you had to mount various files to either make it recognize new (private) Helm repositories or make it gain access to Helm and/or Git repositories. While this approach was declarative, it did not provide a great user experience and was at times hard to set up.

By moving this configuration to [`HelmRepository`](../components/source/helmrepositories.md), [`GitRepository`](../components/source/gitrepositories.md), [`Bucket`](../components/source/buckets.md) and [`HelmChart`](../components/source/helmcharts.md) Custom Resources, they can now be declaratively described (including their credentials using references to `Secret` resources), and applied to the cluster.

The reconciliation of these resources has been offloaded to a dedicated [Source Controller](../components/source/controller.md), specialized in the acquisition of artifacts from external sources.

The result of this all is an easier and more flexible configuration, with much better observability. Failures are traceable to the level of the resource that lead to a failure, and are easier to resolve. As polling intervals can now be configured per resource, you can customize your repository and/or chart configuration to a much finer grain.

From a technical perspective, this also means less overhead, as the resources managed by the Source  Controller can be shared between multiple `HelmRelease` resources, or even reused by other controllers like the [Kustomize Controller](../components/kustomize/controller.md).

### The `HelmRelease` Custom Resource group domain changed

Due to the Helm Controller becoming part of the extensive set of controller components Flux now has, the Custom Resource group domain has changed from `helm.fluxcd.io` to `helm.toolkit.fluxcd.io`.

Together with the new API version (`v2beta1` at time of writing), the full `apiVersion` you use in your YAML document becomes `helm.toolkit.fluxcd.io/v2beta1`.

### The API specification changed (quite a lot), for the better

While developing the Helm Controller, we were given the chance to rethink what a declarative API for driving automated Helm releases would look like. This has, in short, resulted in the following changes:

- Extensive configuration options per Helm action (install, upgrade, test, rollback); this includes things like timeouts, disabling hooks, and ignoring failures for tests.
- Strategy-based remediation on failures. This makes it possible, for example, to uninstall a release instead of rolling it back after a failed upgrade. The number of retries or keeping the last failed state when the retries are exhausted is now a configurable option.
- Better observability. The `Status` field in the `HelmRelease` provides a much better view of the current state of the release, including dedicated `Ready`, `Released`, `TestSuccess`, and `Remediated` conditions.

For a comprehensive overview, see the [API spec changes](#api-spec-changes).

### Helm storage drift detection no longer relies on dry-runs

The Helm Controller no longer uses dry-runs as a way to detect mutations to the Helm storage. Instead, it uses a simpler model of bookkeeping based on the observed state and revisions. This has resulted in much better performance, a lower memory and CPU footprint, and more reliable drift detection.
   
### No longer supports [Helm downloader plugins](https://helm.sh/docs/topics/plugins/#downloader-plugins)

We have reduced our usage of Helm packages to a bare minimum (that being: as much as we need to be able to work with chart repositories and charts), and are avoiding shell outs as much as we can.

Given the latter, and the fact that Helm (downloader) plugins work based on shelling out to another command and/or binary, support for this had to be dropped.

We are aware some of our users are using this functionality to be able to retrieve charts from S3 or GCS. The Source Controller already has support for S3 storage compatible buckets ([this includes GCS](https://cloud.google.com/storage/docs/interoperability)), and we hope to extend this support in the foreseeable future to be on par with the plugins that offered support for these Helm repository types.

### Values from `ConfigMap` and `Secret` resources in other namespaces are no longer supported

Support for values references to `ConfigMap` and `Secret` resources in other namespaces than the namespace of the `HelmRelease` has been dropped, as this allowed information from other namespaces to leak into the composed values for the Helm release.

### Values from external source references (URLs) are no longer supported
   
We initially introduced this feature to support alternative (production focused) `values.yaml` files that sometimes come with charts. It was also used by users to use generic and/or dynamic `values.yaml` files in their `HelmRelease` resources.

The former can now be achieved by defining a [`ValuesFile` overwrite in the `HelmChartTemplateSpec`](#chart-file-references), which will make the Source Controller look for the referenced file in the chart, and overwrite the default values with the contents from that file.

Support for the latter use has been dropped, as it goes against the principles of GitOps and declarative configuration. You can not reliably restore the cluster state from a Git repository if the configuration of a service relies on some URL being available.

Getting similar behaviour is still possible [using a workaround that makes use of a `CronJob` to download the contents of the external URL on an interval](#external-source-references).

### You can now merge single values at a given path

There was a long outstanding request for the Helm Operator to support merging single values at a given path.

With the Helm Controller this now possible by defining a [`targetPath` in the `ValuesReference`](../components/helm/api.md#helm.toolkit.fluxcd.io/v2beta1.ValuesReference), which supports the same formatting as you would supply as an argument to the `helm` binary using `--set [path]=[value]`. In addition to this, the referred value can contain the same value formats (e.g. `{a,b,c}` for a list). You can read more about the available formats and limitations in the [Helm documentation](https://helm.sh/docs/intro/using_helm/#the-format-and-limitations-of---set).

### Support added for depends-on relationships

We have added support for depends-on relationships to install `HelmRelease` resources in a given order; for example, because a chart relies on the presence of a Custom Resource Definition installed by another `HelmRelease` resource.

Entries defined in the `spec.dependsOn` list of the `HelmRelease` must be in a `Ready` state before the Helm Controller proceeds with installation and/or upgrade actions.

Note that this does not account for upgrade ordering. Kubernetes only allows applying one resource (`HelmRelease` in this case) at a time, so there is no way for the controller to know when a dependency `HelmRelease` may be updated. 

Also, circular dependencies between `HelmRelease` resources must be avoided, otherwise the interdependent `HelmRelease` resources will never be reconciled.

### You can now suspend a HelmRelease

There is a new `spec.suspend` field, that if set to `true` causes the Helm Controller to skip reconciliation for the resource. This can be utilized to e.g. temporarily ignore chart changes, and prevent a Helm release from getting upgraded.

### Helm releases can target another cluster

We have added support for making Helm releases to other clusters. If the `spec.kubeConfig` field in the `HelmRelease` is set, Helm actions will run against the default cluster specified in that KubeConfig instead of the local cluster that is responsible for the reconciliation of the `HelmRelease`.

The Helm storage is stored on the remote cluster in a namespace that equals to the namespace of the `HelmRelease`, or the configured `spec.storageNamespace`. The release itself is made in a namespace that equals to the namespace of the `HelmRelease`, or the configured `spec.targetNamespace`. The namespaces are expected to exist, and can for example be created using the [Kustomize Controller](https://toolkit.fluxcd.io/components/kustomize/controller/) which has the same cross-cluster support.
Other references to Kubernetes resources in the `HelmRelease`, like `ValuesReference` resources, are expected to exist on the reconciling cluster.

### Added support for notifications and webhooks

Sending notifications and/or alerts to Slack, Microsoft Teams, Discord, or Rocker is now possible using the [Notification Controller](../components/notification/controller.md), [`Provider` Custom Resources](../components/notification/provider.md) and [`Alert` Custom Resources](../components/notification/alert.md).

It does not stop there, using [`Receiver` Custom Resources](../components/notification/receiver.md) you can trigger **push based** reconciliations from Harbor, GitHub, GitLab, BitBucket or your CI system by making use of the webhook endpoint the resource creates.

### Introduction of the `flux` CLI to create and/or generate Custom Resources

With the new [`flux` CLI](../cmd/flux.md) it is now possible to create and/or generate the Custom Resources mentioned earlier. To generate the YAML for a `HelmRepository` and `HelmRelease` resource, you can for example run:

```console
$ flux create source helm podinfo \
    --url=https://stefanprodan.github.io/podinfo \
    --interval=10m \
    --export
---
apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: HelmRepository
metadata:
  name: podinfo
  namespace: flux-system
spec:
  interval: 10m0s
  url: https://stefanprodan.github.io/podinfo

$ flux create helmrelease podinfo \
    --interval=10m \
    --source=HelmRepository/podinfo \
    --chart=podinfo \
    --chart-version=">4.0.0" \
    --export
---
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: podinfo
  namespace: flux-system
spec:
  chart:
    spec:
      chart: podinfo
      sourceRef:
        kind: HelmRepository
        name: podinfo
      version: '>4.0.0'
  interval: 10m0s

```

## API spec changes

The following is an overview of changes to the API spec, including behavioral changes compared to how the Helm Operator performs actions. For a full overview of the new API spec, consult the [API spec documentation](../components/helm/helmreleases.md#specification).

### Defining the Helm chart

#### Helm repository

For the Helm Operator, you used to configure a chart from a Helm repository as follows:

```yaml
---
apiVersion: helm.fluxcd.io/v1
kind: HelmRelease
metadata:
  name: my-release
  namespace: default
spec:
  chart:
    # The repository URL
    repository: https://charts.example.com
    # The name of the chart (without an alias)
    name: my-chart
    # The SemVer version of the chart
    version: 1.2.3
```

With the Helm Controller, you now create a `HelmRepository` resource in addition to the `HelmRelease` you would normally create (for all available fields, consult the [Source API reference](../components/source/api.md#source.toolkit.fluxcd.io/v1beta1.HelmRepository)):

```yaml
---
apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: HelmRepository
metadata:
  name: my-repository
  namespace: default
spec:
  # The interval at wich to check the upstream for updates
  interval: 10m
  # The repository URL, a valid URL contains at least a protocol and host
  url: https://chart.example.com
```

If you make use of a private Helm repository, instead of configuring the credentials by mounting a `repositories.yaml` file, you can now configure the HTTP/S basic auth and/or TLS credentials by referring to a `Secret` in the same namespace as the `HelmRepository`:

```yaml
---
apiVersion: v1
kind: Secret
metadata:
  name: my-repository-creds
  namespace: default
data:
  # HTTP/S basic auth credentials
  username: <base64 encoded username>
  password: <base64 encoded password>
  # TLS credentials (certFile and keyFile, and/or caCert)
  certFile: <base64 encoded certificate>
  keyFile: <base64 encoded key>
  caCert: <base64 encoded CA certificate>
---
apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: HelmRepository
metadata:
  name: my-repository
  namespace: default
spec:
  # ...omitted for brevity
  secretRef:
    name: my-repository-creds
```

In the `HelmRelease`, you then use a reference to the `HelmRepository` resource in the `spec.chart.spec` (for all available fields, consult the [Helm API reference](../components/helm/api.md#helm.toolkit.fluxcd.io/v2beta1.HelmChartTemplate)):

```yaml
---
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: my-release
  namespace: default
spec:
  # The interval at which to reconcile the Helm release
  interval: 10m
  chart:
    spec:
      # The name of the chart as made available by the HelmRepository
      # (without any aliases)
      chart: my-chart
      # A fixed SemVer, or any SemVer range
      # (i.e. >=4.0.0 <5.0.0)
      version: 1.2.3
      # The reference to the HelmRepository
      sourceRef:
        kind: HelmRepository
        name: my-repository
        # Optional, defaults to the namespace of the HelmRelease
        namespace: default
```

The `spec.chart.spec` values are used by the Helm Controller as a template to create a new `HelmChart` resource in the same namespace as the `sourceRef`, to be reconciled by the Source Controller. The Helm Controller watches `HelmChart` resources for (revision) changes, and performs an installation or upgrade when it notices a change.

#### Git repository

For the Helm Operator, you used to configure a chart from a Git repository as follows:

```yaml
---
apiVersion: helm.fluxcd.io/v1
kind: HelmRelease
metadata:
  name: my-release
  namespace: default
spec:
  chart:
    # The URL of the Git repository
    git: https://example.com/org/repo
    # The Git branch (or other Git reference)
    ref: master
    # The path of the chart relative to the repository root
    path: ./charts/my-chart
```

With the Helm Controller, you create a `GitRepository` resource in addition to the `HelmRelease` you would normally create (for all available fields, consult the [Source API reference](../components/source/api.md#source.toolkit.fluxcd.io/v1beta1.GitRepository):

```yaml
---
apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: GitRepository
metadata:
  name: my-repository
  namespace: default
spec:
  # The interval at which to check the upstream for updates
  interval: 10m
  # The repository URL, can be a HTTP/S or SSH address
  url: https://example.com/org/repo
  # The Git reference to checkout and monitor for changes
  # (defaults to master)
  # For all available options, see:
  # https://toolkit.fluxcd.io/components/source/api/#source.toolkit.fluxcd.io/v1beta1.GitRepositoryRef
  ref:
    branch: master
```

If you make use of a private Git repository, instead of configuring the credentials by mounting a private key and making changes to the `known_hosts` file, you can now configure the credentials for both HTTP/S and SSH by referring to a `Secret` in the same namespace as the `GitRepository`:

```yaml
---
apiVersion: v1
kind: Secret
metadata:
  name: my-repository-creds
  namespace: default
data:
  # HTTP/S basic auth credentials
  username: <base64 encoded username>
  password: <base64 encoded password>
  # SSH credentials
  identity: <base64 encoded private key>
  identity.pub: <base64 public key>
  known_hosts: <base64 encoded known_hosts>
---
apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: GitRepository
metadata:
  name: my-repository
  namespace: default
spec:
  # ...omitted for brevity
  secretRef:
    name: my-repository-creds
```

In the `HelmRelease`, you then use a reference to the `GitRepository` resource in the `spec.chart.spec` (for all available fields, consult the [Helm API reference](../components/helm/api.md#helm.toolkit.fluxcd.io/v2beta1.HelmChartTemplate)):

```yaml
---
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: my-release
  namespace: default
spec:
  # The interval at which to reconcile the Helm release
  interval: 10m
  chart:
    spec:
      # The path of the chart relative to the repository root
      chart: ./charts/my-chart
      # The reference to the GitRepository
      sourceRef:
        kind: GitRepository
        name: my-repository
        # Optional, defaults to the namespace of the HelmRelease
        namespace: default
```

The `spec.chart.spec` values are used by the Helm Controller as a template to create a new `HelmChart` resource in the same namespace as the `sourceRef`, to be reconciled by the Source Controller. The Helm Controller watches `HelmChart` resources for (revision) changes, and performs an installation or upgrade when it notices a change.

### Defining values

#### Inlined values

Inlined values (defined in the `spec.values` of the `HelmRelease`) still work as with the Helm operator. It represents a YAML map as you would put in a file and supply to `helm` with `-f values.yaml`, but inlined into the `HelmRelease` manifest:

```yaml
---
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: my-release
  namespace: default
spec:
  # ...omitted for brevity
  values:
    foo: value1
    bar:
      baz: value2
    oof:
    - item1
    - item2
```

#### Values from sources

As described in the [overview of changes](#overview-of-changes), there have been multiple changes to the way you can refer to values from sources (like `ConfigMap` and `Secret` references), including the [drop of support for external source (URL) references](#values-from-external-source-references-urls-are-no-longer-supported) and [added support for merging single values at a specific path](#you-can-now-merge-single-values-at-a-given-path).

Values are still merged in the order given, with later values overwriting earlier. The values from sources always have a lower priority than the values inlined in the `HelmRelease` via the `spec.values` key.

##### `ConfigMap` and `Secret` references

`ConfigMap` and `Secret` references used to be defined as follows:

```yaml
---
apiVersion: helm.fluxcd.io/v1
kind: HelmRelease
metadata:
  name: my-release
  namespace: default
spec:
  # ...omitted for brevity
  valuesFrom:
  - configMapKeyRef:
      name: my-config-values
      namespace: my-ns
      key: values.yaml
      optional: false
  - secretKeyRef:
      name: my-secret-values
      namespace: my-ns
      key: values.yaml
      optional: true
```

In the new API spec the individual `configMapKeyRef` and `secretKeyRef` objects are bundled into a single [`ValuesReference`](../components/helm/api.md#helm.toolkit.fluxcd.io/v2beta1.ValuesReference) which [does no longer allow refering to resources in other namespaces](#values-from-external-source-references-urls-are-no-longer-supported):

```yaml
---
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: my-release
  namespace: default
spec:
  # ...omitted for brevity
  valuesFrom:
  - kind: ConfigMap
    name: my-config-values
    valuesKey: values.yaml
    optional: false
  - kind: Secret
    name: my-secret-values
    valuesKey: values.yaml
    optional: true
```

Another thing to take note of is that the behavior for values references marked as `optional` has changed. When set, a "not found" error for the values reference is ignored, but any `valuesKey`, `targetPath` or transient error will still result in a reconciliation failure.

##### Chart file references

With the Helm Operator it was possible to refer to an alternative values file (for e.g. production usage) in the directory of a chart from a Git repository:

```yaml
---
apiVersion: helm.fluxcd.io/v1
kind: HelmRelease
metadata:
  name: my-release
  namespace: default
spec:
  # ...omitted for brevity
  valuesFrom:
  # Values file to merge in,
  # expected to be a relative path in the chart directory
  - chartFileRef:
      path: values-prod.yaml
```

With the Helm Controller, this declaration has moved to the `spec.chart.spec`, and the feature is no longer limited to charts from a Git repository:

```yaml
---
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: my-release
  namespace: default
spec:
  # ...omitted for brevity
  chart:
    spec:
      chart: my-chart
      version: 1.2.3
      # Alternative values file to use as the default values,
      # expected to be a relative path in the sourceRef
      valuesFile: values-prod.yaml
      sourceRef:
        kind: HelmRepository
        name: my-repository
```

When the `valuesFile` is defined, the chart will be (re)packaged with the values from the referenced file as the default values. Note that this behavior is different from the Helm Operator and requires a full set of alternative values, as the referenced values are no longer merged with the default values.

##### External source references

While [the support for external source references has been dropped](#values-from-external-source-references-urls-are-no-longer-supported), it is possible to work around this limitation by creating a `CronJob` that periodically fetches the values from an external URL and saves them to a `ConfigMap` or `Secret` resource.

First, create a `ServiceAccount`, `Role` and `RoleBinding` capable of updating a limited set of `ConfigMap` resources:

```yaml
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: values-fetcher
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: configmap-updater
  namespace: default
rules:
- apiGroups: [""]
  resources: ["configmaps"]
  # ResourceNames limits the access of the role to
  # a defined set of ConfigMap resources
  resourceNames: ["my-external-values"]
  verbs: ["patch", "get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: update-values-configmaps
  namespace: default
subjects:
- kind: ServiceAccount
  name: values-fetcher
  namespace: default
roleRef:
  kind: Role
  name: configmap-updater
  apiGroup: rbac.authorization.k8s.io
```

As `resourceNames` scoping in the `Role` [does not allow restricting `create` requests](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#referring-to-resources), we need to create empty placeholder(s) for the `ConfigMap` resource(s) that will hold the fetched values:

```yaml
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-external-values
  namespace: default
data: {}
```

Lastly, create a `CronJob` that uses the `ServiceAccount` defined above, fetches the external values on an interval, and applies them to the `ConfigMap`:

```yaml
---
apiVersion: batch/v1beta1
kind: CronJob
metadata:
  name: fetch-external-values
spec:
  concurrencyPolicy: Forbid
  schedule: "*/5 * * * *"
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 3
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: values-fetcher
          containers:
          - name: kubectl
            image: bitnami/kubectl:1.19
            volumeMounts:
            - mountPath: /tmp
              name: tmp-volume
            command:
            - sh
            - -c
            args:
            - >-
              curl -f -# https://example.com/path/to/values.yaml -o /tmp/values.yaml &&
              kubectl create configmap my-external-values --from-file=/tmp/values.yaml -oyaml --dry-run=client |
              kubectl apply -f -
          volumes:
          - name: tmp-volume
            emptyDir:
              medium: Memory
          restartPolicy: OnFailure
```

You can now refer to the `my-external-values` `ConfigMap` resource in your `HelmRelease`:

```yaml
---
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: my-release
  namespace: default
spec:
  # ...omitted for brevity
  valuesFrom:
  - kind: ConfigMap
    name: my-external-values
```

### Defining release options

With the Helm Operator the release options used to be configured in the `spec` of the `HelmRelease` and applied to both Helm install and upgrade actions.

This has changed for the Helm Controller, where some defaults can be defined in the [`spec`](../components/helm/api.md#helm.toolkit.fluxcd.io/v2beta1.HelmReleaseSpec), but specific action configurations and overwrites for the defaults can be defined in the [`spec.install`](../components/helm/api.md#helm.toolkit.fluxcd.io/v2beta1.Install), [`spec.upgrade`](../components/helm/api.md#helm.toolkit.fluxcd.io/v2beta1.Upgrade) and [`spec.test`](../components/helm/api.md#helm.toolkit.fluxcd.io/v2beta1.Test) sections of the `HelmRelease`.

### Defining a rollback / uninstall configuration

With the Helm Operator, uninstalling a release after an installation failure was done automatically, and rolling back from a faulty upgrade and configuring options like retries was done as follows:

```yaml
---
apiVersion: helm.fluxcd.io/v1
kind: HelmRelease
metadata:
  name: my-release
  namespace: default
spec:
  # ...omitted for brevity
  rollback:
    enable: true
    retries: true
    maxRetries: 5
    disableHooks: false
    force: false
    recreate: false
    timeout: 300
```

The Helm Controller offers an extensive set of configuration options to remediate when a Helm release fails, using [`spec.install.remediate`](../components/helm/api.md#helm.toolkit.fluxcd.io/v2beta1.InstallRemediation), [`spec.upgrade.remediate`](../components/helm/api.md#helm.toolkit.fluxcd.io/v2beta1.UpgradeRemediation), [`spec.rollback`](../components/helm/api.md#helm.toolkit.fluxcd.io/v2beta1.Rollback) and [`spec.uninstall`](../components/helm/api.md#helm.toolkit.fluxcd.io/v2beta1.Uninstall). Some of the new features include the option to remediate with an uninstall after an upgrade failure, and the option to keep a failed release for debugging purposes when it has run out of retries.

#### Automated uninstalls

The configuration below mimics the uninstall behavior of the Helm Operator (for all available fields, consult the [`InstallRemediation`](../components/helm/api.md#helm.toolkit.fluxcd.io/v2beta1.InstallRemediation) and [`Uninstall`](../components/helm/api.md#helm.toolkit.fluxcd.io/v2beta1.Uninstall) API references):

```yaml
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: my-release
  namespace: default
spec:
  # ...omitted for brevity
  install:
    # Remediation configuration for when the Helm install
    # (or sequent Helm test) action fails
    remediation:
      # Number of retries that should be attempted on failures before
      # bailing, a negative integer equals to unlimited retries
      retries: -1
  # Configuration options for the Helm uninstall action
  uninstall:
    timeout: 5m
    disableHooks: false
    keepHistory: false
```

#### Automated rollbacks

The configuration below shows an automated rollback configuration that equals [the configuration for the Helm Operator showed above](#defining-a-rollback-uninstall-configuration) (for all available fields, consult the [`UpgradeRemediation`](../components/helm/api.md#helm.toolkit.fluxcd.io/v2beta1.UpgradeRemediation) and [`Rollback`](../components/helm/api.md#helm.toolkit.fluxcd.io/v2beta1.Rollback) API references):

```yaml
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: my-release
  namespace: default
spec:
  # ...omitted for brevity
  upgrade:
    # Remediaton configuration for when an Helm upgrade action fails
    remediation:
      # Amount of retries to attempt after a failure,
      # setting this to 0 means no remedation will be
      # attempted
      retries: 5
  # Configuration options for the Helm rollback action
  rollback:
    timeout: 5m
    disableWait: false
    disableHooks: false
    recreate: false
    force: false
    cleanupOnFail: false
```

## Migration strategy

Due to the high number of changes to the API spec, there are no detailed instructions available to provide a simple migration path. But there is a [simple procedure to follow](#steps), which combined with the detailed list of [API spec changes](#api-spec-changes) should make the migration path relatively easy. 

Here are some things to know:

* The Helm Controller will ignore the old custom resources (and the Helm Operator will ignore the new resources).
* Deleting a resource while the corresponding controller is running will result in the Helm release also being deleted.
* Deleting a `CustomResourceDefinition` will also delete all custom resources of that kind.
* If both the Helm Controller and Helm Operator are running, and both a new and old custom resources define a release, they will fight over the release.
* The Helm Controller will always perform an upgrade the first time it encounters a new `HelmRelease` for an existing release; this is [due to the changes to release mechanics and bookkeeping](#helm-storage-drift-detection-no-longer-relies-on-dry-runs).

The safest way to upgrade is to avoid deletions and fights by stopping the Helm Operator. Once the operator is not running, it is safe to deploy the Helm Controller (e.g., by following the [Get Started guide](../get-started/index.md), [utilizing `flux install`](../cmd/flux_install.md), or using the manifests from the [release page](https://github.com/fluxcd/helm-controller/releases)), and start replacing the old resources with new resources. You can keep the old resources around during this process, since the Helm Controller will ignore them.

### Steps

The recommended migration steps for a single `HelmRelease` are as follows:

1. Ensure the Helm Operator is not running, as otherwise the Helm Controller and Helm Operator will fight over the release.
1. Create a [`GitRepository` or `HelmRepository` resource for the `HelmRelease`](#defining-the-helm-chart), including any `Secret` that may be required to access the source. Note that it is possible for multiple `HelmRelease` resources to share a `GitRepository` or `HelmRepository` resource.
1. Create a new `HelmRelease` resource ([with the `helm.toolkit.fluxcd.io` group domain](#the-helmrelease-custom-resource-group-domain-changed)), define the `spec.releaseName` (plus the `spec.targetNamespace` and `spec.storageNamespace` if applicable) to match that of the existing release, and rewrite the configuration to adhere to the [API spec changes](#api-spec-changes).
1. Confirm the Helm Controller successfully upgrades the release.

### Example

As a full example, this is an old resource:

```yaml
---
apiVersion: helm.fluxcd.io/v1
kind: HelmRelease
metadata:
  name: podinfo
  namespace: default
spec:
  chart:
    repository: https://stefanprodan.github.io/podinfo
    name: podinfo
    version: 5.0.3
  values:
    replicaCount: 1
```

The custom resources for the Helm Controller would be:

```yaml
---
apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: HelmRepository
metadata:
  name: podinfo
  namespace: default
spec:
  interval: 10m
  url: https://stefanprodan.github.io/podinfo
---
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: podinfo
  namespace: default
spec:
  interval: 5m
  releaseName: default-podinfo
  chart:
    spec:
      chart: podinfo
      version: 5.0.3
      sourceRef:
        kind: HelmRepository
        name: podinfo
      interval: 10m
  values:
    replicaCount: 1
```

### Migrating gradually

Gradually migrating to the Helm Controller is possible by scaling down the Helm Operator while you move over resources, and scaling it up again once you have migrated some of the releases to the Helm Controller.

While doing this, make sure that once you scale up the Helm Operator again, there are no old and new `HelmRelease` resources pointing towards the same release, as they will fight over the release.

### Deleting old resources

Once you have migrated all your `HelmRelease` resources to the Helm Controller. You can remove all of the old resources by removing the old Custom Resource Definition.

```sh
kubectl delete crd helm.fluxcd.io
```

## Frequently Asked Questions

### Are automated image updates supported?

Not yet, but the feature is under active development. See the [image update feature parity section on the roadmap](https://toolkit.fluxcd.io/roadmap/#flux-image-update-feature-parity) for updates on this topic.

### How do I automatically apply my `HelmRelease` resources to the cluster?

If you are currently a Flux v1 user, you can commit the `HelmRelease` resources to Git, and Flux will automatically apply them to the cluster like any other resource. It does however not support automated image updates for Helm Controller resources.

If you are not a Flux v1 user or want to fully migrate to Flux v2, the [Kustomize Controller](https://toolkit.fluxcd.io/components/kustomize/controller/) will serve your needs.

### I am still running Helm v2, what is the right upgrade path for me?

Migrate your Helm v2 releases to v3 using [the Helm Operator's migration feature](https://docs.fluxcd.io/projects/helm-operator/en/stable/helmrelease-guide/release-configuration/#migrating-from-helm-v2-to-v3), or make use of the [`helm-2to3`](https://github.com/helm/helm-2to3) plugin directly, before continuing following the [migration steps](#steps).

### Is the Helm Controller ready for production?

Probably, but with some side notes:

1. It is still under active development, and while our focus has been to stabilize the API as much as we can during the first development phase, we do not guarantee there will not be any breaking changes before we reach General Availability. We are however committed to provide [conversion webhooks](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/#webhook-conversion) for upcoming API versions.
1. There may be (internal) behavioral changes in upcoming releases, but they should be aimed at further stabilizing the Helm Controller itself, solving edge case issues, providing better logging, observability, and/or other improvements.

### Can I use Helm Controller standalone?

Helm Controller depends on [Source Controller](../components/source/controller.md), you can install both controllers
and manager Helm releases in a declarative way without GitOps.
For more details please see this [answer](../faq/index.md#can-i-use-flux-helmreleases-without-gitops).

### I have another question

Given the amount of changes, it is quite possible that this document did not provide you with a clear answer for you specific setup. If this applies to you, do not hestitate to ask for help in the [GitHub Discussions](https://github.com/fluxcd/flux2/discussions/new?category_id=31999889) or on the [`#flux` CNCF Slack channel](https://slack.cncf.io)!
