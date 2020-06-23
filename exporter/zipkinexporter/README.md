# Zipkin Exporter
 
Exports trace data to a [Zipkin](https://zipkin.io/) back-end.

The following settings are required:

- `format` (default = JSON): The format to sent events in. Can be set to JSON or proto.
- `url` (no default): URL to which the exporter is going to send Zipkin trace data.

The following settings can be optionally configured:

- `defaultservicename` (no default): What to name services missing this information

Example:

```yaml
exporters:
zipkin:
 url: "http://some.url:9411/api/v2/spans"
```

The full list of settings exposed for this exporter are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).
