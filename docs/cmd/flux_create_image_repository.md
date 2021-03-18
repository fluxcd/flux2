## flux create image repository

Create or update an ImageRepository object

### Synopsis

The create image repository command generates an ImageRepository resource.
An ImageRepository object specifies an image repository to scan.

```
flux create image repository [name] [flags]
```

### Examples

```
  # Create an ImageRepository object to scan the alpine image repository:
  flux create image repository alpine-repo --image alpine --interval 20m

  # Create an image repository that uses an image pull secret (assumed to
  # have been created already):
  flux create image repository myapp-repo \
    --secret-ref image-pull \
    --image ghcr.io/example.com/myapp --interval 5m

  # Create a TLS secret for a local image registry using a self-signed
  # host certificate, and use it to scan an image. ca.pem is a file
  # containing the CA certificate used to sign the host certificate.
  flux create secret tls local-registry-cert --ca-file ./ca.pem
  flux create image repository app-repo \
    --cert-secret-ref local-registry-cert \
    --image local-registry:5000/app --interval 5m

  # Create a TLS secret with a client certificate and key, and use it
  # to scan a private image registry.
  flux create secret tls client-cert \
    --cert-file client.crt --key-file client.key
  flux create image repository app-repo \
    --cert-secret-ref client-cert \
    --image registry.example.com/private/app --interval 5m

```

### Options

```
      --cert-ref string         the name of a secret to use for TLS certificates
  -h, --help                    help for repository
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

* [flux create image](flux_create_image.md)	 - Create or update resources dealing with image automation

