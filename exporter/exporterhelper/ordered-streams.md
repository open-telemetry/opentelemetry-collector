# Ordered streams with deferred completion

`xexporterhelper.NewLogsRequests` is an experimental logs exporter constructor for
protocols whose ordered write phase finishes before their acknowledgement arrives.
The helper owns queue admission, retries, persistence, and request retirement.

## Conversion and admission

The converter returns a slice of `Descriptor[plog.Logs]`. Each descriptor contains
one request, its partition key, and either `PositionContinue` or `PositionEnd`.
Descriptors from one conversion enter the queue together or admission fails.
Request sizing charges every child of this envelope. Item and byte sizing charge
the complete envelope.

Producers must submit each logical stream serially using the same partition key.
The helper preserves queue insertion order within that partition. Independent
partitions can write concurrently. `PositionEnd` establishes a stream boundary;
the next request may use a new stream attempt after that boundary is released.
The helper does not combine request bodies or infer boundaries from their data.

## Write and completion contract

The sender receives a `Dispatch` and a `Completion` handle for each request.

1. Write the request, then call `Release` to allow the next request in its partition
   to start. An acknowledgement need not have arrived yet.
2. Call `Succeed` when the protocol acknowledges the request, or `Fail` when its
   outcome fails. A returned sender error also reports failure. `Succeed` implies
   release if it has not already happened.
3. Stop accessing the request when the sender function returns. Retain the
   completion handle, rather than pdata, for a later acknowledgement callback.

Repeated completion calls are harmless. Failure fences the stream attempt, so
callbacks from that attempt cannot retire requests sent by a later attempt.
`StreamAttempt` identifies the attempt, and `StreamContext` is canceled when it
ends. A sender can use these to manage connection leases. `Recovery` identifies
the retained recovery request that must precede unresolved requests on retry.

## Queue and resource settings

Pass the usual exporterhelper queue, timeout, and retry options. Batching must be
disabled. Queue consumption uses one FIFO reader; `MaxConcurrentWrites` in
`OrderedStreamSettings` bounds independent partition writes and defaults to one
when nonpositive. Set it from the exporter queue configuration when
`num_consumers` should control this concurrency.

The remaining settings bound staged requests, active partitions, released
requests and bytes, recovery-tail bytes, conversion envelopes, and partition-key
length. These limits must be positive and satisfy `Validate`.

With persistent storage, the helper records child retirement and one recovery
tail per open partition in the queue's storage namespace. Restart replays older
dispatched requests before newly queued work. Checkpoint updates use a redo
journal because storage batches need not be transactional. A storage failure
fences recovery until restart. Memory queues retain recovery state in memory.
Delivery is at least once; an ambiguous write can cause duplicates. This
constructor enables ordered restart replay. Other exporters retain their
existing recovery path by default and can opt into original-index replay with
`xexporterhelper.QueueBatchSettings.ReplayInOrder`. That setting controls queue
recovery order; the ordered constructor also schedules partition writes.
