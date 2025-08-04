# OpenTelemetry Collector Processor Exploration

**Status:** *Draft*

## Objective

To describe a user experience and strategies for configuring processors in the OpenTelemetry collector.

This work is being prototyped in opentelemetry-collector-contrib, the design doc is here for broader discussion.

## Summary

The OpenTelemetry (OTel) collector is a tool to set up pipelines to receive telemetry from an application and export it
to an observability backend. Part of the pipeline can include processing stages, which executes various business logic
on incoming telemetry before it is exported.

Over time, the collector has added various processors to satisfy different use cases, generally in an ad-hoc way to
support each feature independently. We can improve the experience for users of the collector by consolidating processing
patterns in terms of user experience, and this can be supported by defining a querying model for processors
within the collector core, and likely also for use in SDKs, to simplify implementation and promote the consistent user
experience and best practices.

## Goals and non-goals

Goals:
- List out use cases for processing within the collector
- Consider what could be an ideal configuration experience for users

Non-Goals:
- Merge every processor into one. Many use cases overlap and generalize, but not all of them
- Technical design or implementation of configuration experience. Currently focused on user experience.

## Use cases for processing

### Telemetry mutation

Processors can be used to mutate the telemetry in the collector pipeline. OpenTelemetry SDKs collect detailed telemetry
from applications, and it is common to have to mutate this into a way that is appropriate for an individual use case.

Some types of mutation include

- Remove a forbidden attribute such as `http.request.header.authorization`
- Reduce cardinality of an attribute such as translating `http.target` value of `/user/123451/profile` to `/user/{userId}/profile`
- Decrease the size of the telemetry payload by removing large resource attributes such as `process.command_line`
- Filtering out signals such as by removing all telemetry with a `http.target` of `/health`
- Attach information from resource into telemetry, for example adding certain resource fields as metric dimensions

The processors implementing this use case are `attributesprocessor`, `filterprocessor`, `metricstransformprocessor`, 
`resourceprocessor`, `spanprocessor`.

### Metric generation

The collector may generate new metrics based on incoming telemetry. This can be for covering gaps in SDK coverage of
metrics vs spans, or to create new metrics based on existing ones to model the data better for backend-specific
expectations.

- Create new metrics based on information in spans, for example to create a duration metric that is not implemented in the SDK yet
- Apply arithmetic between multiple incoming metrics to produce an output one, for example divide an `amount` and a `capacity` to create a `utilization` metric

The components implementing this use case are `metricsgenerationprocessor` and the former `spanmetricsprocessor` (now `spanmetricsconnector`).

### Grouping

Some processors are stateful, grouping telemetry over a window of time based on either a trace ID or an attribute value,
or just general batching.

- Batch incoming telemetry before sending to exporters to reduce export requests
- Group spans by trace ID to allow doing tail sampling
- Group telemetry for the same path

The processors implementing this use case are `batchprocessor`, `groupbyattrprocessor`, `groupbytraceprocessor`.

### Metric temporality

Two processors convert between the two types of temporality, cumulative and delta. The conversion is generally expected
to happen as close to the source data as possible, for example within receivers themselves. The same configuration
mechanism could be used for selecting metrics for temporality conversion as other cases, but it is expected that in
practice configuration will be limited.

The processors implementing this use case are `cumulativetodeltaprocessor`.

### Telemetry enrichment

OpenTelemetry SDKs focus on collecting application specific data. They also may include resource detectors to populate
environment specific data but the collector is commonly used to fill gaps in coverage of environment specific data.

- Add environment about a cloud provider to `Resource` of all incoming telemetry

The processors implementing this use case are `k8sattributesprocessor`, `resourcedetectionprocessor`.

## OpenTelemetry Transformation Language

When looking at the use cases, there are certain common features for telemetry mutation and metric generation.

- Identify the type of signal (`span`, `metric`, `log`).
- Navigate to a path within the telemetry to operate on it
- Define an operation, and possibly operation arguments

We can try to model these into a transformation language, in particular allowing the first two points to be shared among all
processing operations, and only have implementation of individual types of processing need to implement operators that
the user can use within an expression.

Telemetry is modeled in the collector as [`pdata`](https://github.com/open-telemetry/opentelemetry-collector/tree/main/pdata)
which is roughly a 1:1 mapping of the [OTLP protocol](https://github.com/open-telemetry/opentelemetry-proto/tree/main/opentelemetry/proto).
This data can be navigated using field expressions, which are fields within the protocol separated by dots. For example,
the status message of a span is `status.message`. A map lookup can include the key as a string, for example `attributes["http.status_code"]`.

Operations are scoped to the type of a signal (`span`, `metric`, `log`), with all of the flattened points of that
signal being part of a transformation space. Virtual fields are added to access data from a higher level before flattening, for
`resource`, `library_info`. For metrics, the structure presented for processing is actual data points, e.g. `NumberDataPoint`, 
`HistogramDataPoint`, with the information from higher levels like `Metric` or the data type available as virtual fields.

Virtual fields for all signals: `resource`, `library_info`.  
Virtual fields for metrics: `metric`, which contains `name`, `description`, `unit`, `type`, `aggregation_temporality`, and `is_monotonic`.

Navigation can then be used with a simple expression language for identifying telemetry to operate on.

```
... where name = "GET /cats"
```
```
... from span where attributes["http.target"] = "/health"
```
```
... where resource.attributes["deployment"] = "canary"
```
```
... from metric where metric.type = gauge
```
```
... from metric where metric.name = "http.active_requests"
```

Fields should always be fully specified - for example `attributes` refers to the `attributes` field in the telemetry, not
the `resource`. In the future, we may allow shorthand for accessing scoped information that is not ambiguous.

Having selected telemetry to operate on, any needed operations can be defined as functions. Known useful functions should
be implemented within the collector itself, provide registration from extension modules to allow customization with
contrib components, and in the future can even allow user plugins possibly through WASM, similar to work in 
[HTTP proxies](https://github.com/proxy-wasm/spec). The arguments to operations will primarily be field expressions,
allowing the operation to mutate telemetry as needed.

There are times when the transformation language input and the underlying telemetry model do not translate cleanly.  For example, a span ID is represented in pdata as a SpanID struct, but in the transformation language it is more natural to represent the span ID as a string or a byte array.  The solution to this problem is Factories. Factories are functions that help translate between the transformation language input into the underlying pdata structure.  These types of functions do not change the telemetry in any way.  Instead, they manipulate the transformation language input into a form that will make working with the telemetry easier or more efficient.

### Examples

These examples contain a SQL-like declarative language.  Applied statements interact with only one signal, but statements can be declared across multiple signals.

Remove a forbidden attribute such as `http.request.header.authorization` from spans only

```
traces:
  delete(attributes["http.request.header.authorization"])
metrics:
  delete(attributes["http.request.header.authorization"])
logs:
  delete(attributes["http.request.header.authorization"])
```

Remove all attributes except for some

```
traces:
  keep_keys(attributes, "http.method", "http.status_code")
metrics:
  keep_keys(attributes, "http.method", "http.status_code")
logs:
  keep_keys(attributes, "http.method", "http.status_code")
```

Reduce cardinality of an attribute

```
traces:
  replace_match(attributes["http.target"], "/user/*/list/*", "/user/{userId}/list/{listId}")
```

Reduce cardinality of a span name

```
traces:
  replace_match(name, "GET /user/*/list/*", "GET /user/{userId}/list/{listId}")
``` 

Reduce cardinality of any matching attribute

```
traces:
  replace_all_matches(attributes, "/user/*/list/*", "/user/{userId}/list/{listId}")
``` 

Decrease the size of the telemetry payload by removing large resource attributes

```
traces:
  delete(resource.attributes["process.command_line"])
metrics:
  delete(resource.attributes["process.command_line"])
logs:
  delete(resource.attributes["process.command_line"])
```

Filtering out signals such as by removing all metrics with a `http.target` of `/health`

```
metrics:
  drop() where attributes["http.target"] = "/health"
```

Attach information from resource into telemetry, for example adding certain resource fields as metric attributes

```
metrics:
  set(attributes["k8s_pod"], resource.attributes["k8s.pod.name"])
```

Group spans by trace ID

```
traces:
  group_by(trace_id, 2m)
```


Update a spans ID

```
logs:
  set(span_id, SpanID(0x0000000000000000))
traces:
  set(span_id, SpanID(0x0000000000000000))
```

Create utilization metric from base metrics. Because navigation expressions only operate on a single piece of telemetry,
helper functions for reading values from other metrics need to be provided.

```
metrics:
  create_gauge("pod.cpu.utilized", read_gauge("pod.cpu.usage") / read_gauge("node.cpu.limit")
```

A lot of processing. Queries are executed in order. While initially performance may degrade compared to more specialized
processors, the expectation is that over time, the transform processor's engine would improve to be able to apply optimizations 
across queries, compile into machine code, etc.

```yaml
receivers:
  otlp:

exporters:
  otlp:

processors:
  transform:
    # Assuming group_by is defined in a contrib extension module, not baked into the "transform" processor
    extensions: [group_by]
    traces:
      queries:
        - drop() where attributes["http.target"] = "/health"
        - delete(attributes["http.request.header.authorization"])
        - replace_wildcards("/user/*/list/*", "/user/{userId}/list/{listId}", attributes["http.target"])
        - group_by(trace_id, 2m)
    metrics:
      queries:
        - drop() where attributes["http.target"] = "/health"
        - delete(attributes["http.request.header.authorization"])
        - replace_wildcards("/user/*/list/*", "/user/{userId}/list/{listId}", attributes["http.target"])
        - set(attributes["k8s_pod"], resource.attributes["k8s.pod.name"])
    logs:
      queries:
        - drop() where attributes["http.target"] = "/health"
        - delete(attributes["http.request.header.authorization"])
        - replace_wildcards("/user/*/list/*", "/user/{userId}/list/{listId}", attributes["http.target"])

pipelines:
  - receivers: [otlp]
    exporters: [otlp]
    processors: [transform]
```

The expressions would be executed in order, with each expression either mutating an input telemetry, dropping input
telemetry, or adding additional telemetry. One caveat to note is that we would like to implement optimizations
in the transform engine, for example to only apply filtering once for multiple operations with a shared filter. Functions
with unknown side effects may cause issues with optimization we will need to explore.

## Declarative configuration

The telemetry transformation language presents an SQL-like experience for defining telemetry transformations - it is made up of
the three primary components described above, however, and can be presented declaratively instead depending on what makes
sense as a user experience.

```yaml
- type: span
  filter:
    match:
      path: status.code
      value: OK
  operation:
    name: drop
- type: all
  operation:
    name: delete
    args:
      - attributes["http.request.header.authorization"]
```

An implementation of the transformation language would likely parse expressions into this sort of structure so given an SQL-like
implementation, it would likely be little overhead to support a YAML approach in addition.

## Function syntax

Functions should be named and formatted according to the following standards.
- Function names MUST start with a verb unless it is a Factory.
- Factory functions MUST be UpperCamelCase and named based on the object being created.
- Function names that contain multiple words MUST separate those words with `_`.
- Functions that interact with multiple items MUST have plurality in the name.  Ex: `truncate_all`, `keep_keys`, `replace_all_matches`.
- Functions that interact with a single item MUST NOT have plurality in the name.  If a function would interact with multiple items due to a condition, like `where`, it is still considered singular.  Ex: `set`, `delete`, `drop`, `replace_match`.
- Functions that change a specific target MUST set the target as the first parameter.
- Functions that take a list MUST set the list as the last parameter.

## Implementing a processor function

The `replace_match` function may look like this.

```go

package replaceMatch

import "regexp"

import "github.com/open-telemetry/opentelemetry/processors"

// Assuming this is not in "core"
processors.register("replace_match", replace_match)

func replace_match(path processors.TelemetryPath, pattern regexp.Regexp, replacement string) processors.Result  {
    val := path.Get()
	if val == nil {
		return processors.CONTINUE
    }
	
	// replace finds placeholders in "replacement" and swaps them in for regex matched substrings.
	replaced := replace(val, pattern, replacement)
	path.Set(replaced)
	return processors.CONTINUE
}
```

Here, the processor framework recognizes the second parameter of the function is `regexp.Regexp` so will compile the string
provided by the user in the config when processing it. Similarly for `path`, it recognizes properties of type `TelemetryPath`
and will resolve it to the path within a matched telemetry during execution and pass it to the function. The path allows
scalar operations on the field within the telemetry. The processor does not need to be aware of telemetry filtering,
the `where ...` clause, as that will be handled by the framework before passing to the function.

## Embedded processors

The above describes a transformation language for configuring processing logic in the OpenTelemetry collector. There will be a
single processor that exposes the processing logic into the collector config; however, the logic will be implemented
within core packages rather than directly inside a processor. This is to ensure that where appropriate, processing
can be embedded into other components, for example metric processing is often most appropriate to execute within a
receiver based on receiver-specific requirements.

## Limitations

There are some known issues and limitations that we hope to address while iterating on this idea.

- Handling array-typed attributes
- Working on a array of points, rather than a single point
- Metric alignment - for example defining an expression on two metrics, that may not be at the same timestamp
- The collector has separate pipelines per signal - while the transformation language could apply cross-signal, we will need to remain single-signal for now
