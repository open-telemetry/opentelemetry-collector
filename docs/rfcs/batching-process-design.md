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
Both are core components, and it is an explicit goal that these two
options for batching cover a wide range of requirements.
We prefer to improve the core batching-process components instead
of introduce new components, in order to achieve desired batching 
behavior.

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

### Sequence of Events

A batching process breaks the normal flow of Context through a
Collector pipeline.  Here are the events that take place as a request
makes its way through a batching process:

A. The request arrives when the caller calls a `Consume()` or `Send()`
   method on this component with Context and data.
B. The request is added to the currently-pending batch.
C. The batching process calls export one or more times containing data from the original request.
D. The batching process receives the response from the export(s), possibly with an error.
E. The `Consume()` or `Send()` call returns control to the caller, possibly with an error.

The two batch processors execute these steps with different sequences.

In the batch processor, we observe two independent sequences.  One is
A ⮕ B ⮕ E: the request arrives, then is placed in a batch, then
returns success.  The other is B ⮕ C ⮕ D: once in a batch, the request
is exported, then errors (if any) are logged.

In the batch_sender, we observe a single sequence, A ⮕ B ⮕ C ⮕ D ⮕ E:
the request arrives, is placed in a batch, the batch is sent, the
response is received, and the caller returns the error (if any).

To resolve the inconsistency, this document proposes to modify the
batch processor to use a single sequence, i.e., A ⮕ B ⮕ C ⮕ D ⮕ E.

### Request handling

There are a number of open questions related to the arriving request
and its context.

- The arriving request has no items of telemetry.  Does the batching process return success immediately?

Consider the arriving deadline:

- The arriving Context deadline has already expired.  Does the batching process fail the request immediately?
- The arriving Context deadline expires while waiting for the export(s).  What happens?

Considering the arriving Context's trace context:

- An export contains data from multiple requests, is a new root span instrumented?
- An export contains data from a single request, is a child span instrumented?

The batching process may be configured to use client metadata as a
batching identifier
([batchprocessor](https://github.com/open-telemetry/opentelemetry-collector/issues/4544)
is complete,
[batch_sender](https://github.com/open-telemetry/opentelemetry-collector/issues/10825)
is incomplete).  Considering the arriving Context's client metadata:

- An export contains data from a single request, are there circumstances when the request's client metadata passes through?

### Error handling

A batching process determines what happens when some or all of a
request fails to be processed.  Consider when an incoming request has
partially or completely failed:

- Does the caller receive an error?
- Does the remaining portion of the request still export?
- Under what conditions is the error returned by a batching process retryable?

### Concurrency handling

Here are some questions about concurrency in the batching process.
Consider what happens when there is more than one batch of data
available to send:

- Does the batching process wait for one batch to complete before sending another?
- Does the batching process use a caller's goroutine to export, or can it create its own?
- Is there any limit on the number of concurrent exports?

## Proposed Requirements

The questions posed above are meant to help us identify areas where
the two batching processes are either inconsisent with each other or
inconsistent with the goals of the project.

