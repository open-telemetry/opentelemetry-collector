# General Information

An exporter is how data gets sent to different systems/back-ends. Generally, an
exporter translates the internal format into another defined format.

Supported trace exporters (sorted alphabetically):

- [Jaeger](#jaeger)
- [OpenCensus](#opencensus)
- [Zipkin](#zipkin)

Supported metric exporters (sorted alphabetically):

- [OpenCensus](#opencensus)
- [Prometheus](#prometheus)

Supported local exporters (sorted alphabetically):

- [File](#file)
- [Logging](#logging)

The [contributors repository](https://github.com/open-telemetry/opentelemetry-collector-contrib)
 has more exporters that can be added to custom builds of the Collector.

## Proxy Support

Beyond standard YAML configuration as outlined in the sections that follow,
exporters that leverage the net/http package (all do today) also respect the
following proxy environment variables:

- HTTP_PROXY
- HTTPS_PROXY
- NO_PROXY

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

# <a name="trace-exporters"></a>Trace Exporters

## <a name="jaeger"></a>Jaeger Exporter
Exports trace data to [Jaeger](https://www.jaegertracing.io/) collectors
accepting one of the following protocols:

- [gRPC](#jaeger_grpc)
- [Thrift HTTP](#jaeger_thirft_http)

Each different supported protocol has its own configuration settings.

### <a name="jaeger_grpc"></a>gRPC

The following settings are required:

- `endpoint` (no default): target to which the exporter is going to send Jaeger trace data,
using the gRPC protocol. The valid syntax is described at
https://github.com/grpc/grpc/blob/master/doc/naming.md

The following settings can be optionally configured:

- `cert_pem_file`: certificate file for TLS credentials of gRPC client. Should
only be used if `secure` is set to true.
- `keepalive`: keepalive parameters for client gRPC. See
[grpc.WithKeepaliveParams()](https://godoc.org/google.golang.org/grpc#WithKeepaliveParams).
- `secure`: whether to enable client transport security for the exporter's gRPC
connection. See [grpc.WithInsecure()](https://godoc.org/google.golang.org/grpc#WithInsecure).
- `server_name_override`: If set to a non empty string, it will override the virtual host name 
of authority (e.g. :authority header field) in requests (typically used for testing).

Example:

```yaml
exporters:
  jaeger_grpc:
    endpoint: jaeger-all-in-one:14250
    cert_pem_file: /my-cert.pem
    server_name_override: opentelemetry.io
```

The full list of settings exposed for this exporter are documented [here](jaeger/jaegergrpcexporter/config.go)
with detailed sample configurations [here](jaeger/jaegergrpcexporter/testdata/config.yaml).

### <a name="jaeger_thrift_http"></a>Thrift HTTP

The following settings are required:

- `url` (no default): target to which the exporter is going to send Jaeger trace data,
using the Thrift HTTP protocol.

The following settings can be optionally configured:

- `timeout` (default = 5s): the maximum time to wait for a HTTP request to complete
- `headers` (no default): headers to be added to the HTTP request

Example:

```yaml
exporters:
  jaeger:
    url: "http://some.other.location/api/traces"
    timeout: 2s
    headers:
      added-entry: "added value"
      dot.test: test
```

The full list of settings exposed for this exporter are documented [here](jaeger/jaegerthrifthttpexporter/config.go)
with detailed sample configurations [here](jaeger/jaegerthrifthttpexporter/testdata/config.yaml).

## <a name="opencensus-traces"></a>OpenCensus Exporter
Exports traces and/or metrics to another Collector via gRPC using OpenCensus format.

The following settings are required:

- `endpoint`: target to which the exporter is going to send traces or metrics,
using the gRPC protocol. The valid syntax is described at
https://github.com/grpc/grpc/blob/master/doc/naming.md.

The following settings can be optionally configured:

- `cert_pem_file`: certificate file for TLS credentials of gRPC client. Should
only be used if `secure` is set to true.
- `compression`: compression key for supported compression types within
collector. Currently the only supported mode is `gzip`.
- `headers`: the headers associated with gRPC requests.
- `keepalive`: keepalive parameters for client gRPC. See
[grpc.WithKeepaliveParams()](https://godoc.org/google.golang.org/grpc#WithKeepaliveParams).
- `num_workers` (default = 2): number of workers that send the gRPC requests. Optional.
- `reconnection_delay`: time period between each reconnection performed by the
exporter.
- `secure`: whether to enable client transport security for the exporter's gRPC
connection. See [grpc.WithInsecure()](https://godoc.org/google.golang.org/grpc#WithInsecure).

Example:

```yaml
exporters:
  opencensus:
    endpoint: localhost:14250
    reconnection_delay: 60s
    secure: false
```

The full list of settings exposed for this exporter are documented [here](opencensusexporter/config.go)
with detailed sample configurations [here](opencensusexporter/testdata/config.yaml).

## <a name="zipkin"></a>Zipkin Exporter
Exports trace data to a [Zipkin](https://zipkin.io/) back-end.

The following settings are required:

- `format` (default = JSON): The format to sent events in. Can be set to JSON or proto.
- `url` (no default): URL to which the exporter is going to send Zipkin trace data.

The following settings can be optionally configured:

- (temporary flag) `export_resource_labels` (default = true): Whether Resource labels are going to be merged with span attributes
Note: this flag was added to aid the migration to new (fixed and symmetric) behavior and is going to be 
removed soon. See https://github.com/open-telemetry/opentelemetry-collector/issues/595 for more details
- `defaultservicename` (no default): What to name services missing this information

Example:

```yaml
exporters:
  zipkin:
    url: "http://some.url:9411/api/v2/spans"
```

The full list of settings exposed for this exporter are documented [here](zipkinexporter/config.go)
with detailed sample configurations [here](zipkinexporter/testdata/config.yaml).

## <a name="opencensus-metrics"></a>OpenCensus Exporter
The OpenCensus exporter supports both traces and metrics. Configuration
information can be found under the trace section [here](#opencensus-traces).

## <a name="prometheus"></a>Prometheus Exporter
Exports metric data to a [Prometheus](https://prometheus.io/) back-end.

The following settings are required:

- `endpoint` (no default): Where to send metric data

The following settings can be optionally configured:

- `constlabels` (no default): key/values that are applied for every exported metric.
- `namespace` (no default): if set, exports metrics under the provided value.

Example:

```yaml
exporters:
  prometheus:
    endpoint: "1.2.3.4:1234"
    namespace: test-space
    const_labels:
      label1: value1
      "another label": spaced value
```

The full list of settings exposed for this exporter are documented [here](prometheusexporter/config.go)
with detailed sample configurations [here](prometheusexporter/testdata/config.yaml).

# <a name="local-exporters"></a>Local Exporters

Local exporters send data to a local endpoint such as the console or a log file.

## <a name="file"></a>File Exporter
This exporter will write the pipeline data to a JSON file.
The data is written in Protobuf JSON encoding
(https://developers.google.com/protocol-buffers/docs/proto3#json).
Note that there are no compatibility guarantees for this format, since it
just a dump of internal structures which can be changed over time.
This intended for primarily for debugging Collector without setting up backends.

The following settings are required:

- `path` (no default): where to write information.

Example:

```yaml
exporters:
  file:
    path: ./filename.json
```

The full list of settings exposed for this exporter are documented [here](fileexporter/config.go)
with detailed sample configurations [here](fileexporter/testdata/config.yaml).

## <a name="logging"></a>Logging Exporter
Exports traces and/or metrics to the console via zap.Logger.

The following settings can be configured:

- `loglevel`: the log level of the logging export (debug|info|warn|error). Default is `info`.

Example:

```yaml
exporters:
  logging:
```

The full list of settings exposed for this exporter are documented [here](loggingexporter/config.go)
with detailed sample configurations [here](loggingexporter/testdata/config.yaml).
