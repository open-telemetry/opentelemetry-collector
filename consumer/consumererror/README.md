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
- `consumererror.NewHTTPStatus` for errors resulting from an HTTP call with an error status code.
- `consumererror.NewGRPCStatus` for errors resulting from a gRPC call with an error status code.

Any error that is not retryable is considered to be a permanent error and will not be retried.

Errors can be joined by passing them to a call to `errors.Join`.

## Other considerations

To keep error analysis simple when looking at an error upstream in a pipeline,
the component closest to the source of an error or set of errors should make a
decision about the nature of the error. The following are a few places where
special considerations may need to be made.

### Fanouts

Pipeline components that perform fanouts should determine for themselves the
precedence of errors they receive when multiple downstream components report an
error.

### Signal conversion

When converting between signals in a pipeline, it is expected that the component
performing the conversion should perform the translation necessary in the error
for any signal item counts.

### Asynchronous processing

Note that the use of any components that do asynchronous processing will cut off
the upward flow of information at the asynchronous component.

## Examples

Creating an error:

```golang
errors.Join(
    consumererror.NewTraces(
        error,
        traces,
        consumererror.WithRetryDelay(2 * time.Duration)
    ),
    consumererror.NewHTTPStatus(error, http.StatusTooManyRequests)
)
```

Using an error:

```golang
err := nextConsumer(ctx, traces)
status := consumererror.StatusError{}
retry := consumererror.Traces{}

if errors.As(err, &status) {
  if status.HTTPStatus >= 500 && errors.As(err, &retry) {
    doRetry(retry.Data())
  }
}
```
