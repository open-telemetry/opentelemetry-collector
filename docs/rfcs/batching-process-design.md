# Error transmission through a batching processor with concurrency

Establish normative guidelines for components that batch telemetry to follow so that the batchprocessor and exporterhelper-based batch_sender behave in similar ways with good defaults.

## Motivation

We are motivated, first, to have consistent behavior across two forms of batching process: (1) `batchprocessor`, and (2) `exporterhelper/internal/batch_sender`.  Today, these two core components exhibit diferent behaviors.

Second, to establish conditions and requirements for error transmission, tracing instrumentation, and concurrency from these components.

Third, to establish how batching processors should handle incoming context deadlines.

## Explanation

Here, "batching process" refers to specifically the batch processor
(i.e., a processor component) or the exporterhelper's batch_sender
(i.e., part of an exporter component).
Both are core componentry, and it is an explicit goal that these two
options for batching cover a wide range of requirements.

We use the term "request" as a signal-independent descriptor of the
concrete data types used in Collector APIs (e.g., a `plog.Logs`,
`ptrace.Traces`, `pmetric.Metrics`).

We use the term "export" generically to refer to the next action taken
by a component, which is to call either `Consume()` or `Send()` on the
next component or sender in the pipeline.

The primary goal of a batching process is to combine and split
requests.  Small requests and large requests go in to a batching
process, and moderately-sized requests come out.  Batching processes
can be used to amortize overhead costs associated with small requests,
and batching processes can be used to prevent requests from exceeding
request-size limits later in a pipeline.

## Detailed design

### Batching process configuration

A batching process is usually configured with a timeout, which limits
individual requests from waiting too long for the arrival of a
sufficiently-large batch.  At a high-level, the typical configuration
parameters of a batching process are:

1. Minimum acceptable size (required)
2. Maximum acceptable size (optional)
3. Timeout (optional)

Today, both batching processes restrict size configuration to an item
count, [however the desire to use request count and byte count (in
some encoding) is well recognized](https://github.com/open-telemetry/opentelemetry-collector/issues/9462).

### Batch processor logic: existing

The batch processor operates over Pipeline data for both its input
and output. It takes advantage of the top-level `repeated` portion
of the OpenTelemetry data model to combine many requests into one
request.  Here is the logical sequence of events that
takes place as a request makes its way through a batch processor:

A. The request arrives when the preceding component calls `Consume()`
   on this component with Context and data.
B. The request is placed into a channel.
C. The request is removed from a channel by a background thread and
   entered into the pending state.
D. The batching process calls export one or more times containing data
   from the original request, receiving responses possibly with errors
   which are logged.
E. The `Consume()` call returns control to the caller.

In the batch processor, the request producer performs step A and B,
and then it skips to F. Since the processor returns before the export
completes, it always returns success. We refer to this behavior as
"error suppression". The background thread, independently, observes steps
C, D, and E, after which errors (if any) are logged.

The batch procsesor performs steps D and E multiple times in sequence, with
never more than one export at a time. Effective concurrency is limited to
1 within the component, however this is usually alleviated by the use of
a "queue_sender" later in the pipeline. When the exporterhelper's queue sender
is enabled and the queue has space, it immediately returns success (a
form of error suppression), which allows the batch processor to issue multiple
batches at a time.

The batch processor does not consult the incoming request's Context deadline
or allow request context cancellation to interrupt step B. Step D is executed
without a context deadline.

Trace context is interrupted. By default, incoming gRPC metadata is not propagated.
To address the loss of metadata, the batch processor has been extended with a
`metadata_keys` configuration; with this option, independent batch processes
are constructed for each distinct combination of metadata key values.  Per-group
metadata values are placed into the outgoing context.

### Batch sender logic: existing

The batch sender is a feature of the exporterhelper; it is an optional
sub-component situated after the queue sender, and it is used to compute
batches in the intended encoding used by the exporter. It follows a different
sequence of events compared to the processor:

A. Check if there is a pending batch.
B. If there is no pending batch, it creates a new one and starts a timer.
C. Add the incoming data to the pending batch.
D. Send the batch to the exporter.
E. Wait for the batch-error.
F. Each caller returns the batch-error.

Unlike the batch processor, errors are propagated, not suppressed.

Trace context is interrupted. Outgoing requests have empty `client.Metadata`.

The context deadline of the caller is not considered in step E. In step D, the
export is made without a context deadline; a subsequent timeout sender typically
configures a timeout for the export.

The pending batch is managed through a Golang interface, making it possible
to accumulate protocol-specific intermediate data. There are two specific
interfaces an exporter component provides:

- Merge(): when request batches have no upper bound. In this case, the interface
  produces single outputs.
- MergeSplit(): when there is a maximum size imposed. In this case, the interface
  produces potentially more than one output request.

Concurrency behavior varies. In the case where `MergeSplit()` is used, there is
a potential for multiple requests to emit from a single request. In this case,
steps D through F are executed repeatedly while there are more requests, meaning:

1. Exports are synchronous and sequential.
2. An error causes aborting of subsequent parts of the request.

### Queue sender logic: existing

The queue sender provides key functionality that determines the overall behavior
of both batching components. When enabled, the queue sender will return success
to the caller as soon as the request is enqueued. In the background, it concurrently
exports requests in the queue using a configurable number of threads.

It is worth evaluating the behavior of the queue sender with a persistent queue
and with an in-memory queue:

- Persistent queue: In this case, the queue stores the request before returning
  success. There is not a chance of data loss.
- In-memory queue: In this case, the queue acts as a form of error suppression.
  Callers do not wait for the export to return, so there is a chance of data loss
  in this configuration.

The queue sender does not consider the caller's context deadline when it attempts
to enqueue the request. If the queue is full, the queue sender returns a queue-full
error immediately.

### Feature matrix

| Support area | Batch processor | Batch sender | Explanation |
| -----| -- |  -- | -- |
| Merges requests | Yes | Yes | Does it merge smaller into larger batches? |
| Splits requests | Yes | Yes, however sequential | Does it split larger into smaller batches? |
| Cancellation  | No | No | Does it respect caller cancellation? |
| Deadline  | No | No | Does it set an outgoing deadline? |
| Metadata  | Yes | No | Can it batch by metadata key value(s)? |
| Tracing  | No | No | Instrumented for tracing? |
| Error transmission | No | Yes | Are export errors returned to callers? |
| Concurrency allowed | No | Yes | Does the component limit concurrency? |
| Independence | Yes | No | Are all data exported independently? |

### Change proposal

#### Batch processor: required

The batch processor MUST be modified to achieve the following
outcomes:

- Allow concurrent exports. When the processor has a batch of
  data available to send, it will send the data immediately.
- Transmit errors back to callers. Callers will be blocked
  while one or more requests are issued and wait on responses.
- Respect context cancellation. Callers will return control
  to the pipeline when their context is cancelled.

#### Batch sender: required

The batch sender MUST be modified to achieve the following
outcomes:

- Allow concurrent splitting. When multiple full-size requests
  are produced from an input, they are exported concurrently and
  independently.
- Recognize partial splitting. When a request is split, leaving
  part of a request that is not full, it remains in the active
  request (i.e., still pending).
- Respect key metadata. Implement the `metadata_keys` feature
  supported by the batch processor.
- Respect context cancellation. Callers will return control
  to the pipeline when their context is cancelled.

### Open questions

#### Batch trace context

Should outgoing requests be instrumented by a trace Span linked to the incoming trace contexts? This document proposes yes, in
one of two ways:

1. When an export corresponds with data for a single incoming
   request, the request's original context is used as the parent.
2. When an export corresponds with data from multiple incoming
   requests, the incoming trace contexts are linked with the new
   root span.

#### Empty request handling

How should a batching process handle requests that contain no
concrete items of data? [These requests may be seen as empty
containers](https://github.com/open-telemetry/opentelemetry-proto/issues/598),
for example, tracing requests with no spans, metric requests with
no metric data points, and logs requests with no log records.

For a batching process, these requests can be problematic.  If
request size is measured in item count, these "empty" requests
leave batch size unchanged, and could cause unbounded memory
growth.

This document proposes a consistent treatment for empty requests:
batching processes should return immediate success, which is the
behavior of the batching processor currently.

#### Outgoing context deadline

Should the batching process set an outgoing context deadline
to convey the maximum amount of time to consider processing
the request?

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

This proposal makes no specific recommendation.
