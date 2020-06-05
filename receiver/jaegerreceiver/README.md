# Jaeger Receiver

This receiver supports [Jaeger](https://www.jaegertracing.io)
formatted traces.

It supports the Jaeger Collector and Agent protocols:
- gRPC
- Thrift HTTP
- Thrift TChannel
- Thrift Compact
- Thrift Binary

By default, the Jaeger receiver will not serve any protocol. A protocol must be named
for the jaeger receiver to start.  The following demonstrates how to start the Jaeger
receiver with only gRPC enabled on the default port.
```yaml
receivers:
  jaeger:
    protocols:
      grpc:
```

It is possible to configure the protocols on different ports, refer to
[config.yaml](./testdata/config.yaml) for detailed config
examples. The full list of settings exposed for this receiver are
documented [here](./config.go).

## Communicating over TLS
The Jaeger receiver supports communication using Transport Layer Security (TLS), but
only using the gRPC protocol. It can be configured by specifying a
`tls_credentials` object in the gRPC receiver configuration.
```yaml
receivers:
  jaeger:
    protocols:
      grpc:
        tls_credentials:
          key_file: /key.pem # path to private key
          cert_file: /cert.pem # path to certificate
        endpoint: "localhost:9876"
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

Note: gRPC must be enabled for this to work as Jaeger serves its remote
sampling strategies over gRPC.
