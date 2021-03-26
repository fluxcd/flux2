---
title: "flux create alert command"
---
## flux create alert

Create or update a Alert resource

### Synopsis

The create alert command generates a Alert resource.

```
flux create alert [name] [flags]
```

### Examples

```
  # Create an Alert for kustomization events
  flux create alert \
  --event-severity info \
  --event-source Kustomization/flux-system \
  --provider-ref slack \
  flux-system
```

### Options

```
      --event-severity string      severity of events to send alerts for
      --event-source stringArray   sources that should generate alerts (<kind>/<name>)
  -h, --help                       help for alert
      --provider-ref string        reference to provider
```

### Options inherited from parent commands

```
      --context string      kubernetes context to use
      --export              export in YAML format to stdout
      --interval duration   source sync interval (default 1m0s)
      --kubeconfig string   absolute path to the kubeconfig file
      --label strings       set labels on the resource (can specify multiple labels with commas: label1=value1,label2=value2)
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux create](../flux_create/)	 - Create or update sources and resources

