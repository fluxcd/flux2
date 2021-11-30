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
- Submit an RFC by opening a pull request using [RFC-0000](RFC-0000/README.md) as template.
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
