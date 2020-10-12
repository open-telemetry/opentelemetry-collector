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
 - `match_type`: strict|regexp|expr
 - `metric_names`: (only for a `match_type` of 'strict' or 'regexp') list of strings or re2 regex patterns
 - `expressions`: (only for a `match_type` of 'expr') list of expr expressions (see "Using an 'expr' match_type" below)

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

Refer to the config files in [testdata](./testdata) for detailed
examples on using the processor.

### Using an 'expr' match_type

In addition to matching metric names with the 'strict' or 'regexp' match types, the filter processor
supports matching entire `Metric`s using the [expr](https://github.com/antonmedv/expr) expression engine.

The expr filter evaluates the supplied boolean expressions _per datapoint_ on a metric, and returns a result
for the entire metric. If any datapoint evaluates to true then the entire metric evaluates to true, otherwise
false.

Made available to the expression environment are the following:

* `MetricName`
    a variable containing the current Metric's name
* `Label(name)`
    a function that takes a label name string as an argument and returns a string: the value of a label with that
    name if one exists, or ""
* `HasLabel(name)`
    a function that takes a label name string as an argument and returns a boolean: true if the datapoint has a label
    with that name, false otherwise

Example:

```yaml
processors:
  filter/1:
    metrics:
      exclude:
        match_type: expr
        expressions:
        - MetricName == "my.metric" && Label("my_label") == "abc123"
```

The above config will filter out any Metric that both has the name "my.metric" and has at least one datapoint
with a label of `my_label="abc123"`.
