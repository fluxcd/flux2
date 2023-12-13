# :warning: Removal Notice

Starting Flux v2.1.0, released August 24, 2023, the Flux monitoring
configurations in this repository were marked as deprecated. The new monitoring
docs are available at [Flux monitoring](https://fluxcd.io/flux/monitoring/)
docs with new example configurations in
[fluxcd/flux2-monitoring-example](https://github.com/fluxcd/flux2-monitoring-example/).

The deprecated configurations were removed in Flux v2.2 on December 13, 2023. All
users of these configurations are advised to use the new monitoring setup,
following the [docs](https://fluxcd.io/flux/monitoring/) and the
[examples](https://github.com/fluxcd/flux2-monitoring-example/).

After collecting a lot of user feedback about our monitoring recommendation, in
order to serve most of the needs of the users, we decided to create a new
monitoring setup leveraging more of the kube-prometheus-stack, specifically
kube-state-metrics, to enable configuring Flux custom metrics, see the [Flux
custom Prometheus metrics](https://fluxcd.io/flux/monitoring/custom-metrics/)
docs to learn more about it. Please refer to
[fluxcd/flux2/4128](https://github.com/fluxcd/flux2/issues/4128) for a detailed
explanation about this change and the new capabilities offered by the new
monitoring setup.
