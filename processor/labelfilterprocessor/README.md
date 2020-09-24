### Background

Today, the Collector has a [filterprocessor](https://github.com/open-telemetry/opentelemetry-collector/tree/master/processor/filterprocessor)
which filters metrics based on whether metric names match a given list of strings. Matches can be `regexp` or
`strict` and can be in an `exclude` or `include` block.

Otel Collector users have expressed an interest in also filtering metrics by metrics+label_keys, and
metrics+label_keys+label_values.

### Proposal

Create a new filter processor whose configuration could indicate one or more
`metric name` -> `label key` -> `label value`. If there is a match along a supplied path of
`metric name` -> `label key` -> `label value` to a metric or any of the time series in a Metric, the entire
Metric is considered a match and is either included or excluded, depending on the configuration.

In keeping with `filterprocessor`'s convention, matches against any of the search strings can be `regexp` or
`strict`. Furthermore, the following partial paths may be supplied:

* `metric name` only
* `metric name` -> `label key` only
* `metric name` -> `label key` -> `label value`

On the other hand, to match against just a label value, a `.*` regex could be supplied to `metric name` and
`label key`.

Multiple values may be supplied, forming a tree:

```
my.metric.name
  key1
    val1
    val2
  key2
```

Borrowing from filterprocessor, each node in the tree may specify a strict or regex match.

Example config:

```
processors:
  labelfilter:
    - exclude:
        - metric_names:
            - "my.metric.1":
                match_type: strict
                label_keys:
                  - "my-label-key1":
                      match_type: strict
                      label_vals:
                        - "my-label-value1":
                            match_type: strict
                        - "my-label-value2":
                            match_type: strict
                  - "my-.*-key2":
                      match_type: regex
```

As with the filterprocessor, if both include and exclude are specified, the includes are checked before the excludes.

#### Questions

* Assuming we'll want to support regex caching like filtermetrics does, should it be a top-level config, with
the option to override at lower levels?
* Do we want to support what's been proposed in #1081?
