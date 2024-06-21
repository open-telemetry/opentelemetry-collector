# OpenTelemetry Collector internal observability

The [Internal telemetry] page on OpenTelemetry's website contains the
documentation for the Collector's internal observability, including:

- Which types of observability are emitted by the Collector.
- How to enable and configure these signals.
- How to use this telemetry to monitor your Collector instance.

If you need to troubleshoot the Collector, see [Troubleshooting].

Read on to learn about experimental features and the project's overall vision
for internal telemetry.

## Experimental trace telemetry

The Collector does not expose traces by default, but an effort is underway to
[change this][issue7532]. The work includes supporting configuration of the
OpenTelemetry SDK used to produce the Collector's internal telemetry. This
feature is behind two feature gates:

```bash
  --feature-gates=telemetry.useOtelWithSDKConfigurationForInternalTelemetry
```

The gate `useOtelWithSDKConfigurationForInternalTelemetry` enables the Collector
to parse any configuration that aligns with the [OpenTelemetry Configuration]
schema. Support for this schema is experimental, but it does allow telemetry to
be exported using OTLP.

The following configuration can be used in combination with the aforementioned
feature gates to emit internal metrics and traces from the Collector to an OTLP
backend:

```yaml
service:
  telemetry:
    metrics:
      readers:
        - periodic:
            interval: 5000
            exporter:
              otlp:
                protocol: grpc/protobuf
                endpoint: https://backend:4317
    traces:
      processors:
        - batch:
            exporter:
              otlp:
                protocol: grpc/protobuf
                endpoint: https://backend2:4317
```

See the [example configuration][kitchen-sink] for additional options.

> This configuration does not support emitting logs as there is no support for
> [logs] in the OpenTelemetry Go SDK at this time.

You can also configure the Collector to send its own traces using the OTLP
exporter. Send the traces to an OTLP server running on the same Collector, so it
goes through configured pipelines. For example:

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

## Goals of internal telemetry

The Collector's internal telemetry is an important part of fulfilling
OpenTelemetry's [project vision](vision.md). The following section explains the
priorities for making the Collector an observable service.

### Observable elements

The following aspects of the Collector need to be observable.

- [Current values]
  - Some of the current values and rates might be calculated as derivatives of
    cumulative values in the backend, so it's an open question whether to expose
    them separately or not.
- [Cumulative values]
- [Trace or log events]
  - For start or stop events, an appropriate hysteresis must be defined to avoid
    generating too many events. Note that start and stop events can't be
    detected in the backend simply as derivatives of current rates. The events
    include additional data that is not present in the current value.
- [Host metrics]
  - Host metrics can help users determine if the observed problem in a service
    is caused by a different process on the same host.

### Impact

The impact of these observability improvements on the core performance of the
Collector must be assessed.

### Configurable level of observability

Some metrics and traces can be high volume and users might not always want to
observe them. An observability verboseness “level” allows configuration of the
Collector to send more or less observability data or with even finer
granularity, to allow turning on or off specific metrics.

The default level of observability must be defined in a way that has
insignificant performance impact on the service.

[Internal telemetry]:
  https://opentelemetry.io/docs/collector/internal-telemetry/
[Troubleshooting]: https://opentelemetry.io/docs/collector/troubleshooting/
[issue7532]:
  https://github.com/open-telemetry/opentelemetry-collector/issues/7532
[issue7454]:
  https://github.com/open-telemetry/opentelemetry-collector/issues/7454
[logs]: https://github.com/open-telemetry/opentelemetry-go/issues/3827
[OpenTelemetry Configuration]:
  https://github.com/open-telemetry/opentelemetry-configuration
[kitchen-sink]:
  https://github.com/open-telemetry/opentelemetry-configuration/blob/main/examples/kitchen-sink.yaml
[Current values]:
  https://opentelemetry.io/docs/collector/internal-telemetry/#values-observable-with-internal-metrics
[Cumulative values]:
  https://opentelemetry.io/docs/collector/internal-telemetry/#values-observable-with-internal-metrics
[Trace or log events]:
  https://opentelemetry.io/docs/collector/internal-telemetry/#events-observable-with-internal-logs
[Host metrics]:
  https://opentelemetry.io/docs/collector/internal-telemetry/#lists-of-internal-metrics
