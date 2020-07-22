# Kafka Exporter

Kafka exporter exports traces to Kafka.

The following settings are required:
- `protocol_version` (no default): Kafka protocol version e.g. 2.0.0

The following settings can be optionally configured:
- `brokers` (default = localhost:9092): The list of kafka brokers
- `topic` (default = otlp_spans): The name of the kafka topic to export to

Example configuration:

```yaml
exporters:
  kafka:
    brokers:
      - localhost:9092
    protocol_version: 2.0.0
```
