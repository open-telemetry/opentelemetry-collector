# OpenCensus Exporter

Exports traces and/or metrics via gRPC using
[OpenCensus](https://opencensus.io/) format.

The following settings are required:

- `endpoint`: host:port to which the exporter is going to send traces or
  metrics, using the gRPC protocol. The valid syntax is described at
  https://github.com/grpc/grpc/blob/master/doc/naming.md.

The following settings can be optionally configured:

- `cert_pem_file`: certificate file for TLS credentials of gRPC client. Should
  only be used if `secure` is set to true.
- `compression` (default = gzip): compression key for supported compression
  types within collector. Currently the only supported mode is `gzip`.
- `headers`: the headers associated with gRPC requests.
- `insecure` (default = false): whether to enable client transport security for
  the exporter's gRPC connection. See
  [grpc.WithInsecure()](https://godoc.org/google.golang.org/grpc#WithInsecure).
- `keepalive`: keepalive parameters for client gRPC. See
  [grpc.WithKeepaliveParams()](https://godoc.org/google.golang.org/grpc#WithKeepaliveParams).
- `num_workers` (default = 2): number of workers that send the gRPC requests.
  Optional.
- `reconnection_delay` (default = unset): time period between each reconnection
  performed by the exporter.
- `balancer_name`(default = pick_first): Sets the balancer in grpclb_policy to discover the servers.
See [grpc loadbalancing example](https://github.com/grpc/grpc-go/blob/master/examples/features/load_balancing/README.md).

Example:

```yaml
exporters:
  opencensus:
    endpoint: otelcol2:55678
    reconnection_delay: 60s
```

The full list of settings exposed for this exporter are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).
