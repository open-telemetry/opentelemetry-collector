# Exporters

A variety of exporters are available to the OpenTelemetry Service:

* [Jaeger](#jaeger)
* [Logging](#logging)
* [OpenCensus](#opencensus)
* [Prometheus](#prometheus)
* [Zipkin](#zipkin)

## <a name="jaeger"></a>Jaeger 
TODO: document settings 

## <a name="logging"></a>Logging
TODO: document settings 

## <a name="opencensus"></a>OpenCensus
TODO: document settings 

## <a name="prometheus"></a>Prometheus
TODO: document settings 

## <a name="zipkin"></a>Zipkin
Exports trace data to a [Zipkin](https://zipkin.io/) endpoint.

### Configuration

The following settings can be configured:

* `endpoint:` URL to which the exporter is going to send Zipkin trace data. This
setting doesn't have a default value and must be specified in the configuration.

Example:

```yaml
exporters:
  zipkin:
    endpoint: "http://some.url:9411/api/v2/spans"
```
