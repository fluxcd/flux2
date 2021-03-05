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
create a Kubernetes secret named `sops-gpg` in the `flux-system` namespace:

```sh
gpg --export-secret-keys \
--armor 1F3D1CED2F865F5E59CA564553241F147E7C5FA4 |
kubectl create secret generic sops-gpg \
--namespace=flux-system \
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

!!! hint
    Note that you shouldn't apply the encrypted secrets onto the cluster with kubectl. SOPS encrypted secrets are designed to be consumed by kustomize-controller.

## Configure secrets decryption

Registry the Git repository on your cluster:

```sh
flux create source git my-secrets \
--url=https://github.com/my-org/my-secrets
```

Create a kustomization for reconciling the secrets on the cluster:

```sh
flux create kustomization my-secrets \
--source=my-secrets \
--prune=true \
--interval=10m \
--decryption-provider=sops \
--decryption-secret=sops-gpg
```

Note that the `sops-gpg` can contain more than one key, sops will try to decrypt the
secrets by iterating over all the private keys until it finds one that works.

### Using various cloud providers

When using AWS/GCP KMS, you don't have to include the gpg `secretRef` under
`spec.provider` (you can skip the `--decryption-secret` flag when running `flux create kustomization`),
instead you'll have to bind an IAM Role with access to the KMS
keys to the `kustomize-controller` service account of the `flux-system` namespace for
kustomize-controller to be able to fetch keys from KMS.

#### AWS 

IAM Role example:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": [
                "kms:Encrypt",
                "kms:Decrypt",
                "kms:ReEncrypt*",
                "kms:GenerateDataKey*",
                "kms:DescribeKey"
            ],
            "Effect": "Allow",
            "Resource": "arn:aws:kms:eu-west-1:XXXXX209540:key/4f581f5b-7f78-45e9-a543-83a7022e8105"
        }
    ]
}
```

#### Azure

When using Azure Key Vault you need to authenticate the kustomize controller either by passing
[Service Principal credentials as environment variables](https://github.com/mozilla/sops#encrypting-using-azure-key-vault)
or with [add-pod-identity](https://github.com/Azure/aad-pod-identity).

There are several authentication methods available in SOPS for connecting to an
Azure Key Vault. SOPS looks for specific environment variables to determine
which method to use, and then uses the credentials in those environment
variables. Please refer to the SOPS documentation to determine which
environment variables you will need to set for your preferred authentication
method.

For example, to use a service principal for authentication, you would need to
have these environment variables set for SOPS:

```
AZURE_TENANT_ID=XXX
AZURE_CLIENT_SECRET=XXX
AZURE_CLIENT_ID=XXX
```

Since SOPS is running in the kustomize-controller, these environment variables
will need to be set in the kustomize controller deployment definition.

Create a secret with the appropriate environment variables:

```sh
kubectl create secret flux-azure-service-principal \
  --namespace flux-system \
  --from-literal=AZURE_TENANT_ID="XXX" \
  --from-literal=AZURE_TENANT_ID="XXX" \
  --from-literal=AZURE_TENANT_ID="XXX"
```

You'll need a separate process from Flux for bootstrapping this specific secret
before you bootstrap Flux, or you'll end up with a dependency cycle.

Finally, update your kustomize controller deployment definition in
`flux-system/gotk-components.yaml` to mount the secret data as environment
variables:

```diff
@@ -2495,6 +2495,9 @@ spec:
           valueFrom:
             fieldRef:
               fieldPath: metadata.namespace
+        envFrom:
+        - secretRef:
+            name: flux-azure-service-principal
         image: ghcr.io/fluxcd/kustomize-controller:v0.9.1
         imagePullPolicy: IfNotPresent
         livenessProbe:
```

#### Google Cloud

Please ensure that the GKE cluster has Workload Identity enabled.

1. Create a service account with the role `Cloud KMS CryptoKey Encrypter/Decrypter`.
2. Create an IAM policy binding between the GCP service account to the `kustomize-controller` service account of the `flux-system`.
3. Annotate the `kustomize-controller` service account in the `flux-system` with the GCP service account.

```sh
kubectl annotate serviceaccount kustomize-controller \
  --namespace flux-system \
  iam.gke.io/gcp-service-account=<name-of-serviceaccount>@project-id.iam.gserviceaccount.com
```

## GitOps workflow

A cluster admin should create the Kubernetes secret with the PGP keys on each cluster and
add the GitRepository/Kustomization manifests to the fleet repository.

Git repository manifest:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: GitRepository
metadata:
  name: my-secrets
  namespace: flux-system
spec:
  interval: 1m
  url: https://github.com/my-org/my-secrets
```

Kustomization manifest:

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1beta1
kind: Kustomization
metadata:
  name: my-secrets
  namespace: flux-system
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
    You can generate the above manifests using `flux create <kind> --export > manifest.yaml`.

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
