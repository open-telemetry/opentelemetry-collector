# Long-term Roadmap

This is the long-term, draft Roadmap. Note that this a vision document that reflects our
current desires. It is not a commitment to implement everything listed in this roadmap.
The primary purpose of this document is to ensure all contributors work in alignment.
As our vision changes over time maintainers reserve the right to modify this roadmap,
add and _remove_ items from it.

Description|Status|Links|
-----------|------|-----|
**Testing**|
Add metric testing to the testbed|Done|
Implement matrix testing in the testbed and convert benchmarks for receivers and exporters to use it|Done|
Implement E2E tests for all core formats (Jaeger, Zipkin, OpenCensus, OTLP, Prometheus)| |[#482](https://github.com/open-telemetry/opentelemetry-collector/issues/482)
Improve comparison with baseline performance and fail CI build on degradations| |
Add code coverage collection to Contrib repository| |
Add 2-stage build pipeline: separate build and certification jobs| |
| |
**Performance**|
Convert internal memory representation to OTLP Protobuf| |[#478](https://github.com/open-telemetry/opentelemetry-collector/issues/478)
Research: explore Fast pass-through mode| |[Partial Protobufs](https://blog.najaryan.net/posts/partial-protobuf-encoding/)
| |
**Observability**|
Add pipeline metrics| |[observability.md](observability.md)
High-level observations|Draft|[#485](https://github.com/open-telemetry/opentelemetry-collector/issues/485)
Research: Explore ways to expose pipelines state as a graph| |[#486](https://github.com/open-telemetry/opentelemetry-collector/issues/486)
| |
**New Formats**|
OTLP Receiver and Exporter| |[#480](https://github.com/open-telemetry/opentelemetry-collector/issues/480) [#479](https://github.com/open-telemetry/opentelemetry-collector/issues/479)
Research: OTLP over HTTP 1.1| |
Add Logs and Events as new data type|
| |
**5 Min to Value**|
Distribution packages for most common targets (e.g. Docker image, RPM, etc)|
Detection and collection of environment metrics and tags (AWS, k8s, etc)|
| |
**Other Features**|
Graceful shutdown (pipeline draining)| |[#483](https://github.com/open-telemetry/opentelemetry-collector/issues/483)
