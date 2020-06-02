# zPages

Enables an extension that serves zPages, an HTTP endpoint that provides live
data for debugging different components that were properly instrumented for such.
All core exporters and receivers provide some zPage instrumentation.

The following settings are required:

- `endpoint` (default = localhost:55679): Specifies the HTTP endpoint that serves
zPages. Use localhost:<port> to make it available only locally, or ":<port>" to
make it available on all network interfaces.

Example:
```yaml
extensions:
  zpages:
```

The full list of settings exposed for this exporter are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).
