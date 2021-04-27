This package provides a common set of marshalers and unmarshalers for various formats to and from OTLP.

# Protocols

| Type | Description | Traces | Metrics | Logs |
|---|---|:---:|:---:|:---:|
| `otlp/protobuf` | OTLP protobuf | ✅ |  | ✅ |
| `zipkin/protobuf` | List of zipkin protobuf spans | ✅ |  | |
| `zipkin/json` | List of Zipkin v2 JSON spans | ✅ |  | |
| `zipkin/thrift` | List of Zipkin Thrift spans | ✅ | | |
| `jaeger/protobuf` | Jaeger `Batch` using protobuf | ✅ | | |
| `jaeger/json` | Jaeger `Batch` using jsonpb | ✅ | | |
