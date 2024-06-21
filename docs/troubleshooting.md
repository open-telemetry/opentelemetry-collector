# Troubleshooting

To troubleshoot the Collector, see the [Troubleshooting] page on the project's
website. The following document offers experimental methods for troubleshooting.

### Traces

OpenTelemetry Collector has an ability to send it's own traces using OTLP
exporter. You can send the traces to OTLP server running on the same
OpenTelemetry Collector, so it goes through configured pipelines. For example:

```yaml
service:
  telemetry:
    traces:
      processors:
        batch:
          exporter:
            otlp:
              protocol: grpc/protobuf
              endpoint: ${MY_POD_IP}:4317
```

### Null Maps in Configuration

If you've ever experienced issues during configuration resolution where
sections, like `processors:` from earlier configuration are removed, see
[confmap](../confmap/README.md#troubleshooting)

[Troubleshooting]: https://opentelemetry.io/docs/collector/troubleshooting/
