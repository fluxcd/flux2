# Roadmap

!!! hint "Work in Progress"
    We will be building the roadmap together with the Flux community,
    our end-users and everyone who is interested in integrating with us.
    So a lot of this is still TBD - read this as our shopping list of
    ideas after some brainstorming as Flux maintainers.

## The road to Flux v2

### Flux read-only feature parity

This would be the first stepping stone: we want the GitOps Toolkit to be on-par with today's Flux in
[read-only mode](https://github.com/fluxcd/flux/blob/master/docs/faq.md#can-i-run-flux-with-readonly-git-access)
and [FluxCloud](https://github.com/justinbarrick/fluxcloud) notifications.

Goals

- Offer an in-place migration tool for those that are using Flux in read-only mode to synchronize plain manifests
- Offer a migration guide for those that are using Flux in read-only mode to synchronize Kustomize overlays
- Offer a dedicated component for forwarding events to external messaging platforms 

Non-Goals

- Migrate users that are using Flux to run custom scripts with `flux.yaml`
- Automate the migration of `flux.yaml` kustomize users

Tasks

- Review the git source and kustomize APIs
- Design the events API
- Implement events in source and kustomize controllers
- Implement Prometheus metrics in source and kustomize controllers
- Make the kustomize-controller apply/gc events on-par with Flux v1 apply events
- Design the notifications and events filtering API
- Implement a notification controller for Slack, MS Teams, Discord, Rocket
- Implement the migration command in tk
- Create a migration guide for `flux.yaml` kustomize users

### Flux image update feature parity

Goals

- Offer a dedicated component that can replace Flux v1 image update feature

Non-Goals

- Maintain backwards compatibility with Flux v1 annotations

Tasks

- Design the Git push API
- Implement Git push in source controller
- Design the image scanning API
- Implement an image scanning controller
- Design the manifests patching component
- Implement the image scan/patch/push workflow
- Integrate the new components in the toolkit assembler
- Create a migration guide from Flux annotations

## The road to Helm Operator v2

### Helm v3 feature parity

Goals

- Offer a migration guide for those that are using Helm Operator with Helm v3 and Helm repositories

Non-Goals

- Migrate users that are using Helm v2
- Migrate users that are using Helm charts from Git

Tasks

- Review the Helm release, chart and repository APIs
- Design Helm releases based on source API
- Implement a Helm controller for Helm v3 covering all the current release options
- Implement events in Helm controller
- Implement Prometheus metrics in Helm controller
- Create a migration guide for Helm Operator users
