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

To install Flux plugins alongside the CLI:

```yaml
steps:
  - name: Setup Flux CLI
    uses: fluxcd/flux2/action@main
    with:
      plugins: |
        mirror@0.0.1
  - name: Run Flux plugin
    run: flux mirror --help
```

Installing plugins requires a Flux version with plugin support (v2.9.0 or later).
The `plugins` input accepts one plugin per line in `<name>`,
`<name>@<version>`, or `<name>@<digest>` format. Entries without a version or
digest install the latest version from the catalog. The `plugin-dir` input is
only used when `plugins` is set; when plugins are installed, the action exports
`FLUXCD_PLUGINS` for subsequent steps.

The Flux GitHub Action can be used to automate various tasks in CI, such as:

- [Automate Flux upgrades on clusters via Pull Requests](https://fluxcd.io/flux/flux-gh-action/#automate-flux-updates)
- [Push Kubernetes manifests to container registries](https://fluxcd.io/flux/flux-gh-action/#push-kubernetes-manifests-to-container-registries)
- [Run end-to-end testing with Flux and Kubernetes Kind](https://fluxcd.io/flux/flux-gh-action/#end-to-end-testing)

For more information, please see the [Flux GitHub Action documentation](https://fluxcd.io/flux/flux-gh-action/).

