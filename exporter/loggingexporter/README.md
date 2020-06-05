# Logging Exporter

Exports traces and/or metrics to the console via zap.Logger. This includes generic information
about the package (with `info` loglevel) or details of the trace (when `debug` is set)

The following settings can be configured:

- `loglevel`: the log level of the logging export (debug|info|warn|error). Default is `info`. When it is set to `debug`, 
the trace related data (e.g. node, attributes, spans, metadata) are verbosely logged.
- `sampling_initial`: number of messages initially logged each second. Default is 2. 
- `sampling_thereafter`: sampling rate after the initial messages are logged (every Mth message 
is logged). Default is 500.  Refer to [Zap docs](https://godoc.org/go.uber.org/zap/zapcore#NewSampler) for 
more details on how sampling parameters impact number of messages.

Example:

```yaml
exporters:
  logging:
    loglevel: info
    sampling_initial: 5
    sampling_thereafter: 200
```

The full list of settings exposed for this exporter are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).
