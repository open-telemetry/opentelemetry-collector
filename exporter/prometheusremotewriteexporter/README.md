# Prometheus Remote Write Exporter

This Exporter sends metrics data in Prometheus TimeSeries format to Cortex or any Prometheus [remote write compatible backend](https://prometheus.io/docs/operating/integrations/).

Non-cumulative monotonic, histogram, and summary OTLP metrics are dropped by this exporter. 

The following settings are required:
- `endpoint`: protocol:host:port to which the exporter is going to send traces or metrics, using the HTTP/HTTPS protocol. 

The following settings can be optionally configured:
- `namespace`: prefix attached to each exported metric name.
- `headers`: additional headers attached to each HTTP request. If `X-Prometheus-Remote-Write-Version` is set by user, its value must be `0.1.0`
- `insecure` (default = false): whether to enable client transport security for the exporter's connection.
- `ca_file`: path to the CA cert. For a client this verifies the server certificate. Should only be used if `insecure` is set to false.
- `cert_file`: path to the TLS cert to use for TLS required connections. Should only be used if `insecure` is set to false.
- `key_file`: path to the TLS key to use for TLS required connections. Should only be used if `insecure` is set to false.
- `timeout` (default = 5s): How long to wait until the connection is close.
- `read_buffer_size` (default = 0): ReadBufferSize for HTTP client.
- `write_buffer_size` (default = 512 * 1024): WriteBufferSize for HTTP client.

Example:

```yaml
exporters:
prometheusremotewrite:
 endpoint: "http://some.url:9411/api/prom/push"
```
The full list of settings exposed for this exporter are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).

_Here is a link to the overall project [design](./DESIGN.md)_
