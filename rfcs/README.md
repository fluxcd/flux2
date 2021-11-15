# Flux RFCs

In many cases, new features and enhancements are proposed on [flux2/discussions](https://github.com/fluxcd/flux2/discussions).
A proposal is discussed in public by maintainers, contributors, users and other interested parties.
After some form of consensus is reached between participants, the proposed changes go through the
pull request process where the implementation details are reviewed, approved or rejected by maintainers.

Some proposals may be **substantial**, and for these we ask for a design process to be followed
so that all stakeholders can be confident about the direction Flux is evolving in.

The "RFC" (request for comments) process is intended to provide a consistent and
controlled path for substantial changes to enter Flux.

Examples of substantial changes:

- API additions (new kinds of resources, new relationships between existing APIs)
- API breaking changes (new required fields, field removals)
- Security related changes (Flux controllers permissions, tenant isolation and impersonation)
- Impactful UX changes (new required inputs to the bootstrap process)
- Drop capabilities (sunset an existing integration with an external service due to security concerns)

## RFC Template

```text
# RFC-NNNN Title

<!--
The title must be short and descriptive.
-->

## Summary

<!--
One paragraph explanation of the proposed feature or enhancement.
-->

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this RFC. Describe why the change is important and the benefits to users.
-->

### Goals

<!--
List the specific goals of this RFC. What is it trying to achieve? How will we
know that this has succeeded?
-->

### Non-Goals

<!--
What is out of scope for this RFC? Listing non-goals helps to focus discussion
and make progress.
-->

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation.
-->

### User Stories

<!--
Optional if existing discussions and/or issues are linked in the motivation section.
-->

### Alternatives

<!--
List plausible alternatives to the proposal and
motivate why the current proposal is superior.
-->

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs and code snippets.

The design details should answer the following questions:
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
```
