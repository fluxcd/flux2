## gotk create kustomization

Create or update a Kustomization resource

### Synopsis

The kustomization source create command generates a Kustomize resource for a given GitRepository source.

```
gotk create kustomization [name] [flags]
```

### Examples

```
  # Create a Kustomization resource from a source at a given path
  gotk create kustomization contour \
    --source=contour \
    --path="./examples/contour/" \
    --prune=true \
    --interval=10m \
    --validation=client \
    --health-check="Deployment/contour.projectcontour" \
    --health-check="DaemonSet/envoy.projectcontour" \
    --health-check-timeout=3m

  # Create a Kustomization resource that depends on the previous one
  gotk create kustomization webapp \
    --depends-on=contour \
    --source=webapp \
    --path="./deploy/overlays/dev" \
    --prune=true \
    --interval=5m \
    --validation=client

  # Create a Kustomization resource that runs under a service account
  gotk create kustomization webapp \
    --source=webapp \
    --path="./deploy/overlays/staging" \
    --prune=true \
    --interval=5m \
    --validation=client \
    --sa-name=reconclier \
    --sa-namespace=staging

```

### Options

```
      --decryption-provider string      enables secrets decryption, provider can be 'sops'
      --decryption-secret string        set the Kubernetes secret name that contains the OpenPGP private keys used for sops decryption
      --depends-on stringArray          Kustomization that must be ready before this Kustomization can be applied
      --health-check stringArray        workload to be included in the health assessment, in the format '<kind>/<name>.<namespace>'
      --health-check-timeout duration   timeout of health checking operations (default 2m0s)
  -h, --help                            help for kustomization
      --path string                     path to the directory containing the Kustomization file (default "./")
      --prune                           enable garbage collection
      --sa-name string                  service account name
      --sa-namespace string             service account namespace
      --source string                   GitRepository name
      --validation string               validate the manifests before applying them on the cluster, can be 'client' or 'server'
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

* [gotk create](gotk_create.md)	 - Create or update sources and resources

