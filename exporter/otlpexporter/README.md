# OTLP Exporter

Exports traces and/or metrics via gRPC using
[OpenTelemetry](https://opentelemetry.io/) format.

The following settings are required:

- `endpoint`: host:port to which the exporter is going to send traces or
  metrics, using the gRPC protocol. The valid syntax is described at
  https://github.com/grpc/grpc/blob/master/doc/naming.md.

The following settings can be optionally configured:

- `cert_pem_file`: certificate file for TLS credentials of gRPC client. Should
  only be used if `insecure` is set to `false`.
- `compression`: compression key for supported compression types within
  collector. Currently the only supported mode is `gzip`.
- `headers`: the headers associated with gRPC requests.
- `insecure` (default = false): whether to enable client transport security for
  the exporter's gRPC connection. See
  [grpc.WithInsecure()](https://godoc.org/google.golang.org/grpc#WithInsecure).
- `keepalive`: keepalive parameters for client gRPC. See
  [grpc.WithKeepaliveParams()](https://godoc.org/google.golang.org/grpc#WithKeepaliveParams).
- `reconnection_delay`: time period between each reconnection performed by the
  exporter.
- `insecure`: whether to enable client transport security for the exporter's
  gRPC connection. See
  [grpc.WithInsecure()](https://godoc.org/google.golang.org/grpc#WithInsecure).
- `balancer_name`(default = pick_first): Sets the balancer in grpclb_policy to discover the servers.
See [grpc loadbalancing example](https://github.com/grpc/grpc-go/blob/master/examples/features/load_balancing/README.md).
- `timeout` (default = 5s): Is the timeout for each operation.
- `retry_settings`
  - `disabled` (default = false)
  - `initial_backoff` (default = 5s): Time to wait after the first failure before retrying; ignored if retry_on_failure is false
  - `max_backoff` (default = 30s): Is the upper bound on backoff; ignored if retry_on_failure is false
  - `max_elapsed_time` (default = 120s): Is the maximum amount of time spent trying to send a batch; ignored if retry_on_failure is false
- `queued_settings`
  - `disabled` (default = true)
  - `num_workers` (default = 10): Number of workers that dequeue batches
  - `queue_size` (default = 5000): Maximum number of batches kept in memory before data

Example:

```yaml
exporters:
  otlp:
    endpoint: otelcol2:55680
    reconnection_delay: 60s
    insecure: true
```

The full list of settings exposed for this exporter are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).
