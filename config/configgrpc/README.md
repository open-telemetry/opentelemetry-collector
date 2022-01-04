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
- `compression` Compression type to use among `gzip`, `snappy`, `zstd`, and `none`. Defaults to `gzip`. See compression comparison below for more details.
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

### Compression Comparison

[configgrpc_benchmark_test.go](./configgrpc_benchmark_test.go) contains benchmarks comparing the supported compression algorithms. It performs compression using `gzip`, `zstd`, and `snappy` compression on small, medium, and large sized log, trace, and metric payloads. Each test case outputs the uncompressed payload size, the compressed payload size, and the average nanoseconds spent on compression. 

The following table summarizes the results, including some additional columns computed from the raw data. Compression ratios will vary in practice as they are highly dependent on the data's information entropy. Compression rates are dependent on the speed of the CPU, and the size of payloads being compressed: smaller payloads compress at slower rates relative to larger payloads, which are able to amortize fixed computation costs over more bytes. 

The default `gzip` compression is not as fast as snappy, but achieves better compression ratios and has reasonable performance. It's also the only required compression algorithm for [OTLP servers](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/protocol/otlp.md#protocol-details). If your collector is CPU bound and your OTLP server supports it, you may benefit from using `snappy` compression. If your collector is CPU bound and has a very fast network link, you may benefit from disabling compression with `none`. 

| Request           | Compressor | Raw Bytes | Compressed bytes | Compression ratio | Ns / op | Mb compressed / second | Mb saved / second |
|-------------------|------------|-----------|------------------|-------------------|---------|------------------------|-------------------|
| lg_log_request    | gzip       | 5150      | 262              | 19.66             | 51371   | 100.25                 | 95.15             |
| lg_metric_request | gzip       | 6800      | 201              | 33.83             | 53983   | 125.97                 | 122.24            |
| lg_trace_request  | gzip       | 9200      | 270              | 34.07             | 60681   | 151.61                 | 147.16            |
| md_log_request    | gzip       | 363       | 268              | 1.35              | 40763   | 8.91                   | 2.33              |
| md_metric_request | gzip       | 320       | 145              | 2.21              | 36724   | 8.71                   | 4.77              |
| md_trace_request  | gzip       | 451       | 288              | 1.57              | 41440   | 10.88                  | 3.93              |
| sm_log_request    | gzip       | 166       | 168              | 0.99              | 39879   | 4.16                   | -0.05             |
| sm_metric_request | gzip       | 185       | 142              | 1.30              | 37546   | 4.93                   | 1.15              |
| sm_trace_request  | gzip       | 233       | 205              | 1.14              | 36069   | 6.46                   | 0.78              |
| lg_log_request    | snappy     | 5150      | 475              | 10.84             | 1494    | 3,447.12               | 3,129.18          |
| lg_metric_request | snappy     | 6800      | 466              | 14.59             | 1738    | 3,912.54               | 3,644.42          |
| lg_trace_request  | snappy     | 9200      | 644              | 14.29             | 2510    | 3,665.34               | 3,408.76          |
| md_log_request    | snappy     | 363       | 300              | 1.21              | 615.5   | 589.76                 | 102.36            |
| md_metric_request | snappy     | 320       | 162              | 1.98              | 457.7   | 699.15                 | 345.20            |
| md_trace_request  | snappy     | 451       | 330              | 1.37              | 698.7   | 645.48                 | 173.18            |
| sm_log_request    | snappy     | 166       | 184              | 0.90              | 443.6   | 374.21                 | -40.58            |
| sm_metric_request | snappy     | 185       | 154              | 1.20              | 397.9   | 464.94                 | 77.91             |
| sm_trace_request  | snappy     | 233       | 251              | 0.93              | 526.0   | 442.97                 | -34.22            |
| lg_log_request    | zstd       | 5150      | 223              | 23.09             | 15810   | 325.74                 | 311.64            |
| lg_metric_request | zstd       | 6800      | 144              | 47.22             | 11134   | 610.74                 | 597.81            |
| lg_trace_request  | zstd       | 9200      | 208              | 44.23             | 13696   | 671.73                 | 656.54            |
| md_log_request    | zstd       | 363       | 261              | 1.39              | 10258   | 35.39                  | 9.94              |
| md_metric_request | zstd       | 320       | 145              | 2.21              | 8147    | 39.28                  | 21.48             |
| md_trace_request  | zstd       | 451       | 301              | 1.50              | 11380   | 39.63                  | 13.18             |
| sm_log_request    | zstd       | 166       | 165              | 1.01              | 11553   | 14.37                  | 0.09              |
| sm_metric_request | zstd       | 185       | 139              | 1.33              | 7645    | 24.20                  | 6.02              |
| sm_trace_request  | zstd       | 233       | 203              | 1.15              | 9334    | 24.96                  | 3.21              |


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
