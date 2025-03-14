# Middleware configuration

**Status: development**

This module defines necessary interfaces to configure middleware
extensions for use in components. Multiple kinds of middleware may be
defined, however middlewares are configured through a single list, and
middleware components are invoked in order. Specific kinds of
middleware are be configured as specific extension types, and more
than one extension type is supported:

- `Limiter`: a weighted protocol-agnostic form of middleware, meant
  for use in receivers.  To implement a limiter extension, components
  should implement the interface in
  [`extension/extensionlimiter`](#../../extension/extensionlimiter/README.md).
- `Interceptor`: a protocol-specific form of middleware, may be client
  or server, stream or unary. To implement an interceptor extension,
  components should implement the interface in
  [`extension/extensioninterceptor`](#../../extension/extensioninterceptor/README.md).

Middleware is included in the basic HTTP and gRPC Server Config
structs, making it so that exporters and receivers in push-based
protocols automatically through `confighttp` and `configgrpc`.

The currently known extensions are listed below.

## Limiter implementations

- [Memory Limiter Extension](../../extension/memorylimiterextension/README.md)
- [Admission Limiter Extension](../../extension/admissionlimiterextension/README.md)

Example:

```yaml
extensions:
  # Used with gRPC traffic, consults GC statistics.
  memory_limiter/cold
    request_limit_mib: 100
	waiting_limit_mib: 10

  # Used with HTTP traffic, counts request bytes in flight.
  admission_limiter/warm:
    request_limit_mib: 10
    waiting_limit_mib: 10

receivers:
  otlp:
    protocols:
	  http:
	    # ...
	    middleware:
		- limiter: admission_limiter/warm
      grpc:
	    # ...
        middleware:
        - limiter: memory_limiter/cold
```
