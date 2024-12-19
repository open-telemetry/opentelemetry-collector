# Error transmission through a batching processor with concurrency

Establish normative guidelines for components that batch telemetry to follow
so that the batchprocessor and exporterhelper-based batch_sender have similar behavior.

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

1. The request arrives when the preceding component calls `Consume()`
   on this component with Context and data.
1. The request is placed into a channel.
1. The request is removed from a channel by a background thread and
   entered into the pending state.
1. The batching process calls export one or more times containing data
   from the original request, receiving responses possibly with errors
   which are logged.
1. The `Consume()` call returns control to the caller.

In the batch processor, the request producer performs step 1 and 2,
and then it skips to 5, returning success before the export completes. We refer to this behavior as
"error suppression". The background thread, independently, observes steps
3 and 4, after which any errors are logged.

The batch procsesor performs steps 4 multiple times in sequence, with
never more than one export at a time. Effective concurrency is limited to
1 within the component, however this is usually alleviated by the use of
a Queue sender, later in the pipeline. When the exporterhelper's Queue sender
is enabled and the queue has space, it immediately returns success (a
form of error suppression), which allows the batch processor to issue multiple
batches at a time into the queue.

The batch processor does not consult the incoming request's Context deadline
or allow request context cancellation to interrupt step 2. Step 4 is executed
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

1. Check if there is a pending batch.
1. If there is no pending batch, it creates a new one and starts a timer.
1. Add the incoming data to the pending batch.
1. Send the batch to the exporter.
1. Wait for the batch error.
1. Each caller returns the batch error.

Unlike the batch processor, errors are transmitted back to the callers, not suppressed.

Trace context is interrupted. Outgoing requests have empty `client.Metadata`.

The context deadline of the caller is not considered in step 5. In step 4, the
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
steps 4 and 5 are executed repeatedly while there are more requests, meaning:

1. Exports are synchronous and sequential.
2. An error causes aborting of subsequent parts of the request.

### Queue sender logic: existing

The queue sender provides key functionality that determines the overall behavior
of both batching components. When enabled, the queue sender will return success
to the caller as soon as the request is enqueued. In the background, it concurrently
exports requests in the queue using a configurable number of execution threads.

It is worth evaluating the behavior of the queue sender with a persistent queue
and with an in-memory queue:

- Persistent queue: In this case, the queue stores the request before returning
  success. This component is not directly responsible for data loss.
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
| Concurrency enabled | No | Merge: Yes<br> MergeSplit: No | Does the component allow concurrent export? |
| Independence | Yes | No | Are all data exported independently? |

### Change proposal

#### Queue sender defaults

A central aspect of this presentation focuses on the queue sender,
which along with one of the batching processors defines the behavior
of most OpenTelemetry pipelines. Therefore, the default queue sender
behavior is critical.

This proposal argues in favor of the user, who does not want default
behavior that leads to data loss. This requires one or more changes
in the exporterhelper:

1. Disable the queue sender by default. In this configuration, requests
   become synchronous through the batch processor, and responses are delayed
   until whole batches covering the caller's input have been processed.
   Assuming the other requirements in this proposal are also carried out,
   this means that pipelines will block each caller awaiting end-to-end
   success, by default.
2. Prevent start-up without a persistent queue configured; users would
   have to opt-in to the in-memory queue if they want to return success
   with no persistent store and not await end-to-end success.

#### Batch processor: required

The batch processor MUST be modified to achieve the following
outcomes:

- Allow concurrent exports. When the processor has a complete batch of
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

Should outgoing requests be instrumented by a trace Span linked
to the incoming trace contexts? This document proposes yes, as
follows.

1. Create a new root span at the moment each batch starts.
2. For each new request incorporated into the batch, call
   `AddLink()` on the pair of spans.
3. Use the associated root span as the context for each
   export call.

#### Empty request handling

How should a batching process handle requests that contain no
concrete items of data? [These requests may be seen as empty
containers](https://github.com/open-telemetry/opentelemetry-proto/issues/598),
for example, tracing requests with no spans, metric requests with
no metric data points, and logs requests with no log records.

For a batching process, these requests can be problematic.  If
request size is measured in item count, these "empty" requests
leave batch size unchanged, therefore they can cause unbounded
memory growth.

This document proposes a consistent treatment for empty requests:
batching processes should return immediate success, which is the
current behavior of the batch processor.

#### Outgoing context deadline

Should the batching process set an outgoing context deadline
to convey the maximum amount of time to consider processing
the request?

This and several related questions are broken out into a
[companion RFC](https://github.com/open-telemetry/opentelemetry-collector/pull/11948).

#### Prototypes

##### Concurrent batch processor

The OpenTelemetry Protocol with Apache Arrow project's [`concurrentbatch`
processor](https://github.com/open-telemetry/otel-arrow/blob/main/collector/processor/concurrentbatchprocessor/README.md)
is derived from the core batch processor. It has added solutions for
the problems outlined above, including error propagation, trace
propagation, and concurrency.

This code can be contributed back to the core with a series of minor
changes, some having an associated feature gate.

1. Add tracing support, as described above.
1. Make "early return" a new feature gate from on (current behavior) to off (desired behavior); when early return is enabled, suppress errors and return; otherwise, wait for the response and return the error.
1. Make "concurrency_limit" a new setting measuring concurrency added by this component, feature gate from 1 (current behavior) to limited (e.g., 10, 100)

##### Batch sender

This has not been prototyped. The exporterhelper code can be modified,
for the batch sender to conform with this proposal.

1. Add tracing support, as described above.
1. Make "concurrency_limit" a new setting measuring concurrency added by this component, feature gate from 0 (current behavior) to limited (e.g., 10, 100)
1. Add metadata keys support, identical to the batch processor.
