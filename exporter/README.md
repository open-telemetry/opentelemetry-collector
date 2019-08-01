# Exporters

Below is the list of exporters directly supported by the OpenTelemetry Service.

* [Jaeger](#jaeger)
* [Logging](#logging)
* [OpenCensus](#opencensus)
* [Prometheus](#prometheus)
* [Zipkin](#zipkin)

The [contributors repository](https://github.com/open-telemetry/opentelemetry-service-contrib)
 has more exporters that can be added to custom builds of the service.

## <a name="jaeger"></a>Jaeger 

Exports trace data to [Jaeger](https://www.jaegertracing.io/) collectors 
accepting one of the following protocols:

* [gRPC](#jaeger-grpc)

### Configuration

Each different supported protocol has its own configuration settings.

#### <a name="jaeger-grpc"></a>gRPC

* `endpoint:` target to which the exporter is going to send Jaeger trace data, 
using the gRPC protocol. The valid syntax is described at 
https://github.com/grpc/grpc/blob/master/doc/naming.md

Example:

```yaml
exporters:
  jaeger-grpc:
    endpoint: jaeger-all-in-one:14250
```

## <a name="logging"></a>Logging
TODO: document settings 

## <a name="opencensus"></a>OpenCensus
TODO: document settings 

## <a name="prometheus"></a>Prometheus
TODO: document settings 

## <a name="zipkin"></a>Zipkin
Exports trace data to a [Zipkin](https://zipkin.io/) back-end.

### Configuration

The following settings can be configured:

* `url:` URL to which the exporter is going to send Zipkin trace data. This
setting doesn't have a default value and must be specified in the configuration.

Example:

```yaml
exporters:
  zipkin:
    url: "http://some.url:9411/api/v2/spans"
```
