# General Information

An exporter is how data gets sent to different systems/back-ends. Generally, an
exporter translates the internal format into another defined format.

Available trace exporters (sorted alphabetically):

- [OTLP gRPC](otlpexporter/README.md)
- [OTLP HTTP](otlphttpexporter/README.md)

Available metric exporters (sorted alphabetically):

- [OTLP gRPC](otlpexporter/README.md)
- [OTLP HTTP](otlphttpexporter/README.md)

Available log exporters (sorted alphabetically):

- [OTLP gRPC](otlpexporter/README.md)
- [OTLP HTTP](otlphttpexporter/README.md)

Available local exporters (sorted alphabetically):

- [File](fileexporter/README.md)
- [Logging](loggingexporter/README.md)

The [contrib
repository](https://github.com/open-telemetry/opentelemetry-collector-contrib)
has more exporters available in its builds.

## Configuring Exporters

Exporters are configured via YAML under the top-level `exporters` tag.

The following is a sample configuration for the `exampleexporter`.

```yaml
exporters:
  # Exporter 1.
  # <exporter type>:
  exampleexporter:
    # <setting one>: <value one>
    endpoint: 1.2.3.4:8080
    # ...
  # Exporter 2.
  # <exporter type>/<name>:
  exampleexporter/settings:
    # <setting two>: <value two>
    endpoint: 0.0.0.0:9211
```

An exporter instance is referenced by its full name in other parts of the config,
such as in pipelines. A full name consists of the exporter type, '/' and the
name appended to the exporter type in the configuration. All exporter full names
must be unique.

For the example above:

- Exporter 1 has full name `exampleexporter`.
- Exporter 2 has full name `exampleexporter/settings`.

Exporters are enabled upon being added to a pipeline. For example:

```yaml
service:
  pipelines:
    # Valid pipelines are: traces, metrics or logs
    # Trace pipeline 1.
    traces:
      receivers: [examplereceiver]
      processors: []
      exporters: [exampleexporter, exampleexporter/settings]
    # Trace pipeline 2.
    traces/another:
      receivers: [examplereceiver]
      processors: []
      exporters: [exampleexporter, exampleexporter/settings]
```

## Data Ownership

When multiple exporters are configured to send the same data (e.g. by configuring multiple
exporters for the same pipeline) the exporters will have a shared access to the data.
Exporters get access to this shared data when `ConsumeTraces`/`ConsumeMetrics`/`ConsumeLogs`
function is called. Exporters MUST NOT modify the `pdata.Traces`/`pdata.Metrics`/`pdata.Logs` argument of
these functions. If the exporter needs to modify the data while performing the exporting
the exporter can clone the data and perform the modification on the clone or use a
copy-on-write approach for individual sub-parts of `pdata.Traces`/`pdata.Metrics`/`pdata.Logs`.
Any approach that does not mutate the original `pdata.Traces`/`pdata.Metrics`/`pdata.Logs` is allowed.

## Proxy Support

Beyond standard YAML configuration as outlined in the individual READMEs above,
exporters that leverage the net/http package (all do today) also respect the
following proxy environment variables:

- HTTP_PROXY
- HTTPS_PROXY
- NO_PROXY

If set at Collector start time then exporters, regardless of protocol,
will or will not proxy traffic as defined by these environment variables.
