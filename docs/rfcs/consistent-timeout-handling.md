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

- the "timeout" of a context is the duration until its deadline. An expired deadline
  has no timeout, whereas a viable deadline has not yet expired. A context can have
  no deadline, and thus no timeout.
- the "timeout" of a configuration is the duration added relative to current time
  until its deadline. No timeout means no deadline.

Here are some of the scenarios that need to have well-defined behavior:

- A request arrives with an already-expired deadline. Should the component
  immediately return a deadline-exceeded error status?
- A request arrives with a viable deadline, but the request does not
  succeed in time. Should the component immediately return a deadline-exceeded
  status, or should it wait for a response?
- A request arrives and needs to obtain a resource (e.g., space in a queue) that
  is not immediately available. Should the component fail "fast" or stall the
  request, hoping the resource will become available before the deadline?
- A request arrives and the component calls for an additive timeout that is
  greater than the request's deadline, for example:
  - batch processor adds 1s of potential batching delay, arriving request has 0.5s deadline
  - timeout sender has 5s configured timeout, arriving request has 2s timeout
  - retry sender has a maximum elapsed timeout of 1 minute, and the arriving request has 5s timeout.

## Explanation

Scope:

- Batch processor
- Batch sender
- Timeout sender
- Retry sender
- Queue sender

## Existing behavior

Note that the Golang context mechanism will not accidentally lengthen a deadline.

### Feature matrix sender

- Overreide
- Sustain
- Abort
- Passthrough
- Batch-by-timeout
- Minimum timeout for service?
- Retry sender: do not wait for cancellation
...

## Specific proposals

### Timeout sender

### Retry sender

### Queue sender

Respect timeout.

### Batch processor/sender

Neither existing component uses an outgoing context deadline.
This could lead to resource exhaustion, in some cases, by
allowing requests to remain pending indefinitely.

On the one hand, this support may not be necessary, since in
most cases the batching process is followed by an exporter, which
includes a timeout sender option, capable of ensuring a default
timeout.

On the other hand, the batching process knows the callers'
actual deadlines, and it could even use this information to
form batches.
