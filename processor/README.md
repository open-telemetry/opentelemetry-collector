# General Information
Processors are used at various stages of a [pipeline](../docs/pipelines.md).
Generally, a processor pre-processes data before it is exported (e.g.
modify attributes or sample) or helps ensure that data makes it through a
pipeline successfully (e.g. batch/retry).

Supported processors (sorted alphabetically):
- [Attributes Processor](#attributes)
- [Batch Processor](#batch)
- [Queued Retry Processor](#queued_retry)
- [Sampling Processor](#sampling)
- [Span Processor](#span)

The [contributors repository](https://github.com/open-telemetry/opentelemetry-collector-contrib)
 has more processors that can be added to custom builds of the Collector.

## Data Ownership

The ownership of the `TraceData` and `MetricsData` in a pipeline is passed as the data travels
through the pipeline. The data is created by the receiver and then the ownership is passed
to the first processor when `ConsumeTraceData`/`ConsumeMetricsData` function is called.

Note: the receiver may be attached to multiple pipelines, in which case the same data
will be passed to all attached pipelines via a data fan-out connector.

From data ownership perspective pipelines can work in 2 modes:
* Exclusive data ownership
* Shared data ownership

The mode is defined during startup based on data modification intent reported by the
processors. The intent is reported by each processor via `MutatesConsumedData` field of
the struct returned by `GetCapabilities` function. If any processor in the pipeline
declares an intent to modify the data then that pipeline will work in exclusive ownership
mode. In addition, any other pipeline that receives data from a receiver that is attached
to a pipeline with exclusive ownership mode will be also operating in exclusive ownership
mode.

### Exclusive Ownership

In exclusive ownership mode the data is owned exclusively by a particular processor at a
given moment of time and the processor is free to modify the data it owns.

Exclusive ownership mode is only applicable for pipelines that receive data from the
same receiver. If a pipeline is marked to be in exclusive ownership mode then any data
received from a shared receiver will be cloned at the fan-out connector before passing
further to each pipeline. This ensures that each pipeline has its own exclusive copy of
data and the data can be safely modified in the pipeline.

The exclusive ownership of data allows processors to freely modify the data while
they own it (e.g. see `attributesprocessor`). The duration of ownership of the data
by processor is from the beginning of `ConsumeTraceData`/`ConsumeMetricsData` call
until the processor calls the next processor's `ConsumeTraceData`/`ConsumeMetricsData`
function, which passes the ownership to the next processor. After that the processor
must no longer read or write the data since it may be concurrently modified by the
new owner.

Exclusive Ownership mode allows to easily implement processors that need to modify
the data by simply declaring such intent.

### Shared Ownership

In shared ownership mode no particular processor owns the data and no processor is
allowed the modify the shared data.

In this mode no cloning is performed at the fan-out connector of receivers that
are attached to multiple pipelines. In this case all such pipelines will see
the same single shared copy of the data. Processors in pipelines operating in shared
ownership mode are prohibited from modifying the original data that they receive
via `ConsumeTraceData`/`ConsumeMetricsData` call. Processors may only read the data but
must not modify the data.

If the processor needs to modify the data while performing the processing but
does not want to incur the cost of data cloning that Exclusive mode brings then
the processor can declare that it does not modify the data and use any
different technique that ensures original data is not modified. For example,
the processor can implement copy-on-write approach for individual sub-parts of
`TraceData`/`MetricsData` argument. Any approach that does not mutate the
original `TraceData`/`MetricsData` argument (including referenced data, such as
`Node`, `Resource`, `Spans`, etc) is allowed.

If the processor uses such technique it should declare that it does not intend
to modify the original data by setting `MutatesConsumedData=false` in its capabilities
to avoid marking the pipeline for Exclusive ownership and to avoid the cost of
data cloning described in Exclusive Ownership section.

## Ordering Processors

The order processors are specified in a pipeline is important as this is the
order in which each processor is applied to traces and metrics.

### Include/Exclude Spans

The [attribute processor](#attributes) and the [span processor](#span) expose
the option to provide a set of properties of a span to match against to determine
if the span should be included or excluded from the processor. By default, all
spans are processed by the processor.

To configure this option, under `include` and/or `exclude`:
- at least one of `services`, `span_names` or `attributes` is required.

Note: If both `include` and `exclude` are specified, the `include` properties
are checked before the `exclude` properties.

```yaml
{span, attributes}:
    # include and/or exclude can be specified. However, the include properties
    # are always checked before the exclude properties.
    {include, exclude}:
      # At least one of services, span_names or attributes must be specified.
      # It is supported to have more than one specified, but all of the specified
      # conditions must evaluate to true for a match to occur.

      # match_type controls how items in "services" and "span_names" arrays are
      # interpreted. Possible values are "regexp" or "strict".
      # This is a required field.
      match_type: {strict, regexp}

      # services specify an array of items to match the service name against.
      # A match occurs if the span service name matches at least of the items.
      # This is an optional field.
      services: [<item1>, ..., <itemN>]

      # The span name must match at least one of the items.
      # This is an optional field.
      span_names: [<item1>, ..., <itemN>]

      # Attributes specifies the list of attributes to match against.
      # All of these attributes must match exactly for a match to occur.
      # Only match_type=strict is allowed if "attributes" are specified.
      # This is an optional field.
      attributes:
          # Key specifies the attribute to match against.
        - key: <key>
          # Value specifies the exact value to match against.
          # If not specified, a match occurs if the key is present in the attributes.
          value: {value}
```

# Processors

## <a name="attributes"></a>Attributes Processor

The attributes processor modifies attributes of a span. Please refer to
[config.go](attributesprocessor/config.go) for the config spec.

It optionally supports the ability to [include/exclude spans](#includeexclude-spans).

It takes a list of actions which are performed in order specified in the config.
The supported actions are:
- `insert`: Inserts a new attribute in spans where the key does not already exist.
- `update`: Updates an attribute in spans where the key does exist.
- `upsert`: Performs insert or update. Inserts a new attribute in spans where the
  key does not already exist and updates an attribute in spans where the key
  does exist.
- `delete`: Deletes an attribute from a span.
- `hash`: Hashes (SHA1) an existing attribute value.

For the actions `insert`, `update` and `upsert`,
 - `key`  is required
 - one of `value` or `from_attribute` is required
 - `action` is required.
```yaml
  # Key specifies the attribute to act upon.
- key: <key>
  action: {insert, update, upsert}
  # Value specifies the value to populate for the key.
  # The type is inferred from the configuration.
  value: <value>

  # Key specifies the attribute to act upon.
- key: <key>
  action: {insert, update, upsert}
  # FromAttribute specifies the attribute from the span to use to populate
  # the value. If the attribute doesn't exist, no action is performed.
  from_attribute: <other key>
```

For the `delete` action,
 - `key` is required
 - `action: delete` is required.
```yaml
# Key specifies the attribute to act upon.
- key: <key>
  action: delete
```


For the `hash` action,
 - `key` is required
 - `action: hash` is required.
```yaml
# Key specifies the attribute to act upon.
- key: <key>
  action: hash 
```

The list of actions can be composed to create rich scenarios, such as
back filling attribute, copying values to a new key, redacting sensitive information.
The following is a sample configuration.

```yaml
processors:
  attributes/example:
    actions:
      - key: db.table
        action: delete
      - key: redacted_span
        value: true
        action: upsert
      - key: copy_key
        from_attribute: key_original
        action: update
      - key: account_id
        value: 2245
      - key: account_password
        action: delete
      - key: account_email 
        action: hash

```

Refer to [config.yaml](attributesprocessor/testdata/config.yaml) for detailed
examples on using the processor.

## <a name="batch"></a>Batch Processor

The batch processor accepts spans and places them into batches grouped by node
and resource. Batching helps better compress the data and reduce the number of
outgoing connections required to transmit the data. This processor supports
both size and time based batching. Please refer to
[config.go](batchprocessor/config.go) for the config spec.

The following configuration options can be modified:
- `num_tickers` (default = 4): the number of tickers that loop over batch buckets
- `remove_after_ticks` (default = 10): the number of ticks passed without a span arriving for a node at which time batcher is deleted
- `send_batch_size` (default = 8192): size after which a batch will be sent regardless of time
- `tick_time` (default = 1s): interval in which the tickers tick
- `timeout` (default = 1s): time duration after which a batch will be sent regardless of size

Examples:

```yaml
processors:
  batch:
  batch/2:
    num_tickers: 10
    remove_after_ticks: 20
    send_batch_size: 1000
    tick_time: 5s
    timeout: 10s
```

Refer to [config.yaml](batchprocessor/testdata/config.yaml) for detailed
examples on using the processor.


## <a name="queued_retry"></a>Queued Retry Processor

The queued_retry processor uses a bounded queue to relay trace data from the receiver
or previous processor to the next processor. Received trace data is enqueued
immediately if the queue is not full. At the same time, the processor has one
or more workers which consume the trace data in the queue by sending them to
the next processor. If relaying the trace data to the next processor or
exporter in the pipeline fails, the processor retries after some backoff delay
depending on the configuration (see below). Please refer to
[config.go](queuedprocessor/config.go) for the config spec.

The following configuration options can be modified:
- `backoff_delay` (default = 5s): time interval to wait before retrying
- `num_workers` (default = 10): the number of workers that dequeue batches
- `queue_size` (default = 5000): the maximum number of batches allowed before drop
- `retry_on_failure` (default = true): whether to retry on failure or give up and drop

Examples:

```yaml
processors:
  queued_retry/example:
    backoff_delay: 5s
    num_workers: 2
    queue_size: 10
    retry_on_failure: true
```

Refer to [config.yaml](queuedprocessor/testdata/config.yaml) for detailed
examples on using the processor.


## <a name="sampling"></a>Sampling Processor

The sampling processor supports sampling spans. A couple of sampling processors
are provided today and it is straight forward to add others.

### <a name="tail_sampling"></a>Tail Sampling Processor

The tail sampling processor samples traces based on a set of defined policies.
Today, this processor only works with a single instance of the collector.
Technically, trace ID aware load balancing could be used to support multiple
collector instances, but this configuration has not been tested. Please refer to
[config.go](samplingprocessor/tailsamplingprocessor/config.go) for the config spec.

The following configuration options can be modified:
- `decision_wait` (default = 30s): wait time since the first span before making a sampling decision
- `num_traces` (default = 50000): number of traces kept in memory
- `expected_new_traces_per_sec` (default = 0): expected number of new traces (helps in allocating data structures)
- `policies` (default = ): sets the policies used to make a sampling decision

Multiple policies exist today and it is straight forward to add more. These include:
- `always_sample`: sample all traces
- `numeric_attribute`: sample based on number attributes
- `string_attribute`: sample based on string attributes
- `rate_limiting`: sample based on rate

Examples:

```yaml
processors:
  tail_sampling:
    decision_wait: 10s
    num_traces: 100
    expected_new_traces_per_sec: 10
    policies:
      [
          {
            name: test-policy-1,
            type: always_sample
          },
          {
            name: test-policy-2,
            type: numeric_attribute,
            numeric_attribute: {key: key1, min_value: 50, max_value: 100}
          },
          {
            name: test-policy-3,
            type: string_attribute,
            string_attribute: {key: key2, values: [value1, value2]}
          },
          {
            name: test-policy-4,
            type: rate_limiting,
            rate_limiting: {spans_per_second: 35}
         }
      ]
```

Refer to [config.yaml](samplingprocessor/tailsamplingprocessor/testdata/config.yaml) for detailed
examples on using the processor.

### <a name="probabilistic_sampling"></a>Probabilistic Sampling Processor

The probabilistic sampler sets trace sampling by hashing the trace id of each
span and making the sampling decision based on the hashed value. It also
implements the "sampling.priority" [semantic
convention](https://github.com/opentracing/specification/blob/master/semantic_conventions.md#span-tags-table)
as defined by OpenTracing. The "sampling.priority" semantic hash priority over
trace id hashing. Please refer to
[config.go](samplingprocessor/probabilisticprocessor/config.go) for the config spec.

The following configuration options can be modified:
- `hash_seed` (no default): all collectors for a given tier should have the same hash_seed
- `sampling_percentage` (default = 0): the percentage at which traces are sampled; >= 100 samples all traces

Examples:

```yaml
processors:
  probabilistic_sampler:
    hash_seed: 22
    sampling_percentage: 15.3
```

Refer to [config.yaml](samplingprocessor/probabilisticprocessor/testdata/config.yaml) for detailed
examples on using the processor.

## <a name="span"></a>Span Processor

The span processor modifies either the span name or attributes of a span based
on the span name. Please refer to
[config.go](spanprocessor/config.go) for the config spec.

It optionally supports the ability to [include/exclude spans](#includeexclude-spans).

The following actions are supported:

- `name`: Modify the name of attributes within a span

### Name a span

The following settings are required:

- `from_attributes`: The attribute value for the keys are used to create a
new name in the order specified in the configuration.

The following settings can be optionally configured:

- `separator`: A string, which is specified will be used to split values

Note: If renaming is dependent on attributes being modified by the `attributes`
processor, ensure the `span` processor is specified after the `attributes`
processor in the `pipeline` specification.

```yaml
span:
  name:
    # from_attributes represents the attribute keys to pull the values from to generate the
    # new span name.
    from_attributes: [<key1>, <key2>, ...]
    # Separator is the string used to concatenate various parts of the span name.
    separator: <value>
```

Example:

```yaml
span:
  name:
    from_attributes: ["db.svc", "operation"]
    separator: "::"
```

Refer to [config.yaml](spanprocessor/testdata/config.yaml) for detailed
examples on using the processor.

### Extract attributes from span name

Takes a list of regular expressions to match span name against and extract
attributes from it based on subexpressions. Must be specified under the
`to_attributes` section.

The following settings are required:

- `rules` is a list of rules to extract attribute values from span name. The values
in the span name are replaced by extracted attribute names. Each rule in the list
is regex pattern string. Span name is checked against the regex and if the regex
matches then all named subexpressions of the regex are extracted as attributes
and are added to the span. Each subexpression name becomes an attribute name and
subexpression matched portion becomes the attribute value. The matched portion
in the span name is replaced by extracted attribute name. If the attributes
already exist in the span then they will be overwritten. The process is repeated
for all rules in the order they are specified. Each subsequent rule works on the
span name that is the output after processing the previous rule.
- `break_after_match` (default = false): specifies if processing of rules should stop after the first
match. If it is false rule processing will continue to be performed over the
modified span name.

```yaml
span/to_attributes:
  name:
    to_attributes:
      rules:
        - regexp-rule1
        - regexp-rule2
        - regexp-rule3
        ...
      break_after_match: <true|false>

```

Example:

```yaml
# Let's assume input span name is /api/v1/document/12345678/update
# Applying the following results in output span name /api/v1/document/{documentId}/update
# and will add a new attribute "documentId"="12345678" to the span.
span/to_attributes:
  name:
    to_attributes:
      rules:
        - ^\/api\/v1\/document\/(?P<documentId>.*)\/update$
```

Refer to [config.yaml](spanprocessor/testdata/config.yaml) for detailed
examples on using the processor.
