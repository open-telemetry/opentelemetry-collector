# OpenTelemetry Receiver

This is the default receiver for the OpenTelemetry project.

To get started, all that is required to enable the OpenTelemetry receiver is to
include it in the receiver definitions and defined the enabled protocols. This will
enable the default values as specified [here](./factory.go).
The following is an example:
```yaml
receivers:
  otlp:
    protocols:
      grpc:
      http:
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
can be configured by specifying a `tls_credentials` object in the receiver
configuration for receivers that support it.
```yaml
receivers:
  otlp:
    protocols:
      grpc:
        tls_credentials:
          key_file: /key.pem # path to private key
          cert_file: /cert.pem # path to certificate
```

## Writing with HTTP/JSON
The OpenTelemetry receiver can receive trace export calls via HTTP/JSON in
addition to gRPC. The HTTP/JSON address is the same as gRPC as the protocol is
recognized and processed accordingly. Note the format needs to be [protobuf
JSON
serialization](https://developers.google.com/protocol-buffers/docs/proto3#json).

IMPORTANT: bytes fields are encoded as base64 strings.

To write traces with HTTP/JSON, `POST` to
`[address]/v1/trace`.

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
