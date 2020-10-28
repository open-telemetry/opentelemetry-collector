# General Information
A receiver is how data gets into the OpenTelemetry Collector. Generally, a receiver
accepts data in a specified format, translates it into the internal format and
passes it to [processors](../processor/README.md)
and [exporters](../exporter/README.md)
defined in the applicable [pipelines](../docs/pipelines.md).
The format of the traces and metrics supported are receiver specific.

Supported trace receivers (sorted alphabetically):
- [Jaeger Receiver](jaegerreceiver/README.md)
- [Kafka Receiver](kafkareceiver/README.md)
- [OpenCensus Receiver](opencensusreceiver/README.md)
- [OpenTelemetry Receiver](otlpreceiver/README.md)
- [Zipkin Receiver](zipkinreceiver/README.md)

Supported metric receivers (sorted alphabetically):
- [Host Metrics Receiver](hostmetricsreceiver/README.md)
- [OpenCensus Receiver](opencensusreceiver/README.md)
- [OpenTelemetry Receiver](otlpreceiver/README.md)
- [Prometheus Receiver](prometheusreceiver/README.md)

The [contributors repository](https://github.com/open-telemetry/opentelemetry-collector-contrib)
 has more receivers that can be added to custom builds of the collector.

## Configuring Receiver(s)
Receivers are configured via YAML under the top-level `receivers` tag. There
must be at least one enabled receiver for a configuration to be considered
valid.

The following is a sample configuration for the `examplereceiver`.
```yaml
receivers:
  # Receiver 1.
  # <receiver type>:
  examplereceiver:
    # <setting one>: <value one>
    endpoint: 1.2.3.4:8080
    # ...
  # Receiver 2.
  # <receiver type>/<name>:
  examplereceiver/settings:
    # <setting two>: <value two>
    endpoint: 0.0.0.0:9211
```

A receiver instance is referenced by its full name in other parts of the config,
such as in pipelines. A full name consists of the receiver type, '/' and the
name appended to the receiver type in the configuration. All receiver full names
must be unique. For the example above:
- Receiver 1 has full name `examplereceiver`.
- Receiver 2 has full name `examplereceiver/settings`.

All receivers expose a setting to disable it, by default receivers are enabled.
At least one receiver must be enabled per [pipeline](../docs/pipelines.md) to be a
valid configuration.
