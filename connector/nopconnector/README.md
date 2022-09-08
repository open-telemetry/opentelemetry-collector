# Nop Connector

The `nopconnector` can be used to chain pipelines of the same type together.
For example, it can replicate a signal to multiple pipelines so that each pipeline
can process the signal independently in varying ways. Alternately, it can be used to
merge two pipelines together so they can be processed as one pipeline.

## Supported connection types

Connectors are always used in two or more pipelines. Therefore, support and stability
are defined per _pair of signal types_. The pipeline in which a connector is used as
an exporter is referred to below as the "Exporter pipeline". Likewise, the pipeline in
which the connector is used as a receiver is referred to below as the "Receiver pipeline".

| Exporter pipeline | Receiver pipeline | Stability         |
| ----------------- | ----------------- | ----------------- |
| traces            | traces            | [in development]  |
| metrics           | metrics           | [in development]  |
| logs              | logs              | [in development]  |

[in development]:https://github.com/open-telemetry/opentelemetry-collector#in-development
