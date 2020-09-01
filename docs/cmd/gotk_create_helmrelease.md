## gotk create helmrelease

Create or update a HelmRelease resource

### Synopsis

The helmrelease create command generates a HelmRelease resource for a given HelmRepository source.

```
gotk create helmrelease [name] [flags]
```

### Examples

```
  # Create a HelmRelease from a source
  gotk create hr podinfo \
    --interval=10m \
    --release-name=podinfo \
    --target-namespace=default \
    --source=podinfo \
    --chart-name=podinfo \
    --chart-version=">4.0.0"

  # Create a HelmRelease with values for a local YAML file
  gotk create hr podinfo \
    --target-namespace=default \
    --source=podinfo \
    --chart-name=podinfo \
    --chart-version=4.0.5 \
    --values=./my-values.yaml

  # Create a HelmRelease definition on disk without applying it on the cluster
  gotk create hr podinfo \
    --target-namespace=default \
    --source=podinfo \
    --chart-name=podinfo \
    --chart-version=4.0.5 \
    --values=./values.yaml \
    --export > podinfo-release.yaml

```

### Options

```
      --chart-name string         Helm chart name
      --chart-version string      Helm chart version, accepts semver range
      --depends-on stringArray    HelmReleases that must be ready before this release can be installed
  -h, --help                      help for helmrelease
      --release-name string       name used for the Helm release, defaults to a composition of '<target-namespace>-<hr-name>'
      --source string             HelmRepository name
      --target-namespace string   namespace to install this release, defaults to the HelmRelease namespace
      --values string             local path to the values.yaml file
```

### Options inherited from parent commands

```
      --export              export in YAML format to stdout
      --interval duration   source sync interval (default 1m0s)
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
      --namespace string    the namespace scope for this operation (default "gitops-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk create](gotk_create.md)	 - Create or update sources and resources

