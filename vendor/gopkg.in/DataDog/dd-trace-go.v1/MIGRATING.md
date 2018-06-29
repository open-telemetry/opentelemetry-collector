# Migration Guide

This document outlines migrating from an older version of the Datadog tracer (0.6.x) to v1.

Datadog's v1 version of the Go tracer provides not only an overhauled core that comes with huge performance improvements, but also the promise of a new and stable API to be relied on. It is the result of continuous feedback from customers, the community, as well as our extensive internal usage.

As is common and recommended in the Go community, the best way to approach migrating to this new API is by using the [gradual code repair](https://talks.golang.org/2016/refactor.article) method. We have done the same internally and it has worked just great! For this exact reason we have provided a new, [semver](https://semver.org/) friendly import path to help with using both tracers in parallel, without conflict, for the duration of the migration. This new path is `gopkg.in/DataDog/dd-trace-go.v1`.

Our [godoc page](https://godoc.org/gopkg.in/DataDog/dd-trace-go.v1/ddtrace) should deem helpful during this process. We also have the ":warn: soon to be published" [official documentation](https://docs.datadoghq.com/tracing/setup/go/), which contains a couple of examples.

This document will further outline some _before_ and _after_ examples.

## Starting the tracer

The new tracer needs to be started before it can be used. A default started tracer is no longer available. The default tracer is now a no-op.

Here is an example of starting a custom tracer with a non-default agent endpoint using the old API:

```go
t := tracer.NewTracerTransport(tracer.NewTransport("localhost", "8199"))
t.SetDebugLogging(true)
defer t.ForceFlush()
```

This would now become:

```go
tracer.Start(
    tracer.WithAgentAddr("localhost:8199"),
    tracer.WithDebugMode(true),
)
defer tracer.Stop()
```

Notice that the tracer object is no longer returned. Consult the documentation to see [all possible parameters](https://godoc.org/gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer#StartOption) to the `Start` call.

## Service Information

The [`tracer.SetServiceInfo`](https://godoc.org/github.com/DataDog/dd-trace-go/tracer#Tracer.SetServiceInfo) method has been deprecated. The service information is now set automatically based on the value of the [`ext.SpanType`](https://godoc.org/gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext#SpanType) tag that was set on the root span of a trace.

## Spans

Starting spans is now possible with [functional options](https://godoc.org/gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer#StartSpanOption). Which means that all span properties (or none) can be set when starting a span dynamically. Before:

```go
span := tracer.NewRootSpan("web.request", "my_service", "resource_name")
```

Becomes:

```go
span := tracer.StartSpan("web.request", tracer.WithServiceName("my_service"), tracer.WithResourceName("resource_name"))
```

We've done this because in many cases the extra parameters could become tedious, given that service names can be inherited and resource names can default to the operation name. This also allows us to have one single, more dynamic API for starting both root and child spans. Check out all possible [StartSpanOption](https://godoc.org/gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer#StartSpanOption) values to get an idea.

### Children

Here is an example for spawning a child of the previously declared span:
```go
child := tracer.StartSpan("process.user", tracer.ChildOf(span.Context()))
```
You will notice that the new tracer also introduces the concept of [SpanContext](https://godoc.org/gopkg.in/DataDog/dd-trace-go.v1/ddtrace#SpanContext), which is different from Go's context and is used to carry information needed to spawn children of a specific span and can be propagated cross-process. To learn more about distributed tracing check the package-level [documentation](https://godoc.org/gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer#ChildOf) of the `tracer` package.

### Using Go's context

It is also possible to create children of spans that live inside Go's [context](https://golang.org/pkg/context/):
```go
child, ctx := tracer.StartSpanFromContext(ctx, "process.user", tracer.Tag("key", "value"))
```
This will create a child of the span which exists inside the passed context and return it, along with a new context which contains the new span. To add or retrieve a span from a context use the [`ContextWithSpan`](https://godoc.org/gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer#ContextWithSpan) or [`SpanFromContext`](https://godoc.org/gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer#SpanFromContext) functions.

### Setting errors

The [`SetError`](https://godoc.org/github.com/DataDog/dd-trace-go/tracer#Span.SetError) has been deprecated in favour of the [`ext.Error`](https://godoc.org/gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext#Error) tag value which matches other tracing libraries in the wild. Whereas before we had:

```go
span.SetError(err)
```

Now we have:

```go
span.SetTag(ext.Error, err)
```

Note that this tag can accept value of the types `error`, `string` and `bool` as well for setting errors.

### Finishing

The [`FinishWithErr`](https://godoc.org/github.com/DataDog/dd-trace-go/tracer#Span.FinishWithErr) and [`FinishWithTime`](https://godoc.org/github.com/DataDog/dd-trace-go/tracer#Span.FinishWithTime) methods have been removed in favour of a set of [`FinishOption`](https://godoc.org/gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer#FinishOption). For example, this would now become:

```go
span.Finish(tracer.WithError(err), tracer.FinishTime(t))
```

Providing a `nil` value as an error is perfectly fine and will not mark the span as erroneous.

## Further reading

The new version of the tracer also comes with a lot of new features, such as support for distributed tracing and distributed sampling priority. 

* package level documentation of the [`tracer` package](https://godoc.org/gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer) for a better overview.
* [official documentation](https://docs.datadoghq.com/tracing/setup/go/)
