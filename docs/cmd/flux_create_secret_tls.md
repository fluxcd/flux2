---
title: "flux create secret tls command"
---
## flux create secret tls

Create or update a Kubernetes secret with TLS certificates

### Synopsis


The create secret tls command generates a Kubernetes secret with certificates for use with TLS.

```
flux create secret tls [name] [flags]
```

### Examples

```

  # Create a TLS secret on disk and encrypt it with Mozilla SOPS.
  # Files are expected to be PEM-encoded.
  flux create secret tls certs \
    --namespace=my-namespace \
    --cert-file=./client.crt \
    --key-file=./client.key \
    --export > certs.yaml

  sops --encrypt --encrypted-regex '^(data|stringData)$' \
    --in-place certs.yaml

```

### Options

```
      --ca-file string     TLS authentication CA file path
      --cert-file string   TLS authentication cert file path
  -h, --help               help for tls
      --key-file string    TLS authentication key file path
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

* [flux create secret](/cmd/flux_create_secret/)	 - Create or update Kubernetes secrets

