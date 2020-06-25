## tk create source git

Create or update a GitRepository source

### Synopsis


The create source git command generates a GitRepository resource and waits for it to sync.
For Git over SSH, host and SSH keys are automatically generated and stored in a Kubernetes secret.
For private Git repositories, the basic authentication credentials are stored in a Kubernetes secret.

```
tk create source git [name] [flags]
```

### Examples

```
  # Create a source from a public Git repository master branch
  create source git podinfo \
    --url=https://github.com/stefanprodan/podinfo \
    --branch=master

  # Create a source from a Git repository pinned to specific git tag
  create source git podinfo \
    --url=https://github.com/stefanprodan/podinfo \
    --tag="3.2.3"

  # Create a source from a public Git repository tag that matches a semver range
  create source git podinfo \
    --url=https://github.com/stefanprodan/podinfo \
    --tag-semver=">=3.2.0 <3.3.0"

  # Create a source from a Git repository using SSH authentication
  create source git podinfo \
    --url=ssh://git@github.com/stefanprodan/podinfo \
    --branch=master

  # Create a source from a Git repository using SSH authentication and an
  # ECDSA P-521 curve public key
  create source git podinfo \
    --url=ssh://git@github.com/stefanprodan/podinfo \
    --branch=master \
    --ssh-key-algorithm=ecdsa \
    --ssh-ecdsa-curve=p521

  # Create a source from a Git repository using basic authentication
  create source git podinfo \
    --url=https://github.com/stefanprodan/podinfo \
    --username=username \
    --password=password

```

### Options

```
      --branch string                          git branch (default "master")
  -h, --help                                   help for git
  -p, --password string                        basic authentication password
      --ssh-ecdsa-curve ecdsaCurve             SSH ECDSA public key curve (p256, p384, p521) (default p384)
      --ssh-key-algorithm publicKeyAlgorithm   SSH public key algorithm (rsa, ecdsa, ed25519) (default rsa)
      --ssh-rsa-bits rsaKeyBits                SSH RSA public key bit size (multiplies of 8) (default 2048)
      --tag string                             git tag
      --tag-semver string                      git tag semver range
      --url string                             git address, e.g. ssh://git@host/org/repository
  -u, --username string                        basic authentication username
```

### Options inherited from parent commands

```
      --components strings   list of components, accepts comma-separated values (default [source-controller,kustomize-controller])
      --export               export in YAML format to stdout
      --interval duration    source sync interval (default 1m0s)
      --kubeconfig string    path to the kubeconfig file (default "~/.kube/config")
      --namespace string     the namespace scope for this operation (default "gitops-system")
      --timeout duration     timeout for this operation (default 5m0s)
      --verbose              print generated objects
```

### SEE ALSO

* [tk create source](tk_create_source.md)	 - Create or update sources

