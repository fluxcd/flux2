---
title: "flux suspend image command"
---
## flux suspend image

Suspend image automation objects

### Synopsis

The suspend image sub-commands suspend the reconciliation of an image automation object.

### Options

```
  -h, --help   help for image
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

* [flux suspend](/cmd/flux_suspend/)	 - Suspend resources
* [flux suspend image repository](/cmd/flux_suspend_image_repository/)	 - Suspend reconciliation of an ImageRepository
* [flux suspend image update](/cmd/flux_suspend_image_update/)	 - Suspend reconciliation of an ImageUpdateAutomation

