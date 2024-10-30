# OpenTelemetry Collector Releases approvers

## Overview 

I propose we create a new `collector-releases-approvers` Github team that are approvers on the
opentelemetry-collector-releases repository and code owners for the Builder and release workflows.
The existing approvers teams will focus on the Collector and Collector components codebases. This
opens future possibilities for creating a separate WG/SIG for this and further improving our release
process.

## Team scope and responsibilities

The `collector-releases-approvers` team will have the following responsibilities:

- Approvers for the opentelemetry-collector-releases repository.
- Code owners for the OpenTelemetry Collector Builder.
- Code owners for the release workflows for the opentelemetry-collector and
  opentelemetry-collector-contrib repositories.

## Rationale

Release engineering requires a different set of skills and interests than developing the Collector
codebase. As such, the set of contributors for the Collector releases has overlap with but is
different from the set of contributors for the Collector codebase. We are missing out on retaining
people who are more interested in these aspects. We can see this with the examples of the
OpenTelemetry Operator repository and the OpenTelemetry Collector Helm Chart repository; they are
able to work independently from the other Collector repositories and have been able to create an
independent community. 

We also have a [growing backlog][1] of issues related to the release process that would benefit from
a dedicated set of people.

## Initial team members

Initially, all members of the `collector-contrib-approvers` team will be part of the
`collector-releases-approvers` team. The Collector maintainers will reach out to existing
contributors to the Collector releases to invite them to join the team.

## Future work

If we are able to grow a healthy community around the releases repository, in the future we can
consider the following:

- Having a dedicated SIG/WG for the opentelemetry-collector-releases repository.
- Having dedicated meetings for release retros that allow us to iteratively improve the Collector
  release process.
- Splitting off the artifact and container release process to be independent from the source code
  release from opentelemetry-collector and opentelemetry-collector-contrib.

This proposal does not require us to do any of the above, but they are interesting possibilities for
the future.

[1]: https://github.com/search?q=org%3Aopen-telemetry+label%3Arelease-retro++&type=issues&state=open
