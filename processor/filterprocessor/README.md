# Filter Processor

Supported pipeline types: metrics

The filter processor can be configured to include or exclude metrics based on
metric name. Please refer to [config.go](./config.go) for the
config spec.

It takes a pipeline type, of which only `metrics` is supported, followed by an
action:
- `include`: Any names NOT matching filters are excluded from remainder of pipeline
- `exclude`: Any names matching filters are excluded from remainder of pipeline

For the actions the following parameters are required:
 - `match_type`: strict|regexp
 - `metric_names`: list of strings or re2 regex patterns

More details can found at [include/exclude metrics](../README.md#includeexclude-metrics).

Examples:

```yaml
processors:
  filter/1:
    metrics:
      include:
        match_type: regexp
        metric_names:
        - prefix/.*
        - prefix_.*
      exclude:
        match_type: strict
        metric_names:
        - hello_world
        - hello/world
```

Refer to [config.yaml](./testdata/config.yaml) for detailed
examples on using the processor.
