# RFC-0003 Flux Multi-Tenancy Mode

## Summary

For multi-tenant environments, we want to offer an easy way of configuring Flux to enforce tenant isolation
(as defined by the Soft Multi-Tenancy model from RFC-0001).

When running in the multi-tenant mode, Flux will lock down access to sources (as defined by RFC-0002),
and will use the tenant service account instead of defaulting to `cluster-admin`.

From an end-user perspective, the multi-tenancy mode means that:

- Platform admins have to create a Kubernetes service account and RBAC in each namespace where
  Flux performs source-to-cluster reconciliation on behalf of tenants.
  By default, Flux will have no permissions to reconcile the tenants sources onto clusters.
- Source owners have to specify with which tenants they wish to share their sources.
  By default, nothing is shared between tenants.

## Motivation

As of Flux v0.23.0, configuring Flux for soft multi-tenancy requires additional tooling such as Kyverno or OPA Gatekeeper
to overcome caveats such as:
- Flux does not require for a service account name to be specified on Flux custom resources that perform
  source-to-cluster reconciliation. When a service account is not specified, Flux defaults to cluster-admin.
- Flux does not prevent tenants from accessing known sources outside of their namespaces.
- Flux does not prevent tenants from subscribing to other tenant's events.

Flux users have been asking for a way to enforce multi-tenancy
without having to use 3rd party validation webhooks e.g.
[fluxcd/kustomize-controller#422](https://github.com/fluxcd/kustomize-controller/issues/422).

### Goals

- Enforce service account impersonation for source-to-cluster reconciliation.
- Enforce ACLs for cross-namespace access to sources.

### Non-Goals

- Enforce tenant's workload isolation with network policies and pod security standards as described
  [here](https://kubernetes.io/blog/2021/04/15/three-tenancy-models-for-kubernetes/#security-considerations).

## Proposal

### User Stories

#### Story 1

> As a platform admin, I want to install Flux with lowest privilege/permission level possible.

#### Story 2

> As a platform admin, I want to give tenants full control over their assigned namespaces.
> So that tenants could use their own repositories and manager the app delivery with Flux.

#### Story 3

> As a platform admin, I want to prevent tenants from changing the cluster-wide configuration.
> If a tenant adds to their repository a cluster-scoped resource such as a namespace or cluster role,
> Flux should reject the change and notify the tenant that this operation is not allowed.

### Multi-tenant Bootstrap

When bootstrapping Flux, platform admins should have the option to lock down Flux for multi-tenant environments e.g.:

```shell
flux bootstrap --security-profile=multi-tenant
```

The security profile flag accepts two values: `single-tenant` and `multi-tenant`.
Platform admins may switch between the two modes at any time, either by rerunning bootstrap
or by patching the Flux manifests in Git.

The `multi-tenant` profile is just a shortcut to setting the following container args in the Flux deployment manifests:

```yaml
      containers:
      - name: manager
        args:
          - --default-service-account=flux
          - --enable-source-acl=true
```

And for disabling cross-namespace references when using the notification API:

```yaml
kind: Deployment
metadata:
  name: notification-controller
spec:
  template:
    spec:
      containers:
      - name: manager
        args:
          - --same-namespace-refs=true
```

When running in the `multi-tenant` mode, Flux behaves differently:

- The source-to-cluster reconciliation no longer runs under the service account of
  the Flux controllers. The controller service account, is only used to impersonate
  the service account specified in the Flux custom resources (`Kustomizations`, `HelmReleases`).
- When no service account name is specified in a Flux custom resource,
  a default will be used e.g. `system:serviceaccount:<tenant-namespace>:flux`.
- When a Flux custom resource (`Kustomizations`, `HelmReleases`, `ImagePolicies`, `ImageUpdateAutomations`)
  refers to a source in a different namespace, access is granted based the source access control list.
  If no ACL is defined for a source, cross-namespace access is denied.
- When a Flux notification (`Alerts`, `Receivers`)
  refers to a resource in a different namespace, access is denied.

### Tenants Onboarding

When onboarding tenants, platform admins should have the option to assign namespaces, set
permissions and register the tenants repositories onto clusters in a declarative manner. 

The Flux CLI offers an easy way of generating all the Kubernetes manifests needed to onboard tenants:

- `flux create tenant` command generates namespaces, service accounts and Kubernetes RBAC
  with restricted access to the cluster resources, given tenants access only to their namespaces.
- `flux create secret git` command generates SSH keys used by Flux to clone the tenants repositories.
- `flux create source git` command generates the configuration that tells Flux which repositories belong to tenants.
- `flux create kustomization` command generates the configuration that tells Flux how to reconcile the manifests found in the tenants repositories.

All the above commands have an `--export` flag for generating the Kubernetes resources in YAML format.
The platform admins should place the generated manifests in the repository that defines the cluster(s) desired state.

Here is an example of the generated manifests:

```yaml
---
apiVersion: v1
kind: Namespace
metadata:
  name: tenant1
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: flux
  namespace: tenant1
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: flux
  namespace: tenant1
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: flux
    namespace: tenant1
---
apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: GitRepository
metadata:
  name: tenant1
  namespace: tenant1
spec:
  interval: 5m0s
  ref:
    branch: main
  secretRef:
    name: tenant1-git-auth
  url: ssh://git@github.com/org/tenant1
---
apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: tenant1
  namespace: tenant1
spec:
  interval: 10m0s
  path: ./
  prune: true
  serviceAccountName: flux
  sourceRef:
    kind: GitRepository
    name: tenant1
```

Note that the [cluster-admin](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#user-facing-roles)
role is used in a `RoleBinding`, this only gives full control over every resource in the role binding's namespace.

Once the tenants repositories are registered on the cluster(s), the tenants can configure their app delivery 
in Git using Kubernetes namespace-scoped resources such as `Deployments`, `Services`, Flagger `Canaries`,
Flux `Kustomizations`, `HelmReleases`, `ImageUpdateAutomations`, `Alerts`, `Receivers`, etc.

## Alternatives

Instead of introducing the security profile flag to `flux bootstrap`,
we could document how to patch each controller deployment with Kustomize.

Having an easy way of locking down Flux with a single flag, make users aware of the security implications
and improves the user experience.
