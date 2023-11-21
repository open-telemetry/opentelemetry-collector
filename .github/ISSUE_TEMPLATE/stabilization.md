---
name: Module stabilization
about: Stabilize a module before a 1.0 release
title: 'Stabilize module X'
labels: 'stabilization'
assignees: ''
---

Before stabilizing a module, an approver or maintainer must make sure that the following criteria are met:

- [ ] One RC release or more have been done of this module
- [ ] No open issues or PRs in the module that would require breaking changes
- [ ] No TODOs in the module code that would require breaking changes
- [ ] No deprecated symbols in the module
- [ ] No symbols marked as experimental in the module
- [ ] The module follows the [Coding guidelines](https://github.com/open-telemetry/opentelemetry-collector/blob/main/CONTRIBUTING.md)

Please also make sure to publicly announce our intent to stabilize the module on:

- [ ] The #otel-collector CNCF Slack Channel
- [ ] The #opentelemetry CNCF Slack channel
- [ ] A Collector SIG meeting (if unable to attend, just add to the agenda)

To help other people verify the above criteria, please link to an RC release, the announcement and other links used to complete the above in a comment on this issue.
