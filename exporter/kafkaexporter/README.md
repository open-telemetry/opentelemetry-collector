# Kafka Exporter

Kafka exporter exports traces to Kafka. This exporter uses a synchronous producer
that blocks and does not batch messages, therefore it should be used with batch and queued retry
processors for higher throughput and resiliency.
 
Message payloads are serialized OTLP `ExportTraceServiceRequest`.

The following settings are required:
- `protocol_version` (no default): Kafka protocol version e.g. 2.0.0

The following settings can be optionally configured:
- `brokers` (default = localhost:9092): The list of kafka brokers
- `topic` (default = otlp_spans): The name of the kafka topic to export to
- `metadata.full` (default = true): Whether to maintain a full set of metadata. 
                                    When disabled the client does not make the initial request to broker at the startup.
- `metadata.retry.max` (default = 3): The number of retries to get metadata
- `metadata.retry.backoff` (default = 250ms): How long to wait between metadata retries

Example configuration:

```yaml
exporters:
  kafka:
    brokers:
      - localhost:9092
    protocol_version: 2.0.0
```
