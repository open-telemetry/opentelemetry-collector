# Count Connector

The `countconnector` can be used to count signals.

## Supported connection types

Connectors are always used in two or more pipelines. Therefore, support and stability
are defined per _pair of signal types_. The pipeline in which a connector is used as
an exporter is referred to below as the "Exporter pipeline". Likewise, the pipeline in
which the connector is used as a receiver is referred to below as the "Receiver pipeline".

| Exporter pipeline | Receiver pipeline | Stability         |
| ----------------- | ----------------- | ----------------- |
| traces            | metrics           | [in development]  |
| metrics           | metrics           | [in development]  |
| logs              | metrics           | [in development]  |

[in development]:https://github.com/open-telemetry/opentelemetry-collector#in-development
