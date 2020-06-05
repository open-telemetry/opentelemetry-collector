# OTLP Exporter

Exports traces and/or metrics to another Collector via gRPC using OTLP format.

The following settings are required:

- `endpoint`: target to which the exporter is going to send traces or metrics,
using the gRPC protocol. The valid syntax is described at
https://github.com/grpc/grpc/blob/master/doc/naming.md.

The following settings can be optionally configured:

- `cert_pem_file`: certificate file for TLS credentials of gRPC client. Should
only be used if `secure` is set to true.
- `compression`: compression key for supported compression types within
collector. Currently the only supported mode is `gzip`.
- `headers`: the headers associated with gRPC requests.
- `keepalive`: keepalive parameters for client gRPC. See
[grpc.WithKeepaliveParams()](https://godoc.org/google.golang.org/grpc#WithKeepaliveParams).
- `num_workers` (default = 2): number of workers that send the gRPC requests. Optional.
- `reconnection_delay`: time period between each reconnection performed by the
exporter.
- `secure`: whether to enable client transport security for the exporter's gRPC
connection. See [grpc.WithInsecure()](https://godoc.org/google.golang.org/grpc#WithInsecure).

Example:

```yaml
exporters:
  otlp:
    endpoint: localhost:14250
    reconnection_delay: 60s
    secure: false
```

The full list of settings exposed for this exporter are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).
