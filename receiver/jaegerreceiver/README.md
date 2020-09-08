# Jaeger Receiver

Receives trace data in [Jaeger](https://www.jaegertracing.io/) format.

By default, the Jaeger receiver will not serve any protocol. A protocol must be
named under the `protocols` object for the jaeger receiver to start. The
below protocols are supported and each supports an optional `endpoint`
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

The full list of settings exposed for this receiver are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).

## Communicating over TLS
The Jaeger receiver supports communication using Transport Layer Security (TLS), but
only using the `grpc` protocol. It can be configured by specifying a
`tls_credentials` object in the `grpc` object receiver configuration.
```yaml
receivers:
  jaeger:
    protocols:
      grpc:
        tls_credentials:
          key_file: /key.pem # path to private key
          cert_file: /cert.pem # path to certificate
```

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
      fetch_endpoint: "jaeger-collector:1234"
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
