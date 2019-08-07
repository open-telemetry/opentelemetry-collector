**Note** This documentation is still in progress. For any questions, please
reach out in the [OpenTelemetry Gitter](https://gitter.im/open-telemetry/opentelemetry-service)
or refer to the [issues page](https://github.com/open-telemetry/opentelemetry-service/issues).

# Receivers
A receiver is how data gets into OpenTelemetry Service. Generally, a receiver
accepts data in a specified format and can support traces and/or metrics. The
format of the traces and metrics supported are receiver specific.

Supported receivers (sorted alphabetically):
- [Jaeger Receiver](#jaeger)
- [OpenCensus Receiver](#opencensus)
- [Prometheus Receiver](#prometheus)
- [VM Metrics Receiver](#vmmetrics)
- [Zipkin Receiver](#zipkin)

## Configuring Receiver(s)
Receivers are configured via YAML under the top-level `receivers` tag. There
must be at least one enabled receiver for this configuration to be considered
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
    endpoint: 127.0.0.1:9211
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

## <a name="opencensus"></a>OpenCensus Receiver
**Traces and metrics are supported.**

This receiver receives trace and metrics from [OpenCensus](https://opencensus.io/)
instrumented applications. It translates them into the internal format sent to
processors and exporters in the pipeline.

To get started, all that is required to enable the OpenCensus receiver is to
include it in the receiver definitions. This will enable the default values as
specified [here](https://github.com/open-telemetry/opentelemetry-service/blob/master/receiver/opencensusreceiver/factory.go).
The following is an example:
```yaml
receivers:
  opencensus:
```

The full list of settings exposed for this receiver are documented [here](https://github.com/open-telemetry/opentelemetry-service/blob/master/receiver/opencensusreceiver/config.go)
with detailed sample configurations [here](https://github.com/open-telemetry/opentelemetry-service/blob/master/receiver/opencensusreceiver/testdata/config.yaml).

### Writing with HTTP/JSON 
// TODO(ccaraman) The cors setting wasn't ported accidentally. In a follow up
pr, add the functionality, tests and update this section of documentation.

The OpenCensus receiver for the agent can receive trace export calls via
HTTP/JSON in addition to gRPC. The HTTP/JSON address is the same as gRPC as the
protocol is recognized and processed accordingly.

To write traces with HTTP/JSON, `POST` to `[address]/v1/trace`. The JSON message
format parallels the gRPC protobuf format, see this
[OpenApi spec for it](https://github.com/census-instrumentation/opencensus-proto/blob/master/gen-openapi/opencensus/proto/agent/trace/v1/trace_service.swagger.json).

The HTTP/JSON endpoint can also optionally 
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

## <a name="jaeger"></a>Jaeger Receiver
**Only traces are supported.**
// TODO(ccaraman) Update Jaeger receiver documentation.

This receiver receives spans from Jaeger collector HTTP and Thrift uploads and translates them into the internal span types that are then sent to the collector/exporters.
Only traces are supported. This receiver does not support metrics.

Its address can be configured in the YAML configuration file under section "receivers", subsection "jaeger" and fields "collector_http_port", "collector_thrift_port".

For example:

```yaml
receivers:
  jaeger:
    collector_thrift_port: 14267
    collector_http_port: 14268
```

## <a name="prometheus"></a>Prometheus Receiver
**Only metrics are supported.**

This receiver is a drop-in replacement for getting Prometheus to scrape your services. Just like you would write in a
YAML configuration file before starting Prometheus, such as with:
```shell
prometheus --config.file=prom.yaml
```

you can copy and paste that same configuration under section
```yaml
receivers:
  prometheus:
    config:
```

such as:
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

## <a name="vmmetrics"></a>VM Metrics Receiver
**Only metrics are supported.**

<Add more information - I'm lonely.>

## <a name="zipkin"></a>Zipkin Receiver
**Only traces are supported.**

This receiver receives spans from Zipkin (V1 and V2) HTTP uploads and translates them into the internal span types that are then sent to the collector/exporters.

Its address can be configured in the YAML configuration file under section "receivers", subsection "zipkin" and field "address".  The syntax of the field "address" is `[address|host]:<port-number>`.

For example:

```yaml
receivers:
  zipkin:
    address: "127.0.0.1:9411"
```

## Common Configuration Errors
<Fill this in as we go with common gotchas experienced by users. These should eventually be made apart of the validation test suite.>
