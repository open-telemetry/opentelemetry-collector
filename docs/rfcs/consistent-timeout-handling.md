# Consistent deadline handling and request cancellation in OpenTelemetry pipelines

Establish normative guidelines for components to follow regarding context
deadline and cancellation across the OpenTelemetry Collector.

## Motivation

We are motivated, first, to have consistent behavior for components
that are directly or indirectly involved in real-time decisions about
deadline and cancellation.

Because the OpenTelemetry Collector is anchored in the Golang language,
that language's Context mechanism can be found throughout Collector
interfaces. It is unclear what and when components are meant to do with
the Context, except to pass it, synchronously, to the next component.

For components that batch telemetry (e.g., batch processor, batch sender),
for those that may or may not block (e.g., queue sender), and for those
that are explicitly focused on time (e.g., retry, timeout senders), we
can ideally wish for consistent behavior.

Definitions:

- the *timeout* of a context is the duration until its deadline. An expired
  deadline has no timeout, whereas a viable deadline has not yet expired. A
  context can have no deadline, and thus no timeout.
- the *timeout* of a configuration struct is the duration that will be added
  relative to the current time until a deadline. No timeout means no deadline.

Here are some of the scenarios where consistent behavior is desirable:

- A request arrives with an already-expired deadline. Should the component
  immediately return a deadline-exceeded error status?
- A request arrives with a viable deadline, but the request does not
  succeed in time. Should the component immediately return a deadline-exceeded
  status, or should it wait for its response?
- A request arrives and has to acquire a resource (e.g., space in a queue) that
  is not immediately available. Should the component fail "fast" or stall the
  request, hoping the resource will become available before the deadline?
- A request arrives and the component calls for an additive timeout that is
  greater than the request's deadline.  For example:
  - a batch processor is configured with `1s` timeout, and an arriving
    request has a `0.5s` timeout
  - timeout sender has `5s` configured timeout, arriving request has `2s` timeout
  - retry sender has a maximum elapsed timeout of `1m`, arriving request has `5s`
    timeout.

Since there are many components, we choose a success criteria aimed at
giving OpenTelemetry SDK users a first-class experience. Our goal is that
an OpenTelemetry SDK configured with its OTLP/gRPC exporter sending to
an OpenTelemetry Collector gateway which forwards to another OpenTelemetry
service has a good user experience. This requires that when pipeline latency
rises, the user can adjust their export timeout to improve success,
without unnecessary retries.

For this outcome to be achieved, we require:

- Batch processor/sender propagates maximum deadline in batch
- If enabled, queue sender blocks until queue space is available
- Timeout sender configured not to lower an already-configured timeout.

## Explanation

The questions listed in the motivation above correspond with common
questions faced by the core sub-components.

- Batch processor
- Batch sender (exporterbatcher)
- Timeout sender (exporterhelper)
- Retry sender (exporterhelper)
- Queue sender (exporterhelper)

In the following sub-sections, we raise questions about the existing
support for timeouts.

### Timeout sender: existing

The existing behavior of the Golang context mechanism is defined
by [`context.WithDeadline()`](https://pkg.go.dev/context#WithDeadline),
as follows:

> WithDeadline returns a copy of the parent context with the deadline
> adjusted to be no later than d. If the parent's deadline is already
> earlier than d, WithDeadline(parent, d) is semantically equivalent
> to parent.

The Timeout sender component makes a synchronous `Send()` call on the
next component after calling `context.WithTimeout()`:

> WithTimeout returns WithDeadline(parent, time.Now().Add(timeout)).

The Timeout sender, therefore, can only be used to shorten a timeout.
There is no built-in support for extending, ignoring, or preserving
existing context deadlines in a Collector pipeline. No special treatment
is given to the case where request deadline has already expired.

### Retry sender: existing

The Retry sender is configured with a maximum elapsed time and a
programmable delay parameter. This component synchronously calls
the next component, after which it checks the response and possibly
waits for another attempt.

While in the synchronous call, deadline handling is the responsibility
of the next component in the pipeline, which is the Timeout sender.
When a request will be retried, the component delays in a `select`
statement, waiting for a timer wakeup or context cancellation.

This component is deadline-aware. It does not enter the `select`
statement in the case where its deadline will expire before the next
calculated retry attempt.

### Batch processor and sender: existing

These components are expected to have identical batching behavior,
as discussed in a [companion RFC](https://github.com/open-telemetry/opentelemetry-collector/pull/11947). As stated in that proposal,
these components are not deadline-aware. For error transmission and
consistent treatment of timeout, a batching process should wait for
the request to complete, until context cancellation, or until the
deadline expires.

Should these components set a deadline for the batch? Should these
components prioritize requests with approaching deadlines?

### Queue sender: existing

This component attempts to enqueue a request. If the queue is full
or unavailable, it immediately returns a queue-full error.

When the deadline permits it, is there an option to wait for entry
into the queue? If no deadline is set, is there an option to
wait indefinitely?

### Receiver support

Especially when gRPC is in use, requests typically arrive with a pre-
configured deadline. In a collector-to-collector pipeline scenario, the
receiver will set a request deadline corresponding with the timeout
set by the Timeout sender.

If the timeout is especially small, can it be set to a minimum value?
If the timeout is especially large or unset, can it set to a maximum value?

## Golang-specific guidelines

The Golang Context follows pipeline data through the OpenTelemetry Collector.
There are several signals combined into this parameter that component authors
should be aware of.

- Deadline: the Context carries an optional deadline. Libraries such as
  gRPC propagate deadline from client-to-server.
- Cancellation: the Context carries a `Done()` channel, which allows a
  goroutine to abort when the caller is no longer interested in a
  result due to deadline expired, broken connection, etc.
- client.Metadata: the collector `client` package supports propagating
  the source connection details, authorization data, and metadata expressed
  as key-value pairs.

### Making a synchronous call

Many components, generally those designed to transform data in a pipeline,
are described as synchronous. In this calling convention, the caller
becomes responsible for deadline and cancellation.

```golang
func (my *myComponent) Consume(ctx context.Context, req data.Request) error {
    // prepare to call the following component
    nextReq := someLogic(req)
    // pass the context in synchronous calls;
    if err := my.next.Consume(ctx, nextReq); err != nil {
        // propagate the returned error
        return err
    }
    return nil
}
```

### Making an asynchronous call

When a component performs an operation that potentially blocks the request,
be sure to use the `ctx.Done()` channel in every `select` statement.  Components
should avoid bare, blocking channel operations:

```golang
    // don't do this
    go asyncTask(ctx, req, someChan)
    resp := <-someChannel
```

Do this instead:

```golang
    // pass context, so the async task can be deadline-aware
    go asyncTask(ctx, req, someChan)
    var resp Response
    select {
    case <-ctx.Done():
        // cancellation: return the error cause.
        return ctx.Err()
    case resp = <-respChan:
        // success: continue processing
    }
```

### Check for cancellation

To simply check whether a request context is viable, not cancelled,
before proceeding with a calculation:

```golang
    select {
    case <-ctx.Done():
        // the context is cancelled, maybe deadline-exceeded.
        return ctx.Err()
    default:
        // OK to continue
    }
```

## Specific proposals

### Timeout sender

The existing TimeoutConfig struct has one field.  The current documentation
for `timeout` reads:

```golang
 // Timeout is the timeout for every attempt to send data to the backend.
 // A zero timeout means no timeout.
```

These statements are only true in case the Queue sender is used, which
ignores the request deadline. Otherwise, due to the "no later than" semantics of
`WithDeadline()`, the `timeout` field can be interpreted as a maximum
timeout value. We propose an additional `min_timeout` to limit timeout
in the other direction:

```golang
type TimeoutConfig struct {
 // Timeout is the maximum allowed timeout for requests made
 // by this exporter component. If non-zero, the outgoing
 // request context will have a timeout no-later-than this
 // duration. This field must not be negative.
 //
 // If this field is zero, the interpretation depends on
 // the setting of MinTimeout.
 Timeout time.Duration `mapstructure:"timeout"`

  // MinTimeout, if >= 0, and an arriving request has a timeout less
  // than this duration. the request immediately fails with a
  // deadline-exceeded error. If negative, the arriving request
  // deadline is unchecked. MinTimeout=0 implies checking for an
  // expired deadline. MinTimeout>0 implies requiring the user to
  // have a minimum timeout.
  //
  // In case MinTimeout<0 (i.e, deadline unchecked) and Timeout == 0,
  // special meaning is given. In this case, the timeout is erased from the
  // context using context.WithoutCancel().
  MinTimeout time.Duration `mapstructure:"min_timeout"
}
```

The two fields are meant to be interepreted in combination, according to
this matrix:

| Explanation | Timeout=0 | Timeout>0 |
| -- | -- | -- |
| **MinTimeout<0** | No minimum, no maximum case. In this case, the deadline is explicitly erased. | A maximum timeout is imposed, but the deadline is not checked. |
| **MinTimeout≥0** | The deadline is checked, but a new maximum is not imposed. Allows existing and no-deadline requests to pass. | The deadline is checked and a maximum is imposed. |

The current component uses a default Timeout of `5s`. This proposal
would use default MinTimeout of `0`, meaning to enforce a non-negative
deadline, rejecting expired contexts. Proposed context-handling logic is listed below.

### Retry sender

This component is already deadline-aware.

### Queue sender

A new field will be introduced, with default matching the
original behavior of this component.

```golang
  // FailFast indicates that the queue should immediately
  // reject requests when the queue is full, without considering
  // the request deadline. Default: true.
  FailFast bool `mapstructure:"fail_fast"`
```

In case the new FailFast flag is false, there are two cases:

1. The request has a deadline. In this case, wait until the deadline
   to enqueue the request.
2. The request has no deadline. In this case, let the request block
   indefinitely until it can be enqueued.

### Batch processor/sender

As discussed in the companion RFC, batching processes should
transmit errors, which means waiting for batch responses, and
they should respect cancellation.

This proposal states that batching processes should also use
an outgoing context deadline. Batching logic should keep track
of the maximum deadline over requests with a deadline in each
batch. The batch deadline shall be set to the maximum deadline
of any in the batch. Only when all requests in the batch do not
have a deadline, will the outgoing batch request have no
request deadline.

Under this scheme, requests with no deadline, combined into a batch
of requests that do have a deadline will cause the deadline-free request
to have a deadline. This is intentional. Users are encouraged
to use deadlines always and/or use separate pipelines for requests
with and without deadlines.

### Receiver helper

Receivers are the entry point for data entering a Collector
pipeline. Receivers are responsible for setting the initial
context deadline for each request. This proposal states that
receivers should be encouraged with an opt-in mechanism to
offer standard timeout configuration. The configuration of
this opt-in support follows the Timeout sender:

- `min_timeout` (duration): Limits the allowable timeout for new requests to a minimum value. >=0 means deadline checking.
- `timeout` (duration): Limits the allowable timeout for new requests to a maximum value. Must be >= 0.

The Timeout sender's behavior matrix applies. `min_timeout`
controls whether an arriving timeout, if any, is enforced, and
`timeout` controls whether a timeout is imposed. If `min_timeout`
is negative and `timeout` is zero, the timeout is explicitly
erased from the context.

The default `min_timeout` is zero (i.e., enforce arriving deadlines)
and `timeout` is zero (i.e., allow no-deadline to pass).

## Implementation details

The logic to modify a context given timeout configuration is
shown below. This logic would be used in the exporterhelper
Timeout sender as wel as a new, receiverhelper mechanism that
allowing components to opt-in to this behavior.

```golang
 if ts.cfg.MinTimeout >= 0 {
     if deadline, has := ctx.Deadline(); has {
        if time.Until(deadline) <= ts.cfg.MinTimeout {
            return context.DeadlineExceeded
        }
    }
 } else if ts.cfg.Timeout == 0 {
   // No minimum, no maximum case. Erase incoming request timeout.
   ctx = context.WithoutCancel(ctx)
 }

 if ts.cfg.Timeout != 0 {
   var cancel context.CancelFunc
   ctx, cancel = context.WithTimeout(ctx, ts.cfg.Timeout)
   defer cancel()
 }
 // pass ctx in a synchronous call to the next component
```

## Summary

This proposal introduces several new configuration fields and
behaviors, including:

- Timeout sender: `min_timeout` enforcement
- Queue sender: `fail_fast` option
- Batch processor/sender: cancellation, outgoing deadline
- Receiver helper: `timeout` limit and `min_timeout` enforcement

The example posed in the introduction, aiming to allow an OpenTelemetry
SDK user to control end-to-end timeout behavior, can be configured as
follows:

- The SDK's export `timeout` value is configured by the user
- The OTLP receiver propagates the client's original timeout
- The Batch processor, if configured, propagates a batch timeout
- The Queue sender, if configured, has `fail_fast` false
- The Batch sender, if configured, propagates a batch timeout
- The Retry sender or Timeout component is configured to take
  advantage of the client's timeout, either by allowing retries
  and raising `max_elapsed`, or by raising the maximum `timeout`.

In another example, consider the case where a collector wishes
to explicitly disregard a client-specified timeout. While ignoring
the timeout means potentially breaking error transmission, this
can also ensure successful delivery in cases where the deadline
is unrealistic. For example:

- The SDK's export `timeout` value is set unrealistically low
- The receiver's `min_timeout` is -1, `timeout` is 0, which erases
  the arriving timeout
- The batch processor will propagate no deadline
- The Queue sender, if enabled, has `fail_fast` false
- The Timeout sender sets the final deadline based on its `timeout`.

Taken together, these new features will ensure OpenTelemetry pipelines
can be configured to support a range of timeout behavior while at the
same time recognizing request cancellation to avoid wasteful
computation.
