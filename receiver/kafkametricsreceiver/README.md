# Kafka Metrics Receiver

Kafka metrics receiver collects kafka metrics (brokers, topics, partitions, consumer groups) from kafka server,
converting into otlp.

## Getting Started

Required settings (no defaults):

- `protocol_version`: Kafka protocol version
- `scrapers`: any combination of the following scrapers can be enabled.
    - `topics`
    - `consumers`
    - `brokers`

Optional Settings (with defaults):

- `brokers` (default = localhost:9092): the list of brokers to read from.
- `topic_match` (default = *): regex pattern of topics to filter for metrics collection.
- `group_match` (default = *): regex pattern of consumer groups to filter on for metrics.
- `client_id` (default = otel-metrics-receiver): consumer client id
- `collection_interval` (default = 1m): frequency of metric collection/scraping.
- `auth` (default none)
    - `plain_text`
        - `username`: The username to use.
        - `password`: The password to use
    - `tls`
        - `ca_file`: path to the CA cert. For a client this verifies the server certificate. Should only be used
          if `insecure` is set to true.
        - `cert_file`: path to the TLS cert to use for TLS required connections. Should only be used if `insecure` is
          set to true.
        - `key_file`: path to the TLS key to use for TLS required connections. Should only be used if `insecure` is set
          to true.
        - `insecure` (default = false): Disable verifying the server's certificate chain and host
          name (`InsecureSkipVerify` in the tls config)
        - `server_name_override`: ServerName indicates the name of the server requested by the client in order to
          support virtual hosting.
    - `kerberos`
        - `service_name`: Kerberos service name
        - `realm`: Kerberos realm
        - `use_keytab`:  Use of keytab instead of password, if this is true, keytab file will be used instead of
          password
        - `username`: The Kerberos username used for authenticate with KDC
        - `password`: The Kerberos password used for authenticate with KDC
        - `config_file`: Path to Kerberos configuration. i.e /etc/krb5.conf
        - `keytab_file`: Path to keytab file. i.e /etc/security/kafka.keytab

## Examples:

Basic configuration with all scrapers:

```yaml
receivers:
  kafkametrics:
    protocol_version: 2.0.0
    scrapers:
      - brokers
      - topics
      - consumers
```
