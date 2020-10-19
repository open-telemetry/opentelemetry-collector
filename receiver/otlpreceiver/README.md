# OpenTelemetry Receiver

Receives traces and/or metrics via gRPC using
[OpenTelemetry](https://opentelemetry.io/) format.

To get started, all that is required to enable the OpenTelemetry receiver is to
include it in the receiver definitions.

The following settings are required:

- `endpoint` (default = 0.0.0.0:55680): host:port to which the receiver is
  going to receive traces or metrics, using the gRPC protocol. The valid syntax
  is described at https://github.com/grpc/grpc/blob/master/doc/naming.md.
- `transport` (default = tcp): which transport to use between `tcp` and `unix`.

The following settings are optional:

- `cors_allowed_origins` (default = unset): allowed CORS origins for HTTP/JSON
  requests. See the HTTP/JSON section below.
- `keepalive`: see
  https://godoc.org/google.golang.org/grpc/keepalive#ServerParameters for more
  information
  - `MaxConnectionIdle` (default = infinity)
  - `MaxConnectionAge` (default = infinity)
  - `MaxConnectionAgeGrace` (default = infinity)
  - `Time` (default = 2h)
  - `Timeout` (default = 20s)
- `max_recv_msg_size_mib` (default = 4MB): sets the maximum size of messages accepted
- `max_concurrent_streams`: sets the limit on the number of concurrent streams
- `tls_credentials` (default = unset): configures the receiver to use TLS. See
  TLS section below.

Examples:

```yaml
receivers:
  otlp:
    protocols:
        grpc:
  otlp/withendpoint:
    protocols:
        grpc:
            endpoint: 127.0.0.1:55680
```

The full list of settings exposed for this receiver are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).

A protocol can be disabled by simply not specifying it in the list of protocols:
```yaml
receivers:
  otlp/only_grpc:
    protocols:
      grpc:
```

## Communicating over TLS
This receiver supports communication using Transport Layer Security (TLS). TLS
can be configured by specifying a `tls_settings` object in the receiver
configuration for receivers that support it.
```yaml
receivers:
  otlp:
    protocols:
      grpc:
        tls_settings:
          key_file: /key.pem # path to private key
          cert_file: /cert.pem # path to certificate
```

## Writing with HTTP/JSON
The OpenTelemetry receiver can receive trace export calls via HTTP/JSON in
addition to gRPC. The HTTP/JSON address is the same as gRPC as the protocol is
recognized and processed accordingly. Note the format needs to be [protobuf JSON
serialization](https://developers.google.com/protocol-buffers/docs/proto3#json).

IMPORTANT: bytes fields are encoded as base64 strings.

To write traces with HTTP/JSON, `POST` to `[address]/v1/traces` for traces,
to `[address]/v1/metrics` for metrics, to `[address]/v1/logs` for logs. The default
port is `55681`.

The HTTP/JSON endpoint can also optionally configure
[CORS](https://fetch.spec.whatwg.org/#cors-protocol), which is enabled by
specifying a list of allowed CORS origins in the `cors_allowed_origins` field:

```yaml
receivers:
  otlp:
    protocols:
      http:
        endpoint: "localhost:55681"
        cors_allowed_origins:
        - http://test.com
        # Origins can have wildcards with *, use * by itself to match any origin.
        - https://*.example.com
```
