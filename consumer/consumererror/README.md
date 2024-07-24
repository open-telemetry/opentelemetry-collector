# Consumer errors

This package contains error types that should be returned by a consumer when an
error occurs while processing telemetry. The error types included in this
package provide functionality for communicating details about
the error for use upstream in the pipeline. Ideally the error returned by a
component in its `consume` function should be from this package.

## Error classes

**Retryable**: Errors are retryable if re-submitting data to a sink may result
in a successful submission.

**Permanent**: Errors are permanent if submission will always fail for the
current data.

## Use cases

**Retry logic**: Errors should be allowed to include information necessary to
perform retries.

**Indicating partial success**: Errors can indicate that not all items were
accepted, for example as in an OTLP partial success message.

**Communicating network error codes**: Errors should allow embedding information
necessary for the Collector to act as a proxy for a backend, i.e. relay a status
code returned from a backend in a response to a system upstream from the
Collector.

## Current targets for using errors

**Receivers**: Receivers should be able to consume multiple errors downstream
and determine the best course of action based on the user's configuration. This
may entail either keeping the retry queue inside of the Collector by having the
receiver keep track of retries, or may involve having the caller manage the
retry queue by returning a retryable network code with relevant information.

**exporterhelper**: When an exporter returns a retryable error, the
exporterhelper can use this information to manage the sending queue if it is
enabled. Permanent errors will be forwarded back up the pipeline.

## Creating Errors

Errors can be created by calling `consumererror.New(err, opts...)` where `err`
is the underlying error, and `opts` is one of the provided options for supplying
additional metadata:

- `consumererror.WithRetry[Signal]`
- `consumererror.WithPartial`
- `consumererror.WithGRPCStatus`
- `consumererror.WithHTTPStatus`
- `consumererror.WithMetadata`

### Retrying data submission

If an error is transient, the `WithRetry` option corresponding to the relevant
signal should be used to indicate that the error is retryable and to pass on any
retry settings. These settings can come from the data sink or be determined by
the component, such as through runtime conditions or from user settings.

**Usage:**

```go
consumererror.WithRetry(
  componentID,
  consumerrerror.WithRetryDelay(10 * time.Second)
)
```

`componentID` should be the ID of the component creating the error, so it is
known which components to retry submission on. The delay is an optional setting
that can be provided if it is available.

### Indicating partial success

If the component receives an OTLP partial success message (or other indication
of partial success), it should include this information with a count of the
failed records.

**Usage:**

```go
consumererror.WithPartial(failedRecords)
```

### Indicating error codes from network transports

If the failure occurred due to a network transaction, the exporter should record
the status code of the message received from the backend. This information can
be then relayed to the receiver caller if necessary. Note that when the upstream
component reads a code, it will read a code for its desired transport, and the
code may be translated depending whether the input and output transports are
different. For example, a gRPC exporter may record a gRPC status. If a gRPC
receiver reads this status, it will be exactly the provided status. If an HTTP
receiver reads the status, it wants an HTTP status, and the gRPC status will be
converted to an equivalent HTTP code.

**Usage:**

```go
consumererror.WithGRPCStatus(codes.InvalidArgument)
consumererror.WithHTTPStatus(http.StatusTooManyRequests)
```

### Including custom data

Custom data can be included as well for any additional information that needs to
be propagated back up the pipeline. It is up to the consuming component if or
how this data will be used.

**Usage:**

```go
consumererror.WithMetadata(MyMetadataStuct{})
```

To keep error analysis simple when looking at an error upstream in a pipeline,
the component closest to the source of an error or set of errors should make a
decision about the nature of the error. The following are a few places where
special considerations may need to be made.

## Using errors

### Fanouts

When fanouts receive multiple errors, they will combine them with
`(consumererror.Error).Combine(errs...)` and pass them upstream. The upstream
component can then later pull all errors out for analysis.

### Retrieving errors

When a receiver gets a response that includes an error, it can get the data out
by doing something similar to the following. Note that this uses the `ErrorData`
type, which is for reading error data, as opposed to the `Error` type, which is
for recording errors.

```go
cerr := consumererror.Error{}
var errData []consumerError.ErrorData 

if errors.As(err, &cerr) {
  errData := cerr.Data()

  for _, data := range errData {
    data.HTTPCode()
    data.RetryableTraces()
    data.Partial()
  }
}
```

## Other considerations

### Signal conversion

When converting between signals in a pipeline, it is expected that the component
performing the conversion should perform the translation necessary in the error
for any signal item counts. If the converted count cannot be determined, the full
count of pre-converted signals should be returned.

### Asynchronous processing

The use of any components that do asynchronous processing in a pipeline will cut
off error backpropagation at the asynchronous component. The asynchronous
component may communicate error information using the Collector's own signals.
