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

### <a name="jaeger-configuration"></a>Configuration

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
Exports traces and/or metrics to the console via zap.Logger

### <a name="logging-configuration"></a>Configuration

* `loglevel`: the log level of the logging export (debug|info|warn|error). Default is `info`.

## <a name="opencensus"></a>OpenCensus
Exports traces and/or metrics to another OTel-Svc endpoint via gRPC.

### <a name="opencensus-configuration"></a>Configuration

* `endpoint`: target to which the exporter is going to send traces or metrics,
using the gRPC protocol. The valid syntax is described at
https://github.com/grpc/grpc/blob/master/doc/naming.md. Required.

* `compression`: compression key for supported compression types within
collector. Currently the only supported mode is `gzip`. Optional.

* `headers`: the headers associated with gRPC requests. Optional.

* `num-workers`: number of workers that send the gRPC requests. Optional.

* `secure`: whether to enable client transport security for the exporter's gRPC
connection. See [grpc.WithInsecure()](https://godoc.org/google.golang.org/grpc#WithInsecure).
Optional.

* `cert-pem-file`: certificate file for TLS credentials of gRPC client. Should
only be used if `secure` is set to true. Optional.

* `reconnection-delay`: time period between each reconnection performed by the
exporter. Optional.

* `keepalive`: keepalive parameters for client gRPC. See
[grpc.WithKeepaliveParams()](https://godoc.org/google.golang.org/grpc#WithKeepaliveParams).
Optional.

Example:

```yaml
exporters:
  opencensus:
    endpoint: 127.0.0.1:14250
    reconnection-delay: 60s
    secure: false
```

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
