A variety of receivers are available to the OpenCensus Service (both Agent and Collector)

## OpenCensus

This receiver receives spans from OpenCensus instrumented applications and translates them into the internal span types that are then sent to the collector/exporters.

Its address can be configured in the YAML configuration file under section "receivers", subsection "opencensus" and field "address".

For example:

```yaml
receivers:
  opencensus:
    address: "localhost:55678"
``````

By default this receiver is ALWAYS started on the OpenCensus Agent

## Jaeger

This receiver receives spans from Jaeger collector HTTP and Thrift uploads and translates them into the internal span types that are then sent to the collector/exporters.

Its address can be configured in the YAML configuration file under section "receivers", subsection "jaeger" and fields "collector_http_port", "collector_thrift_port".

For example:

```yaml
receivers:
  jaeger
    collector_thrift_port: 14267
    collector_http_port: 14268
``````

## Zipkin

This receiver receives spans from Zipkin "/v2" API HTTP uploads and translates them into the internal span types that are then sent to the collector/exporters.

Its address can be configured in the YAML configuration file under section "receivers", subsection "zipkin" and field "address".

For example:

```yaml
receivers:
  zipkin:
    address: "localhost:9411"
``````
