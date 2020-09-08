# Zipkin Exporter

Exports trace data to a [Zipkin](https://zipkin.io/) back-end.

The following settings are required:

- `endpoint` (no default): URL to which the exporter is going to send Zipkin trace data.
- `format` (default = JSON): The format to sent events in. Can be set to JSON or proto.

The following settings can be optionally configured:

- `insecure` (default = false): whether to enable client transport security for
  the exporter's connection.
- `ca_file` path to the CA cert. For a client this verifies the server certificate. Should
  only be used if `insecure` is set to false.
- `cert_file` path to the TLS cert to use for TLS required connections. Should
  only be used if `insecure` is set to false.
- `key_file` path to the TLS key to use for TLS required connections. Should
  only be used if `insecure` is set to false.
- `defaultservicename` (default = <missing service name>): What to name services missing this information.
- `timeout` (default = 5s): How long to wait until the connection is close.
- `read_buffer_size` (default = 0): ReadBufferSize for HTTP client.
- `write_buffer_size` (default = 512 * 1024): WriteBufferSize for HTTP client.

Example:

```yaml
exporters:
zipkin:
 endpoint: "http://some.url:9411/api/v2/spans"
```

The full list of settings exposed for this exporter are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).
