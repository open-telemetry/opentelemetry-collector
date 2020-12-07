# Prometheus Remote Write Exporter

This exporter sends data in Prometheus TimeSeries format to Cortex or any
Prometheus [remote write compatible
backend](https://prometheus.io/docs/operating/integrations/).
By default, this exporter requires TLS and offers queued retry capabilities.

:warning: Non-cumulative monotonic, histogram, and summary OTLP metrics are
dropped by this exporter.

_Here is a link to the overall project [design](./DESIGN.md)_

Supported pipeline types: metrics

## Getting Started

The following settings are required:

- `endpoint` (no default): protocol:host:port to which the exporter is going to send data.

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

Example:

```yaml
exporters:
  prometheusremotewrite:
    endpoint: "http://some.url:9411/api/prom/push"
```

## Advanced Configuration

Several helper files are leveraged to provide additional capabilities automatically:

- [HTTP settings](https://github.com/open-telemetry/opentelemetry-collector/blob/master/config/confighttp/README.md)
- [TLS and mTLS settings](https://github.com/open-telemetry/opentelemetry-collector/blob/master/config/configtls/README.md)
- [Queuing, retry and timeout settings](https://github.com/open-telemetry/opentelemetry-collector/blob/master/exporter/exporterhelper/README.md)
