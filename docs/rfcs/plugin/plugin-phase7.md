# Phase 7: Plugin Architecture for Go and Rust Components

## Overview

We will add dynamic plugin support for Go and Rust components in the
OpenTelemetry Collector. This gives Collector users the ability to run
components compiled separately from the main Collector binary.

## Design Goals

1. **Dynamic Extensibility**: Load Go and Rust components as plugins at runtime
2. **Hermetic Builds**: Guarantee plugins and main binary use identical dependencies, compiler versions, and build flags
3. **Version Safety**: Prevent mixmatched runtimes, dependencies
4. **Distribution Verification**: Verify plugin compatibility before loading using manifests.

## Go Plugin System

- Uses Go's [`plugin` package](https://pkg.go.dev/plugin) to compile shared libraries
- Plugins must use identical Go version, module versions, and build flags as the main binary
- The builder will:
  - Generate manifests describing Go version, module hashes, and build flags using embedded JSON
  - Embed manifests in both main binary and plugins
  - Verify manifests at runtime before loading plugins
- Plugins export one of the component `NewFactory` methods.

We recognize caveats with this approach:

- Go plugins only supported on Linux and macOS, not Windows
- All dependencies must be statically linked and version-matched.

Plugin components cannot be dynamically unloaded. The Collector will
require a restart to reconfigure plugins under this design.

## Rust Plugin System

- Uses Cargo to build shared libraries as `cdylib` crates, loaded via dynamic linking
- Plugins must use identical Rust version, crate versions, and Cargo settings as the main binary
- The builder will:
  - Generate manifests with Rust version, crate hashes, and build flags using embedded JSON
  - Embed manifests in both main binary and plugins as sections or exported symbols
  - Verify manifests at runtime before loading plugins
- Plugins export one of the C-API component `NewFactory` methods following `rust2go` integration

We recognize caveats with this approach:

- Rust has no stable ABI; all FFI boundaries must use C-compatible types
- All dependencies must be statically linked and version-matched.

## Hermetic Build & Distribution

The builder supports "hermetic" build mode:

- Uses Docker to encapsulate combined Go and Rust build environment (compiler versions, OS, etc.)
- Dockerfile is generated or versioned alongside the distribution
- All plugins and main binary built in the same container for identical environments
- Build process outputs:
  - Main Collector binary
  - Plugin shared libraries
  - Version manifests for each artifact
  - Dockerfile or build environment hash

At runtime, the Collector will:

- Refuse to load plugins with mismatched manifests
- Provide CLI commands to verify plugin compatibility and inspect metadata

## Unified Plugin Manifest Schema

Both Go and Rust plugins will embed a manifest with the following structure (JSON):

```json
{
  "language": "go", // or "rust"
  "toolchain_version": "1.22.3", // go version or rustc version
  "build_profile": "release",
  "dependency_hashes": {
    "otel-arrow": "b1c2d3...",
    "go.opentelemetry.io/collector": "a9b8c7..."
  },
  "build_flags": ["-buildmode=plugin"],
  "target": "x86_64-unknown-linux-gnu",
  "build_time": "2025-07-03T12:34:56Z"
}
```

- For Go: toolchain_version contains Go version output; dependency_hashes contains module hashes
- For Rust: toolchain_version contains Rust compiler version output; dependency_hashes contains crate hashes
- Manifest exported as named symbol for Rust plugins, embedded in Go binaries/plugins for extraction

## Command-line Support

The Collector CLI will provide new commands for plugin inspection and validation:

- **List plugins**: Display all plugins in configured directory with manifest info (language, version, hashes)
- **Validate plugin**: Load specific plugin, extract manifest, check compatibility with main binary
- **Inspect plugin**: Print detailed manifest and exported symbols without loading into main process
- **Check all plugins**: Validate all plugins in directory, report mismatches or issues

These commands enable safe, transparent plugin management for automation.

## Success criteria

- [ ] Builder can build shared-library forms of Go and Rust components
- [ ] New component manifest structure records metadata from `go.sum` and `Cargo.lock`
- [ ] Plugins can be dynamically identified at runtime
- [ ] Loader validates manifest compatibilitiy before loading shared libraries
- [ ] Shared-library performance overhead near zero compared with static component build.
