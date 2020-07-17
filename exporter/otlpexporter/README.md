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
- `timeout` (default = 5s): Is the timeout for every attempt to send data to the backend.
- `retry_on_failure`
  - `disabled` (default = false)
  - `initial_interval` (default = 5s): Time to wait after the first failure before retrying; ignored if `disabled` is `true`
  - `max_interval` (default = 30s): Is the upper bound on backoff; ignored if `disabled` is `true`
  - `max_elapsed_time` (default = 120s): Is the maximum amount of time spent trying to send a batch; ignored if `disabled` is `true`
- `sending_queue`
  - `disabled` (default = true)
  - `num_consumers` (default = 10): Number of consumers that dequeue batches; ignored if `disabled` is `true`
  - `queue_size` (default = 5000): Maximum number of batches kept in memory before data; ignored if `disabled` is `true`;
  User should calculate this as `num_seconds * requests_per_second` where:
    - `num_seconds` is the number of seconds to buffer in case of a backend outage
    - `requests_per_second` is the average number of requests per seconds.

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
