# Phase 3: Go-Side Extension Component Lifecycle

## Overview

Phase 3 implements the Go mechanics for FFI-component lifecycle,
allowing component factories to be loaded, used to parse and validate
configuration, used to build the pipeline graph, used to start, and
lastly shutdown a pipeline.

This design is based on the [rust2go FFI
library](https://github.com/ihciah/rust2go) which integrates Rust and
Go calling conventions and memory-sharing operations efficiently.

The Golang Collector supports a number of component varieties
(Exporter, ...), however we will start with the simplest which is
Extension, because it requires nothing more than lifecycle support.

## Design Goals

1. **Basic Lifecycle**: Distinct Validate, Create, Start phases matching Collector
2. **Go Extension Factory**: Complete implementation of `extension.Factory` interface
3. **rust2go Integration**: Use rust2go for high-performance component lifecycle FFI calls

## Component Factory Architecture

### Extension interface

Phase 3 implements the complete `extension.Factory` interface and
corresponding `NewFactory` function, which requires:

- `Type`: the name of the factory
- `CreateDefaultConfig`: from phase 2
- `Create`: return an `extension.Extension` instance
- `Stability`: report stability information
- `Start`: start the component
- `Shutdown`: shutdown the component.

These functions will be implemented using FFI to Rust-based components.

### Rust factory registration

Rust components use are explicitly registered via a main compilation
unit, which until later phases will be a single static-library with
dependencies on each component. This "monolithic" build approach will
stay until phase 7, at which point we develop the option to build
these as plugins and load them at runtime.

```rust
mod extension {

pub trait NewFactory {
  type Factory;

  // new_factory contsructs a new factory for extension component
  fn new_factory() -> impl Self::Factory;
}

}
```

The builder (from phase 1) will create an explicit call for each
factory in the build (e.g., `OTLPExporter::new_factory()`), similar to
the corresponding generated code in `components.go`.

## Component lifecycle

Component lifecycle is separated into three steps:

1. Configuration: the Collector's whole configuration is parsed and
   validated; the component-oriented sections `exporters`,
   `receivers`, `processors`, `exporters`, and `extensions` resolve
   named-type components into factories using a table
   lookup. Component-level configuration is validated by the
   corresponding factory.
2. Creation: the Collector creates components to form a graph. By this
   time, individual component instances have been resolved. For the
   present stage, we will create only extension components.
3. Start/Shutdown: the Collector starts components in topological
   order so that consumers are started before producers for each edge
   of the graph. Components are later shutdown individually as the
   Collector manages its own lifecycle.

## Core Architecture

### Go Extension Factory Implementation

The Go extension factory is responsible for:

- Implementing the `extension.Factory` interface
- Creating Go wrappers for each of the resulting interfaces (i.e., `component.Component`, `extension.Extension`).
- Use FFI to Rust for each function implementation (e.g., `Type`, `CreateDefaultConfig`, `Stability`)
- Maintain component ID, logger, validated config, and runtime state (e.g., Rust handle, started flag).

### Cross-runtime implementation

The Go extension instance implementation manages the lifecycle of a
Rust-backed extension component:

- **Configuration Phase**: The JSON configuration used for validation is
  saved for the subsequent call to `Create` to avoid managing
  configuration-object lifetime.
- **Create Phase**: The Go and Rust side will correspond using a
  "handle" type, an implementation detail. This handle will be retained
  from Create to Start and Shutdown.
- **Run Phase**: The handle may be used for additional interfaces,
  such as liveness checking and status reporting across the runtimes.

We use placeholders at this time for the `Host` and `Context` abstractions.

- `Context` provides metadata and deadline support, see [phase 5](./plugin-phase5.md).
- `Host` provides a means for components to access other components in the configuration, see [phase 6](./plugin-phase6.md).

## Decision to use rust2go

The [`rust2go` library](https://github.com/ihciah/rust2go) is a proven
solution for high-performance synchronous and asynchronous calling
between Go and Rust runtimes. The design of this library allows
sharing of pointers to raw data structures, such as byte arrays, which
permits us to avoid additional copies of data as it transits beteween
runtimes.

The basic approach used by this library involves making multiple,
bidirectional FFI calls to facilitate safe sharing of pointers. In our
case, we expect the following data paths:

- **Configuration**: JSON data will be marshaled in Go to `[]byte`,
  the resulting data will be shared with Rust as `[]u8`, where it can
  be parsed without copying the underlying bytes out of the original
  Go buffer. Likewise, `rust2go` arranges for return values to be
  placed directly in the caller's memory.
- **Pipeline data**: OTLP bytes will be serialized (to `[]byte`) using
  the built-in `pdata` support, from a Collector-internal fork of the
  [`Gogo` protocol-buffer runtime
  library](https://github.com/gogo/protobuf).  This data is parsed
  using [a "view" interface under development in the OTel-Arrow
  `otap-dataflow`
  crate](https://github.com/open-telemetry/otel-arrow/tree/main/rust/otap-dataflow/crates/pdata-views)
  that directly translates OTLP (Protobuf) bytes into OTAP (Arrow)
  record batches, again avoiding a copy of the underlying bytes.

## Success Criteria

- [ ] Extension factory registers successfully with collector service
- [ ] Clear separation between lifecycle phases
- [ ] Configuration validation (from phase 2)
- [ ] Start/Shutdown lifecycle works uses rust2go async calls
- [ ] Error messages propagate properly from Rust to Go collector logs
- [ ] No memory leaks in FFI boundary during normal and error scenarios
- [ ] Integration tests pass with real collector service startup/shutdown
- [ ] Handle management works correctly for multiple extension instances
- [ ] Component ordering and dependency management works with collector service
- [ ] rust2go performance benefits realized compared to traditional CGO approaches
