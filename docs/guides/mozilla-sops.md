# Manage Kubernetes secrets with Mozilla SOPS

In order to store secrets safely in a public or private Git repository, you can use
Mozilla's [SOPS](https://github.com/mozilla/sops) CLI to encrypt
Kubernetes secrets with OpenPGP, AWS KMS, GCP KMS and Azure Key Vault.

## Prerequisites

To follow this guide you'll need a Kubernetes cluster with the GitOps
toolkit controllers installed on it.
Please see the [get started guide](../get-started/index.md)
or the [installation guide](installation.md).

Install [gnupg](https://www.gnupg.org/) and [SOPS](https://github.com/mozilla/sops):

```sh
brew install gnupg sops
```

## Generate a GPG key

Generate a GPG/OpenPGP key with no passphrase (`%no-protection`):

```sh
export KEY_NAME="cluster0.yourdomain.com"
export KEY_COMMENT="flux secrets"

gpg --batch --full-generate-key <<EOF
%no-protection
Key-Type: 1
Key-Length: 4096
Subkey-Type: 1
Subkey-Length: 4096
Expire-Date: 0
Name-Comment: ${KEY_COMMENT}
Name-Real: ${KEY_NAME}
EOF
```

The above configuration creates an rsa4096 key that does not expire.
For a full list of options to consider for your environment, see
[Unattended GPG key generation](https://www.gnupg.org/documentation/manuals/gnupg/Unattended-GPG-key-generation.html).

Retrieve the GPG key fingerprint (second row of the sec column):

```sh
gpg --list-secret-keys "${KEY_NAME}"

sec   rsa4096 2020-09-06 [SC]
      1F3D1CED2F865F5E59CA564553241F147E7C5FA4
```

Store the key fingerprint as an environment variable:

```sh
export KEY_FP=1F3D1CED2F865F5E59CA564553241F147E7C5FA4
```

Export the public and private keypair from your local GPG keyring and
create a Kubernetes secret named `sops-gpg` in the `flux-system` namespace:

```sh
gpg --export-secret-keys --armor "${KEY_FP}" |
kubectl create secret generic sops-gpg \
--namespace=flux-system \
--from-file=sops.asc=/dev/stdin
```

It's a good idea to back up this secret-key/K8s-Secret with a password manager or offline storage.
Also consider deleting the secret decryption key from you machine:

```sh
gpg --delete-secret-keys "${KEY_FP}"
```

## Configure in-cluster secrets decryption

Register the Git repository on your cluster:

```sh
flux create source git my-secrets \
--url=https://github.com/my-org/my-secrets
```

Create a kustomization for reconciling the secrets on the cluster:

```sh
flux create kustomization my-secrets \
--source=my-secrets \
--path=./clusters/cluster0 \
--prune=true \
--interval=10m \
--decryption-provider=sops \
--decryption-secret=sops-gpg
```

Note that the `sops-gpg` can contain more than one key, SOPS will try to decrypt the
secrets by iterating over all the private keys until it finds one that works.

## Optional: Export the public key into the Git directory

Commit the public key to the repository so that team members who clone the repo can encrypt new files:

```sh
gpg --export --armor "${KEY_FP}" > ./clusters/cluster0/.sops.pub.asc
```

Check the file contents to ensure it's the public key before adding it to the repo and committing.

```sh
git add ./clusters/cluster0/.sops.pub.asc
git commit -am 'Share GPG public key for secrets generation'
```

Team members can then import this key when they pull the Git repository:

```sh
gpg --import ./clusters/cluster0/.sops.pub.asc
```

!!! hint
    The public key is sufficient for creating brand new files.
    The secret key is required for decrypting and editing existing files because SOPS computes a MAC on all values.
    When using solely the public key to add or remove a field, the whole file should be deleted and recreated.

## Configure the Git directory for encryption

Write a [SOPS config file](https://github.com/mozilla/sops#using-sops-yaml-conf-to-select-kms-pgp-for-new-files)
to the specific cluster or namespace directory used
to store encrypted objects with this particular GPG key's fingerprint.

```yaml
cat <<EOF > ./clusters/cluster0/.sops.yaml
creation_rules:
  - path_regex: .*.yaml
    encrypted_regex: ^(data|stringData)$
    pgp: ${KEY_FP}
EOF
```

This config applies recursively to all sub-directories.
Multiple directories can use separate SOPS configs.
Contributors using the `sops` CLI to create and encrypt files
won't have to worry about specifying the proper key for the target cluster or namespace.

`encrypted_regex` helps encrypt the `data` and `stringData` fields for Secrets.
You may wish to add other fields if you are encrypting other types of Objects.

!!! hint
    Note that you should encrypt only the `data` or `stringData` section. Encrypting the Kubernetes
    secret metadata, kind or apiVersion is not supported by kustomize-controller.

## Encrypt secrets

Generate a Kubernetes secret manifest with kubectl:

```sh
kubectl -n default create secret generic basic-auth \
--from-literal=user=admin \
--from-literal=password=change-me \
--dry-run=client \
-o yaml > basic-auth.yaml
```

Encrypt the secret with SOPS using your GPG key:

```sh
sops --encrypt --in-place basic-auth.yaml
```

You can now commit the encrypted secret to your Git repository.

!!! hint
    Note that you shouldn't apply the encrypted secrets onto the cluster with kubectl.
    SOPS encrypted secrets are designed to be consumed by kustomize-controller.

### Using various cloud providers

When using AWS/GCP KMS, you don't have to include the gpg `secretRef` under
`spec.provider` (you can skip the `--decryption-secret` flag when running `flux create kustomization`),
instead you'll have to bind an IAM Role with access to the KMS
keys to the `kustomize-controller` service account of the `flux-system` namespace for
kustomize-controller to be able to fetch keys from KMS.

#### AWS 

Enabled the [IAM OIDC provider](https://eksctl.io/usage/iamserviceaccounts/) on your EKS cluster:

```sh
eksctl utils associate-iam-oidc-provider --cluster=<clusterName>
```

Create an IAM Role with access to AWS KMS e.g.:

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

Bind the IAM role to the `kustomize-controller` service account:

```sh
eksctl create iamserviceaccount \
--override-existing-serviceaccounts \
--name=kustomize-controller \
--namespace=flux-system \
--attach-policy-arn=<policyARN> \
--cluster=<clusterName>
```

Restart kustomize-controller for the binding to take effect:

```sh
kubectl -n flux-system rollout restart deployment/kustomize-controller
```

#### Azure

When using Azure Key Vault you need to authenticate kustomize-controller either with [aad-pod-identity](../use-cases/azure.md#aad-pod-identity)
or by passing [Service Principal credentials as environment variables](https://github.com/mozilla/sops#encrypting-using-azure-key-vault).

Create the Azure Key-Vault:

```sh
export VAULT_NAME="fluxcd-$(uuidgen | tr -d - | head -c 16)"
export KEY_NAME="sops-cluster0"

az keyvault create --name "${VAULT_NAME}"
az keyvault key create --name "${KEY_NAME}" \
  --vault-name "${VAULT_NAME}"
  --protection software \
  --ops encrypt decrypt
az keyvault key show --name "${KEY_NAME}" \
  --vault-name "${VAULT_NAME}" \
  --query key.kid
```

If using AAD Pod-Identity, create an identity within Azure to bind against, then create an `AzureIdentity` object to match:

```yaml
# Create an identity in Azure and assign it a role to access Key Vault  (note: the identity's resourceGroup should match the desired Key Vault):
#     az identity create -n sops-akv-decryptor
#     az role assignment create --role "Key Vault Crypto User" --assignee-object-id "$(az identity show -n sops-akv-decryptor -o tsv --query principalId)"
# Fetch the clientID and resourceID to configure the AzureIdentity spec below:
#     az identity show -n sops-akv-decryptor -otsv --query clientId
#     az identity show -n sops-akv-decryptor -otsv --query resourceId
---
apiVersion: aadpodidentity.k8s.io/v1
kind: AzureIdentity
metadata:
  name: sops-akv-decryptor  # kustomize-controller label will match this name
  namespace: flux-system
spec:
  clientID: 58027844-6b86-424b-9888-b5ae2dc28b4f
  resourceID: /subscriptions/8c69185e-55f9-4d00-8e71-a1b1bb1386a1/resourcegroups/stealthybox/providers/Microsoft.ManagedIdentity/userAssignedIdentities/sops-akv-decryptor
  type: 0  # user-managed identity
```

[Customize your Flux Manifests](../guides/installation.md#customize-flux-manifests) so that kustomize-controller has the proper credentials.
Patch the kustomize-controller Pod template so that the label matches the `AzureIdentity` name.
Additionally, the SOPS specific environment variable `AZURE_AUTH_METHOD=msi` to activate the proper auth method within kustomize-controller.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kustomize-controller
  namespace: flux-system
spec:
  template:
    metadata:
      labels:
        aadpodidbinding: sops-akv-decryptor  # match the AzureIdentity name
    spec:
      containers:
      - name: manager
        env:
        - name: AZURE_AUTH_METHOD
          value: msi
```

Alternatively, if using a Service Principal stored in a K8s Secret, patch the Pod's envFrom
to reference the `AZURE_TENANT_ID`/`AZURE_CLIENT_ID`/`AZURE_CLIENT_SECRET`
fields from your Secret.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kustomize-controller
  namespace: flux-system
spec:
  template:
    spec:
      containers:
      - name: manager
        envFrom:
        - secretRef:
            name: sops-akv-decryptor-service-principal
```

At this point, kustomize-controller is now authorized to decrypt values in
SOPS encrypted files from your Sources via the related Key Vault.

See Mozilla's guide to
[Encrypting Using Azure Key Vault](https://github.com/mozilla/sops#encrypting-using-azure-key-vault)
to get started committing encrypted files to your Git Repository or other Sources.

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
* kustomize-controller decrypts the Kubernetes secrets with SOPS and applies them on the cluster
* kubelet creates the pods and mounts the secret as a volume or env variable inside the app container
