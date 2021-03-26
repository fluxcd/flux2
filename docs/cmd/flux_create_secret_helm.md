---
title: "flux create secret helm command"
---
## flux create secret helm

Create or update a Kubernetes secret for Helm repository authentication

### Synopsis

The create secret helm command generates a Kubernetes secret with basic authentication credentials.

```
flux create secret helm [name] [flags]
```

### Examples

```
 # Create a Helm authentication secret on disk and encrypt it with Mozilla SOPS
  flux create secret helm repo-auth \
    --namespace=my-namespace \
    --username=my-username \
    --password=my-password \
    --export > repo-auth.yaml

  sops --encrypt --encrypted-regex '^(data|stringData)$' \
    --in-place repo-auth.yaml

  # Create a Helm authentication secret using a custom TLS cert
  flux create secret helm repo-auth \
    --username=username \
    --password=password \
    --cert-file=./cert.crt \
    --key-file=./key.crt \
    --ca-file=./ca.crt
```

### Options

```
      --ca-file string     TLS authentication CA file path
      --cert-file string   TLS authentication cert file path
  -h, --help               help for helm
      --key-file string    TLS authentication key file path
  -p, --password string    basic authentication password
  -u, --username string    basic authentication username
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

* [flux create secret](../flux_create_secret/)	 - Create or update Kubernetes secrets

