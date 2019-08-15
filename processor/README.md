# Processors
*Note* This documentation is still in progress. For any questions, please reach
out in the [OpenTelemetry Gitter](https://gitter.im/open-telemetry/opentelemetry-service)
or refer to the [issues page](https://github.com/open-telemetry/opentelemetry-service/issues).

Supported processors (sorted alphabetically):
- [Add Attributes Processor](#add-addtributes)
- [Attribute Key Processor](#attribute-key)
- [Multi-Consumer Processor](#multi-consumer)
- [Node Batcher Processor](#node-batcher)
- [Probabilistic Sampler Processor](#probabilistic-sampler)
- [Queued Processor](#queued)
- [Span Rename Processor](#span-rename)
- [Tail Sampling Processor](#tail-sampling)

## <a name="add-addtributes""></a>Add Attributes Processor
<FILL ME IN - I'M LONELY!>

## <a name="attribute-key"></a>Attribute Key Processor
<FILL ME IN - I'M LONELY!>

## <a name="multi-consumer"></a>Multi-Consumer Processor
<FILL ME IN - I'M LONELY!>

## <a name="node-batcher"></a>Node Batcher Processor
<FILL ME IN - I'M LONELY!>

## <a name="probabilistic-sampler"></a>Probabilistic Sampler Processor
<FILL ME IN - I'M LONELY!>

## <a name="queued"></a>Queued Processor
<FILL ME IN - I'M LONELY!>

## <a name="span"></a>Span Processor
The span processor modifies top level settings of a span. Currently, only 
renaming a span is supported. 

### Rename a span
It takes a list of `keys` and an optional `separator` string. The attribute
value for the keys are used to create a new name in the order specified in the
configuration. If a separator is specified, it will separate values.

For more information, refer to [config.go](span/config.go)
```yaml
span:
  rename:
    # Keys represents the attribute keys to pull the values from to generate the
    # new span name.
    keys: [<key1>, <key2>, ...]
    # Separator is the string used to concatenate various parts of the span name.
    separator: <value>
```

### Example configuration
For more examples with detailed comments, refer to [config.yaml](span/testdata/config.yaml)
```yaml
span:
  rename:
    keys: ["db.svc", "operation"]
    separator: "::"
```

## <a name="tail-sampling"></a>Tail Sampling Processor
<FILL ME IN - I'M LONELY!>
