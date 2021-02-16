# Flux GitHub Action

Usage:

```yaml
    steps:
      - name: Setup Flux CLI
        uses: fluxcd/flux2/action@main
      - name: Run Flux commands
        run: flux -v
```

Note that this action can only be used on GitHub **Linux AMD64** runners.
The latest stable version of the `flux` binary is downloaded from
GitHub [releases](https://github.com/fluxcd/flux2/releases)
and placed at `/usr/local/bin/flux`.

You can download a specific version with:

```yaml
    steps:
      - name: Setup Flux CLI
        uses: fluxcd/flux2/action@main
        with:
          version: 0.8.0
```

### Automate Flux updates

Example workflow for updating Flux's components generated with `flux bootstrap --path=clusters/production`:

```yaml
name: update-flux

on:
  workflow_dispatch:
  schedule:
    - cron: "0 * * * *"

jobs:
  components:
    runs-on: ubuntu-latest
    steps:
      - name: Check out code
        uses: actions/checkout@v2
      - name: Setup Flux CLI
        uses: fluxcd/flux2/action@main
      - name: Check for updates
        id: update
        run: |
          flux install \
            --export > ./clusters/production/flux-system/gotk-components.yaml

          VERSION="$(flux -v)"
          echo "::set-output name=flux_version::$VERSION"
      - name: Create Pull Request
        uses: peter-evans/create-pull-request@v3
        with:
            token: ${{ secrets.GITHUB_TOKEN }}
            branch: update-flux
            commit-message: Update to ${{ steps.update.outputs.flux_version }}
            title: Update to ${{ steps.update.outputs.flux_version }}
            body: |
              ${{ steps.update.outputs.flux_version }}
```

### End-to-end testing

Example workflow for running Flux in Kubernetes Kind:

```yaml
name: e2e

on:
  push:
    branches:
      - '*'

jobs:
  kubernetes:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v2
      - name: Setup Flux CLI
        uses: fluxcd/flux2/action@main
      - name: Setup Kubernetes Kind
        uses: engineerd/setup-kind@v0.5.0
      - name: Install Flux in Kubernetes Kind
        run: flux install
```

A complete e2e testing workflow is available here
[flux2-kustomize-helm-example](https://github.com/fluxcd/flux2-kustomize-helm-example/blob/main/.github/workflows/e2e.yaml)
