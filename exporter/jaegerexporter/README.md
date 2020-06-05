# Jaeger Exporter

Exports trace data to [Jaeger](https://www.jaegertracing.io/) collectors. The following settings are required:

- `endpoint` (no default): target to which the exporter is going to send Jaeger trace data,
using the gRPC protocol. The valid syntax is described at
https://github.com/grpc/grpc/blob/master/doc/naming.md

The following settings can be optionally configured:

- `cert_pem_file`: certificate file for TLS credentials of gRPC client. Should
only be used if `insecure` is set to false.
- `keepalive`: keepalive parameters for client gRPC. See
[grpc.WithKeepaliveParams()](https://godoc.org/google.golang.org/grpc#WithKeepaliveParams).
- `insecure`: whether to disable client transport security for the exporter's gRPC
connection. See [grpc.WithInsecure()](https://godoc.org/google.golang.org/grpc#WithInsecure).
- `server_name_override`: If set to a non empty string, it will override the virtual host name 
of authority (e.g. :authority header field) in requests (typically used for testing).

Example:

```yaml
exporters:
  jaeger:
    endpoint: jaeger-all-in-one:14250
    cert_pem_file: /my-cert.pem
    server_name_override: opentelemetry.io
```

The full list of settings exposed for this exporter are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).
