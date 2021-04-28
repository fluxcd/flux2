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
      --all                 suspend all resources in that namespace
      --context string      kubernetes context to use
      --kubeconfig string   absolute path to the kubeconfig file
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux suspend](../flux_suspend/)	 - Suspend resources
* [flux suspend image repository](../flux_suspend_image_repository/)	 - Suspend reconciliation of an ImageRepository
* [flux suspend image update](../flux_suspend_image_update/)	 - Suspend reconciliation of an ImageUpdateAutomation

