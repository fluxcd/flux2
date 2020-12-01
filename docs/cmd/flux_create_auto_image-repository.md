## flux create auto image-repository

Create or update an ImageRepository object

### Synopsis

The create auto image-repository command generates an ImageRepository resource.
An ImageRepository object specifies an image repository to scan.

```
flux create auto image-repository <name> [flags]
```

### Options

```
  -h, --help                    help for image-repository
      --image string            the image repository to scan; e.g., library/alpine
      --scan-timeout duration   a timeout for scanning; this defaults to the interval if not set
      --secret-ref string       the name of a docker-registry secret to use for credentials
```

### Options inherited from parent commands

```
      --context string      kubernetes context to use
      --export              export in YAML format to stdout
      --interval duration   source sync interval (default 1m0s)
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
      --label strings       set labels on the resource (can specify multiple labels with commas: label1=value1,label2=value2)
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux create auto](flux_create_auto.md)	 - Create or update resources dealing with automation

