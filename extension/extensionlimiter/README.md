# Limiter extension

The limiter extension interface supports components that limit
requests by several measures of weight.  Extension components are able
to limit entry to pipeline by the number of requests, the
(signal-specific) number of items, and/or the number of bytes of
network and memory consumed as they transit the pipeline.

The limiter interface supports a single method, `Acquire(ctx,
settings, weight)` accepting context, settings, and weight
parameters. 

## Limiter context

The context passed to a Limiter is the one created following the Auth
extension, and it includes the `client.Metadata` of the
request. 

## Limiter settings

Secondary information, including the signal kind of the request, is
included in the settings.  Other information may be included in this
struct in the future, for example the ID of the calling component.

## Limiter weight

The `Weight` struct includes three fields for three different ways to
limit requests, including by request count, by item count, and by
bytes.

The weight `Bytes` field is particularly important for memory-based
limiters. For measuring bytes, components are encouraged to be
approximate. The goal is to estimate the amount of actual memory that
will be used by pipeline data as the request is in transit, so
components may use the uncompressed size of the data as a proxy for
the amount of memory that will be used. The corresponding pipeline
data `Sizer` class may be used.

In cases where components allocate memory in multiple steps, they can
make multiple calls to Acquire to report additional bytes used.
