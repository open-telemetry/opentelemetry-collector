# OpenTelemetry Collector Extension Limiter Package

**Document status: development**

The `extensionlimiter` package provides interfaces for limiting
pipelines in the OpenTelemetry Collector, enabling control over data
flow and resource usage through extensions which are configured
through middleware and/or directly by pipeline components.

## Overview

This package defines two foundational limiter **kinds**, with similar
but distinct interfaces.  A limiter extension can be:

- **Rate Limiter**: Controls time-based limits over weights such as
  bytes or items per second.
- **Resource Limiter**: Controls physical limits over weights such as
  concurrent requests or active memory in use.

Requests are quantified with an integer value and identified by
**weight key**, indicating the type of quantity being measured and
limited. There are currently four weight keys with a standard
definition:

1. Network bytes (compressed)
2. Request count
3. Request items
4. Request bytes (uncompressed)

## Early-as-possible application

Limiter extensions should be used as early as possible in a
pipeline. There are two automatic ways that receivers can integrate
with rate limiters:

- Middleware application: rate limiters are automatically recognized
  in the list of middleware. Middleware supports HTTP and gRPC, client
  and server, unary and streaming cases. Middleware automatically
  implements request bytes and request count limits.
- Consumer application: rate limiters can be applied before the next
  consumer in the pipeline, using standard `consumerlimiter.LimiterConfig`
  configuration.

Limiters should be applied, if possible, before work on a request
begins. Work that is done before a limit is requested is subject to
loss, in case the limiter causes failure.

Limiters are not always applied in receivers, but all receivers should
support limiters through middleware and/or `consumerlimiter`.

## Delay-the-caller application

Limiters should be applied so that they delay the caller. This is an
important case of the early-as-possible rule: limit requests should be
made before returning control, in order to slow the process that is
contributing to the limit.

In order to support delaying the caller in complex scenarios,
non-blocking interfaces are provided for each of the limiter
interfaces. Non-blocking APIs allow callers to delay the caller while
requesting the limit and then to perform their work asynchronously.

The `limiterhelper.Wrapper` limiter interface is provided which
simplifies the application of limits to a scoped callback, making it
easy to use a blocking limit request.

## Failure options

Limiters at their discretion can block or fail requests that would
exceed a limit. The decision may influenced by limiter configuration
(e.g., burst, maximum wait parameters) and/or the deadline of the
request context. When the delay is small, it is usually beneficial to
wait instead of failing because (a) avoids the wasted effort (e.g.,
re-transmitting data), (b) delays the caller for the effect of
back-pressure.

When a limiter returns failure, the client should return a
protocol-specific failure code indicating resource exhaustion. The
recognized resource exhaustion codes are HTTP 429 and gRPC
RESOURCE_EXHAUSTED. Receivers should follow protocol-specific
recommendations, which for
[OTLP](https://opentelemetry.io/docs/specs/otlp/) includes returning a
`RetryDelay` parameter.

If the limit request results in waiting, limiters should delay to
allow the request to proceed, however they should give up and return
at some point. As a recommendation, receivers and other components
should allow requests to wait up to configurable fraction of their
deadline. If the request cannot enter a pipeline before for example
half of its deadline, return failure instead of allowing it to
proceed.

### Built-in limiters

#### MemoryLimiter extension

The `memorylimiterextension` gives access to an internal component
named `MemoryLimiter` with an interface named `MustDeny()`.
Components can call this component directly, however when configured
as a limiter extension, this component is modeled as a `RateLimiter`
that is on or off based on the current result of `MustDeny`.

#### RateLimiter

A built-in helper implementation of the RateLimiter interface is
provided, based on `golang.org/x/time/rate.Limter`. These underlying
rate limiters are parameterized by two numbers:

- `limit` (float64): the maximum frequency of weight-units per second
- `burst` (uint64): the "burst" value of the Token-bucket algorithm.

#### ResourceLimiter

A built-in helper implementation of the ResourceLimiter interface is
provided, based on a bounded queue with LIFO behavior.  These
underlying resource limiters are parameterized by two numbers:

- `request` (uint64): the maximum of concurrent resource value admitted
- `waiting` (uint64): the maximum of concurrent resource value permitted to wait

### Examples

#### OTLP receiver

Limiters applied through middleware and/or via receiver-level
limiters.  Middleware limiters are automatically configured using
`configgrpc` or `confighttp`. Receivers can add support at the factory
level using helpers in `consumerlimiter`.

For the OTLP receiver (e.g., with three `ratelimiter` extensions and a
`resourcelimiter` extension):

```yaml
extensions:
  ratelimiter/limit_for_grpc:
    network_bytes: ...

  ratelimiter/limit_for_http:
    network_bytes: ...

  ratelimiter/limit_items:
    request_items: ...

  resourcelimiter/limit_memory:
    request_bytes: ...

receivers:
  otlp:
    protocols:
      grpc:
        middleware:
        - ratelimiter/limit_for_grpc
        - resourcelimiter/limit_memory
      http:
        middleware:
        - ratelimiter/limit_for_http
        - resourcelimiter/limit_memory
    limiters:
      request_items: ratelimiter/limit_items
```

Note that in general, middleware components do not have access to the
number of items in a request, so users are directed to receiver-level
`limiters` configuration to limit items.

#### HTTP metrics scraper

A HTTP pull-based receiver can implement a basic limited scraper loop
as follows. The HTTP client config object's `middlewares` field
automatically configures network bytes and request count limits:

```yaml
receivers:
  httpscraper:
    http:
      middleware:
      - ratelimiter/scraper_request_bytes
      - compression/zstd
      - ratelimiter/scraper_network_bytes
    limiters:
      request_count: ratelimiter/scraper_count
      request_items: ratelimiter/scraper_items
```

##### Data-dependent limits

When a single unit of data contains limits that are assignable to
multiple distinct limiters, we can use the non-blocking rate limiter
interface and drop data that would exceed a limit.  For example, to
limit based on metadata extracted from the OpenTelemetry resource
value:

```go
func (p *processor) limitLogs(ctx context.Context, logsData plog.Logs) (plog.Logs, extensionlimiter.ReleaseFunc, error) {
    var rels extensionlimiter.ReleaseFuncs
	logsData.ResourceLogs().RemoveIf(func(rl plog.ResourceLogs) bool {
	        // For an individual resource, ...
		md := resourceToMetadata(rl.Resource())
		reservation, err := p.limiter.ReserveRate(withMetadata(ctx, md))
		if err != nil {
		    return false
		}
		if reservation.WaitTime() > 0 {
			reservation.Cancel()
			return false
		}
		default:
			return true
		}
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

Here, the limiter's `ReserveRate` function does not block the caller,
allowing the processor to drop data instead.  Note the call to
`RateReservation.Cancel` undoes the effect of the untaken reservation.
The same approach works for `ResourceLimiter` as well using using
`ResourceReservation`, its `Delay` channel `Release` function.

#### Open questions

##### Provider options

An `Option` type has been added as a placeholder in the provider
interfaces. **NOTE: No options are implemented.** Potential options:

- The protocol name
- The signal kind
- The caller's component ID

Because the set of each of these is small, it is possible to
pre-compute limiter instances for the cross product of configurations.

##### Middleware and/or LimiterConfig

The question is how we avoid double-count certain limits whether they
are implemented in middleware, through a factory, through custom
receiver code, or other.

The current proposal avoids double-counting by maintaining state in
the Context indicating when a Limiter component has already seen a
request. This allows early-as-possible limiting to be achieved while
falling-back to consumerlimiter for limits that have not been enforced
automatically.

But, there are variations, like

```yaml
extensions:
  ratelimiter/1:      # weights go here?
  ratelimiter/3:      # weights go here?
  admissionlimiter/2: # weights go here?
  admissionlimiter/4: # weights go here?
receivers:
  otlp:
    protocols:
      grpc:
        middleware:
        - ratelimiter/1
        - admissionlimiter/2
    limiters:
    - ratelimiter/3    # multiple limiters here?
    - admissionlimiter/4 # multiple limiters here?
```

The configuration leaves all weight configuration to the limiter
components. Limiters can implement all weights, but they cannot be
both a rate limiter and a resource limiter.

Should we add indirection? Relax the requirement that a limiter cannot
be both a rate and a resource limiter? In this example, indirection is
added in the form of a "limitermux" extension. This can be an
implementation detail.

```yaml
extensions:
  ...
  limitermux/grpc:
    request_count:
    - ratelimiter/1
    network_bytes:
    - ratelimiter/2
    request_bytes:
    - ratelimiter/3
    - admissionlimiter/1
    request_items:
    - ratelimiter/4

  limitermux/otlp:
    ...

  ...
receivers:
  otlp:
    protocols:
      grpc:
        middleware:
        - limitermux/grpc
    limiters:
    - limitermux/otlp
```

Another route is to back up from the current PR, which makes
configmiddleware.Config into a non-struct `component.ID` type,
eliminating the struct-value with single `id:` field. Instead,
we could add limiter-weight bindings to the middleware config
and/or limiter config:

```yaml
extensions:
  ...
receivers:
  otlp:
    protocols:
      grpc:
        middleware:
        - network_bytes: ratelimiter/1
          request_count: ratelimiter/2
	  request_bytes: admissionlimiter/1
      http:
        middleware:
        - network_bytes: ratelimiter/1
          request_count: ratelimiter/2
	- decompression: zstd,snappy,gzip
        - request_bytes: admissionlimiter/1
	- instrumentation: opentelemetry/1
    limiters:
    - request_items: ratelimiter/3
```

##### Instrumentation

It is possible to extract Counter (RateLimiter) and UpDownCounter
(ResourceLimiter) instrumentation to convey the rates and totals
associated with each limiter.

This should investigated along with investigation into the topics
raised above. Users will be well served if we are able to confidently
extract network-bytes, request-bytes, request-items, and request-count
information without double-counting and provide instrumentation
covering these variables.

##### Are built-in rate and resource limiters needed?

The provided helper implementations are based in
`golang.org/x/time/rate` and
`collector-contrib/internal/otelarrow/admission2`.  We could instead
create two extension implementations for these. The code is a hundred
lines or so each.

The [Elastic rate limiter
processor](https://github.com/elastic/opentelemetry-collector-components/blob/main/processor/ratelimitprocessor/README.md)
would be a good contribution for the community. We are interested in
real-world features such as the ability to set `metadata_keys`.

