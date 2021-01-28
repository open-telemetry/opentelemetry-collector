# OpenCensus gRPC Exporter

Exports traces and/or metrics via gRPC using
[OpenCensus](https://opencensus.io/) format.

Supported pipeline types: traces, metrics

## Getting Started

The following settings are required:

- `endpoint` (no default): host:port to which the exporter is going to send Jaeger trace data,
using the gRPC protocol. The valid syntax is described
[here](https://github.com/grpc/grpc/blob/master/doc/naming.md)

By default, TLS is enabled:

- `insecure` (default = `false`): whether to enable client transport security for
  the exporter's connection.

As a result, the following parameters are also required:

- `cert_file` (no default): path to the TLS cert to use for TLS required connections. Should
  only be used if `insecure` is set to false.
- `key_file` (no default): path to the TLS key to use for TLS required connections. Should
  only be used if `insecure` is set to false.

Example:

```yaml
exporters:
  opencensus:
    endpoint: opencensus2:55678
    cert_file: file.cert
    key_file: file.key
  otlp/2:
    endpoint: opencensus2:55678
    insecure: true
```

## Advanced Configuration

Several helper files are leveraged to provide additional capabilities automatically:

- [gRPC settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configgrpc/README.md)
- [TLS and mTLS settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configtls/README.md)
- [Queuing, retry and timeout settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/exporter/exporterhelper/README.md)
