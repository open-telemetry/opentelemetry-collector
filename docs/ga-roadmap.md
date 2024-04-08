# Collector v1 Roadmap

This document contains the roadmap for the Collector. The main goal of this roadmap is to provide clarity on the areas of focus in order to release a v1 of the Collector.

## Proposal

The proposed approach to delivering a stable release of the OpenTelemetry Collector is to produce a distribution of the Collector that contains a minimum set of components which have been stabilized. By doing so, the project contributors will ensure dependencies of those components have also been released under a stable version.

The proposed distribution is set to include the following components only:

- OTLP receiver
- OTLP exporter
- OTLP HTTP exporter

These modules depend on a list of other modules, the full list is available in issue [#9375](https://github.com/open-telemetry/opentelemetry-collector/issues/9375).

All stabilized modules will conform to the API expectations outlined in the [VERSIONING.md](../VERSIONING.md) document.

## Out of scope

Explicitly, the following are not in the scope of v1 for the purposes of this document:

* stabilization of additional components/APIs needed by distribution maintainers. Vendors are not the audience
* Collector Builder
* telemetrygen
* mdatagen
* Operator

Those components are free to pursue v1 at their own pace and may be the focus of future stability work.

## Additional Requirements

The following is a list of requirements for this minimal Collector distribution to be deemed as 1.0:

* The Collector MUST be observable
  * Metrics and traces SHOULD be produced for data in the hot path
  * Metrics should be documented in the end-user documentation
  * Metrics, or a subset of them, should be marked as stable in the documentation
  * Logs SHOULD be produced for Collector lifecycle events
  * Telemetry should be stable, so that users can rely on that on for their dashboards and alerts
* The Collector should correctly handle backpressure and queueing
  * Support backpressure from the exporter all the way back to the receiver
  * Setting performance metrics in place and following best practices for benchmarking, load testing
  * Enshrine in tests (generated for contrib) lifecycle expectations of components
* End-user documentation should be provided as part of the official project’s documentation under opentelemetry.io, including:
  * Getting started with the Collector
  * Available (stable) components and how to use them
  * Blueprints for common use cases
  * Error scenarios and error propagation
  * Troubleshooting and how to obtain telemetry from the Collector for the purposes of bug reporting
  * Queueing, batching, and handling of backpressure
* The Collector MUST be supported
  * Processes, workflows and expectations regarding support, bug reporting and questions should be documented.
  * We need to have a support policy in place for 1.0 with an end of life date.
