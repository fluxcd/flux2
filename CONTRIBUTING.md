# Contributing

Flux is [Apache 2.0 licensed](https://github.com/fluxcd/flux2/blob/main/LICENSE) and accepts contributions via GitHub pull requests.
This document outlines the conventions to get your contribution accepted.
We gratefully welcome improvements to documentation as well as code contributions.

If you are new to the project, we recommend starting with documentation improvements or
small bug fixes to get familiar with the codebase and the contribution process.

## Project Structure

The Flux project consists of a set of Kubernetes controllers and tools that implement the GitOps pattern.
The main repositories in the Flux project are:

- [fluxcd/flux2](https://github.com/fluxcd/flux2): The Flux distribution and command-line interface (CLI)
- [fluxcd/pkg](https://github.com/fluxcd/pkg): The GitOps Toolkit Go SDK for building Flux controllers and CLI plugins
- [fluxcd/source-controller](https://github.com/fluxcd/source-controller): Kubernetes operator for managing sources (Git, OCI and Helm repositories, S3-compatible Buckets)
- [fluxcd/source-watcher](https://github.com/fluxcd/source-watcher): Kubernetes operator for advanced source composition and decomposition patterns
- [fluxcd/kustomize-controller](https://github.com/fluxcd/kustomize-controller): Kubernetes operator for building GitOps pipelines with Kustomize
- [fluxcd/helm-controller](https://github.com/fluxcd/helm-controller): Kubernetes operator for lifecycle management of Helm releases
- [fluxcd/notification-controller](https://github.com/fluxcd/notification-controller): Kubernetes operator for handling inbound and outbound events (alerts and webhook receivers)
- [fluxcd/image-reflector-controller](https://github.com/fluxcd/image-reflector-controller): Kubernetes operator for scanning container registries for new image tags and digests
- [fluxcd/image-automation-controller](https://github.com/fluxcd/image-automation-controller): Kubernetes operator for patching container image tags and digests in Git repositories
- [fluxcd/website](https://github.com/fluxcd/website): The Flux documentation website accessible at <https://fluxcd.io/>

## Certificate of Origin

By contributing to this project you agree to the Developer Certificate of Origin (DCO).
This document was created by the Linux Kernel community and is a simple statement that you,
as a contributor, have the legal right to make the contribution.

We require all commits to be signed. By signing off with your signature, you certify that you wrote
the patch or otherwise have the right to contribute the material by the rules of the [DCO](https://raw.githubusercontent.com/fluxcd/flux2/refs/heads/main/DCO):

`Signed-off-by: Jane Doe <jane.doe@example.com>`

The signature must contain your real name (sorry, no pseudonyms or anonymous contributions).
If your `user.name` and `user.email` are set in your Git config,
you can sign your commit automatically with `git commit -s`.

## Acceptance policy

These things will make a PR more likely to be accepted:

- Addressing an open issue, if one doesn't exist, please open an issue to discuss the problem and the proposed solution before submitting a PR.
- Flux is GA software and we are committed to maintaining backward compatibility. If your contribution introduces a breaking change, expect for your PR to be rejected.
- New code and tests must follow the conventions in the existing code and tests. All new code must have good test coverage and be well documented.
- All top-level Go code and exported names should have doc comments, as should non-trivial unexported type or function declarations.
- Before submitting a PR, make sure that your code is properly formatted by running `make tidy fmt vet` and that all tests are passing by running `make test`.

In general, we will merge a PR once one maintainer has endorsed it.
For substantial changes, more people may become involved, and you might
get asked to resubmit the PR or divide the changes into more than one PR.

## Format of the Commit Message

For the Flux project we prefer the following rules:

- Limit the subject to 50 characters, start with a capital letter and do not end with a period.
- Explain what and why in the body, if more than a trivial change; wrap it at 72 characters.
- Use the imperative mood in the subject line (e.g., "Add support for X" instead of "Added support for X" or "Adds support for X").
- Do not include the issue number in the commit message, use the PR description instead (e.g., "Fixes #123" or "Closes #123").
- Do not include GitHub mentions (e.g., `@username` or `@team`) within the commit message.

## Pull Request Process

Fork the repository and create a new branch for your changes, do not commit directly to the `main` branch.
Once you have made your changes and committed them, push your branch to your fork and open a pull request
against the `main` branch of the Flux repository.

During the review process, you may be asked to make changes to your PR. Add commits to address the feedback
without force pushing, as this will make it easier for reviewers to see the changes.
Before committing, make sure to run `make test` to ensure that your code will pass the CI checks.

When the review process is complete, you will be asked to **squash** the commits and **rebase** your branch.
**Do not merge** the `main` branch into your branch, instead, rebase your branch on top of the latest `main`
branch after **syncing your fork** with the latest changes from the Flux repository. After rebasing,
you can push your branch with the `--force-with-lease` option to update the PR.

## Communications

For realtime communications we use Slack. To reach out to the Flux maintainers and contributors,
join the [CNCF](https://slack.cncf.io/) Slack workspace and use the [#flux-contributors](https://cloud-native.slack.com/messages/flux-contributors/) channel.
To discuss ideas and specifications we use [GitHub Discussions](https://github.com/fluxcd/flux2/discussions).

For announcements, we use a mailing list as well. Subscribe to
[flux-dev on cncf.io](https://lists.cncf.io/g/cncf-flux-dev), there you can also add calendar invites
to your Google calendar for our [Flux dev meeting](https://docs.google.com/document/d/1l_M0om0qUEN_NNiGgpqJ2tvsF2iioHkaARDeh6b70B0/view).
