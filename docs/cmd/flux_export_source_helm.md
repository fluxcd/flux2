---
title: "flux export source helm command"
---
## flux export source helm

Export HelmRepository sources in YAML format

### Synopsis

The export source git command exports one or all HelmRepository sources in YAML format.

```
flux export source helm [name] [flags]
```

### Examples

```
  # Export all HelmRepository sources
  flux export source helm --all > sources.yaml

  # Export a HelmRepository source including the basic auth credentials
  flux export source helm my-private-repo --with-credentials > source.yaml

```

### Options

```
  -h, --help   help for helm
```

### Options inherited from parent commands

```
      --all                 select all resources
      --context string      kubernetes context to use
      --kubeconfig string   absolute path to the kubeconfig file
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
      --with-credentials    include credential secrets
```

### SEE ALSO

* [flux export source](/cmd/flux_export_source/)	 - Export sources

