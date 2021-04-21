# Syslog Receiver

Parses Syslogs from tcp/udp using
the [opentelemetry-log-collection](https://github.com/open-telemetry/opentelemetry-log-collection) library.

Supported pipeline types: logs

> :construction: This receiver is in alpha and configuration fields are subject to change.

## Configuration

| Field      | Default          | Description                                                  |
| ---------- | ---------------- | ------------------------------------------------------------ |
| `tcp`      | `nil`               | Defined tcp_input operator. (see the TCP configuration section)  |
| `udp`      |`nil`                | Defined udp_input operator. (see the UDP configuration section)  |
| `protocol`    | required         | The protocol to parse the syslog messages as. Options are `rfc3164` and `rfc5424` |
| `location`    | `UTC`            | The geographic location (timezone) to use when parsing the timestamp (Syslog RFC 3164 only). The available locations depend on the local IANA Time Zone database. [This page](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) contains many examples, such as `America/New_York`. |
| `timestamp`   | `nil`            | An optional [timestamp](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/types/timestamp.md) block which will parse a timestamp field before passing the entry to the output operator                                                                                               |
| `severity`    | `nil`            | An optional [severity](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/docs/types/severity.md) block which will parse a severity field before passing the entry to the output operator
| `attributes`   | {}               | A map of `key: value` labels to add to the entry's attributes    |
| `resource` | {}               | A map of `key: value` labels to add to the entry's resource  |
| `operators`            | []               | An array of [operators](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/README.md#what-operators-are-available). See below for more details |

### Operators

Each operator performs a simple responsibility, such as parsing a timestamp or JSON. Chain together operators to process logs into a desired format.

- Every operator has a `type`.
- Every operator can be given a unique `id`. If you use the same type of operator more than once in a pipeline, you must specify an `id`. Otherwise, the `id` defaults to the value of `type`.
- Operators will output to the next operator in the pipeline. The last operator in the pipeline will emit from the receiver. Optionally, the `output` parameter can be used to specify the `id` of another operator to which logs will be passed directly.
- Only parsers and general purpose operators should be used.

### UDP Configuration

| Field             | Default          | Description                                                                       |
| ---               | ---              | ---                                                                               |
| `listen_address`  | required         | A listen address of the form `<ip>:<port>`                                        |

### TCP Configuration

| Field             | Default          | Description                                                                       |
| ---               | ---              | ---                                                                               |
| `max_buffer_size` | `1024kib`        | Maximum size of buffer that may be allocated while reading TCP input              |
| `listen_address`  | required         | A listen address of the form `<ip>:<port>`                                        |
| `tls`             |                  | An optional `TLS` configuration (see the TLS configuration section)               |

#### TLS Configuration

The `tcp_input` operator supports TLS, disabled by default.

| Field             | Default          | Description                               |
| ---               | ---              | ---                                       |
| `cert_file`       |                  | Path to the TLS cert to use for TLS required connections.       |
| `key_file`        |                  | Path to the TLS key to use for TLS required connections.|
| `ca_file`         |                  | Path to the CA cert. For a client this verifies the server certificate. For a server this verifies client certificates. If empty uses system root CA.  |
| `client_ca_file`  |                  | (optional) Path to the TLS cert to use by the server to verify a client certificate. This sets the ClientCAs and ClientAuth to RequireAndVerifyClientCert in the TLSConfig. Please refer to godoc.org/crypto/tls#Config for more information. |

## Additional Terminology and Features

- An [entry](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/types/entry.md) is the base representation of log data as it moves through a pipeline. All operators either create, modify, or consume entries.
- A [field](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/types/field.md) is used to reference values in an entry.
- A common [expression](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/types/expression.md) syntax is used in several operators. For example, expressions can be used to [filter](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/filter.md) or [route](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/router.md) entries.
- [timestamp](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/types/timestamp.md) parsing is available as a block within all parser operators, and also as a standalone operator. Many common timestamp layouts are supported.
- [severity](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/types/severity.md) parsing is available as a block within all parser operators, and also as a standalone operator. Stanza uses a flexible severity representation which is automatically interpreted by the stanza receiver.

## Example Configurations

TCP Configuration:

```yaml
receivers:
  syslog:
    tcp:
      listen_address: "0.0.0.0:54526"
    protocol: rfc5424
```

UDP Configuration:

```yaml
receivers:
  syslog:
    udp:
      listen_address: "0.0.0.0:54526"
    protocol: rfc3164
    location: UTC
```
