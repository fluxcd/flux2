## tk create source helm

Create or update a HelmRepository source

### Synopsis


The create source helm command generates a HelmRepository resource and waits for it to fetch the index.
For private Helm repositories, the basic authentication credentials are stored in a Kubernetes secret.

```
tk create source helm [name] [flags]
```

### Examples

```
  # Create a source from a public Helm repository
  tk create source helm podinfo \
    --url=https://stefanprodan.github.io/podinfo \
    --interval=10m

  # Create a source from a Helm repository using basic authentication
  tk create source helm podinfo \
    --url=https://stefanprodan.github.io/podinfo \
    --username=username \
    --password=password

```

### Options

```
  -h, --help              help for helm
  -p, --password string   basic authentication password
      --url string        Helm repository address
  -u, --username string   basic authentication username
```

### Options inherited from parent commands

```
      --export              export in YAML format to stdout
      --interval duration   source sync interval (default 1m0s)
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
      --namespace string    the namespace scope for this operation (default "gitops-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [tk create source](tk_create_source.md)	 - Create or update sources

