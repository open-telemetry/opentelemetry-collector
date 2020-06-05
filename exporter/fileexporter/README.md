# File Exporter

This exporter will write the pipeline data to a JSON file.
The data is written in Protobuf JSON encoding
(https://developers.google.com/protocol-buffers/docs/proto3#json).
Note that there are no compatibility guarantees for this format, since it
just a dump of internal structures which can be changed over time.
This intended for primarily for debugging Collector without setting up backends.

The following settings are required:

- `path` (no default): where to write information.

Example:

```yaml
exporters:
  file:
    path: ./filename.json
```

The full list of settings exposed for this exporter are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).
