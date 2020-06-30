# OpenCensus Receiver

This receiver receives trace and metrics from [OpenCensus](https://opencensus.io/)
instrumented applications.

To get started, all that is required to enable the OpenCensus receiver is to
include it in the receiver definitions. This will enable the default values as
specified [here](./factory.go).
The following is an example:
```yaml
receivers:
  opencensus:
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
    endpoint: "0.0.0.0:55678"
    cors_allowed_origins:
    - http://test.com
    # Origins can have wildcards with *, use * by itself to match any origin.
    - https://*.example.com
```
