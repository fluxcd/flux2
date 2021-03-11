# GitHub Actions Manifest Generation

This example implements "build-time" manifest generation on GitHub Actions.

Third-party tools are used to generate YAML manifests in a CI job. The updated YAML are committed and pushed to Git, where `kustomize-controller` finally applies them.

### Background

There are many use cases for manifest generation tools, but Flux v2 no longer permits embedding arbitrary binaries with the Flux machinery to run at apply time.

Flux (kustomize-controller) will apply whatever revision of the manifests are at the latest commit, on any branch it is pointed at. By design, Flux doesn't care for any details of how a commit is generated.

Since ["select latest by build time" image automation][flux2/discussions/802] is deprecated, and since [`.flux.yaml` is also deprecated][flux2/issues/543], some staple workflows are no longer possible without new accommodations from infrastructure.

#### What Should We Do?

We first recommend users [adjust their tagging strategies][Sortable image tags], which is made clear elsewhere in the docs. This is usually a straightforward adjustment, and enables the use of [Image Update Policies][image update guide]; however this may not be feasible or desired in some cases.

## Use Manifest Generation

Introducing, Manifest Generation with Jsonnet, for [any old app] on GitHub!

If you have followed the [Flux bootstrap guide] and only have one `fleet-infra` repository, it is recommended to create a separate repository that represents your application for this use case guide, or clone the repository linked above in order to review these code examples which have already been implemented there.

### Primary Uses of Flux

Flux's primary use case for `kustomize-controller` is to apply YAML manifests from the latest `Revision` of an `Artifact`.

### Security Consideration

Flux v2 can not be configured to call out to arbitrary binaries that a user might supply with an `InitContainer`, as it was possible to do in Flux v1.

#### Motivation for this Guide

In Flux v2 it is assumed if users want to run more than `Kustomize` with `envsubst`, that it will be done outside of Flux; the goal of this guide is to show several common use cases of this pattern in secure ways.

#### Demonstrated Concepts

It is intended, finally, to show through this use case, three fundamental ideas for use in CI to accompany Flux automation:

1. Writing workflow that can commit changes back to the same branch of a working repository.
1. A workflow to commit generated content from one directory into a different branch in the repository.
1. Workflow to commit from any source directory into a target branch on a different repository.

Readers can interpret this document with adaptations for use with other CI providers, or Git source hosts, or manifest generators.

Jsonnet is demonstrated with examples presented in sufficient depth that, hopefully, Flux users who are not already familiar with manifest generation or Jsonnet can pick up `kubecfg` and start using it to solve novel and interesting configuration problems.

### The Choice of GitHub Actions

There are authentication concerns to address with every CI provider and they also differ by Git provider.

Given that GitHub Actions are hosted on GitHub, this guide can be streamlined in some ways. We can almost completely skip configuring authentication. The cross-cutting concern is handled by the CI platform, except in our fourth and final example, the *Commit Across Repositories Workflow*.

From a GitHub Action, as we must have been authenticated to write to a branch, Workflows also can transitively gain write access to the repo safely.

Mixing and matching from other providers like Bitbucket Cloud, Jenkins, or GitLab will need more attention to these details for auth configurations. GitHub Actions is a platform that is designed to be secure by default.

## Manifest Generation Examples

There are several use cases presented.

* [String Substitution with sed -i]
* [Docker Build and Tag with Version]
* [Jsonnet for YAML Document Rehydration]
* [Commit Across Repositories Workflow]

In case these examples are too heavy, this short link guide can help you navigate the four main examples. Finally, the code examples we've all been waiting for, the answer to complex `.flux.yaml` configs in Flux v2! 🎉🎁

### String Substitution with `sed -i`

The entry point for these examples begins at `.github/workflows/` in any GitHub source repository where your YAML manifests are stored.

!!! warning "`GitRepository` source only targets one branch"
    While this first example operates on any branch (`branches: ['*']`), each `Kustomization` in Flux only deploys manifests from **one branch or tag** at a time. Understanding this is key for managing large Flux deployments and clusters with multiple `Kustomizations` and/or crossing several environments.

First add this directory if needed in your repositories. Find the example below in context, and read on to understand how it works: [01-manifest-generate.yaml].

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

In the `Prepare` step, even before the clone, GitHub Actions provides metadata about the commit. Then, `Checkout repo` performs a shallow clone for the build.

```bash
# excerpt from above - set two outputs named "VERSION" and "BUILD_DATE"
VERSION=${GITHUB_SHA::8}
echo ::set-output name=BUILD_DATE::$(date -u +'%Y-%m-%dT%H:%M:%SZ')
echo ::set-output name=VERSION::${VERSION}
```

!!! note "When migrating to Flux v2"
    Users will find that [some guidance has changed since Flux v1]. Tagging images with a `GIT_SHA` was a common practice that is no longer supported by Flux's Image Automation. A newer alternative is adding timestamp or build number in [Sortable image tags], preferred by the `image-automation-controller`.

Next we call out to a shell script `update-k8s.sh` taking one argument, the Git SHA value from GitHub:

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

The convention of including a `k8s.yml` file in one's application repository is borrowed from [Okteto's Getting Started Guides], as a simplified example.

The `k8s.yml` file in the application root is not meant to be applied by Flux, but might be a handy template to keep fresh as a developer reference nonetheless.

The file below, `configmap.yaml`, is placed in a directory `flux-config/` which will be synchronized to the cluster by a `Kustomization` that we will add in the following step.

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

These are the two files that are re-written in the `sed -i` example above.

A configmap is an ideal place to write a variable that is needed by any downstream `Kustomization`, for example to use with `envsubst`.

```yaml
---
apiVersion: kustomize.toolkit.fluxcd.io/v1beta1
kind: Kustomization
metadata:
  name: any-old-app-devl
spec:
  interval: 15m0s
  path: ./
  prune: true
  sourceRef:
    kind: GitRepository
    name: any-old-app-prod
  targetNamespace: prod
  validation: client
  postBuild:
    substituteFrom:
      - kind: ConfigMap
        name: any-old-app-version
---
apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: GitRepository
metadata:
  name: any-old-app-prod
  namespace: prod
spec:
  interval: 20m0s
  ref:
    branch: deploy
  secretRef:
    name: flux-secret
  url: ssh://git@github.com/kingdonb/csh-flux
```

Now, any downstream `Deployment` in the `Kustomization` can write a `PodSpec` like this one, to reference the image from the latest commit referenced by the `ConfigMap`:

```yaml
# flux/ some-example-deployment.yaml
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
      - image: kingdonb/any-old-app:${GIT_SHA}
        name: any-old-app
```

Deployment specifications will vary, so adapting this example is left as exercise for the reader. Write it together with a kustomization.yaml, or just add this to a subdirectory anywhere within your Flux Kustomization path.

### Docker Build and Tag with Version

Now for another staple workflow: building and pushing an OCI image tag from a Dockerfile in any branch or tag.

From the Actions marketplace, [Build and push Docker images] provides the heavy lifting in this example. Flux has nothing to do with building images, but we include this still — as some images will need to be built for our use in these examples.

!!! hint "`ImageRepository` can reflect both branches and tags"
    This example builds an image for any branch or tag ref and pushes it to Docker Hub. (Note the omission of `branches: ['*']` that was in the prior example.) GitHub Secrets `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` are used here to authenticate with Docker Hub from within GitHub Actions.

We again borrow a [Prepare step] from Kustomize Controller's own release workflow. Find the example below in context, [02-docker-build.yaml], or copy it from below.

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

The [Docker Login Action] is used here to enable an authenticated image push.

Any secrets from GitHub Secrets can be used as shown, and support for image registries is explained in the linked README. Add a setting for `registry` if your app uses any private registry, rather than the implicit Docker Hub registry above.

```
# for example
with:
  registry: registry.cloud.okteto.net
```

The image tag `VERSION` comes from the branch or Git tag that triggered the build. Whether that version is a `GIT_SHA` or a Semantic Version, (or anything in between!) the same workflow can be used to build an OCI image as shown here.

### Jsonnet for YAML Document Rehydration

As mentioned before, Flux only monitors one branch or tag per Kustomization.

In the earlier examples, no fixed branch target was specified. Whatever branch triggered the workflow, received the generated YAMLs in the next commit.

If you created your deployment manifests in any branch, the `deploy` branch or otherwise, it is necessary to add another `Kustomization` and `GitRepository` source to apply manifests from that branch and path in the cluster.

In application repositories, it is common to maintain an environment branch, a release branch, or both. Some additional Flux objects may be needed for each new environment target with its own branch. Jsonnet can be used for more easily managing heavyweight repetitive boilerplate configuration such as this.

It is recommended to follow these examples as they are written for better understanding, then later change and adapt them for your own release practices and environments.

!!! note "`GitRepository` source only targets one branch"
    Since Flux uses one branch per Kustomization, to trigger an update we must write to a `deploy` branch or tag. Even when new app images can come from any branch (eg. for Dev environments where any latest commit is to be deployed) the YAML manifests to deploy will be sourced from just one branch.

It is advisable to protect repository main and release branches with eg. branch policies and review requirements, as through automation, these branches can directly represent the production environment.

The CI user for this example should be allowed to push directly to the `deploy` branch that Kustomize deploys from; this branch also represents the environment so must be protected in a similar fashion to `release`.

Only authorized people (and build robots) should be allowed to make writes to a `deploy` branch.

#### Jsonnet Render Action

In this example, the outputted YAML manifests, (on successful completion of the Jsonnet render step,) are staged on the `deploy` branch, then committed and pushed.

The latest commit on the `deploy` branch is reconciled into the cluster by another `Kustomization` that is omitted here, as it is assumed that users who have read this far already added this in the previous examples.

You may find the example below in context, [03-release-manifests.yaml], or simply copy it from below.

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

While the remaining examples will be written to depend on `kubecfg`, some use cases may prefer to use pure Jsonnet only as it is sandboxed and therefore safer. We plan to use the `kubecfg` capability to take input from other sources, like variables and references, but also network-driven imports and functions.

```yaml
# from above - substitute these steps in 03-release-manifests.yaml,
# between "Checkout repo" and "Commit changes" to use plain Jsonnet instead of kubecfg
- id: jsonnet-render
  uses: alexdglover/jsonnet-render@v1
  with:
    file: manifests/example.jsonnet
    output_file: output/production.yaml
    params: dryrun=true;env=prod

- name: Prepare target branch
  run: ./ci/rake.sh deploy
```

The `jsonnet-render` step is borrowed from another source, again find it on [GitHub Actions Marketplace][actions/jsonnet-render] for more information. For Tanka users, there is also [letsbuilders/tanka-action] which describes itself as heavily inspired by `jsonnet-render`.

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
mv -f ./production.yaml ../	# Overwrite any existing files (no garbage collection here)
git diff

# All done (the commit will take place in the next action!)
popd
```

Give this file `chmod +x` before adding and committing; tailor this workflow to your needs. We render from a file `manifests/example.jsonnet`, it can be anything. The output is a single K8s YAML file, `production.yaml`.

```yaml
- name: Commit changes
  uses: EndBug/add-and-commit@v7
  with:
    add: 'production.yaml'
    branch: deploy
    message: "[ci skip] from ${{ steps.prep.outputs.VERSION }}"
    signoff: true
```

This is [Add & Commit] with a `branch` option, to set the target branch. We've added a `signoff` option as well here, to demonstrate another feature of this GitHub Action. There are many ways to use this workflow step. The link provides more information.

The examples that follow can be copied and pasted into `manifests/example.jsonnet`, then committed to the `release` branch and pushed to GitHub in order to execute them.

Pushing this to your repository will fail at first, unless and until a `deploy` branch is created.

Run `git checkout --orphan deploy` to create a new empty HEAD in your repo.

Run `git reset` and `git stash` to dismiss any files that were staged, then run `git commit --allow-empty` to create an initial empty commit on the branch.

Now you can copy and run these commands to create an empty branch:

```
#- Add a basic kustomization.yaml file
cat <<EOF > kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- production.yaml
EOF

#- Add a gitignore so output directory is created
cat <<EOF > .gitignore
/output/**
!/output/.keep
EOF

#- Add the .keep file
touch output/.keep

#- Push these files into an empty branch, and go back to main branch
git add output/.keep .gitignore kustomization.yaml
git commit -m'seed deploy kustomization.yaml for prod'
git push -u origin deploy
git checkout main
```

On the main branch as well, create a `.gitignore` and `/output/.keep` for that branch too. We need to make sure that the `output/` directory is present for writing whenever the Jsonnet workflow begins. The `rake.sh` script sweeps files from the output into the root directory of the `deploy` branch.

Now run `git checkout -b release; git push -u origin release` to trigger this action and see it working! 🤞🤞

You may need to be sure that GitHub Actions are enabled on your repository before this will work.

Read onward to see some basic as well as more advanced uses of `kubecfg`.

##### Jsonnet `ConfigMap` with `envsubst`

The next example "enforces," or copies, a value from a `configMap` from one namespace into many namespaces. This is done so that Kustomizations for each namespace can maintain similar config data in their reconciliations while staying DRY, with some configurations that reach across namespace boundaries.

With Jsonnet, Kustomize, and Kustomization Controller's `postBuild` which uses `envsubst`, there are usually a handful of different ways to accomplish the same task.

This example demonstrates a feature called `ext_vars` or [External Variables] in the Jsonnet `stdlib`.

These examples assume (since no such protection has been presented) that nothing prevents `production.yaml` from writing resources throughout multiple namespaces. This may be difficult or impossible to achieve depending on your environment.

In order to permit a Kustomization to write into different namespaces, some RBAC configuration may be required.

When you write a `Kustomization` to apply this, be sure you are aware of whether or not you have set `targetNamespace` on the Flux `Kustomization` as it may override any namespace settings in the Jsonnet output. You may note similar configuration in the `kustomization.yaml` we wrote into the deploy branch as described above, in the step for **Jsonnet Render Action**.

##### External Variable Substitution

```javascript
# Any Old App Jsonnet example 0.10.1 - manifests/example.jsonnet

local kube = import 'https://github.com/bitnami-labs/kube-libsonnet/raw/73bf12745b86718083df402e89c6c903edd327d2/kube.libsonnet';

local example = import 'example.libsonnet';

{
  version_configmap: kube.ConfigMap('any-old-app-version') {
    metadata+: {
      namespace: 'prod',
    },
    data+: {
      VERSION: std.extVar('VERSION'),
    },
  },
  flux_kustomization: example.kustomization('any-old-app-prod') {
    metadata+: {
      namespace: 'flux-system',
    },
    spec+: {
      path: './flux-config/',
      postBuild+: {
        substituteFrom+: [
          {
            kind: 'ConfigMap',
            name: 'any-old-app-version',
          },
        ],
      },
    },
  },
  flux_gitrepository: example.gitrepository('any-old-app-prod') {
    metadata+: {
      namespace: 'flux-system',
    },
    spec+: {
      url: 'https://github.com/kingdonb/any_old_app',
    },
  },
}
```

The above jsonnet declaration `example.jsonnet` will not complete without its neighbor `example.libsonnet` (which can be found [linked here][example 10.1 library].) This part of the example contains some boilerplate detail not meant to be copied, like the name `any-old-app-prod` and the string `'sops-gpg'` in `decryption.secretRef` which should be changed to match your environment).

If you visited the linked `example.libsonnet` you may have noticed definitions for `kustomization` and `gitrepository` that are frankly pretty specific for a library function. They include details you wouldn't expect to find in a vendor library, like a default git repository URL, and a default hardcoded ref to the name of our Source gitrepository.

This is **our library file**, so it can have our own implementation-specific details in it if we want to include them. Now, the power of Jsonnet is visible; we get to decide which configuration needs to be exposed in our main `example.jsonnet` file, and which parameters are defaults provided by the library, that can be treated like boilerplate and re-defined however we want.

```json
data+: {
  VERSION: std.extVar('VERSION'),
},
```

This is `std.extVar` from `ext_vars` mentioned earlier. Arrange for the version to be passed in through the GitHub Actions workflow:

```yaml
# adapted from above - 03-release-manifests.yaml
- name: kubecfg show
  run: kubecfg show -V VERSION=${{ steps.prep.outputs.VERSION }} manifests/example.jsonnet > output/production.yaml
```

The neighbor `example.libsonnet` file contains some boring (but necessary) boilerplate, so that `kubecfg` can fulfill this jsonnet, to generate and commit the full Kustomize-ready YAML into a `deploy` branch as specified in the workflow. (The `Kustomization` for this example is provided [from my fleet-infra repo here][any-old-app-deploy-kustomization.yaml]. Personalize and adapt this for use with your own application or manifest generations.)

The values provided are for example only and should be personalized, or restructured/rewritten completely to suit your preferred template values and instances. For more idiomatic examples written recently by actual Jsonnet pros, the [Tanka - Using Jsonnet tutorial] is great, and so is [Tanka - Parameterizing] which I'll call out specifically for the `_config::` object example that is decidedly more elegant than my version of parameter passing.

If you've not previously used Jsonnet before, then you might be wondering about that code example you just read and that's OK! If you **have** previously used Jsonnet and already know what idiomatic Jsonnet looks like, you might be wondering too... you can probably tell I (author, Kingdon) practically haven't ever written a lick of Jsonnet before today.

These examples are going to get progressively more advanced as I learn Jsonnet while I go. At this point I already think it's pretty cool and I barely know how to use it, but I am starting to understand what type of problems people are using it to solve.

ConfigMap values are not treated as secret data, so there is no encryption to contend with; this makes for what seems like a good first example. Jsonnet enthusiasts, please forgive my newness. I am sure that my interpretation of how to write Jsonnet is most likely not optimal or idiomatic.

Above we showed how to pass in a string from our build pipeline, and use it to write back generated Jsonnet manifests into a commit.

##### Make Two Environments

Here's a second example, defining two environments in separate namespaces, instead of just one:

```javascript
# Any Old App Jsonnet example 0.10.2-alpha1 - manifests/example.jsonnet
# Replicate a section of config and change nothing else about it

// ...

{
  version_configmap: kube.ConfigMap('any-old-app-version') {
    metadata+: {
      namespace: 'flux-system',
    },
    data+: {
      VERSION: std.extVar('VERSION'),
    },
  },
  test_flux_kustomization: example.kustomization('any-old-app-test') {
    metadata+: {
      namespace: 'flux-system',
    },
    spec+: {
      path: './flux-config/',
      postBuild+: {
        substituteFrom+: [
          {
            kind: 'ConfigMap',
            name: 'any-old-app-version',
          },
        ],
      },
      targetNamespace: 'test-tenant',
    },
  },
  prod_flux_kustomization: example.kustomization('any-old-app-prod') {
    metadata+: {
      namespace: 'flux-system',
    },
    spec+: {
      path: './flux-config/',
      postBuild+: {
        substituteFrom+: [
          {
            kind: 'ConfigMap',
            name: 'any-old-app-version',
          },
        ],
      },
      targetNamespace: 'prod-tenant',
    },
  },
  flux_gitrepository: example.gitrepository('any-old-app-prod') {
    metadata+: {
      namespace: 'flux-system',
    },
    spec+: {
      url: 'https://github.com/kingdonb/any_old_app',
    },
  },
}
```

Wait, what? I thought Jsonnet was supposed to be DRY. (Be gentle, refactoring is a slow and deliberate process.) We simply copied the original one environment into two environments, test and prod, which differ only in name.

In the next example, we will subtly change one of them to be configured differently from the other.

##### Change Something and Refactor

Note the string 'flux-system' only occurs once now, having been factored into a variable `config_ns`. These are some basic abstractions in Jsonnet that we can use to start to DRY up our source manifests.

Again, practically nothing changes functionally, this still does exactly the same thing. With another refactoring, we can express this manifest more concisely, thanks to a new library function we can invent, named `example.any_old_app`.

```javascript
# Any Old App Jsonnet example 0.10.2-alpha4 - manifests/example.jsonnet
# Make something different between test and prod
{
  version_configmap: kube.ConfigMap('any-old-app-version') {
    metadata+: {
      namespace: config_ns,
    },
    data+: {
      VERSION: std.extVar('VERSION'),
    },
  },
  test_flux_kustomization: example.any_old_app('test') {
    spec+: {
      prune: true,
    },
  },
  prod_flux_kustomization: example.any_old_app('prod') {
    spec+: {
      prune: false,
    },
  },
  flux_gitrepository: example.gitrepository('any-old-app-prod') {
    metadata+: {
      namespace: config_ns,
    },
    spec+: {
      url: 'https://github.com/kingdonb/any_old_app',
    },
  },
}
```

Two things have changed to make this refactoring of the config differ from the first version. Hopefully you'll notice it's a lot shorter.

Redundant strings have been collapsed into a variable, and more boilerplate has been moved into the library.

This refactored state is perhaps the most obvious to review, and most intentionally clear about its final intent to the reader. Hopefully you noticed the original environments were identical (or if your eyes glossed over because of the wall of values, you've at least taken my word for it.)

But now, these two differ. We're creating two configurations for `any_old_app`, named `test` and `prod`. One of them has `prune` enabled, the test environment, and `prod` is set more conservatively to prevent accidental deletions, with a setting of `prune: false`.

Since the two environments should each differ only by the boolean setting of `spec.prune`, we can now pack up and hide away the remainder of the config in with the rest of the boilerplate.

Hiding the undifferentiated boilerplate in a library makes it easier to detect and observe this difference in a quick visual review.

Here's the new library function's definition:

```javascript
any_old_app(environment):: self.kustomization('any-old-app-' + environment) {
  metadata+: {
    namespace: 'flux-system',
  },
  spec+: {
    path: './flux-config/',
    postBuild+: {
      substituteFrom+: [
        {
          kind: 'ConfigMap',
          name: 'any-old-app-version',
        },
      ],
    },
    targetNamespace: environment + '-tenant',
  },
},
```

This excerpt is taken from [the 10.2 release version][example 10.2 library excerpt] of `example.libsonnet`, where you can also read the specific definition of `kustomization` that is invoked with the expression `self.kustomization('any-old-app-' + environment)`.

The evolution of this jsonnet snippet has gone from *unnecessarily verbose* to **perfect redundant clarity**. I say redundant, but I'm actually fine with this exactly the way it is. I think, if nothing further changes, we already have met the best way to express this particular manifest with Jsonnet.

But given that config will undoubtedly have to change as the differing requirements of our development teams and their environments grow, this perfect clarity unfortunately can't last forever in its current form. It will have to scale.

Notice that strings and repeated invocations of `any_old_app` are written with parallel structure and form, but there's nothing explicit linking them.

The object-oriented programmer in me can't help but ask now, "what happens when we need another copy of the environment, this time slightly more different than those two, and ... how about two more after that, (and actually, can we really get by with only ten environments?)" — I am inclined towards thinking of this repeated structure as a sign that an object cries out, waiting to be recognized and named, and defined, (and refactored and defined again.)

##### List Comprehension

So get ready for _obfuscated nightmare mode_, (which is the name we thoughtfully reserved for the [best and final version][example 10.2 jsonnet] of the example,) shown below.

```javascript
# Any Old App Jsonnet example 0.10.2 - manifests/example.jsonnet

// This is a simple manifest generation example to demo some simple tasks that
// can be automated through Flux, with Flux configs rehydrated through Jsonnet.

// This example uses kube.libsonnet from Bitnami.  There are other
// Kubernetes libraries available, or write your own!
local kube = import 'https://github.com/bitnami-labs/kube-libsonnet/raw/73bf12745b86718083df402e89c6c903edd327d2/kube.libsonnet';

// The declaration below adds configuration to a more verbose base, defined in
// more detail at the neighbor libsonnet file here:
local example = import 'example.libsonnet';
local kubecfg = import 'kubecfg.libsonnet';
local kustomize = import 'kustomize.libsonnet';

local config_ns = 'flux-system';

local flux_config = [
  kube.ConfigMap('any-old-app-version') {
    data+: {
      VERSION: std.extVar('VERSION'),
    },
  },
  example.gitrepository('any-old-app-prod') {
    spec+: {
      url: 'https://github.com/kingdonb/any_old_app',
    },
  },
] + kubecfg.parseYaml(importstr 'examples/configMap.yaml');

local kustomization = kustomize.applyList([
  kustomize.namespace(config_ns),
]);

local kustomization_output = std.map(kustomization, flux_config);

{ flux_config: kustomization_output } + {

  local items = ['test', 'prod'],

  joined: {
    [ns + '_flux_kustomization']: {
      data: example.any_old_app(ns) {
        spec+: {
          prune: if ns == 'prod' then false else true,
        },
      },
    }
    for ns in items

    // Credit:
    // https://groups.google.com/g/jsonnet/c/ky6sjYj4UZ0/m/d4lZxWbhAAAJ
    // thanks Dave for showing how to do something like this in Jsonnet
  },
}
```

This is the sixth revision of this example, (some have been omitted from the story, but they are [in Git history][examples 0.10.2-all].) I think it's really perfect now. If you're a programmer, I think, this version is perhaps much clearer. That's why I called it _obfuscated nightmare mode_, right? (I'm a programmer, I swear.)

I'm trying to learn Jsonnet as fast as I can, I hope you're still with me and if not, don't worry. Where did all of this programming come from? (And what's a list comprehension?) It really doesn't matter.

The heavy lifting libraries for this example are from [anguslees/kustomize-libsonnet], which implements some basic primitives of Kustomize in Jsonnet. YAML parser is provided by [bitnami/kubecfg][kubecfg yaml parser], and the Jsonnet implementations of Kubernetes primitives by [bitnami-labs/kube-libsonnet].

It is a matter of taste whether you consider from above the first, second, or third example to be better stylistically. It is a matter of taste and circumstances, to put a finer point on it. They each have strengths and weaknesses, depending mostly on whatever changes we will have to make to them next.

We can compare these three versions to elucidate the intent of the programmatically most expressive version which followed the other two. If you're new at this, you may try to explain how these three examples are similar, and also how they differ. Follow the explanation below for added clarity.

If you haven't studied Jsonnet, this last version may daunt you with its complexity. The fact is YAML is a document store and Jsonnet is a programming language. This complexity is exactly what we came here for, we want our configuration language to be more powerful! Bring on more complex Jsonnet examples!

#### Breaking It Down

We define a configmap and a gitrepository (in Jsonnet), then put it together with another configmap (from plain YAML). That's called `flux_config`.

```javascript
local kustomization = kustomize.applyList([
  kustomize.namespace(config_ns),
]);

local kustomization_output = std.map(kustomization, flux_config);
```

This little diddy (above) has the same effect as a Kustomization based on the following instruction to `kustomize build`, (except it's all jsonnet):

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: ${config_ns}
```

... setting the namespace for all objects in `flux_config` to the value of `config_ns`.

Next, we join it together with a list comprehension (at least I think that's what this is called):

```javascript
local items = ['test', 'prod'],

joined: {
  [ns + '_flux_kustomization']: {
    data: example.any_old_app(ns) {
      spec+: {
        prune: if ns == 'prod' then false else true,
      },
    },
  }
  for ns in items
},
```

Two `any_old_app` templates are invoked programmatically, with different properties and names, and in target namespaces that are based on the environment names. They go on the end of the document list, and Jsonnet renders them alongside of the others, in various namespaces.

This is the same technique as in `01-manifest-generate.yaml`, only this time with Jsonnet and `kubecfg` instead of `sed`, (and what a difference it makes!)

This is the foundation for some real release machinery for your applications, this is not just a bunch of shell scripts. Whenever any commit hits the `release` branch, or when any tag in the form `release/*` is pushed, the repo is configured to push generated manifest changes to a `deploy` branch.

This behavior is self-contained within the example `any_old_app` repository in these examples.

We can use GitHub Actions and Jsonnet to populate parameters through ConfigMap values or with `extVars`, and at the same time, apply `Kustomization` and `GitRepository` as new sync infrastructure for the `deploy` branch with dependencies on those ConfigMaps. The `Kustomization` refers to the configmap and makes the `VERSION` or `GIT_SHA` variable available as a `postBuild` substitution, with values pulled from that same configmap we just applied.

Later, we can repeat this process with a SOPS encrypted secret.

The process is not very different, though some of the boilerplate is longer, we've already learned to pack away boilerplate. Copying and renaming encrypted secrets within the same cluster is possible wherever cluster operators are permitted to both decrypt and encrypt them with the decryption provider.

A credential at `spec.decryption.secretRef` holds the key for decryption. Without additional configuration secrets can usually be copied freely around the cluster, as it is possible to decrypt them freely anywhere the decryption keys are made available.

##### Copy `ConfigMap`s

Assume that each namespace will be separately configured as a tenant by itself somehow later, and that each tenant performs its own git reconciliation within the tenant namespace. That config is out of scope for this example. We are only interested in briefly demonstrating some Jsonnet use cases here.

The app version to install is maintained in a `ConfigMap` in each namespace based on our own decision logic. This can be implemented as a human operator who goes in and updates this variable's value before release time.

This Jsonnet creates from a list of namespaces, and injects a `ConfigMap` into each namespace, [another example.jsonnet][example 10.3 jsonnet].

```javascript
# Any Old App Jsonnet example 0.10.3 - manifests/example.jsonnet

local release_config = kube.ConfigMap('any-old-app-version');
local namespace_list = ['prod', 'stg', 'qa', 'uat', 'dev'];

local release_version = '0.10.3';
local latest_candidate = '0.10.3-alpha1';

{
  [ns + '_tenant']: {
    [ns + '_namespace']: {
      namespace: kube.Namespace(ns),
    },
    [ns + '_configmap']: {
      version_data: release_config {
        metadata+: {
          namespace: ns,
        },
        data+: {
          VERSION: if ns == 'prod' || ns == 'stg' then release_version else latest_candidate,
        },
      },
    },
  }
  for ns in namespace_list
}
```

In this example, we have set up an additional 3 namespaces and assumed that a Flux Kustomization is provided some other way. The deploy configuration of all 5 environments is maintained here, in a single deploy config.

Imagine that two policies should exist for promoting releases into environments. The environments for `dev`elopment, `U`ser `A`cceptance `T`esting (`uat`), and `Q`uality `A`ssurance (`qa`) can all be primed with the latest release candidate build at any given time.

This is perhaps an excessive amount of formality for an open source or cloud-native project, though readers working in regulated environments may recognize this familiar pattern.

This example will possibly fail to apply with the recommended validations enabled, failing with errors that you can review by running `flux get kustomization` in your flux namespace, like these:

```
validation failed: namespace/dev created (server dry run)
namespace/prod created (server dry run)
...
Error from server (NotFound): error when creating "14f54b89-2456-4c15-862e-34670dfcda79.yaml": namespaces "dev" not found
Error from server (NotFound): error when creating "14f54b89-2456-4c15-862e-34670dfcda79.yaml": namespaces "prod" not found
```

As you can perhaps see, the problem is that objects created within the namespace are not valid before creating the namespace. You can disable the validation temporarily by adding a setting `validation: none` to your Flux Kustomization to get past this error.

In the deployment configuration above, both `prod` and staging (`stg`) are kept in sync with the latest release (not pre-release).

Left as an exercise to the reader, we can also ask next: is it possible to supplement this configuration with a Flagger canary so that updates to the production config are able to be manually verified in the staging environment before they are promoted into Production?

(Hint: Look at the [Manual Gating] feature of Flagger.)

##### Copy `Secret`s

This example writes the same `secretRef` into many `HelmReleases`, to provide for the cluster to be able to use the same `imagePullSecret` across several `Deployments` in a namespace. It is a common problem that `jsonnet` can solve quite handily, without repeating the `Secret` name over and over as a string.

Because we have decided to create tenants for each namespace, now is a good time to mention [flux create tenant].

We can take the output of `flux create tenant prod --with-namespace prod --export` and use it to create `manifests/examples/tenant.yaml`. Perhaps in a full implementation, we would create a tenant library function and call it many times to create our tenants.

For this example, you may discard the `Namespace` and/or `ClusterRoleBinding` as they are not needed. Here, we actually just need a ServiceAccount to patch.

```yaml
---
# manifests/examples/tenant.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    toolkit.fluxcd.io/tenant: prod
  name: prod
  namespace: prod
```

A namespace will be created by our Jsonnet example code instead (or you may comment this line out in the jsonnet below, if you are already working within a tenant.) The ClusterRoleBinding, or a more restrictive RoleBinding, is important for a functioning Flux tenant installation, but it is not needed for this example.

(For more information on multi-tenancy, read the [Flux 2 Multi-Tenancy Guide].)

We make an image pull secret with some docker registry credentials, for the purpose of completing the example. This is just for an example, it can be any secret that you want to replicate through several namespaces in the cluster with Jsonnet.

```
kubectl create secret docker-registry prod-docker-pull-secret \
  --namespace=prod \
  --docker-username=youruser --docker-password=secretpassword \
  --docker-email=required@example.org --dry-run=client -oyaml \
    > manifests/examples/sops-image-pull-secret.yaml
sops -e -i manifests/examples/sops-image-pull-secret.yaml
```

If you are not familiar with SOPS encryption, you should complete the [Mozilla SOPS] Flux guide before replicating this example, and internalize all of the concepts explained there.

You will need to have configured your cluster's Kustomization with a decryption provider and decryption keys to enable the inclusion of encrypted secrets in your config repos. It is not safe to write unencrypted secrets into your git repository, and this should be avoided at all costs even if your repository is kept private.

This final Jsonnet example is presented in context as a working reference in the `any_old_app` repository, once again as [example.jsonnet][example 10.4 jsonnet].

```yaml
# Any Old App Jsonnet example 0.10.4 - manifests/example.jsonnet

local kubecfg = import 'kubecfg.libsonnet';
local kustomize = import 'kustomize.libsonnet';

local tenant =
  kubecfg.parseYaml(importstr 'examples/tenant.yaml');
local pull_secrets =
  kubecfg.parseYaml(importstr 'examples/sops-image-pull-secret.yaml');

local prod_ns = 'prod';
local staging_ns = 'stg';
local image_pull_secret = 'prod-docker-pull-secret';

// Set the Image Pull Secret on each ServiceAccount
local updateConfig(o) = (
  if o.kind == 'ServiceAccount' then o {
    imagePullSecrets: [{ name: image_pull_secret }],
  } else o
);

// Create a namespace, and add to it Namespace and Secret
local prod_tenant = [
  kube.Namespace(prod_ns),
] + pull_secrets + tenant;

// Prod kustomization - apply the updateConfig
local prod_kustomization = kustomize.applyList([
  updateConfig,
]);

// Stg kustomization - apply the updateConfig and "stg" ns
local staging_kustomization = kustomize.applyList([
  updateConfig,
  kustomize.namespace(staging_ns),
]);

// Include both kustomizations in the Jsonnet object
{
  prod: std.map(prod_kustomization, prod_tenant),
  stg: std.map(staging_kustomization, prod_tenant),
}
```

The `kubecfg.parseYaml` instruction returns a list of Jsonnet objects. Our own Jsonnet closely mirrors the [example provided][anguslees example jsonnet] by anguslees, with a few differences.

The power of kubecfg is well illustrated through this example inspired by [anguslees/kustomize-libsonnet]. We parse several YAML files and make several minor updates to them. It really doesn't matter if one document involved is a secret, or that its data is encrypted by SOPS.

If you are coming to Mozilla SOPS support in Flux v2, having used the SealedSecrets controller before when it was recommended in Flux v1, then you are probably surprised that this works. SOPS does not encrypt secret metadata when used with Flux's Kustomize Controller integration, which makes examples like this one possible.

The ServiceAccount is a part of Flux's `tenant` configuration, and a fundamental concept of Kubernetes RBAC. If this concept is still new to you, read more in the [Kubernetes docs on Using Service Accounts].

The other fundamental concept to understand is a Namespace.

Secrets are namespaced objects, and ordinary users with tenant privileges cannot reach outside of their namespace. If tenants should manage a Flux Kustomization within their own namespace boundaries, then a `sops-gpg` secret must be present in the Namespace with the Kustomization. Cross-namespace secret refs are not supported.

However, any principal with access to read a `sops-gpg` secret can decrypt any data that are encrypted for it.

Each ServiceAccount can list one or more `imagePullSecrets`, and any pod that binds the ServiceAccount will automatically include any pull secrets provided there. By adding the imagePullSecret to a ServiceAccount, we can streamline including it everywhere that it is needed.

We can apply a list of transformations with `kustomize.applyList` that provides a list of functions for Jsonnet to apply to each list of Jsonnet objects; in our case we use the `updateConfig` function to patch each ServiceAccount with the ImagePullSecret that we want it to use.

Finally, for staging, we additionally apply `kustomize.namespace` to update all resources to use the `stg` namespace instead of the `prod` namespace. The secret can be copied anywhere we want within the reach of our Flux Kustomization, and since our Flux Kustomization still has `cluster-admin` and local access to the decryption key, there is no obstacle to copying secrets.

#### Handling `Secret`s

Because a `secret` is not safe to store in Git unencrypted, Flux recommends using SOPS to encrypt it.

SOPS will produce a [different data key][sops/issues/315] for each fresh invocation of `sops -e`, producing different cipher data even for the same input data. This is true even when the secret content has not changed. This means, unfortunately, it is not practical for a Manifest Generation routine to implement secret transparency without also granting the capability to read secrets to the CI infrastructure.

SOPS stores the metadata required to decrypt each secret in the metadata of the secret, which must be stored unencrypted to allow encrypted secrets to be read by the private key owners.

Secret transparency means that it should be possible for an observer to know when a stored secret has been updated or rotated. Transparency can be achieved in SOPS by running `sops` as an editor, using `sops [encrypted.yaml]`, which decrypts for editing and re-encrypts the secret upon closing the editor, thereby only changing the cipher text when secret data also changes.

Depending on your access model, this suggestion could be either a complete non-starter, or a helpful add-on.

As an example, Secrets could be read from GitHub Secrets during a CI job, then written encrypted into a secret that is pushed to the deploy branch. This implementation provides a basic solution for simple centralized secrets rotation. But as this would go way beyond simple manifest generation, we consider this beyond the scope of the tutorial, and it is mentioned only as an example of a more complex usage scenario for users to consider.

#### Replicate `Secrets` Across Namespaces

When the data of a `secret` is stored in the Git repository, it can be encrypted to store and transmit safely. SOPS in Kustomize supports encryption of only `(stringData|data)` fields, not secret `metadata` including `namespace`. This means that secrets within the same repo can be copied freely and decrypted somewhere else, just as long as the `Kustomization` still has access to the SOPS private key.

Because of these properties though, copying a SOPS-encrypted secret from one namespace to another within one single Flux tenant is as easy as cloning the YAML manifest and updating the `namespace` field. Compared to SealedSecrets controller, which does not permit this type of copying; SOPS, on the other hand, does not currently prevent this without some attention being paid to RBAC.

Remember to protect your secrets with RBAC! This is not optional, when handling secrets as in this example.

#### Protecting `Secrets` from Unauthorized Access

The logical boundary of a secret is any cluster or tenant where the private key is available for decrypting.

This means that any SOPS secret, once encrypted, can be copied anywhere or used as a base for other Kustomizations in the cluster, so long as the Kustomization itself has access to the decryption keys.

It is important to understand that the `sops-gpg` key that is generated in the Flux SOPS guide can be used by any `Kustomization` in the `flux-system` namespace.

It cannot be over-emphasized; if users want secrets to remain secret, the `flux-system` namespace (and indeed the entire cluster itself) must be hardened and protected, managed by qualified cluster admins. It is recommended that changes which could access encrypted secrets are tightly controlled as much as deemed appropriate.

#### More Advanced Secrets Usage

The use of KMS as opposed to in-cluster GPG keys with SOPS is left as an exercise for the reader. The basics of KMS with various cloud providers is covered in more depth by the [Mozilla SOPS][using various cloud providers] guide.

Another scenario we considered, but rejected for these examples, requires to decrypt and then re-encrypt SOPS secrets, for use with the `secretGenerator` feature of Kustomize. This workflow is not supported here for reasons already explained.

Flux suggests maintaining the only active copy of the decryption key for a cluster inside of that cluster (though there may be a provision for backups, or some alternate keys permitted to decrypt.) This arrangement makes such use cases significantly more complicated to explain, beyond the scope of this guide.

For those uses though, additional Workflow Actions are provided:

The [Decrypt SOPS Secrets] action may be useful and it is mentioned here, (but no example uses are provided.)

The [Sops Binary Installer] action enables more advanced use cases, like encrypting or re-encrypting secrets.

#### Jsonnet Recap

While much of this type of manipulation could be handled in `Kustomization`'s `postBuild`, via `envsubst`, some configurations are more complicated this way. They can be better handled in CI, where access to additional tools can be provided.

By writing YAML manifests into a Git commit, the same manifests as `Kustomize` directly applies, they can be saved for posterity. Or projected out into a new pull request where they can be reviewed before application, or with the proper safe-guards in place they can be applied immediately through a more direct-driven automation.

With generated YAML that Flux applies in the cluster directly from Git commits, **fui-yoh** - that's GitOps!

### Commit Across Repositories Workflow

Flux will not deploy from pushes on just any branch; GitRepository sources target just one specific branch. Merging to a `staging` branch, for example, can be used to trigger a deployment to a Staging environment.

Manifest generation can be used to solve, broadly, very many problems, such that even with many examples, this guide would never be totally exhaustive.

This is the final example in this guide.

Here we show 🥁 ... how to replicate the original behavior of Flux v1's image automation! 🤯 🎉

You can put this workflow in your application repo, and target it toward your `fleet-infra` repo.

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

This is [Push directory to another repository]. This is especially useful because Flux v2 is made to work with more than one GitRepository.

If you must use a mono-repo, consider adding a deploy branch to it! There is no need for branches in the same repo to always share a parent and intersect again at a merge point.

A mono-repo can be counter-productive for performance and will create bottlenecks for Flux, as large commits will take longer to clone, and therefore to reconcile. Ignoring with `.sourceignore` or `spec.ignore` will unfortunately not help much with this. Some limitations can only be overcome by changing the data structure.

The `flux-system` is in the `main` branch of `kingdonb/fleet-infra`, as is the default. We prepared in advance, an empty commit with no parent in the same repository, on the `deploy` branch, so that this checkout would begin with an empty workspace that `ci/rake.sh` could copy the `output/` of Jsonnet into.

```bash
git checkout --orphan deploy
git reset --hard
git commit --allow-empty -m'initial empty commit'
git push origin deploy
```

This is not technically regressive when compared to the behavior of Flux v1's `fluxcd.io/automated`, actually avoiding image pull depending on push instead to write the latest Git tag, externally and functionally identical to how Flux v1 did automation. Little else is good that we can say about it.

It is a compatibility shim, to bridge the gap for Flux v1 users. If possible, users are encouraged to migrate to using timestamps, build numbers, or semver tags, that are all supported by some [Flux v2 image automation] features that are still in alpha at the time of this writing.

Flux's new [Image Automation Controllers] are the new solution for Production use!

### Adapting for Flux v2

In Flux v2, with `ImagePolicy`, these examples may be adjusted to order tags by their `BUILD_DATE`, by adding more string information to the tags. Besides a build timestamp, we can also add branch name.

Why not have it all: `${branch}-${sha}-${ts}` – this is the suggestion given in:

* [Example of a build process with timestamp tagging].

Example formats and alternative strings to use for tagging are at:

* [Sortable image tags to use with automation].

We don't expect you to follow these examples to the letter. They present an evolution and are meant to show some of the breadth of options that are available, rather than as prescriptive guidance.

If you are on GitHub, and are struggling to get started using GitHub Actions, or maybe still waiting to make a move on your planned migration from Flux v1; we hope that these GitHub Actions examples can help Flux users to better bridge the gap between both versions.

[flux2/discussions/802]: https://github.com/fluxcd/flux2/discussions/802
[flux2/issues/543]: https://github.com/fluxcd/flux2/issues/543
[image update guide]: /guides/image-update/
[any old app]: https://github.com/kingdonb/any_old_app
[Flux bootstrap guide]: /get-started/
[String Substitution with sed -i]: #string-substitution-with-sed-i
[Docker Build and Tag with Version]: #docker-build-and-tag-with-version
[Jsonnet for YAML Document Rehydration]: #jsonnet-for-yaml-document-rehydration
[Commit Across Repositories Workflow]: #commit-across-repositories-workflow
[01-manifest-generate.yaml]: https://github.com/kingdonb/any_old_app/blob/main/.github/workflows/01-manifest-generate.yaml
[some guidance has changed since Flux v1]: https://github.com/fluxcd/flux2/discussions/802#discussioncomment-320189
[Sortable image tags]: /guides/sortable-image-tags/
[Okteto's Getting Started Guides]: https://github.com/okteto/go-getting-started/blob/master/k8s.yml
[Build and push Docker images]: https://github.com/marketplace/actions/build-and-push-docker-images
[Prepare step]: https://github.com/fluxcd/kustomize-controller/blob/5da1fc043db4a1dc9fd3cf824adc8841b56c2fcd/.github/workflows/release.yml#L17-L25
[02-docker-build.yaml]: https://github.com/kingdonb/any_old_app/blob/main/.github/workflows/02-docker-build.yaml
[Docker Login Action]: https://github.com/marketplace/actions/docker-login
[03-release-manifests.yaml]: https://github.com/kingdonb/any_old_app/blob/main/.github/workflows/03-release-manifests.yaml
[actions/jsonnet-render]: https://github.com/marketplace/actions/jsonnet-render
[letsbuilders/tanka-action]: https://github.com/letsbuilders/tanka-action
[Add & Commit]: https://github.com/marketplace/actions/add-commit
[External Variables]: https://jsonnet.org/ref/stdlib.html#ext_vars
[example 10.1 library]: https://github.com/kingdonb/any_old_app/blob/release/0.10.1/manifests/example.libsonnet
[any-old-app-deploy-kustomization.yaml]: https://github.com/kingdonb/csh-flux/commit/7c3f1e62e2a87a2157bc9a22db4f913cc30dc12e#diff-f6ebc9688433418f0724f3545c96c301f029fd5a15847b824eab04545e057e84
[Tanka - Using Jsonnet tutorial]: https://tanka.dev/tutorial/jsonnet
[Tanka - Parameterizing]: https://tanka.dev/tutorial/parameters
[example 10.2 library excerpt]: https://github.com/kingdonb/any_old_app/blob/release/0.10.2/manifests/example.libsonnet#L47-L63
[example 10.2 jsonnet]: https://github.com/kingdonb/any_old_app/blob/release/0.10.2/manifests/example.jsonnet
[examples 0.10.2-all]: https://github.com/kingdonb/any_old_app/releases?after=0.10.2-alpha5
[anguslees/kustomize-libsonnet]: (https://github.com/anguslees/kustomize-libsonnet)
[kubecfg yaml parser]: https://github.com/bitnami/kubecfg/blob/master/lib/kubecfg.libsonnet#L25
[bitnami-labs/kube-libsonnet]: https://github.com/bitnami-labs/kube-libsonnet
[example 10.3 jsonnet]: https://github.com/kingdonb/any_old_app/blob/release/0.10.3/manifests/example.jsonnet
[Manual Gating]: https://docs.flagger.app/usage/webhooks#manual-gating
[flux create tenant]: /cmd/flux_create_tenant
[Flux 2 Multi-Tenancy Guide]: https://github.com/fluxcd/flux2-multi-tenancy
[Mozilla SOPS]: /guides/mozilla-sops/
[example 10.4 jsonnet]: https://github.com/kingdonb/any_old_app/blob/release/0.10.4/manifests/example.jsonnet
[anguslees example jsonnet]: https://github.com/anguslees/kustomize-libsonnet/blob/master/example.jsonnet
[Kubernetes docs on Using Service Accounts]: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#use-multiple-service-accounts
[sops/issues/315]: https://github.com/mozilla/sops/issues/315
[using various cloud providers]: /guides/mozilla-sops/#using-various-cloud-providers
[Decrypt SOPS Secrets]: https://github.com/marketplace/actions/decrypt-sops-secrets
[Sops Binary Installer]: https://github.com/marketplace/actions/sops-binary-installer
[Push directory to another repository]: https://github.com/marketplace/actions/push-directory-to-another-repository
[Flux v2 image automation]: /guides/image-update/
[Image Automation Controllers]: /components/image/controller/
[Example of a build process with timestamp tagging]: /guides/sortable-image-tags/#example-of-a-build-process-with-timestamp-tagging
[Sortable image tags to use with automation]: /guides/sortable-image-tags/#formats-and-alternatives
