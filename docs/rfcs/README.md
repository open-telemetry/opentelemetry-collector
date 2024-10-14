# Collector RFCs

This folder contains accepted design documents for the Collector. Proposals here imply changes only
on the OpenTelemetry Collector and not on other parts of OpenTelemetry; if you have a cross-cutting
proposal, file an [OTEP][1] instead.

# RFC Process

## Scope

The RFC process is intended to be used for significant changes to the Collector. Major design
decisions, specially those that imply a change that is hard to reverse, should probably be
documented as an RFC. Controversial changes should also be documented as an RFC, so that the
community can have a chance to provide feedback. The goal of this process is to ensure we have a
coherent vision before embarking on significant work.

Ultimately, if any Collector maintainer feels that a change requires an RFC, then a merged RFC is a
requirement for said change.

## Contents

We are purposefully light on the structure of RFCs, with the focus being the decision process and
not the document itself. When in doubt, the [OTEP template][2] can be a good starting point.
Regardless of the structure, the RFC should make clear what commitments are being made by the
Collector SIG when merging it.

## Announcement

RFCs should be announced widely including in the Collector SIG Slack channels and the Collector SIG
meetings.

## Approval process

An RFC is just like any other PR in this repository, with the exception of more stringent criteria
for merging it.

### Stakeholders

As with any other change, everybody is welcome to participate in the discussion of an RFC. To define
merge criteria and voting, Each RFC has a set of 'stakeholders', which should contain, at a minimum,
the Collector approvers, but may contain additional people (e.g. maintainers of related SIGs or
experts). If the stakeholder set is not defined explicitly by the RFC author, the approvers set will
be assumed.

### Merge rules

We use a Lazy Consensus method with the following rules:

1. *Quorum*: For an RFC to be mergeable, it needs to have at least **two approvals** from the
   stakeholder set.
2. *Waiting period*: Maintainers need to announce their intent to merge the RFC with a Github
   comment, and wait for at least **4 business days** after making the announcement.
3. *Objections*: Objections should be communicated as a 'request changes' review. All objections
   must be addressed or voted before merging an RFC. If unable to resolve an objection, any
   maintainer may call for a vote to be made on the objection (see below).
4. *Modifications*: Non-trivial modifications to an RFC reset the waiting period.

### Voting

If there is no consensus on a particular issue, a vote may be called by any of the maintainers. The
vote will be open to all stakeholders for at least **5 business days** or until at least one third
of the stakeholders have voted, whichever comes last. The vote will be decided by simple majority.

Voting may be done through any means that ensure all stakeholders can vote and that the result is
viewable by all stakeholders. The voting result should be documented in the RFC itself.

# Modifications to this document

Non-trivial modifications to this document should be done through the RFC process itself and have
the same merge criteria.

[1]: https://github.com/open-telemetry/oteps
[2]: https://github.com/open-telemetry/oteps/blob/main/0000-template.md
