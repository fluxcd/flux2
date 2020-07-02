# Notification Controller

The Notification Controller is a Kubernetes operator,
specialized in dispatching events to external systems such as
Slack, Microsoft Teams, Discord and Rocket chat.

The controller receives events via HTTP and dispatch them to external
webhooks based on event severity and involved objects.

The controller can be configured with Kubernetes custom resources that
define how events are processed and where to dispatch them.

Links:

- Source code [fluxcd/notification-controller](https://github.com/fluxcd/notification-controller)
- Specification [docs](https://github.com/fluxcd/notification-controller/tree/master/docs/spec)
