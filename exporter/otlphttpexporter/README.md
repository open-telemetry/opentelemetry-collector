# OTLP/HTTP Exporter

Exports traces and/or metrics via HTTP using [OTLP](
https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/protocol/otlp.md)
format.

*Important: OTLP metrics format is currently marked as "Alpha" and may change in
incompatible way any time.*

The following settings are required:

- `endpoint` (no default): The target base URL to send data to (e.g.: https://example.com:4318).
  To send each signal a corresponding path will be added to this base URL, i.e. for traces
  "/v1/traces" will appended, for metrics "/v1/metrics" will be appended, for logs
  "/v1/logs" will be appended. 

The following settings can be optionally configured:

- `traces_endpoint` (no default): The target URL to send trace data to (e.g.: https://example.com:4318/v1/traces).
   If this setting is present the `endpoint` setting is ignored for traces.
- `metrics_endpoint` (no default): The target URL to send metric data to (e.g.: https://example.com:4318/v1/metrics).
   If this setting is present the `endpoint` setting is ignored for metrics.
- `logs_endpoint` (no default): The target URL to send log data to (e.g.: https://example.com:4318/v1/logs).
   If this setting is present the `endpoint` setting is ignored logs.

- `insecure` (default = false): when set to true disables verifying the server's
  certificate chain and host name. The connection is still encrypted but server identity
  is not verified.
- `ca_file` path to the CA cert. For a client this verifies the server certificate. Should
  only be used if `insecure` is set to false.
- `cert_file` path to the TLS cert to use for TLS required connections. Should
  only be used if `insecure` is set to false.
- `key_file` path to the TLS key to use for TLS required connections. Should
  only be used if `insecure` is set to false.

- `compression` (default = none): Compression type to use (only gzip is supported today)

- `timeout` (default = 30s): HTTP request time limit. For details see https://golang.org/pkg/net/http/#Client
- `read_buffer_size` (default = 0): ReadBufferSize for HTTP client.
- `write_buffer_size` (default = 512 * 1024): WriteBufferSize for HTTP client.


Example:

```yaml
exporters:
  otlphttp:
    endpoint: https://example.com:4318/v1/traces
```

The full list of settings exposed for this exporter are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).
