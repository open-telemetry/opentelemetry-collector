# OpenCensus Receiver

Receives traces and/or metrics via gRPC using
[OpenCensus](https://opencensus.io/) format.

To get started, all that is required to enable the OpenCensus receiver is to
include it in the receiver definitions.

The following settings are required:

- `endpoint` (default = 0.0.0.0:55678): host:port to which the exporter is
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
- `max_recv_msg_size_mib` (default = infinity): sets the maximum size of messages accepted
- `max_concurrent_streams`: sets the limit on the number of concurrent streams
- `tls_credentials` (default = unset): configures the receiver to use TLS. See
  TLS section below.

Examples:

```yaml
receivers:
  opencensus:
  opencensus/withendpoint:
    endpoint: 127.0.0.1:55678
```

The full list of settings exposed for this receiver are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).

## Communicating over TLS
This receiver supports communication using Transport Layer Security (TLS). TLS
can be configured by specifying a `tls_settings` object in the receiver
configuration for receivers that support it.
```yaml
receivers:
  opencensus:
    tls_settings:
      key_file: /key.pem # path to private key
      cert_file: /cert.pem # path to certificate
```

## Writing with HTTP/JSON
The OpenCensus receiver can receive trace export calls via HTTP/JSON in
addition to gRPC. The HTTP/JSON address is the same as gRPC as the protocol is
recognized and processed accordingly.

To write traces with HTTP/JSON, `POST` to `[address]/v1/trace`. The JSON message
format parallels the gRPC protobuf format, see this
[OpenApi spec for it](https://github.com/census-instrumentation/opencensus-proto/blob/master/gen-openapi/opencensus/proto/agent/trace/v1/trace_service.swagger.json).

The HTTP/JSON endpoint can also optionally configure
[CORS](https://fetch.spec.whatwg.org/#cors-protocol), which is enabled by
specifying a list of allowed CORS origins in the `cors_allowed_origins` field:

```yaml
receivers:
  opencensus:
    cors_allowed_origins:
    - http://test.com
    # Origins can have wildcards with *, use * by itself to match any origin.
    - https://*.example.com
```
