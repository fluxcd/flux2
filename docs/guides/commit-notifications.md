# Commit based Notifications

The GitHub, GitLab, Bitbucket, and Azure DevOps providers are slightly different to the other providers. Instead of
a stateless stream of events, the git notification providers will link the event with accompanying git commit which
triggered the event. The linking is done by updating the commit status of a specific commit.

  - [GitHub](https://docs.github.com/en/github/collaborating-with-issues-and-pull-requests/about-status-checks)
  - [GitLab](https://docs.gitlab.com/ee/api/commits.html)
  - [Bitbucket](https://developer.atlassian.com/server/bitbucket/how-tos/updating-build-status-for-commits/)
  - [Azure DevOps](https://docs.microsoft.com/en-us/rest/api/azure/devops/git/statuses?view=azure-devops-rest-6.0)

In GitHub the commit status set by notification-controller will result in a green checkmark or red cross next to the commit hash.
Clicking the icon will show more detailed information about the status.
![commit status GitHub overview](../_files/commit-status-github-overview.png)

Receiving an event in the form of a commit status has the benefit that it closes the deployment loop. This gives quick and visible feedback if a commit has reconciled and if it succeeded. This means deployments operate in a similar manner to "traditional" push based CD.
Status can be fetched from the git providers API for a specific commit. This allows for custom automation tools that can automatically promote, commit to a new directory, after receiving a successful commit status. This can all be done without requiring any access to the Kubernetes cluster

The provider works by referencing the same git repository as the Kustomization controller does.
When a new commit is pushed to the repository, source-controller will sync the commit, triggering the kustomize-controller
to reconcile the new commit. After this is done the kustomize-controller sends an event to the notification-controller
with the result and the commit hash it reconciled. Then notification-controller can update the correct commit and repository when receiving the event.


![commit status flow](../_files/commit-status-flow.png)

!!! hint "Limitations"
    The git notification providers require that a commit hash present in the meta data
    of the event. There for the the providers will only work with `Kustomization` as an
    event source, as it is the only resource which includes this data.

## Prerequisites

### Authentication Token

#### Obtaining your git providers credentials

The token/app-password will need to have write access for the repository it will be updating the commit status in.

Follow the respective guide for your git provider

- [GitHub personal access token](https://docs.github.com/en/free-pro-team@latest/github/authenticating-to-github/creating-a-personal-access-token)
- [GitLab personal access token](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html)
- [Azure DevOps personal access token](https://docs.microsoft.com/en-us/azure/devops/organizations/accounts/use-personal-access-tokens-to-authenticate?view=azure-devops&tabs=preview-page)
- [app password](https://support.atlassian.com/bitbucket-cloud/docs/app-passwords/)

#### Creating Git Secret for your token/app-password

Use the following command to store the token/app-password in a secret

=== "GitHub, GitLab, Azure DevOps"
    ```bash
    $ kubectl -n flux-system create secret generic git \
      --from-literal=token='<YOURTOKEN>' 
    ---
    apiVersion: v1
    data:
      token: PFlPVVJUT0tFTj4=
    kind: Secret
    metadata:
      creationTimestamp: null
      name: git
      namespace: flux-system
    ```
=== "BitBucket"
    ```bash
    $ kubectl -n flux-system create secret generic bitbucket\
      --from-literal=token='<username>:<app-password>' 
    ---
    apiVersion: v1
    kind: Secret
    metadata:
      name: git
      namespace: default
    data:
      token: <username>:<app-password>
    ```

## Creating a Deployment

Copy the manifest content in the "[kustomize](https://github.com/stefanprodan/podinfo/tree/master/kustomize)" directory
into the directory "./clusters/my-cluster/podinfo" in your fleet-infra repository. Make sure that you also add the
namespace podinfo.

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: podinfo
```

Then create a Kustomization to deploy podinfo.
```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1beta1
kind: Kustomization
metadata:
  name: podinfo
  namespace: flux-system
spec:
  interval: 5m
  targetNamespace: podinfo
  path: ./clusters/my-cluster/podinfo
  prune: true
  sourceRef:
    kind: GitRepository
    name: flux-system
  healthChecks:
    - apiVersion: apps/v1
      kind: Deployment
      name: podinfo
      namespace: podinfo
  timeout: 1m
```

## Creating the Git provider

Creating a git provider is similar to creating other providers.
The only difference is that with git providers `spec.address` needs to point to the same git repository that the event source originates from.

=== "GitHub"
    ```yaml
    apiVersion: notification.toolkit.fluxcd.io/v1beta1
    kind: Provider
    metadata:
      name: flux-system
      namespace: flux-system
    spec:
      type: github
      address: https://github.com/<username>/fleet-infra
      secretRef:
        name: git
    ```

=== "GitLab"
    ```yaml
    apiVersion: notification.toolkit.fluxcd.io/v1beta1
    kind: Provider
    metadata:
      name: flux-system
      namespace: flux-system
    spec:
      type: gitlab
      address: https://gitlab.com/<username>/fleet-infra
      secretRef:
        name: git
    ```
=== "Bitbucket"
    ```bash
    ```
=== "Azure DevOps"
    ```bash
    ```

## Creating the alert

=== "GitHub"
    ```bash
    $ flux create alert podinfo \
    --provider-ref git \
    --event-severity info \
    --event-source 'Kustomization/podinfo' \
    --namespace 'flux-system' \
    --export
    ---
    apiVersion: notification.toolkit.fluxcd.io/v1beta1
    kind: Alert
    metadata:
      name: podinfo
      namespace: flux-system
    spec:
      providerRef:
        name: flux-system
      eventSeverity: info
      eventSources:
        - kind: Kustomization
          name: podinfo
          namespace: flux-system
    ```
=== "GitLab"
    ```bash
    ```
=== "Bitbucket"
    ```bash
    ```
=== "Azure DevOps"
    ```bash
    ```


By now the fleet-infra repository should have a similar directory structure.
```
fleet-infra
└── clusters/
    └── my-cluster/
        ├── flux-system/
        │   ├── gotk-components.yaml
        │   ├── gotk-sync.yaml
        │   └── kustomization.yaml
        ├── podinfo/
        │   ├── namespace.yaml
        │   ├── deployment.yaml
        │   ├── hpa.yaml
        │   ├── service.yaml
        │   └── kustomization.yaml
        ├── podinfo-kustomization.yaml
        └── podinfo-notification.yaml
```

If podinfo is deployed and the health checks pass you should get a successful status in
your forked podinfo repository.

If everything is setup correctly there should now be a green check-mark next to the latest commit.
Clicking the check-mark should show a detailed view.

=== "Github"
    ![commit status GitHub successful](../_files/commit-status-github-success.png) 
=== "GitLab"
    ![commit status GitLab successful](../_files/commit-status-gitlab-success.png) |

## Generate an error

A deployment failure can be forced by setting an invalid image tag in the podinfo deployment.
```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: podinfod
        image: ghcr.io/stefanprodan/podinfo:fake
```

After the commit has been reconciled it should return a failed commit status.

=== "GitHub"
    ![commit status GitHub failure](../_files/commit-status-github-failure.png)
=== "GitLab"
    ![commit status GitLab failure](../_files/commit-status-gitlab-failure.png) |


