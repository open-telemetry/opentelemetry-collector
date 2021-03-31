# Collector GA Roadmap

This document defines the roadmap followed by the OpenTelemetry Collector,
along with tentative dates and requirements for GA (stability).

In this document, the term “OpenTelemetry Collector packages" refers to all the golang
modules and packages that are part of the “OpenTelemetry Collector” ecosystem which
include the [core](https://github.com/open-telemetry/opentelemetry-collector) and
[contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib).

In this document, the terms "OpenTelemetry Collector" and "Collector" both specifically
refer to the entire OpenTelemetry Collector ecosystem’s including core and contrib.
These terms do not refer to the specification or the Client libraries in this document.

## Current Status

The OpenTelemetry Collector ecosystem right now has a lot of packages that are in different
stages of stability (experimental, alpha, beta, etc.). All these packages have different
public APIs/Interfaces (e.g. code API, configuration, etc.).

A significant amount of legacy code was inherited from the Collector's ancestor
[OpenCensus Service](https://github.com/census-instrumentation/opencensus-service), since then
the Collector changed the internal data model and other significant changes were made.

Trying to mark the entire ecosystem GA, at the same moment, will be a significant effort and
will take a significant amount of time.

## Proposal

This document proposes a GA Roadmap based on multiple phases, where different parts of the
collector will be released as stable at different moments of time.

At this moment we are completely defining only the first two phases of the process, and the
next phases will be defined at a later stage once the Collector maintainers will have
better understanding of the process and implications.

The primary focus is on the tracing parts. When other signal's data models (proto definition)
will be marked as stable, the amount of work necessary to stabilize their APIs will be minimal:
`pdata` is auto-generated so all changes that we do for trace will apply to all of them,
`consumer` is minimal interface, `component` as well.

Metrics components such as (`receiver/prometheus`, `exporter/prometheusremotewrite`) are
explicitly left out of this roadmap document because metrics data model is not complete.
When that work finishes, we can add them to the Phase 3, or later.

### Phase 1

**Tentative Date:** 2021-03-31

**Key Results:** At the end of this phase the Collector’s core API will be marked as Stable.

At the end of this phase we want to achieve core APIs stability. This will allow developers
to implement custom components and extend the collector will be marked as stable.
The complete list of the packages/modules will be finalized during the first action item of
this phase, but the tentative list is:

* `consumer`
  * Official internal data model `pdata`.
  * Interfaces and utils to build a Consumer for (trace, metrics, logs).
* `config`
  * Core `config` including service definition, component definition will be stabilized.
  * To be determined which config helpers will be marked as stable (e.g. configgrpc, etc.).
* `component`
  * Interfaces and utils to build a Collector component (receiver, processor, exporter, extension).
* `obsreport`
  * Focus on the public API of this package. It is out of scope to ensure stability for the
  metrics emitted (focus in phase 2).
* `service`
  * Public API to construct a OpenTelemetry Collector Service.

**Action Items:**

* Create a new milestone for this phase, create issues for all the other action items and add
them to the milestone.
* Agreement on all packages/modules that will be marked as stable during this phase.
* Write a version doc as per [version and stability document](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/versioning-and-stability.md).
  * Previously it was discussed that for the Collector it is fine to release stable golang modules
  that contain APIs marked as experimental.
  * Define status schema (experimental/stable), what are they applicable to every module.
* Investigate if splitting into smaller, more granular, modules is possible.
  * Define the modules schema, try to not break everyone.
  See [here](https://github.com/golang/go/wiki/Modules#is-it-possible-to-add-a-module-to-a-multi-module-repository).
  * Investigate how can we release multiple golang modules from the same repo, without asking
  people that consume them to use a replace statement. See problem in contrib.
  * Investigate how to release test-only utils.
* Review all public APIs and godocs for modules that we want to release in this phase.
  * Fix all critical issues, and remove unnecessary (“When in doubt leave it out”) public APIs.
  * Remove all already deprecated code from the stable modules.
* Transition to opentelemetry-go trace library from opencensus?
* Investigate if any config helper needs to be released as stable, if any do the review of
the public API, godoc and configuration.
* Investigate tools that check for API compatibility for go modules, enable them for modules
that we mark as stable.

### Phase 2

**Tentative Date:** 2021-04-31

**Key Results:** At the end of this phase the Collector’s end-to-end support for OTLP traces
only be marked as GA.

At the end of this phase we want to ensure that the Collector can be run in production, it can receive
OTLP trace traffic and emit OTLP trace traffic. The complete list of the packages/modules will be
finalized during the first part of this phase, but the tentative list is:

* `receiver`
  * `receiverhelper` - without scraper utils in this phase.
  * `otlp`
* `processor`
  * `processorhelper`
  * `batch`
  * `memory_limiter`
* `exporter`
  * `exporterhelper`
  * `otlp`
  * `otlphttp`
* `extension`
  * `extensionhelper`
  * `healthcheck`
* `obsreport`
  * Stabilize the observability metrics (user public metrics).

**Action Items:**

* Create a new milestone for this phase, create issues for all the other action items and add them
to the milestone.
* Agreement on all packages/modules that will be marked as stable during this phase.
* Review all public APIs and godocs for modules that we want to release in this phase.
  * Fix all critical issues, and remove unnecessary (“When in doubt leave it out”) public APIs.
  * Remove all already deprecated code from the stable modules.
* Review all public configuration for all the modules, fix issues.
* Setup a proper loadtest environment and continuously publish results.
* Ensure correctness tests produce the expected results, improve until confident that a binary
that passes them is good to be shipped.
* Enable security checks on every PR (currently some are ignored like `codeql`).

### Phase 3

Tentative Date: 2021-05-31
**Key Results:** At the end of this phase all Collector’s core components (receivers,
processors, exporters, extensions) for traces only will be marked as GA.

At the end of this phase we want to ensure that the Collector can be run in production, it can receive the
trace traffic and emit OTLP trace traffic. The complete list of the packages/modules will be finalized
during the first part of this phase, but the tentative list is:

* `receiver`
  * `jaeger`
  * `opencensus`
  * `zipkin`
* `processor`
  * `spantransformer` - there are good reasons to merge `attributes` and `span`.
  * `resource`
  * `filter` - we will consider offering a filter processor for all telemetry signals not just for metrics
* `exporter`
  * `jaeger`
  * `opencensus`
  * `zipkin`
* `extension`
  * `pprof`
  * `zpages`

TODO: Add action items list.

### Phase N

TODO: Add more phases if/when necessary.

## Alternatives

One alternative proposal is to try to GA all packages at the same time. This proposal was rejected
because of the complexity and size of the ecosystem that may force the GA process to take too much time.
