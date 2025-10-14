# Pipeline Component Telemetry

## Motivation and Scope

The collector should be observable and this must naturally include observability of its pipeline components. Pipeline components
are those components of the collector which directly interact with data, specifically receivers, processors, exporters, and connectors.

It is understood that each _type_ (`filelog`, `otlp`, etc) of component may emit telemetry describing its internal workings,
and that these internally derived signals may vary greatly based on the concerns and maturity of each component. Naturally
though, there is much we can do to normalize the telemetry emitted from and about pipeline components.

Two major challenges in pursuit of broadly normalized telemetry are (1) consistent attributes, and (2) automatic capture.

This RFC represents an evolving consensus about the desired end state of component telemetry. It does _not_ claim
to describe the final state of all component telemetry, but rather seeks to document some specific aspects. It proposes a set of
attributes which are both necessary and sufficient to identify components and their instances. It also articulates one specific
mechanism by which some telemetry can be automatically captured. Finally, it describes some specific metrics and logs which should
be automatically captured for each kind of pipeline component.

## Goals

1. Define attributes that are (A) specific enough to describe individual component [_instances_](https://github.com/open-telemetry/opentelemetry-collector/issues/10534)
   and (B) consistent enough for correlation across signals.
2. Articulate a mechanism which enables us to _automatically_ capture telemetry from _all pipeline components_.
3. Define specific metrics for each kind of pipeline component.
4. Define specific logs for all kinds of pipeline component.

## Attributes

Traces, logs, and metrics should carry the following instrumentation scope attributes:

### Receivers

- `otelcol.component.kind`: `receiver`
- `otelcol.component.id`: The component ID
- `otelcol.signal`: `logs`, `metrics`, `traces`, `profiles`

### Processors

- `otelcol.component.kind`: `processor`
- `otelcol.component.id`: The component ID
- `otelcol.pipeline.id`: The pipeline ID
- `otelcol.signal`: `logs`, `metrics`, `traces`, `profiles`

### Exporters

- `otelcol.component.kind`: `exporter`
- `otelcol.component.id`: The component ID
- `otelcol.signal`: `logs`, `metrics`, `traces`, `profiles`

### Connectors

- `otelcol.component.kind`: `connector`
- `otelcol.component.id`: The component ID
- `otelcol.signal`: `logs`, `metrics` `traces`
- `otelcol.signal.output`: `logs`, `metrics`, `traces`, `profiles`

Note: The `otelcol.signal`, `otelcol.signal.output`, or `otelcol.pipeline.id` attributes may be omitted if the corresponding component instances
are unified by the component implementation. For example, the `otlp` receiver is a singleton, so its telemetry is not specific to a signal.
Similarly, the `memory_limiter` processor is a singleton, so its telemetry is not specific to a pipeline.

These instrumentation scope attributes are automatically injected into the telemetry associated with a component, by wrapping the Logger, TracerProvider, and MeterProvider provided to it.

## Auto-Instrumentation Mechanism

The mechanism of telemetry capture should be _external_ to components. Specifically, we should observe telemetry at each point where a
component passes data to another component, and, at each point where a component consumes data from another component. In terms of the
component graph, every _edge_ in the graph will have two layers of instrumentation - one for the producing component and one for the
consuming component. Importantly, each layer generates telemetry ascribed to a single component instance, so by having two layers per
edge we can describe both sides of each handoff independently.

Telemetry captured by this mechanism should be associated with an instrumentation scope with a name corresponding to the package which implements
the mechanism. Currently, that package is `go.opentelemetry.io/collector/service`, but this may change in the future. Notably, this telemetry is not
ascribed to individual component packages, both because the instrumentation scope is intended to describe the origin of the telemetry,
and because no mechanism is presently identified which would allow us to determine the characteristics of a component-specific scope.

### Instrumentation Scope

All telemetry described in this RFC should include a scope name which corresponds to the package which implements the telemetry. If the
package is internal, then the scope name should be that of the module which contains the package. For example,
`go.opentelemetry.io/service` should be used instead of `go.opentelemetry.io/service/internal/graph`.

### Auto-Instrumented Metrics

There are two straightforward measurements that can be made on any pdata:

1. A count of "items" (spans, data points, or log records). These are low cost but broadly useful, so they should be enabled by default.
2. A measure of size, based on [ProtoMarshaler.Sizer()](https://github.com/open-telemetry/opentelemetry-collector/blob/9907ba50df0d5853c34d2962cf21da42e15a560d/pdata/ptrace/pb.go#l11).
  These may be high cost to compute, so by default they should be disabled (and not calculated). This default setting may change in the future if it is demonstrated that the cost is generally acceptable.

The location of these measurements can be described in terms of whether the data is "consumed" or "produced", from the perspective of the
component to which the telemetry is attributed. Metrics which contain the term "produced" describe data which is emitted from the component,
while metrics which contain the term "consumed" describe data which is received by the component.

For both metrics, an `otelcol.component.outcome` attribute with possible values `success`, `failure`, and `refused` should be automatically recorded,
based on whether the corresponding function call returned successfully, returned an error originating from the associated component, or propagated an error from a component further downstream.

Specifically, a call to `ConsumeX` is recorded with:
- `otelcol.component.outcome = success` if the call returns `nil`;
- `otelcol.component.outcome = failure` if the call returns a regular error;
- `otelcol.component.outcome = refused` if the call returns an error tagged as coming from downstream.

After inspecting the error, the instrumentation layer should tag it as coming from downstream before returning it to the caller. Since there are two instrumentation layers between each pair of successive components (one recording produced data and one recording consumed data), this means that a call recorded with `outcome = failure` by the "consumer" layer will be recorded with `outcome = refused` by the "producer" layer, reflecting the fact that only the "consumer" component failed. In all other cases, the `outcome` recorded by both layers should be identical.

Errors should be "tagged as coming from downstream" the same way permanent errors are currently handled: they can be wrapped in a `type downstreamError struct { err error }` wrapper error type, then checked with `errors.As`. Note that care may need to be taken when dealing with the `multiError`s returned by the `fanoutconsumer`. If PR #11085 introducing a single generic `Error` type is merged, an additional `downstream bool` field can be added to it to serve the same purpose instead.

```yaml
    otelcol.receiver.produced.items:
      enabled: true
      description: Number of items emitted from the receiver.
      unit: "{item}"
      sum:
        value_type: int
        monotonic: true
    otelcol.processor.consumed.items:
      enabled: true
      description: Number of items passed to the processor.
      unit: "{item}"
      sum:
        value_type: int
        monotonic: true
    otelcol.processor.produced.items:
      enabled: true
      description: Number of items emitted from the processor.
      unit: "{item}"
      sum:
        value_type: int
        monotonic: true
    otelcol.connector.consumed.items:
      enabled: true
      description: Number of items passed to the connector.
      unit: "{item}"
      sum:
        value_type: int
        monotonic: true
    otelcol.connector.produced.items:
      enabled: true
      description: Number of items emitted from the connector.
      unit: "{item}"
      sum:
        value_type: int
        monotonic: true
    otelcol.exporter.consumed.items:
      enabled: true
      description: Number of items passed to the exporter.
      unit: "{item}"
      sum:
        value_type: int
        monotonic: true

    otelcol.receiver.produced.size:
      enabled: false
      description: Size of items emitted from the receiver.
      unit: "By"
      sum:
        value_type: int
        monotonic: true
    otelcol.processor.consumed.size:
      enabled: false
      description: Size of items passed to the processor.
      unit: "By"
      sum:
        value_type: int
        monotonic: true
    otelcol.processor.produced.size:
      enabled: false
      description: Size of items emitted from the processor.
      unit: "By"
      sum:
        value_type: int
        monotonic: true
    otelcol.connector.consumed.size:
      enabled: false
      description: Size of items passed to the connector.
      unit: "By"
      sum:
        value_type: int
        monotonic: true
    otelcol.connector.produced.size:
      enabled: false
      description: Size of items emitted from the connector.
      unit: "By"
      sum:
        value_type: int
        monotonic: true
    otelcol.exporter.consumed.size:
      enabled: false
      description: Size of items passed to the exporter.
      unit: "By"
      sum:
        value_type: int
        monotonic: true
```

#### Additional Attribute for Connectors

Connectors can route telemetry to specific pipelines. Therefore, `otelcol.connector.produced.*` metrics should carry an
additional data point attribute, `otelcol.pipeline.id`, to describe the pipeline ID to which the data is sent.

### Auto-Instrumented Logs

Metrics provide most of the observability we need but there are some gaps which logs can fill. Although metrics would describe the overall
item counts, it is helpful in some cases to record more granular events. For example, if a produced batch of 10,000 spans results in an error, but
100 batches of 100 spans succeed, this may be a matter of batch size that can be detected by analyzing logs, while the corresponding metric
reports only that a 50% success rate is observed.

For security and performance reasons, it would not be appropriate to log the contents of telemetry.

It's very easy for logs to become too noisy. Even if errors are occurring frequently in the data pipeline, only the errors that are not
handled automatically will be of interest to most users.

With the above considerations, this proposal includes only that we add a DEBUG log for each error, with the attributes from the corresponding 
metrics as well as the error message and item count. This should be sufficient for detailed troubleshooting but does not impact users otherwise.

In the future, it may be helpful to define triggers for reporting repeated failures at a higher severity level. e.g. N number of failures in
a row, or a moving average success %. For now, the criteria and necessary configurability is unclear so this is mentioned only as an example
of future possibilities.

### Auto-Instrumented Spans

It is not clear that any spans can be captured automatically with the proposed mechanism. We have the ability to insert instrumentation both
before and after processors and connectors. However, we generally cannot assume a 1:1 relationship between consumed and produced data.

## Additional Context

This proposal pulls from a number of issues and PRs:

- [Demonstrate graph-based metrics](https://github.com/open-telemetry/opentelemetry-collector/pull/11311)
- [Attributes for component instancing](https://github.com/open-telemetry/opentelemetry-collector/issues/11179)
- [Simple processor metrics](https://github.com/open-telemetry/opentelemetry-collector/issues/10708)
- [Component instancing is complicated](https://github.com/open-telemetry/opentelemetry-collector/issues/10534)
