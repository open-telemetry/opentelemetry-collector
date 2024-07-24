# Defining guidelines for internal telemetry

## Overview

The Collector supports generating internal telemetry
that assists end users when operating the Collector. So
far, much of the telemetry has been added to components as
needed, without guidelines for component authors to provide
a consistent experience to end users. The goal of this document
is to:

- describe the naming and attributes currently in use
- define consistent units + naming for telemetry emitted by the Collector
- define the process that should be used to configure new metrics

## Out of scope

This document is not intending to dictate when telemetry should be
emitted by various Collector components. Considering the various types
of components, this will be better discussed in a future document.

This document is not intending to provide a comprehensive plan for how
the Collector or the health of telemetry pipelines should be monitored. There
is an [OpenTelemetry Enhancement Proposal](https://github.com/open-telemetry/oteps/pull/259) that has already started the process
to provide this information. The information in this document will be updated once
the OTEP lands to follow its recommendations.

## Internal telemetry properties

Telemetry produced by the Collector have the following properties:

- metrics produced by Collector components use the prefix `otelcol_`. Until v0.106.0, this
  was done via the Prometheus namespace, which made metrics inconsistent for users wishing
  to emit metrics via another exporter. See https://github.com/open-telemetry/opentelemetry-collector/issues/9315
  for more details
- code is instrumented using the OpenTelemetry API and telemetry is produced via the OpenTelemetry Go SDK
- instrumentation scope is defined via configuration in `metadata.yaml`
- metrics are defined via `metadata.yaml` except in components that have specific cases where
  it is not possible to do so. See [contrib issue #33523](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/33523)
  which list such components

## Units

The following units should be used for metrics emitted by the Collector
for the purpose of its internal telemetry:

| Field type                                                       | Unit           |
| ---------------------------------------------------------------- | -------------- |
| Metric counting the number of log records received, processed, or exported        | `{records}`    |
| Metric about receiving, processing, exporting spans              | `{spans}`      |
| Metric about receiving, processing, exporting metric data points | `{datapoints}` |

## Process for defining new metrics

Metrics in the Collector are defined via `metadata.yaml`, which is used by [mdatagen](https://github.com/open-telemetry/opentelemetry-collector/tree/main/cmd/mdatagen) to
produce:

- code to create metric instruments that can be used by components
- documentation for internal metrics
- a consistent prefix for all internal metrics
- convenience accessors for meter and tracer
- a consistent instrumentation scope for components
- test methods for validating the telemetry

The process to generate new metrics is to configure them via
`metadata.yaml`, and run `go generate` on the component.
