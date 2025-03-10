# Limiter configuration

This module defines necessary interfaces to implement limiter
extensions.  Limiters are included in the basic HTTP and gRPC Server
Config structs, so users will rarely create interact with extensions
these directly.

To imlpement a limiter extension, components should implement the
`GetClient` interface in
[`extension/xextension/limit`](#../../extension/xextension/limit/README.md).

The currently known limiter extensions  are listed below.

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
	    limiter:
		- admission_limiter/warm
      grpc:
	    # ...
        limiter:
        - memory_limiter/cold
```
