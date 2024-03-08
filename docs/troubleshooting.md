# Troubleshooting

## Observability

The Collector offers multiple ways to measure the health of the Collector
as well as investigate issues.

### Logs

Logs can be helpful in identifying issues. Always start by checking the log
output and looking for potential issues.
The verbosity level defaults to `INFO` and can be adjusted.

Set the log level in the config `service::telemetry::logs`

```yaml
service:
  telemetry:
    logs:
      level: "debug"
```

### Metrics

Prometheus metrics are exposed locally on port `8888` and path `/metrics`. For
containerized environments it may be desirable to expose this port on a
public interface instead of just locally.

Set the address in the config `service::telemetry::metrics`

```yaml
service:
  telemetry:
    metrics:
      address: ":8888"
```

A Grafana dashboard for these metrics can be found
[here](https://grafana.com/grafana/dashboards/15983-opentelemetry-collector/).

You can enhance metrics telemetry level using `level` field. The following is a list of all possible values and their explanations.

- "none" indicates that no telemetry data should be collected;
- "basic" is the recommended and covers the basics of the service telemetry.
- "normal" adds some other indicators on top of basic.
- "detailed" adds dimensions and views to the previous levels.

For example:
```yaml
service:
  telemetry:
    metrics:
      level: detailed
      address: ":8888"
```

Also note that a Collector can be configured to scrape its own metrics and send
it through configured pipelines. For example:

```yaml
receivers:
  prometheus:
    config:
      scrape_configs:
      - job_name: 'otelcol'
        scrape_interval: 10s
        static_configs:
        - targets: ['0.0.0.0:8888']
        metric_relabel_configs:
          - source_labels: [ __name__ ]
            regex: '.*grpc_io.*'
            action: drop
exporters:
  debug:
service:
  pipelines:
    metrics:
      receivers: [prometheus]
      processors: []
      exporters: [debug]
```

### Traces

OpenTelemetry Collector has an ability to send it's own traces using OTLP exporter. You can send the traces to OTLP server running on the same OpenTelemetry Collector, so it goes through configured pipelines. For example:

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

### zPages

The
[zpages](https://github.com/open-telemetry/opentelemetry-collector/tree/main/extension/zpagesextension/README.md)
extension, which if enabled is exposed locally on port `55679`, can be used to
check receivers and exporters trace operations via `/debug/tracez`. `zpages`
may contain error logs that the Collector does not emit.

For containerized environments it may be desirable to expose this port on a
public interface instead of just locally. This can be configured via the
extensions configuration section. For example:

```yaml
extensions:
  zpages:
    endpoint: 0.0.0.0:55679
```

### Local exporters

[Local
exporters](https://github.com/open-telemetry/opentelemetry-collector/tree/main/exporter#general-information)
can be configured to inspect the data being processed by the Collector.

For live troubleshooting purposes consider leveraging the `debug` exporter,
which can be used to confirm that data is being received, processed and
exported by the Collector.

```yaml
receivers:
  zipkin:
exporters:
  debug:
service:
  pipelines:
    traces:
      receivers: [zipkin]
      processors: []
      exporters: [debug]
```

Get a Zipkin payload to test. For example create a file called `trace.json`
that contains:

```json
[
  {
    "traceId": "5982fe77008310cc80f1da5e10147519",
    "parentId": "90394f6bcffb5d13",
    "id": "67fae42571535f60",
    "kind": "SERVER",
    "name": "/m/n/2.6.1",
    "timestamp": 1516781775726000,
    "duration": 26000,
    "localEndpoint": {
      "serviceName": "api"
    },
    "remoteEndpoint": {
      "serviceName": "apip"
    },
    "tags": {
      "data.http_response_code": "201"
    }
  }
]
```

With the Collector running, send this payload to the Collector. For example:

```console
$ curl -X POST localhost:9411/api/v2/spans -H'Content-Type: application/json' -d @trace.json
```

You should see a log entry like the following from the Collector:

```
2023-09-07T09:57:43.468-0700    info    TracesExporter  {"kind": "exporter", "data_type": "traces", "name": "debug", "resource spans": 1, "spans": 2}
```

You can also configure the `debug` exporter so the entire payload is printed:

```yaml
exporters:
  debug:
    verbosity: detailed
```

With the modified configuration if you re-run the test above the log output should look like:

```
2023-09-07T09:57:12.820-0700    info    TracesExporter  {"kind": "exporter", "data_type": "traces", "name": "debug", "resource spans": 1, "spans": 2}
2023-09-07T09:57:12.821-0700    info    ResourceSpans #0
Resource SchemaURL: https://opentelemetry.io/schemas/1.4.0
Resource attributes:
     -> service.name: Str(telemetrygen)
ScopeSpans #0
ScopeSpans SchemaURL: 
InstrumentationScope telemetrygen 
Span #0
    Trace ID       : 0c636f29e29816ea76e6a5b8cd6601cf
    Parent ID      : 1a08eba9395c5243
    ID             : 10cebe4b63d47cae
    Name           : okey-dokey
    Kind           : Internal
    Start time     : 2023-09-07 16:57:12.045933 +0000 UTC
    End time       : 2023-09-07 16:57:12.046058 +0000 UTC
    Status code    : Unset
    Status message : 
Attributes:
     -> span.kind: Str(server)
     -> net.peer.ip: Str(1.2.3.4)
     -> peer.service: Str(telemetrygen)
```

### Health Check

The
[health_check](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/extension/healthcheckextension/README.md)
extension, which by default is available on all interfaces on port `13133`, can
be used to ensure the Collector is functioning properly.

```yaml
extensions:
  health_check:
service:
  extensions: [health_check]
```

It returns a response like the following:

```json
{
  "status": "Server available",
  "upSince": "2020-11-11T04:12:31.6847174Z",
  "uptime": "49.0132518s"
}
```

### pprof

The
[pprof](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/extension/pprofextension/README.md)
extension, which by default is available locally on port `1777`, allows you to profile the
Collector as it runs. This is an advanced use-case that should not be needed in most circumstances.

## Common Issues

To see logs for the Collector:

On a Linux systemd system, logs can be found using `journalctl`:  
`journalctl | grep otelcol`

or to find only errors:  
`journalctl | grep otelcol | grep Error`

### Collector exit/restart

The Collector may exit/restart because:

- Memory pressure due to missing or misconfigured
  [memory_limiter](https://github.com/open-telemetry/opentelemetry-collector/blob/main/processor/memorylimiterprocessor/README.md)
  processor.
- Improperly sized for load.
- Improperly configured (for example, a queue size configured higher
  than available memory).
- Infrastructure resource limits (for example Kubernetes).

### Data being dropped

Data may be dropped for a variety of reasons, but most commonly because of an:

- Improperly sized Collector resulting in Collector being unable to process and export the data as fast as it is received.
- Exporter destination unavailable or accepting the data too slowly.

To mitigate drops, it is highly recommended to configure the
[batch](https://github.com/open-telemetry/opentelemetry-collector/blob/main/processor/batchprocessor/README.md)
processor. In addition, it may be necessary to configure the [queued retry
options](https://github.com/open-telemetry/opentelemetry-collector/tree/main/exporter/exporterhelper#configuration)
on enabled exporters.

### Receiving data not working

If you are unable to receive data then this is likely because
either:

- There is a network configuration issue
- The receiver configuration is incorrect
- The receiver is defined in the `receivers` section, but not enabled in any `pipelines`
- The client configuration is incorrect

Check the Collector logs as well as `zpages` for potential issues.

### Processing data not working

Most processing issues are a result of either a misunderstanding of how the
processor works or a misconfiguration of the processor.

Examples of misunderstanding include:

- The attributes processors only work for "tags" on spans. Span name is
  handled by the span processor.
- Processors for trace data (except tail sampling) work on individual spans.

### Exporting data not working

If you are unable to export to a destination then this is likely because
either:

- There is a network configuration issue
- The exporter configuration is incorrect
- The destination is unavailable

Check the collector logs as well as `zpages` for potential issues.

More often than not, exporting data does not work because of a network
configuration issue. This could be due to a firewall, DNS, or proxy
issue. Note that the Collector does have
[proxy support](https://github.com/open-telemetry/opentelemetry-collector/tree/main/exporter#proxy-support).

### Startup failing in Windows Docker containers (v0.90.1 and earlier)

The process may fail to start in a Windows Docker container with the following
error: `The service process could not connect to the service controller`. In
this case the `NO_WINDOWS_SERVICE=1` environment variable should be set to force
the collector to be started as if it were running in an interactive terminal,
without attempting to run as a Windows service.
