## flux create receiver

Create or update a Receiver resource

### Synopsis

The create receiver command generates a Receiver resource.

```
flux create receiver [name] [flags]
```

### Examples

```
  # Create a Receiver
  flux create receiver github-receiver \
	--type github \
	--event ping \
	--event push \
	--secret-ref webhook-token \
	--resource GitRepository/webapp \
	--resource HelmRepository/webapp

```

### Options

```
      --event stringArray      
  -h, --help                   help for receiver
      --resource stringArray   
      --secret-ref string      
      --type string            
```

### Options inherited from parent commands

```
      --export              export in YAML format to stdout
      --interval duration   source sync interval (default 1m0s)
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
      --label strings       set labels on the resource (can specify multiple labels with commas: label1=value1,label2=value2)
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux create](flux_create.md)	 - Create or update sources and resources

