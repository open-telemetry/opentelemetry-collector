# Long-term Roadmap

This long-term roadmap (draft) is a vision document that reflects our
current desires. It is not a commitment to implement everything listed in this roadmap.
The primary purpose of this document is to ensure that all contributors work in alignment.
As our vision changes over time, maintainers reserve the right to add, modify, and _remove_
items from this roadmap.

Description|Status|Links|
-----------|------|-----|
**Testing**|
Metrics correctness tests|In progress|[#652](https://github.com/open-telemetry/opentelemetry-collector/issues/652)
| |
**New Formats**|
Complete OTLP/HTTP support| |[#882](https://github.com/open-telemetry/opentelemetry-collector/issues/882)
Add logs support for all primary core processors (attributes, batch, k8s_tagger, etc)|In progress|
| |
**5 Min to Value**|
Distribution packages for most common targets (e.g. Docker, RPM, Windows, etc)|
Detection and collection of environment metrics and tags on AWS||
Detection and collection of k8s telemetry|In progress|
Host metric collection|In progress|
Support more application-specific metric collection (e.g. Kafka, Hadoop, etc)
| |
**Other Features**|
Graceful shutdown (pipeline draining)| |[#483](https://github.com/open-telemetry/opentelemetry-collector/issues/483)
Deprecate queue retry processor and enable queuing per exporter by default||[#1721](https://github.com/open-telemetry/opentelemetry-collector/issues/1721)
