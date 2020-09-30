# Sealed Secrets

In order to store secrets safely in a public or private Git repository, you can use
Bitnami's [sealed-secrets controller](https://github.com/bitnami-labs/sealed-secrets)
and encrypt your Kubernetes Secrets into SealedSecrets.
The sealed secrets can be decrypted only by the controller running in your cluster and
nobody else can obtain the original secret, even if they have access to the Git repository.

## Prerequisites

To follow this guide you'll need a Kubernetes cluster with the GitOps 
toolkit controllers installed on it.
Please see the [get started guide](../get-started/index.md)
or the [installation guide](installation.md).

The sealed-secrets controller comes with a companion CLI tool called kubeseal.
With kubeseal you can create SealedSecret custom resources in YAML format
and store those in your Git repository.

Install the kubeseal CLI:

```sh
brew install kubeseal
```

For Linux or Windows you can download the kubeseal binary from
[GitHub](https://github.com/bitnami-labs/sealed-secrets/releases).

## Deploy sealed-secrets with a HelmRelease

You'll be using [helm-controller](../components/helm/controller.md) APIs to install
the sealed-secrets controller from its [Helm chart](https://hub.kubeapps.com/charts/stable/sealed-secrets).

First you have to register the Helm repository where the sealed-secrets chart is published:

```sh
gotk create source helm stable \
--interval=1h \
--url=https://kubernetes-charts.storage.googleapis.com
```

With `interval` we configure [source-controller](../components/source/controller.md) to download
the Helm repository index every hour. If a newer version of sealed-secrets is published,
source-controller will signal helm-controller that a new chart is available.

Create a Helm release that installs the latest version of sealed-secrets controller:

```sh
gotk create helmrelease sealed-secrets \
--interval=1h \
--release-name=sealed-secrets \
--target-namespace=gotk-system \
--source=HelmRepository/stable \
--chart=sealed-secrets \
--chart-version="1.10.x"
```

With chart version `1.10.x` we configure helm-controller to automatically upgrade the release
when a new chart patch version is fetched by source-controller.

At startup, the sealed-secrets controller generates a 4096-bit RSA key pair and 
persists the private and public keys as Kubernetes secrets in the `gotk-system` namespace.

You can retrieve the public key with:

```sh
kubeseal --fetch-cert \
--controller-name=sealed-secrets \
--controller-namespace=gotk-system \
> pub-sealed-secrets.pem
``` 

The public key can be safely stored in Git, and can be used to encrypt secrets
without direct access to the Kubernetes cluster.

## Encrypt secrets

Generate a Kubernetes secret manifest with kubectl:

```sh
kubectl -n default create secret generic basic-auth \
--from-literal=user=admin \
--from-literal=password=change-me \
--dry-run \
-o yaml > basic-auth.yaml
```

Encrypt the secret with kubeseal:

```sh
kubeseal --format=yaml --cert=pub-sealed-secrets.pem \
< basic-auth.yaml > basic-auth-sealed.yaml
```

Delete the plain secret and apply the sealed one:

```sh
rm basic-auth.yaml
kubectl apply -f basic-auth-sealed.yaml
```

Verify that the sealed-secrets controller has created the `basic-auth` Kubernetes Secret:

```console
$ kubectl -n default get secrets basic-auth

NAME         TYPE     DATA   AGE
basic-auth   Opaque   2      1m43s
```

## GitOps workflow

A cluster admin should add the stable `HelmRepository` manifest and the sealed-secrets `HelmRelease`
to the fleet repository.

Helm repository manifest:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1alpha1
kind: HelmRepository
metadata:
  name: stable
  namespace: gotk-system
spec:
  interval: 1h0m0s
  url: https://kubernetes-charts.storage.googleapis.com
```

Helm release manifest:

```yaml
apiVersion: helm.toolkit.fluxcd.io/v2alpha1
kind: HelmRelease
metadata:
  name: sealed-secrets
  namespace: gotk-system
spec:
  chart:
    spec:
      chart: sealed-secrets
      sourceRef:
        kind: HelmRepository
        name: stable
      version: "1.10.x"
  interval: 1h0m0s
  releaseName: sealed-secrets
  targetNamespace: gotk-system
```

!!! hint
    You can generate the above manifests using `gotk create <kind> --export > manifest.yaml`.

Once the sealed-secrets controller is installed, the admin fetches the 
public key and shares it with the teams that operate on the fleet clusters via Git.

When a team member wants to create a Kubernetes Secret on a cluster,
they uses kubeseal and the public key corresponding to that cluster to generate a SealedSecret.

Assuming a team member wants to deploy an application that needs to connect
to a database using a username and password, they'll be doing the following:

* create a Kubernetes Secret manifest locally with the db credentials e.g. `db-auth.yaml`
* encrypt the secret with kubeseal as `db-auth-sealed.yaml`
* delete the original secret file `db-auth.yaml`
* create a Kubernetes Deployment manifest for the app e.g. `app-deployment.yaml`
* add the Secret to the Deployment manifest as a [volume mount or env var](https://kubernetes.io/docs/concepts/configuration/secret/#using-secrets) using the original name `db-auth`
* commit the manifests `db-auth-sealed.yaml` and `app-deployment.yaml` to a Git repository that's being synced by the GitOps toolkit controllers

Once the manifests have been pushed to the Git repository, the following happens:

* source-controller pulls the changes from Git
* kustomize-controller applies the SealedSecret and the Deployment manifests
* sealed-secrets controller decrypts the SealedSecret and creates a Kubernetes Secret
* kubelet creates the pods and mounts the secret as a volume or env variable inside the app container
