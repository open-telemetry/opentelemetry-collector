# Collector RFCs

This folder contains accepted design documents for the Collector. Proposals here imply changes only
on the OpenTelemetry Collector and not on other parts of OpenTelemetry; if you have a cross-cutting
proposal, file an [OTEP][1] instead.

# RFC Process

## Scope

The Request For Comments (RFC) process is intended to be used for significant changes to the Collector. Major design
decisions, especially those that imply a change that is hard to reverse, should generally be
documented as an RFC. Controversial changes should also be documented as an RFC, so that the
community can have a chance to provide feedback. The goal of this process is to ensure we have a
coherent vision before embarking on significant work.

Ultimately, if any opentelemetry-collector maintainer feels that a change requires an RFC, then a
merged RFC is a requirement for said change.

## Contents

We are purposefully light on the structure of RFCs, with the focus being the decision process and
not the document itself. When in doubt, the [OTEP template][2] can be a good starting point.
Regardless of the structure, the RFC should make clear what commitments are being made by the
Collector SIG when merging it.

## Announcement

RFCs should be announced in a Collector SIG meeting and on the #otel-collector-dev Slack channel.

## Approval process

An RFC is just like any other PR in this repository, with the exception of more stringent criteria
for merging it.

### Stakeholders

To define merge criteria and voting, each RFC has a set of 'stakeholders'. All
opentelemetry-collector approvers are considered stakeholders. Additional stakeholders (e.g.
maintainers of related SIGs or experts) may be explicitly noted in the RFC.

### Merge rules

We use a [Lazy Consensus](https://www.apache.org/foundation/glossary.html#LazyConsensus) method with the following rules:

1. *Quorum*: For an RFC to be mergeable, it needs to have at least **two approvals** from the
   approvers set as well as approvals from any additional stakeholders.
2. *Waiting period*: Maintainers need to announce their intent to merge the RFC with a GitHub
   comment. They will need to add a `rfc:final-comment-period` label to the PR, comment on the PR
   and note the final comment period in the #otel-collector-dev CNCF Slack channel, and wait for at
   least **4 business days** after making the announcement to merge the RFC.
3. *Objections*: Objections should be communicated as a 'request changes' review. All objections
   must be addressed before merging an RFC. If addressing an objection does not appear feasible, any
   maintainer may call for a vote to be made on the objection (see below). This will be signified by
   adding a `rfc:vote-needed` label to the PR. The voting result is binding and a maintainer can
   drop any 'request changes' reviews based on the vote results or consensus.
4. *Modifications*: Non-trivial modifications to an RFC reset the waiting period. RFC authors must
   re-request any approving, comment, or 'request changes' reviews if the RFC has been modified significantly.
5. *All conversations are resolved*: All Github conversations on the PR must be marked as resolved
   before merging. The RFC author may resolve conversations at their discretion, but they must
   explain in the conversation thread why they believe it is appropriate to do so.

### Voting

If there is no consensus on a particular issue, a vote may be called by any of the maintainers. The
vote will be open to all stakeholders for at least **5 business days** or until at least one third
of the stakeholders have voted, whichever comes last. The vote will be decided by simple majority,
restricting the set to approvers and then maintainers in case of a tie among stakeholders.

Voting should be done on a dedicated issue via comments. The related discussion threads should be
linked in the issue. The voting result should be documented in the RFC itself.

# Modifications to existing RFCs and to this document

Non-trivial modifications to this document and to existing RFCs should be done through the RFC
process itself and have the same merge criteria.

[1]: https://github.com/open-telemetry/oteps
[2]: https://github.com/open-telemetry/oteps/blob/main/0000-template.md
