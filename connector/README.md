# Connectors

A connector is both an exporter and receiver. As the name suggests a Connector connects
two pipelines: it emits data as an exporter at the end of one pipeline and consumes data
as a receiver at the start of another pipeline. It may consume and emit data of the same data
type, or of different data types. A connector may generate and emit data to summarize the
consumed data, or it may simply replicate or route data.

## Supported Data Types

Each type of connector is designed to work with one or more _pairs_ of data types and may only
be used to connect pipelines accordingly. (Recall that every pipeline is associated with a single
data type, either traces, metrics, or logs.)

For example, the `count` connector counts traces, metrics, and logs, and reports the counts as a
metric. Therefore, it may be used to connect the following types of pipelines.

| [Exporter Pipeline Type] | [Receiver Pipeline Type] |
| ------------------------ | ------------------------ |
| traces                   | metrics                  |
| metrics                  | metrics                  |
| logs                     | metrics                  |

Another example, the `router` connector, is useful for routing data onto the appropriate pipeline
so that it may be processed in distinct ways  and/or exported to an appropriate backend. It does not
alter the data it consumes in any ways, nor does it produce any additional data. Therefore, it may be
used to connect the following types of pipelines.

| [Exporter Pipeline Type] | [Receiver Pipeline Type] |
| ------------------------ | ------------------------ |
| traces                   | traces                   |
| metrics                  | metrics                  |
| logs                     | logs                     |

## Available Connectors and Distributions

Connectors are modular components compiled into the OpenTelemetry Collector binary. The component name specified in the configuration (e.g. `forward`, `count`, `router`) must correspond to a connector component type available in your Collector distribution:

- **Core Distribution**: The core OpenTelemetry Collector includes the [`forward` connector](./forwardconnector/README.md), which simply passes telemetry unchanged from an exporter pipeline to a receiver pipeline of the same data type.
- **Contrib Distribution**: Additional connectors (such as `count`, `router`, `spanmetrics`, etc.) are available in the [OpenTelemetry Collector Contrib distribution](https://github.com/open-telemetry/opentelemetry-collector-contrib).
- **Custom Distributions**: You can build custom Collector distributions that bundle specific connectors using the [OpenTelemetry Collector Builder (OCB)](../cmd/builder/README.md).

> **Note:** If you use a connector type in your configuration that is not included in the Collector distribution you are running, you will encounter an error like `'connectors' unknown type: "<name>" for id: "<name>" (valid values: [...])`. Ensure you are running a distribution (such as `otelcol-contrib`) that includes the desired connector.

## Configuration

### Declaration

Connectors are defined within a dedicated `connectors` section at the top level of the collector config.
The declaration syntax follows the standard Collector component ID format: `<type>` or `<type>/<name>` (e.g., `forward`, `count`, `forward/custom_name`).

Connectors can be configured with default settings or with custom parameters defined by each connector's configuration:

```yaml
receivers:
  otlp:
exporters:
  otlp:
connectors:
  forward:
  count: # (Available in Contrib distribution)
  router: # (Available in Contrib distribution)
```

### Usage

Recall that a connector _is_ an exporter _and_ a receiver and that each connector
MUST be used as both, in separate pipelines.

```yaml
receivers:
  foo:
exporters:
  bar:
connectors:
  forward:
service:
  pipelines:
    traces:
      receivers: [foo]
      exporters: [forward]
    traces/downstream:
      receivers: [forward]
      exporters: [bar]
```

Connectors can be used alongside traditional exporters.

```yaml
receivers:
  foo:
exporters:
  bar/traces_backend:
  bar/metrics_backend:
connectors:
  count:
service:
  pipelines:
    traces:
      receivers: [foo]
      exporters: [bar/traces_backend, count]
    metrics:
      receivers: [count]
      exporters: [bar/metrics_backend]
```

Connectors can be used alongside traditional receivers.

```yaml
receivers:
  foo/traces:
  foo/metrics:
exporters:
  bar:
connectors:
  count:
service:
  pipelines:
    traces:
      receivers: [foo/traces]
      exporters: [count]
    metrics:
      receivers: [foo/metrics, count]
      exporters: [bar]
```

A connector can be an exporter in multiple pipelines.

```yaml
receivers:
  foo/traces:
  foo/metrics:
  foo/logs:
exporters:
  bar/traces_backend:
  bar/metrics_backend:
  bar/logs_backend:
connectors:
  count:
service:
  pipelines:
    traces:
      receivers: [foo/traces]
      exporters: [bar/traces_backend, count]
    metrics:
      receivers: [foo/metrics]
      exporters: [bar/metrics_backend, count]
    logs:
      receivers: [foo/logs]
      exporters: [bar/logs_backend, count]
    metrics/counts:
      receivers: [count]
      exporters: [bar/metrics_backend]
```

A connector can be a receiver in multiple pipelines.

```yaml
receivers:
  foo/traces:
  foo/metrics:
exporters:
  bar/traces_backend:
  bar/metrics_backend:
  bar/metrics_backend/2:
connectors:
  count:
service:
  pipelines:
    traces:
      receivers: [foo/traces]
      exporters: [bar/traces_backend, count]
    metrics:
      receivers: [count]
      exporters: [bar/metrics_backend]
    metrics/2:
      receivers: [count]
      exporters: [bar/metrics_backend/2]
```

Multiple connectors can be used in sequence.

```yaml
receivers:
  foo:
exporters:
  bar:
connectors:
  count:
  count/the_counts:
service:
  pipelines:
    traces:
      receivers: [foo]
      exporters: [count]
    metrics:
      receivers: [count]
      exporters: [bar/metrics_backend, count/the_counts]
    metrics/count_the_counts:
      receivers: [count/the_counts]
      exporters: [bar]
```

A connector can only be used in a pair of pipelines when it supports the combination of
[Exporter Pipeline Type] and [Receiver Pipeline Type].

```yaml
receivers:
  foo:
exporters:
  bar:
connectors:
  count:
service:
  pipelines:
    traces:
      receivers: [foo]
      exporters: [count]
    logs:
      receivers: [count] # Invalid. The count connector does not support traces -> logs.
      exporters: [bar]
```

#### Exporter Pipeline Type

The type of pipeline in which a connector is used as an exporter.

#### Receiver Pipeline Type

The type of pipeline in which the connector is used as a receiver.

[Exporter Pipeline Type]:#exporter-pipeline-type
[Receiver Pipeline Type]:#receiver-pipeline-type
