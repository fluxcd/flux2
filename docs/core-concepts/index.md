# Core Concepts

!!! note "Work in progress"
    This document is a work in progress.

These are some core concepts in Flux.

## GitOps

GitOps is a way of managing your infrastructure and applications so that whole system is described declaratively and version controlled (most likely in a Git repository), and having an automated process that ensures that the deployed environment matches the state specified in a repository.

For more information, take a look at ["What is GitOps?"](https://www.gitops.tech/#what-is-gitops).

## Sources

A *Source* defines the origin of a source and the requirements to obtain
it (e.g. credentials, version selectors). For example, the latest `1.x` tag
available from a Git repository over SSH.

Sources produce an artifact that is consumed by other Flux elements to perform
actions, like applying the contents of the artifact on the cluster. A source
may be shared by multiple consumers to deduplicate configuration and/or storage.

The origin of the source is checked for changes on a defined interval, if
there is a newer version available that matches the criteria, a new artifact
is produced.

All sources are specified as Custom Resources in a Kubernetes cluster, examples
of sources are `GitRepository`, `HelmRepository` and `Bucket` resources. 

For more information, take a look at [the source controller documentation](../components/source/controller.md).

## Reconciliation

Reconciliation refers to ensuring that a given state (e.g application running in the cluster, infrastructure) matches a desired state declaratively defined somewhere (e.g a git repository). There are various examples of these in flux e.g:

- HelmRelease reconciliation: ensures the state of the Helm release matches what is defined in the resource, performs a release if this is not the case (including revision changes of a HelmChart resource).
- Bucket reconciliation: downloads and archives the contents of the declared bucket on a given interval and stores this as an artifact, records the observed revision of the artifact and the artifact itself in the status of resource.
- [Kustomization](#kustomization) reconciliation: ensures the state of the application deployed on a cluster matches resources contained in a git repository.

## Kustomization

The kustomization represents a local set of Kubernetes resources that Flux is supposed to reconcile in the cluster. The reconciliation runs every one minute by default but this can be specified in the kustomization. If you make any changes to the cluster using `kubectl edit` or `kubectl patch`, it will be promptly reverted. You either suspend the reconciliation or push your changes to a Git repository.

For more information, take a look at [this documentation](../components/kustomize/kustomization.md).

## Bootstrap

The process of installing the Flux components in a complete GitOps way is called a bootstrap. The manifests are applied to the cluster, a `GitRepository` and `Kustomization` are created for the Flux components, and the manifests are pushed to an existing Git repository (or a new one is created). Flux can manage itself just as it manages other resources. 
The bootstrap is done using the `flux` CLI `flux bootstrap`.

For more information, take a look at [the documentation for the bootstrap command](../cmd/flux_bootstrap.md).
