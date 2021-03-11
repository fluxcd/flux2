# GitHub Actions Manifest Generation

This is an example implementation of "build-time" manifest generation on GitHub Actions.

We first use third-party tools to prepare and update YAML manifests in a CI job. Then changes are committed to a deploy branch from CI and finally pushed, where `kustomize-controller` can apply them to a cluster.

A complementary feature in Flux v2 is [post-build variable substition](https://toolkit.fluxcd.io/components/kustomize/kustomization/#variable-substitution), but its use is limited by some constraints such as lacking support for [run-time access to Git repository metadata](https://github.com/fluxcd/flux2/discussions/1054) about the commit that is deployed.

We recommend to solve this from within CI, and many then asked, "how do I do that" – so this guide was written. To show an implementation of Manifest Generation that can be used with [any old app](https://github.com/kingdonb/any_old_app) on GitHub. (The linked example is a basic Rails app which has been used to develop this guide.)

From end-to-end, with a recommended architecture... 🆗 👍 🥇

## Primary Uses of Flux

Flux's primary use case for `kustomize-controller` is to apply YAML manifests from the latest `Revision` of an `Artifact`.

For a `GitRepository`, this is the latest commit in a branch or tag ref. The use cases in this tutorial for manifest generation assume that the CI is given write access to one or more Git repos **in order to affect changes to** what deploys on the cluster.

For so many CI providers, this can be complicated to set up. Secure authentication and authorization is hard. In GitHub Actions though, within a single repo, this is simply provided for without additional configuration, whenever Actions are enabled!

In the final example, since there is more than a single repository involved, an additional Personal Access Token will be used to access the alternate repository, in addition to the native GitHub Actions baked-in authentication described above.

### Security Consideration

Flux v2 can not be configured to call out to arbitrary binaries that a user might supply with an `InitContainer`, as it was possible to do in Flux v1.

In Flux v2 it is assumed if users want to run more than `Kustomize` with `envsubst`, that it will be done outside of Flux; "solve this from within CI" – showing how to do this is the motivation for writing this guide.

#### Motivation for this Guide

* To show a way to implement manifest generation, with Flux best practices in mind.
* To provide a "compatibility shim" for a common favorite Flux v1 feature that no longer works in Flux v2.

Flux's behavior of deploying `GIT_SHA` image tags as they are pushed [is being deprecated in Flux v2](https://github.com/fluxcd/flux2/discussions/802). This is an extremely common use case but what is deprecated is also very narrowly defined. It is addressing a performance bottleneck. The shim provided here works around this limitation in a different way than Flux v2's `image-automation-controller` solution.

**Providing timestamps in the image build tags** can resolve any need for temporary use of such a shim.

!!! warning "Migrate to use `ImagePolicy`, the `fluxcd.io/automated` behavior is deprecated"
    Qualified feature parity is provided through `ImagePolicies`. Those who use Flux's automation cannot automate `GIT_SHA` as the tag for automated workloads anymore in Flux v2.

For especially users who manage a lot of workloads that manage both Flux v1 and v2 in parallel, and who want to safely move just one piece at a time in order to limit the blast radius of upgrading, can follow these examples in order to reproduce the specific behavior of `fluxcd.io/automated` from CI, without anything else.

Readers should become familiar with Flux v2's [image scanning configuration](/guides/image-update/#configure-image-scanning) to avoid falling into the trap of [doing CI-ops](https://www.weave.works/blog/kubernetes-anti-patterns-let-s-do-gitops-not-ciops), where releases are driven by CI rather than coming from Git.

This compatibility shim does not perform image scanning in any way, it just approximates the behavior in a way that is mostly compatible for Dev environments. We drive the deployment of new images directly from CI. There is no reconciliation watching image repos for new images, (this is not a replacement for Flux automation.)

Recognizing also that not all Flux users are the ones publishing their own images, it is considered that making [this update](/guides/sortable-image-tags/) to CI build processes is not always practical for end-users.

#### Deprecated Features

So we address some deprecated features for users of Flux migrating their development operations from v1 to v2.

We recognize that for those migrating from Flux v1, many or most will have only followed guidance that we've recommended them in the past, so might be feeling betrayed now. Some of our guides and docs certainly have gone so far as to recommend using `GIT_SHA` as the whole image tag. (Even [Flux's own CI jobs](https://github.com/fluxcd/kustomize-controller/blob/main/.github/workflows/release.yml#L20) still write image tags like this.)

The Flux development team recognizes that this feature having changed may unfortunately cause pain for some of our users and for downstream technology adopters, who might rightly blame Flux for causing them this pain.

If your build process doesn't already meet Flux's new requirements for image automation, and you can not fix it now; this should not block upgrading to Flux v2. We can solve it adding only a few new CI build processes.

So, these two ideas are fundamentally separate, but this guide covers both ideas in tandem, with the same thread: how to do manifest generation, and how to use it to drive deploys from CI.

In a way that is GitOps-friendly and reminiscent of Flux v1, (even though it is also technically CI-ops!) we show how to reproduce the approximate behavior of a deprecated feature (`fluxcd.io/automated`), without imposing any new constraints or requirements on image tagging policy.

It is intended, finally, to show through this use case, three fundamental ideas for use in CI to accompany Flux automation:

1. Writing workflow that can commit changes back to the same branch of a working repository.
1. A workflow to commit generated content from one directory into a different branch in the repository.
1. Workflow to commit from any source directory into a target branch on some completely other repository.

Readers can interpret this document with adaptations for use with other CI providers, or Git source hosts.

#### The Choice of GitHub Actions

There are authentication concerns with every CI provider and they differ by Git provider.

Being that GitHub Actions is hosted on GitHub, this guide can be uniquely simple in some ways, as we can skip talking about authentication almost altogether. It is handled by the CI platform, we get write access to git.

(Other providers will need more attention for authn and authz configuration in order to ensure that connections between CI provider and Source provider's git host are all handled safely and securely.)

### Use Cases of Manifest Generation

There are several use cases presented. Shown below are a few different ways to add commits and push updates to Git, through GitHub Actions.

Our first example is doing a simple in-place string substitution using `sed -i`.

Then, a basic example of Docker Build and Push shows tagging and pushing images. While not exactly manifest generation, this is a key example shown for completeness. (It is however assumed that users will bring their own applications and Dockerfiles.)

The next example is using a third-party tool, `jsonnet`, to do what we refer to as "YAML rehydration." This example may grow to show more YAML hydrators as needed to account for differences and variability of these tools. Flux is not prescriptive about what YAML tools to use, as long as the pipeline ends with valid Kubernetes YAML. (Future examples planned include: cdk8s, mustache, ... requests?)

Finally it is shown how a Personal Access Token can be used to enable commits across repositories. These examples aim to provide as many variations as possible to show different options. As the actions used are provided by third parties, they are also furnished with their own separate documentation that will be linked within the text of each example below.

* [String Substitution with `sed -i`](#string-substitution-with-sed-i)
* [Docker Build and Tag with Version](#docker-build-and-tag-with-version)
* [Jsonnet for YAML Document Rehydration](#jsonnet-for-yaml-document-rehydration)
* [Commit Across Repositories Workflow](#commit-across-repositories-workflow)

The examples below assume no prior familiarity with GitHub Actions. Together these examples show several patterns for deployments with manifest generation.

This is the example we've been waiting for, the answer to `.flux.yaml` in Flux v2! 🎉🎁

#### String Substitution with `sed -i`

The entry point for these examples starts at `.github/workflows/` in a source repository. Add this directory if needed in your repositories. The first use case is to build a manifest with the commit hash of the latest commit, to deploy an image built from that commit. This example triggers from any commit on any branch.

In the example provided below, we will write a commit with the `GIT_SHA` of the latest tag on any branch. This is similar to how Flux v1 behaved when the `fluxcd.io/automated` annotation was enabled. Instead of expecting Flux to scan the image repository and find the latest tag that way, we will make that update to deployment manifest directly from CI.

We don't need to do any image scanning in order to find the latest image, we have the latest build, and we can know that easily since it's the same build that is running right now!

!!! warning "`GitRepository` source only targets one branch"
    While this example updates any branch (`branches: ['*']`) from CI, each `Kustomization` in Flux only deploys manifests from **one branch or tag** at a time. Other useful ideas will be employed in the later examples, which are all meant to be changed and adapted for more specific use cases.

This workflow example could go into an application's repository, wherever images are built from – each CI build triggers a commit writing the `GIT_SHA` to a configmap in the branch, and into a `deployment` YAML manifest to be deployed with a `Kustomization`, or by using `kubectl apply -f k8s.yml`. This example doesn't need or take advantage of Kustomize or `envsubst` in any way. More variations will follow.

Take and adapt these to any use case; it is meant to show that some problems are better solved with well-known tools like `sed`. Later examples will build up to using more sophisticated strategies.

```yaml
# ./.github/workflows/manifest-generate.yaml
name: Manifest Generate
on:
  push:
    branches:
    - '*'

jobs:
  run:
    name: Push Git Update
    runs-on: ubuntu-latest
    steps:
      - name: Prepare
        id: prep
        run: |
          VERSION=${GITHUB_SHA::8}
          echo ::set-output name=BUILD_DATE::$(date -u +'%Y-%m-%dT%H:%M:%SZ')
          echo ::set-output name=VERSION::${VERSION}

      - name: Checkout repo
        uses: actions/checkout@v2

      - name: Update manifests
        run: ./update-k8s.sh $GITHUB_SHA

      - name: Commit changes
        uses: EndBug/add-and-commit@v7
        with:
          add: '.'
          message: "[ci skip] deploy from ${{ steps.prep.outputs.VERSION }}"
          signoff: true
```

```yaml
# excerpted from above - workflow steps 1 and 2
- name: Prepare
- name: Checkout repo
```

We borrow parts of a [Prepare step](https://github.com/fluxcd/kustomize-controller/blob/5da1fc043db4a1dc9fd3cf824adc8841b56c2fcd/.github/workflows/release.yml#L17-L25) from Kustomize Controller's own release workflow.

In the `Prepare` step, even before the clone, GitHub Actions provides metadata about the commit. Then, `Checkout repo` performs a shallow clone for the build.

```bash
VERSION=${GITHUB_SHA::8}
echo ::set-output name=BUILD_DATE::$(date -u +'%Y-%m-%dT%H:%M:%SZ')
echo ::set-output name=VERSION::${VERSION}
```

!!! note "When migrating to Flux v2"
    Users will find that [some guidance has changed since Flux v1](https://github.com/fluxcd/flux2/discussions/802#discussioncomment-320189). Tagging images with a `GIT_SHA` is a common practice no longer supported with Flux's Image Automation. A newer alternative is using timestamp or build number for [Sortable image tags](/guides/sortable-image-tags), preferred by the `image-automation-controller`.

Whenever CI can provide images tagged with a build number or timestamp in the tag, [Image Automation](/guides/image-update/) can be deployed instead (without need for this example, or performing any other manifest generation or build-time commits back to Git.)

If `GIT_SHA` based tags are used, they are not updated by `image-automation-controller` because they are not string-sortable. The remainder of this guide assumes that readers are aware of this, but still want the capability for Flux to deploy a latest unique tag (for example, on a Dev cluster.)

If you just want to learn more techniques for manifest generation, don't worry, this guide is still for you too. (Keep reading!)

Using CI in this way obviates the [image pull rate limit issue](https://github.com/fluxcd/flux/issues/208) that is still observed when scaling up Flux v1 automation to work on larger numbers of deployments and images.

With that out of the way, next we call out to a shell script `update-k8s.sh` taking one argument, the Git SHA value from GitHub:

```yaml
# excerpted from above - run a shell script
- name: Update manifests
  run: ./update-k8s.sh $GITHUB_SHA
```

That script is below. It performs two in-place string substitutions using `sed`.

```bash
#!/bin/bash

# update-k8s.sh
set -feu	# Usage: $0 <GIT_SHA>	# Fails when GIT_SHA is not provided

GIT_SHA=${1:0:8}
sed -i "s|image: kingdonb/any-old-app:.*|image: kingdonb/any-old-app:$GIT_SHA|" k8s.yml
sed -i "s|GIT_SHA: .*|GIT_SHA: $GIT_SHA|" flux-config/configmap.yaml
```

`update-k8s.sh` receives `GITHUB_SHA` that the script trims down to 8 characters.

Then, `sed -i` runs twice, updating `k8s.yml` and `flux-config/configmap.yaml` which are also provided as examples here. The new SHA value is added twice, once in each file.

```yaml
# k8s.yml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: any-old-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: any-old-app
  template:
    metadata:
      labels:
        app: any-old-app
    spec:
      containers:
      - image: kingdonb/any-old-app:4f314627
        name: any-old-app
---
apiVersion: v1
kind: Service
metadata:
  name: any-old-app
spec:
  type: ClusterIP
  ports:
  - name: "any-old-app"
    port: 3000
  selector:
    app: any-old-app
```

If your entire application can be represented as a single `deployment` and `service`, then this might be all you need. The convention of including a `k8s.yml` file in one's application repository is borrowed from [Okteto's Getting Started Guides](https://github.com/okteto/go-getting-started/blob/master/k8s.yml), to help introduce some concepts to developers new to working with Kubernetes.

```yaml
# flux-config/configmap.yaml
apiVersion: v1
data:
  GIT_SHA: 4f314627
kind: ConfigMap
metadata:
  creationTimestamp: null
  name: any-old-app-version
  namespace: devl
```

A configmap is an ideal place to write a variable that is needed by any downstream `Kustomization`, for example to use with `envsubst`. This is written to a subdirectory to allow a `GitRepository` source to select it, without scanning the entire repository.

While this example works, it has some glaring limitations.  Keeping manifests in the same place as application source code invites confusion, as it's not always clear which version of the app is deployed from a commit, even after reading through the manifests.

If CI fails to build, your branch may contain a new copy of the source code while still pointing to an old release version. We may also find it points sometimes at an image which hasn't finished building yet, or even one that failed and will never be pushed.

This can leave the Kustomization in an error state which can be hard to detect, and may need to be manually remediated. This is not a strategy for Production. (Instead, use SemVer tags with `ImagePolicy`.)

#### Docker Build and Tag with Version

This is how to build an image and push a tag from the latest commit on the branch.

From the Actions marketplace, [Build and push Docker images](https://github.com/marketplace/actions/build-and-push-docker-images) provides the heavy lifting in this example. It has little to do with Flux, but we include it for completeness. (No prior knowledge of GitHub Actions is assumed on the part of the reader.)

!!! hint "`ImageRepository` can reflect both branches and tags"
    This example builds an image for any branch or tag ref and pushes it to Docker Hub. (Note the omission of `branches: ['*']` that was in the prior example.) GitHub Secrets `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` are used here to authenticate with Docker Hub from within GitHub Actions.

We again borrow a [Prepare step](https://github.com/fluxcd/kustomize-controller/blob/5da1fc043db4a1dc9fd3cf824adc8841b56c2fcd/.github/workflows/release.yml#L17-L25) from Kustomize Controller's own release workflow.

```yaml
# ./.github/workflows/docker-build.yaml
name: Docker Build, Push

on: push

jobs:
  docker:
    runs-on: ubuntu-latest
    steps:
      - name: Prepare
        id: prep
        run: |
          VERSION=${GITHUB_SHA::8}
          if [[ $GITHUB_REF == refs/tags/* ]]; then
            VERSION=${GITHUB_REF/refs\/tags\//}
          fi
          echo ::set-output name=BUILD_DATE::$(date -u +'%Y-%m-%dT%H:%M:%SZ')
          echo ::set-output name=VERSION::${VERSION}

      - name: Set up QEMU
        uses: docker/setup-qemu-action@v1

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v1

      - name: Login to DockerHub
        uses: docker/login-action@v1
        with:
          username: ${{ secrets.DOCKERHUB_USERNAME }}
          password: ${{ secrets.DOCKERHUB_TOKEN }}

      - name: Build and push
        id: docker_build
        uses: docker/build-push-action@v2
        with:
          push: true
          tags: kingdonb/any-old-app:${{ steps.prep.outputs.VERSION }}

      - name: Image digest
        run: echo ${{ steps.docker_build.outputs.digest }}
```

[Docker Login](https://github.com/marketplace/actions/docker-login) is used here to enable image push.

Any secrets can be used, and support for image registries is explained in the linked README. We can add a setting for `registry` if your app uses any private registry, rather than the implicit Docker Hub registry above.

```
# for example
with:
  registry: registry.cloud.okteto.net
```

Little above differs from the linked examples, provided from Docker via the GitHub Actions Marketplace.

In our example when a commit is pushed on a branch, `VERSION` is set from `GIT_SHA`, but if it is a tag that is pushed then the image tag is written and pushed instead.

* TODO: Add caching examples if needed to make sure that the merge commit does not reflect a different Image when it is rebuilt without any changes. That way, the image that has already been tested can be promoted, rather than building a new image assuming it hasn't changed, and pushing out an untested image.

* TODO: This type of error (an untested image is promoted into production) should always be prevented as early as possible through merge checks in staging and production, and using safe promotion strategies. (The todo is to document this strategy from end to end.)

```bash
# excerpted from above - set VERSION for tag refs
if [[ $GITHUB_REF == refs/tags/* ]]; then
  VERSION=${GITHUB_REF/refs\/tags\//}
fi
```

There is only one other substantial difference from the upstream example:

```yaml
name: Prepare
id: prep
run: |
  ...
  echo ::set-output name=VERSION::${VERSION}
...

with:
  push: true
  tags: kingdonb/any-old-app:${{ steps.prep.outputs.VERSION }}
```

The image tag `VERSION` comes from the branch or Git tag that triggered the build. Whether that version is a `GIT_SHA` or a semantic version tag, the same workflow can be used to build the image as shown here!

#### Jsonnet for YAML Document Rehydration

!!! note "`GitRepository` source only targets one branch"
    Since Flux uses one branch or tag per Kustomization, for CI to trigger an update to the cluster it must write into a `deploy` branch. Even when a new build can come from any branch (as is frequently the case for Dev environments) the YAML manifests to deploy are written to just one branch.

```yaml
name: Build jsonnet
on:
  push:
    tags: ['release/*']
    branches: ['release']

jobs:
  run:
    name: jsonnet push
    runs-on: ubuntu-latest
    steps:
      - name: Prepare
        id: prep
        run: |
          VERSION=${GITHUB_SHA::8}
          if [[ $GITHUB_REF == refs/tags/release/* ]]; then
            VERSION=${GITHUB_REF/refs\/tags\/release\//}
          fi
          echo ::set-output name=BUILD_DATE::$(date -u +'%Y-%m-%dT%H:%M:%SZ')
          echo ::set-output name=VERSION::${VERSION}

      - name: Checkout repo
        uses: actions/checkout@v2

      - id: jsonnet-render
        uses: alexdglover/jsonnet-render@v1
        with:
          file: manifests/example.jsonnet
          output_dir: output/
          params: dryrun=true;env=prod

      - name: Prepare target branch
        run: ./ci/rake.sh deploy

      - name: Commit changes
        uses: EndBug/add-and-commit@v7
        with:
          add: 'output/'
          branch: deploy
          message: "[ci skip] from ${{ steps.prep.outputs.VERSION }}"
          signoff: true
```

!!! note "The `EndBug/add-and-commit` action is used again"
    This time, with the help of `rake.sh`, our change is staged into a different target branch. This is the same `deploy` branch, regardless of where the build comes from; any push event can trigger this deploy.

```bash
#!/bin/bash

# ./ci/rake.sh
set -feux	# Usage: $0 <BRANCH>   # Fails when BRANCH is not provided
BRANCH=$1

# The output/ directory is listed in .gitignore, where jsonnet rendered output.
pushd output

# Fetch git branch 'deploy' and run `git checkout deploy`
/usr/bin/git -c protocol.version=2 fetch \
        --no-tags --prune --progress --no-recurse-submodules \
        --depth=1 origin $BRANCH
git checkout $BRANCH --

# Prepare the output to commit by itself in the deploy branch's root directory.
mv -f ./ ../	# Overwrite any existing files (no garbage collection here)
git diff

# All done (the commit will take place in the next action!)
popd
```

Tailor this workflow to your needs. We render from a file `manifests/example.jsonnet`, it can be anything. The output is a directory of K8s YAML files.

```yaml
- name: Commit changes
  uses: EndBug/add-and-commit@v7
  with:
    add: 'output/'
    branch: deploy
    message: "[ci skip] from ${{ steps.prep.outputs.VERSION }}"
    signoff: true
```

This is [Add & Commit](https://github.com/marketplace/actions/add-commit) with a `branch` option, to set the target branch. We've added a `signoff` option as well, to demonstrate another feature of this GitHub Action. There are many ways to use this workflow, (we will not be covering them exhaustively.)


This example writes the same `secretRef` into many `HelmReleases`, to provide for the cluster to be able to use the same `imagePullSecret` across several `Deployments` in a namespace. It is a common problem that `jsonnet` can solve quite handily, without repeating the `Secret` name over and over as a string.

```yaml
TODO: Add some jsonnet here to demonstrate how one does ^that
```

We could solve this in `Kustomization`'s `postBuild`, via `envsubst`, or handle it in CI as in this example. Un-templated manifests can now be read in directly by `Kustomize`, or reviewed by a person (before or after deployment) at any time.

This is the same actual YAML that is applied. (Now that's GitOps!)

Most use cases for deployments will not take just any push, preferring to target instead one specific branch. Merging in our application's Git repository to a `staging` branch can trigger staging a deployment to some Test environment, for example.

Manifest generation can be used to solve, broadly, very many problems. Even with many examples, this guide would never be totally exhaustive.

#### Commit Across Repositories Workflow

This is the final example in this guide. It shows how to replicate the behavior of Flux v1's image automation totally.

To replicate the best nearest approximation of Flux's "deploy latest image" feature of yesteryore, we use push events to do the job without invoking Flux v1's expensive and redundant image pull requirements to get access to image build metadata.

This is racy and doesn't guarantee the latest commit will be the one deployed, since it depends on the time that each commit is pushed and even how long the build takes to complete; fine for Dev environments, not for Production strategy.

This is done by leverage of CI to commit and push a subfolder from an application repository into a separate deploy branch of plain YAML manifests for `Kustomization` to apply, that can be pushed to any branch on any separate repository to which CI is granted write access.

```yaml
name: Update Fleet-Infra
on:
  push:
    branches:
    - 'main'

jobs:
  run:
    name: Push Update
    runs-on: ubuntu-latest
    steps:
      - name: Prepare
        id: prep
        run: |
          VERSION=${GITHUB_SHA::8}
          if [[ $GITHUB_REF == refs/tags/* ]]; then
            VERSION=${GITHUB_REF/refs\/tags\//}
          fi
          echo ::set-output name=BUILD_DATE::$(date -u +'%Y-%m-%dT%H:%M:%SZ')
          echo ::set-output name=VERSION::${VERSION}

      - name: Checkout repo
        uses: actions/checkout@v2

      - name: Update manifests
        run: ./update-k8s.sh $GITHUB_SHA

      - name: Push directory to another repository
        uses: cpina/github-action-push-to-another-repository@v1.2

        env:
          API_TOKEN_GITHUB: ${{ secrets.API_TOKEN_GITHUB }}
        with:
          source-directory: 'flux-config'
          destination-github-username: 'kingdonb'
          destination-repository-name: 'fleet-infra'
          target-branch: 'deploy'
          user-email: kingdon+bot@weave.works
          commit-message: "[ci skip] deploy from ${{ steps.prep.outputs.VERSION }}"
```

This is [Push directory to another repository](https://github.com/marketplace/actions/push-directory-to-another-repository). Flux v2 is made to work with more than one GitRepository.

If you use a mono-repo, consider adding a deploy branch to it! There is no need for branches in the same repo to always share a parent and intersect again at a merge point.

This is counter-productive for performance and will create bottlenecks for Flux, as large commits will take longer to clone, and therefore to reconcile; ignoring with `.sourceignore` or `spec.ignore` will unfortunately not help much. There are some limitations that can only be surpassed by changing the data structure.

The `flux-system` is in the `main` branch of `kingdonb/fleet-infra`, as is the default. We prepared in advance, an empty commit with no parent in the same repository, on the `deploy` branch, so that this checkout would begin with an empty workspace that `ci/rake.sh` could copy the `output/` of jsonnet into.

```bash
git checkout --orphan deploy
git reset --hard
git commit --allow-empty -m'initial empty commit'
git push origin deploy
```

The most generous characterization of this example as a whole that I can give, is that it is not regressive compared to the behavior of Flux v1's `fluxcd.io/automated`, that it would perform actually much better at scale, and at much lower cost in terms of control-plane and network overhead.

It is a compatibility shim, to bridge the gap for Flux v1 users. If possible, users are encouraged to migrate to using timestamps, build numbers, or semantic version tags, all supported by [Flux v2 image automation](/guides/image-update/) features that are still in alpha at the time of this writing.

Flux's new [Image Automation Controllers](/components/image/controller/) are the new solution for Production use!

### Adapting for Flux v2

In Flux v2, with `ImagePolicy`, these examples may be adjusted to order tags by their `BUILD_DATE`, by adding more string information to the tags. Besides a build timestamp, we can also add branch name.

Why not have it all: `${branch}-${sha}-${ts}` – this is the suggestion given in:

* [Example of a build process with timestamp tagging](/guides/sortable-image-tags/#example-of-a-build-process-with-timestamp-tagging).

Example formats and alternative strings to use for tagging are at:

* [Sortable image tags to use with automation](/guides/sortable-image-tags/#formats-and-alternatives).

We don't expect you to follow these examples to the letter. They present an evolution and are meant to show some of the breadth of options that are available, rather than as prescriptive guidance.

If you are on GitHub, and are struggling to get started using GitHub Actions, or maybe still waiting to make a move on your planned migration from Flux v1; we hope that these GitHub Actions examples can help Flux users to better bridge the gap between both versions.
