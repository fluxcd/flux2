# Flux release note template

This is a template for release notes. It is intended to be used as a
starting point for writing release notes for a new release. It should be copied
to a temporary file, and then edited to reflect the changes in the release.

Once the release notes are complete, you can tag the release and push it to
GitHub.

After the release is tagged, the CI will build the release artifacts and upload
them to the GitHub release page. The release notes can then be copied from the
temporary file to the GitHub release page.

The release notes should be formatted using [Markdown](https://guides.github.com/features/mastering-markdown/),
and not make use of line breaks unless they function as paragraph breaks.

For examples of release notes, including language and formatting of the release
highlights, see the [Flux release notes](https://github.com/fluxcd/flux2/releases).

## GitHub release template

The following template can be used for the GitHub release page:

```markdown
## Highlights

<!-- Text describing the most important changes in this release -->

### Fixes and improvements

<!-- List of fixes and improvements to the controllers and CLI -->

## New documentation

<!-- List of new documentation pages, if applicable -->

## Components changelog

- <name>-controller [v<version>](https://github.com/fluxcd/<name>-controller/blob/<version>/CHANGELOG.md

## CLI changelog

<!-- auto-generated list of pull requests to the CLI starts here -->
```

In some scenarios, you may want to include specific information about API
changes and/or upgrade procedures. Consult [the formatting of
`v2.0.0-rc.1`](https://github.com/fluxcd/flux2/releases/tag/v2.0.0-rc.1) for
such an example.

## Slack message template

The following template can be used for the Slack release message:

```markdown
:sparkles: *We are pleased to announce the release of Flux [<version>](https://github.com/fluxcd/flux2/releases/tag/<version>/)!*

<!-- Brief description of most important changes to this release -->

:hammer_and_pick: *Fixes and improvements*

<!-- List of fixes and improvements as in the GitHub release template -->

:books: Documentation

<!-- List of new documentation pages, if applicable -->

:heart: Big thanks to all the Flux contributors that helped us with this release! 
```

For more concrete examples, see the pinned messages in the [Flux Slack
channel](https://cloud-native.slack.com/archives/CLAJ40HV3).