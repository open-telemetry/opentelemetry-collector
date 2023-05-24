# Consumer errors

This package contains error types that should be returned by a consumer when an
error occurs while processing telemetry. The error types included in this
include functionality for communicating upstream in the pipeline details about
the error for use by the caller. Ideally the top-level error returned by a
consumer in its consume function should be from this package.

## Use cases

**Retry logic**: Errors should be allowed to embed rejected telemetry records to
be retried by the consumer.

**Indicating partial success**: Errors can indicate that not all records were
accepted, for example as in an OTLP partial success message.

**Communicating network error codes**: Errors should allow embedding information
necessary for the Collector to act as a proxy for a backend, i.e. relay a status
code returned from a backend in a response to a system upstream from the
Collector.

**Combining errors from multiple downstream consumers**: If errors occur in
multiple components downstream from the caller in a pipeline, the caller should
be able to combine the downstream errors into an error it returns.

## Creating an error

To create a new error, call the `consumererror.NewError` function with the error
you want to wrap and any options that add additional metadata. Errors are either
considered permanent or retryable, depending on whether the
`WithRetryable[Signal]` option is passed. A permanent error indicates that the
error will always occur for a given set of telemetry records and should be
dropped. The permanence of an error can be checked with
`consumererror.IsPermanent`.

To help communicate information about the error to the caller, options with
additional metadata may be included. While some of the options are semantically
mutually-exclusive and shouldn't be combined, any set of options can be used
together and the package will determine which option takes precedence.

### WithRejectedCount(rejected int)

Include a count of the records that were permanently rejected (spans,
datapoints, log records, etc.). The caller should have a full count of rejected
records, so this option is only needed to indicate a partial success.

When using this option in an exporter or other component dealing with mapping
non-pdata formats, the rejected count should be based on the count of pdata
records that failed.

### WithRetryable\[Signal\](pdata, ...RetryOptions)

Indicate that a temporary condition is the cause of an error and that the
request should be retried with the default delay. Use of this option means that
an error is not considered permanent.

#### WithRetryDelay(delay time.Duration)

Indicate that the payload should be retried after a certain amount of time.

### WithHTTPStatus(int)

Annotate the error with an HTTP status code obtained from a response during
exporting.

### WithGRPCStatus(*status.Status)

Annotate the error with a gRPC status code obtained from a response during
exporting.

### WithDownstreamMetadata(consumererror.Error)

Take metadata from a downstream error (e.g. network codes) when creating this
error. Does not copy any signal data from the downstream error.

### WithDownstreamErrors(...error)

Allows fanout consumers to include multiple downstream errors as the cause of
this error.

## Reading from an error

The `consumererror.Error` type supports the following methods. Each method has a
method signature like `(data, bool)` with the second value indicating whether
the option was passed during the error's creation.

- `Count`: Get the count of permanently rejected records.
- `Retryable[Signal]`: Gets the information necessary to retry the request for a
  given signal.
- `ToHTTP`: Returns an integer representing the HTTP status code. If both
  `WithHTTPStatus` and `WithGRPCStatus` were passed, the HTTP status is used
  first and gRPC status second.
- `ToGRPC`: Returns a `*status.Status` object representing the status returned
  from the gRPC server. If both `WithHTTPStatus` and `WithGRPCStatus` were
  passed, the gRPC status is used first and HTTP status second.
- `DownstreamErrors`: Returns multiple downstream errors if multiple errors
  happened downstream from this component.

## Passing errors upstream

When passing errors upstream, a component should ensure that all signal data and
counts in an error come from the component itself and not from a downstream
error. This way any processing that was done in the component isn't reflected in
the data that is passed to the caller, which is expecting the included data to
reflect what it sent downstream. To do this, use the `WithDownstreamMetadata`
option when creating the new error, and pass any related signal data as part of
it.

## Other considerations

### Asynchronous processing

Note that the use of any components that do asynchronous processing, for example
the batch processor, will cut off the upward flow of information at the
asynchronous component. This means that something like a network status code
cannot be propagated from an exporter to a receiver if the batch processor is
used in the pipeline.

## Examples

Creating an error:

```golang
consumererror.NewError(
    consumererror.WithRetryableTraces(
        traces,
        consumererror.WithRetryDelay(2 * time.Duration)
    ),
    consumererror.WithHTTPStatus(429)
)
```

Using an error:

```golang
err := nextConsumer(ctx, traces)

if cErr, ok := consumererror.As(err); ok {
    code, ok := cErr.ToHTTP()
    if ok {
        statusCode = code
    }
    
    retry, ok := cErr.RetryableTraces()

    if ok {
        doRetry(retry)
    }
}
```

