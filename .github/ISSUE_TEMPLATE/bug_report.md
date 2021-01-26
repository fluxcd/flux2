---
name: Bug report
about: Create a report to help us improve Flux v2
title: ''
assignees: ''

---

<!--

Find out more about your support options and getting help at

   https://fluxcd.io/support/

-->

### Describe the bug

A clear and concise description of what the bug is.

### To Reproduce

Steps to reproduce the behaviour:

1. Provide Flux install instructions
2. Provide a GitHub repository with Kubernetes manifests

### Expected behavior

A clear and concise description of what you expected to happen.

### Additional context

- Kubernetes version:
- Git provider:
- Container registry provider:

Below please provide the output of the following commands:

```cli
flux --version
flux check
kubectl -n <namespace> get all
kubectl -n <namespace> logs deploy/source-controller
kubectl -n <namespace> logs deploy/kustomize-controller
```
