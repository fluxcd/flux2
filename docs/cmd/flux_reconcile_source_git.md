---
title: "flux reconcile source git command"
---
## flux reconcile source git

Reconcile a GitRepository source

### Synopsis

The reconcile source command triggers a reconciliation of a GitRepository resource and waits for it to finish.

```
flux reconcile source git [name] [flags]
```

### Examples

```
  # Trigger a git pull for an existing source
  flux reconcile source git podinfo
```

### Options

```
  -h, --help   help for git
```

### Options inherited from parent commands

```
      --context string      kubernetes context to use
      --kubeconfig string   absolute path to the kubeconfig file
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux reconcile source](/cmd/flux_reconcile_source/)	 - Reconcile sources

