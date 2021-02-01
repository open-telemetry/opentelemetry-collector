# Zipkin Exporter

Exports data to a [Zipkin](https://zipkin.io/) back-end.
By default, this exporter requires TLS and offers queued retry capabilities.

Supported pipeline types: traces

## Getting Started

The following settings are required:

- `endpoint` (no default): URL to which the exporter is going to send Zipkin trace data.
- `format` (default = `JSON`): The format to sent events in. Can be set to `JSON` or `proto`.

By default, TLS is enabled:

- `insecure` (default = `false`): whether to enable client transport security for
  the exporter's connection.

As a result, the following parameters are also required:

- `cert_file` (no default): path to the TLS cert to use for TLS required connections. Should
  only be used if `insecure` is set to false.
- `key_file` (no default): path to the TLS key to use for TLS required connections. Should
  only be used if `insecure` is set to false.

The following settings are optional:

- `defaultservicename` (default = `<missing service name>`): What to name
  services missing this information.

Example:

```yaml
exporters:
  zipkin:
    endpoint: "http://some.url:9411/api/v2/spans"
    cert_file: file.cert
    key_file: file.key
  zipkin/2:
    endpoint: "http://some.url:9411/api/v2/spans"
    insecure: true
```

## Advanced Configuration

Several helper files are leveraged to provide additional capabilities automatically:

- [HTTP settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/confighttp/README.md)
- [TLS and mTLS settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configtls/README.md)
- [Queuing, retry and timeout settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/exporter/exporterhelper/README.md)
