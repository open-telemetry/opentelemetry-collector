# Troubleshooting

The Collector offers multiple ways to measure the health of the Collector
as well as investigate issues:

- The [zpages](https://github.com/open-telemetry/opentelemetry-collector/tree/master/extension/zpagesextension/README.md)
  extension, which by default is on port `55679`, can be used to check
  receivers and exporters trace operations via `/debug/tracez`. `zpages`
  often contains error logs that the collector does not emit.
- By default, Prometheus metrics are exposed on port `8888` and path
  `/metrics`. A grafana dashboard can be found
  [here](https://grafana.com/grafana/dashboards/11575).
- Logs can be helpful in identifying issues. The verbosity level, which
  defaults to `INFO` can also be adjusted by passing the `--log-level` flag
  to the `otelcol` process. See `--help` for more details.
- [Local exporters](https://github.com/open-telemetry/opentelemetry-collector/tree/master/exporter#general-information)
  can be configured to inspect the data being processed by the Collector.
- The [health_check](https://github.com/open-telemetry/opentelemetry-collector/tree/master/extension/healthcheckextension/README.md)
  extension, which by default is on port `13133`, can be used to ensure
  the Collector is functioning properly.
- The [pprof](https://github.com/open-telemetry/opentelemetry-collector/tree/master/extension/pprofextension/README.md)
  extension, which by default is on port `1777`, allows you to profile the
  Collector as it runs.

## Common Issues

### Collector exit/restart

The Collector may exit/restart because:

- Memory pressure due to missing or misconfigured
  [memory_limiter](https://github.com/open-telemetry/opentelemetry-collector/blob/master/processor/memorylimiter/README.md)
  processor.
- [Improperly sized](https://github.com/open-telemetry/opentelemetry-collector/blob/master/docs/performance.md)
  for load.
- Improperly configured (for example, a queue size configured higher
  than available memory).
- Infrastructure resource limits (for example Kubernetes).

### Data being dropped

Data may be dropped for a variety of reasons, but most commonly because of an:

- [Improperly sized Collector](https://github.com/open-telemetry/opentelemetry-collector/blob/master/docs/performance.md) resulting in Collector being unable to process and export the data as fast as it is received.
- Exporter destination unavailable or accepting the data too slowly.

To mitigate drops, it is highly recommended to configure the
[batch](https://github.com/open-telemetry/opentelemetry-collector/blob/master/processor/batchprocessor/README.md)
and
[queued_retry](https://github.com/open-telemetry/opentelemetry-collector/blob/master/processor/queuedprocessor/README.md)
processor.

### Receiving data not working

If you are unable to receive data then this is likely because
either:

- There is a network configuration issue
- The receiver configuration is incorrect
- The client configuration is incorrect

Check the collector logs as well as `zpages` for potential issues.

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
[proxy support](https://github.com/open-telemetry/opentelemetry-collector/tree/master/exporter#proxy-support).
