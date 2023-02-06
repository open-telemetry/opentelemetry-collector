# Forward Connector

| Status                   |                                                           |
|------------------------- |---------------------------------------------------------- |
| Stability                | [Alpha]                                          |
| Supported pipeline types | See [Supported Pipeline Types](#supported-pipeline-types) |
| Distributions            | []                                                        |

The `forward` connector can merge or fork pipelines of the same type.

## Supported Pipeline Types

| [Exporter Pipeline Type] | [Receiver Pipeline Type] |
| ------------------------ | ------------------------ |
| traces                   | traces                   |
| metrics                  | metrics                  |
| logs                     | logs                     |

## Configuration

If you are not already familiar with connectors, you may find it helpful to first visit the [Connectors README].

The `forward` connector does not have any configuration settings.

```yaml
receivers:
  foo:
exporters:
  bar:
connectors:
  forward:
```

### Example Usage

Annotate distinct log streams, then merge them together, batch, and export.

```yaml
receivers:
  foo/blue:
  foo/green:
processors:
  attributes/blue:
  attributes/green:
  batch:
exporters:
  bar:
connectors:
  forward:
service:
  pipelines:
    logs/blue:
      receivers: [foo/blue]
      processors: [attributes/blue]
      exporters: [forward]
    logs/green:
      receivers: [foo/green]
      processors: [attributes/green]
      exporters: [forward]
    logs:
      receivers: [forward]
      processors: [batch]
      exporters: [bar]
```

Preprocess data, then replicate and handle in distinct ways.

```yaml
receivers:
  foo:
processors:
  resourcedetection:
  sample:
  attributes:
  batch:
exporters:
  bar/hot:
  bar/cold:
connectors:
  forward:
service:
  pipelines:
    traces:
      receivers: [foo]
      processors: [resourcedetection]
      exporters: [forward]
    traces/hot:
      receivers: [forward]
      processors: [sample, batch]
      exporters: [bar/hot]
    traces/cold:
      receivers: [forward]
      processors: [attributes]
      exporters: [bar/cold]
```

Add a temporary debugging exporter. (Uncomment to enable.)

```yaml
receivers:
  foo:
processors:
  filter:
  batch:
exporters:
  bar:
# connectors:
#   forward:
service:
  pipelines:
    traces:
      receivers:
        - foo
      processors:
        - filter
        - batch
      exporters:
        - bar
      # - forward
  # traces/log:
  #   receivers: [forward]
  #   exporters: [logging]
```

[Alpha]:https://github.com/open-telemetry/opentelemetry-collector#alpha
[Connectors README]:https://github.com/open-telemetry/opentelemetry-collector/blob/main/connector/README.md
[Exporter Pipeline Type]:https://github.com/open-telemetry/opentelemetry-collector/blob/main/connector/README.md#exporter-pipeline-type
[Receiver Pipeline Type]:https://github.com/open-telemetry/opentelemetry-collector/blob/main/connector/README.md#receiver-pipeline-type
