## flux create helmrelease

Create or update a HelmRelease resource

### Synopsis

The helmrelease create command generates a HelmRelease resource for a given HelmRepository source.

```
flux create helmrelease [name] [flags]
```

### Examples

```
  # Create a HelmRelease with a chart from a HelmRepository source
  flux create hr podinfo \
    --interval=10m \
    --source=HelmRepository/podinfo \
    --chart=podinfo \
    --chart-version=">4.0.0"

  # Create a HelmRelease with a chart from a GitRepository source
  flux create hr podinfo \
    --interval=10m \
    --source=GitRepository/podinfo \
    --chart=./charts/podinfo

  # Create a HelmRelease with a chart from a Bucket source
  flux create hr podinfo \
    --interval=10m \
    --source=Bucket/podinfo \
    --chart=./charts/podinfo

  # Create a HelmRelease with values from a local YAML file
  flux create hr podinfo \
    --source=HelmRepository/podinfo \
    --chart=podinfo \
    --values=./my-values.yaml

  # Create a HelmRelease with a custom release name
  flux create hr podinfo \
    --release-name=podinfo-dev
    --source=HelmRepository/podinfo \
    --chart=podinfo \

  # Create a HelmRelease targeting another namespace than the resource
  flux create hr podinfo \
    --target-namespace=default \
    --source=HelmRepository/podinfo \
    --chart=podinfo

  # Create a HelmRelease definition on disk without applying it on the cluster
  flux create hr podinfo \
    --source=HelmRepository/podinfo \
    --chart=podinfo \
    --values=./values.yaml \
    --export > podinfo-release.yaml

```

### Options

```
      --chart string              Helm chart name or path
      --chart-version string      Helm chart version, accepts a semver range (ignored for charts from GitRepository sources)
      --depends-on stringArray    HelmReleases that must be ready before this release can be installed, supported formats '<name>' and '<namespace>/<name>'
  -h, --help                      help for helmrelease
      --release-name string       name used for the Helm release, defaults to a composition of '[<target-namespace>-]<HelmRelease-name>'
      --source helmChartSource    source that contains the chart in the format '<kind>/<name>',where kind can be one of: (HelmRepository, GitRepository, Bucket)
      --target-namespace string   namespace to install this release, defaults to the HelmRelease namespace
      --values string             local path to the values.yaml file
```

### Options inherited from parent commands

```
      --context string      kubernetes context to use
      --export              export in YAML format to stdout
      --interval duration   source sync interval (default 1m0s)
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
      --label strings       set labels on the resource (can specify multiple labels with commas: label1=value1,label2=value2)
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux create](flux_create.md)	 - Create or update sources and resources

