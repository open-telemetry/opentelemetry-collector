# OTLP Receiver

Receives data via gRPC or HTTP using [OTLP](
https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/protocol/otlp.md)
format.

Supported pipeline types: traces, metrics, logs

:warning: OTLP metrics format is currently marked as "Alpha" and may change in
incompatible way any time.

## Getting Started

All that is required to enable the OTLP receiver is to include it in the
receiver definitions. A protocol can be disabled by simply not specifying it in
the list of protocols.

```yaml
receivers:
  otlp:
    protocols:
      grpc:
      http:
```

The following settings are configurable:

- `endpoint` (default = 0.0.0.0:4317 for grpc protocol, 0.0.0.0:4318 http protocol):
  host:port to which the receiver is going to receive data. The valid syntax is
  described at https://github.com/grpc/grpc/blob/master/doc/naming.md.

## Advanced Configuration

Several helper files are leveraged to provide additional capabilities automatically:

- [gRPC settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configgrpc/README.md) including CORS
- [TLS and mTLS settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configtls/README.md)
- [Queuing, retry and timeout settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/exporter/exporterhelper/README.md)

## Writing with HTTP/JSON

The OTLP receiver can receive trace export calls via HTTP/JSON in addition to
gRPC. The HTTP/JSON address is the same as gRPC as the protocol is recognized
and processed accordingly. Note the format needs to be [protobuf JSON
serialization](https://developers.google.com/protocol-buffers/docs/proto3#json).

To write traces with HTTP/JSON, `POST` to `[address]/v1/traces` for traces,
to `[address]/v1/metrics` for metrics, to `[address]/v1/logs` for logs. The default
port is `4318`.

The HTTP/JSON endpoint can also optionally configure
[CORS](https://fetch.spec.whatwg.org/#cors-protocol), which is enabled by
specifying a list of allowed CORS origins in the `cors_allowed_origins`
and optionally headers in `cors_allowed_headers`:

```yaml
receivers:
  otlp:
    protocols:
      http:
        endpoint: "localhost:4318"
        cors_allowed_origins:
        - http://test.com
        # Origins can have wildcards with *, use * by itself to match any origin.
        - https://*.example.com
        cors_allowed_headers:
        - TestHeader
```
