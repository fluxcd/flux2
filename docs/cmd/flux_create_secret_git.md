## flux create secret git

Create or update a Kubernetes secret for Git authentication

### Synopsis


The create secret git command generates a Kubernetes secret with Git credentials.
For Git over SSH, the host and SSH keys are automatically generated and stored in the secret.
For Git over HTTP/S, the provided basic authentication credentials are stored in the secret.

```
flux create secret git [name] [flags]
```

### Examples

```
  # Create a Git SSH authentication secret using an ECDSA P-521 curve public key

  flux create secret git podinfo-auth \
    --url=ssh://git@github.com/stefanprodan/podinfo \
    --ssh-key-algorithm=ecdsa \
    --ssh-ecdsa-curve=p521

  # Create a secret for a Git repository using basic authentication
  flux create secret git podinfo-auth \
    --url=https://github.com/stefanprodan/podinfo \
    --username=username \
    --password=password

  # Create a Git SSH secret on disk and print the deploy key
  flux create secret git podinfo-auth \
    --url=ssh://git@github.com/stefanprodan/podinfo \
	--export > podinfo-auth.yaml

  yq read podinfo-auth.yaml 'data."identity.pub"' | base64 --decode

  # Create a Git SSH secret on disk and encrypt it with Mozilla SOPS
  flux create secret git podinfo-auth \
    --namespace=apps \
    --url=ssh://git@github.com/stefanprodan/podinfo \
	--export > podinfo-auth.yaml

  sops --encrypt --encrypted-regex '^(data|stringData)$' \
    --in-place podinfo-auth.yaml

```

### Options

```
  -h, --help                                   help for git
  -p, --password string                        basic authentication password
      --ssh-ecdsa-curve ecdsaCurve             SSH ECDSA public key curve (p256, p384, p521) (default p384)
      --ssh-key-algorithm publicKeyAlgorithm   SSH public key algorithm (rsa, ecdsa, ed25519) (default rsa)
      --ssh-rsa-bits rsaKeyBits                SSH RSA public key bit size (multiplies of 8) (default 2048)
      --url string                             git address, e.g. ssh://git@host/org/repository
  -u, --username string                        basic authentication username
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

* [flux create secret](flux_create_secret.md)	 - Create or update Kubernetes secrets

