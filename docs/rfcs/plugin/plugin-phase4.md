# Phase 4: Data Pipeline Components

## Overview

Phase 4 integrates the OpenTelemetry Collector with the existing
`otel-arrow/rust/otap-dataflow` engine for access to Rust telemetry
pipelines. Here, we continue the adoption of
[`rust2go`](https://github.com/ihciah/rust2go), adding support for
passing OTLP-serialized content between the Go and Rust runtimes.

## Factory Pattern For Consumer Components

We will extend the factory pattern from phase 3 to the data-processing
components (processors, exporters, receivers, connectors). These
factories are similar to the `extension.Factory` interface, but with
one `Create` and `Stability` method per signal kind (e.g.,
`CreateTraces`, `TracesStability`).

The major additions in this phase are the `consumer` interfaces
(e.g,. `consumer.Traces`) which permit passing data from a component
to the next component in the pipeline (with specific signal kinds).

The Go interfaces are:

- `receiver.Factory`: creates components that are only
  producers. These components are supplied with a consumer
  implementation of the appropriate signal kind in the corresponding
  `Create` factory method.
- `processor.Factory`: creates components that are both producers and
  consumers (of the same signal kind) in the pipeline. They implement
  the consumer interface for the preceding component, and they call a
  consumer interface corresponding with the next component in the
  pipeline.
- `exporter.Factory`: creates components that are only
  consumers. These components are passed as consumers to preceding
  component(s) in the pipeline.
- `connector.Factory`: creates components that change signal
  kind. These components act as both exporters and receivers in the
  Collector pipeline-component model.

## Detailed Design

### Bridging Rust `otap-dataflow` Components

We will create a translation layer between the producer and consumer
interfaces of each component runtime using `rust2go` as the underlying
mechanism to pass `[]byte` / `[]u8` values across the runtime boundary
without copying the data.

The [`otap-dataflow`
crate](https://github.com/open-telemetry/otel-arrow/blob/main/rust/otap-dataflow/README.md)
supports the three basic component kinds; Connector-type components
will be modeled as a hybrid case of both its Exporter and Receiver
traits.

### Serialization and Deserialization

The essential work is translating between consumer interfaces. We will
serialize data as it crosses the runtime boundary. On both sides, we
rely on standard mechanisms to translate between the pipeline data
object (a.k.a. "PData object", e.g., `ptrace.Traces` or the Rust
equivalent) and OTLP bytes.

For the Go side, we wil use existing functions to serialize the
`pdata` types as OTLP bytes using the `ProtoMarshaler`,
`ProtoUnmarshaler`, `MarshalSizer` defined by each of the pipeline
data types (e.g., `plog.Logs`).

For the Rust side, we will use [a "view" interface under development
in the OTel-Arrow `otap-dataflow`
crate](https://github.com/open-telemetry/otel-arrow/tree/main/rust/otap-dataflow/crates/pdata-views)
for efficient translation between OTLP and OTAP representations, which
supports traversal through specific OpenTelemetry-structured data.

## Rust Runtime Configuration

The runtime configuration manages the Rust execution environment and
rust2go communication. We will introduce a new `Runtime` struct at the
top-level of the Collector configuration, something like this:

```yaml
runtime:
  rust:
    executor:
      type: "tokio_local"  # Local task executor
    rust2go:
      queue_size: 65536    # Shared memory ring buffer size
```

Specific fields in this configuration struct are outside the scope of
this design, since they are particular to the `otap-dataflow`
engine.

### Consumer implementation

In both Rust and Go, the Collector will require a runtime-aware method
cross the language boundary. For Go calling Rust, the Go component
will receive a consumer stub that internally serializes the data,
calls the Rust consumer, then waits for its response. For Rust calling
Go, the Rust component will serialize the data, make an `async` call
to the Go runtime and follow `otap-dataflow` conventions for handling
the response.

When one component calls another in the same runtime, we must ensure
there is not a cross-runtime method call. This is automatic for the Go
runtime; for Rust, components will require a way to directly locate
their consumers, in case they are in the same runtime.

## Success Criteria

- [ ] Receiver, Exporter, Processor, Connector factories implemented
- [ ] Consumer interface stubs for cross-runtime calls (consumer.Traces, consumer.Logs, consumer.Metrics, ...)
- [ ] Efficient otap-dataflow libraries used for translation between OTLP pdata (Go) to OTAP pdata (Rust)
- [ ] Rust Runtime configuration for Rust execution environment from `otap-dataflow`
- [ ] Rust2go configuration for determining cross-runtime buffer and queue sizes.
