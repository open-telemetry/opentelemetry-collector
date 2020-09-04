# OpenCensus Exporter

Exports traces and/or metrics via gRPC using
[OpenCensus](https://opencensus.io/) format.

The following settings are required:

- `endpoint`: host:port to which the exporter is going to send traces or
  metrics, using the gRPC protocol. The valid syntax is described at
  https://github.com/grpc/grpc/blob/master/doc/naming.md.

The following settings can be optionally configured:

- `insecure` (default = false): whether to enable client transport security for
  the exporter's gRPC connection. See
  [grpc.WithInsecure()](https://godoc.org/google.golang.org/grpc#WithInsecure).
- `ca_file` path to the CA cert. For a client this verifies the server certificate. Should
  only be used if `insecure` is set to false.
- `cert_file` path to the TLS cert to use for TLS required connections. Should
  only be used if `insecure` is set to false.
- `key_file` path to the TLS key to use for TLS required connections. Should
  only be used if `insecure` is set to false.
- `compression` (default = gzip): compression key for supported compression
  types within collector. Currently the only supported mode is `gzip`.
- `headers` the headers associated with gRPC requests.
- `keepalive` keepalive parameters for client gRPC. See
  [grpc.WithKeepaliveParams()](https://godoc.org/google.golang.org/grpc#WithKeepaliveParams).
- `num_workers` (default = 2): number of workers that send the gRPC requests.
  Optional.
- `balancer_name`(default = pick_first): Sets the balancer in grpclb_policy to discover the servers.
See [grpc loadbalancing example](https://github.com/grpc/grpc-go/blob/master/examples/features/load_balancing/README.md).
- `wait_for_ready`: Option not supported.

Example:

```yaml
exporters:
  opencensus:
    endpoint: otelcol2:55678
```

The full list of settings exposed for this exporter are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).
