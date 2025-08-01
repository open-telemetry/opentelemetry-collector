# Phase 1: Mixed Go/Rust Component Support in Builder

## Overview

Phase 1 establishes the foundation for supporting both Go and Rust
components within the OpenTelemetry Collector builder tool. This phase
focuses on extending the existing YAML configuration format and Module
struct to accommodate Rust components while maintaining full backward
compatibility with existing Go-only configurations.

## Design Goals

1. **Minimal Configuration Changes**: Reuse existing builder YAML structure as much as possible
2. **Consistency**: Follow the same patterns for Rust as established for Go components

## Core Design Decisions

### Unified Module Approach

Rather than creating separate configuration sections for Rust
components, we extend the existing `Module` struct to support both
languages within the same component lists (receivers, exporters,
processors, extensions, connectors).

### Literal Text Insertion Pattern

Following the existing `gomod` field pattern where content is inserted
literally into go.mod files, the new `cargo` field inserts content
literally into Cargo.toml dependency sections.

## Configuration

### Enhanced Module Struct

```go
// Module represents a receiver, exporter, processor or extension for the distribution
type Module struct {
    Name   string `mapstructure:"name"`   // if not specified, this is package part of the go mod (last part of the path)
    Import string `mapstructure:"import"` // if not specified, this is the path part of the go mods
    GoMod  string `mapstructure:"gomod"`  // text for the go.mod file
    Path   string `mapstructure:"path"`   // an optional path to the local version of this module
    Cargo  string `mapstructure:"cargo"`  // text for the Cargo.toml file
}
```

### YAML Configuration Examples

**Rust component (cargo syntax):**

```yaml
exporters:
   cargo: otel-otlp-exporter = "0.129.0"
```

**Go components unchanged (go.mod syntax):**

```yaml
extensions:
  - gomod: go.opentelemetry.io/collector/extension/zpagesextension v0.129.0
```

### Validation Updates

Current validation required the `gomod` field to be non-empty. In Phase 1, validation is updated to:

- Accept modules with either a `gomod` or `cargo` field (but not both)
- Produce clear error messages if neither or both are specified
- Ensure each component specifies exactly one language

This ensures configuration clarity and prevents ambiguous component definitions.

### Name Collisions

It is permitted for components to have the same name in both Go and
Rust runtimes, provided they are functionally identical and one uses
the other's configuration struct (for example, `otlp` and `otap`
receivers/exporters maintained in parallel, in both languages).

### YAML String Handling

YAML provides multiple quoting options.

For simple dependencies (no special characters):

```yaml
    cargo: myexporter = "1.0.1"
```

For complex dependencies (using YAML literal strings):

```yaml
    cargo: |
      myexporter = { version = "1.0", features = ["derive"] }
```

For complex dependencies (using quoted strings):

```yaml
    cargo: 'myexporter = { version = "1.0", features = ["derive"] }'
```

## Component Factory Architecture

### Factory Registration Strategy

In Go, component factories are registered through inclusion in the
main package's `components.go` file, generated from builder
templates. Each factory implements the relevant interface
(`component.Factory` base with specialized `extension.Factory`,
`processor.Factory`, etc.) with two essential methods:

- **`Type()`**: Returns the component type name (e.g., "otlp",
  "debug", "prometheus")
- **`CreateDefaultConfig()`**: Returns the default configuration
  struct, implicitly defining the config type and its parsing metadata
  (mapstructure tags)

In Rust, a compilation module will be synthesized by the builder, as a
subdirectory of the main Go package, and follow the same explicit
registration pattern as Go. Rust factories will support the Factory
interface beginning in [phase 2](./plugin-phase2.md).

## Success Criteria

- [ ] Enhanced `Module` and coresponding YAML with `cargo` field for Rust components
- [ ] Updated validation logic for mutually exclusive language specifications
- [ ] Placeholder for Rust `Factory` traits
- [ ] Backward compatibility preservation for existing Go factory patterns
