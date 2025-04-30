# OpenTelemetry Collector Extension Limiter Package

**Document status: development**

The `extensionlimiter` package provides interfaces for rate limiting
and resource limiting in the OpenTelemetry Collector, enabling control
over data flow and resource usage through extensions which can be
configured through middleware and/or directly by pipeline components.

## Overview

This package defines two primary limiter **kinds**, which have
different interfaces:

- **Rate Limiters**: Control time-based limits on quantities such as
  bytes or items per second.
- **Resource Limiters**: Manage physical limits on quantities such as
  concurrent requests or memory usage.

Both limiter kinds are unified through the `LimiterWrapper` interface,
which simplifies consumers in most cases by providing a consistent
`LimitCall` interface.

A limiter is **saturated** by definition when a limit is completely
overloaded, generally it means a limit request of any size would fail
at this moment and should be taken as a strong signal to stop
accepting requests.

Each kind of limiter as well as the wrapper type have corresponding
**provider** interface that returns a limiter instance based on a
weight key or keys.

Weight keys describe the standard limiting dimensions. There are
currently four standard weight keys: network bytes, request count,
request items, and memory size.

## Key Interfaces

- `LimiterWrapper`: Provides a callback-based limiting interface that
  works with both rate and resource limiters, has a `LimitCall` method,
  plus a provider type.
- `RateLimiter`: Applies time-based limits, has a `Limit` method,
  plus provider type.
- `ResourceLimiter`: Manages physical resource limits, has
  an `Acquire` method and corresponding `ReleaseFunc`,
  plus a provider type.
- `Limiter`: Any of the above, has a `MustDeny` method.

### Limiter helpers

The `limiterhelper` subpackage provides:

- Consumer wrappers apply limits to a collector pipeline (e.g.,
  `NewLimitedLogs` for a limiter combined with `consumer.NewLogs`)
- Multi-limiter combinators: for simple combined limiter functionality
- Middleware conversion utilities: convert middleware configurations to 
  liiter providers.

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

#### OTLP receiver

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

Note that the OTLP receiver specifically supports multiple protocols
with separate middleware configurations, thus it configures limiters
for request items and memory size on a protocol-by-protocol
basis.

#### HTTP metrics scraper

A HTTP pull-based receiver can implement a basic limited scraper loop
as follows. The HTTP client config object's `middlewares` field
automatically configures network bytes and request count limits:

```
receivers:
  scraper:
    http:
      middlewares:
	  - ratelimiter/scraper
```

Limiter extensions are derived from a host, a middlewares list, and a
list of weight keys. When middleware is configurable at the factory
level, it may be added via `receiver.NewFactory` using
`receiver.WithLimiters(getLimiters)`:

```
func NewFactory() receiver.Factory {
	return xreceiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		xreceiver.WithMetrics(createMetrics, metadata.MetricsStability),
		xreceiver.WithLimiters(getLimiters),
	)
}
```

Here, `getLimiters` is a function to get the effective
`[]configmiddleware.Config` and derive pipeline consumers using
`limiterhelper` adapters.

To acquire a limiter, use `MiddlewaresToLimiterWrapperProvider` to
obtain a combined limiter wrapper around the input `nextMetrics`
consumer.  It will pass `StandardNotMiddlewareKeys()` indicating to
apply request items and memory size:

```
	// Extract limiter provider from middlewares.
	s.limiterProvider, err = limiterhelper.MiddlewaresToLimiterWrapperProvider(
		cfg.Middlewares)
	if err != nil { ... }

	// Here get a limiter-wrapped pipeline and a combination of weight-specific
	// limiters for MustDeny() functionality.
	s.anyLimiter, s.nextMetrics, err = limiterhelper.NewLimitedMetrics(
		s.nextMetrics, limiterhelper.StandardNotMiddlewareKeys(), s.limiterProvider)
	if err != nil { ... }
```

In the scraper loop, use `MustDeny` before starting a scrape:

```
func (s *scraper) scrapeOnce(ctx context.Context) error {
    if err := s.anyLimiter.MustDeny(ctx); err != nil {
		return err		
	}
	
	// Network bytes and request count limits are applied in middleware.
	// before this returns:
	data, err := s.getData(ctx)
	if err != nil {
	    return err
	}

	// Request items and memory size are applied in the pipeline.
	return s.nextMetrics.ConsumeMetrics(ctx, data)
}
```

#### gRPC stream receiver

A gRPC streaming receiver that holds memory across its allocated in
`Send()` and does not release it until after a corresponding `Recv()`
requires use of the lower-level `ResourceLimiter` interface. 
The gRPC  config object's `middlewares` field
automatically configures network bytes and request count limits:

```
receivers:
  streamer:
    grpc:
      middlewares:
	  - ratelimiter/streamer
```

The receiver will check `s.anyLimiter.MustDeny()` as above.  In a
stream, limiters are expected to block the stream until limit requests
succeed, however after the limit requests succeed, the receiver may
wish to return from `Send()` to continue accepting new requests while
the consumer works in a separate goroutine. The limit will be released
after the consumer returns.

```
func (s *scraper) LogsStream(ctx context.Context, stream *Stream) error {
    for {
		// Check saturation for all limiters.
		err := s.anyLimiter.MustDeny(ctx)
		if err != nil { ... }

        // The network bytes and request count are applied in middleware.
		req, err := stream.Recv()
		if err != nil { ... }

		// Allocate memory objects.
		data, err := s.getLogs(ctx, req)
		if err != nil { ... }

		release, err := s.memorySizeLimiter.Acquire(ctx, pdataSize(data))
		if err != nil { ... }
	
		go func() {
			// Request items limit is applied in the pipeline consumer
			err := s.nextMetrics.ConsumeMetrics(ctx, data)
			
			// Release the memory.
			release()
			
			// Reply to the caller.
			stream.Send(streamResponseFromConsumerError(err))
		}
	}
}
```

#### Data-dependent limiter processor

**NOTE: This is not implemented.**

The provider interfaces can be extended to accept a
`map[string]string` that identify limiter instances based on
additional metadata, such as tenant information. Since the limits are
data specific, the limiter will be computed for each request and for
each specific weight key.

Limiter implementations would support options, likely assisted by
`limiterhelper` features to configure them, for configuring
metadata-specific limits.

```
func handleRequest(ctx context.Context, req *Request) error {
	// Get a data-specific limiter:
	md := metadataFromRequest(req)
	lim, err := s.limiterProvider.LimiterWrapper(weightKey, md)
	if err != nil { ... }
	
	if err = lim.MustDeny(ctx); err != nil { ... }

	// Calculate the data and its weight.
	data := dataFromReq(req)
	weight := getWeight(data)

	return lim.LimitCall(ctx, weight, func(ctx context.Context) error {
		return s.nextLogs.ConsumeLogs(ctx, data)
	})
```
