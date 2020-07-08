# Contributing

The GitOps Toolkit is [Apache 2.0 licensed](https://github.com/fluxcd/toolkit/blob/master/LICENSE)
and accepts contributions via GitHub pull requests. This document outlines
some of the conventions on to make it easier to get your contribution accepted.

We gratefully welcome improvements to issues and documentation as well as to
code.

## Certificate of Origin

By contributing to this project you agree to the Developer Certificate of
Origin (DCO). This document was created by the Linux Kernel community and is a
simple statement that you, as a contributor, have the legal right to make the
contribution. No action from you is required, but it's a good idea to see the
[DCO](DCO) file for details before you start contributing code to FluxCD
organization.

## Communications

The project uses Slack: To join the conversation, simply join the
[CNCF](https://slack.cncf.io/) Slack workspace and use the
[#flux-dev](https://cloud-native.slack.com/messages/flux-dev/) channel.

The developers use a mailing list to discuss development as well.
Simply subscribe to [flux-dev on cncf.io](https://lists.cncf.io/g/cncf-flux-dev)
to join the conversation (this will also add an invitation to your
Google calendar for our [Flux
meeting](https://docs.google.com/document/d/1l_M0om0qUEN_NNiGgpqJ2tvsF2iioHkaARDeh6b70B0/edit#)).

### How to run the test suite

You can run the unit tests by simply doing

```bash
make test
```

## Acceptance policy

These things will make a PR more likely to be accepted:

- a well-described requirement
- tests for new code
- tests for old code!
- new code and tests follow the conventions in old code and tests
- a good commit message (see below)
- all code must abide [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- names should abide [What's in a name](https://talks.golang.org/2014/names.slide#1)
- code must build on both Linux and Darwin, via plain `go build`
- code should have appropriate test coverage and tests should be written
  to work with `go test`

In general, we will merge a PR once one maintainer has endorsed it.
For substantial changes, more people may become involved, and you might
get asked to resubmit the PR or divide the changes into more than one PR.

### Format of the Commit Message

For the GitOps Toolkit controllers we prefer the following rules for good commit messages:

- Limit the subject to 50 characters and write as the continuation
  of the sentence "If applied, this commit will ..."
- Explain what and why in the body, if more than a trivial change;
  wrap it at 72 characters.

The [following article](https://chris.beams.io/posts/git-commit/#seven-rules)
has some more helpful advice on documenting your work.

## Understanding the GitOps Toolkit

If you are entirely new to the GitOps Toolkit,
you might want to take a look at the [introductory talk and demo](https://www.youtube.com/watch?v=qQBtSkgl7tI).

This project is composed of:

- [/f/toolkit](https://github.com/fluxcd/toolkit): The GitOps Toolkit CLI
- [/f/source-manager](https://github.com/fluxcd/source-controller): Kubernetes operator for managing sources
- [/f/kustomize-controller](https://github.com/fluxcd/kustomize-controller): Kubernetes operator for building GitOps pipelines with Kustomize
- [/f/helm-controller](https://github.com/fluxcd/helm-controller): Kubernetes operator for building GitOps pipelines with Helm
- [/f/notification-controller](https://github.com/fluxcd/notification-controller): Kubernetes operator for handling inbound and outbound events
