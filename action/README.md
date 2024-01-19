# Flux GitHub Action

To install the latest Flux CLI on Linux, macOS or Windows GitHub runners:

```yaml
steps:
  - name: Setup Flux CLI
    uses: fluxcd/flux2/action@main
    with:
      version: 'latest'
  - name: Run Flux CLI
    run: flux version --client
```

The Flux GitHub Action can be used to automate various tasks in CI, such as:

- [Automate Flux upgrades on clusters via Pull Requests](https://fluxcd.io/flux/flux-gh-action/#automate-flux-updates)
- [Push Kubernetes manifests to container registries](https://fluxcd.io/flux/flux-gh-action/#push-kubernetes-manifests-to-container-registries)
- [Run end-to-end testing with Flux and Kubernetes Kind](https://fluxcd.io/flux/flux-gh-action/#end-to-end-testing)

For more information, please see the [Flux GitHub Action documentation](https://fluxcd.io/flux/flux-gh-action/).

