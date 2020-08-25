# Stanza Receiver

Tails and parses logs from a wide variety of sources using the [stanza](https://github.com/observIQ/stanza/tree/master/docs) log processor.

## Input Sources

Stanza supports pre-defined log sources for dozens of [specific technologies](https://github.com/observIQ/stanza-plugins/tree/master/plugins).

It can also be easily configured to tail and parse any structured or unstructured log file, Windows Event Log, and journald. It can also receive arbitrary logs via TCP and UDP.

## Required Parameters

- `pipeline` is an array of [operators](https://github.com/observIQ/stanza/blob/master/docs/README.md#what-operators-are-available). Each operator performs a simple responsibility, such as reading from a file, or parsing JSON. Chain together operators to process logs into a desired format.

## Optional Parameters

- `plugin_dir` is the path to a directory which contains `stanza` [plugins](https://github.com/observIQ/stanza/blob/master/docs/plugins.md). Plugins are parameterized pipelines that are designed for specific use cases.
- `offsets_file` is the path to a file that `stanza` will use to remember where it left off when reading from files or other persistent input sources. If specified, `stanza` will create and manage this file.

## Operator Basics

- Every operator has a `type`.
- Every operator can be given a unique `id`. If you use the same type of operator more than once in a pipeline, you must specify an `id`. Otherwise, the `id` defaults to the value of `type`.
- Operators will output to the next operator in the pipeline. The last operator in the pipeline will emit from the receiver. Optionally, the `output` parameter can be used to specify the `id` of another operator to which logs will be passed directly.

## Additional Terminology and Features

- An [entry](https://github.com/observIQ/stanza/blob/master/docs/types/entry.md) is the base representation of log data as it moves through a pipeline. All operators either create, modify, or consume entries.
- A [field](https://github.com/observIQ/stanza/blob/master/docs/types/field.md) is used to reference values in an entry.
- A common [expression](https://github.com/observIQ/stanza/blob/master/docs/types/expression.md) syntax is used in several operators. For example, expressions can be used to [filter](https://github.com/observIQ/stanza/blob/master/docs/operators/filter.md) or [route](https://github.com/observIQ/stanza/blob/master/docs/operators/router.md) entries.
- [timestamp](https://github.com/observIQ/stanza/blob/master/docs/types/timestamp.md) parsing is available as a block within all parser operators, and also as a standalone operator. Many common timestamp layouts are supported.
- [severity](https://github.com/observIQ/stanza/blob/master/docs/types/severity.md) parsing is available as a block within all parser operators, and also as a standalone operator. Stanza uses a flexible severity representation which is automatically interpreted by the stanza receiver.


## Example - Tailing a simple json file

Receiver Configuration 
```yaml
receivers:
  stanza:
    pipeline:
      - type: file_input
        include: [ /var/log/myservice/*.json ]
      - type: json_parser
        timestamp:
          parse_from: time
          layout: '%Y-%m-%d %H:%M:%S'
```