# Contributing

Flux is [Apache 2.0 licensed](https://github.com/fluxcd/flux2/blob/main/LICENSE) and
accepts contributions via GitHub pull requests. This document outlines
some of the conventions on to make it easier to get your contribution
accepted.

We gratefully welcome improvements to issues and documentation as well as to
code.

## Certificate of Origin

By contributing to this project you agree to the Developer Certificate of
Origin (DCO). This document was created by the Linux Kernel community and is a
simple statement that you, as a contributor, have the legal right to make the
contribution.

We require all commits to be signed. By signing off with your signature, you
certify that you wrote the patch or otherwise have the right to contribute the
material by the rules of the [DCO](DCO):

`Signed-off-by: Jane Doe <jane.doe@example.com>`

The signature must contain your real name
(sorry, no pseudonyms or anonymous contributions)
If your `user.name` and `user.email` are configured in your Git config,
you can sign your commit automatically with `git commit -s`.

## Communications

For realtime communications we use Slack: To join the conversation, simply
join the [CNCF](https://slack.cncf.io/) Slack workspace and use the
[#flux-contributors](https://cloud-native.slack.com/messages/flux-contributors/) channel.

To discuss ideas and specifications we use [Github
Discussions](https://github.com/fluxcd/flux2/discussions).

For announcements we use a mailing list as well. Simply subscribe to
[flux-dev on cncf.io](https://lists.cncf.io/g/cncf-flux-dev)
to join the conversation (there you can also add calendar invites
to your Google calendar for our [Flux
meeting](https://docs.google.com/document/d/1l_M0om0qUEN_NNiGgpqJ2tvsF2iioHkaARDeh6b70B0/view)).

## Understanding Flux and the GitOps Toolkit

If you are entirely new to Flux and the GitOps Toolkit,
you might want to take a look at the [introductory talk and demo](https://www.youtube.com/watch?v=qQBtSkgl7tI).

This project is composed of:

- [flux2](https://github.com/fluxcd/flux2): The Flux CLI
- [source-manager](https://github.com/fluxcd/source-controller): Kubernetes operator for managing sources (Git and Helm repositories, S3-compatible Buckets)
- [kustomize-controller](https://github.com/fluxcd/kustomize-controller): Kubernetes operator for building GitOps pipelines with Kustomize
- [helm-controller](https://github.com/fluxcd/helm-controller): Kubernetes operator for building GitOps pipelines with Helm
- [notification-controller](https://github.com/fluxcd/notification-controller): Kubernetes operator for handling inbound and outbound events
- [image-reflector-controller](https://github.com/fluxcd/image-reflector-controller): Kubernetes operator for scanning container registries
- [image-automation-controller](https://github.com/fluxcd/image-automation-controller): Kubernetes operator for patches container image tags in Git

### Understanding the code

To get started with developing controllers, you might want to review
[our guide](https://fluxcd.io/docs/gitops-toolkit/source-watcher/) which
walks you through writing a short and concise controller that watches out
for source changes.

## How to run the test suite

Prerequisites:

* go >= 1.17
* kubectl >= 1.20
* kustomize >= 4.4
* coreutils (on Mac OS)

Install the [controller-runtime/envtest](https://github.com/kubernetes-sigs/controller-runtime/tree/master/tools/setup-envtest) binaries with:

```bash
make install-envtest
```

Then you can run the unit tests with:

```bash
make test
```

After [installing Kubernetes kind](https://kind.sigs.k8s.io/docs/user/quick-start#installation) on your machine,
create a cluster for testing with:

```bash
make setup-kind
```

Then you can run the end-to-end tests with:

```bash
make e2e
```

When the output of the Flux CLI changes, to automatically update the golden
files used in the test, pass `-update` flag to the test as:

```bash
make e2e TEST_ARGS="-update"
```

Since not all packages use golden files for testing, `-update` argument must be
passed only for the packages that use golden files. Use the variables
`TEST_PKG_PATH` for unit tests and `E2E_TEST_PKG_PATH` for e2e tests, to set the
path of the target test package:

```bash
# Unit test
make test TEST_PKG_PATH="./cmd/flux" TEST_ARGS="-update"
# e2e test
make e2e E2E_TEST_PKG_PATH="./cmd/flux" TEST_ARGS="-update"
```

Teardown the e2e environment with:

```bash
make cleanup-kind
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
