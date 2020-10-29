## gotk uninstall

Uninstall the toolkit components

### Synopsis

The uninstall command removes the namespace, cluster roles, cluster role bindings and CRDs from the cluster.

```
gotk uninstall [flags]
```

### Examples

```
  # Dry-run uninstall of all components
  gotk uninstall --dry-run --namespace=flux-system

  # Uninstall all components and delete custom resource definitions
  gotk uninstall --resources --crds --namespace=flux-system

```

### Options

```
      --crds        removes all CRDs previously installed
      --dry-run     only print the object that would be deleted
  -h, --help        help for uninstall
      --resources   removes custom resources such as Kustomizations, GitRepositories and HelmRepositories (default true)
  -s, --silent      delete components without asking for confirmation
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk](gotk.md)	 - Command line utility for assembling Kubernetes CD pipelines

