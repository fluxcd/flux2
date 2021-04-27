# Onboarding a Private Git Repository

This guide walks you through the common use-case of onboarding a private GitRepository source.

The below examples and instructions assume you are using GitHub, but the same 
can apply for other providers and their support for access-token password auth.

## Creating Per-Repo Deploy Keys
For detailed instructions on setting up per-repo deploy keys using SOPS/SSH- or
 token-based Git authentication, see [the documentation in our flux2-multi-tenancy example](https://github.com/fluxcd/flux2-multi-tenancy#onboard-tenants-with-private-repositories).

## Sharing Token Auth Across Repos
If you are managing a multi-tenant cluster with Flux, you can cut a Personal 
Access Token with `repo` access from a GitHub service account and use that 
token for authenticating multiple private GitRepository sources.

The benefits of sharing a token across multiple tenants instead of using 
per-repo deploy keys include:
* Control access for all of Flux using GitHub's ACLs like Teams (composes well
 if you're using Terraform to manage GitHub)
* You only have to manage/create one secret per cluster/namespace instead of 
per-repo.

The tradeoff is that revocation of access with a shared token is less granular 
than on a per-repo basis; you will have to manage access on the org/team/
user-level. 

### HTTP/S Basic Auth Token Access
1. Log into a GitHub service account that has `admin` access to the repo you 
are onboarding, and visit [settings/tokens](https://github.com/settings/tokens) 
to generate a Personal Access Token with `repo` scope. 
2. Bootstrap your cluster with the token. Note that we add the `--token-auth` 
flag, which lets Flux know that it should use your `GITHUB_TOKEN` permanently 
for auth in the cluster.
If you already bootstrapped your cluster when getting started but did not add 
this flag, it is fine to re-run the bootstrap command. 
```sh
export GITHUB_TOKEN=$GITHUB_TOKEN
flux bootstrap github \
  --owner=$GITHUB_USER \
  --repository=fleet-infra \
  --branch=main \
  --path=./clusters/my-cluster \
  --token-auth
```
3.  You can verify that Flux has saved your `GITHUB_TOKEN` for ongoing 
authentication by checking the `flux-system` secret in the `flux-system` 
namespace:
```sh
kubectl get secret flux-system -n flux-system -o yaml
# You should see the `data.username` and `data.password` fields
```
4. When onboarding a new GitRepository that is private, make sure to use the 
HTTP/S protocol in `spec.url` and reference the `flux-system` secret with your 
creds in `spec.secretRef`. 

#### CLI Example
You can do this on the command-line like so:
```sh
flux create source git podinfo \
    --url=https://github.com/my-org/my-private-repo \
    --branch=master \
    --secret-ref=flux-system
```

#### YAML Manifest Example
If you are onboarding your source by checking in a YAML to your platform admin
 repo, it should look like this:
```yaml
apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: GitRepository
metadata:
    name: my-private-repo
    namespace: flux-system
spec:
    interval: 1m
    url: https://github.com/my-org/my-private-repo
    # since we are using a private GitHub repo, the below 
    # section is needed to tell Flux to use the GitHub token in the 
    # flux-system Secret for HTTP/S basic auth when syncing
    secretRef:
        name: flux-system
    ref:
        branch: master
```

### Note on namespaces
**Note** that if you wish to manage your tenant `GitRepository` sources as part of a namespace that is _not_ flux-system (e.g. the `apps` namespace, [as in the multi-tenancy example](https://github.com/fluxcd/flux2-multi-tenancy/blob/main/tenants/base/dev-team/sync.yaml#L5)), the private source will not be able to read from the `flux-system` Secret.

You will have to manually create a Kubernetes Secret with the `data.username` and `data.password` fields in the namespace that you are creating your `GitRepository` sources in. To keep Secrets private, consider using [kubeseal](sealed-secrets.md).
```sh
# for GitRepository private sources stored in the `apps` namespace
kubectl create secret generic repo-auth -n apps \
	--from-literal=username=$GITHUB_USER \
	--from-literal=password=$GITHUB_TOKEN
```

To get your sources to use the Secret, just reference it by name in the `GitRepository.spec.secretRef` field:
```yaml
apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: GitRepository
metadata:
    name: my-private-repo
    namespace: apps
spec:
    interval: 1m
    url: https://github.com/my-org/my-private-repo
    secretRef:
        name: repo-auth
    ref:
        branch: master
```


