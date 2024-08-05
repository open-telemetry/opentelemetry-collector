# Consumer errors

This package contains error types that should be returned by a consumer when an
error occurs while processing telemetry. The error types included in this
package provide functionality for communicating details about the error for use
upstream in the pipeline. Ideally the error returned by a component in its
`consume` function should be from this package.

## Error classes

**Retryable**: Errors are retryable if re-submitting data to a sink may result
in a successful submission.

**Permanent**: Errors are permanent if submission will always fail for the
current data. Errors are considered permanent unless they are explicitly marked
as retryable.

## Use cases

**Retry logic**: Errors should be allowed to include information necessary to
perform retries.

**Indicating partial success**: Errors can indicate that not all items were
accepted, for example as in an OTLP partial success message. OTLP backends will
return failed item counts if a partial success occurs, and this information can
be propagated up to a receiver and returned to the caller.

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

**scraperhelper**: The scraper helper can use information about errors from
downstream to affect scraping. For example, if the backend is struggling with
the volume of data, scraping could be slowed, or the amount of data collected
could be reduced. 

**exporterhelper**: When an exporter returns a retryable error, the
exporterhelper can use this information to retry. Permanent errors will be
forwarded back up the pipeline.

**obsreport**: Recording partial success information can ensure we correctly
track the number of failed telemetry records in the pipeline. Right now, all
records will be considered to be failed, which isn't accurate when partial
successes occur.

## Creating Errors

Errors can be created by calling `consumererror.New(err, opts...)` where `err`
is the underlying error, and `opts` is one of the provided options for supplying
additional metadata:

- `consumererror.WithGRPCStatus`
- `consumererror.WithHTTPStatus`

The following options are not currently available, but may be made available in
the future:

- `consumererror.WithRetry`
- `consumererror.WithPartial`
- `consumererror.WithMetadata`

All options can be combined, we assume that the component knows what it is doing
when seemingly conflicting options.

Two examples:

- `WithRetry` and `WithPartial` are included together: Partial successes are
  considered permanent errors in OTLP, which conflicts with making an error
  retryable by including `WithRetry`. However, per our definition of what makes
  a permanent error, this error has been marked as retryable, and therefore we
  assume the component producing this error supports retyable partial success
  errors.
- `WithGRPCStatus` and `WithHTTPStatus` are included together: While the
  component likely only sent data over one of these transports, our errors will
  produce the given status if it is included on the error, otherwise it will
  translate a status from the status for the other transport. If both of these
  are given, we assume the component wanted to perform its own status
  conversion, and we will simply give the status for the requested transport
  without performing any conversion.

**Example**:

```go
consumererror.New(err,
  consumererror.WithRetry(
    consumerrerror.WithRetryDelay(10 * time.Second)
  ),
  consumererror.WithGRPCStatus(codes.InvalidArgument),
)
```

### Retrying data submission

> [!WARNING] This function is currently in the design phase. It is not available
> and may not be added. The below is a design describing how this may work.

If an error is transient, the `WithRetry` option corresponding to the relevant
signal should be used to indicate that the error is retryable and to pass on any
retry settings. These settings can come from the data sink or be determined by
the component, such as through runtime conditions or from user settings.

The data for the retry will be provided by the component performing the retry.
This will require all processing to be completely redone; in the future,
including data from the failed component so as to not retry this processing may
be made as an available option.

To ensure only the failed pipeline branch is retried, the sequence of components
that created the error will be recorded by a pipeline utility as the error goes
back up the pipeline.

**Note**: If retry information is not included in an error, the error will be
considered permanent and will not be retried.

**Usage:**

```go
consumererror.WithRetry(
  consumerrerror.WithRetryDelay(10 * time.Second)
)
```

The delay is an optional setting that can be provided if it is available.

### Indicating partial success

> [!WARNING] This function is currently in the design phase. It is not available
> and may not be added. The below is a design describing how this may work.

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

> [!WARNING] This function is currently in the design phase. It is not available
> and may not be added. The below is a design describing how this may work.

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

When a fanout receives multiple errors, it will combine them with
`(consumererror.Error).Combine(errs...)` and pass them upstream. The upstream
component can then later pull all errors out for analysis.

### Retrieving errors

> [!WARNING] This functionality is currently experimental, and the description
> here is for design purposes. The code snippet may not work as-written.

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
    data.HTTPStatus()
    data.Retryable()
    data.Partial()
  }
}
```

### Error data

> [!WARNING] The description below is a design proposal for how this
> functionality may work. See `error.go` within this package for the current
> functionality.

Obtaining data from an error can be done using an interface that looks something
like this:

```go
type ErrorData interface {
  // Returns the underlying error
  Error() error

  // Second argument is `false` if no code is available.
  HTTPStatus() (int, bool)

  // Second argument is `false` if no code is available.
  GRPCStatus() (*status.Status, bool)

  // Second argument is `false` if no retry information is available.
  Retryable() (Retryable, bool)

  // Second argument is `false` if no partial counts were recorded.
  Partial() (Partial, bool)
}

type Retryable struct {}

// Returns nil if no delay was set, indicating to use the default.
// This makes it so a delay of `0` indicates to resend immediately.
func (r *Retryable) Delay() *time.Duration {}

type Partial struct {}
```

## Other considerations

### Mixed error classes

When a receiver sees a mixture of permanent and retryable errors from downstream
in the pipeline, it must first consider whether retries are enabled within the
Collector.

**Retries are enabled**: Ignore the permanent errors, retry data submission for
only components that indicated retryable errors.

**Retries are disabled**: In an asynchronous pipeline, simply do not retry any
data. In a synchronous pipeline, the receiver should return a permanent error
code indicating to the caller that it should not retry the data submission. This
is intended to not induce extra failures where we know the data submission will
fail, but this behavior could be made configurable by the user.

### Signal conversion

When converting between signals in a pipeline, it is expected that the connector
performing the conversion should perform the translation necessary in the error
for any signal item counts. If the converted count cannot be determined, the
full count of pre-converted signals should be returned.

### Asynchronous processing

The use of any components that do asynchronous processing in a pipeline will cut
off error backpropagation at the asynchronous component. The asynchronous
component may communicate error information using the Collector's own signals.
