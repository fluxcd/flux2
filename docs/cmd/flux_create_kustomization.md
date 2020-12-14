## flux create kustomization

Create or update a Kustomization resource

### Synopsis

The kustomization source create command generates a Kustomize resource for a given source.

```
flux create kustomization [name] [flags]
```

### Examples

```
  # Create a Kustomization resource from a source at a given path
  flux create kustomization contour \
    --source=contour \
    --path="./examples/contour/" \
    --prune=true \
    --interval=10m \
    --validation=client \
    --health-check="Deployment/contour.projectcontour" \
    --health-check="DaemonSet/envoy.projectcontour" \
    --health-check-timeout=3m

  # Create a Kustomization resource that depends on the previous one
  flux create kustomization webapp \
    --depends-on=contour \
    --source=webapp \
    --path="./deploy/overlays/dev" \
    --prune=true \
    --interval=5m \
    --validation=client

  # Create a Kustomization resource that references a Bucket
  flux create kustomization secrets \
    --source=Bucket/secrets \
    --prune=true \
    --interval=5m

```

### Options

```
      --decryption-provider decryptionProvider   decryption provider, available options are: (sops)
      --decryption-secret string                 set the Kubernetes secret name that contains the OpenPGP private keys used for sops decryption
      --depends-on stringArray                   Kustomization that must be ready before this Kustomization can be applied, supported formats '<name>' and '<namespace>/<name>'
      --health-check stringArray                 workload to be included in the health assessment, in the format '<kind>/<name>.<namespace>'
      --health-check-timeout duration            timeout of health checking operations (default 2m0s)
  -h, --help                                     help for kustomization
      --path safeRelativePath                    path to the directory containing a kustomization.yaml file (default ./)
      --prune                                    enable garbage collection
      --service-account string                   the name of the service account to impersonate when reconciling this Kustomization
      --source kustomizationSource               source that contains the Kubernetes manifests in the format '[<kind>/]<name>', where kind must be one of: (GitRepository, Bucket), if kind is not specified it defaults to GitRepository
      --target-namespace string                  overrides the namespace of all Kustomization objects reconciled by this Kustomization
      --validation string                        validate the manifests before applying them on the cluster, can be 'client' or 'server'
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

* [flux create](flux_create.md)	 - Create or update sources and resources

