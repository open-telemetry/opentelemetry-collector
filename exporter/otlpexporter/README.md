# OTLP gRPC Exporter

Exports data via gRPC using [OTLP](
https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/protocol/otlp.md)
format. By default, this exporter requires TLS and offers queued retry capabilities.

Supported pipeline types: traces, metrics, logs

:warning: OTLP logs format is currently marked as "Beta" and may change in
incompatible ways.

## Getting Started

The following settings are required:

- `endpoint` (no default): host:port to which the exporter is going to send OTLP trace data,
using the gRPC protocol. The valid syntax is described
[here](https://github.com/grpc/grpc/blob/master/doc/naming.md).
If a scheme of `https` is used then client transport security is enabled and overrides the `insecure` setting.

By default, TLS is enabled:

- `tls:`

  - `insecure` (default = `false`): whether to enable client transport security for the exporter's connection.

As a result, the following parameters are also required:

- `tls:`

  - `cert_file` (no default): path to the TLS cert to use for TLS required connections. Should only be used if `insecure` is set to false.
  - `key_file` (no default): path to the TLS key to use for TLS required connections. Should only be used if `insecure` is set to false.

Example:

```yaml
exporters:
  otlp:
    endpoint: otelcol2:4317
    tls:
      cert_file: file.cert
      key_file: file.key
  otlp/2:
    endpoint: otelcol2:4317
    tls:
      insecure: true
```

## Advanced Configuration

Several helper files are leveraged to provide additional capabilities automatically:

- [gRPC settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configgrpc/README.md)
- [TLS and mTLS settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configtls/README.md)
- [Queuing, retry and timeout settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/exporter/exporterhelper/README.md)
