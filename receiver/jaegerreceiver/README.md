# Jaeger Receiver

Receives trace data in [Jaeger](https://www.jaegertracing.io/) format.

Supported pipeline types: traces

## Getting Started

By default, the Jaeger receiver will not serve any protocol. A protocol must be
named under the `protocols` object for the jaeger receiver to start. The
below protocols are supported, each supports an optional `endpoint`
object configuration parameter.

- `grpc` (default `endpoint` = 0.0.0.0:14250)
- `thrift_binary` (default `endpoint` = 0.0.0.0:6832)
- `thrift_compact` (default `endpoint` = 0.0.0.0:6831)
- `thrift_http` (default `endpoint` = 0.0.0.0:14268)

Examples:

```yaml
receivers:
  jaeger:
    protocols:
      grpc:
  jaeger/withendpoint:
    protocols:
      grpc:
        endpoint: 0.0.0.0:14260
```

## Advanced Configuration

UDP protocols (currently `thrift_binary` and `thrift_compact`) allow setting additional 
server options:

- `queue_size` (default 1000) sets max not yet handled requests to server
- `max_packet_size` (default 65_000) sets max UDP packet size
- `workers` (default 10) sets number of workers consuming the server queue
- `socket_buffer_size` (default 0 - no buffer) sets buffer size of connection socket in bytes

Examples:

```yaml
protocols:
  thrift_binary:
    endpoint: 0.0.0.0:6832
    queue_size: 5_000
    max_packet_size: 131_072
    workers: 50
    socket_buffer_size: 8_388_608
```

Several helper files are leveraged to provide additional capabilities automatically:

- [gRPC settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configgrpc/README.md) including CORS
- [TLS and mTLS settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configtls/README.md)

## Remote Sampling

The Jaeger receiver also supports fetching sampling configuration from a remote
collector. It works by proxying client requests for remote sampling
configuration to the configured collector.

        +------------+                   +-----------+              +---------------+
        |            |       get         |           |    proxy     |               |
        |   client   +---  sampling ---->+   agent   +------------->+   collector   |
        |            |     strategy      |           |              |               |
        +------------+                   +-----------+              +---------------+

Remote sample proxying can be enabled by specifying the following lines in the
jaeger receiver config:

```yaml
receivers:
  jaeger:
    protocols:
      grpc:
    remote_sampling:
      endpoint: "jaeger-collector:14250"
      insecure: true
```

Remote sampling can also be directly served by the collector by providing a
sampling json file:

```yaml
receivers:
  jaeger:
    protocols:
      grpc:
    remote_sampling:
      strategy_file: "/etc/strategy.json"
```

Note: the `grpc` protocol must be enabled for this to work as Jaeger serves its
remote sampling strategies over gRPC.
