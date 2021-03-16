## flux logs

Display formatted logs for toolkit components

### Synopsis

The logs command displays formatted logs from various toolkit components.

```
flux logs [flags]
```

### Examples

```
# Get logs from toolkit components
	 flux logs

	# Stream logs from toolkit components
	flux logs --follow
 
	# Get logs from toolkit components in a particular namespace
	flux logs --flux-namespace my-namespace

	# Get logs for a particular log level
	flux logs --level=info

	# Filter logs by kind, name, or namespace
	flux logs --kind=kustomization --name podinfo --namespace default
    
```

### Options

```
  -A, --all-namespaces          displays logs for objects across all namespaces
      --flux-namespace string   the namespace where the Flux components are running. (default "flux-system")
  -f, --follow                  Specifies if the logs should be streamed
  -h, --help                    help for logs
      --kind string             displays errors of a particular toolkit kind e.g GitRepository
      --level logLevel          log level, available options are: (debug, info, error)
      --name string             specifies the name of the object logs to be displayed
      --tail int                lines of recent log file to display (default -1)
```

### Options inherited from parent commands

```
      --context string      kubernetes context to use
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux](flux.md)	 - Command line utility for assembling Kubernetes CD pipelines

