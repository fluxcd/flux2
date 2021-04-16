# Chat based notifications

When operating a cluster, you may wish to receive notifications about the clusters status in a messaging service.

For example, the status of a deployment, GitResource, or Kustomization.

This guide will walk you through how to setup Chat based notifications, for a variety of messaging providers.

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

First create a secret with your incoming webhook:

```sh
kubectl -n flux-system create secret generic webhook-url \
--from-literal=address=https://webhooks.com/services/YOUR/WEBHOOK
```

Note that the secret must contain an `address` field,

it can be a Slack, Microsoft Teams, Discord, Google Chat or Rocket webhook URL.

Create a notification provider for your service by referencing the above secret:

=== "Slack"
    ```bash
    $ flux create alert-provider slack \
      --type slack \
      --secret-ref webhook-url \
      --channel general \
      --export > slack-provider.yaml
    ---
    # slack-provider.yaml
    apiVersion: notification.toolkit.fluxcd.io/v1beta1
    kind: Provider
    metadata:
      name: slack
      namespace: flux-system
    spec:
      type: slack
      channel: general
      secretRef:
        name: webhook-url
    ```

=== "Discord"

    ```bash
    $ flux create alert-provider discord \
      --type discord \
      --secret-ref webhook-url \
      --channel general \
      --username flux \
      --export > discord-provider.yaml
    ---
    # discord-provider.yaml
    apiVersion: notification.toolkit.fluxcd.io/v1beta1
    kind: Provider
    metadata:
      name: discord
      namespace: flux-system
    spec:
      channel: notifications-alerts-webhooks-example
      secretRef:
        name: webhook-url
      type: discord
      username: notifications-alerts-webhooks-example-bot

    ```
=== "Microsoft Teams"
    ```bash
    $ flux create alert-provider teams \
      --type msteams \
      --secret-ref webhook-url \
      --channel general \
      --export > teams-provider.yaml
    ---
    # teams-provider.yaml
    apiVersion: notification.toolkit.fluxcd.io/v1beta1
    kind: Provider
    metadata:
      name: teams
      namespace: flux-system
    spec:
      channel: general
      secretRef:
        name: webhook-url
      type: msteams
    ---
=== "Google Chat"
    ```bash
    $ flux create alert-provider google-chat \
      --type googlechat \   
      --secret-ref webnook-url \
      --export > gchat-provider.yaml
    ---
    # gchat-provider.yaml
    apiVersion: notification.toolkit.fluxcd.io/v1beta1
    kind: Provider
    metadata:
      name: google-chat
      namespace: flux-system
    spec:
      secretRef:
        name: webhook-url
      type: googlechat
    ```

The provider type can be `slack`, `discord`, `msteams`, `googlechat`, `rocket`, `sentry`, `webex` or `generic`

When type `generic` is specified, the notification controller will post the incoming
[event](../components/notification/event.md) in JSON format to the webhook address.
This way you can create custom handlers that can store the events in
Elasticsearch, CloudWatch, Stackdriver, etc.

## Define an alert

Create an alert definition for all repositories and kustomizations:

```bash
flux create alert teams-alert \
--provider-ref teams \
--event-severity info \
--event-source Kustomization/'*' \
--event-source GitRepository/'*' \
--namespace flux-system

```
```yaml
apiVersion: notification.toolkit.fluxcd.io/v1beta1
kind: Alert
metadata:
  name: on-call-webapp
  namespace: flux-system
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
$ kubectl -n flux-system get alerts

NAME             READY   STATUS        AGE
on-call-webapp   True    Initialized   1m
```

Multiple alerts can be used to send notifications to different channels or Slack organizations.
The event severity can be set to `info` or `error`.
When the severity is set to `error`, the kustomize controller will alert on any error encountered during the reconciliation process.
This includes kustomize build and validation errors, apply errors and health check failures.

![error alert](../_files/slack-error-alert.png)

When the verbosity is set to `info`, the controller will alert if:

* a Kubernetes object was created, updated or deleted
* heath checks are passing
* a dependency is delaying the execution
* an error occurs

![info alert](../_files/slack-info-alert.png)
