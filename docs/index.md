# GitOps Toolkit

The GitOps Toolkit is a set of composable APIs and specialized tools
that can be used to build a Continuous Delivery platform on top of Kubernetes.

These tools are built with Kubernetes controller-runtime libraries and they
can be dynamically configured with Kubernetes custom resources either by
cluster admins or by other automated tools.
The GitOps Toolkit components interact with each other via Kubernetes
events and are responsible for the reconciliation of their designated API objects. 

!!! hint "Work in Progress"
    We envision a feature where **Flux v2** and **Helm Operator v2** will be assembled from
    the GitOps Toolkit components. The Flux CD team is looking for feedback and help as 
    the toolkit is in an active experimentation phase.
    If you wish to take part in this quest please reach out to us on Slack and GitHub.

Components:

- [Toolkit CLI](https://github.com/fluxcd/toolkit)
- [Source Controller](components/source/controller.md)
- [Kustomize Controller](components/kustomize/controller.md)

To get started with the toolkit please follow this [guide](get-started/index.md).

