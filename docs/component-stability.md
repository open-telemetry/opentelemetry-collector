# Stability Levels and versioning

## Stability levels

The collector components and implementation are in different stages of stability, and usually split between
functionality and configuration. The status for each component is available in the README file for the component. While
we intend to provide high-quality components as part of this repository, we acknowledge that not all of them are ready
for prime time. As such, each component should list its current stability level for each telemetry signal, according to
the following definitions:

### Development

Not all pieces of the component are in place yet and it might not be available as part of any distributions yet. Bugs and performance issues should be reported, but it is likely that the component owners might not give them much attention. Your feedback is still desired, especially when it comes to the user-experience (configuration options, component observability, technical implementation details, ...). Configuration options might break often depending on how things evolve. The component should not be used in production.

### Alpha

The component is ready to be used for limited non-critical workloads and the authors of this component would welcome your feedback. Bugs and performance problems should be reported, but component owners might not work on them right away.

#### Configuration changes

Configuration for alpha components can be changed with minimal notice. Documenting them as part of the changelog is
sufficient. We still recommend giving users one or two minor versions' notice before breaking the configuration, such as
when removing or renaming a configuration option. Providing a migration path in the component's repository is NOT
required for alpha components, although it is still recommended.

- when adding a new configuration option, components MAY mark the new option as required and are not required to provide
  a reasonable default.
- when renaming a configuration option, components MAY treat the old name as an alias to the new one and log a WARN
  level message in case the old option is being used.
- when removing a configuration option, components MAY keep the old option for a few minor releases and log a WARN level
  message instructing users to remove the option.


### Beta

Same as Alpha, but the configuration options are deemed stable. While there might be breaking changes between releases, component owners should try to minimize them. A component at this stage is expected to have had exposure to non-critical production workloads already during its **Alpha** phase, making it suitable for broader usage.

#### Configuration changes

Backward incompatible changes should be rare events for beta components. Users of those components are not expecting to
have their Collector instances failing at startup because of a configuration change. When doing backward incompatible
changes, component owners should add the migration path to a place within the component's repository, linked from the
component's main README. This is to ensure that people using older instructions can understand how to migrate to the
latest version of the component.

When adding a new required option:
- the option MUST come with a sensible default value

When renaming or removing a configuration option:
- the option MUST be deprecated in one version
- a WARN level message should be logged, with a link to a place within the component's repository where the change is
  documented and a migration path is provided
- the option MUST be kept for at least N+1 version and MAY be hidden behind a feature gate in N+2
- the option and the WARN level message MUST NOT be removed earlier than N+2 or 6 months, whichever comes later

Additionally, when removing an option:
- the option MAY be made non-operational already by the same version where it is deprecated

### Stable

The component is ready for general availability. Bugs and performance problems should be reported and there's an expectation that the component owners will work on them. Breaking changes, including configuration options and the component's output are not expected to happen without prior notice, unless under special circumstances.

#### Configuration changes

Stable components MUST be compatible between minor versions unless critical security issues are found. In that case, the
component owner MUST provide a migration path and a reasonable time frame for users to upgrade. The same rules from beta
components apply to stable when it comes to configuration changes.

### Deprecated

The component is planned to be removed in a future version and no further support will be provided. Note that new issues will likely not be worked on. When a component enters "deprecated" mode, it is expected to exist for at least two minor releases. See the component's readme file for more details on when a component will cease to exist.

### Unmaintained

A component identified as unmaintained does not have an active code owner. Such component may have never been assigned a code owner or a previously active code owner has not responded to requests for feedback within 6 weeks of being contacted. Issues and pull requests for unmaintained components will be labelled as such. After 3 months of being unmaintained, these components will be removed from official distribution. Components that are unmaintained are actively seeking contributors to become code owners.

Components that were accepted based on being vendor-specific components will be marked as unmaintained if
they have no active code owners from the vendor even if there are other code owners listed. As part of being marked unmaintained, we'll attempt to contact the vendor to notify them of the change. Other active code
owners may petition for its continued maintenance if they want, at which point the component will no
longer be considered vendor-specific.

## Graduating between stability levels

Components can graduate between stability levels. The process for doing so is as follows:

1. One of the component owners should file an issue with the 'Graduation' issue template to request
   the graduation.
2. An approver is assigned in a rotating basis to evaluate the request and provide feedback. For
   vendor specific components, the approver should be from a different employer to the one owning
   the component.
3. If approved, a PR to change the stability level should be opened and MUST be approved by all
   listed code owners.

## Graduation criteria

In addition to the requirements outlined above, additional criteria should be met before a component
can graduate to a higher stability level. These ensure that the component is ready for the increased
usage and scrutiny that comes with a higher stability level, and that the community around it is
sufficiently healthy.

If the graduation criteria are not met, the approver should provide feedback on what is missing and
how to address it. The component owners can then address the feedback and re-request graduation on
the same issue.

## In development to alpha

No additional criteria are required to graduate from development to alpha.
The component still needs to meet the general requirements for alpha components.

## Alpha to beta

To graduate any signal from alpha to beta on a component:
1. The component MUST have at least two active code owners.
2. The code owners for non-vendor-specific components SHOULD have at least two different employers.
3. Within the 30 days prior to the graduation request, the code owners MUST have reviewed and
   replied to at least 80% of the issues and pull requests opened against the component. This
   excludes general PRs or issues that are not specific to the component itself (e.g. repo-wide API
   updates). It is not necessary that the issues and PRs are closed or merged, but that they have
   been reviewed and replied to appropriately.

## Beta to stable

To graduate any signal from beta to stable on a component:
1. The component MUST have at least three active code owners.
2. The code owners for non-vendor-specific components SHOULD have at least two different employers.
3. Within the 60 days prior to the graduation request, the code owners MUST have reviewed and
   replied to at least 80% of the issues and pull requests opened against the component. This
   excludes general PRs or issues that are not specific to the component itself (e.g. repo-wide API
   updates). It is not necessary that the issues and PRs are closed or merged, but that they have
   been reviewed and replied to appropriately.
