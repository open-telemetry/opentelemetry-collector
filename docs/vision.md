# OpenTelemetry Collector Long-term Vision

The following are high-level items that define our long-term vision for OpenTelemetry Collector, what we aspire to achieve. This vision is our daily guidance when we design new features and make changes to the Collector.

This is a living document that is expected to evolve over time.

## Performant
Highly stable and performant under varying loads. Well-behaved under extreme load, with predictable, low resource consumption.

## Observable
Expose own operational metrics in a clear way. Be an exemplar of observable service. Allow configuring the level of observability (more or less metrics, traces, logs, etc reported). See [more details](https://opentelemetry.io/docs/collector/internal-telemetry/).

## Multi-Data
Support traces, metrics, logs and other relevant data types.

## Usable Out of the Box
Reasonable default configuration, supports popular protocols, runs and collects out of the box.

## Extensible
Extensible and customizable without touching the core code. Can create custom agents based on the core and extend with own components. Welcoming 3rd party contribution policy.

## Unified Codebase
One codebase for daemon (Agent) and standalone service (Collector).
