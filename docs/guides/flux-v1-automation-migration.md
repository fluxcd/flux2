<!-- -*- fill-column: 100 -*- -->
# Migrating image update automation to Flux v2

"Image Update Automation" is a process in which Flux makes commits to your Git repository when it
detects that there is a new image to be used in a workload (e.g., a Deployment). In Flux v2 this
works quite differently to how it worked in Flux v1. This guide explains the differences and how to
port your cluster configuration from v1 to v2. There is also a [tutorial for using image update
automation with a new cluster][image-update-tute].

## Overview of changes between v1 and v2

In Flux v1, image update automation (from here, just "automation") was built into the Flux daemon,
which scanned everything it found in the cluster and updated the Git repository it was syncing.

In Flux v2,

 - automation is controlled with custom resources, not annotations
 - ordering images by build time is not supported (there is [a section
   below](#how-to-migrate-annotations-to-image-policies) explaining what to do instead)
 - the fields to update in files are marked explicitly, rather than inferred from annotations.

#### Automation is now controlled by custom resources

Flux v2 breaks down the functions in Flux v1's daemon into controllers, with each having a specific
area of concern. Automation is now done by two controllers: one which scans image repositories to
find the latest images, and one which uses that information to commit changes to git
repositories. These are in turn separate to the syncing controllers.

This means that automation in Flux v2 is governed by custom resources. In Flux v1 the daemon scanned
everything, and looked at annotations on the resources to determine what to update. Automation in v2
is more explicit than in v1 -- you have to mention exactly which images you want to be scanned, and
which fields you want to be updated.

A consequence of using custom resources is that with Flux v2 you can have an arbitrary number of
automations, targeting different Git repositories if you wish, and updating different sets of
images. If you run a multitenant cluster, the tenants can define automation in their own namespaces,
for their own Git repositories.

#### Selecting an image is more flexible

The ways in which you choose to select an image have changed. In Flux v1, you generally supply a
filter pattern, and the latest image is the image with the most recent build time out of those
filtered. In Flux v2, you choose an ordering, and separately specify a filter for the tags to
consider. These are dealt with in detail below.

Selecting an image by build time is no longer supported. This is the implicit default in Flux v1. In
Flux v2, you will need to tag images so that they sort in the order you would like -- [see
below](#how-to-use-sortable-image-tags) for how to do this conveniently.

#### Fields to update are explicitly marked

Lastly, in Flux v2 the fields to update in files are marked explicitly. In Flux v1 they are inferred
from the type of the resource, along with the annotations given. The approach in Flux v1 was limited
to the types that had been programmed in, whereas Flux v2 can update any Kubernetes object (and some
files that aren't Kubernetes objects, like `kustomization.yaml`).

## Preparing for migration

It is best to complete migration of your system to _Flux v2 syncing_ first, using the [Flux v1
migration guide][flux-v1-migration]. This will remove Flux v1 from the system, along with its image
automation. You can then reintroduce automation with Flux v2 by following the instructions in this
guide.

It is safe to leave the annotations for Flux v1 in files while you reintroduce automation, because
Flux v2 will ignore them.

To migrate to Flux v2 automation, you will need to do three things:

 - make sure you are running the automation controllers; then,
 - declare the automation with an `ImageUpdateAutomation` object; and,
 - migrate each manifest by translate Flux v1 annotations to Flux v2 `ImageRepository` and
   `ImagePolicy` objects, and putting update markers in the manifest file.

### Where to keep `ImageRepository`, `ImagePolicy` and `ImageUpdateAutomation` manifests

This guide assumes you want to manage automation itself via Flux. In the following sections,
manifests for the objects controlling automation are saved in files, committed to Git, and applied
in the cluster with Flux.

A Flux v2 installation will typically have a Git repository structured like this:

```
<...>/flux-system/
        gotk-components.yaml
        gotk-sync.yaml
<...>/app/
        # deployments etc.
```

The `<...>` is the path to a particular cluster's definitions -- this may be simply `.`, or
something like `clusters/my-cluster`. To get the files in the right place, set a variable for this
path:

```bash
$ CLUSTER_PATH=<...> # e.g., "." or "clusters/my-cluster", or ...
$ AUTO_PATH=$CLUSTER_PATH/automation
$ mkdir ./$AUTO_PATH
```

The file `$CLUSTER_PATH/flux-system/gotk-components.yaml` has definitions of all the Flux v2
controllers and custom resource definitions. The file `gotk-sync.yaml` defines a `GitRepository` and
a `Kustomization` which will sync manifests under `$CLUSTER_PATH/`.

To these will be added definitions for automation objects. This guide puts manifest files for
automation in `$CLUSTER_PATH/automation/`, but there is no particular structure required
by Flux. The automation objects do not have to be in the same namespace as the objects to be
updated.

#### Migration on a branch

This guide assumes you will commit changes to the branch that is synced by Flux, as this is the
simplest way to understand.

It may be less disruptive to put migration changes on a branch, then merging when you have completed
the migration. You would need to either change the `GitRepository` to point at the migration branch,
or have separate `GitRepository` and `Kustomization` objects for the migrated parts of your Git
repository. The main thing to avoid is syncing the same objects in two different places; e.g., avoid
having Kustomizations that sync both the unmigrated and migrated application configuration.

### Installing the command-line tool `flux`

The command-line tool `flux` will be used below; see [these instructions][install-cli] for how to
install it.

## Running the automation controllers

The first thing to do is to deploy the automation controllers to your cluster. The best way to
proceed will depend on the approach you took when following the [Flux read-only migration
guide][flux-v1-migration].

 - If you used `flux bootstrap` to create a new Git repository, then ported your cluster
   configuration to that repository, use [After `flux bootstrap`](#after-flux-bootstrap);
 - If you used `flux install` to install the controllers directly, use [After migrating Flux v1 in
   place](#after-migrating-flux-v1-in-place);
 - If you used `flux install` and exported the configuration to a file, use [After committing Flux
   v2 configuration to Git](#after-committing-a-flux-v2-configuration-to-git).

### After `flux bootstrap`

When starting from scratch, you are likely to have used `flux bootstrap`. Rerun the command, and
include the image automation controllers in your starting configuration with the flag
`--components-extra`, [as shown in the installation guide][flux-bootstrap].

This will commit changes to your Git repository and sync them in the cluster.

```bash
flux check --components-extra=image-reflector-controller,image-automation-controller
```

Now jump to the section [Migrating each manifest to Flux v2](#migrating-each-manifest-to-flux-v2).

### After migrating Flux v1 in place

If you followed the [Flux v1 migration guide][flux-v1-migration], you will already be running some
Flux v2 controllers. The automation controllers are currently considered an optional extra to those,
but are installed and run in much the same way. You may or may not have committed the Flux v2
configuration to your Git repository. If you did, go to the section [After committing Flux v2
configuration to Git](#after-committing-flux-v2-configuration-to-git).

If _not_, you will be installing directly to the cluster:

```bash
$ flux install --components-extra=image-reflector-controller,image-automation-controller
```

It is safe to repeat the installation command, or to run it after using `flux bootstrap`, so long as
you repeat any arguments you supplied the first time.

Now jump ahead to [Migrating each manifest to Flux v2](#migrating-each-manifest-to-flux-v2).

#### After committing a Flux v2 configuration to Git

If you added the Flux v2 configuration to your git repository, assuming it's in the file
`$CLUSTER_PATH/flux-system/gotk-components.yaml` as used in the guide, use `flux install` and write
it back to that file:

```bash
$ flux install \
    --components-extra=image-reflector-controller,image-automation-controller \
    --export > "$CLUSTER_PATH/flux-system/gotk-components.yaml"
```

Commit changes to the `$CLUSTER_PATH/flux-system/gotk-components.yaml` file and sync the cluster:

```bash
$ git add $CLUSTER_PATH/flux-system/gotk-components.yaml
$ git commit -s -m "Add image automation controllers to Flux config"
$ git push
$ flux reconcile kustomization --with-source flux-system
```

## Controlling automation with an `ImageUpdateAutomation` object

In Flux v1, automation was run by default. With Flux v2, you have to explicitly tell the controller
which Git repository to update and how to do so. These are defined in an `ImageUpdateAutomation`
object; but first, you need a `GitRepository` with write access, for the automation to use.

If you followed the [Flux v1 read-only migration guide][flux-v1-migration], you will have a
`GitRepository` defined in the namespace `flux-system`, for syncing to use. This `GitRepository`
will have _read_ access to the Git repository by default, and automation needs _write_ access to
push commits.

To give it write access, you can replace the secret it refers to. How to do this will depend on what
kind of authentication you used to install Flux v2.

### Replacing the Git credentials secret

The secret with Git credentials will be named in the `.spec.secretRef.name` field of the
`GitRepository` object. Say your `GitRepository` is in the _namespace_ `flux-system` and _named_
`flux-system` (these are the defaults if you used `flux bootstrap`); you can retrieve the secret
name and Git URL with:

```bash
$ FLUX_NS=flux-system
$ GIT_NAME=flux-system
$ SECRET_NAME=$(kubectl -n $FLUX_NS get gitrepository $GIT_NAME -o jsonpath={.spec.secretRef.name})
$ GIT_URL=$(kubectl -n $FLUX_NS get gitrepository $GIT_NAME -o jsonpath='{.spec.url}')
$ echo $SECRET_NAME $GIT_URL # make sure they have values
```

If you're not sure which kind of credentials you're using, look at the secret:

```bash
$ kubectl -n $FLUX_NS describe secret $SECRET_NAME
```

An entry at `.data.identity` indicates that you are using an SSH key (the [first
section](#replacing-an-ssh-key-secret) below); an entry at `.data.username` indicates you are using
a username and password or token (the [second section](#replacing-a-usernamepassword-secret)
below).

#### Replacing an SSH key secret

When using an SSH (deploy) key, create a new key:

```bash
$ flux create secret git -n $FLUX_NS $SECRET_NAME --url=$GIT_URL
```

You will need to copy the public key that's printed out, and install that as a deploy key for your
Git repo **making sure to check the 'All write access' box** (or otherwise give the key write
permissions). Remove the old deploy key.

#### Replacing a username/password secret

When you're using a username and password to authenticate, you may be able to change the permissions
associated with that account.

If not, you will need to create a new access token (e.g., ["Personal Access Token"][github-pat] in
GitHub). In this case, once you have the new token you can replace the secret with the following:

```bash
$ flux create secret git -n $FLUX_NS $SECRET_NAME \
    --username <username> --password <token> --url $GIT_URL
```

#### Checking the new credentials

To check if your replaced credentials still work, try syncing the `GitRepository` object:

```bash
$ flux reconcile source git -n $FLUX_NS $GIT_NAME
► annotating GitRepository flux-system in flux-system namespace
✔ GitRepository annotated
◎ waiting for GitRepository reconciliation
✔ GitRepository reconciliation completed
✔ fetched revision main/d537304e8f5f41f1584ca1e807df5b5752b2577e
```

When this is successful, it tells you the new credentials have at least read access.

### Making an automation object

To set automation running, you create an [`ImageUpdateAutomation`][auto-ref] object. Each object
will update a Git repository, according to the image policies in the namespace.

Here is an `ImageUpdateAutomation` manifest for the example (note: you will have to supply your own
value for at least the host part of the email address):

```yaml
$ # the environment variables $AUTO_PATH and $GIT_NAME are set above
$ FLUXBOT_EMAIL=fluxbot@example.com # supply your own host or address here
$ flux create image update my-app-auto \
    --author-name FluxBot --author-email "$FLUXBOT_EMAIL" \
    --git-repo-ref $GIT_NAME --branch main \
    --interval 5m \
    --export > ./$AUTO_PATH/my-app-auto.yaml
$ cat my-app-auto.yaml
---
apiVersion: image.toolkit.fluxcd.io/v1alpha1
kind: ImageUpdateAutomation
metadata:
  name: my-app-auto
  namespace: flux-system
spec:
  checkout:
    branch: main
    gitRepositoryRef:
      name: flux-system
  commit:
    authorEmail: fluxbot@example.com
    authorName: FluxBot
  interval: 5m0s
```

#### Commit and check that the automation object works

Commit the manifeat file and push:

```bash
$ git add ./$AUTO_PATH/my-app-auto.yaml
$ git commit -s -m "Add image update automation"
$ git push
# ...
```

Then sync and check the object status:

```bash
$ flux reconcile kustomization --with-source flux-system
► annotating GitRepository flux-system in flux-system namespace
✔ GitRepository annotated
◎ waiting for GitRepository reconciliation
✔ GitRepository reconciliation completed
✔ fetched revision main/401dd3b550f82581c7d12bb79ade389089c6422f
► annotating Kustomization flux-system in flux-system namespace
✔ Kustomization annotated
◎ waiting for Kustomization reconciliation
✔ Kustomization reconciliation completed
✔ reconciled revision main/401dd3b550f82581c7d12bb79ade389089c6422f
$ flux get image update
NAME            READY   MESSAGE         LAST RUN                SUSPENDED
my-app-auto     True    no updates made 2021-02-08T14:53:43Z    False
```

Read on to the next section to see how to change each manifest file to work with Flux v2.

## Migrating each manifest to Flux v2

In Flux v1, the annotation

    fluxcd.io/automated: "true"

switches automation on for a manifest (a description of a Kubernetes object). For each manifest that
has that annotation, you will need to create custom resources to scan for the latest image, and to
replace the annotations with field markers.

The following sections explain these steps, using this example Deployment manifest which is
initially annotated to work with Flux v1:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: default
  annotations:
    fluxcd.io/automated: "true"
    fluxcd.io/tag.app: semver:^5.0
  selector:
    matchLabels:
      app: podinfo
spec:
  template:
    metadata:
      labels:
        app: podinfo
    spec:
      containers:
      - name: app
        image: ghcr.io/stefanprodan/podinfo:5.0.0
```

!!! warning
    A YAML file may have more than one manifest in it, separated with
    `---`. Be careful to account for each manifest in a file.

You may wish to try migrating the automation of just one file or manifest and follow it through to
the end of the guide, before returning here to do the remainder.

### How to migrate annotations to image policies

For each image repository that is the subject of automation you will need to create an
`ImageRepository` object, so that the image repository is scanned for tags. The image repository in
the example deployment is `ghcr.io/stefanprodan/podinfo`, which is the image reference minus its
tag:

```yaml
$ cat $CLUSTER_PATH/app/my-app.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: default
  annotations:
    fluxcd.io/automated: "true"
    fluxcd.io/tag.app: semver:^5.0
  selector:
    matchLabels:
      app: podinfo
spec:
  template:
    metadata:
      labels:
        app: podinfo
    spec:
      containers:
      - name: app
        image: ghcr.io/stefanprodan/podinfo:5.0.0 # <-- image reference
```

The command-line tool `flux` will help create a manifest for you. Note that the output is redirected
to a file under `$AUTO_PATH`, so it can be added to the Git repository and synced to the cluster.

```bash
$ # the environment variable $AUTO_PATH was set earlier
$ flux create image repository podinfo-image \
    --image ghcr.io/stefanprodan/podinfo \
    --interval 5m \
    --export > ./$AUTO_PATH/podinfo-image.yaml
$ cat ./$AUTO_PATH/podinfo-image.yaml
---
apiVersion: image.toolkit.fluxcd.io/v1alpha1
kind: ImageRepository
metadata:
  name: podinfo-image
  namespace: flux-system
spec:
  image: ghcr.io/stefanprodan/podinfo
  interval: 5m0s
```

!!! hint
    If you are using the same image repository in several manifests, you only need one
    `ImageRepository` object for it.

##### Using image registry credentials for scanning

When your image repositories are private, you supply Kubernetes with "image pull secrets" with
credentials for accessing the image registry (e.g., DockerHub). The image reflector controller needs
the same kind of credentials to scan image repositories.

There are several ways that image pull secrets can be made available for the image reflector
controller. The [image update tutorial][image-update-tute-creds] describes how to create or arrange
secrets for scanning to use. Also see later in the tutorial for [instructions specific to some cloud
platforms][image-update-tute-clouds].

##### Committing and checking the ImageRepository

Add the `ImageRepository` manifest to the Git index and commit it:

```bash
$ git add ./$AUTO_PATH/podinfo-image.yaml
$ git commit -s -m "Add image repository object for podinfo"
$ git push
# ...
```

Now you can sync the new commit, and check that the object is working:

```bash
$ flux reconcile kustomization --with-source flux-system
► annotating GitRepository flux-system in flux-system namespace
✔ GitRepository annotated
◎ waiting for GitRepository reconciliation
✔ GitRepository reconciliation completed
✔ fetched revision main/fd2fe8a61d4537bcfa349e4d1dbc480ea699ba8a
► annotating Kustomization flux-system in flux-system namespace
✔ Kustomization annotated
◎ waiting for Kustomization reconciliation
✔ Kustomization reconciliation completed
✔ reconciled revision main/fd2fe8a61d4537bcfa349e4d1dbc480ea699ba8a
$ flux get image repository podinfo-image
NAME            READY   MESSAGE                         LAST SCAN               SUSPENDED
podinfo-image   True    successful scan, found 16 tags  2021-02-08T14:31:38Z    False
```

#### Replacing automation annotations

For each _field_ that's being updated by automation, you'll need an `ImagePolicy` object to describe
how to select an image for the field value. In the example, the field `.image` in the container
named `"app"` is the field being updated.

In Flux v1, annotations describe how to select the image to update to, using a prefix. In the
example, the prefix is `semver:`:

```yaml
  annotations:
    fluxcd.io/automated: "true"
    fluxcd.io/tag.app: semver:^5.0
```

These are the prefixes supported in Flux v1, and what to use in Flux v2:

| Flux v1 prefix | Meaning | Flux v2 equivalent |
|----------------|---------|--------------------|
| `glob:`        | Filter for tags matching the glob pattern, then select the newest by build time | [Use sortable tags](#how-to-use-sortable-image-tags) |
| `regex:`       | Filter for tags matching the regular expression, then select the newest by build time |[Use sortable tags](#how-to-use-sortable-image-tags) |
| `semver:`      | Filter for tags that represent versions, and select the highest version in the given range | [Use semver ordering](#how-to-use-semver-image-tags) |

#### How to use sortable image tags

To give image tags a useful ordering, you can use a timestamp or serial number as part of each
image's tag, then sort either alphabetically or numerically.

This is a change from Flux v1, in which the build time was fetched from each image's config, and
didn't need to be included in the image tag. Therefore, this is likely to require a change to your
build process.

The guide [How to make sortable image tags][image-tags-guide] explains how to change your build
process to tag images with a timestamp. This will mean Flux v2 can sort the tags to find the most
recently built image.

##### Filtering the tags in an `ImagePolicy`

The recommended format for image tags using a timestamp is:

    <branch>-<sha1>-<timestamp>

The timestamp (or serial number) is the part of the tag that you want to order on. The SHA1 is there
so you can trace an image back to the commit from which it was built. You don't need the branch for
sorting, but you may want to include only builds from a specific branch.

Say you want to filter for only images that are from `main` branch, and pick the most recent. Your
`ImagePolicy` would look like this:

```yaml
apiVersion: image.toolkit.fluxcd.io/v1alpha1
kind: ImagePolicy
metadata:
  name: my-app-policy
  namespace: flux-system
spec:
  imageRepositoryRef:
    name: podinfo-image
  filterTags:
    pattern: '^main-[a-f0-9]+-(?P<ts>[0-9]+)'
    extract: '$ts'
  policy:
    numerical:
      order: asc
```

The `.spec.filterTags.pattern` field gives a regular expression that a tag must match to be included. The
`.spec.filterTags.extract` field gives a replacement pattern that can refer back to capture groups in the
filter pattern. The extracted values are sorted to find the selected image tag. In this case, the
timestamp part of the tag will be extracted and sorted numerically in ascending order. See [the
reference docs][imagepolicy-ref] for more examples.

Once you have made sure you have image tags and an `ImagePolicy`, jump ahead to [Checking
the ImagePolicy works](#checking-that-the-image-policy-works).

### How to use SemVer image tags

The other kind of sorting is by [SemVer][semver], picking the highest version from among those
included by the filter. A semver range will also filter for tags that fit in the range. For example,

```yaml
    semver:
      range: ^5.0
```

includes only tags that have a major version of `5`, and selects whichever is the highest.

This can be combined with a regular expression pattern, to filter on other parts of the tags. For
example, you might put a target environment as well as the version in your image tags, like
`dev-v1.0.3`.

Then you would use an `ImagePolicy` similar to this one:

```yaml
apiVersion: image.toolkit.fluxcd.io/v1alpha1
kind: ImagePolicy
metadata:
  name: my-app-policy
  namespace: flux-system
spec:
  imageRepositoryRef:
    name: podinfo-image
  filterTags:
    pattern: '^dev-v(?P<version>.*)'
    extract: '$version'
  policy:
    semver:
      range: '^1.0'
```

Continue on to the next sections to see an example, and how to check that your `ImagePolicy` works.

#### An `ImagePolicy` for the example

The example Deployment has annotations using `semver:` as a prefix, so the policy object also uses
semver:

```bash
$ # the environment variable $AUTO_PATH was set earlier
$ flux create image policy my-app-policy \
    --image-ref podinfo-image \
    --semver '^5.0' \
    --export > ./$AUTO_PATH/my-app-policy.yaml
$ cat ./$AUTO_PATH/my-app-policy.yaml
---
apiVersion: image.toolkit.fluxcd.io/v1alpha1
kind: ImagePolicy
metadata:
  name: my-app-policy
  namespace: flux-system
spec:
  imageRepositoryRef:
    name: podinfo-image
  policy:
    semver:
      range: ^5.0
```

#### Checking that the `ImagePolicy` works

Commit the manifest file, and push:

```bash
$ git add ./$AUTO_PATH/my-app-policy.yaml
$ git commit -s -m "Add image policy for my-app"
$ git push
# ...
```

Then you can reconcile and check that the image policy works:

```bash
$ flux reconcile kustomization --with-source flux-system
► annotating GitRepository flux-system in flux-system namespace
✔ GitRepository annotated
◎ waiting for GitRepository reconciliation
✔ GitRepository reconciliation completed
✔ fetched revision main/7dcf50222499be8c97e22cd37e26bbcda8f70b95
► annotating Kustomization flux-system in flux-system namespace
✔ Kustomization annotated
◎ waiting for Kustomization reconciliation
✔ Kustomization reconciliation completed
✔ reconciled revision main/7dcf50222499be8c97e22cd37e26bbcda8f70b95
$ flux get image policy flux-system
NAME            READY   MESSAGE                                                                 LATEST IMAGE
my-app-policy   True    Latest image tag for 'ghcr.io/stefanprodan/podinfo' resolved to: 5.1.4  ghcr.io/stefanprodan/podinfo:5.1.4
```

### How to mark up files for update

The last thing to do in each manifest is to mark the fields that you want to be updated.

In Flux v1, the annotations in a manifest determines the fields to be updated. In the example, the
annotations target the image used by the container `app`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: default
  annotations:
    fluxcd.io/automated: "true"
    fluxcd.io/tag.app: semver:^5.0 # <-- `.app` here
  selector:
    matchLabels:
      app: podinfo
spec:
  template:
    metadata:
      labels:
        app: podinfo
    spec:
      containers:
      - name: app                  # <-- targets `app` here
        image: ghcr.io/stefanprodan/podinfo:5.0.0
```

This works straight-forwardly for Deployment manifests, but when it comes to `HelmRelease`
manifests, it [gets complicated][helm-auto], and it doesn't work at all for many kinds of resources.

For Flux v2, you mark the field you want to be updated directly, with the namespaced name of the
image policy to apply. This is the example Deployment, marked up for Flux v2:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: default
  name: my-app
  selector:
    matchLabels:
      app: podinfo
spec:
  template:
    metadata:
      labels:
        app: podinfo
    spec:
      containers:
      - name: app
        image: ghcr.io/stefanprodan/podinfo:5.0.0 # {"$imagepolicy": "flux-system:my-app-policy"}
```

The value `flux-system:my-app-policy` names the policy that selects the desired image.

This works in the same way for `DaemonSet` and `CronJob` manifests. For `HelmRelease` manifests, put
the marker alongside the part of the `values` that has the image tag. If the image tag is a separate
field, you can put `:tag` on the end of the name, to replace the value with just the selected
image's tag. The [image automation guide][image-update-tute-custom] has examples for `HelmRelease`
and other custom resources.

### Committing the marker change and checking that automation works

Referring to the image policy created earlier, you can see the example Deployment does not use the
most recent image. When you commit the manifest file with the update marker added, you would expect
automation to update the file.

Commit the change that adds an update marker:

```bash
$ git add app/my-app.yaml # the filename of the example
$ git commit -s -m "Add update marker to my-app manifest"
$ git push
# ...
```

Now to check that the automation makes a change:

```bash
$ flux reconcile image update my-app-auto
► annotating ImageUpdateAutomation my-app-auto in flux-system namespace
✔ ImageUpdateAutomation annotated
◎ waiting for ImageUpdateAutomation reconciliation
✔ ImageUpdateAutomation reconciliation completed
✔ committed and pushed a92a4b654f520c00cb6c46b2d5e4fb4861aa58fc
```

## Troubleshooting

If a change was not pushed by the image automation, there's several things you can check:

 - it's possible it made a change that is not reported in the latest status -- pull from the origin
   and check the commit log
 - check that the name used in the marker corresponds to the namespace and name of an `ImagePolicy`
 - check that the `ImageUpdateAutomation` is in the same namespace as the `ImagePolicy` objects
   named in markers
 - check that the image policy and the image repository are both reported as `Ready`
 - check that the credentials referenced by the `GitRepository` object have write permission, and
   create new credentials if necessary.

As a fallback, you can scan the logs of the automation controller to see if it logged errors:

```bash
$ kubectl logs -n flux-system deploy/image-automation-controller
```

Once you are satisfied that it is working, you can migrate the rest of the manifests using the steps
from ["Migrating each manifest to Flux v2"](#migrating-each-manifest-to-flux-v2) above.

[image-update-tute]: https://toolkit.fluxcd.io/guides/image-update/
[imagepolicy-ref]: https://toolkit.fluxcd.io/components/image/imagepolicies/
[helm-auto]: https://docs.fluxcd.io/en/1.21.1/references/helm-operator-integration/#automated-image-detection
[image-update-tute-custom]: https://toolkit.fluxcd.io/guides/image-update/#configure-image-update-for-custom-resources
[flux-v1-migration]: https://toolkit.fluxcd.io/guides/flux-v1-migration/
[install-cli]: https://toolkit.fluxcd.io/get-started/#install-the-flux-cli
[flux-bootstrap]: https://toolkit.fluxcd.io/guides/installation/#bootstrap
[github-pat]: https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token
[auto-object-ref]: https://toolkit.fluxcd.io/components/image/imageupdateautomations/
[image-update-tute-creds]: https://toolkit.fluxcd.io/guides/image-update/#configure-image-scanning
[image-update-tute-clouds]: https://toolkit.fluxcd.io/guides/image-update/#imagerepository-cloud-providers-authentication
[image-tags-guide]: https://toolkit.fluxcd.io/guides/sortable-image-tags/
[auto-ref]: https://toolkit.fluxcd.io/components/image/imageupdateautomations/
[semver]: https://semver.org
