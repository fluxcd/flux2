---
title: "flux create tenant command"
---
## flux create tenant

Create or update a tenant

### Synopsis

The create tenant command generates namespaces, service accounts and role bindings to limit the
reconcilers scope to the tenant namespaces.

```
flux create tenant [flags]
```

### Examples

```
  # Create a tenant with access to a namespace 
  flux create tenant dev-team \
    --with-namespace=frontend \
    --label=environment=dev

  # Generate tenant namespaces and role bindings in YAML format
  flux create tenant dev-team \
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

* [flux create](/cmd/flux_create/)	 - Create or update sources and resources

