# Fluent Forward Receiver Extension

This receiver is an extension to the [Fluent Forward receiver](../fluentforwardreceiver/) that allows the receiver to be installed both in the logs and traces pipelines, and forwards data to the appropriate pipelines based on the tag.

With the addition of the FluentBit exporter, it is possible to send data to the collector using the Fluentd Forward protocol. The fluentforwardreceiver runs a server that accepts events using this protocol, and this component acts an extension to it to support the parsing and conversion of span data to opentelemetry format.

Installing the receiver is same as the fluentforwardreceiver, with the following example config allowung the receiver to listen on all interfaces on port 8006:

```yaml
receivers:
  fluentforwardext:
    endpoint: 0.0.0.0:8006
```

The difference here is that now we can install this receiver in the traces pipeline too, with the following example config:

```yaml
receivers:
  fluentforwardext:
    endpoint: 127.0.0.1:8006

processors:
  batch:

exporters:
  logging:
  jaeger:
    endpoint: localhost:14250
    insecure: true

service:
  pipelines:
    traces:
      receivers: [fluentforwardext]
      processors: [batch]
      exporters: [logging, jaeger]
    logs:
      receivers: [fluentforwardext]
      processors: [batch]
      exporters: [logging]
```

In the above config, fluentforwardext receiver listens on port 8006 for events, and routes the events to the appropriate pipeline based on the tag. All events with the tag `data.span` will be parsed as spans and passed on to the traces pipeline, where they are eventually exported to Jaeger, while all other events will be parsed as logs and will be logged to the console using the logging exporter.