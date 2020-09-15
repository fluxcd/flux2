# Setup Notifications

When operating a cluster, different teams may wish to receive notifications about
the status of their GitOps pipelines.
For example, the on-call team would receive alerts about reconciliation
failures in the cluster, while the dev team may wish to be alerted when a new version
of an app was deployed and if the deployment is healthy.

## Prerequisites

To follow this guide you'll need a Kubernetes cluster with the GitOps
toolkit controllers installed on it.
Please see the [get started guide](../get-started/index.md)
or the [installation guide](installation.md).

The GitOps toolkit controllers emit Kubernetes events whenever a resource status changes.
You can use the [notification-controller](../components/notification/controller.md)
to forward these events to Slack, Microsoft Teams, Discord or Rocket chart.
The notification controller is part of the default toolkit installation.

## Define a provider

First create a secret with your Slack incoming webhook:

```sh
kubectl -n gotk-system create secret generic slack-url \
--from-literal=address=https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK
```

Note that the secret must contain an `address` field,
it can be a Slack, Microsoft Teams, Discord or Rocket webhook URL.

Create a notification provider for Slack by referencing the above secret:

```yaml
apiVersion: notification.toolkit.fluxcd.io/v1alpha1
kind: Provider
metadata:
  name: slack
  namespace: gotk-system
spec:
  type: slack
  channel: general
  secretRef:
    name: slack-url
```

The provider type can be `slack`, `msteams`, `discord`, `rocket` or `generic`.

When type `generic` is specified, the notification controller will post the incoming
[event](../components/notification/event.md) in JSON format to the webhook address.
This way you can create custom handlers that can store the events in
Elasticsearch, CloudWatch, Stackdriver, etc.

## Define an alert

Create an alert definition for all repositories and kustomizations:

```yaml
apiVersion: notification.toolkit.fluxcd.io/v1alpha1
kind: Alert
metadata:
  name: on-call-webapp
  namespace: gotk-system
spec:
  providerRef:
    name: slack
  eventSeverity: info
  eventSources:
    - kind: GitRepository
      name: '*'
    - kind: Kustomization
      name: '*'
```

Apply the above files or commit them to the `fleet-infra` repository.

To verify that the alert has been acknowledge by the notification controller do:

```console
$ kubectl -n gotk-system get alerts

NAME             READY   STATUS        AGE
on-call-webapp   True    Initialized   1m
```

Multiple alerts can be used to send notifications to different channels or Slack organizations.

The event severity can be set to `info` or `error`.
When the severity is set to `error`, the kustomize controller will alert on any error
encountered during the reconciliation process.
This includes kustomize build and validation errors,
apply errors and health check failures.

![error alert](../diagrams/slack-error-alert.png)

When the verbosity is set to `info`, the controller will alert if:

* a Kubernetes object was created, updated or deleted
* heath checks are passing
* a dependency is delaying the execution
* an error occurs

![info alert](../diagrams/slack-info-alert.png)

## GitHub commit status

The GitHub provider is a special kind of notification provider that based on the
state of a Kustomization resource, will update the
[commit status](https://docs.github.com/en/github/collaborating-with-issues-and-pull-requests/about-status-checks) for the currently reconciled commit id.

The resulting status will contain information from the event in the format `{{ .Kind }}/{{ .Name }} - {{ .Reason }}`.
![github commit status](../diagrams/github-commit-status.png)

It is important to note that the referenced provider needs to refer to the
same GitHub repository as the Kustomization originates from. If these do
not match the notification will fail as the commit id will not be present.

```yaml
apiVersion: notification.toolkit.fluxcd.io/v1alpha1
kind: Provider
metadata:
  name: podinfo
  namespace: gotk-system
spec:
  type: github
  channel: general
  address: https://github.com/stefanprodan/podinfo
  secretRef:
    name: github
---
apiVersion: notification.toolkit.fluxcd.io/v1alpha1
kind: Alert
metadata:
  name: podinfo
  namespace: gotk-system
spec:
  providerRef:
    name: podinfo
  eventSeverity: info
  eventSources:
    - kind: Kustomization
      name: podinfo
      namespace: gotk-system
```

The secret referenced in the provider is expected to contain a [personal access token](https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token)
to authenticate with the GitHub API.
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: github
  namespace: gotk-system
data:
  token: <token>
```


