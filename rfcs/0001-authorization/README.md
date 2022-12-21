# RFC-0001 Memorandum on Flux Authorization

## Summary

This RFC describes in detail, for [Flux version 0.24][] (Nov 2021), how Flux determines which
operations are allowed to proceed, and how this interacts with Kubernetes' access control.

## Motivation

To this point, the Flux project has provided [examples of how to make a multi-tenant
system](https://github.com/fluxcd/flux2-multi-tenancy/tree/v0.1.0), but not explained exactly how
they relate to Flux's authorization model; nor has the authorization model itself been
documented. Further work on support for multi-tenancy, among other things, requires a full account
of Flux's authorization model as a baseline.

### Goals

- Give a comprehensive account of Flux's authorization model

### Non-Goals

- Justify the model as it stands; this RFC simply records the state as at v0.24.

## Flux's authorization model

The Flux controllers undertake operations as specified by custom resources of the kinds defined in
the [Flux API][]. Most of the operations are through the Kubernetes API. Authorization for
operations on external systems is not accounted for here.

Flux controllers defer to [Kubernetes' native RBAC][k8s-rbac] and [namespace isolation][k8s-ns] to
determine which operations are authorized, when processing the custom resources in the Flux API.

In general, **Kubernetes API operations are constrained by the service account under which each
controller pod runs**. In the [default deployment of Flux][flux-rbac] each controller has its own
service account; and, the service accounts for the Kustomize controller and Helm controller have the
[`cluster-admin` cluster role][k8s-cluster-admin] bound to it.

Both the Kustomize controller and the Helm controller create, update and delete arbitrary sets of
configuration that they take as user input. For example, a Kustomization object that references a
GitRepository is processed by taking whatever is in the specified Git repository and applying it to
the cluster. This is informally called "syncing", and these user-supplied configurations will be
called "sync configurations" in the following.

There are five types of access that have a distinct treatment with respect to RBAC and namespace
isolation:

 - reading and writing the Flux API object to be processed
 - accessing dependencies of a Flux API object; for example, a secret that holds a decryption key
 - accessing Flux API objects related to the object being processed; for example, a GitRepository
   referenced by a Kustomization
 - creating, updating and deleting Flux API objects as part of processing; for example, each
   `HelmRelease` object contains a template for a Helm chart spec, which the Helm controller uses to
   create a `HelmChart` object
 - creating, updating, deleting, and health-checking of arbitrary objects as specified by _sync
   configurations_ (as mentioned above).

This table summarises how these operations are subject to RBAC and namespace isolation.

| Type of operation                              | Accessed via               | Namespace isolation          |
|------------------------------------------------|----------------------------|------------------------------|
| Reading and writing the object to be processed | Controller service account | N/A                          |
| Dependencies of object to be processed         | Controller service account | Same namespace only          |
| Access to related Flux API objects             | Controller service account | Some cross-namespace refs[1] |
| CRUD of Flux API objects                       | Controller service account | Created in same namespace    |
| CRUD and healthcheck of sync configurations    | Impersonation[2]           | As directed by spec[2]       |

[1] See "Cross-namespace references" below<br>
[2] See "Impersonation" below

There are two related mechanisms that affect the service account used for the operations marked with
"Impersonation" above: "impersonation" and "remote apply". These are explained in the following
sections.

### Impersonation

The Kustomize controller and Helm controller both apply arbitrary sets of Kubernetes configuration
("_synced configuration_" as above) to a cluster. These controllers use the service account named in
the field `.spec.serviceAccountName` in the `Kustomization` and `HelmRelease` objects respectively,
while applying and health-checking the synced configuration. This mechanism is called
"impersonation".

The `.spec.serviceAccountName` field is optional. If empty, the controller's service account is
used.

### Remote apply

The Kustomize controller and Helm controller are able to apply a set of configuration to a cluster
other than the cluster in which they run. If the `Kustomization` or `HelmRelease` object [refers to
a secret containing a "kubeconfig" file][kubeconfig], the controller will construct a client using
that kubeconfig, and the client is used to apply the prepared set of configuration. The effect of
this is that the configuration will be applied as the user given in the kubeconfig; often this is a
user with the `cluster-admin` role bound to it, but not necessarily so.

All accesses that would use impersonation use the remote client instead.

### Cross-namespace references

Some Flux API kinds have fields which can refer to a Flux API object in another namespace. The Flux
controllers do not respect namespace isolation when dereferencing these fields. The following are
fields that are not restricted to the namespace of the containing object, listed by API kind.

| API kind | field | explanation |
|----------|-------|-------------|
| **`kustomizations.kustomize.toolkit.fluxcd.io/v1beta2`** | `.spec.dependsOn` | Items are references that can include a namespace |
|                                                          | `.spec.healthChecks` | Items are references that can include a namespace (note: these are accessed using impersonation) |
|                                                          | `.spec.sourceRef` | This is a reference that can include a namespace |
|                                                          | `.spec.targetNamespace` | This sets or overrides the namespace given in the top-most `kustomization.yaml` |
| **`helmreleases.helm.toolkit.fluxcd/v2beta1`** | `.spec.dependsOn` | Items are references that can include a namespace |
|                                                | `.spec.targetNamespace` | This gives the namespace into which a Helm chart is installed (note: using impersonation) |
|                                                | `.spec.storageNamespace` | This gives the namespace in which the record of a Helm install is created (note: using impersonation) |
|                                                | `.spec.chart.spec.sourceRef` | This is a reference (in the created `HelmChart` object) that can include a namespace |
| **`alerts.notification.toolkit.fluxcd.io/v1beta2`** | `.spec.eventSources` | Items are references that can include a namespace |
| **`receivers.notification.toolkit.fluxcd.io/v1beta2`** | `.spec.resources` | Items in this field are references that can include a namespace |
| **`imagepolicies.image.toolkit.fluxcd.io/v1beta1`** | `.spec.imageRepositoryRef` | This reference can include a namespace[1] |

[1] This particular cross-namespace reference is subject to additional access control; see "Access
control for cross-namespace references" below.

Note that the field `.spec.sourceRef` of **`imageupdateautomation.image.toolkit.fluxcd.io`** does
_not_ include a namespace.

#### Access control for cross-namespace references

In v0.24, an `ImagePolicy` object can refer to a `ImageRepository` object in another
namespace. Unlike most cross-namespace references, the controller processing `ImagePolicy` objects
applies additional access control, as given in the referenced `ImageRepository`: the field
[`.spec.accessFrom`][access-from-ref] grants access to the namespaces selected therein. Access is
denied unless granted.

## Security considerations

### Impersonation is optional

Flux does not insist on a service account to be supplied in `Kustomization` and `HelmRelease`
specifications, and the default is to use the controller's service account. That means a user with
the ability to create either of those objects can trivially arrange for a configuration to be
applied with the controller service account, which in the default deployment of Flux will have
`cluster-admin` bound to it. This represents a privilege escalation vulnerability in the default
deployment of Flux. To guard against it, an admission controller can be used to make the
`.spec.serviceAccountName` field mandatory; an example which uses Kyverno is given in [the
multi-tenancy implementation][multi-tenancy-eg].

### Cross-namespace references side-step namespace isolation

`HelmRelease` and `Kustomization` objects can refer to `GitRepository`, `HelmRepository`, or
`Bucket` (collectively "sources") in any other namespace. The referenced objects are accessed
through the controller's service account, which by default has `cluster-admin` bound to it. This
means all sources in a cluster are by default usable as a synced configuration, from any
namespace. To restrict access, an admission controller can be used to block cross-namespace
references; the [example using Kyverno][multi-tenancy-eg] from above also does this.

## References

-  [CVE-2021-41254](https://github.com/fluxcd/kustomize-controller/security/advisories/GHSA-35rf-v2jv-gfg7)
  "Privilege escalation to cluster admin on multi-tenant environments" was fixed in flux2 **v0.15.0**.

[Flux version 0.24]: https://github.com/fluxcd/flux2/releases/tag/v0.24.0
[serviceAccountName]: https://fluxcd.io/docs/components/kustomize/api/#kustomize.toolkit.fluxcd.io/v1beta2.KustomizationSpec
[kubeconfig]: https://fluxcd.io/docs/components/kustomize/api/#kustomize.toolkit.fluxcd.io/v1beta2.KubeConfig
[access-from-ref]: https://fluxcd.io/docs/components/image/imagerepositories/#allow-cross-namespace-references
[Flux API]: https://fluxcd.io/docs/components/
[flux-rbac]: https://github.com/fluxcd/flux2/tree/v0.24.0/manifests/rbac
[k8s-ns]: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/
[k8s-rbac]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/
[k8s-cluster-admin]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/#user-facing-roles
[multi-tenancy-eg]: https://github.com/fluxcd/flux2-multi-tenancy/blob/main/infrastructure/kyverno-policies/flux-multi-tenancy.yaml
