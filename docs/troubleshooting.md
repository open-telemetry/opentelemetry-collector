# Troubleshooting

The Collector offers multiple ways to measure the health of the Collector
as well as investigate issues.

- By default, Prometheus metrics are exposed on port `8888` and path
`/metrics`. A grafana dashboard can be found
[here](https://grafana.com/grafana/dashboards/11575).

- Logs can be helpful in identifying issues. The verbosity level, which
defaults to `INFO` can also be adjusted by passing the `--log-level` flag
to the `otelcol` process. See `--help` for more details.

- [Local exporters](https://github.com/open-telemetry/opentelemetry-collector/tree/master/exporter#local-exporters)
can be configured to inspect the data being processed by the Collector.

- The [health_check](https://github.com/open-telemetry/opentelemetry-collector/blob/master/extension/README.md#health_check)
extension, which by default is on port `13133`, can be used to ensure
the Collector is functioning properly.

- The [zpages](https://github.com/open-telemetry/opentelemetry-collector/blob/master/extension/README.md#zpages)
extension, which by default is on port `55679`, can be used to check
receivers and exporters trace operations via `/debug/tracez`.

- The [pprof](https://github.com/open-telemetry/opentelemetry-collector/blob/master/extension/README.md#pprof)
extension, which by default is on port `1777`, allows you to profile the
Collector as it runs.

## Common Issues

### Collector exit/restart

The Collector may exit/restart if it is not
[sized properly](https://github.com/open-telemetry/opentelemetry-collector/blob/master/docs/performance.md)
or configured properly, for example a queue size configured higher than
available memory. In most cases, restarts are due to memory
pressure. To mitigate this, we recommend you configure the
[memory_limiter](https://github.com/open-telemetry/opentelemetry-collector/tree/master/processor#memory-limiter)
processor.

Note: restarts may be due to resource limits configured on the Collector.

### Data being dropped

Data may be dropped for a variety of reasons, but most commonly because of an:

- [Improperly sized Collector](https://github.com/open-telemetry/opentelemetry-collector/blob/master/docs/performance.md)) resulting in Collector being unable to process and export the data as fast as it is received.
- Exporter destination unavailable or accepting the data too slowly.

To mitigate drops, it is highly recommended to configure the
[queued_retry](https://github.com/open-telemetry/opentelemetry-collector/tree/master/processor#queued-retry)
processor.

### Receiving data not working

If you are unable to receive data then this is likely because
either:

- There is a network configuration issue
- The receiver configuration is incorrect
- The client configuration is incorrect

**IMPORTANT:** For containerized environments, you will need to manually set the
receiver address to `0.0.0.0` in order to receive data.

### Exporting data not working

If you are unable to export to a destination then this is likely because
either:

- There is a network configuration issue
- The exporter configuration is incorrect
- The destination is unavailable

More often than not, exporting data does not work because of a network
configuration issue. This could be due to a firewall, DNS, or proxy
issue. Note that the Collector does have
[proxy support](https://github.com/open-telemetry/opentelemetry-collector/tree/master/exporter#proxy-support).
