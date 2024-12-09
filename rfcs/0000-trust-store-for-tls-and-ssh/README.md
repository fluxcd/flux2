# RFC-NNNN Trust Store for TLS and SSH

**Status:** provisional

**Creation date:** 2022-12-02

**Last update:** 2022-12-02

## Summary

Consolidates and formalizes the supported ways to establish trusted connections
with remote servers via Transport Layer Security (TLS) and Secure Shell (SSH).
Resulting on new ways to configure trust, and allowing administrators the
capability to disable some of the existing options.

## Motivation

The current model could be improved by allowing for controller-level trust
configurations, so that multiple objects connecting to the same server don't
need to specify overrides. The approach aligns with both TLS and SSH canonical
OS level implementations, in which they rely on a global trust store to define
machine level trust, but users and applications can further expand on trusted
servers (or CAs), when not blocked by administrators.

Known Hosts (used by SSH connections), and CA Bundles (for TLS), are not
particularly sensitive information - when leaving aside privacy considerations.
Before this RFC, the officially supported approach leans on secrets to pass this
information to the controllers. The same secret is also use to provide user
credentials, which is more sensitive in nature, making this sub-optimal from a
security stand-point.

### Goals

- Consolidate the officially supported trust settings across the Flux ecosystem.
- Formalize support for configuring trust at controller-level.
- Add toggle to block object-level trust overrides.
- Enable users to surface trust information securely.

### Non-Goals

- Maintain backwards compatibility with older versions of Flux.

## Proposal

For configuring system-wide trust, Flux would rely on the well-established OS-level
trust stores. When dynamically mounting of the trust store is required, it will be
enabled by using Kubernetes `Secret` and `ConfigMap` mounting. When immutable trust
store is required, users can build their own version of the controllers, with their
baked-in settings.

TLS and SSH use different techniques to establish the identity of remote servers,
each relying on its own trust store.

The sections below will dive into the specifics of each one, highlighting their
details, changes required and example of the proposed usage.

A new way to configure object-level Trust Store overrides is also being proposed,
in combination with a controller level toggle to disable it.

### SSH

In SSH, the remote server identity is based on [Trust on first use]. At first
connection to a new server, the user confirms whether or not to trust that server
based on the server's Public key fingerprint.

In the context of Flux, which provides no user interaction, if the remote server
finger print is not configured within the provided set of Known Hosts, the
connection is aborted.

#### Controller-level Known Hosts

For setting controller level Known Hosts, we propose the use of the existing
Linux file in disk: [/etc/ssh/ssh_known_hosts].

Users would be able to configure the OS level trust store by mounting either
a `ConfigMap` or `Secret` directly into the Flux Controllers.

`ConfigMap` example:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: flux-trust-store
  namespace: flux-system
data:
  known_hosts: |
    github.com ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBEmKSENjQEezOmxkZMy7opKgwFB9nkt5YRrYMjNuG5N87uRgg6CLrbo5wAdT/y6v0mKV0U2w0WZ2YB/++Tpockg=
```

Patch required on the main `kustomization.yaml`:
```yaml
  - patch: |
      - op: add
        path: /spec/template/spec/containers/0/volumeMounts
        value:
          - name: ssh-trust-store
            mountPath: /etc/ssh/ssh_known_hosts
            subPath: known_hosts
            readOnly: true
      - op: add
        path: /spec/template/spec/volumes
        value:
          - name: ssh-trust-store
            configMap:
              name: flux-trust-store
    target:
      kind: Deployment
      name: "(kustomize|image-automation|source|image-reflector|helm|notification)-controller"
```

#### Object-level Known Hosts Expansion

A new field is to be introduced into the existing kinds `ImageUpdateAutomation` and
`GitRepository`, to allow users to expand on the controller-level known hosts for
SSH operations:
```
spec:
  trustStore:
    ssh:
        secretRef:
        configMapRef:
```

The trust store can be expanded by either setting `spec.trustStore.ssh.secretRef` or
`spec.trustStore.ssh.configMapRef`, not both. Either option should contain the data
under a `known_hosts` key.

Known hosts configured this way will be aggregated with the ones defined at both
system and controller levels.

#### Pre-populated trust store

Flux container images would be pre-populated with [/etc/ssh/ssh_known_hosts] from
the main Git SaaS providers. As a result, users will only need to update their SSH
Trust Store for custom or less well known servers.

#### TLS

In TLS, the remote server identity is based on [public key infrastructure] and the
trust is based on the confirmation that the remote server's certificate was issued
by a "trusted" Certificate Authority (CA).

The OS level trust store contains the root trusted CAs, and any other certificate
that should be trusted by the machine. Note that CAs can verify other CAs, providing
an hierarchical chain of trust. Certificates that are not part of the chain, which
could be your own self-signed certificates, are considered untrustworthy by default.
TLS communications against untrusted remote servers are aborted.

#### Controller-level Trusted Certificates

**Note:** this requires no changes to the controllers, as this is based on the ways
in which TLS surface the trust store. This RFC only formalizes it as a supported
approach.

To trust CAs that are not part of the root trusted CAs, the OS level trust store
needs to be updated by mounting either a `ConfigMap` or `Secret` directly into the
Flux Controllers.

`Secret` example:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: flux-trust-store
  namespace: flux-system
data:
  customCA.pem: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJoekNDQVMyZ0F3SUJBZ0lVZHNBdGlYM2dOMHVrN2RkeEFTV1lFL3RkdjB3d0NnWUlLb1pJemowRUF3SXcKR1RFWE1CVUdBMVVFQXhNT1pYaGhiWEJzWlM1amIyMGdRMEV3SGhjTk1qQXdOREUzTURneE9EQXdXaGNOTWpVdwpOREUyTURneE9EQXdXakFaTVJjd0ZRWURWUVFERXc1bGVHRnRjR3hsTG1OdmJTQkRRVEJaTUJNR0J5cUdTTTQ5CkFnRUdDQ3FHU000OUF3RUhBMElBQks3aC81RDhiVjkzTW1FZGh1MDJKc1M2dWdCOHM2UHpSbDNQVjR4czNTYnIKUk5ra001OSt4M2IwaVd4L2k3NnFQWXBOTG9pVlVWWFFtQTlZKzREYk14aWpVekJSTUE0R0ExVWREd0VCL3dRRQpBd0lCQmpBUEJnTlZIUk1CQWY4RUJUQURBUUgvTUIwR0ExVWREZ1FXQkJRR3lVaVUxUUVaaU1BcWpzbklZVHdaCjR5cDV3ekFQQmdOVkhSRUVDREFHaHdSL0FBQUJNQW9HQ0NxR1NNNDlCQU1DQTBnQU1FVUNJUUR6ZHR2S2RFOE8KMStXUlRaOU11U2lGWWNyRXo3Wm5lN1ZYb3VERUtxS0VpZ0lnTTRXbGJEZXVOQ0ticWhxait4WlYwcGEzcndlYgpPRDhFampDTVk2OVJNTzA9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
```

Patch required on the main `kustomization.yaml`:
```yaml
  - patch: |
      - op: add
        path: /spec/template/spec/containers/0/volumeMounts
        value:
          - name: tls-trust-store
            mountPath: /etc/ssl/certs/ca-cert-flux.pem
            subPath: customCA.pem
            readOnly: true
      - op: add
        path: /spec/template/spec/volumes
        value:
          - name: tls-trust-store
            secret:
              secretName: flux-trust-store
    target:
      kind: Deployment
      name: "(kustomize|image-automation|source|image-reflector|helm|notification)-controller"
```

#### Object-level Trusted Certificates Expansion

A new field is to be introduced into the existing kinds `Bucket`, `GitRepository`,
`HelmRepository`, `OCIRepository`, `ImageUpdateAutomation`, `Provider` and
`ImageRepository`, to allow users to expand on trusted CAs at controller-level for
HTTPS operations:

```yaml
spec:
  trustStore:
    tls:
        secretRef:
        configMapRef:
```

The trust store can be expanded by either setting `spec.trustStore.tls.secretRef`
or `spec.trustStore.tls.configMapRef`, not both. Either option should contain the
data under a `caFile` key.

CA bundles configured this way will be aggregated with the ones defined at both
system and controller levels.

#### Pre-populated trust store

Flux container images already come with pre-populated CA roots, which are
automatically updated by the Linux distribution used on the base images.
As a result, users only need to update their TLS Trust Store when acessing
web servers using certificates that were not signed by a Publicly trusted CA.

### Enabling Object-Level Trust Store

Object-level trust store expansion is disabled by default. To enable it start
the controller with:

`--insecure-object-trust-store={tls-only,ssh-only,both}`

The flag defaults to `disabled`:
`--insecure-object-trust-store={disabled}`

### User Stories

#### Story 1

> As a tenant, I want to be able to expand trust settings so that I can
> connect to my own servers without needing to ask an administrator.

#### Story 2

> As a Platform admin, I want to configure all trusted servers at the controller
> level and block any specific team from overriding those settings.

#### Story 3

> As a Security Auditor, I want to be able to review all Known Hosts and CA Bundles
> being used within a Flux instance, without requiring RBAC access to more sensitive
> information.

### Alternatives

#### Consume controller-level settings via two new flags

Two new flags would be added into the controllers (`--tls-ca-bundles-secret` and
`--ssh-known-hosts-secret`) allowing for secrets to be consumed at startup time.

This would establish a "flux-specific" approach, which would not be aligned with
existing tools and applications that may need to coexist in the same container,
meaning that a Flux controller may trust a server, whilst other applications within
the container would not - or vice-versa.

#### Remove object-level trust store settings

Instead of creating a toggle to disable object-level trust settings, the entire
feature could have been deprecated. We have decided that by keeping the feature
in would allow for an easier transition.

#### Skip the implementation of the object-level blocker

Instead of creating a built-in feature to block the use of object-level Trust
Store expansion, we could rely on other tools and mechanisms within the Kubernetes
ecosystem (e.g. OPA) to enable users to achieve the same outcome.

Due to the importance that Flux has in the bootstrapping of clusters, such an
important requirement (enforce trust at controller level) should be inherit to
the controllers, instead of delegated to third party components.

## Design Details

### Auto-populating SSH Trust Store

Flux container images that access Git SSH servers (e.g. Source Controller, Image
Automation Controller and Flux CLI) will contain entries on [/etc/ssh/ssh_known_hosts]
for the most popular Git SaaS providers.

Each provider will contain one entry for each supported host key algorithm.
The `ssh_known_hosts` will be a static file in the respective repositories, and
the Dockerfile will simply copy it into the final image.

The known hosts will be updated via automation, which will issue PRs for the maintainers
to review and then approve. As a result, the trusted known hosts will be deterministic
based on the container image version used, in the same way that CAs are.

### Refreshing Controller-level Trust Store Values

The proposed approach heavily relies on built-in functionality in Kubernetes
and Linux distributions. Therefore, the disk contents will be automatically
refreshed when either [Secrets] or [ConfigMaps] are changed.

All SSH operations would need to read the file again for each operation, which
is analogous to the existing "load from memory" approach in place.

For TLS, this value is cached on first use and won't be refreshed until the
controller is restarted. In some instances, the recurrent failure by the
controller to establish connections with a remote server could cause the Pod
to be restarted, resulting in the TLS certs being refreshed.

[Secrets]: https://kubernetes.io/docs/concepts/configuration/secret/#mounted-secrets-are-updated-automatically
[ConfigMaps]: https://kubernetes.io/docs/concepts/configuration/configmap/#mounted-configmaps-are-updated-automatically

### CA Trust Location and Auto Discovery

**Note:** this requires no changes to the controllers. The below only calls out
the existing Go standard library behavior.

The CA Trust Store location `/etc/ssl/certs/` referenced here is the default
location in Alpine distros, which is what is currently used across all Flux
images. Users can use other default locations, as per defined in the [Go standard library].
Another option is to define a custom CA Trust Store via [SSL_CERT_DIR].

On first Transport creation, Go will load any bundled `.crt` files and then
append any unique `.pem` files which are inside the certificate directory.
Therefore, from a Go perspective, new `.pem` files will be taken into account,
even when they are not bundled into the default `/etc/ssl/certs/ca-certificates.crt`.

[Go standard library]: https://github.com/golang/go/blob/master/src/crypto/x509/root_linux.go#L18
[SSL_CERT_DIR]: https://github.com/golang/go/blob/master/src/crypto/x509/root_unix.go#L53

### SSH and TLS references 

The new fields `spec.trustStore.tls` and `spec.trustStore.ssh` analogous
to Kubernetes `EnvFromSource`, in which it can be used to define either a
`configMapRef` or a `secretRef`, but not both.

## Implementation History

<!--
Major milestones in the lifecycle of the RFC such as:
- The first Flux release where an initial version of the RFC was available.
- The version of Flux where the RFC graduated to general availability.
- The version of Flux where the RFC was retired or superseded.
-->

[/etc/ssh/ssh_known_hosts]: https://en.wikibooks.org/wiki/OpenSSH/Client_Configuration_Files#/etc/ssh/ssh_known_hosts
[public key infrastructure]: https://en.wikipedia.org/wiki/Public_key_infrastructure
[Trust on first use]: https://en.wikipedia.org/wiki/Trust_on_first_use
