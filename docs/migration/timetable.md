---
hide:
  # The table data on this page is easier to read when wider
  # The TOC right column is already blank anyway
  - toc
---
# Migration and Support Timetable

!!! heart "Flux Migration Commitment"
    This public timetable clarifies our commitment to end users.
    Its purpose is to help improve your experience in deciding how and when to plan infra decisions related to Flux versions.
    Please refer to the [Roadmap](../roadmap/index.md) for additional details.

<!-- Note: this div allows us to set fixed column widths in custom.css -->
<!-- See: https://github.com/squidfunk/mkdocs-material/issues/118 -->
<div markdown="1" class="timetable-explicit-col-widths">

<!-- Requires mkdocs markdown_extensions footnotes and pymdownx.caret -->
<!-- markdownlint-disable-file MD033 -->
| Date | Flux 1 | Flux 2 CLI | GOTK[^1] |
| -- | -- | -- | -- |
| Oct 6, 2020 | ^^[Maintenance Mode](https://github.com/fluxcd/website/pull/25)^^<br><ul><li>Flux 1 releases only include critical bug fixes (which don’t require changing Flux 1 architecture), and security patches (for OS packages, Go runtime, and kubectl). No new features</li><li>Existing projects encouraged to test migration to Flux 2 pre-releases in non-production</li></ul> | ^^Development Mode^^<br><ul><li>Working to finish parity with Flux 1</li><li>New projects encouraged to test Flux 2 pre-releases in non-production</li></ul> | ^^All Alpha^^[^2] |
Feb 18, 2021 | ^^Partial Migration Mode^^<br><ul><li>Existing projects encouraged to migrate to `v1beta1`/`v2beta1` if you only use those features (Flux 1 read-only mode, and Helm Operator)</li><li>Existing projects encouraged to test image automation Alpha in non-production</li></ul> | ^^Feature Parity^^ | ^^Image Automation Alpha. All others reached Feature Parity, Beta^^ |
| TBD | ^^Superseded^^<br><ul><li>All existing projects encouraged to [migrate to Flux 2](https://toolkit.fluxcd.io/guides/flux-v1-migration/), and [report any bugs](https://github.com/fluxcd/flux2/issues/new/choose)</li><li>Flux 1 Helm Operator archived – no further updates due to unsupported dependencies</li></ul> | ^^Needs further testing, may get breaking changes^^<br><ul><li>CLI needs further user testing during this migration period</li></ul> | ^^All Beta, Production Ready^^[^3]<br><ul><li>All Flux 1 features stable and supported in Flux 2</li><li>Promoting Alpha versions to Beta makes this Production Ready</li></ul> |
| TBD | ^^Migration and security support only^^<br><ul><li>Flux 1 releases only include security patches (no bug fixes)</li><li>Maintainers support users with migration to Flux 2 only, no longer with Flux 1 issues</li><li>Flux 1 archive date announced</li></ul> | ^^Public release (GA), Production Ready^^<br><ul><li>CLI commits to backwards compatibility moving forward</li><li>CLI follows kubectl style backwards compatibility support: +1 -1 MINOR version for server components (e.g., APIs, Controllers, validation webhooks)</li></ul> | ^^All Beta, Production Ready^^ |
| TBD | ^^Archived^^<br><ul><li>Flux 1 obsolete, no further releases or maintainer support</li><li>Flux 1 repo archived</li></ul> | ^^Continued active development^^ | ^^Continued active development^^ |

<!-- end .timetable-explicit-col-widths -->
</div>

[^1]: GOTK is shorthand for the [GitOps Toolkit](https://toolkit.fluxcd.io/components/) APIs and Controllers

[^2]: Versioning: Flux 2 is a multi-service architecture, so requires a more complex explanation than Flux 1:
     - Flux 2 CLI follows [Semantic Versioning](https://semver.org/) scheme
     - The GitOps Toolkit APIs follow the [Kubernetes API versioning](https://kubernetes.io/docs/reference/using-api/#api-versioning) pattern. See [Roadmap](https://toolkit.fluxcd.io/roadmap/) for component versions.
     - These are coordinated for cross-compatibility: For each Flux 2 CLI tag, CLI and GOTK versions are end-to-end tested together, so you may safely upgrade from one MINOR/PATCH version to another.

[^3]: The GOTK Custom Resource Definitions which are at `v1beta1` and `v2beta1` and their controllers are considered stable and production ready. Going forward, breaking changes to the beta CRDs will be accompanied by a conversion mechanism.
