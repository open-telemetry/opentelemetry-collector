---
name: Component Graduation
about: Graduate a component from beta to stable
title: 'Graduate component X to stable'
labels: 'graduation'
assignees: ''
---

This issue requests the graduation of a component to stable. Please review the [Component Graduation to Stable](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md#component-graduation-to-stable) documentation for full details.

## Component Information

- **Component name**: <!-- e.g., otlpreceiver -->
- **Component type**: <!-- receiver/processor/exporter/connector/extension -->
- **Repository**: <!-- e.g., opentelemetry-collector or opentelemetry-collector-contrib -->

## Signal Requirements

- [ ] All supported signals are at beta stability or higher
- [ ] At least one signal is at stable stability

## Code Owner Requirements

- [ ] The component has at least three active code owners
- [ ] Within the 60 days prior to this request, the code owners have reviewed and replied to at least 80% of the issues and pull requests opened against the component

List the current code owners:
1. <!-- @username -->
2. <!-- @username -->
3. <!-- @username -->

## Technical Requirements

- [ ] The component meets all [testing requirements](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md#testing-requirements) for stable components
- [ ] The component meets all [documentation requirements](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md#documentation-requirements) for stable components
- [ ] The component meets all [observability requirements](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md#observability-requirements) for stable components
- [ ] The component follows the [coding guidelines](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/coding-guidelines.md), including naming conventions

Please provide links to evidence:
- **Test coverage report**: <!-- link -->
- **Benchmark results**: <!-- link -->
- **Documentation**: <!-- link to component README -->

## Adoption Evidence

The component must have evidence of real-world adoption. Provide at least one of the following:

### Option 1: Public Adopter Testimonials
At least two organizations have publicly stated they use the component in production.

- [ ] Adopter 1: <!-- Link to blog post, conference talk, GitHub issue, vendor documentation or other public statement -->
- [ ] Adopter 2: <!-- Link to blog post, conference talk, GitHub issue, vendor documentation or other public statement -->

### Option 2: Private Attestation
If adopters cannot be named publicly, provide private attestation to the assigned maintainer.

- [ ] Private attestation provided to maintainer

The attestation must include a general description of the scale of usage (e.g., "processing millions of spans per day").

---

## For Maintainers

A maintainer will be assigned on a rotating basis to verify this graduation request.

- [ ] Maintainer assigned: <!-- @username -->
- [ ] All requirements verified
- [ ] Adoption evidence verified as credible

Once verified, the code owners should open a PR to update the component's stability level.

<sub>**Tip**: [React](https://github.blog/news-insights/product-news/add-reactions-to-pull-requests-issues-and-comments/) with 👍 to help prioritize this issue. Please use comments to provide useful context, avoiding `+1` or `me too`, to help us triage it. Learn more [here](https://opentelemetry.io/community/end-user/issue-participation/).</sub>
