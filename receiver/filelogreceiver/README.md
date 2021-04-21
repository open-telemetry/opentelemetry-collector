# Filelog Receiver

Tails and parses logs from files using the [opentelemetry-log-collection](https://github.com/open-telemetry/opentelemetry-log-collection) library.

Supported pipeline types: logs

> :construction: This receiver is in alpha and configuration fields are subject to change.

## Configuration



| Field                  | Default          | Description                                                                                                        |
| ---                    | ---              | ---                                                                                                                |
| `include`              | required         | A list of file glob patterns that match the file paths to be read                                                  |
| `exclude`              | []               | A list of file glob patterns to exclude from reading                                                               |
| `start_at`             | `end`            | At startup, where to start reading logs from the file. Options are `beginning` or `end`                            |
| `write_to`             | $body            | The body [field](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/types/field.md) written to when creating a new log entry                                  |
| `multiline`            |                  | A `multiline` configuration block. See below for more details                                                      |
| `encoding`             | `nop`            | The encoding of the file being read. See the list of supported encodings below for available options               |
| `include_file_name`    | `true`           | Whether to add the file name as the label `file_name`                                                              |
| `include_file_path`    | `false`          | Whether to add the file path as the label `file_path`                                                              |
| `poll_interval`        | 200ms            | The duration between filesystem polls                                                                              |
| `fingerprint_size`     | `1kb`            | The number of bytes with which to identify a file. The first bytes in the file are used as the fingerprint. Decreasing this value at any point will cause existing fingerprints to forgotten, meaning that all files will be read from the beginning (one time) |
| `max_log_size`         | `1MiB`           | The maximum size of a log entry to read before failing. Protects against reading large amounts of data into memory |
| `max_concurrent_files` | 1024             | The maximum number of log files from which logs will be read concurrently. If the number of files matched in the `include` pattern exceeds this number, then files will be processed in batches. One batch will be processed per `poll_interval` |
| `labels`               | {}               | A map of `key: value` labels to add to the entry's labels                                                          |
| `resource`             | {}               | A map of `key: value` labels to add to the entry's resource                                                        |
| `operators`            | []               | An array of [operators](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/README.md#what-operators-are-available). See below for more details |

Note that _by default_, no logs will be read from a file that is not actively being written to because `start_at` defaults to `end`.

### Operators

Each operator performs a simple responsibility, such as parsing a timestamp or JSON. Chain together operators to process logs into a desired format.

- Every operator has a `type`.
- Every operator can be given a unique `id`. If you use the same type of operator more than once in a pipeline, you must specify an `id`. Otherwise, the `id` defaults to the value of `type`.
- Operators will output to the next operator in the pipeline. The last operator in the pipeline will emit from the receiver. Optionally, the `output` parameter can be used to specify the `id` of another operator to which logs will be passed directly.
- Only parsers and general purpose operators should be used.

### Multiline configuration

If set, the `multiline` configuration block instructs the `file_input` operator to split log entries on a pattern other than newlines.

The `multiline` configuration block must contain exactly one of `line_start_pattern` or `line_end_pattern`. These are regex patterns that
match either the beginning of a new log entry, or the end of a log entry.

### Supported encodings

| Key        | Description
| ---        | ---                                                              |
| `nop`      | No encoding validation. Treats the file as a stream of raw bytes |
| `utf-8`    | UTF-8 encoding                                                   |
| `utf-16le` | UTF-16 encoding with little-endian byte order                    |
| `utf-16be` | UTF-16 encoding with little-endian byte order                    |
| `ascii`    | ASCII encoding                                                   |
| `big5`     | The Big5 Chinese character encoding                              |

Other less common encodings are supported on a best-effort basis. See [https://www.iana.org/assignments/character-sets/character-sets.xhtml](https://www.iana.org/assignments/character-sets/character-sets.xhtml) for other encodings available.

## Additional Terminology and Features

- An [entry](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/types/entry.md) is the base representation of log data as it moves through a pipeline. All operators either create, modify, or consume entries.
- A [field](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/types/field.md) is used to reference values in an entry.
- A common [expression](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/types/expression.md) syntax is used in several operators. For example, expressions can be used to [filter](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/filter.md) or [route](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/router.md) entries.
- [timestamp](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/types/timestamp.md) parsing is available as a block within all parser operators, and also as a standalone operator. Many common timestamp layouts are supported.
- [severity](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/types/severity.md) parsing is available as a block within all parser operators, and also as a standalone operator. Stanza uses a flexible severity representation which is automatically interpreted by the stanza receiver.


## Example - Tailing a simple json file

Receiver Configuration
```yaml
receivers:
  filelog:
    include: [ /var/log/myservice/*.json ]
    operators:    
      - type: json_parser
        timestamp:
          parse_from: time
          layout: '%Y-%m-%d %H:%M:%S'
```
