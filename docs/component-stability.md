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

#### Observability requirements

Stable components should emit enough internal telemetry to let users detect errors, as well as data
loss and performance issues inside the component, and to help diagnose them if possible.

For extension components, this means some way to monitor errors (for example through logs or span
events), and some way to monitor performance (for example through spans or histograms). Because
extensions can be so diverse, the details will be up to the component authors, and no further
constraints are set out in this document.

For pipeline components however, this section details the kinds of values that should be observable
via internal telemetry for all stable components.

> [!NOTE]
> - The following categories MUST all be covered, unless justification is given as to why one may
>   not be applicable.
> - However, for each category, many reasonable implementations are possible, as long as the
>   relevant information can be derived from the emitted telemetry; everything after the basic
>   category description is a recommendation, and is not normative.
> - Of course, a component may define additional internal telemetry which is not in this list.
> - Some of this internal telemetry may already be provided by pipeline auto-instrumentation or
>   helper modules (such as `receiverhelper`, `scraperhelper`, `processorhelper`, or
>   `exporterhelper`). Please check the documentation to verify which parts, if any, need to be
>   implemented manually.

**Definition:** In the following, an "item" refers generically to a single log record, metric point,
or span.

The internal telemetry of a stable pipeline component should allow observing the following:

1. How much data the component receives.

    For receivers, this could be a metric counting requests, received bytes, scraping attempts, etc.

    For other components, this would typically be the number of items received through the
    `Consumer` API.

2. How much data the component outputs.

    For exporters, this could be a metric counting requests, sent bytes, etc.

    For other components, this would typically be the number of items forwarded to the next
    component through the `Consumer` API.

3. How much data is dropped because of errors.

    For receivers, this could include a metric counting payloads that could not be parsed in.

    For receivers and exporters that interact with an external service, this could include a metric
    counting requests that failed because of network errors.

    For processors, this could be an `outcome` (`success` or `failure`) attribute on a "received
    items" metric defined for point 1.

    The goal is to be able to easily pinpoint the source of data loss in the Collector pipeline, so
    this should either:
    - only include errors internal to the component, or;
    - allow distinguishing said errors from ones originating in an external service, or propagated
        from downstream Collector components.

4. Details for error conditions.

    This could be in the form of logs or spans detailing the reason for an error. As much detail as
    necessary should be provided to ease debugging. Processed signal data should not be included for
    security and privacy reasons.

5. Other possible discrepancies between input and output, if any. This may include:

    - How much data is dropped as part of normal operation (eg. filtered out).

    - How much data is created by the component.

    - How much data is currently held by the component, and how much can be held if there is a fixed
        capacity.
    
        This would typically be an UpDownCounter keeping track of the size of an internal queue, along
        with a gauge exposing the queue's capacity.

6. Processing performance.

    This could include spans for each operation of the component, or a histogram of end-to-end
    component latency.
    
    The goal is to be able to easily pinpoint the source of latency in the Collector pipeline, so
    this should either:
    - only include time spent processing inside the component, or;
    - allow distinguishing this latency from that caused by an external service, or from time spent
        in downstream Collector components.
    
    As an application of this, components which hold items in a queue should allow differentiating
    between time spent processing a batch of data and time where the batch is simply waiting in the
    queue.
    
    If multiple spans are emitted for a given batch (before and after a queue for example), they
    should either belong to the same trace, or have span links between them, so that they can be
    correlated.

When measuring amounts of data, it is recommended to use "items" as your unit of measure. Where this
can't easily be done, any relevant unit may be used, as long as zero is a reliable indicator of the
absence of data. In any case, all metrics should have a defined unit (not "1").

All internal telemetry emitted by a component should have attributes identifying the specific
component instance that it originates from. This should follow the same conventions as the
[pipeline universal telemetry](rfcs/component-universal-telemetry.md).

If data can be dropped/created/held at multiple distinct points in a component's pipeline (eg.
scraping, validation, processing, etc.), it is recommended to define additional attributes to help
diagnose the specific source of the discrepancy, or to define different signals for each.

### Deprecated

The component is planned to be removed in a future version and no further support will be provided. Note that new issues will likely not be worked on. When a component enters "deprecated" mode, it is expected to exist for at least two minor releases. See the component's readme file for more details on when a component will cease to exist.

### Unmaintained

A component identified as unmaintained does not have an active code owner. Such component may have never been assigned a code owner or a previously active code owner has not responded to requests for feedback within 6 weeks of being contacted. Issues and pull requests for unmaintained components will be labelled as such. After 3 months of being unmaintained, these components will be removed from official distribution. Components that are unmaintained are actively seeking contributors to become code owners.

Components that were accepted based on being vendor-specific components will be marked as unmaintained if
they have no active code owners from the vendor even if there are other code owners listed. As part of being marked unmaintained, we'll attempt to contact the vendor to notify them of the change. Other active code
owners may petition for its continued maintenance if they want, at which point the component will no
longer be considered vendor-specific.
