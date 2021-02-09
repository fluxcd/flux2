<!-- -*- fill-column: 100 -*- -->
# How to make sortable image tags to use with automation

Flux v2 does not support selecting the lastest image by build time. Obtaining the build time needs
the container config for each image, and fetching that is subject to strict rate limiting by image
registries (e.g., by [DockerHub][dockerhub-rates]).

This guide explains how to construct image tags so that the most recent image has the tag that comes
last in alphabetical order. The technique suggested is to put a timestamp or serial number in each
image tag.

## Formats and alternatives

The important properties for sorting alphabetically are that the parts of the timestamp go from most
significant to least (e.g., the year down to the second), and that the output is always the same
number of characters.

Image tags are often shown in user interfaces, so readability matters. Here are some alternatives:

```bash
$ # seconds-since-epoch (used in the example above)
$ date +%s
1611840548
$ # date and time (remember ':' is not allowed in a tag)
$ date +%F.%H%M%S
2021-01-28.133158
```

Alternatively, you can use a stable serial number as part of the tag.  Some CI platforms will
provide a build number in an environment variable, but that may not be reliable to use as a serial
number -- check the platform documentation.

### Other things to include in the image tag

It is also handy to quickly trace an image to the branch and commit of its source code. Including
the branch also means you can filter for images from a particular branch.

In sum, a useful tag format is

    <branch>-<sha1>-<timestamp>

The branch and tag will usually be made available in a CI platform as environment variables. See

 - [CircleCI's built-in variables `CIRCLE_BRANCH` and `CIRCLE_SHA1`][circle-ci-env]
 - [GitHub Actions' `GITHUB_REF` and `GITHUB_SHA`][github-actions-env]
 - [Travis CI's `TRAVIS_BRANCH` and `TRAVIS_COMMIT`][travis-env].

## Example of a build process with timestamp tagging

Here is an example of a [GitHub Actions job][gha-syntax] that creates a "build ID" with the git
branch, SHA1, and a timestamp, and uses it as a tag when building an image:

```yaml
jobs:
  build-push:
    env:
        IMAGE: org/my-app
    runs-on: ubuntu-latest
    steps:

    - name: Generate build ID
      id: prep
      run: |
          branch=${GITHUB_REF##*/}
          sha=${GITHUB_SHA::8}
          ts=$(date +%s)
          echo "::set-output name=BUILD_ID::${branch}-${sha}-${ts}"

    # These are prerequisites for the docker build step
    - name: Set up QEMU
      uses: docker/setup-qemu-action@v1
    - name: Set up Docker Buildx
      uses: docker/setup-buildx-action@v1
    - name: Login to DockerHub
      uses: docker/login-action@v1
      with:
          username: ${{ secrets.DOCKERHUB_USERNAME }}
          password: ${{ secrets.DOCKERHUB_TOKEN }}

    - name: Build and publish container image with tag
      uses: docker/build-push-action@v2
      with:
          push: true
          context: .
          file: ./Dockerfile
          tags: |
            ${{ env.IMAGE }}:${{ steps.prep.outputs.BUILD_ID }}
```

## Using in an `ImagePolicy` object

When creating an `ImagePolicy` object, you will need to extract just the timestamp part of the tag,
using the `tagFilter` field. You can filter for a particular branch to restrict images to only those
built from that branch.

Here is an example that filters for only images built from `main` branch, and selects the most
recent according the timestamp (created with `date +%s`):

```
apiVersion: image.toolkit.fluxcd.io/v1alpha1
kind: ImagePolicy
metadata:
  name: image-repo-policy
  namespace: flux-system
spec:
  imageRepositoryRef:
    name: image-repo
  filterTags:
    pattern: '^main-[a-f0-9]+-(?P<ts>[0-9]+)'
    extract: '$ts'
  policy:
    alphabetical:
      order: asc
```

If you don't care about the branch, that part can be a wildcard in the pattern:

```
apiVersion: image.toolkit.fluxcd.io/v1alpha1
kind: ImagePolicy
metadata:
  name: image-repo-policy
  namespace: flux-system
spec:
  imageRepositoryRef:
    name: image-repo
  filterTags:
    pattern: '^.+-[a-f0-9]+-(?P<ts>[0-9]+)'
    extract: '$ts'
  policy:
    alphabetical:
      order: asc
```

[circle-ci-env]: https://circleci.com/docs/2.0/env-vars/#built-in-environment-variables
[github-actions-env]: https://docs.github.com/en/actions/reference/environment-variables#default-environment-variables
[travis-env]: https://docs.travis-ci.com/user/environment-variables/#default-environment-variables
[dockerhub-rates]: https://docs.docker.com/docker-hub/billing/faq/#pull-rate-limiting-faqs
[gha-syntax]: https://docs.github.com/en/actions/reference/workflow-syntax-for-github-actions
