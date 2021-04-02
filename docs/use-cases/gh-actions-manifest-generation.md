# GitHub Actions Manifest Generation

This example implements "build-time" manifest generation on GitHub Actions.

Third-party tools are used to generate YAML manifests in a CI job. The updated YAML are committed and pushed to Git, where `kustomize-controller` finally applies them.

### Background

A complementary idea to manifest generation in Flux v2 is [post-build variable substition](https://toolkit.fluxcd.io/components/kustomize/kustomization/#variable-substitution). Unfortunately, for at least one popular use case, it is still too limited. Currently there is no support in `Kustomization` for [fetching Git repository metadata](https://github.com/fluxcd/flux2/discussions/1054) from the commit that is being deployed.

It is not possible to reference a commit's own Git SHA, or commit ID, as `kustomize build` runs from that commit. This is a common request from Flux users, especially those working with Mono-repos, as Flux v1 certainly had this capability.

Flux v1 allowed users to provide their own binaries to `kustomize build` by leveraging an `initContainer`, enabling users to `shell_exec` from within `.flux.yaml`, permitting arbitrary commands to be run by manifest generation, with practically no limitations on access to the commit and/or repository data.

This was able to be used productively [with `kustomize edit set image` in a generator](https://github.com/fluxcd/flux2/discussions/1002), for example, to reference tags built from the current commit, even without scanning any image repository (and perhaps before images have even finished building.)

While many productive uses were possible, this capability was a curse to supporting Flux, for site reliability as well as security reasons.

#### What's Not Possible Anymore

Clearly this arrangement offers too much freedom. Manifest generation runtimes always increases as a repository grows, which can be a source of reconciliation timeouts. From a security perspective also there can be no way to permit any of this to behave as described in a multi-tenancy environment.

Still, it is not an unreasonable ask to simply deploy the latest image, and also a reasonably common use case that Flux v1 could already fulfill handily (with no qualifiers!) Flux adopters leveraging this capability have fostered and grown many projects.

Flux is not prescriptive about how updates should happen. Sorting with Number or SemVer policies should not be considered to be exhaustive of the many possible approaches. The `image-automation-controller` is one possible implementation; it is also still in alpha releases as of the time of this writing, and can likely still be further improved, even in ways we maybe haven't thought of yet.

#### What Flux Does

Flux will deploy whatever is in the latest commit, on whatever branch you point it at. It is by design that Flux doesn't care how a commit is generated.

#### What Flux Don't

Since ["select latest by build time" image automation](https://github.com/fluxcd/flux2/discussions/802) is deprecated, and since [`.flux.yaml` is also deprecated](https://github.com/fluxcd/flux2/issues/543), some staple workflows are no longer possible without new accommodations from infrastructure.

Flux still provides image automation which remains separate from CI, but for `ImageAutomationPolicy` now there are some additional requirements on the image tagging plan for auto updating. While this is new, it still remains true that CI things are generally considered to be outside of the scope of FluxCD.

#### What Should We Do?

We first recommend users [adjust their tagging strategies](/guides/sortable-image-tags/), which is made clear elsewhere in the docs. This is usually a straightforward adjustment, and enables the use of [Image Update Policies](/guides/image-update/); however this may not be feasible or desired in some cases.

Parallel deployments of Flux v1 and v2 may be undertaken for change management reasons, and due to the order of implementation, changes to image tag strategies can't be undertaken before Flux v1 is finally taken down.

So as interim or alternative solutions may be needed, we recommended manifest generation to resolve this. Many asked, "how do I do *that* on **my CI**?"

As it turned out, this is neither straightforward nor obvious, so (even though it is not in Flux), this guide was conceived for Flux users.

(Possibly the first in a series,) supporting manifest generation strategies spanning different YAML tools and different CI providers with Flux:

## Use Manifest Generation

Introducing, Manifest Generation with Jsonnet, for [any old app](https://github.com/kingdonb/any_old_app) on GitHub!

Using a separate repo than your `fleet-infra` may be indicated at times through the examples. The `any_old_app` repo linked above is a skeleton app, which was used to develop this guide. (**`FIXME Editor: TODO`** You can also clone and inspect the branches there to find the code from these examples.)

OK, here with a recommended architecture... 🆗 👍 🥇

**SNIP**
Editor: Potentially cut everything between ^SNIP above and >SNIP below
except for perhaps the Security Consideration. (Covered well enough by new "Background" text above?)

### Primary Uses of Flux

Flux's primary use case for `kustomize-controller` is to apply YAML manifests from the latest `Revision` of an `Artifact`.

For a `GitRepository`, this is the latest commit in a branch or tag ref. The use cases in this tutorial for manifest generation assume that the **CI is given write access** to one or more Git repos **in order to affect changes to** what deploys on the cluster.

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

In case users need to run some workloads from both Flux v1 and v2 in parallel, for example a large migration moving one app at a time a-la [Strangler Fig Pattern](https://martinfowler.com/bliki/StranglerFigApplication.html), in order to maintain continuity of the specific behavior of `fluxcd.io/automated` this compatibility shim can be used.

Readers should become familiar with Flux v2's [image scanning configuration](/guides/image-update/#configure-image-scanning) to avoid [doing CI-ops](https://www.weave.works/blog/kubernetes-anti-patterns-let-s-do-gitops-not-ciops), where releases are driven thru CI rather than promoting tested artifacts from one environment to the next.

This compatibility shim does not perform image scanning in any way, it just approximates the behavior in a way that is mostly compatible for Dev environments. We drive the deployment of new images directly from CI. This is not a replacement for Flux automation. Flux does not recommend keeping this behavior in your CI, migrate to ImagePolicy based functionality of Flux v2 instead. Since this is a breaking change, a compatibility shim is being provided.

Recognizing also that not all Flux users are the ones publishing their own images, it is considered that making [this update](/guides/sortable-image-tags/) to CI build and image tagging processes is not always practical for end-users, who may even be completely dependant on inflexible upstreams.

#### Deprecated Features

So we address some deprecated features for users of Flux migrating their development operations from v1 to v2.

We recognize that for those migrating from Flux v1, many or most will have only followed guidance that we've recommended them in the past, so might be feeling betrayed now. Some of our guides and docs certainly have gone so far as to recommend using `GIT_SHA` as the whole image tag. (Even [Flux's own CI jobs](https://github.com/fluxcd/kustomize-controller/blob/main/.github/workflows/release.yml#L20) still write image tags like this.)

The Flux development team recognizes that this feature having changed may unfortunately cause pain for some of our users and for downstream technology adopters, who might rightly blame Flux for causing them this pain. We hope that making the update to sortable image tags will not be too painful.

If your build process doesn't already meet Flux's new requirements for image automation, and you can not fix it now; this pain should not block upgrading to Flux v2. We can still upgrade preserving same behavior, only adding a few new CI build processes. Then disable `fluxcd.io/automated` annotations, finally turn off Flux v1, completing the migration! We can address any image tagging process changes needed after the upgrade is already completed.

So, these two ideas are fundamentally separate, but this guide covers both ideas in tandem, with the same thread: how to do manifest generation, and how to use it to drive deploys from CI in a way that is GitOps-friendly and reminiscent of Flux v1.

Some may say this is technically CI-ops in GitOps clothing, and this is true too. CI-ops in this case makes it possible to reproduce closely the same behavior of the old deprecated feature (`fluxcd.io/automated`), without imposing any new constraints or additional requirements for image tagging policy.

Editor: Potentially cut everything between ^SNIP above and >SNIP below
**SNIP**

It is intended, finally, to show through this use case, three fundamental ideas for use in CI to accompany Flux automation:

1. Writing workflow that can commit changes back to the same branch of a working repository.
1. A workflow to commit generated content from one directory into a different branch in the repository.
1. Workflow to commit from any source directory into a target branch on some completely other repository.

Readers can interpret this document with adaptations for use with other CI providers, or Git source hosts.

### The Choice of GitHub Actions

There are authentication concerns with every CI provider and they differ by Git provider.

Being that GitHub Actions is hosted on GitHub, this guide can be uniquely simple in some ways, as we can mostly skip configuring authentication. The cross-cutting concern is handled by the CI platform; we write access to Git as we are already a Git user, (and one with write access.)

(Other providers will need more attention for authn and authz configuration in order to ensure that connections between CI provider and Source provider's git host are all handled safely and securely.)

## Use Cases of Manifest Generation

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

### String Substitution with `sed -i`

The entry point for these examples starts at `.github/workflows/` in a source repository. Add this directory if needed in your repositories. The first use case is to build a manifest with the commit hash of the latest commit, to deploy an image built from that commit. This example triggers from any commit on any branch.

In the example provided below, we will write a commit with the `GIT_SHA` of the latest tag on any branch. This is similar to how Flux v1 behaved when the `fluxcd.io/automated` annotation was enabled. Instead of expecting Flux to scan the image repository and find the latest tag that way, we will make that update to deployment manifest directly from CI.

We don't need to do any image scanning in order to find the latest image, we have the latest build, and we can know that easily since it's the same build that is running right now!

!!! warning "`GitRepository` source only targets one branch"
    While this example updates any branch (`branches: ['*']`) from CI, each `Kustomization` in Flux only deploys manifests from **one branch or tag** at a time. Other useful ideas will be employed in the later examples, which are all meant to be changed and adapted for more specific use cases.

This workflow example could go into an application's repository, wherever images are built from – each CI build triggers a commit writing the `GIT_SHA` to a configmap in the branch, and into a `deployment` YAML manifest to be deployed with a `Kustomization`, or by using `kubectl apply -f k8s.yml`. This example doesn't need or take advantage of Kustomize or `envsubst` in any way. More variations will follow.

Take and adapt these to any use case; it is meant to show that some problems are better solved with well-known tools like `sed`. Later examples will build up to using more sophisticated strategies.

```yaml
# ./.github/workflows/01-manifest-generate.yaml
name: Manifest Generation
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
          if [[ $GITHUB_REF == refs/tags/* ]]; then
            VERSION=${GITHUB_REF/refs\/tags\//}
          fi
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

### Docker Build and Tag with Version

This is how to build an image and push a tag from the latest commit on the branch.

From the Actions marketplace, [Build and push Docker images](https://github.com/marketplace/actions/build-and-push-docker-images) provides the heavy lifting in this example. It has little to do with Flux, but we include it for completeness. (No prior knowledge of GitHub Actions is assumed on the part of the reader.)

!!! hint "`ImageRepository` can reflect both branches and tags"
    This example builds an image for any branch or tag ref and pushes it to Docker Hub. (Note the omission of `branches: ['*']` that was in the prior example.) GitHub Secrets `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` are used here to authenticate with Docker Hub from within GitHub Actions.

We again borrow a [Prepare step](https://github.com/fluxcd/kustomize-controller/blob/5da1fc043db4a1dc9fd3cf824adc8841b56c2fcd/.github/workflows/release.yml#L17-L25) from Kustomize Controller's own release workflow.

```yaml
# ./.github/workflows/02-docker-build.yaml
name: Docker Build, Push

on:
  push:
    branches:
      - '*'
    tags-ignore:
      - 'release/*'

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

We should note, this example does not implement caching. This means that even when subsequent commits represent a no-op, totally new images will be built instead of reusing existing layers. This is wasteful, but for us it is out of scope. [Docker Buildx Caching in GitHub Actions](https://evilmartians.com/chronicles/build-images-on-github-actions-with-docker-layer-caching#the-cache-dance-off) is a great topic and a totally other guide.

* TODO: Document strategies to prevent untested commits from being promoted into staging or production environments, through the use of release staging and validation with merge checks, for safe promotion. (The todo is to document such a strategy from end to end – which is again probably out of scope for this guide.)

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

### Jsonnet for YAML Document Rehydration

As mentioned before, Flux only monitors one branch or tag per Kustomization, so it is advisable to protect the release branch with eg. branch policies, as it directly represents the production deployment.

The CI user for this example should be allowed to push directly to the `deploy` branch; it also represents the production environment here and must be protected from normal users in a similar fashion. Only CI (and/or the cluster admin user) should be allowed to write to the `deploy` branch.

!!! note "`GitRepository` source only targets one branch"
    Since Flux uses one branch per Kustomization, to trigger an update we must write to a `deploy` branch or tag. Even when new app images can come from any branch (eg. for Dev environments where any latest commit is to be deployed) the YAML manifests to deploy will be sourced from just one branch.

#### Jsonnet Render Action

Below we build from any tag with the `release/*` format, or any latest commit pushed to the `release` branch. (Users, we expect, may choose either `on.push` strategy, probably not using both `tags` and `branches` as the example below, to avoid confusion.)

The outputted YAML manifests, on successful completion of the Jsonnet render step, are staged on the `deploy` branch, then committed and pushed.

The latest commit on the `deploy` branch is reconciled into the cluster by a `Kustomization`.

```yaml
# ./.github/workflows/03-release-manifests.yaml
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

      - name: Setup kubecfg CLI
        uses: kingdonb/kubecfg/action@main

      - name: kubecfg show
        run: kubecfg show manifests/example.jsonnet > output/production.yaml

      - name: Prepare target branch
        run: ./ci/rake.sh deploy

      - name: Commit changes
        uses: EndBug/add-and-commit@v7
        with:
          add: 'production.yaml'
          branch: deploy
          message: "[ci skip] from ${{ steps.prep.outputs.VERSION }}"
          signoff: true
```

While it is straightforward to see which commit is the latest pushed on a branch, it is not easy to know which tag was created chronologically last.

It is possible for a `1.3.4` tag to be pushed after `1.3.5`, for example, if tagging is implemented as a manual process. As CI builds execute in the order that they are received, an earlier numbered release could win.

This is not the same Flux v1's `semver` tag filter strategy, nor Flux v2's SemVer ImageAutomationPolicy. It is strictly deploying manifests from the latest commit pushed.

For this reason, we also trigger jobs on push to `branch: release` – now that we are effectively interjecting CI into the build process, we need a way to force the deploy artifacts to a certain release.

Users of this strategy may invoke the `release` branch in case of rollback, or if there is a need to without creating a new version number. Next we point a Kustomization at the new `deploy` branch.

We add three new steps in this example:

```yaml
# excerpted from above - workflow steps 3, 4, and 5
- name: Setup kubecfg CLI
  uses: kingdonb/kubecfg/action@main

- name: kubecfg show
  run: kubecfg show manifests/example.jsonnet > output/production.yaml

- name: Prepare target branch
  run: ./ci/rake.sh deploy

```

As for setting up the kubecfg CLI in GitHub Actions, since no `kubecfg` action appears on the GitHub Actions marketplace, this was an exercise in building a basic GitHub Action for the author.

Mostly borrowed from the [Setup Flux CLI step](https://github.com/fluxcd/flux2/tree/main/action) in Flux's own repo, [kingdonb/kubecfg/action](https://github.com/kingdonb/kubecfg/tree/main/action) is an extremely simple action that you can read and adapt for whatever binary release tool besides `kubecfg` you want to use in your manifest generation pipeline.

!!! warning "The `with: version` option is ignored, remember any person can publish a GitHub Action, and not all actions are trustworthy. Fork and pin, as well as audit upstream actions whenever security is important.

While the remaining examples will be written to depend on `kubecfg`, some use cases may prefer to use pure Jsonnet only as it is sandboxed and therefore safer. We plan to use the `kubecfg` capability to take input from other sources, like variables and references, but also network-driven imports and functions.

```yaml
#  from above - substitute these steps in 03-release-manifests.yaml,
#  between "Checkout repo" and "Commit changes" to use Jsonnet instead of kubecfg
- id: jsonnet-render
  uses: alexdglover/jsonnet-render@v1
  with:
    file: manifests/example.jsonnet
    output_file: output/production.yaml
    params: dryrun=true;env=prod

- name: Prepare target branch
  run: ./ci/rake.sh deploy
```

The `jsonnet-render` step is borrowed from another source, again find it on [GitHub Actions Marketplace](https://github.com/marketplace/actions/jsonnet-render) for more information.

!!! note "The `EndBug/add-and-commit` action is used again"
    This time, with the help of `rake.sh`, our change is staged into a different target branch. This is the same `deploy` branch, regardless of which branch or tag the build comes from; any configured push event can trigger this workflow to trigger an update to the deploy branch.

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

This is [Add & Commit](https://github.com/marketplace/actions/add-commit) with a `branch` option, to set the target branch. We've added a `signoff` option as well here, to demonstrate another feature of this GitHub Action. There are many ways to use this workflow step. The link provides more information.

The examples that follow can be copied and pasted into `manifests/example.jsonnet`, then committed to the `release` branch and pushed to GitHub in order to execute them.

Read onward to see some basic as well as more advanced uses of `kubecfg`.

#### Looping

This example writes the same `secretRef` into many `HelmReleases`, to provide for the cluster to be able to use the same `imagePullSecret` across several `Deployments` in a namespace. It is a common problem that `jsonnet` can solve quite handily, without repeating the `Secret` name over and over as a string.

```yaml
TODO: Add a jsonnet example here that demonstrates simple string interpolation and a looping construct
```

(Say something about Jsonnet example above.)

#### `ConfigMap` with `envsubst`

The next example "enforces," or copies, a `configMap` with a SHA value in it from one namespace into many namespaces, so that Kustomizations in each namespace can maintain the same config data in their reconciliations and stay DRY, without building configurations that reach across namespace boundaries.

When you write a `Kustomization` to apply this, be sure you don't set `targetNamespace` or it will override any namespace settings in the Jsonnet output

ConfigMap values are not treated as secret data, so there is no confusing encryption to contend with, making for a good first example. This simple example shows how to centralize the reference to these plain-text tag or version values. (Later, we will repeat this process with a SOPS encrypted secret.)

```yaml
TODO: Add example that builds a configmap and clones it into several namespaces,
      making multiple references into the ConfigMap from neighboring HelmReleases
```

(Say something about the configmap example above.)

#### Handling `Secret`s

Because a `secret` is not safe to store in Git unencrypted, Flux recommends using SOPS to encrypt it.

SOPS will produce a [different data key](https://github.com/mozilla/sops/issues/315) for each fresh invocation of `sops -e`, producing different cipher data even for the same input data. This is true even when the secret content has not changed. This means, unfortunately, it is not practical for a Manifest Generation routine to implement secret transparency without granting the capability to read secrets to the CI infrastructure.

SOPS stores the metadata required to decrypt each secret in the metadata of the secret, which must be stored unencrypted to allow encrypted secrets to be read by the private key owners.

Secret transparency means that it should be possible for an observer to know when a stored secret has been updated or rotated. Transparency can be achieved in SOPS by running using `sops [encrypted.yaml]` as an editor, which decrypts and re-encrypts the secret, only changing the cipher text when secret data also changes.

Depending on your access model, this suggestion could be either a complete non-starter, or a helpful add-on.

As an example, Secrets could be read from GitHub Secrets during a CI job, then written encrypted into a secret that is pushed to the deploy branch. This implementation provides a basic solution for simple centralized secrets rotation. But as this would go way beyond simple manifest generation, we consider this beyond the scope of the tutorial, and it is mentioned only as an example of a more complex usage scenario for users to consider.

#### Replicate `Secrets` Across Namespaces

When the data of a `secret` is stored in the git repository, it can be encrypted to store and transmit safely. SOPS in Kustomize supports encryption of only `(stringData|data)` fields, not secret `metadata` including `namespace`. This means that secrets within the same repo can be copied freely and decrypted somewhere else, just as long as the `Kustomization` still has access to the SOPS private key.

Because of these properties though, copying a SOPS-encrypted secret from one namespace to another within one single Flux tenant is as easy as cloning the YAML manifest and updating the `namespace` field. Compared to SealedSecrets controller, which does not permit this type of copying; SOPS, on the other hand, does not currently prevent this without some attention being paid to RBAC.

Remember to protect your secrets with RBAC! This is not optional, when handling secrets as in this example.

#### Protecting `Secrets` from Unauthorized Access

The logical boundary of a secret is any cluster or tenant where the private key is available for decrypting.

This means that any SOPS secret, once encrypted, can be copied anywhere or used as a base for other Kustomizations in the cluster, so long as the Kustomization itself has access to the decryption keys.

It is important to understand that the `sops-gpg` key that is generated in the Flux SOPS guide can be used by any `Kustomization` in the `flux-system` namespace.

It cannot be over-emphasized; if users want secrets to remain secret, the `flux-system` namespace (and indeed the entire cluster itself) must be hardened and protected, managed by qualified cluster admins. It is recommended that changes which could access encrypted secrets are tightly controlled as much as deemed appropriate.

#### More Advanced Secrets Usage

The use of KMS as opposed to in-cluster GPG keys with SOPS is left as an exercise for the reader. The basics of KMS with various cloud providers is covered in more depth by the [Mozilla SOPS](/guides/mozilla-sops/#using-various-cloud-providers) guide.

Another scenario we considered, but rejected for these examples, requires to decrypt and then re-encrypt SOPS secrets, for use with the `secretGenerator` feature of Kustomize. This workflow is not supported here for reasons already explained.

Flux suggests maintaining the only active copy of the decryption key for a cluster inside of that cluster (though there may be a provision for backups, or some alternate keys permitted to decrypt.) This arrangement makes such use cases significantly more complicated to explain, beyond the scope of this guide.

For those uses though, additional Workflow Actions are provided:

The [Decrypt SOPS Secrets](https://github.com/marketplace/actions/decrypt-sops-secrets) action may be useful and it is mentioned here, (but no example uses are provided.)

The [Sops Binary Installer](https://github.com/marketplace/actions/sops-binary-installer) action enables more advanced use cases, like encrypting or re-encrypting secrets.

#### Jsonnet Recap

While much of this type of manipulation could be handled in `Kustomization`'s `postBuild`, via `envsubst`, some configurations are more complicated this way. They can be better handled in CI, where access to additional tools can be provided.

By writing YAML manifests into a git commit, this Jsonnet usage pattern enables inspection of manifests as they are read directly by `Kustomize` before being applied, for review at any time; before or after deployment.

When the YAML that is applied in the cluster comes directly from Git commits, that's GitOps!

### Commit Across Repositories Workflow

Flux will not deploy from pushes on just any branch; GitRepository sources target just one specific branch. Merging to a `staging` branch, for example, can be used to trigger a deployment to a Staging environment.

Manifest generation can be used to solve, broadly, very many problems, such that even with many examples, this guide would never be totally exhaustive.

This is the final example in this guide.

Here we show 🥁 ... how to replicate the original behavior of Flux v1's image automation! 🤯 🎉

To replicate the nearest approximation of Flux's "deploy latest image" feature of yesteryore, we use push events to do the job, as we hinted was possible in an earlier example. Without Flux v1's redundant and expensive image pull behavior, used to get access to image build information required to order image tags for deployment.

This is racy and doesn't always guarantee the latest commit will be the one that is deployed, since this behavior depends on the time that each commit is pushed, and even precisely how long the build takes to complete; the difference is fine for Dev environments, but this is not a strategy for Production use cases.

App CI can commit and push a subfolder full of YAML manifests into a separate deploy branch for `Kustomization` to apply, which can be any branch on any separate repository to which the CI is granted write access.

While there are some issues, this is actually perfect for some deployments, eg. in a staging environment!

```yaml
# ./.github/workflows/04-update-fleet-infra.yaml
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

This is [Push directory to another repository](https://github.com/marketplace/actions/push-directory-to-another-repository). This is especially useful because Flux v2 is made to work with more than one GitRepository.

If you must use a mono-repo, consider adding a deploy branch to it! There is no need for branches in the same repo to always share a parent and intersect again at a merge point.

A mono-repo can be counter-productive for performance and will create bottlenecks for Flux, as large commits will take longer to clone, and therefore to reconcile. Ignoring with `.sourceignore` or `spec.ignore` will unfortunately not help much with this. Some limitations can only be overcome by changing the data structure.

The `flux-system` is in the `main` branch of `kingdonb/fleet-infra`, as is the default. We prepared in advance, an empty commit with no parent in the same repository, on the `deploy` branch, so that this checkout would begin with an empty workspace that `ci/rake.sh` could copy the `output/` of Jsonnet into.

```bash
git checkout --orphan deploy
git reset --hard
git commit --allow-empty -m'initial empty commit'
git push origin deploy
```

This is not technically regressive when compared to the behavior of Flux v1's `fluxcd.io/automated`, actually avoiding image pull depending on push instead to write the latest git tag, externally and functionally identical to how Flux v1 did automation. Little else is good that we can say about it.

It is a compatibility shim, to bridge the gap for Flux v1 users. If possible, users are encouraged to migrate to using timestamps, build numbers, or semver tags, that are all supported by some [Flux v2 image automation](/guides/image-update/) features that are still in alpha at the time of this writing.

Flux's new [Image Automation Controllers](/components/image/controller/) are the new solution for Production use!

### Adapting for Flux v2

In Flux v2, with `ImagePolicy`, these examples may be adjusted to order tags by their `BUILD_DATE`, by adding more string information to the tags. Besides a build timestamp, we can also add branch name.

Why not have it all: `${branch}-${sha}-${ts}` – this is the suggestion given in:

* [Example of a build process with timestamp tagging](/guides/sortable-image-tags/#example-of-a-build-process-with-timestamp-tagging).

Example formats and alternative strings to use for tagging are at:

* [Sortable image tags to use with automation](/guides/sortable-image-tags/#formats-and-alternatives).

We don't expect you to follow these examples to the letter. They present an evolution and are meant to show some of the breadth of options that are available, rather than as prescriptive guidance.

If you are on GitHub, and are struggling to get started using GitHub Actions, or maybe still waiting to make a move on your planned migration from Flux v1; we hope that these GitHub Actions examples can help Flux users to better bridge the gap between both versions.
