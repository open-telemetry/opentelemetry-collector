# HTTP Configuration Settings

HTTP exposes a [variety of settings](https://golang.org/pkg/net/http/).
Several of these settings are available for configuration within individual
receivers or exporters.

## Client Configuration

[Exporters](https://github.com/open-telemetry/opentelemetry-collector/blob/main/exporter/README.md)
leverage client configuration.

Note that client configuration supports TLS configuration, the
configuration parameters are also defined under `tls` like server
configuration. For more information, see [configtls
README](../configtls/README.md).

- `endpoint`: address:port
- [`tls`](../configtls/README.md)
- `headers`: name/value pairs added to the HTTP request headers
- [`read_buffer_size`](https://golang.org/pkg/net/http/#Transport)
- [`timeout`](https://golang.org/pkg/net/http/#Client)
- [`write_buffer_size`](https://golang.org/pkg/net/http/#Transport)
- `compression`: Compression type to use among `gzip`, `zstd`, `snappy`, `zlib`, and `deflate`.
  - look at the documentation for the server-side of the communication.
  - `none` will be treated as uncompressed, and any other inputs will cause an error.
- [`max_idle_conns`](https://golang.org/pkg/net/http/#Transport)
- [`max_idle_conns_per_host`](https://golang.org/pkg/net/http/#Transport)
- [`max_conns_per_host`](https://golang.org/pkg/net/http/#Transport)
- [`idle_conn_timeout`](https://golang.org/pkg/net/http/#Transport)

Example:

```yaml
exporter:
  otlp:
    endpoint: otelcol2:55690
    tls:
      ca_file: ca.pem
      cert_file: cert.pem
      key_file: key.pem
    headers:
      test1: "value1"
      "test 2": "value 2"
    compression: zstd
```

## Server Configuration

[Receivers](https://github.com/open-telemetry/opentelemetry-collector/blob/main/receiver/README.md)
leverage server configuration.

- [`cors_allowed_origins`](https://github.com/rs/cors): An empty list means
  that CORS is not enabled at all. A wildcard can be used to match any origin
  or one or more characters of an origin.
- [`cors_allowed_headers`](https://github.com/rs/cors): When CORS is enabled,
  can be used to specify an optional list of allowed headers. By default, it includes `Accept`, 
  `Content-Type`, `X-Requested-With`. `Origin` is also always
  added to the list. A wildcard (`*`) can be used to match any header.
- `endpoint`: Valid value syntax available [here](https://github.com/grpc/grpc/blob/master/doc/naming.md)
- [`tls`](../configtls/README.md)

Example:

```yaml
receivers:
  otlp:
    cors_allowed_origins:
    - https://foo.bar.com
    - https://*.test.com
    cors_allowed_headers:
    - ExampleHeader
    endpoint: 0.0.0.0:55690
    protocols:
      http:
```
