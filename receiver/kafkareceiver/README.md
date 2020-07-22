# Kafka Receiver

Kafka receiver receives traces from Kafka.

Message payloads are deserialized into OTLP `ExportTraceServiceRequest`.

The following settings are required:
- `protocol_version` (no default): Kafka protocol version e.g. 2.0.0

The following settings can be optionally configured:
- `brokers` (default = localhost:9092): The list of kafka brokers
- `topic` (default = otlp_spans): The name of the kafka topic to export to
- `group_id` (default = otel-collector):  The consumer group that receiver will be consuming messages from
- `client_id` (default = otel-collector): The consumer client ID that receiver will use
- `metadata`
  - `full` (default = true): Whether to maintain a full set of metadata. 
           When disabled the client does not make the initial request to broker at the startup.
  - `retry`
    - `max` (default = 3): The number of retries to get metadata
    - `backoff` (default = 250ms): How long to wait between metadata retries

Example configuration:

```yaml
receivers:
  kafka:
    brokers:
      - localhost:9092
    protocol_version: 2.0.0
```
