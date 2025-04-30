# OpenTelemetry Collector Extension Limiter Package

**Document status: development**

The `extensionlimiter` package provides interfaces for rate limiting
and resource limiting in the OpenTelemetry Collector, enabling control
over data flow and resource usage through extensions which can be
configured through and middleware and/or directly by pipeline
components.

## Overview

This package defines two primary limiter types with their respective
interfaces:

- **Rate Limiters**: Control time-based limits on quantities such as
  bytes or items per second.
- **Resource Limiters**: Manage physical limits on quantities such as
  concurrent requests or memory usage.

Both limiter types are unified through the `LimiterWrapper` interface,
which simplifies consumer usage by providing a consistent `LimitCall`
interface.

A limiter is **saturated** by definition when a limit is completely
overloaded, generally it means a limit request of any size would fail.

Each each base limiter type and the wrapper type have corresponding
providers that give access to a limiter instance based on a weight
key.

Weight keys describes the standard limiting dimensions. There are
currently four standard weight keys: network bytes, request count,
request items, and memory size.

## Key Interfaces

- `LimiterWrapper`: Provides a callback-based limiting interface that
  works with both rate and resource limiters, has a `LimitCall` method.
- `RateLimiter`: Applies time-based limits, has a `Limit` method.
- `ResourceLimiter`: Manages physical resource limits, has
  an `Acquire` method and corresponding `ReleaseFunc`.

### Limiter helpers

The `limiterhelper` subpackage provides:

- Consumer wrappers apply limits to a collector pipeline (e.g.,
  `NewLimitedLogs` to combine a limiter using `consumer.NewLogs`)
- Multi-limiter combinators: `MultiLimiterWrapperProvider` builds a sequence of wrapped limiters.
- Middleware conversion utilities: Convert middleware configurations to `LimiterWrapperProvider`.

## Recommendations

For general use cases, prefer the `LimiterWrapper` interface with its
callback-based approach because it is agnostic to the difference between
rate and resource limiters.

Use the direct `RateLimiter` or `ResourceLimiter` interfaces only in
special cases where control flow can't be easily scoped.

Middleware configuration typically automates the configuration of
network bytes and request count weight keys relatively early in a
pipeline.  Receivers are responsible for limiting request items and
memory size through one of the available helpers.

Processors can apply limiters for specific reasons, for example to
apply limits in data-dependent ways.  Exporters can apply limiters for
the same reasons, for example to apply limits in destination-dependent
ways.

### Limiter blocking and failing

Limiters implementations MAY block the request or fail immediately,
subject to internal logic. A limiter aims to avoid waste, which
requires balancing several factors. To fail a request that has already
been transmitted, received and parsed is sometimes more wasteful than
waiting for a little while; on the other hand waiting for a long time
risks wasting memory. In general, an overloaded limiter that is saturated SHOULD
fail requests immediately.

Limiters implementations SHOULD consider the context deadline when
they block. If the deadline is likely to expire before the limit
becomes available, they should return a standard overload signal.

### Limiter saturation

All limiters feature a `MustDeny` method which is made available for
applications to test when a limit is fully saturated. This special
limit request is defined as the equivalent of passing a zero value to
the limiter.

Limiters SHOULD treat a request for zero units of the limit as a
special case, used for indicating when non-zero limit requests are
likely to fail. This is not an exact requirement; implementations are
free to define their own saturation parameters.

### Limit before or after use

It is sometimes possible to request a limit before it is actually
used. As an example, consider a protocol using a compressed payload,
such that the receivers knows how much memory will be allocated before
the fact. In this case the receiver can request the limit before using
it, but this will not always be the case. Generally, prefer to limit
before use, but either way be consistent.

When using the low-level interfaces directly, limits SHOULD be applied
before creating new concurrent work.

### Examples

Limiters applied through middleware are an implementation detail,
simply configure them using `configgrpc` or `confighttp`.  For the
OTLP receiver (e.g., with two `ratelimiter` extensions):

```
extensions:
  ratelimiter/limit_for_grpc:
    # rate limiter settings for gRPC
  ratelimiter/limit_for_grpc:
    # rate limiter settings for HTTP

receivers:
  otlp:
    protocols:
	  grpc:
	    middlewares:
		- ratelimiter/limit_for_grpc
	  http:
	    middlewares:
		- ratelimiter/limit_for_http
```

@@@
a stream one
a pull-based one
a data-dependent one
