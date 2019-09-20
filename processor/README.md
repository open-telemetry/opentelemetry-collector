# Processors
*Note* This documentation is still in progress. For any questions, please reach
out in the [OpenTelemetry Gitter](https://gitter.im/open-telemetry/opentelemetry-service)
or refer to the [issues page](https://github.com/open-telemetry/opentelemetry-service/issues).

Supported processors (sorted alphabetically):
- [Attributes Processor](#attributes)
- [Node Batcher Processor](#node-batcher)
- [Probabilistic Sampler Processor](#probabilistic_sampler)
- [Queued Processor](#queued)
- [Span Processor](#span)
- [Tail Sampling Processor](#tail_sampling)

## Ordering Processors
The order processors are specified in a pipeline is important as this is the
order in which each processor is applied to traces.

## <a name="attributes"></a>Attributes Processor
The attributes processor modifies attributes of a span.

It takes a list of actions which are performed in order specified in the config.
The supported actions are:
- insert: Inserts a new attribute in spans where the key does not already exist.
- update: Updates an attribute in spans where the key does exist.
- upsert: Performs insert or update. Inserts a new attribute in spans where the
  key does not already exist and updates an attribute in spans where the key
  does exist.
- delete: Deletes an attribute from a span.

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

Please refer to [config.go](attributesprocessor/config.go) for the config spec.

### Include/Exclude Spans
It is optional to provide a set of properties of a span to match against to determine
if the span should be included or excluded from the processor. By default, all
spans are processed by the processor. 

To configure this option, under `include` and/or `exclude`:
- at least one of or both `services` and `attributes` is required.

Note: If both `include` and `exclude` are specified, the `include` properties
are checked before the `exclude` properties.

```yaml
attributes:
    # include and/or exclude can be specified. However, the include properties
    # are always checked before the exclude properties.
    {include, exclude}:
      # At least one of services or attributes must be specified. It is supported
      # to have both specified, but both `services` and `attributes` must evaluate
      # to true for a match to occur.
    
      # Services specify the list of service name to match against.
      # A match occurs if the span service name is in this list.
      # Note: This is an optional field.
      services: [<key1>, ..., <keyN>]
      # Attributes specifies the list of attributes to match against.
      # All of these attributes must match exactly for a match to occur.
      # Note: This is an optional field.
      attributes:
          # Key specifies the attribute to match against.
        - key: <key>
          # Value specifies the exact value to match against.
          # If not specified, a match occurs if the key is present in the attributes.
          value: {value} 
```

### Example
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

```
Refer to [config.yaml](attributesprocessor/testdata/config.yaml) for detailed
examples on using the processor.

## <a name="node-batcher"></a>Node Batcher Processor
<FILL ME IN - I'M LONELY!>

## <a name="probabilistic_sampler"></a>Probabilistic Sampler Processor
<FILL ME IN - I'M LONELY!>

## <a name="queued"></a>Queued Processor
<FILL ME IN - I'M LONELY!>

## <a name="span"></a>Span Processor
The span processor modifies top level settings of a span. Currently, only
renaming a span is supported.

### Name a span
It takes a list of `from_attributes` and an optional `separator` string. The
attribute value for the keys are used to create a new name in the order
specified in the configuration. If a separator is specified, it will separate
values.

If renaming is dependent on attributes being modified by the `attributes`
processor, ensure the `span` processor is specified after the `attributes`
processor in the `pipeline` specification.

For more information, refer to [config.go](spanprocessor/config.go)
```yaml
span:
  name:
    # from_attributes represents the attribute keys to pull the values from to generate the
    # new span name.
    from_attributes: [<key1>, <key2>, ...]
    # Separator is the string used to concatenate various parts of the span name.
    separator: <value>
```

### Example configuration
For more examples with detailed comments, refer to [config.yaml](spanprocessor/testdata/config.yaml)
```yaml
span:
  name:
    from_attributes: ["db.svc", "operation"]
    separator: "::"
```

## <a name="tail_sampling"></a>Tail Sampling Processor
<FILL ME IN - I'M LONELY!>
