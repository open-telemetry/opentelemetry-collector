# General Information

An exporter is how data gets sent to different systems/back-ends. Generally, an
exporter translates the internal format into another defined format.

Supported trace exporters (sorted alphabetically):

- [Jaeger](jaegerexporter/README.md)
- [Kafka](kafkaexporter/README.md)
- [OpenCensus](opencensusexporter/README.md)
- [OTLP](otlpexporter/README.md)
- [Zipkin](zipkinexporter/README.md)

Supported metric exporters (sorted alphabetically):

- [OpenCensus](opencensusexporter/README.md)
- [Prometheus](prometheusexporter/README.md)

Supported local exporters (sorted alphabetically):

- [File](fileexporter/README.md)
- [Logging](loggingexporter/README.md)

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
