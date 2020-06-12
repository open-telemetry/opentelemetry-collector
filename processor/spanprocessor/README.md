# Span Processor

Supported pipeline types: traces

The span processor modifies either the span name or attributes of a span based
on the span name. Please refer to
[config.go](./config.go) for the config spec.

It optionally supports the ability to [include/exclude spans](../README.md#includeexclude-spans).

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

Refer to [config.yaml](./testdata/config.yaml) for detailed
examples on using the processor.

### Extract attributes from span name

Takes a list of regular expressions to match span name against and extract
attributes from it based on subexpressions. Must be specified under the
`to_attributes` section.

The following settings are required:

- `rules`: A list of rules to extract attribute values from span name. The values
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

Refer to [config.yaml](./testdata/config.yaml) for detailed
examples on using the processor.
