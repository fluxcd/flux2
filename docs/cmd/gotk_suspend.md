## gotk suspend

Suspend resources

### Synopsis

The suspend sub-commands suspend the reconciliation of a resource.

### Options

```
  -h, --help   help for suspend
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
* [gotk suspend alert](gotk_suspend_alert.md)	 - Suspend reconciliation of Alert
* [gotk suspend helmrelease](gotk_suspend_helmrelease.md)	 - Suspend reconciliation of HelmRelease
* [gotk suspend kustomization](gotk_suspend_kustomization.md)	 - Suspend reconciliation of Kustomization
* [gotk suspend receiver](gotk_suspend_receiver.md)	 - Suspend reconciliation of Receiver

