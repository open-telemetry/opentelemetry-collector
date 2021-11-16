# gRPC Configuration Settings

gRPC exposes a [variety of settings](https://godoc.org/google.golang.org/grpc).
Several of these settings are available for configuration within individual
receivers or exporters. In general, none of these settings should need to be
adjusted.

## Client Configuration

[Exporters](https://github.com/open-telemetry/opentelemetry-collector/blob/main/exporter/README.md)
leverage client configuration.

Note that client configuration supports TLS configuration, the
configuration parameters are also defined under `tls` like server
configuration. For more information, see [configtls
README](../configtls/README.md).

- [`balancer_name`](https://github.com/grpc/grpc-go/blob/master/examples/features/load_balancing/README.md)
- `compression` Compression type to use among `gzip`, `snappy` and `zstd`
- `endpoint`: Valid value syntax available [here](https://github.com/grpc/grpc/blob/master/doc/naming.md)
- [`tls`](../configtls/README.md)
- `headers`: name/value pairs added to the request
- [`keepalive`](https://godoc.org/google.golang.org/grpc/keepalive#ClientParameters)
  - `permit_without_stream`
  - `time`
  - `timeout`
- [`read_buffer_size`](https://godoc.org/google.golang.org/grpc#ReadBufferSize)
- [`write_buffer_size`](https://godoc.org/google.golang.org/grpc#WriteBufferSize)

Please note that [`per_rpc_auth`](https://pkg.go.dev/google.golang.org/grpc#PerRPCCredentials) which allows the credentials to send for every RPC is now moved to become an [extension](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/extension/bearertokenauthextension). Note that this feature isn't about sending the headers only during the initial connection as an `authorization` header under the `headers` would do: this is sent for every RPC performed during an established connection.

Example:

```yaml
exporters:
  otlp:
    endpoint: otelcol2:55690
    tls:
      ca_file: ca.pem
      cert_file: cert.pem
      key_file: key.pem
    headers:
      test1: "value1"
      "test 2": "value 2"
```

## Server Configuration

[Receivers](https://github.com/open-telemetry/opentelemetry-collector/blob/main/receiver/README.md)
leverage server configuration.

Note that transport configuration can also be configured. For more information,
see [confignet README](../confignet/README.md).

- [`keepalive`](https://godoc.org/google.golang.org/grpc/keepalive#ServerParameters)
  - [`enforcement_policy`](https://godoc.org/google.golang.org/grpc/keepalive#EnforcementPolicy)
    - `min_time`
    - `permit_without_stream`
  - [`server_parameters`](https://godoc.org/google.golang.org/grpc/keepalive#ServerParameters)
    - `max_connection_age`
    - `max_connection_age_grace`
    - `max_connection_idle`
    - `time`
    - `timeout`
- [`max_concurrent_streams`](https://godoc.org/google.golang.org/grpc#MaxConcurrentStreams)
- [`max_recv_msg_size_mib`](https://godoc.org/google.golang.org/grpc#MaxRecvMsgSize)
- [`read_buffer_size`](https://godoc.org/google.golang.org/grpc#ReadBufferSize)
- [`tls`](../configtls/README.md)
- [`write_buffer_size`](https://godoc.org/google.golang.org/grpc#WriteBufferSize)
