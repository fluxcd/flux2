# Roadmap

In our planning discussions for the GitOps Toolkit we identified largely three areas of work:

- Feature parity with Flux v1 in read-only mode
- Feature parity with the image-update functionality in Flux v1
- Feature parity with Helm Operator v1

All of the above will constitute "Flux v2".

## The road to Flux v2

### Flux read-only feature parity

[= 90% "90%"]

This would be the first stepping stone: we want the GitOps Toolkit to be on-par with today's Flux in
[read-only mode](https://github.com/fluxcd/flux/blob/master/docs/faq.md#can-i-run-flux-with-readonly-git-access)
and [FluxCloud](https://github.com/justinbarrick/fluxcloud) notifications.

Goals

-  Offer a migration guide for those that are using Flux in read-only mode to synchronize plain manifests
-  Offer a migration guide for those that are using Flux in read-only mode to synchronize Kustomize overlays
-  <span class="check-bullet">:material-check-bold:</span> [Offer a dedicated component for forwarding events to external messaging platforms](https://toolkit.fluxcd.io/guides/notifications/)

Non-Goals

-  Migrate users that are using Flux to run custom scripts with `flux.yaml`
-  Automate the migration of `flux.yaml` kustomize users

Tasks

- [x]  <span style="color:grey">Design the events API</span>
- [x]  <span style="color:grey">Implement events in source and kustomize controllers</span>
- [x]  <span style="color:grey">Make the kustomize-controller apply/gc events on-par with Flux v1 apply events</span>
- [x]  <span style="color:grey">Design the notifications and events filtering API</span>
- [x]  <span style="color:grey">Implement a notification controller for Slack, MS Teams, Discord, Rocket</span>
- [x]  <span style="color:grey">Implement Prometheus metrics in source and kustomize controllers</span>
- [x]  <span style="color:grey">Review the git source and kustomize APIs</span>
- [ ]  Create a migration guide for `flux.yaml` kustomize users
- [x]  <span style="color:grey">Include support for SOPS</span>

### Flux image update feature parity

[= 30% "30%"]

Goals

-  Offer components that can replace Flux v1 image update feature

Non-Goals

-  Maintain backwards compatibility with Flux v1 annotations

Tasks

- [x]  <span style="color:grey">[Design the image scanning and automation API](https://github.com/fluxcd/flux2/discussions/107)</span>
- [ ]  Implement an image scanning controller
- [x]  <span style="color:grey">Design the automation component</span>
- [ ]  Implement the image scan/patch/push workflow
- [ ]  Integrate the new components in the toolkit assembler
- [ ]  Create a migration guide from Flux annotations

## The road to Helm Operator v2

### Helm v3 feature parity

[= 100% "100%"]

Goals

-  Offer a migration guide for those that are using Helm Operator with Helm v3 and charts from
   Helm and Git repositories

Non-Goals

-  Migrate users that are using Helm v2

Tasks

- [x]  <span style="color:grey">Implement a Helm controller for Helm v3 covering all the current release options</span>
- [x]  <span style="color:grey">Discuss and design Helm releases based on source API:</span>
    * [x]  <span style="color:grey">Providing values from sources</span>
    * [x]  <span style="color:grey">Conditional remediation on failed Helm actions</span>
    * [x]  <span style="color:grey">Support for Helm charts from Git</span>
- [x]  <span style="color:grey">Review the Helm release, chart and repository APIs</span>
- [x]  <span style="color:grey">Implement events in Helm controller</span>
- [x]  <span style="color:grey">Implement Prometheus metrics in Helm controller</span>
- [x]  <span style="color:grey">Implement support for values from `Secret` and `ConfigMap` resources</span>
- [x]  <span style="color:grey">Implement conditional remediation on (failed) Helm actions</span>
- [x]  <span style="color:grey">Implement support for Helm charts from Git</span>
- [x]  <span style="color:grey">Implement support for referring to an alternative chart values file</span>
- [ ]  Stabilize API
- [ ]  Create a migration guide for Helm Operator users
