---
name: Module stabilization
about: Stabilize a module before a 1.0 release
title: 'Stabilize module X'
labels: 'stabilization'
assignees: ''
---

Before stabilizing a module, an approver or maintainer must make sure that the following criteria have been met for at least two successive minor version releases (regardless of when this issue was opened):

- [ ] No open issues or PRs in the module that would require breaking changes
- [ ] No TODOs in the module code that would require breaking changes
- [ ] No deprecated symbols in the module
- [ ] No symbols marked as experimental in the module
- [ ] The module follows the [Coding guidelines](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/coding-guidelines.md)

Please also make sure to publicly announce our intent to stabilize the module on:

- [ ] The #otel-collector CNCF Slack Channel
- [ ] The #opentelemetry CNCF Slack channel
- [ ] A Collector SIG meeting (if unable to attend, just add to the agenda)

To help other people verify the above criteria, please link to the announcement and other links used to complete the above in a comment on this issue.

Once all criteria are met, close this issue by moving this module to the `stable` module set.

<sub>**Tip**: [React](https://github.blog/news-insights/product-news/add-reactions-to-pull-requests-issues-and-comments/) with 👍 to help prioritize this issue. Please use comments to provide useful context, avoiding `+1` or `me too`, to help us triage it. Learn more [here](https://opentelemetry.io/community/end-user/issue-participation/).</sub>
