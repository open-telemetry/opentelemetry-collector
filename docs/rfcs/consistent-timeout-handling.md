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

Here are some of the scenarios where consistent behavior is desireable:

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
- Timeout sender configured not to lower already-set timeout.

## Explanation

The questions listed in the motivation above correspond with common
questions faced by the core sub-components.

- Batch processor
- Batch sender (exporterbatcher)
- Timeout sender (exporterhelper)
- Retry sender (exporterhelper)
- Queue sender (exporterhelper)

There appears to several patterns, in general:

1. The component synchronously waits for another process, subject to a timeout in the request context;
2. The component synchronously calls another process, while subject
   to a timeout in the request context;
3. The component is configured with a timeout, while subject to
   a shorter timeout.

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
as discussed in a [companion RFC](). As stated in that proposal,
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
into the queue? If no deadline is set, is there be an option to
wait indefinitely?

### Receiver support

Especially when gRPC is in use, requests typically arrive with a pre-set
deadline. In a collector-to-collector pipeline scenario, the receiver
will set a request deadline corresponding with the deadline set in the
exporter's Timeout sender.

If the timeout is especially small, can it be set to a minimum value?
If the timeout is especially large or unset, can it set to a maximum value?

## Specific proposal

### Timeout sender

We propose additional fields in the configuration:

- `min_timeout` (duration): When an arriving request has a timeout less
  than this value, the request immediately fails with a deadline-exceeded
  error. Invalid if set greater than the `timeout` field.
- `max_timeout` (duration): Limits the allowable timeout for exports.
- `ignore_timeout` (bool): When true, the incoming timeout is erased,
  results in `timeout` taking effect unconditionally.

### Retry sender

A new field, consistent with the timeout sender:

- `ignore_timeout` (bool): When true, the incoming timeout is erased,
  results in the `max_elapsed` and the subsequent Timeout sender's
  `timeout` taking effect unconditionally.

### Queue sender

A new field, consistent with the timeout sender:

- `ignore_timeout` (bool): When true, the incoming timeout is erased,
  results in fail-fast semantics. The caller will not block awaiting
  queue space.

When there is a timeout, block until queue space is available.

### Batch processor/sender

Respect cancellation.

Use outgoing context deadline equal to the maximum deadline
in the batch.

### Receiver helper

New configuration fields, matching the Timeout sender:

- `min_timeout` (duration): When an arriving request has a timeout less
  than this value, the request immediately fails with a deadline-exceeded
  error.
- `max_timeout` (duration): Limits the allowable timeout for new requests.
- `ignore_timeout` (bool): Erase the arriving request context timeout,
  apply `max_timeout` instead.
