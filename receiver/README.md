# General Information
A receiver is how data gets into the OpenTelemetry Collector. Generally, a receiver
accepts data in a specified format, translates it into the internal format and
passes it to [processors](../processor/README.md)
and [exporters](../exporter/README.md)
defined in the applicable [pipelines](../docs/pipelines.md).
The format of the traces and metrics supported are receiver specific.

Supported trace receivers (sorted alphabetically):
- [Jaeger Receiver](#jaeger)
- [OpenCensus Receiver](#opencensus)
- [Zipkin Receiver](#zipkin)

Supported metric receivers (sorted alphabetically):
- [Host Metrics Receiver](#hostmetrics)
- [OpenCensus Receiver](#opencensus)
- [Prometheus Receiver](#prometheus)
- [VM Metrics Receiver](#vmmetrics)

The [contributors repository](https://github.com/open-telemetry/opentelemetry-collector-contrib)
 has more receivers that can be added to custom builds of the collector.

## Configuring Receiver(s)
Receivers are configured via YAML under the top-level `receivers` tag. There
must be at least one enabled receiver for a configuration to be considered
valid.

The following is a sample configuration for the `examplereceiver`.
```yaml
receivers:
  # Receiver 1.
  # <receiver type>:
  examplereceiver:
    # <setting one>: <value one>
    endpoint: 1.2.3.4:8080
    # ...
  # Receiver 2.
  # <receiver type>/<name>:
  examplereceiver/settings:
    # <setting one>: <value one>
    disabled: false
    # <setting two>: <value two>
    endpoint: localhost:9211
```

A receiver instance is referenced by its full name in other parts of the config,
such as in pipelines. A full name consists of the receiver type, '/' and the
name appended to the receiver type in the configuration. All receiver full names
must be unique. For the example above:
- Receiver 1 has full name `examplereceiver`.
- Receiver 2 has full name `examplereceiver/settings`.

All receivers expose a setting to disable it, by default receivers are enabled.
At least one receiver must be enabled per [pipeline](docs/pipelines.md) to be a
valid configuration.

# <a name="trace-receivers"></a>Trace Receivers

## <a name="jaeger"></a>Jaeger Receiver

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
[config.yaml](jaegerreceiver/testdata/config.yaml) for detailed config
examples. The full list of settings exposed for this receiver are
documented [here](jaegerreceiver/config.go).

### Communicating over TLS
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

### Remote Sampling
The Jaeger receiver also supports fetching sampling configuration from a remote collector.
It works by proxying client requests for remote sampling configuration to the configured collector.

+---------------+                   +--------------+              +-----------------+
|               |       get         |              |    proxy     |                 |
|    client     +---  sampling ---->+    agent     +------------->+    collector    |
|               |     strategy      |              |              |                 |
+---------------+                   +--------------+              +-----------------+

Remote sample proxying can be enabled by specifying the following lines in the jaeger receiver config:

```yaml
receivers:
  jaeger:
    protocols:
      grpc:
      .
      .
    remote_sampling:
      fetch_endpoint: "jaeger-collector:1234"
``` 

Remote sampling can also be directly served by the collector by providing a sampling json file:

```yaml
receivers:
  jaeger:
    protocols:
      grpc:
    remote_sampling:
      strategy_file: "/etc/strategy.json"
``` 

Note: gRPC must be enabled for this to work as Jaeger serves its remote sampling strategies over gRPC.

## <a name="opencensus-traces"></a>OpenCensus Receiver

This receiver receives trace and metrics from [OpenCensus](https://opencensus.io/)
instrumented applications.

To get started, all that is required to enable the OpenCensus receiver is to
include it in the receiver definitions. This will enable the default values as
specified [here](opencensusreceiver/factory.go).
The following is an example:
```yaml
receivers:
  opencensus:
```

The full list of settings exposed for this receiver are documented [here](opencensusreceiver/config.go)
with detailed sample configurations [here](opencensusreceiver/testdata/config.yaml).

### Communicating over TLS
This receiver supports communication using Transport Layer Security (TLS). TLS
can be configured by specifying a `tls_credentials` object in the receiver
configuration for receivers that support it.
```yaml
receivers:
  opencensus:
    tls_credentials:
      key_file: /key.pem # path to private key
      cert_file: /cert.pem # path to certificate
```

### Writing with HTTP/JSON
The OpenCensus receiver can receive trace export calls via HTTP/JSON in
addition to gRPC. The HTTP/JSON address is the same as gRPC as the protocol is
recognized and processed accordingly.

To write traces with HTTP/JSON, `POST` to `[address]/v1/trace`. The JSON message
format parallels the gRPC protobuf format, see this
[OpenApi spec for it](https://github.com/census-instrumentation/opencensus-proto/blob/master/gen-openapi/opencensus/proto/agent/trace/v1/trace_service.swagger.json).

The HTTP/JSON endpoint can also optionally configure
[CORS](https://fetch.spec.whatwg.org/#cors-protocol), which is enabled by
specifying a list of allowed CORS origins in the `cors_allowed_origins` field:

```yaml
receivers:
  opencensus:
    endpoint: "localhost:55678"
    cors_allowed_origins:
    - http://test.com
    # Origins can have wildcards with *, use * by itself to match any origin.
    - https://*.example.com
```

## <a name="zipkin"></a>Zipkin Receiver

This receiver receives spans from [Zipkin](https://zipkin.io/) (V1 and V2).

To get started, all that is required to enable the Zipkin receiver is to
include it in the receiver definitions. This will enable the default values as
specified [here](zipkinreceiver/factory.go).
The following is an example:
```yaml
receivers:
  zipkin:
```

The full list of settings exposed for this receiver are documented [here](zipkinreceiver/config.go)
with detailed sample configurations [here](zipkinreceiver/testdata/config.yaml).

# <a name="metric-receivers"></a>Metric Receivers

## <a name="opencensus-metrics"></a>OpenCensus Receiver

The OpenCensus receiver supports both traces and metrics. Configuration
information can be found under the trace section [here](#opencensus-traces).

## <a name="prometheus"></a>Prometheus Receiver

This receiver is a drop-in replacement for getting Prometheus to scrape your
services. Just like you would write in a YAML configuration file before
starting Prometheus, such as with:
```shell
prometheus --config.file=prom.yaml
```

You can copy and paste that same configuration under section
```yaml
receivers:
  prometheus:
    config:
```

For example:
```yaml
receivers:
    prometheus:
      config:
        scrape_configs:
          - job_name: 'opencensus_service'
            scrape_interval: 5s
            static_configs:
              - targets: ['localhost:8889']

          - job_name: 'jdbc_apps'
            scrape_interval: 3s
            static_configs:
              - targets: ['localhost:9777']
```

### Include Filter
Include Filter provides ability to filter scraping metrics per target. If a
filter is specified for a target then only those metrics which exactly matches
one of the metrics specified in the `Include Filter` list will be scraped. Rest
of the metrics from the targets will be dropped.

#### Syntax
* Endpoint should be double quoted.
* Metrics should be specified in form of a list.

#### Example
```yaml
receivers:
    prometheus:
      include_filter: {
        "localhost:9777" : [http/server/server_latency, custom_metric1],
        "localhost:9778" : [http/client/roundtrip_latency],
      }
      config:
        scrape_configs:
          ...
```

The full list of settings exposed for this receiver are documented [here](prometheusreceiver/config.go)
with detailed sample configurations [here](prometheusreceiver/testdata/config.yaml).

## <a name="vmmetrics"></a>VM Metrics Receiver

Collects metrics from the host operating system. This is applicable when the
OpenTelemetry Collector is running as an agent.
```yaml
receivers:
  vmmetrics:
    scrape_interval: 10s
    metric_prefix: "testmetric"
    mount_point: "/proc"
    #process_mount_point: "/data/proc" # Only using when running as an agent / daemonset
```

The full list of settings exposed for this receiver are documented [here](vmmetricsreceiver/config.go)
with detailed sample configurations [here](vmmetricsreceiver/testdata/config.yaml).
