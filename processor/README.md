# Processors
*Note* This documentation is still in progress. For any questions, please reach
out in the [OpenTelemetry Gitter](https://gitter.im/open-telemetry/opentelemetry-service)
or refer to the [issues page](https://github.com/open-telemetry/opentelemetry-service/issues).

## <a name="attributes"></a>Attributes Processor
The attributes processor modifies attributes of a span.

It takes a list of actions which are performed in order specified in the config.
The supported actions are:
- inserts: Inserts a new attribute to spans without the specified key.
- update: Updates an attribute of spans with the specified key.
- upsert: Performs insert or update. Inserts an attribute for spans without
  the specified key and updates an attribute for spans with the specified key.
- delete: Deletes an attribute from a span.

**INSERT is the default action.**


For the actions `insert`, `update` and `upsert`, a `key` must be specified,
 one of `value` or `from_attribute` and `action`.
```yaml
  # Key specifies the attribute to act upon.
- key: <key>
  action: {insert, update, upsert}
  # Value specifies the value to populate for the key.
  value: <value>

  # Key specifies the attribute to act upon.
- key: <key>
  action: {insert, update, upsert}
  # FromAttribute specifies the attribute from the span to use to populate
  # the value. If the attribute doesn't exist, no action is performed.
  from_attribute: <other key>
```

For the `delete` action, a `key` and `action`must be specified.
```yaml
# Key specifies the attribute to act upon.
- key: <key>
  action: delete
```

Please refer to [config.go](processors/attributes/config.go) for the config spec.

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
Refer to [config.yaml](processors/attributes/testdata/config.yaml) for detailed
examples on using the processor.
