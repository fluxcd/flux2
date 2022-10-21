# RFC-NNNN Artifact Revision format and introduction of digests

**Status:** provisional

<!--
Status represents the current state of the RFC.
Must be one of `provisional`, `implementable`, `implemented`, `deferred`, `rejected`, `withdrawn`, or `replaced`.
-->

**Creation date:** 2022-10-20

**Last update:** 2022-10-21

## Summary

This RFC proposes to establish a canonical format for an `Artifact` which
points to a specific checksum (e.g. an OCI manifest digest or Git commit SHA)
of a named pointer (e.g. an OCI image tag or Git tag). In addition, it proposes
to include the algorithm name (e.g. `sha256`) as a prefix to any advertised
checksum in an `Artifact` and further referring to it as a `Digest` as opposed
to a `Checksum`.

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
- Allow compatibility with SemVer name references which might contain an `@`
  symbol already (e.g. `package@v1.0.0@sha256:...`, as opposed to OCI's
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

For a `Bucket`'s `Artifact` with a revision based on an SHA-256 calculation of
a list of object keys and their etags, the `Revision` field value would become:

```text
sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
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

#### Artifact revision verification

> As a user of the source-controller, I want to be able to see the exact
> revision of an Artifact that is being used, so that I can verify that it
> matches the expected revision at a remote source.

For a Source kind that has an `Artifact` with a `Revision` which contains a
checksum, the field value can be retrieved using the Kubernetes API. For
example:

```console
$ kubectl get gitrepository -o jsonpath='{.status.artifact.revision}' <name>
main@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

#### Artifact checksum verification

> As a user of the source-controller, I want to be able to verify the checksum
> of an Artifact.

For a Source kind with an `Aritfact` the digest consisting of the algorithm
alias and checksum is advertised in the `Digest` field, and can be retrieved
using the Kubernetes API. For example:

```console
$ kubectl get gitrepository -o jsonpath='{.status.artifact.digest}' <name>
sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

#### Artifact checksum algorithm configuration

> As a user of the source-controller, I want to be able to configure the
> algorithm used to calculate the checksum of an Artifact.

The source-controller binary accepts a `--artifact-digest-algo` flag which
configures the algorithm used to calculate the checksum of an `Artifact`.
The default value is `sha256`, but can be changed to `sha384`, `sha512`
or `blake3`.

When set, newly advertised `Artifact`'s `Digest` fields will be calculated
using the configured algorithm. For previous `Artifact`'s that were set using
a previous configuration, the `Artifact`'s `Digest` field will be calculated
using the advertised algorithm.

#### Artifact revisions in notifications

> As a user of the notification-controller, I want to be able to see the
> exact revision a notification is referring to.

The notification-controller can use the revision for a Source's `Artifact`
attached as an annotation to an `Event`, and correctly parses the value field
when attempting to extract e.g. a Git commit digest from an event for a
`GitRepository`. As currently already applicable for the `/` delimiter.

> As a user of the notification-controller, I want to be able to observe what
> commit has been applied on my (supported) Git provider.

The notification-controller can use the revision attached as an annotation to
an `Event`, and is capable of extracting the correct reference for a Git
provider integration (e.g. GitHub, GitLab) to construct a payload. For example,
extracting `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
from `main@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`.

#### Artifact revisions in listed views

> As a Flux CLI user, I want to see the current revision of my Source in a
> listed overview.

By running `flux get source <kind>`, the listed view of Sources would show a
truncated version of the checksum in the `Revision` field.

```console
$ flux get source gitrepository
NAME            REVISION              SUSPENDED       READY   MESSAGE                                                                      
flux-monitoring main@sha1:6f6c0979    False           True    stored artifact for revision 'main@sha1:6f6c0979809c12ce4aa687fb42be913f5dc78a75'

$ flux get source oci
NAME            REVISION               SUSPENDED       READY   MESSAGE                                                                                              
apps-source     local@sha256:b1ad9be6  False           True    stored artifact for digest 'local@sha256:b1ad9be6fe5fefc76a93f462ef2be1295fa6693d57e9d783780af99cd7234dc8'

$ flux get source bucket
NAME            REVISION         SUSPENDED       READY   MESSAGE                                                                                              
apps-source     sha256:e3b0c442  False           True    stored artifact for revision 'sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855'
```

### Alternatives

The two main alternatives around the `Revision` parts in this RFC are to either
keep the current field value formats as is, or to invent another format. Given
the [motivation](#motivation) for this RFC outlines the reasoning for not
keeping the current `Revision` format, and the proposed is a commonly known
format. Neither strike as a better alternative.

For the changes related to `Checksum` and `Digest`, the alternative is to keep
the current field name as is, and only change the field value format. However,
given the naming of the field is more correct with the introduction of the
algorithm alias, and now is the time to make last (breaking) changes to the
API. This does not strike as a better alternative.

## Design Details

### Artifact Revision format

For an `Artifact`'s `Revision` which contains a checksum referring to an exact
revision, the checksum within the value MUST be appended with an alias for the
algorithm separated by `:` (e.g. `sha256:...`), further referred to as a
"digest". The algorithm alias and checksum of the digest MUST be lowercase and
alphanumeric.

For an `Artifact`'s `Revision` which contains a digest and a named pointer,
it MUST be prefixed with `@`, and appended at the end of the `Revision` value.
The named pointer MAY contain arbitrary characters, including but not limited
to `/` and `@`.

#### Format

```text
[ <named pointer> ] [ [ "@" ] <algo> ":" <checksum> ]
```

Where `[ ]` indicates an optional element, `" "` a literal string, and `< >`
a variable.

#### Parsing

When parsing the `Revision` field value of an `Artifact` to extract the digest,
the value after the last `@` is considered to be the digest. The remaining
value on the left side is considered to be the named pointer, which MAY contain
an additional `@` delimiter if applicable for the domain of the `Source`
implementation.

#### Truncation

When truncating the `Revision` field value of an `Artifact` to display in a
view with limited space, the `<checksum>` of the digest MAY be truncated to
7 or more characters. The `<algo>` of the digest MUST NOT be truncated.
In addition, a digest MUST always contain the full length checksum for the
algorithm.

#### Backwards compatibility

To allow backwards compatability in the notification-controller, Flux CLI and
other applicable components. The `Revision` new field value format could be
detected by the presence of the `@` or `:` characters. Falling back to their
current behaviour if not present, phasing out the old format in a future
release.

### Artifact Digest

The `Artifact`'s `Digest` field advertises the checksum of the file in the
`URL`. The checksum within the value MUST be appended with an alias for the
algorithm separated by `:` (e.g. `sha256:...`). This follows the
[digest format][go-digest] of OCI.

#### Format

```text
<algo> ":" <checksum>
```

Where `" "` indicates a literal string, and `< >` a variable.

#### Library

The library used for calculating the `Digest` field value is
`github.com/opencontainers/go-digest`. This library is used by various
OCI libraries which we already depend on, stable and extensible.

#### Configuration

The checksum MUST be calculated using the algorithm configured in the
`--artifact-digest-algo` flag of the source-controller binary. The default
value is `sha256`, but can be changed to `sha384`, `sha512` or `blake3`.

**Note:** availability of BLAKE3 is at present dependent on an explicit import
of `github.com/opencontainers/go-digest/blake3`.

When the provided algorithm is NOT supported, the source-controller MUST
fail to start.

When the configured algorithm changes, the `Digest` MAY be recalculated to
update the value.

#### Calculation

The checksum MUST be calculated using the algorithm configured in the
`--artifact-digest-algo` flag of the source-controller binary.

#### Verification

The checksum of a downloaded artifact MUST be verified against the `Digest`
field value. If the checksum does not match, the verification MUST fail.

### Deprecation of Checksum

The `Artifact`'s `Checksum` field is deprecated and MUST be removed in a
future release. The `Digest` field MUST be used instead.

#### Backwards compatibility

To allow backwards compatability, the source-controller could continue
to advertise the checksum part of a `Digest` in the `Checksum` field until
the field is removed.

## Implementation History

<!--
Major milestones in the lifecycle of the RFC such as:
- The first Flux release where an initial version of the RFC was available.
- The version of Flux where the RFC graduated to general availability.
- The version of Flux where the RFC was retired or superseded.
-->

[BLAKE3]: https://github.com/BLAKE3-team/BLAKE3
[go-digest]: https://pkg.go.dev/github.com/opencontainers/go-digest#hdr-Basics