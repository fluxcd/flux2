---
title: "flux create image command"
---
## flux create image

Create or update resources dealing with image automation

### Synopsis

The create image sub-commands work with image automation objects; that is,
object controlling updates to git based on e.g., new container images
being available.

### Options

```
  -h, --help   help for image
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
* [flux create image policy](../flux_create_image_policy/)	 - Create or update an ImagePolicy object
* [flux create image repository](../flux_create_image_repository/)	 - Create or update an ImageRepository object
* [flux create image update](../flux_create_image_update/)	 - Create or update an ImageUpdateAutomation object

