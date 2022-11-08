# Long-term Roadmap

This long-term roadmap (draft) is a vision document that reflects our
current desires. It is not a commitment to implement everything listed in this roadmap.
The primary purpose of this document is to ensure that all contributors work in alignment.
As our vision changes over time, maintainers reserve the right to add, modify, and _remove_
items from this roadmap.

Description|Status|Links|
-----------|------|-----|
**Testing**|
Metrics correctness tests|Done|[#652](https://github.com/open-telemetry/opentelemetry-collector/issues/652)
| |
**New Formats**|
Complete OTLP/HTTP support|Done|[#882](https://github.com/open-telemetry/opentelemetry-collector/issues/882)
Add logs support for all primary core processors (attributes, batch, k8sattributes, etc)| Done |
| |
**5 Min to Value**|
Distribution packages for most common targets (e.g. Docker, RPM, Windows, etc)| Done | https://github.com/open-telemetry/opentelemetry-collector-releases/releases |
Detection and collection of environment metrics and tags on AWS| Beta| https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/resourcedetectionprocessor |
Detection and collection of k8s telemetry| Beta | https://pkg.go.dev/github.com/open-telemetry/opentelemetry-collector-contrib/processor/k8sattributesprocessor |
Host metric collection|Beta| https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/hostmetricsreceiver |
Support more application-specific metric collection (e.g. Kafka, Hadoop, etc) | In Progress | https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver|
| |
**Other Features**|
Graceful shutdown (pipeline draining)|Done|[#483](https://github.com/open-telemetry/opentelemetry-collector/issues/483)
Deprecate queue retry processor and enable queuing per exporter by default|Done|[#1721](https://github.com/open-telemetry/opentelemetry-collector/issues/1721)

At this time, much of the effort from the OpenTelemetry Collector SIG is focused on achieving GA status across various packages. See additional details in the [GA roadmap](ga-roadmap.md) document.
