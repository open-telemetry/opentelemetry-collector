# OpenTelemetry Collector

**Status**: [Beta](https://github.com/open-telemetry/opentelemetry-specification/blob/main/oteps/0232-maturity-of-otel.md#beta)

The OpenTelemetry Collector consists of the following components:

* A mechanism that _MUST_ be able to load and parse an [OpenTelemetry Collector configuration
  file](#configuration-file).
* A mechanism that _MUST_ be able to include compatible
  [Collector components](#opentelemetry-collector-components) that
  the user wishes to include.

These combined provide users the ability to easily switch between
[OpenTelemetry Collector Distributions](#opentelemetry-collector-distribution) while also ensuring that components produced by
the OpenTelemetry Collector SIG are able to work with any vendor who claims
support for an OpenTelemetry Collector.

## Configuration file

An OpenTelemetry Collector configuration file is defined as YAML and _MUST_ support
the following [minimum structure](https://pkg.go.dev/go.opentelemetry.io/collector/otelcol#Config):

```yaml
receivers:
processors:
exporters:
connectors:
extensions:
service:
  telemetry:
  pipelines:
```

## OpenTelemetry Collector components

For a library to be considered an OpenTelemetry Collector component, it _MUST_
implement a [Component interface](https://pkg.go.dev/go.opentelemetry.io/collector/component#Component)
defined by the OpenTelemetry Collector SIG.

Components require a [unique identifier](https://pkg.go.dev/go.opentelemetry.io/collector/component#ID)
to be included in an OpenTelemetry Collector. In the event of a name collision,
the components resulting in the collision cannot be used simultaneously in a single OpenTelemetry
Collector. In order to resolve this, the clashing components must use different identifiers.

### Compatibility requirements

A component is defined as compatible with an OpenTelemetry Collector when its dependencies are
source- and version-compatible with the Component interfaces of that Collector.

For example, a Collector derived from version tag v0.100.0 of the [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector) _MUST_ support all components that
are version-compatible with the Golang Component API defined in the `github.com/open-telemetry/opentelemetry-collector/component` module found in that repository for that version tag.

## OpenTelemetry Collector Distribution

An OpenTelemetry Collector Distribution (Distro) is a compiled instance
of an OpenTelemetry Collector with a specific set of components and features. A
Distribution author _MAY_ choose to produce a distribution by utilizing tools
and/or documentation supported by the OpenTelemetry project. Alternatively, a
Distribution author _MUST_ provide end users with the capability for adding
their own components to the Distribution's components. Note that the resulting
binary from updating a Distribution to include new components
is a different Distribution.
