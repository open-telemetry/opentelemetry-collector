# OTel-Arrow Rust and Go Interoperability Plan

## Towards Plugins for the OpenTelemetry Collector

This document outlines a number of steps to develop the ability to

- Mix Golang and Rust components in the OpenTelemetry Collector
- Build Collector components as static libraries, shared libraries, or subprocess-based
- Support dynamic loading of separately compiled "plugin" components.

### 1. Builder Integration ([plugin-phase1.md](plugin-phase1.md))

Enable the Collector `builder` to support both Go and Rust
components in a unified configuration.

- Extends the builder YAML and `Module` struct to allow Rust (`cargo`)
  and Go (`gomod`) modules side-by-side.
- Ensures backward compatibility and minimal changes for Go-only users.

### 2. Config Integration ([plugin-phase2.md](plugin-phase2.md))

Provide a seamless configuration experience for Rust
components, matching Go's `mapstructure` with Rust's `serde`.

- Rust components define config structs using idiomatic serde patterns.
- Go and Rust config validation and defaults are bridged via FFI and JSON.

### 3. Rust Async Lifecycle ([plugin-phase3.md](plugin-phase3.md))

Implement the Go-side lifecycle for Rust extension
components using FFI (rust2go).

- Separates component creation (validation) from activation (runtime start).
- Demonstrates the full `extension.Factory` pattern for Rust components.

### 4. Pipeline Components ([plugin-phase4.md](plugin-phase4.md))

Integrate Rust pipeline components (processors, exporters,
receivers) with the Collector's dataflow.

- Bridges Go's consumer interfaces with Rust's `otap-dataflow` engine.
- Handles efficient data translation and registration for all pipeline stages.

### 5. Contextual Bridge ([plugin-phase5.md](plugin-phase5.md))

Enable robust context and error propagation between Go and Rust runtimes.

- Propagates deadlines, cancellation, and metadata from Go to Rust.
- Maps Rust error enums to Go error types, supporting observability and retry semantics.

### 6. Fallback Option ([plugin-phase6.md](plugin-phase6.md))

Provide an out-of-process fallback when in-process FFI is not possible.

- Partitions the service graph and spawns a Rust subprocess, communicating via OTLP/gRPC.
- Ensures all context and error propagation works via standard OTLP conventions.

### 7. Plugin Design ([plugin-phase7.md](plugin-phase7.md))

Support dynamic, hermetic plugin loading for both Go and Rust components.

- Defines manifest schemas, build requirements, and runtime validation for plugins.
- Enables safe, versioned, and extensible plugin management with CLI tooling.
