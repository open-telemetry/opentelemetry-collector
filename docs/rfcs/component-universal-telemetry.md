# Pipeline Component Telemetry

## Motivation and Scope

The collector should be observable and this must naturally include observability of its pipeline components. Pipeline components
are those components of the collector which directly interact with data, specifically receivers, processors, exporters, and connectors.

It is understood that each _type_ (`filelog`, `batch`, etc) of component may emit telemetry describing its internal workings,
and that these internally derived signals may vary greatly based on the concerns and maturity of each component. Naturally
though, there is much we can do to normalize the telemetry emitted from and about pipeline components.

Two major challenges in pursuit of broadly normalized telemetry are (1) consistent attributes, and (2) automatic capture.

This RFC represents an evolving consensus about the desired end state of component telemetry. It does _not_ claim
to describe the final state of all component telemetry, but rather seeks to document some specific aspects. It proposes a set of
attributes which are both necessary and sufficient to identify components and their instances. It also articulates one specific
mechanism by which some telemetry can be automatically captured. Finally, it describes some specific metrics and logs which should
be automatically captured for each kind of pipeline component.

## Goals

1. Define attributes that are (A) specific enough to describe individual component[_instances_](https://github.com/open-telemetry/opentelemetry-collector/issues/10534)
   and (B) consistent enough for correlation across signals.
2. Articulate a mechanism which enables us to _automatically_ capture telemetry from _all pipeline components_.
3. Define specific metrics for each kind of pipeline component.
4. Define specific logs for all kinds of pipeline component.

## Attributes

All signals should use the following attributes:

### Receivers

- `otel.component.kind`: `receiver`
- `otel.component.id`: The component ID
- `otel.signal`: `logs`, `metrics`, `traces`

### Processors

- `otel.component.kind`: `processor`
- `otel.component.id`: The component ID
- `otel.pipeline.id`: The pipeline ID
- `otel.signal`: `logs`, `metrics`, `traces`

### Exporters

- `otel.component.kind`: `exporter`
- `otel.component.id`: The component ID
- `otel.signal`: `logs`, `metrics` `traces`

### Connectors

- `otel.component.kind`: `connector`
- `otel.component.id`: The component ID
- `otel.signal`: `logs`, `metrics` `traces`
- `otel.output.signal`: `logs`, `metrics` `traces`

Note: The `otel.signal`, `otel.output.signal`, or `otel.pipeline.id` attributes may be omitted if the corresponding component instances
are unified by the component implementation. For example, the `otlp` receiver is a singleton, so its telemetry is not specific to a signal.
Similarly, the `memory_limiter` processor is a singleton, so its telemetry is not specific to a pipeline.

## Auto-Instrumentation Mechanism

The mechanism of telemetry capture should be _external_ to components. Specifically, we should observe telemetry at each point where a
component passes data to another component, and, at each point where a component consumes data from another component. In terms of the
component graph, every _edge_ in the graph will have two layers of instrumentation - one for the producing component and one for the
consuming component. Importantly, each layer generates telemetry ascribed to a single component instance, so by having two layers per
edge we can describe both sides of each handoff independently.

Telemetry captured by this mechanism should be associated with an instrumentation scope corresponding to the package which implements
the mechanism. Currently, that package is `service/internal/graph`, but this may change in the future. Notably, this telemetry is not
ascribed to individual component packages, both because the instrumentation scope is intended to describe the origin of the telemetry,
and because no mechanism is presently identified which would allow us to determine the characteristics of a component-specific scope.

### Auto-Instrumented Metrics

There are two straightforward measurements that can be made on any pdata:

1. A count of "items" (spans, data points, or log records). These are low cost but broadly useful, so they should be enabled by default.
2. A measure of size, based on [ProtoMarshaler.Sizer()](https://github.com/open-telemetry/opentelemetry-collector/blob/9907ba50df0d5853c34d2962cf21da42e15a560d/pdata/ptrace/pb.go#L11).
  These are high cost to compute, so by default they should be disabled (and not calculated).

The location of these measurements can be described in terms of whether the data is "incoming" or "outgoing", from the perspective of the
component to which the telemetry is ascribed.

1. Incoming measurements are attributed to the component which is _consuming_ the data.
2. Outgoing measurements are attributed to the component which is _producing_ the data.

For both metrics, an `outcome` attribute with possible values `success` and `failure` should be automatically recorded, corresponding to
whether or not the corresponding function call returned an error. Specifically, incoming measurements will be recorded with `outcome` as
`failure` when a call from the previous component the `ConsumeX` function returns an error, and `success` otherwise. Likewise, outgoing
measurements will be recorded with `outcome` as `failure` when a call to the next consumer's `ConsumeX` function returns an error, and
`success` otherwise.

```yaml
    otelcol_component_incoming_items:
      enabled: true
      description: Number of items passed to the component.
      unit: "{items}"
      sum:
        value_type: int
        monotonic: true
    otelcol_component_outgoing_items:
      enabled: true
      description: Number of items emitted from the component.
      unit: "{items}"
      sum:
        value_type: int
        monotonic: true

    otelcol_component_incoming_size:
      enabled: false
      description: Size of items passed to the component.
      unit: "By"
      sum:
        value_type: int
        monotonic: true
    otelcol_component_outgoing_size:
      enabled: false
      description: Size of items emitted from the component.
      unit: "By"
      sum:
        value_type: int
        monotonic: true
```

### Auto-Instrumented Logs

Metrics provide most of the observability we need but there are some gaps which logs can fill. Although metrics would describe the overall
item counts, it is helpful in some cases to record more granular events. e.g. If an outgoing batch of 10,000 spans results in an error, but
100 batches of 100 spans succeed, this may be a matter of batch size that can be detected by analyzing logs, while the corresponding metric
reports only that a 50% success rate is observed.

For security and performance reasons, it would not be appropriate to log the contents of telemetry.

It's very easy for logs to become too noisy. Even if errors are occurring frequently in the data pipeline, they may only be of interest to
many users if they are not handled automatically.

With the above considerations, this proposal includes only that we add a DEBUG log for each individual outcome. This should be sufficient for
detailed troubleshooting but does not impact users otherwise.

In the future, it may be helpful to define triggers for reporting repeated failures at a higher severity level. e.g. N number of failures in
a row, or a moving average success %. For now, the criteria and necessary configurability is unclear so this is mentioned only as an example
of future possibilities.

### Auto-Instrumented Spans

It is not clear that any spans can be captured automatically with the proposed mechanism. We have the ability to insert instrumentation both
before and after processors and connectors. However, we generally cannot assume a 1:1 relationship between incoming and outgoing data.

## Additional Context

This proposal pulls from a number of issues and PRs:

- [Demonstrate graph-based metrics](https://github.com/open-telemetry/opentelemetry-collector/pull/11311)
- [Attributes for component instancing](https://github.com/open-telemetry/opentelemetry-collector/issues/11179)
- [Simple processor metrics](https://github.com/open-telemetry/opentelemetry-collector/issues/10708)
- [Component instancing is complicated](https://github.com/open-telemetry/opentelemetry-collector/issues/10534)
