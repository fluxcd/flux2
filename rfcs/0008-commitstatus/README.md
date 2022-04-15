
# RFC-xxxx Create a new API kind for Commit Status

**Status:** provisional

**Creation date:** 2022-05-12

**Last update:** 2022-05-12

## Summary

There should be a dedicated kind in the notification controller for sending commit status notifications to git providers.

## Motivation

Currently, The Alert type can reference both git providers and chat providers. However, the differences between the two providers and how notifications being sent to them should be handled has continued to diverge. 
For example, there is a limit on the length of the name/context of the status for git provider.

When one commit triggers more than one deployment, the commit status notification from one overwrites the other as seen in this [issue](https://github.com/fluxcd/notification-controller/issues/195).
A new field specific to the git commit status will allow Flux users to specify a prefix that will be prepended to the context to differentiate the statuses.

By creating a separate kind for git commit status, the specific nuances of the provider can be properly taken into account and it would allow more flexibility in adding fields specific to each kind.

### Goals

- Add a new kind `CommitStatus` to Flux notification controller for handling commit status notifications.
- Add support for sending different commit status for different environments/clusters.

### Non-Goals

- Add new `spec` fields to the `Alert` kind.

## Proposal

Introduce a new kind in the notification-controller called `CommitStatus`.
The `CommitStatus` kind will similar to the `Alert` kind, referencing a provider and including event sources to accept events from. The major difference is that it will only reference git providers. Additionally, it can only have a `Kustomization` as its event source since it requires the `revision` field in the event metadata.
It would include a new field `.spec.prefix` that will be used to differentiate results of multiple deployments on different clusters triggered by the same commit and prevent one from overwriting the other.

This proposal will also introduce breaking changes as the `Alert` kind will no longer send notifications to git providers.

### User Stories

If a user want to receive status notification on github for a deployment, a `CommitStatus` kind can be used:

```yaml
apiVersion: notification.toolkit.fluxcd.io/v1beta1
kind: CommitStatus
metadata:
  name: webapp
  namespace: flux-system
spec:
  providerRef:
    name: github # The `spec.type` of the provider has to be github, gitlab, or bitbucket
  eventSeverity: info
  eventSources:
    - kind: Kustomization
      name: webapp
  prefix: dev
```

### Alternatives

Alternatively, we could keep using the `Alert` kind for sending notifications to git providers. While this might not be much of a pain now, it would continue to grow as the implementations details of new features might differ for the chat and git providers.

## Implementation History

- [Implement GitHub commit status notifier](https://github.com/fluxcd/notification-controller/pull/27)
- [Add Gitlab notifier](https://github.com/fluxcd/notification-controller/pull/43)
- [Add bitbucket notifier](https://github.com/fluxcd/notification-controller/pull/73)
- [Add Azure DevOps provider](https://github.com/fluxcd/notification-controller/pull/86)

