# Prometheus Exporter

Exports data to a [Prometheus](https://prometheus.io/) back-end.

Supported pipeline types: metrics

## Getting Started

The following settings are required:

- `endpoint` (no default): Where to send metric data

The following settings can be optionally configured:

- `constlabels` (no default): key/values that are applied for every exported metric.
- `namespace` (no default): if set, exports metrics under the provided value.
- `send_timestamps` (default = `false`): if true, sends the timestamp of the underlying
  metric sample in the response.

Example:

```yaml
exporters:
  prometheus:
    endpoint: "1.2.3.4:1234"
    namespace: test-space
    const_labels:
      label1: value1
      "another label": spaced value
    send_timestamps: true
```
