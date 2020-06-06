# Zipkin Exporter
 
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

The full list of settings exposed for this exporter are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).
