# RFC-0009 Custom Health Checks for Kustomization using Common Expression Language (CEL)

**Status:** implemented

**Creation date:** 2024-01-05

**Last update:** 2025-02-22

## Summary

This RFC proposes to extend the Flux `Kustomization` API with custom health checks for
custom resources using the Common Expression Language (CEL).

In order to provide flexibility, we propose to use CEL expressions for defining the
conditions that need to be met in order to determine the health of a custom resource.
We will introduce a new field called `healthCheckExprs` in the `Kustomization` CRD
which will be a list of CEL expressions for evaluating the status of a particular
Kubernetes resource kind.

## Motivation

Flux uses the `kstatus` library during the health check phase to compute owned
resources status. This works just fine for all the Kubernetes core resources
and custom resources that comply with the `kstatus` conventions.

There are cases where the status of a custom resource does not follow the
`kstatus` conventions. For example, we might want to compute the status of a custom
resource based on a condition other than `Ready`. This is the case for resources
that perform intermediary patching, like `Certificate` from cert-manager, where one
should look at the `Issuing` condition to know if the certificate is being issued or
not before looking at the `Ready` condition.

We need to provide a way for users to express the conditions that need to be
met in order to determine the health of a custom resource. We seek a solution
flexible enough to cover all possible use cases that does not require changing
the Flux source code for each new CRD.

### Goals

- Provide a generic solution for users to customise the health check evaluation of custom resources.
- Provide a space for the community to contribute custom health checks for popular custom resources.

### Non-Goals

- We do not plan to support custom health checks for Kubernetes core resources.

## Proposal

### Introduce a new field `HealthCheckExprs` in the `Kustomization` CRD

The `HealthCheckExprs` field will be a list of `CustomHealthCheck` objects.
The `CustomHealthCheck` object fields would be: `apiVersion`, `kind`, `inProgress`,
`failed` and `current`.

For example, consider the following `Certificate` resource:

```yaml
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: app-certificate
spec:
  commonName: cert-manager-tls
  dnsNames:
  - app.ns.svc.cluster.local
  ipAddresses:
  - x.x.x.x
  isCA: true
  issuerRef:
    group: cert-manager.io
    kind: ClusterIssuer
    name: app-issuer
  secretName: app-tls-certs
  subject:
    organizations:
    - example.com
```

This `Certificate` resource will transition through the following `conditions`:
`Issuing` and `Ready`.

In order to compute the status of this resource, we need to look at both the `Issuing`
and `Ready` conditions.

The Flux `Kustomization` object used to apply the `Certificate` will look like this:

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: certs
spec:
  interval: 5m
  prune: true
  sourceRef:
    kind: GitRepository
    name: flux-system
  path: ./certs
  wait: true
  healthCheckExprs:
  - apiVersion: cert-manager.io/v1
    kind: Certificate
    inProgress: "status.conditions.filter(e, e.type == 'Issuing').all(e, e.observedGeneration == metadata.generation && e.status == 'True')"
    failed: "status.conditions.filter(e, e.type == 'Ready').all(e, e.observedGeneration == metadata.generation && e.status == 'False')"
    current: "status.conditions.filter(e, e.type == 'Ready').all(e, e.observedGeneration == metadata.generation && e.status == 'True')"
```

The `.spec.healthCheckExprs` field contains an entry for the `Certificate` kind, its `apiVersion`,
and the CEL expressions that need to be met in order to determine the health status of all custom resources
of this kind reconciled by the Flux `Kustomization`.

### Custom Health Check Library

To help users define custom health checks, we will provide on the [fluxcd.io](https://fluxcd.io)
website a library of custom health checks for popular custom resources.

The Flux community will be able to contribute to this library by submitting pull requests
to the [fluxcd/website](https://github.com/fluxcd/website) repository.

### User Stories

#### Configure health checks for non-standard custom resources

> As a Flux user, I want to be able to specify health checks for
> custom resources that don't have a Ready condition, so that I can be notified
> when the status of my resources transitions to a failed state based on the evaluation
> of a different condition.

Using `.spec.healthCheckExprs`, Flux users have the ability to
specify the conditions that need to be met in order to determine the health of
a custom resource. This enables Flux to query any `.status` field,
besides the standard `Ready` condition, and evaluate it using a CEL expression.

Example for `SealedSecret` which has a `Synced` condition:

```yaml
  - apiVersion: bitnami.com/v1alpha1
    kind: SealedSecret
    failed: "status.conditions.filter(e, e.type == 'Synced').all(e, e.status == 'False')"
    current: "status.conditions.filter(e, e.type == 'Synced').all(e, e.status == 'True')"
```

#### Use Flux dependencies for Kubernetes ClusterAPI

> As a Flux user, I want to be able to use Flux dependencies bases on the
> readiness of ClusterAPI resources, so that I can ensure that my applications
> are deployed only when the ClusterAPI resources are ready.

The ClusterAPI resources have a `Ready` condition, but this is set in the status
after the cluster is first created. Given this behavior, at creation time Flux
cannot find any condition to evaluate the status of the ClusterAPI resources,
thus it considers them as static resources which are always ready.

Using `.spec.healthCheckExprs`, Flux users can specify that the `Cluster`
kind is expected to have a `Ready` condition which will force Flux into waiting
for the ClusterAPI resources status to be populated.

Example for `Cluster`:

```yaml
  - apiVersion: cluster.x-k8s.io/v1beta1
    kind: Cluster
    failed: "status.conditions.filter(e, e.type == 'Ready').all(e, e.status == 'False')"
    current: "status.conditions.filter(e, e.type == 'Ready').all(e, e.status == 'True')"
```

### Alternatives

We need an expression language that is flexible enough to cover all possible use
cases, without having to change Flux source code for each new use case.

An alternative that has been considered was to use `CUE` instead of `CEL`.
`CUE` lang is a more powerful expression language, but given the fact that
Kubernetes makes use of `CEL` for CRD validation and admission control,
we have decided to also use `CEL` in Flux in order to be consistent with
the Kubernetes ecosystem.

## Design Details

### Introduce a new field `HealthCheckExprs` in the `Kustomization` CRD

The `api/v1/kustomization_types.go` file will be updated to add the `HealthCheckExprs`
field to the `KustomizationSpec` struct.

```go
type KustomizationSpec struct {
	// +optional
	HealthCheckExprs []CustomHealthCheck `json:"healthCheckExprs,omitempty"`
}

type CustomHealthCheck struct {
	// APIVersion of the custom resource under evaluation.
	// +required
	APIVersion string `json:"apiVersion"`
	// Kind of the custom resource under evaluation.
	// +required
	Kind string `json:"kind"`

	HealthCheckExpressions `json:",inline"`
}

type HealthCheckExpressions struct {
	// Current is the CEL expression that determines if the status
	// of the custom resource has reached the desired state.
	// +required
	Current string `json:"current"`
	// InProgress is the CEL expression that determines if the status
	// of the custom resource has not yet reached the desired state.
	// +optional
	InProgress string `json:"inProgress,omitempty"`
	// Failed is the CEL expression that determines if the status
	// of the custom resource has failed to reach the desired state.
	// +optional
	Failed string `json:"failed,omitempty"`
}
```

If a CEL expression evaluation results in an error, for example, looking for a field that does not exist,
the health check will fail. Users will be encouraged to test their expressions
in the [CEL Playground](https://playcel.undistro.io/). Here is where the community-maintained
[library](#custom-health-check-library) will be super useful as some of the expressions might be complex.

The evaluation logic will be as follows.

First, we check if the custom resource has a `.status.observedGeneration` field, if it does
we compare it with the `.metadata.generation` field to determine if the custom resource is in
progress. We consider it in progress if these fields differ, and don't evaluate any of the
expressions if that's the case. However, if these fields are equal there's no immediate
conclusion about the health of the custom resource, so we proceed with the evaluation.

For each of the `InProgress`, `Failed` and `Current` expressions, we evaluate the expressions
that are specified (`InProgress` and `Failed` are optional) in this specific order and return
the respective status of the first expression that evaluates to `true`. If none of the
expressions evaluate to `true`, we consider the custom resource in progress.

When the `Failed` expression is not specified the controller will keep evaluating the
`Current` expression until it returns `true`, and will give up after the timeout defined in the Kustomization's `spec.timeout` field is reached.
Users will be encouraged to provide a `Failed` expression to avoid stalling the reconciliation
loop until the timeout is reached.

### Introduce a generic custom status reader

We'll Introduce a `StatusReader` that will be used to compute the status
of custom resources based on the `CEL` expressions provided in the `CustomHealthCheck`:

```go
import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"github.com/fluxcd/cli-utils/pkg/kstatus/polling/engine"
	"github.com/fluxcd/cli-utils/pkg/kstatus/polling/event"
	kstatusreaders "github.com/fluxcd/cli-utils/pkg/kstatus/polling/statusreaders"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
)

type CELStatusReader struct {
	genericStatusReader engine.StatusReader
	gvk                 schema.GroupVersionKind
}

func NewCELStatusReader(mapper meta.RESTMapper, gvk schema.GroupVersionKind,
	exprs *kustomizev1.HealthCheckExpressions) engine.StatusReader {

	genericStatusReader := kstatusreaders.NewGenericStatusReader(mapper, genericConditions(gvk.Kind, exprs))
	return &CELStatusReader{
		genericStatusReader: genericStatusReader,
		gvk:                 gvk,
	}
}

func (g *CELStatusReader) Supports(gk schema.GroupKind) bool {
	return gk == g.gvk.GroupKind()
}

func (g *CELStatusReader) ReadStatus(ctx context.Context, reader engine.ClusterReader, resource object.ObjMetadata) (*event.ResourceStatus, error) {
	return g.genericStatusReader.ReadStatus(ctx, reader, resource)
}

func (g *CELStatusReader) ReadStatusForObject(ctx context.Context, reader engine.ClusterReader, resource *unstructured.Unstructured) (*event.ResourceStatus, error) {
	return g.genericStatusReader.ReadStatusForObject(ctx, reader, resource)
}
```

The `genericConditions` function takes the set of `CEL` expressions and returns a
function that takes an `Unstructured` object and returns a `status.Result` object.

```go
import (
	"github.com/fluxcd/cli-utils/pkg/kstatus/status"
	"github.com/fluxcd/pkg/runtime/cel"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func genericConditions(exprs *kustomizev1.HealthCheckExpressions) func(u *unstructured.Unstructured) (*status.Result, error) {
	return func(u *unstructured.Unstructured) (*status.Result, error) {
		obj := u.UnstructuredContent()

		// if status.observedGeneration exists and differs from metadata.generation return status.InProgress

		for _, e := range []struct{
			expr   string
			status status.Status
		}{
			{expr: exprs.InProgress, status: status.InProgress},
			{expr: exprs.Failed, status: status.Failed},
			{expr: exprs.Current, status: status.Current},
		} {
			if e.expr != "" {
				result, err := cel.EvaluateBooleanExpr(e.expr, obj)
				if err != nil {
					return nil, err
				}
				if result {
					return &status.Result{Status: e.status}, nil
				}
			}
		}

		return &status.Result{Status: status.InProgress}, nil
	}
}
```

The CEL status reader will be used by the `statusPoller` provided to the kustomize-controller `reconciler`
to compute the status of the resources for the registered custom resources GVKs.

We will implement a `CEL` environment that will use the Kubernetes CEL library to evaluate the `CEL` expressions.

## Implementation History

* RFC implemented and generally available in Flux v2.5.
