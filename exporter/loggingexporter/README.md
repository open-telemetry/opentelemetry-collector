# Logging Exporter

Exports data to the console via zap.Logger.

Supported pipeline types: traces, metrics, logs

## Getting Started

The following settings are optional:

- `loglevel` (default = `info`): the log level of the logging export
  (debug|info|warn|error). When set to `debug`, pipeline data is verbosely
  logged.
- `format` (default = `text`): the serialization format of the logging export.
  Valid values are `text`, `json` and `jsonstream`. When set to `json`, pipeline
  data is serialized and logged as JSON. Both `text` and `json` formats emit
  multiple elements per line (e.g. with batch processing) and therefore are not
  suitable for use with `grep`, unlikea the `jsonstream` format that splits each
  batch element into multi-line JSON sequence.
- `sampling_initial` (default = `2`): number of messages initially logged each
  second.
- `sampling_thereafter` (default = `500`): sampling rate after the initial
  messages are logged (every Mth message is logged). Refer to [Zap
  docs](https://godoc.org/go.uber.org/zap/zapcore#NewSampler) for more details.
  on how sampling parameters impact number of messages.

Example:

```yaml
exporters:
  logging:
    loglevel: debug
    format: jsonstream
    sampling_initial: 5
    sampling_thereafter: 200
```
