# Flux for Helm Users

Welcome Helm users!
We think Flux's Helm Controller is the best way to do Helm according to GitOps principles, and we're dedicated to doing what we can to help you feel the same way.

## What Does Flux add to Helm?

Helm 3 was designed with both a client and an SDK, but no running software agents.
This architecture intended anything outside of the client scope to be addressed by other tools in the ecosystem, which could then make use of Helm's SDK.

Built on Kubernetes controller-runtime, Flux's Helm Controller is an example of a mature software agent that uses Helm's SDK to full effect.
<!-- Flux is the only CD project that uses Helm as a library – not shelling out to the client – and does not fork the SDK to diverge from how Helm does things. -->

Flux's biggest addition to Helm is a structured declaration layer for your releases that automatically gets reconciled to your cluster based on your configured rules:

- While the Helm client commands let you imperatively do things
- Flux Helm Custom Resources let you declare what you want the Helm SDK to do automatically

Additional benefits Flux adds to Helm include:

- Managing / structuring multiple environments
- A control loop, with configurable retry logic
- Automated drift detection between the desired and actual state of your operations
- Automated responses to that drift, including reconciliation, notifications, and unified logging

## Getting Started

The simplest way to explain is by example.
Lets translate imperative Helm commands to Flux Helm Controller Custom Resources:

Helm client:

```console
helm repo add traefik https://helm.traefik.io/traefik
helm install my-traefik traefik/traefik \
  --version 9.18.2 \
  --namespace traefik
```

Flux client:

```console
flux create source helm traefik --url https://helm.traefik.io/traefik
flux create helmrelease --chart my-traefik \
  --source HelmRepository/traefik \
  --chart-version 9.18.2 \
  --namespace traefik
```

The main difference is Flux client will not imperatively create resources in the cluster.
Instead these commands create Custom Resource *files*, which are committed to version control as instructions only (note: you may use the `--export` flag to manage any file edits with finer grained control before pushing to version control).
Separately, the Flux Helm Controller software agent automatically reconciles these instructions with the running state of your cluster based on your configured rules.

Lets check out what the Custom Resoruce instruction files look like:

```yaml
# /flux/boot/traefik/helmrepo.yaml
apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: HelmRepository
metadata:
  name: traefik
  namespace: traefik
spec:
  interval: 1m0s
  url: https://helm.traefik.io/traefik
```

```yaml
# /flux/boot/traefik/helmrelease.yaml
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: traefik
  namespace: traefik
spec:
  chart:
    spec:
      chart: traefik
      sourceRef:
        kind: HelmRepository
        name: traefik
      version: 9.18.2
  interval: 1m0s
```

<!-- Using the Flux Kustomize Controller, these are automatically applied to your cluster, just as any other Kubernetes resources are. Note that while you may find value in combining Kustomize overlays with your Helm Controller manifests to further reduce file duplication, that is entirely optional and unrelated to the Helm Controller. -->

Once these are applied to your cluster, the Flux Helm Controller automatically uses the Helm SDK to do your bidding according to the rules you've set.

Why is this important?
If you or your team has ever collaborated with multiple engineers on one or more apps, and/or in more than one namespace or cluster, you probably have a good idea of how declarative, automatic reconciliation can help solve common problems.
If not, or either way, you may want to check out this [short introduction to GitOps](https://youtu.be/r-upyR-cfDY).

## Next Steps

- [Guides > Manage Helm Releases](/guides/helmreleases/)
- [Toolkit Components > Helm Controller](/components/helm/controller/)
- [Migration > Migrate to the Helm Controller](/guides/helm-operator-migration/)
