## flux suspend

Suspend resources

### Synopsis

The suspend sub-commands suspend the reconciliation of a resource.

### Options

```
  -h, --help   help for suspend
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
* [flux suspend alert](flux_suspend_alert.md)	 - Suspend reconciliation of Alert
* [flux suspend auto](flux_suspend_auto.md)	 - Suspend automation objects
* [flux suspend helmrelease](flux_suspend_helmrelease.md)	 - Suspend reconciliation of HelmRelease
* [flux suspend kustomization](flux_suspend_kustomization.md)	 - Suspend reconciliation of Kustomization
* [flux suspend receiver](flux_suspend_receiver.md)	 - Suspend reconciliation of Receiver
* [flux suspend source](flux_suspend_source.md)	 - Suspend sources

