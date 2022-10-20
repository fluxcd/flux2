# RFC-NNNN Artifact Revision format and introduction of digests

**Status:** provisional

<!--
Status represents the current state of the RFC.
Must be one of `provisional`, `implementable`, `implemented`, `deferred`, `rejected`, `withdrawn`, or `replaced`.
-->

**Creation date:** 2022-10-20

**Last update:** 2022-10-20

## Summary

This RFC proposes to establish a canonical format for an `Artifact` which
points to a specific checksum (e.g. an OCI manifest digest or Git commit SHA)
of a named pointer (e.g. an OCI image tag or Git tag). In addition, it proposes
to include the algorithm name (e.g. `sha256`) as a prefix to any advertised
checksum in an `Artifact` and further referring to it as a `Digest` opposed to
a `Checksum`.

## Motivation

The current `Artifact` type's `Revision` field format is not "officially"
standardized (albeit assumed throughout our code bases), and has mostly been
derived from `GitRepository` which uses `/` as a delimiter between the named
pointer (a Git branch or tag) and a specific (SHA-1, or theoretical SHA-256)
revision.

Since the introduction of `OCIRepository` and with the recent changes around
`HelmChart` objects to allow the consumption of charts from OCI registries.
This could be seen as outdated or confusing due to the format differing from
the canonical format used by OCI, which is `<tag>@<algo>:<checksum>` (the
part after `@` formally known as a "digest") to refer to a specific version
of a tagged OCI manifest.

While also taking note that Git does not have an official canonical format for
e.g. branch references at a specific commit, and `/` has less of a symbolic
meaning than `@`. Which could be interpreted as "`<branch>` _at_
`<commit SHA>`".

In addition, with the introduction of algorithm prefixes for an `Artifact`'s
checksum. It would be possible to add support and allow user configuration of
other algorithms than SHA-256. For example SHA-384 and SHA-512, or the more
performant (parallelizable) [BLAKE3][].

Besides this, it would make it easier to implement a client that can verify the
checksum without having to resort to an assumed format or guessing
method based on the length of it, and allows for a more robust solution in
which it can continue to calculate against the algorithm of a previous
configuration.

The inclusion of the `Artifact`'s algorithm prefix has been proposed before in
[source-controller#855](https://github.com/fluxcd/source-controller/issues/855),
with supportive response from Core Maintainers.

### Goals

- Establish a canonical format to refer to an `Artifact`'s `Revision` field
  which consists of a named pointer and a specific checksum reference.
- Allow easier verification of the `Artifact`'s checksum by including an
  alias for the algorithm.
- Allow configuration of the algorithm used to calculate the checksum of an
  `Artifact`.
- Allow configuration of [BLAKE3][] as an alternative to SHA for calculating
  checksums. This has promising performance improvements over SHA-256, which
  could allow for performance improvements in large scale environments.
- Allow compatability with SemVer name references which might contain an `@`
  symbol already (e.g. `package@v1.0.0@sha256:...`, opposed to OCI's
  `tag:v1.0.0@sha256:...`).

### Non-Goals

- Define a canonical format for an `Artifact`'s `Revision` field which contains
  a named pointer and a different reference than a checksum.

## Proposal

### Establish an Artifact Revision format

Change the format of the `Revision` field of the `source.toolkit.fluxcd.io`
Group's `Artifact` type across all `Source` kinds to contain an `@` delimiter
opposed to `/`, and include the algorithm alias as a prefix to the checksum
(creating a "digest").

```text
[ <named pointer> ] [ [ "@" ] <algo> ":" <checksum> ]
```

Where `<named pointer>` is the name of e.g. a Git branch or OCI tag,
`<checksum>` is the exact revision (e.g. a Git commit SHA or OCI manifest
digest), and `<algo>` is the alias of the algorithm used to calculate the
checksum (e.g. `sha256`). In case only a named pointer or digest is advertised,
the `@` is omitted.

For a `GitRepository`'s `Artifact` pointing towards an SHA-256 Git commit on
branch `main`, the `Revision` field value would become:

```text
main@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

For a `GitRepository`'s `Artifact` pointing towards a specific SHA-1 Git commit
without a defined branch or tag, the `Revision` field value would become:

```text
sha1:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

### Change the Artifact Checksum to a Digest

Change the format of the `Checksum` field of the `source.toolkit.fluxcd.io`
Group's `Artifact` to `Digest`, and include the alias of the algorithm used to
calculate the Artifact checksum as a prefix (creating a "digest").

```text
<algo> ":" <checksum>
```

For a `GitRepository` `Artifact`'s checksum calculated using SHA-256, the
`Digest` field value would become:

```text
sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

### User Stories

> As a user of the source-controller, I want to be able to see the exact
> revision of an Artifact that is being used, so that I can verify that it
> matches the expected revision.

> As a user of the notification-controller, I want to be able to see the
> exact revision a notification is referring to.

> As a Flux CLI user, I want to see the current revision of my Source in a
> listed overview.

<!--
Optional if existing discussions and/or issues are linked in the motivation section.
-->

### Alternatives

<!--
List plausible alternatives to the proposal and explain why the proposal is superior.

This is a good place to incorporate suggestions made during discussion of the RFC.
-->

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs and code snippets.

The design details should address at least the following questions:
- How can this feature be enabled / disabled?
- Does enabling the feature change any default behavior?
- Can the feature be disabled once it has been enabled?
- How can an operator determine if the feature is in use?
- Are there any drawbacks when enabling this feature?
-->

## Implementation History

<!--
Major milestones in the lifecycle of the RFC such as:
- The first Flux release where an initial version of the RFC was available.
- The version of Flux where the RFC graduated to general availability.
- The version of Flux where the RFC was retired or superseded.
-->

[BLAKE3]: https://github.com/BLAKE3-team/BLAKE3