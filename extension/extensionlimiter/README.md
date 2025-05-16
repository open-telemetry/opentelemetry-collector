# OpenTelemetry Collector Extension Limiter Package

**Document status: development**

The `extensionlimiter` package provides interfaces for rate limiting
and resource limiting in the OpenTelemetry Collector, enabling control
over data flow and resource usage through extensions that can be
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
overloaded in at least one weight, generally it means callers should
immediately deny work to continue on the request.

Each kind of limiter as well as the wrapper type have corresponding
**provider** interfaces that return a limiter instance based on a
weight keys.

Weight keys describe the standard limiting dimensions. There are
currently four standard weight keys: network bytes, request count,
request items, and memory size.  Callers use the `Checker` interface
to check whether any weight keys (from a set) are saturated.

## Key Interfaces

- `LimiterWrapper`: Provides a callback-based limiting interface that
  works with both rate and resource limiters, has a `LimitCall` method,
  plus a provider type.
- `RateLimiter`: Applies time-based limits, has a `Limit` method,
  plus provider type.
- `ResourceLimiter`: Manages physical resource limits, has
  an `Acquire` method and a corresponding `ReleaseFunc`,
  plus a provider type.
- `Checker`: Has a `MustDeny` method.

### Limiter helpers

The `limiterhelper` subpackage provides:

- Consumer wrappers apply limits to a collector pipeline (e.g.,
  `NewLimitedLogs` for a limiter combined with `consumer.NewLogs`)
- Multi-limiter combinators: for simple combined limiter functionality
- Middleware conversion utilities: convert middleware configurations
  to limiter providers.

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
risks wasting memory. In general, an overloaded limiter that is
saturated SHOULD fail requests immediately.

Limiter implementations SHOULD consider the context deadline when
they block. If the deadline is likely to expire before the limit
becomes available, they should return a standard overload signal.

### Limiter saturation

Rate and resource limiter providers have a `GetChecker` method to
provide a `Checker`, featuring a `MustDeny` method which is made
available for applications to test when any limit is fully
saturated that would eventually deny the request.

The `Checker` is consulted at least once and applies to all weight
keys.  Because a `Checker` can be consulted more than once by a
receiver and/or middleware, it is possible for requests to be denied
over the saturation of limits they were already granted. Users should
configure external load balancers and/or horizontal scaling policies
to avoid cases of limiter saturation.

### Limit before or after use

It is sometimes possible to request a limit before it is actually
used. As an example, consider a protocol using a compressed payload,
such that the receiver knows how much memory will be allocated before
the fact. In this case the receiver can request the limit before using
it, but this will not always be the case. Generally, prefer to limit
before use, but either way be consistent.

When using the low-level interfaces directly, limits SHOULD be applied
before creating new concurrent work.

### Examples

#### OTLP receiver

Limiters applied through middleware are an implementation detail,
simply configure them using `configgrpc` or `confighttp`. For the
OTLP receiver (e.g., with two `ratelimiter` extensions):

```yaml
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
for request items and memory size on a protocol-by-protocol basis.

#### HTTP metrics scraper

A HTTP pull-based receiver can implement a basic limited scraper loop
as follows. The HTTP client config object's `middlewares` field
automatically configures network bytes and request count limits:

```yaml
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

```golang
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
consumer. It will pass `StandardNotMiddlewareKeys()` indicating to
apply request items and memory size:

```golang
    // Extract limiter provider from middlewares.
    s.limiterProvider, err = limiterhelper.MiddlewaresToLimiterWrapperProvider(
        cfg.Middlewares)
    if err != nil { ... }
	
	// Extract a checker from the provider
	s.checker, err = s.limiterProvider.GetChecker()
	if err != nil { ... }

    // Here get a limiter-wrapped pipeline and a combination of weight-specific
    // limiters for MustDeny() functionality.
    s.nextMetrics, err = limiterhelper.NewLimitedMetrics(
        s.nextMetrics, limiterhelper.StandardNotMiddlewareKeys(), s.limiterProvider)
    if err != nil { ... }
```

In the scraper loop, use `MustDeny` before starting a scrape:

```golang
func (s *scraper) scrapeOnce(ctx context.Context) error {
    if err := s.checker.MustDeny(ctx); err != nil {
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

```yaml
receivers:
  streamer:
    grpc:
      middlewares:
      - ratelimiter/streamer
```

The receiver will check `s.checker.MustDeny()` as above.  In a stream,
limiters are expected to block the stream until limit requests
succeed, however after the limit requests succeed, the receiver may
wish to return from `Send()` to continue accepting new requests while
the consumer works in a separate goroutine. The limit will be released
after the consumer returns.

```golang
func (s *scraper) LogsStream(ctx context.Context, stream *Stream) error {
    for {
        // Check saturation for all limiters, all keys.
        err := s.checker.MustDeny(ctx)
        if err != nil { ... }

        // The network bytes and request count limits are applied in middleware.
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

#### Open questions

##### Middleware implementation details

Details are
important. [#12700](https://github.com/open-telemetry/opentelemetry-collector/pull/12700)
contained a `limitermiddleware` implementation which was a middleware
that called a limiter for HTTP and gRPC. Roughly the same code will be
used, and the details will come out.

##### Provider options

An `Option` type has been added as a placeholder in the provider
interfaces. **NOTE: No options are implemented.** Potential options:

- The protocol name
- The signal kind
- The caller's component ID

Because the set of each of these is small, it is possible to
pre-compute limiter instances for the cross product of configurations.

##### Context-dependent limits

Client metadata (i.e., headers) may be used in the context to make
limiter decisions. These details are automatically extracted from the
Context passed to `MustDeny`, `Limit`, `Acquire`, and `LimitCall`
functions. No examples are provided. How will limiters configure, for
example, tenant-specific limits?

##### Data-dependent limits

When a single unit of data contains limits that are assignable to
multiple distinct limiters, one option available to users is to split
requests and add to their context and run them concurrently through
context-dependent limiters.  See
[#39199](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/39199).

Another option is to add support for non-blocking limit requests. For
example, to apply limits using information derived from the
OpenTelemetry resource, we might do something like this pseudo-code:

```
func (p *processor) limitLogs(ctx context.Context, logsData plog.Logs) (plog.Logs, extensionlimiter.ReleaseFunc, error) {
    var rels extensionlimiter.ReleaseFuncs
	logsData.ResourceLogs().RemoveIf(func(rl plog.ResourceLogs) bool {
		md := resourceToMetadata(rl.Resource())
		rel, err := p.nonBlockingLimiter.TryLimitOrAcquire(withMetadata(ctx, md))
		if err != nil {
		    return false
		}
		rels = append(rels, rel)
		return true
	})
	if logsData.ResourceLogs().Len() == 0 {
		return logsData, func() {}, processorhelper.ErrSkipProcessingData
	}
	return logsData, rels.Release, nil
}

func (p *processor) ConsumeLogs(ctx context.Context, logsData plog.Logs) error {
	logsData, release, err = limitLogs(ctx, logsData)
	if err != nil {
	    return err
	}
	defer release()
	return p.nextLogs.ConsumeLogs(ctx, logsData)
}
```

Here, the release a new `TryLimitOrAcquire` function abstracts the
form of a non-blocking call to either form of limiter. If the
underyling limiter is a rate limiter, the release function will be a
no-op.
