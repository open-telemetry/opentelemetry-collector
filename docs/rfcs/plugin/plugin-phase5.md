# Phase 5: Cross-Runtime Metadata Propagation

## Overview

Phase 5 establishes bidirectional metadata propagation between Go and
Rust runtimes, enabling context flow and error propagation across the
FFI boundary.

## Design Goals

1. **Forward Context Propagation**: Propagate Go `context.Context` with timeouts, cancellation, and metadata to `otap-dataflow` EffectHandler
2. **Backward Error Propagation**: Translate Rust enum-structured errors to Go error interface with proper permanent/retryable classification
3. **Standards Compliance**: Follow gRPC and HTTP metadata conventions established in the collector
4. **Performance**: Minimize data copying as much as possible
5. **Observability**: Enable metrics, traces, and logs for debugging across runtime boundaries.

### Forward Propagation: Context to/from EffectHandler

On the Go side, the [`context.Context`
type](https://pkg.go.dev/context#Context) type is passed in parallel
with every PData item, carrying metadata alongside the request, including:

- Deadline/Timeout information
- Cancellation signal
- Request metadata (`client.Metadata`)
- Tracing metadata (`trace.SpanContext`)

On the Rust side, the idiom is different but the capabilities are the
same. The `otap-dataflow` engine provides an abstract interface for
asynchronous interaction with the system called
`EffectHandler<PData>`, which provides platform- and system-specific
optimizations. The PData type, concretely, is a contextual wrapper
with an `inner` batch of pipeline data, for example
`EffectHandler<Context<TracesOTAP>>` describes a component that
interacts with OTAP traces data through a Context providing access
to asynchronous events and metadata.

We expect to use custom structs to convey this information into both
Go and Rust runtimes.

### Backward Propagation: Error propagation

Rust and Go have different error-handling idioms.  We will bridge
between them.

On the Go side, the builtin `error` interface is extended via the
`errors.As` mechanism to convey OpenTelemetry-specific error
conditions, including:

- consumererror.Permanent status
- gRPC status codes
- HTTP status codes

We expect to use custom structs to convey this information out of both
Go and Rust runtimes.

## Detailed Design

### Context Transfer Strategy

We will use custom structs defined through `rust2go` to convey context
information in the forward direction and error information in the
backward direction.

### Go implementation

The context transfer process extracts key information from Go's
`context.Context` to pass forward. When a goroutine calls Rust FFI
with a deadline in effect, it will:

```go
  // Send a request by submitting it to a channel,
  // always consider the Done signal.
  replyChan := make(chan error, 1)
  select {
  case comp.r2gChannel <- callFFI{ctx, data, replyChan}:
    // the receiver will extract metadata from callFFI.ctx
    return
  case <- ctx.Done():
    return ctx.Cause()
  }

  // Wait for timeout or reply,
  select {
  case err := <-replyChan:
    return err
  case <- ctx.Done():
    return ctx.Cause()
  }
```

### Rust implementation

The Rust side will convey this information through the
`EffectHandler`. After receiving a Context struct via `rust2go`, the
asynchronous implementation of Context is detail for the
`otap-dataflow` crate.

We expect to propagate all the same information as covered above, in
both directions (timeout/deadline, asynchronous cancellation,
client.Metadata, trace.SpanContext).

## Success Criteria

- [ ] Go context.Context with deadlines propagates to Rust EffectHandler operations
- [ ] Go context cancellation immediately cancels ongoing Rust async operations
- [ ] Rust ProcessingError enum values correctly map to Go consumererror classification
- [ ] gRPC and HTTP error codes are preserved across runtime boundaries
- [ ] Client metadata follows collector conventions and is accessible in Rust
- [ ] Distributed tracing context flows seamlessly across Go/Rust boundary
- [ ] All timeout and cancellation scenarios work correctly under load
