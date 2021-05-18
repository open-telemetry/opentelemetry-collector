# Prometheus Remote Write Exporter

Prometheus Remote Write Exporter sends OpenTelemetry metrics
to Prometheus [remote write compatible
backends](https://prometheus.io/docs/operating/integrations/)
such as Cortex and Thanos.
By default, this exporter requires TLS and offers queued retry capabilities.

Supported pipeline types: metrics

:warning: Non-cumulative monotonic, histogram, and summary OTLP metrics are
dropped by this exporter.

A [design doc](./DESIGN.md) is available to document in detail
how this exporter works.

## Getting Started

The following settings are required:

- `endpoint` (no default): The remote write URL to send remote write samples.

By default, TLS is enabled:

- `insecure` (default = `false`): whether to enable client transport security for
  the exporter's connection.

As a result, the following parameters are also required:

- `cert_file` (no default): path to the TLS cert to use for TLS required connections. Should
  only be used if `insecure` is set to false.
- `key_file` (no default): path to the TLS key to use for TLS required connections. Should
  only be used if `insecure` is set to false.

The following settings can be optionally configured:

- `external_labels`: list of labels to be attached to each metric data point
- `headers`: additional headers attached to each HTTP request. 
  - *Note the following headers cannot be changed: `Content-Encoding`, `Content-Type`, `X-Prometheus-Remote-Write-Version`, and `User-Agent`.*
- `namespace`: prefix attached to each exported metric name.
- `remote_write_queue`: fine tuning for queueing and sending of the outgoing remote writes.
  - `queue_size`: number of OTLP metrics that can be queued.
  - `num_consumers`: minimum number of workers to use to fan out the outgoing requests.

Example:

```yaml
exporters:
  prometheusremotewrite:
    endpoint: "https://my-cortex:7900/api/v1/push"
```

## Advanced Configuration

Several helper files are leveraged to provide additional capabilities automatically:

- [HTTP settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/confighttp/README.md)
- [TLS and mTLS settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configtls/README.md)
- [Retry and timeout settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/exporter/exporterhelper/README.md), note that the exporter doesn't support `sending_queue` but provides `remote_write_queue`.
