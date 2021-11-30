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

## RFC Process

- Before submitting an RFC please discuss the proposal with the Flux community.
  Start a discussion on GitHub and ask for feedback at the weekly dev meeting.
  You must find a maintainer willing to sponsor the RFC.
- Submit an RFC by opening a pull request using RFC-0000 as template.
- The sponsor will assign the PR to themselves, will label the PR with `area/RFC` and
  will request other maintainers to begin the review process.
- Integrate feedback by adding commits without overriding the history.
- At least two maintainers have to approve the proposal before it can be merged.
  Approvers must be satisfied that an
  [appropriate level of consensus](https://github.com/fluxcd/community/blob/main/GOVERNANCE.md#decision-guidelines)
  has been reached.
- Before the merge, an RFC number is assigned by the sponsor and the PR branch must be rebased with main.
- Once merged, the proposal may be implemented in Flux.
  The progress could be tracked using the RFC number (used as prefix for issues and PRs).
- After the proposal implementation is available in a release candidate or final release,
  the RFC should be updated with the Flux version added to the "Implementation History" section.
- During the implementation phase, the RFC could be discarded due to security or performance concerns.
  In this case, the RFC "Implementation History" should state the rejection motives.
  Ultimately the decision on the feasibility of a particular implementation,
  resides with the maintainers that reviewed the code changes.
- A new RFC could be summited with the scope of replacing an RFC rejected during implementation.
  The new RFC must come with a solution for the rejection motives of the previous RFC.

## RFC Template

```text
# RFC-NNNN Title

<!--
The title must be short and descriptive.
-->

**Status:** provisional

<!--
Status represents the current state of the RFC.
Must be one of `provisional`, `implementable`, `implemented`, `deferred`, `rejected`, `withdrawn`, or `replaced`.
-->

**Creation Date:** YYYY-MM-DD

**Last update:** YYYY-MM-DD

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
```
