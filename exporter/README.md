# Exporters

Below is the list of exporters directly supported by the OpenTelemetry Collector.

* [Jaeger](#jaeger)
* [Logging](#logging)
* [OpenCensus](#opencensus)
* [Prometheus](#prometheus)
* [Zipkin](#zipkin)

The [contributors repository](https://github.com/open-telemetry/opentelemetry-service-contrib)
 has more exporters that can be added to custom builds of the service.

## Proxy Support

Beyond standard YAML configuration as outlined in the sections that follow,
exporters that leverage the net/http package (all do today) also respect the
following proxy environment variables:

* HTTP_PROXY
* HTTPS_PROXY
* NO_PROXY

If set at Collector start time then exporters, regardless of protocol,
will or will not proxy traffic as defined by these environment variables.

## Data Ownership

When multiple exporters are configured to send the same data (e.g. by configuring multiple
exporters for the same pipeline) the exporters will have a shared access to the data.
Exporters get access to this shared data when `ConsumeTraceData`/`ConsumeMetricsData`
function is called. Exporters MUST NOT modify the `TraceData`/`MetricsData` argument of
these functions. If the exporter needs to modify the data while performing the exporting
the exporter can clone the data and perform the modification on the clone or use a
copy-on-write approach for individual sub-parts of `TraceData`/`MetricsData` argument.
Any approach that does not mutate the original `TraceData`/`MetricsData` argument 
(including referenced data, such as `Node`, `Resource`, `Spans`, etc) is allowed.

## <a name="jaeger"></a>Jaeger

Exports trace data to [Jaeger](https://www.jaegertracing.io/) collectors
accepting one of the following protocols:

* [gRPC](#jaeger_grpc)

### <a name="jaeger-configuration"></a>Configuration

Each different supported protocol has its own configuration settings.

#### <a name="jaeger_grpc"></a>gRPC

* `endpoint:` target to which the exporter is going to send Jaeger trace data,
using the gRPC protocol. The valid syntax is described at
https://github.com/grpc/grpc/blob/master/doc/naming.md

* `secure`: whether to enable client transport security for the exporter's gRPC
connection. See [grpc.WithInsecure()](https://godoc.org/google.golang.org/grpc#WithInsecure).
Optional.

* `server_name_override`: If set to a non empty string, it will override the virtual host name 
of authority (e.g. :authority header field) in requests (typically used for testing).

* `cert_pem_file`: certificate file for TLS credentials of gRPC client. Should
only be used if `secure` is set to true. Optional.

* `keepalive`: keepalive parameters for client gRPC. See
[grpc.WithKeepaliveParams()](https://godoc.org/google.golang.org/grpc#WithKeepaliveParams).
Optional.


Example:

```yaml
exporters:
  jaeger_grpc:
    endpoint: jaeger-all-in-one:14250
    cert_pem_file: /my-cert.pem
    server_name_override: opentelemetry.io
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

* `num_workers`: number of workers that send the gRPC requests. Optional.

* `secure`: whether to enable client transport security for the exporter's gRPC
connection. See [grpc.WithInsecure()](https://godoc.org/google.golang.org/grpc#WithInsecure).
Optional.

* `cert_pem_file`: certificate file for TLS credentials of gRPC client. Should
only be used if `secure` is set to true. Optional.

* `reconnection_delay`: time period between each reconnection performed by the
exporter. Optional.

* `keepalive`: keepalive parameters for client gRPC. See
[grpc.WithKeepaliveParams()](https://godoc.org/google.golang.org/grpc#WithKeepaliveParams).
Optional.

Example:

```yaml
exporters:
  opencensus:
    endpoint: localhost:14250
    reconnection_delay: 60s
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
