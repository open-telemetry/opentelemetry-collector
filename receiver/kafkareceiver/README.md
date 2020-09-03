# Kafka Receiver

Kafka receiver receives traces from Kafka. Message payload encoding is configurable.

The following settings are required:
- `protocol_version` (no default): Kafka protocol version e.g. 2.0.0

The following settings can be optionally configured:
- `brokers` (default = localhost:9092): The list of kafka brokers
- `topic` (default = otlp_spans): The name of the kafka topic to export to
- `encoding` (default = otlp_proto): The encoding of the payload sent to kafka. Available encodings:
  - `otlp_proto`: the payload is deserialized to `ExportTraceServiceRequest`.
  - `jaeger_proto`: the payload is deserialized to a single Jaeger proto `Span`.
  - `jaeger_json`: the payload is deserialized to a single Jaeger JSON Span using `jsonpb`.
  - `zipkin_proto`: the payload is deserialized into Zipkin proto spans.
  - `zipkin_json`: the payload is deserialized into Zipkin V2 JSON spans.
  - `zipkin_thrift`: the payload is deserialized into Zipkin Thrift spans.
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
