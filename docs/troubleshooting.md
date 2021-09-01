# Troubleshooting

## Observability

The Collector offers multiple ways to measure the health of the Collector
as well as investigate issues.

### Logs

Logs can be helpful in identifying issues. Always start by checking the log
output and looking for potential issues.

The verbosity level, which defaults to `INFO` can also be adjusted by passing
the `--log-level` flag to the `otelcol` process. See `--help` for more details.

```bash
$ otelcol --log-level DEBUG
```

### Metrics

Prometheus metrics are exposed locally on port `8888` and path `/metrics`.

For containerized environments it may be desirable to expose this port on a
public interface instead of just locally. The metrics address can be configured
by passing the `--metrics-addr` flag to the `otelcol` process. See `--help` for
more details.

```bash
$ otelcol --metrics-addr 0.0.0.0:8888
```

A grafana dashboard for these metrics can be found
[here](https://grafana.com/grafana/dashboards/11575).

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
  logging:
service:
  pipelines:
    metrics:
      receivers: [prometheus]
      processors: []
      exporters: [logging]
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

For live troubleshooting purposes consider leveraging the `logging` exporter,
which can be used to confirm that data is being received, processed and
exported by the Collector.

```yaml
receivers:
  zipkin:
exporters:
  logging:
service:
  pipelines:
    traces:
      receivers: [zipkin]
      processors: []
      exporters: [logging]
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

```bash
$ curl -X POST localhost:9411/api/v2/spans -H'Content-Type: application/json' -d @trace.json
```

You should see a log entry like the following from the Collector:

```json
2020-11-11T04:12:33.089Z	INFO	loggingexporter/logging_exporter.go:296	TraceExporter	{"#spans": 1}
```

You can also configure the `logging` exporter so the entire payload is printed:

```yaml
exporters:
  logging:
    loglevel: debug
```

With the modified configuration if you re-run the test above the log output should look like:

```json
2020-11-11T04:08:17.344Z	DEBUG	loggingexporter/logging_exporter.go:353	ResourceSpans #0
Resource labels:
     -> service.name: STRING(api)
InstrumentationLibrarySpans #0
Span #0
    Trace ID       : 5982fe77008310cc80f1da5e10147519
    Parent ID      : 90394f6bcffb5d13
    ID             : 67fae42571535f60
    Name           : /m/n/2.6.1
    Kind           : SPAN_KIND_SERVER
    Start time     : 2018-01-24 08:16:15.726 +0000 UTC
    End time       : 2018-01-24 08:16:15.752 +0000 UTC
Attributes:
     -> data.http_response_code: STRING(201)
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
  [memory_limiter](https://github.com/open-telemetry/opentelemetry-collector/blob/main/processor/memorylimiter/README.md)
  processor.
- [Improperly sized](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/performance.md)
  for load.
- Improperly configured (for example, a queue size configured higher
  than available memory).
- Infrastructure resource limits (for example Kubernetes).

### Data being dropped

Data may be dropped for a variety of reasons, but most commonly because of an:

- [Improperly sized Collector](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/performance.md) resulting in Collector being unable to process and export the data as fast as it is received.
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

### Startup failing in Windows Docker containers

The process may fail to start in a Windows Docker container with the following
error: `The service process could not connect to the service controller`. In
this case the `NO_WINDOWS_SERVICE=1` environment variable should be set to force
the collector to be started as if it were running in an interactive terminal,
without attempting to run as a Windows service.
