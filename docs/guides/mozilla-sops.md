# Manage Kubernetes secrets with Mozilla SOPS

In order to store secrets safely in a public or private Git repository, you can use
Mozilla's [SOPS](https://github.com/mozilla/sops) CLI to encrypt 
Kubernetes secrets with OpenPGP, AWS KMS, GCP KMS and Azure Key Vault.

## Prerequisites

To follow this guide you'll need a Kubernetes cluster with the GitOps 
toolkit controllers installed on it.
Please see the [get started guide](../get-started/index.md)
or the [installation guide](installation.md).

Install [gnupg](https://www.gnupg.org/) and [sops](https://github.com/mozilla/sops):

```sh
brew install gnupg sops
```

## Generate a GPG key

Generate a GPG key with OpenPGP without specifying a passphrase:

```console
$ gpg --full-generate-key

Real name: stefanprodan
Email address: stefanprodan@users.noreply.github.com
Comment: 
You selected this USER-ID:
    "stefanprodan <stefanprodan@users.noreply.github.com>"
```

Retrieve the GPG key ID (second row of the sec column):

```console
$ gpg --list-secret-keys stefanprodan@users.noreply.github.com

sec   rsa3072 2020-09-06 [SC]
      1F3D1CED2F865F5E59CA564553241F147E7C5FA4
```

Export the public and private keypair from your local GPG keyring and
create a Kubernetes secret named `sops-gpg` in the `gotk-system` namespace:

```sh
gpg --export-secret-keys \
--armor 1F3D1CED2F865F5E59CA564553241F147E7C5FA4 |
kubectl create secret generic sops-gpg \
--namespace=gotk-system \
--from-file=sops.asc=/dev/stdin 
```

## Encrypt secrets

Generate a Kubernetes secret manifest with kubectl:

```sh
kubectl -n default create secret generic basic-auth \
--from-literal=user=admin \
--from-literal=password=change-me \
--dry-run=client \
-o yaml > basic-auth.yaml
```

Encrypt the secret with sops using your GPG key:

```sh
sops --encrypt \
--pgp=1F3D1CED2F865F5E59CA564553241F147E7C5FA4 \
--encrypted-regex '^(data|stringData)$' \
--in-place basic-auth.yaml
```

!!! hint
    Note that you should encrypt only the `data` section. Encrypting the Kubernetes
    secret metadata, kind or apiVersion is not supported by kustomize-controller.

You can now commit the encrypted secret to your Git repository.

## Configure secrets decryption

Registry the Git repository on your cluster:

```sh
gotk create source git my-secrets \
--url=https://github.com/my-org/my-secrets
```

Create a kustomization for reconciling the secrets on the cluster:

```sh
gotk create kustomization my-secrets \
--source=my-secrets \
--prune=true \
--interval=10m \
--decryption-provider=sops \
--decryption-secret=sops-gpg
```

Note that the `sops-gpg` can contain more than one key, sops will try to decrypt the
secrets by iterating over all the private keys until it finds one that works.

!!! hint KMS
    When using AWS/GCP KMS or Azure Key Vault, you'll have to bind an IAM Role
    with read access to the KMS keys to the `default` service account of the
    `gotk-system` namespace for kustomize-controller to be able to fetch
    keys from KMS.

## GitOps workflow

A cluster admin should create the Kubernetes secret with the PGP keys on each cluster and
add the GitRepository/Kustomization manifests to the fleet repository.

Git repository manifest:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1alpha1
kind: GitRepository
metadata:
  name: my-secrets
  namespace: gotk-system
spec:
  interval: 1m
  url: https://github.com/my-org/my-secrets
```

Kustomization manifest:

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1alpha1
kind: Kustomization
metadata:
  name: my-secrets
  namespace: gotk-system
spec:
  interval: 10m0s
  sourceRef:
    kind: GitRepository
    name: my-secrets
  path: ./
  prune: true
  decryption:
    provider: sops
    secretRef:
      name: sops-gpg
```

!!! hint
    You can generate the above manifests using `gotk create <kind> --export > manifest.yaml`.

Assuming a team member wants to deploy an application that needs to connect
to a database using a username and password, they'll be doing the following:

* create a Kubernetes Secret manifest locally with the db credentials e.g. `db-auth.yaml`
* encrypt the secret `data` field with sops
* create a Kubernetes Deployment manifest for the app e.g. `app-deployment.yaml`
* add the Secret to the Deployment manifest as a [volume mount or env var](https://kubernetes.io/docs/concepts/configuration/secret/#using-secrets)
* commit the manifests `db-auth.yaml` and `app-deployment.yaml` to a Git repository that's being synced by the GitOps toolkit controllers

Once the manifests have been pushed to the Git repository, the following happens:

* source-controller pulls the changes from Git
* kustomize-controller loads the GPG keys from the `sops-pgp` secret 
* kustomize-controller decrypts the Kubernetes secrets with sops and applies them on the cluster
* kubelet creates the pods and mounts the secret as a volume or env variable inside the app container
