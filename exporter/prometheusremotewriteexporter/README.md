This Exporter sends metrics data in Prometheus TimeSeries format to Cortex or any Prometheus Remote Write compatible backend.

_TODO: derive namespace and tenant id from Resource attributes or metric labels_

The following settings are required:

- `endpoint`: protocol:host:port to which the exporter is going to send traces or metrics, using 
the HTTP/HTTPS protocol. 

- `namespace`: suffix to metric name attached to each metric.

The following settings can be optionally configured:
- `headers`: additional headers attached to each HTTP request. `X-Prometheus-Remote-Write-Version` cannot be set by users
and is attached to each request. 
- `insecure` (default = false): whether to enable client transport security for
  the exporter's connection.
- `ca_file`: path to the CA cert. For a client this verifies the server certificate. Should
  only be used if `insecure` is set to true.
- `cert_file`: path to the TLS cert to use for TLS required connections. Should
  only be used if `insecure` is set to true.
- `key_file`: path to the TLS key to use for TLS required connections. Should
  only be used if `insecure` is set to true.
- `timeout` (default = 5s): How long to wait until the connection is close.
- `read_buffer_size` (default = 0): ReadBufferSize for HTTP client.
- `write_buffer_size` (default = 512 * 1024): WriteBufferSize for HTTP client.

Example:

```yaml
exporters:
cortex:
 endpoint: "http://some.url:9411/api/prom/push"
```
The full list of settings exposed for this exporter are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).

_Here is a link to the overall project [design](https://github.com/open-telemetry/opentelemetry-collector/pull/1464)_

File structure:

- `cortex.go`: exporter implementation. Converts and sends OTLP metrics

- `helper.go`: helper functions that cortex.go uses. Performs tasks such as sanitizing label and generating signature string

- `config.go`: configuration struct of the exporter

- `factory.go`: initialization methods for creating default configuration and the exporter
