# Flux GitHub Action

Example workflow:

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
