---
title: "flux logs command"
---
## flux logs

Display formatted logs for Flux components

### Synopsis

The logs command displays formatted logs from various Flux components.

```
flux logs [flags]
```

### Examples

```
  # Print the reconciliation logs of all Flux custom resources in your cluster
	 flux logs --all-namespaces

	# Stream logs for a particular log level
	flux logs --follow --level=error --all-namespaces

	# Filter logs by kind, name and namespace
	flux logs --kind=Kustomization --name=podinfo --namespace=default

	# Print logs when Flux is installed in a different namespace than flux-system
	flux logs --flux-namespace=my-namespace
    
```

### Options

```
  -A, --all-namespaces          displays logs for objects across all namespaces
      --flux-namespace string   the namespace where the Flux components are running (default "flux-system")
  -f, --follow                  specifies if the logs should be streamed
  -h, --help                    help for logs
      --kind string             displays errors of a particular toolkit kind e.g GitRepository
      --level logLevel          log level, available options are: (debug, info, error)
      --name string             specifies the name of the object logs to be displayed
      --tail int                lines of recent log file to display (default -1)
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

* [flux](/cmd/flux/)	 - Command line utility for assembling Kubernetes CD pipelines

