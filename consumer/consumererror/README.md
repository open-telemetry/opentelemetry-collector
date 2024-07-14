# Consumer errors

This package contains error types that should be returned by a consumer when an
error occurs while processing telemetry. The error types included in this
package provide functionality for communicating details about
the error for use upstream in the pipeline. Ideally the error returned by a
component in its `consume` function should be from this package.

## Use cases

**Retry logic**: Errors should be allowed to embed rejected telemetry items to
be retried by the consumer.

**Indicating partial success**: Errors can indicate that not all items were
accepted, for example as in an OTLP partial success message.

**Communicating network error codes**: Errors should allow embedding information
necessary for the Collector to act as a proxy for a backend, i.e. relay a status
code returned from a backend in a response to a system upstream from the
Collector.

## Creating Errors

Errors can be created through one of the three constructors provided:

- `consumererror.New[Signal]` for retryable errors.
- `consumererror.NewPartial` for errors where only some of the items failed.
- `consumererror.NewHTTP` for errors resulting from an HTTP call with an error status code.
- `consumererror.NewGRPC` for errors resulting from a gRPC call with an error status code.
- `consumererror.NewComponent` for metadata around the component that produced the error.

Only retryable errors should be retried, all other errors are considered permanent
and should not be retried.

Errors can be joined by passing them to a call to `errors.Join`.


### Per-signal retryable errors

If an error is considered transient and data processing should be retried, the data
that should be retried should be created using `NewTraces`, `NewMetrics` or `NewLogs`
with the data that should be retried, e.g. `NewTraces(err, traces)`. The data that
is contained in the error will be the only data that is retried, the caller should
not use its copy of the data for the retry.

## Other considerations

To keep error analysis simple when looking at an error upstream in a pipeline,
the component closest to the source of an error or set of errors should make a
decision about the nature of the error. The following are a few places where
special considerations may need to be made.

### Fanouts

#### Permanent errors

When a fanout component receives multiple permanent errors from downstream in
the pipeline, it should determine the error with the highest precedence and
forward that upward in the pipeline. If a precedence cannot be determined,
the component should forward all errors.

#### Retryable errors

When a fanout component receives a retryable error, it must only retry the data
the failed pipeline branch.

#### Mixed errors

When a mix of retryable and permanent errors are received from downstream
pipeline branches, the fanout component should continue all retries for
retryable data and return a permanent error upstream regardless of whether
any of the retries succeed. The component performing retries should capture
which pipelines succeeded or failed using the Collector's pipeline observability
mechanisms.

### Signal conversion

When converting between signals in a pipeline, it is expected that the component
performing the conversion should perform the translation necessary in the error
for any signal item counts. If the converted count cannot be determined, the full
count of pre-converted signals should be returned.

### Asynchronous processing

The use of any components that do asynchronous processing in a pipeline will cut
off error backpropagation at the asynchronous component. The asynchronous
component may communicate error information using the Collector's own signals.

## Examples

Creating an error:

```golang
errors.Join(
    consumererror.NewComponent(exp.componentID),
    consumererror.NewTraces(
        error,
        traces,
        consumererror.WithRetryDelay(2 * time.Duration)
    ),
    consumererror.NewHTTP(error, http.StatusTooManyRequests)
)
```

Using an error:

```golang
err := nextConsumer(ctx, traces)
var status consumererror.TransportError
var retry consumererror.Traces

if errors.As(err, &status) {
  if status.HTTPStatus >= 500 && errors.As(err, &retry) {
    doRetry(retry.Data())
  }
}
```
