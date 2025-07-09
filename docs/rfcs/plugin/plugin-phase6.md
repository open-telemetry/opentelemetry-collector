# Phase 6: Out-of-Process Fallback with Service Graph Partitioning

## Overview

Phase 6 provides a fallback architecture when direct Go and Rust
runtime integration (Phases 1-5) cannot be used. Instead of in-process
FFI, this phase implements service graph partitioning where the
collector spawns a subordinate Rust process and communicates via
standard OTLP/gRPC over secure socket pairs.

We extend the builder configuration with a new `build_mode` option
with values "auto", "embedded", and "subprocess".

```yaml
dist:
  name: otelcol-custom
  description: Custom collector with mixed Go/Rust components
  build_mode: auto  # auto | embedded | subprocess

receivers:
  ...

processors:
  ...

exporters:
  ...
```

The builder automatically determines the appropriate mode based on the `subprocess_mode` setting:

- `embedded`: Forces in-process FFI mode (requires CGO_ENABLED=1)
- `subprocess`: Forces out-of-process mode via OTLP (supports CGO_ENABLED=0)
- `auto`: Automatically detects the best mode based on build constraints and CGO_ENABLED.

## Detailed Design

### Context/Error Propagation via OTLP

The Collector's built-in OTLP exporter and receiver propagate the same
information as we propagate from component-to-component in Collector
pipelines. Whether HTTP or gRPC, OTLP components transfer request
context including timeout/deadline/cancellation and other metadata in
the forward direction, and they preserve the required error metadata
in the backward direction:

- Deadline
- Cancellation
- Metadata
- Tracing
- Error classification

We assume both Go and Rust provide an OTLP exporter and receiver
component we can use at this step.

### Configuration Example

The user will provide a configuration mixing Go and Rust components,
for example:

```yaml
receivers:
  prometheus:
    scrape_configs:
      - job_name: 'collector'
        static_configs: [targets: ['localhost:8888']]

processors:
  advanced_filter:
    drop_metrics: ["up", "scrape_*"]

exporters:
  otap:
    endpoint: https://backend.example.com:4318

service:
  pipelines:
    metrics:
      receivers: [prometheus]         # Go receiver
      processors: [advanced_filter]   # Rust processor
      exporters: [otap]               # Rust OTAP exporter
```

When the build_mode is "subprocess", the above will translate into two
configurations, first the parent's effective configuration:

```yaml
# Auto-generated: Go parent Collector effectively runs this configuration
receivers:
  prometheus:
    scrape_configs:
      - job_name: 'collector'
        static_configs: [targets: ['localhost:8888']]

exporters:
  otlp/bridge:  # Bridge exporter, an OTLP exporter
    endpoint: unix:///tmp/collector-bridge.sock
    tls:
      insecure: true

service:
  pipelines:
    metrics:
      receivers: [prometheus]
      exporters: [otlp/bridge]       # To Rust
```

In this example, the child Rust subprocess will start with this
configuration:

```yaml
# Auto-generated: Rust collector child starts with this configuration
receivers:
  otlp/bridge:  # Bridge receiver, an OTLP receiver
    protocols:
      grpc:
        endpoint: unix:///tmp/collector-bridge.sock

processors:
  advanced_filter:
    drop_metrics: ["up", "scrape_*"]

exporters:
  otap:
    endpoint: https://backend.example.com:4318

service:
  pipelines:
    metrics:
      receivers: [otlp/bridge]       # From Go
      processors: [advanced_filter]
      exporters: [otap]
```

## Service Graph Partitioning

The service graph partitioner automatically analyzes user
configuration and splits it into separate Go and Rust sub-graphs.

### Partitioning Algorithm

The partitioner performs three operations to automatically split mixed
Go/Rust configurations:

1. **Component Type Analysis**: Uses Phase 1's component
   identification where `cargo` fields indicate Rust components, while
   `gomod` fields or standard collector components are Go components.
2. **Pipeline Dependency Analysis**: Examines `service.pipelines`
   configuration to build a dependency graph showing data flow between
   components and identifies boundary points where data crosses from
   Go to Rust components.
3. **Bridge Insertion**: Inserts OTLP bridge components at boundary
   points, using a single `otlp/bridge` exporter in the Go
   configuration and corresponding `otlp/bridge` receiver in the Rust
   configuration, connected via secure unix sockets.

## Process Lifecycle Management

The parent Go collector manages the complete lifecycle of the Rust
subprocess using standard process management patterns. The Go parent
creates a private socket pair, spawns the child Rust process, and
establishes background health monitoring of the child process.

The Go parent will be able to restart the child Rust process in case
it fails a health check. We expect to coordinate graceful shutdown
using SIGTERM (e.g., the parent will pass this signal to the child),
with timeout-fallback to SIGKILL.

Health monitoring, logging, and status reporting work seamlessly
across process boundaries. The parent collector configures the
subprocess observability SDK to match its own configuration, however
there will be two distinct SDKs running when both Go and Rust runtime
environments are mixed, hence we expect two OpenTelemetry resources
per combined runtime.

## Success Criteria

- [ ] Builder `cargo` specifications work identically in all build modes
- [ ] Service graph partitioning correctly identifies runtime boundaries
- [ ] Context/error propagation works automatically via OTLP
- [ ] CGO_ENABLED=0 forces subprocess mode
- [ ] Runtime configuration passed to subprocess
- [ ] Standard collector observability signals.
