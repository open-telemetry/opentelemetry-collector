# OpenTelemetry Service
The OpenTelemetry service is a set of components that can collect traces, metrics and eventually other telemetry data (e.g. logs) from processes instrumented by OpenTelementry or other monitoring/tracing libraries (Jaeger, Prometheus, etc.), do aggregation and smart sampling, and export traces and metrics to one or more monitoring/tracing backends. The service will allow to enrich and transform collected telemetry (e.g. add additional attributes or scrab personal information).

The OpenTelemetry service has two primary modes of operation: Agent (a locally running daemon) and Collector (a standalone running service).

## Vision

We have a long-term vision for OpenTelemetry Agent that guides us and helps to decide what features we implement and what the priorities are. See [docs/VISION.md](docs/VISION.md).