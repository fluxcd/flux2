## gotk check

Check requirements and installation

### Synopsis

The check command will perform a series of checks to validate that
the local environment is configured correctly and if the installed components are healthy.

```
gotk check [flags]
```

### Examples

```
  # Run pre-installation checks
  gotk check --pre

  # Run installation checks
  gotk check

```

### Options

```
      --components strings   list of components, accepts comma-separated values (default [source-controller,kustomize-controller,helm-controller,notification-controller])
  -h, --help                 help for check
      --pre                  only run pre-installation checks
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "gotk-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk](gotk.md)	 - Command line utility for assembling Kubernetes CD pipelines

