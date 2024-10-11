# Collector RFCs

This folder contains accepted design documents for the Collector. 
Proposals here imply changes only on the OpenTelemetry Collector and not on other parts of OpenTelemetry; if you have a cross-cutting proposal, file an [OTEP][1] instead.

# RFC Process

## Scope

The RFC process is intended to be used for significant changes to the Collector. Major design decisions, specially those that imply a change that is hard to reverse, should probably be documented as an RFC. Controversial changes should also be documented as an RFC, so that the community can have a chance to provide feedback.

Ultimately, if any Collector maintainer feels that a change requires an RFC, then it should be written. The goal of this process is to ensure changes are reviewed and we have a coherent vision before embarking on significant work.

## Contents

We are purposefully light on the structure of RFCs, with the focus being the decision process and not the document itself. If in doubt, the [OTEP template][2] can be a good starting point. In any case, it should be clear what the Collector SIG is commiting to by accepting the RFC.

## Announcement

RFCs should be announced widely including in the Collector SIG Slack channels and meetings.

## Approval process

An RFC is just like any other PR in this repository, with the exception of more stringent criteria for merging it.

### Stakeholders

As with any other change, everybody is welcome to participate in the discussion of an RFC. Each RFC author should define a set of 'stakeholders', which should contain, at a minimum, the Collector approvers, but may contain additional people (e.g. maintainers of related SIGs or experts). The stakeholders are the minimum set that will take the final decision on the RFC.

### Merge rules

We use a Lazy Consensus method with the following rules:

1. *Quorum*: For an RFC to be mergeable, it needs to have at least **two approvals** from the stakeholder set.
2. *Waiting period*: After requirements to merge an RFC have been met, a maintainer may announce their intent to merge an RFC after at least **4 business days** have passed.
3. *Objections*: Objections should be represented as a 'request changes' review. We should strive to address all objections before merging an RFC. If unable to reach consensus after a waiting period, any maintainer may call for a vote to be made (see below).
4. *Modifications*: Non-trivial modifications to an RFC reset the waiting period.

### Voting

If there is no consensus on a particular issue, a vote may be called by any of the maintainers. The vote will be open to all stakeholders for at least **5 business days** or until at least one third of the stakeholders have voted, whichever comes last. The 'do not merge' option must be available, and the vote will be decided by simple majority.

Voting may be done through any means that ensure all stakeholders can vote. The voting result should be documented in the RFC itself.

# Modifications to this document

Non-trivial modifications to this document should be done through the RFC process itself and have the same merge criteria.

[1]: https://github.com/open-telemetry/oteps
[2]: https://github.com/open-telemetry/oteps/blob/main/0000-template.md
