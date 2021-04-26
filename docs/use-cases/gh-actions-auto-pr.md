# GitHub Actions Auto PR

In the [Image Update Guide] we saw we can [Push updates to a different branch] by using `.spec.git.push.branch` to push image updates to a different branch than the one used for checkout.

In this example, we configure an `ImageUpdateAutomation` resource to push to a `staging` branch, (which we could set up separately as a preview environment to deploy automatic updates in a staging cluster or namespace.)

```yaml
kind: ImageUpdateAutomation
metadata:
  name: flux-system
spec:
  git:
    checkout:
      ref:
        branch: main
    push:
      branch: staging
```

For this use case, we are only interested in showing that once the change is approved and merged, it gets deployed into production. The image automation is gated behind a pull request approval workflow, according to any policy you have in place for your repository.

In your manifest repository, add a GitHub Action workflow as below. This workflow watches for commits on the `staging` branch and opens a pull request with any labels, title, or body that you configure.

```yaml
# ./.github/workflows/staging-auto-pr.yaml
name: Staging Auto-PR
on:
  push:
    branches: ['staging']

jobs:
  pull-request:
    name: Open PR to main
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v2
      name: checkout

    - uses: repo-sync/pull-request@v2
      name: pull-request
      with:
        destination_branch: "main"
        pr_title: "Pulling ${{ github.ref }} into main"
        pr_body: ":crown: *An automated PR*"
        pr_reviewer: "kingdonb"
        pr_draft: true
        github_token: ${{ secrets.GITHUB_TOKEN }}
```

You can use the [github-pull-request-action] workflow to automatically open a pull request against a destination branch. In this case, when `staging` is merged into the `main` branch, changes are deployed in production.

This way you can automatically push changes to a `staging` branch and require manual approval of any automatic image updates before they are applied on your production clusters.

[Image Update Guide]: /guides/image-update/
[Push updates to a different branch]: /guides/image-update/#push-updates-to-a-different-branch
[github-pull-request-action]: https://github.com/marketplace/actions/github-pull-request-action
