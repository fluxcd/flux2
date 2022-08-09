# Flux GitHub Action

Usage:

```yaml
    steps:
      - name: Setup Flux CLI
        uses: fluxcd/flux2/action@main
      - name: Run Flux commands
        run: flux -v
```

The latest stable version of the `flux` binary is downloaded from
GitHub [releases](https://github.com/fluxcd/flux2/releases)
and placed at `/usr/local/bin/flux`.

Note that this action can only be used on GitHub **Linux** runners.
You can change the arch (defaults to `amd64`) with:

```yaml
    steps:
      - name: Setup Flux CLI
        uses: fluxcd/flux2/action@main
        with:
          arch: arm64 # can be amd64, arm64 or arm
```

You can download a specific version with:

```yaml
    steps:
      - name: Setup Flux CLI
        uses: fluxcd/flux2/action@main
        with:
          version: 0.32.0
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

### Push Kubernetes manifests to container registries

Example workflow for publishing Kubernetes manifests bundled as OCI artifacts to GitHub Container Registry:

```yaml
name: push-artifact-staging

on:
  push:
    branches:
      - 'main'

permissions:
  packages: write # needed for ghcr.io access

env:
  OCI_REPO: "oci://ghcr.io/my-org/manifests/${{ github.event.repository.name }}"

jobs:
  kubernetes:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v2
      - name: Setup Flux CLI
        uses: fluxcd/flux2/action@main
      - name: Generate manifests
        run: |
          kustomize build ./manifests/staging > ./deploy/app.yaml
      - name: Push manifests
        run: |
          flux push artifact $OCI_REPO:$(git rev-parse --short HEAD) \
            --path="./deploy" \
            --source="$(git config --get remote.origin.url)" \
            --revision="$(git branch --show-current)/$(git rev-parse HEAD)"
      - name: Deploy manifests to staging
        run: |
          flux tag artifact $OCI_REPO:$(git rev-parse --short HEAD) --tag staging
```

Example workflow for publishing Kubernetes manifests bundled as OCI artifacts to Docker Hub:

```yaml
name: push-artifact-production

on:
  push:
    tags:
      - '*'

env:
  OCI_REPO: "oci://docker.io/my-org/app-config"

jobs:
  kubernetes:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v2
      - name: Setup Flux CLI
        uses: fluxcd/flux2/action@main
      - name: Login to Docker Hub
        uses: docker/login-action@v2
        with:
          username: ${{ secrets.DOCKER_USERNAME }}
          password: ${{ secrets.DOCKER_PASSWORD }}
      - name: Generate manifests
        run: |
          kustomize build ./manifests/production > ./deploy/app.yaml
      - name: Push manifests
        run: |
          flux push artifact $OCI_REPO:$(git tag --points-at HEAD) \
            --path="./deploy" \
            --source="$(git config --get remote.origin.url)" \
            --revision="$(git tag --points-at HEAD)/$(git rev-parse HEAD)"
      - name: Deploy manifests to production
        run: |
          flux tag artifact $OCI_REPO:$(git tag --points-at HEAD) --tag production
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
