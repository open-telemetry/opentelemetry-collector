# Phase 2: Runtime Configuration Design for Rust Components

## Overview

Phase 2 addresses the challenge of enabling Rust components to define
and validate arbitrary configuration structs, similar to how Go
components use mapstructure tags and custom validation. This phase
builds on Phase 1's builder configuration and establishes how
Rust components will be configured.

## Design Goals

1. **Idiomatic Rust configuration**: Use standard [serde](https://serde.rs/) patterns.
2. **Confmap support**: Use JSON to exchange data over FFI.
3. **Validation support**: Idiomatic validation logic.
4. **Performance cost**: Configuration parsing is startup-time only, JSON overhead is acceptable.

## Core Architecture

### Rust Side: Standard Serde Patterns

Rust components use features similar to those found in the Go [`mapstructure` library](https://pkg.go.dev/github.com/go-viper/mapstructure/v2).

- Field renaming
- Conditional serialization
- Embedded struct flattening (like mapstructure `squash`)
- Skipping fields
- Collecting unknown fields
- Field defaults

This approach ensures that configuration is both expressive and
familiar to Rust developers.

### Go Side: Custom confmap Integration

On the Go side, a wrapper struct integrates Rust validation with the
confmap system. The process involves:

- Unmarshaling the config from confmap into a map
- Marshaling the map to JSON for Rust validation via FFI
- Handling errors returned from Rust and reporting them through the standard Go error pipeline
- Storing the validated JSON for runtime use.

### FFI Interface: JSON + String Errors

The Go and Rust sides communicate via simple C-compatible FFI
functions based on the [`rust2go`
library](https://github.com/ihciah/rust2go). These functions exchange
configuration data as JSON strings and return error messages as text
strings.

### Custom Types

For edge cases, such as Go-Rust type compatibility (e.g., durations),
custom serde serializers can be used to ensure that types are
represented in a way that both languages understand. This typically
involves serializing durations as strings and providing custom parsing
logic.

## Configuration Example

```rust
#[derive(Debug, Deserialize, Serialize)]
pub struct HTTPExporterConfig {
    pub endpoint: String,

    #[serde(default)]
    pub headers: HashMap<String, String>,

    #[serde(flatten)]
    pub tls_config: TLSConfig,

    #[serde(flatten)]
    pub retry_config: RetryConfig,

    #[serde(skip_serializing_if = "Option::is_none")]
    pub proxy_url: Option<String>,
}
```

## Success Criteria

- [ ] Rust components can define arbitrary configuration structs using standard serde
- [ ] Configuration validation integrates seamlessly with confmap error reporting
- [ ] Complex configuration scenarios work equivalent to Go mapstructure patterns
- [ ] Error messages are clear and actionable for users
- [ ] No performance regression in configuration parsing
- [ ] Memory management across FFI boundary is leak-free
