# Kafka Receiver

Kafka receiver receives traces from Kafka. Message payload encoding is configurable.

Supported pipeline types: traces, logs

## Getting Started

The following settings are required:

- `protocol_version` (no default): Kafka protocol version e.g. 2.0.0

The following settings can be optionally configured:

- `brokers` (default = localhost:9092): The list of kafka brokers
- `topic` (default = otlp_spans): The name of the kafka topic to read from
- `encoding` (default = otlp/protobuf): The encoding of the payload sent to kafka. See [protocols](../../protocols/README.md) for available encodings.
- `group_id` (default = otel-collector):  The consumer group that receiver will be consuming messages from
- `client_id` (default = otel-collector): The consumer client ID that receiver will use
- `auth`
  - `plain_text`
    - `username`: The username to use.
    - `password`: The password to use
  - `tls`
    - `ca_file`: path to the CA cert. For a client this verifies the server certificate. Should
      only be used if `insecure` is set to true.
    - `cert_file`: path to the TLS cert to use for TLS required connections. Should
      only be used if `insecure` is set to true.
    - `key_file`: path to the TLS key to use for TLS required connections. Should
      only be used if `insecure` is set to true.
    - `insecure` (default = false): Disable verifying the server's certificate
      chain and host name (`InsecureSkipVerify` in the tls config)
    - `server_name_override`: ServerName indicates the name of the server requested by the client
      in order to support virtual hosting.
  - `kerberos`
    - `service_name`: Kerberos service name
    - `realm`: Kerberos realm
    - `use_keytab`:  Use of keytab instead of password, if this is true, keytab file will be used instead of password
    - `username`: The Kerberos username used for authenticate with KDC
    - `password`: The Kerberos password used for authenticate with KDC
    - `config_file`: Path to Kerberos configuration. i.e /etc/krb5.conf
    - `keytab_file`: Path to keytab file. i.e /etc/security/kafka.keytab
- `metadata`
  - `full` (default = true): Whether to maintain a full set of metadata. When
    disabled the client does not make the initial request to broker at the
    startup.
  - `retry`
    - `max` (default = 3): The number of retries to get metadata
    - `backoff` (default = 250ms): How long to wait between metadata retries

Example:

```yaml
receivers:
  kafka:
    protocol_version: 2.0.0
```
