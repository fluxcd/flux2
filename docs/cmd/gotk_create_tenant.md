## gotk create tenant

Create or update a tenant

### Synopsis


The create tenant command generates namespaces and role bindings to limit the
reconcilers scope to the tenant namespaces.

```
gotk create tenant [flags]
```

### Examples

```
  # Create a tenant with access to a namespace 
  gotk create tenant dev-team \
    --with-namespace=frontend \
    --label=environment=dev

  # Generate tenant namespaces and role bindings in YAML format
  gotk create tenant dev-team \
    --with-namespace=frontend \
    --with-namespace=backend \
	--export > dev-team.yaml

```

### Options

```
      --cluster-role string      cluster role of the tenant role binding (default "cluster-admin")
  -h, --help                     help for tenant
      --with-namespace strings   namespace belonging to this tenant
```

### Options inherited from parent commands

```
      --export              export in YAML format to stdout
      --interval duration   source sync interval (default 1m0s)
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
      --label strings       set labels on the resource (can specify multiple labels with commas: label1=value1,label2=value2)
  -n, --namespace string    the namespace scope for this operation (default "gitops-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk create](gotk_create.md)	 - Create or update sources and resources

