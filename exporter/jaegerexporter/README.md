# Jaeger gRPC Exporter

Exports data via gRPC to [Jaeger](https://www.jaegertracing.io/) destinations.
By default, this exporter requires TLS and offers queued retry capabilities. A
Jaeger Thrift HTTP exporter is available in the [contrib
repository](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/master/exporter/jaegerthrifthttpexporter).

Supported pipeline types: traces

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
  jaeger:
    endpoint: jaeger-all-in-one:14250
    cert_file: file.cert
    key_file: file.key
  jaeger/2:
    endpoint: jaeger-all-in-one:14250
    insecure: true
```

## Advanced Configuration

Several helper files are leveraged to provide additional capabilities automatically:

- [gRPC settings](https://github.com/open-telemetry/opentelemetry-collector/blob/master/config/configgrpc/README.md)
- [TLS and mTLS settings](https://github.com/open-telemetry/opentelemetry-collector/blob/master/config/configtls/README.md)
- [Queuing, retry and timeout settings](https://github.com/open-telemetry/opentelemetry-collector/blob/master/exporter/exporterhelper/README.md)
