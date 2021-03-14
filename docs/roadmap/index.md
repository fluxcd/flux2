# Roadmap

!!! hint "Production readiness"
    The Flux custom resource definitions which are at `v1beta1` and `v2beta1`
    and their controllers are considered stable and production ready.
    Going forward, breaking changes to the beta CRDs will be accompanied by a conversion mechanism.
    Please see the [Migration and Suport Timetable](../migration/timetable.md) for our commitment to end users.

The following components (included by default in [flux bootstrap](../guides/installation.md#bootstrap))
are considered production ready:

- [source-controller](../components/source)
- [kustomize-controller](../components/kustomize)
- [notification-controller](../components/notification)
- [helm-controller](../components/helm)

The following GitOps Toolkit APIs are considered production ready:

- `source.toolkit.fluxcd.io/v1beta1`
- `kustomize.toolkit.fluxcd.io/v1beta1`
- `notification.toolkit.fluxcd.io/v1beta1`
- `helm.toolkit.fluxcd.io/v2beta1`

## The road to Flux v2 GA

In our planning discussions we have identified these possible areas of work,
this list is subject to change while we gather feedback:

- Stabilize the image automation APIs
    * Review the spec of `ImageRepository`, `ImagePolicy` and `ImageUpdateAutomation`
    * Promote the image automation APIs to `v1beta1`
    * Include the image automation controllers in the default components list

- Improve the documentation
    * Gather feedback on the [migration guides](https://github.com/fluxcd/flux2/discussions/413) and address more use-cases
    * Incident management and troubleshooting guides
    * Cloud specific guides (AWS, Azure, Google Cloud, more?)
    * Consolidate the docs under [fluxcd.io](https://fluxcd.io) website
    
## The road to Flux v1 feature parity

In our planning discussions we identified three areas of work:

- Feature parity with Flux v1 in read-only mode
- Feature parity with the image-update functionality in Flux v1
- Feature parity with Helm Operator v1

### Flux read-only feature parity

[= 100% "100%"]

Flux v2 read-only is ready to try. See the [Getting
Started](https://toolkit.fluxcd.io/get-started/) how-to, and the
[Migration
guide](https://toolkit.fluxcd.io/guides/flux-v1-migration/).

This would be the first stepping stone: we want Flux v2 to be on-par with today's Flux in
[read-only mode](https://github.com/fluxcd/flux/blob/master/docs/faq.md#can-i-run-flux-with-readonly-git-access)
and [FluxCloud](https://github.com/justinbarrick/fluxcloud) notifications.

Goals

-  <span class="check-bullet">:material-check-bold:</span> [Offer a migration guide for those that are using Flux in read-only mode to synchronize plain manifests](https://toolkit.fluxcd.io/guides/flux-v1-migration/)
-  <span class="check-bullet">:material-check-bold:</span> [Offer a migration guide for those that are using Flux in read-only mode to synchronize Kustomize overlays](https://toolkit.fluxcd.io/guides/flux-v1-migration/)
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
- [x]  <span style="color:grey">Support [bash-style variable substitution](https://toolkit.fluxcd.io/components/kustomize/kustomization/#variable-substitution) as an alternative to `flux.yaml` envsubst/sed usage</span>
- [x]  <span style="color:grey">Create a migration guide for `flux.yaml` kustomize users</span>
- [x]  <span style="color:grey">Include support for SOPS</span>

### Flux image update feature parity

[= 100% "100%"]

Image automation is available as a prerelease. See [this
guide](https://toolkit.fluxcd.io/guides/image-update/) for how to
install and use it.

Goals

-  Offer components that can replace Flux v1 image update feature

Non-Goals

-  Maintain backwards compatibility with Flux v1 annotations
-  [Order by timestamps found inside image layers](https://github.com/fluxcd/flux2/discussions/802)

Tasks

- [x] <span style="color:grey">[Design the image scanning and automation API](https://github.com/fluxcd/flux2/discussions/107)</span>
- [x] <span style="color:grey">Implement an image scanning controller</span>
- [x] <span style="color:grey">Public image repo support</span>
- [x] <span style="color:grey">Credentials from Secret [fluxcd/image-reflector-controller#35](https://github.com/fluxcd/image-reflector-controller/pull/35)</span>
- [x] <span style="color:grey">Design the automation component</span>
- [x] <span style="color:grey">Implement the image scan/patch/push workflow</span>
- [x] <span style="color:grey">Integrate the new components in the Flux CLI [fluxcd/flux2#538](https://github.com/fluxcd/flux2/pull/538)</span>
- [x] <span style="color:grey">Write a guide for how to use image automation ([guide here](https://toolkit.fluxcd.io/guides/image-update/))</span>
- [x] <span style="color:grey">ACR/ECR/GCR integration ([guide here](https://toolkit.fluxcd.io/guides/image-update/#imagerepository-cloud-providers-authentication))</span>
- [x] <span style="color:grey">Write a migration guide from Flux v1 annotations ([guide here](https://toolkit.fluxcd.io/guides/flux-v1-automation-migration/))</span>

### Helm v3 feature parity

[= 100% "100%"]

Helm support in Flux v2 is ready to try. See the [Helm controller
guide](https://toolkit.fluxcd.io/guides/helmreleases/), and the [Helm
controller migration
guide](https://toolkit.fluxcd.io/guides/helm-operator-migration/).

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
- [x]  <span style="color:grey">Stabilize API</span>
- [x]  <span style="color:grey">[Create a migration guide for Helm Operator users](../guides/helm-operator-migration.md)</span>
