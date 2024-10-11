# Auto-Instrumented Component Telemetry

## Motivation

The collector should be observable and this must naturally include observability of its pipeline components. It is understood that each _type_ (`filelog`, `batch`, etc) of component may emit telemetry describing its internal workings, and that these internally derived signals may vary greatly based on the concerns and maturity of each component. Naturally though, the collector should also describe the behavior of components using broadly normalized telemetry. A major challenge in pursuit is that there must be a clear mechanism by which such telemetry can be automatically captured. Therefore, this RFC is first and foremost a proposal for a _mechanism_. Then, based on what _can_ be captured by this mechanism, the RFC describes specific metrics and logs which can be broadly normalized.

## Goals

1. Articulate a mechanism which enables us to _automatically_ capture telemetry from _all pipeline components_.
2. Define attributes that are (A) specific enough to describe individual component [_instances_](https://github.com/open-telemetry/opentelemetry-collector/issues/10534) and (B) consistent enough for correlation across signals.
3. Define specific metrics for each kind of pipeline component.
4. Define specific logs for all kinds of pipeline component.

### Mechanism

The mechanism of telemetry capture should be _external_ to components. Specifically, we should observe telemetry at each point where a component passes data to another component, and, at each point where a component consumes data from another component. In terms of the component graph, every _edge_ in the graph will have two layers of instrumentation - one for the producing component and one for the consuming component. Importantly, each layer generates telemetry ascribed to a single component instance, so by having two layers per edge we can describe both sides of each handoff independently.

### Attributes

All signals should use the following attributes:

#### Receivers

- `otel.component.kind`: `receiver`
- `otel.component.id`: The component ID
- `otel.signal`: `logs`, `metrics`, `traces`

#### Processors

- `otel.component.kind`: `processor`
- `otel.component.id`: The component ID
- `otel.pipeline.id`: The pipeline ID
- `otel.signal`: `logs`, `metrics`, `traces`

#### Exporters

- `otel.component.kind`: `exporter`
- `otel.component.id`: The component ID
- `otel.signal`: `logs`, `metrics` `traces`

#### Connectors

- `otel.component.kind`: `connector`
- `otel.component.id`: The component ID
- `otel.signal`: `logs`, `metrics` `traces`
- `otel.output.signal`: `logs`, `metrics` `traces`

Note: The `otel.signal`, `otel.output.signal`, or `otel.pipeline.id` attributes may be omitted if the corresponding component instances are unified by the component implementation. For example, the `otlp` receiver is a singleton, so its telemetry is not specific to a signal. Similarly, the `memory_limiter` processor is a singleton, so its telemetry is not specific to a pipeline.

### Metrics

There are two straightforward measurements that can be made on any pdata:

1. A count of "items" (spans, data points, or log records). These are low cost but broadly useful, so they should be enabled by default.
2. A measure of size, based on [ProtoMarshaler.Sizer()](https://github.com/open-telemetry/opentelemetry-collector/blob/9907ba50df0d5853c34d2962cf21da42e15a560d/pdata/ptrace/pb.go#L11). These are high cost to compute, so by default they should be disabled (and not calculated).

The location of these measurements can be described in terms of whether the data is "incoming" or "outgoing", from the perspective of the component to which the telemetry is ascribed.

1. Incoming measurements are attributed to the component which is _consuming_ the data.
2. Outgoing measurements are attributed to the component which is _producing_ the data.

For both metrics, an `outcome` attribute with possible values `success` and `failure` should be automatically recorded, corresponding to whether or not the corresponding function call returned an error. Specifically, incoming measurements will be recorded with `outcome` as `failure` when a call from the previous component the `ConsumeX` function returns an error, and `success` otherwise. Likewise, outgoing measurements will be recorded with `outcome` as `failure` when a call to the next consumer's `ConsumeX` function returns an error, and `success` otherwise.

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

### Logs

Metrics provide most of the observability we need but there are some gaps which logs can fill. Although metrics would describe the overall item counts, it is helpful in some cases to record more granular events. e.g. If an outgoing batch of 10,000 spans results in an error, but 100 batches of 100 spans succeed, this may be a matter of batch size that can be detected by analyzing logs, while the corresponding metric reports only that a 50% success rate is observed.

For security and performance reasons, it would not be appropriate to log the contents of telemetry.

It's very easy for logs to become too noisy. Even if errors are occurring frequently in the data pipeline, they may only be of interest to many users if they are not handled automatically.

With the above considerations, this proposal includes only that we add a DEBUG log for each individual outcome. This should be sufficient for detailed troubleshooting but does not impact users otherwise.

In the future, it may be helpful to define triggers for reporting repeated failures at a higher severity level. e.g. N number of failures in a row, or a moving average success %. For now, the criteria and necessary configurability is unclear so this is mentioned only as an example of future possibilities.

### Spans

It is not clear that any spans can be captured automatically with the proposed mechanism. We have the ability to insert instrumentation both before and after processors and connectors. However, we generally cannot assume a 1:1 relationship between incoming and outgoing data.

### Additional context

This proposal pulls from a number of issues and PRs:

- [Demonstrate graph-based metrics](https://github.com/open-telemetry/opentelemetry-collector/pull/11311)
- [Attributes for component instancing](https://github.com/open-telemetry/opentelemetry-collector/issues/11179)
- [Simple processor metrics](https://github.com/open-telemetry/opentelemetry-collector/issues/10708)
- [Component instancing is complicated](https://github.com/open-telemetry/opentelemetry-collector/issues/10534)
