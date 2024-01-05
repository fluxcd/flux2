# RFC-0000 Custom Health Checks for Kustomization using Common Expression Language(CEL)

**Status:** provisional

**Creation date:** 2024-01-05

**Last update:** 2024-01-05

## Summary

This RFC proposes to support customization of the status readers in `Kustomizations`
during the `healthCheck` phase for custom resources. The user will be able to declare
the needed `conditions` in order to compute a custom resource status.
In order to provide flexibility, we propose to use `CEL` expressions to declare 
the expected conditions and their status.
This will introduce a new field `customHealthChecks` in the `Kustomization` CRD
which will be a list of `CustomHealthCheck` objects.

## Motivation

Flux uses the `Kstatus` library during the `healthCheck` phase to compute owned 
resources status. This works just fine for all standard resources and custom resources
that comply with `Kstatus` interfaces.

In the current Kustomization implementation, we have addressed such a problem for
kubernetes Jobs. We have implemented a `customJobStatusReader` that computes the
status of a Job based on a defined set of conditions. This is a good solution for
Jobs, but it is not generic and thus not applicable to other custom resources.

Another use case is relying on non-standard `conditions` to compute the status of
a custom resource. For example, we might want to compute the status of a custom
resource based on a condtion other then `Ready`. This is the case for `Resources`
that do intermediate patching like `Certificate` where you should look at the `Issued`
condition to know if the certificate has been issued or not before looking at the
`Ready` condition.

In order to provide a generic solution for custom resources, that would not imply
writing a custom status reader for each new custom resource, we need to provide a
way for the user to express the `conditions` that need to be met in order to compute
the status of a given custom resource. And we need to do this in a way that is
flexible enough to cover all possible use cases, without having to change `Flux`
source code for each new use case.

### Goals

- provide a generic solution for user to customize the health check of custom resources
- support non-standard resources in `kustomize-controller`

### Non-Goals

- We do not plan to support custom `healthChecks` for core resources.

## Proposal

### Introduce a new field `CustomHealthChecksExprs` in the `Kustomization` CRD

The `CustomHealthChecksExprs` field will be a list of `CustomHealthCheck` objects.
Each `CustomHealthChecksExprs` object will have a `apiVersion`, `kind`, `inProgress`,
`failed` and `current` fields.

To give an example, here is how we would declare a custom health check for a `Certificate`
resource:

```yaml
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: app-certificate
  namespace: cert-manager
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
  privateKey:
    algorithm: RSA
    encoding: PKCS1
    size: 2048
  secretName: app-tls-certs
  subject:
    organizations:
    - example.com
```

This `Certificate` resource will transition through the following `conditions`: 
`Issuing` and `Ready`.

In order to compute the status of this resource, we need to look at both the `Issuing`
and `Ready` conditions.

The resulting `Kustomization` object will look like this:

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1beta1
kind: Kustomization
metadata:
	name: application-kustomization
spec:
  force: false
  interval: 5m0s
  path: ./overlays/application
  prune: false
  sourceRef:
    kind: GitRepository
    name: application-git
  healthChecks:
  - apiVersion: cert-manager.io/v1
    kind: Certificate
    name: service-certificate
    namespace: cert-manager
  - apiVersion: apps/v1
    kind: Deployment
	  name: app
	  namespace: app
  customHealthChecksExprs:
  - apiVersion: cert-manager.io/v1
	  kind: Certificate
	  inProgress: "status.conditions.filter(e, e.type == 'Issuing').all(e, e.observedGeneration == metadata.generation && e.status == 'True')"
	  failed: "status.conditions.filter(e, e.type == 'Ready').all(e, e.observedGeneration == metadata.generation && e.status == 'False')"
	  current: "status.conditions.filter(e, e.type == 'Ready').all(e, e.observedGeneration == metadata.generation && e.status == 'True')"
```

The `HealthChecks` field still contains the objects that should be included in 
the health assessment. The `CustomHealthChecksExprs` field will be used to declare
the `conditions` that need to be met in order to compute the status of the custom resource.

Note that all core resources are discarded from the `CustomHealthChecksExprs` field.


#### Provide an evaluator for `CEL` expressions for users

We will provide a CEL environment that can be used by the user to evaluate `CEL`
expressions. Users will use it to test their expressions before applying them to
their `Kustomization` object.

```shell
$ flux eval --api-version cert-manager.io/v1 --kind Certificate --in-progress "status.conditions.filter(e, e.type == 'Issuing').all(e, e.observedGeneration == metadata.generation && e.status == 'True')" --failed "status.conditions.filter(e, e.type == 'Ready').all(e, e.observedGeneration == metadata.generation && e.status == 'False')" --current "status.conditions.filter(e, e.type == 'Ready').all(e, e.observedGeneration == metadata.generation && e.status == 'True')" --file ./custom_resource.yaml
```

### User Stories

#### Configure custom health checks for a custom resource

> As a user of Flux, I want to be able to specify custom health checks for my
> custom resources, so that I can have more control over the status of my
> resources.

#### Enable health checks support in Flux for non-standard resources

> As a user of Flux, I want to be able to use the health check feature for
> non-standard resources, so that I can have more control over the status of my
> resources.

### Alternatives

We need an expression language that is flexible enough to cover all possible use
cases, without having to change `Flux` source code for each new use case.

On alternative that have been considered is to use `cuelang` instead of `CEL`.
`cuelang` is a more powerful expression language, but it is also more complex and 
requires more work to integrate with `Flux`. it also does not have any support in
`Kubernetes` yet while `CEL` is already used in `Kubernetes` and libraries are
available to use it.

## Design Details

### Introduce a new field `CustomHealthChecksExprs` in the `Kustomization` CRD

The `api/v1/kustomization_types.go` file will be updated to add the `CustomHealthChecksExprs`
field to the `KustomizationSpec` struct.

```go
type KustomizationSpec struct {
...
	// A list of resources to be included in the health assessment.
	// +optional
	HealthChecks []meta.NamespacedObjectKindReference `json:"healthChecks,omitempty"`

	// A list of custom health checks expressed as CEL expressions.
	// The CEL expression must evaluate to a boolean value.
	// +optional
	CustomHealthChecksExprs []CustomHealthCheckExprs `json:"customHealthChecksExprs,omitempty"`
...
}

// CustomHealthCheckExprs defines the CEL expressions for custom health checks.
// The CEL expressions must evaluate to a boolean value. The expressions are used
// to determine the status of the custom resource.
type CustomHealthCheckExprs struct {
	// apiVersion of the custom health check.
	// +required
	APIVersion string `json:"apiVersion"`
	// Kind of the custom health check.
	// +required
	Kind string `json:"kind"`
	// InProgress is the CEL expression that verifies that the status
	// of the custom resource is in progress.
	// +optional
	InProgress string `json:"inProgress"`
	// Failed is the CEL expression that verifies that the status
	// of the custom resource is failed.
	// +optional
	Failed string `json:"failed"`
	// Current is the CEL expression that verifies that the status
	// of the custom resource is ready.
	// +optional
	Current string `json:"current"`
}
```

### Introduce a generic custom status reader

Introduce  a generic custom status reader that will be able to compute the status of
a custom resource based on a list of `conditions` that need to be met.

```go
import (
  "k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/engine"
  "sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	kstatusreaders "sigs.k8s.io/cli-utils/pkg/kstatus/polling/statusreaders"
)
type customGenericStatusReader struct {
	genericStatusReader engine.StatusReader
	gvk                 schema.GroupVersionKind
}

func NewCustomGenericStatusReader(mapper meta.RESTMapper, gvk schema.GroupVersionKind, exprs map[string]string) engine.StatusReader {
	genericStatusReader := kstatusreaders.NewGenericStatusReader(mapper, genericConditions(gvk.Kind, exprs))
	return &customJobStatusReader{
		genericStatusReader: genericStatusReader,
    gvk:                 gvk,
	}
}

func (g *customGenericStatusReader) Supports(gk schema.GroupKind) bool {
	return gk == g.gvk.GroupKind()
}

func (g *customGenericStatusReader) ReadStatus(ctx context.Context, reader engine.ClusterReader, resource object.ObjMetadata) (*event.ResourceStatus, error) {
	return g.genericStatusReader.ReadStatus(ctx, reader, resource)
}

func (g *customGenericStatusReader) ReadStatusForObject(ctx context.Context, reader engine.ClusterReader, resource *unstructured.Unstructured) (*event.ResourceStatus, error) {
	return g.genericStatusReader.ReadStatusForObject(ctx, reader, resource)
}
```

A `genericConditions` closure will takes a `kind` and a map of `CEL` expressions as parameters
and returns a function that takes an `Unstructured` object and returns a `status.Result` object.

````go
import (
  "sigs.k8s.io/cli-utils/pkg/kstatus/status"
  "github.com/fluxcd/pkg/runtime/cel"
  "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func genericConditions(kind string, exprs map[string]string) func(u *unstructured.Unstructured) (*status.Result, error) {
  return func(u *unstructured.Unstructured) (*status.Result, error) {
		obj := u.UnstructuredContent()

		for statusKey, expr := range exprs {
      // Use CEL to evaluate the expression
			result, err := cel.ProcessExpr(expr, obj)
			if err != nil {
				return nil, err
			}
			switch statusKey {
			case status.CurrentStatus.String():
      // If the expression evaluates to true, we return the current status
			case status.FailedStatus.String():
      // If the expression evaluates to true, we return the failed status
			case status.InProgressStatus.String():
      // If the expression evaluates to true, we return the reconciling status
			}
		}
	}
}
````

The generic status reader will be used by the `statusPoller` provided to the `reconciler`
to compute the status of the resources for the registered custom resources `kind`.

We will provide a `CEL` environment that will use the Kubernetes CEL library to
evaluate the `CEL` expressions.

### StatusPoller configuration

The `reconciler` holds a `statusPoller` that is used to compute the status of the
resources during the `healthCheck` phase of the reconciliation. The `statusPoller`
is configured with a list of `statusReaders` that are used to compute the status
of the resources.

The `statusPoller` is not configurable once instantiated. This means
that we cannot add new `statusReaders` to the `statusPoller` once it is created.
This is a problem for custom resources because we need to be able to add new
`statusReaders` for each new custom resource that is declared in the `Kustomization`
object's `customHealthChecksExprs` field. Fortunately, the `cli-utils` library has
been forked in the `fluxcd` organization and we can make a change to the `statusPoller`
exposed the `statusReaders` field so that we can add new `statusReaders` to it.


The `statusPoller` used by `kustomize-controller` will be updated for every reconciliation
in order to add new polling options for custom resources that have a `CustomHealthChecksExprs`
field defined in their `Kustomization` object.

### K8s CEL Library

The `K8s CEL Library` is a library that provides `CEL` functions to help in evaluating
`CEL` expressions on `Kubernetes` objects.

Unfortunately, this means that we will need to follow the `K8s CEL Library` releases
in order to make sure that we are using the same version of the `CEL` library as
`Kubernetes`. As of the time of writing this RFC, the `K8s CEL Library` is using the
`v0.16.1` version of the `CEL` library while the latest version of the `CEL` library
is `v0.18.2`. This means that we will need to use the `v0.16.1` version of the `CEL`
library in order to be able to use the `K8s CEL Library`.


## Implementation History

See current POC implementation under https://github.com/souleb/kustomize-controller/tree/cel-based-custom-health
