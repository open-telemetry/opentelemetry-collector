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

### Scope within each module

The Collector is already used in production at scale and has been tested in a variety of
environments. The focus of the stabilization is primarily not to add missing features but to ensure
the maintainability of the project and to provide a predictable and consistent experience for
end-users.

In particular when considering enhancement proposals we will focus on:
1. Binary end-users impact above other audiences.
2. Parts of the proposals that imply breaking changes to end-users.
3. Small, predictable or self-contained changes that don't imply a major change in the end-user
   experience.

Additionally, when considering bug reports we will prioritize:
1. [Critical bugs](release.md#bugfix-release-criteria) that affect the stability of the Collector.
2. Regressions from previous behavior caused by 1.0-related changes.

## Out of scope

Explicitly, the following are not in the scope of v1 for the purposes of this document:

* stabilization of additional components/APIs needed by distribution maintainers. Vendors are not the audience
  * This explicitly excludes the `service` and `otelcol` modules, for which we will only guarantee that there are no breaking changes impacting end-users of the binary after 1.0, while Go API only changes will continue to be admissible until these modules are tagged as 1.0.
* Collector Builder
* telemetrygen
* mdatagen
* Operator

Those components are free to pursue v1 at their own pace and may be the focus of future stability work.

## Additional Requirements

The following is a list of requirements for this minimal Collector distribution to be deemed as 1.0:

* The Collector must be observable
  * Metrics and traces should be produced for data in the hot path
  * Metrics should be documented in the end-user documentation
  * Metrics, or a subset of them, should be marked as stable in the documentation
  * Logs should be produced for Collector lifecycle events
  * Stability expectations and lifecycle for telemetry should be documented, so that users can know what they can rely on  for their dashboards and alerts
* The Collector must be scalable
  * Backpressure from the exporter all the way back to the receiver should be supported
  * Queueing must be supported to handle increased loads
  * Performance metrics are in place and follow best practices for benchmarking
  * Individual components must:
    * Have their lifecycle expectations enshrined in tests
    * Have goleak enabled
* End-user documentation should be provided as part of the official project’s documentation under opentelemetry.io, including:
  * Getting started with the Collector
  * Available (stable) components and how to use them
  * Blueprints for common use cases
  * Error scenarios and error propagation
  * Troubleshooting and how to obtain telemetry from the Collector for the purposes of bug reporting
  * Queueing, batching, and handling of backpressure
* The Collector must be supported
  * Processes, workflows and expectations regarding support, bug reporting and questions should be documented.
  * A minimum support period for 1.0 is documented, similarly to [API and SDK](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/versioning-and-stability.md#api-support) stability guarantees.
