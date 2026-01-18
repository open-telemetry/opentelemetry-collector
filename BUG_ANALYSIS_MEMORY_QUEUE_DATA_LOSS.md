# Critical Bug: Memory Queue Silent Data Loss During Graceful Shutdown

## Summary

The memory queue silently drops telemetry data when export fails during graceful shutdown due to a missing shutdown error check that the persistent queue correctly implements.

## Affected Components

- **Component type:** Core / Exporter Helper
- **Package and file paths:**
  - `exporter/exporterhelper/internal/queue/memory_queue.go`
  - `exporter/exporterhelper/internal/base_exporter.go`
  - `exporter/exporterhelper/internal/retry_sender.go`
  - `exporter/exporterhelper/internal/queue_sender.go`
- **Function or method names:**
  - `memoryQueue.onDone()` - lines 161-172
  - `BaseExporter.Shutdown()` - lines 135-150
  - `retrySender.Send()` - lines 141-148

## Exact Problem

### Code Path During Shutdown

1. `BaseExporter.Shutdown()` (base_exporter.go:139-141) shuts down the retry sender **first**:

```go
// First shutdown the retry sender, so the queue sender can flush the queue without retries.
if be.RetrySender != nil {
    err = multierr.Append(err, be.RetrySender.Shutdown(ctx))  // Closes stopCh
}
```

2. When the queue drains and any export fails, `retrySender.Send()` (retry_sender.go:144-145) immediately returns a shutdown error:

```go
case <-rs.stopCh:
    return experr.NewShutdownErr(err)  // stopCh already closed!
```

3. The error propagates to `memoryQueue.onDone()` (memory_queue.go:161-172):

```go
func (mq *memoryQueue[T]) onDone(bd *blockingDone, err error) {
    mq.mu.Lock()
    defer mq.mu.Unlock()
    mq.size -= bd.elSize
    mq.hasMoreSpace.Signal()
    if mq.waitForResult {
        bd.ch <- err
        return
    }
    blockingDonePool.Put(bd)  // ERROR IS COMPLETELY IGNORED!
}
```

When `waitForResult=false` (the **default**), the `err` parameter is never checked. Data is removed from the queue regardless of export success or failure.

### Comparison with Persistent Queue

The persistent queue correctly handles this (persistent_queue.go:405-408):

```go
if experr.IsShutdownErr(consumeErr) {
    // The queue is shutting down, don't mark the item as dispatched,
    // so it's picked up again after restart.
    return
}
```

The memory queue has **no equivalent check**, creating an inconsistency:
- **Persistent queue:** shutdown failures -> data preserved for restart
- **Memory queue:** shutdown failures -> **data silently lost**

## Why This Is Critical

### Telemetry Affected
All signal types (traces, metrics, logs) when using the default memory queue configuration (`waitForResult=false`, no `storage:` configured).

### Who Is Impacted
- Production users performing graceful shutdowns (rolling deployments, pod scaling)
- Any deployment where transient network issues occur during shutdown
- Users who configure `retry_on_failure: enabled` expecting data protection

### Why Difficult to Detect
- Error is logged at `queue_sender.go:50-51` as "Exporting failed. Dropping data." but then silently discarded
- Users expect `retry_on_failure` to protect against transient failures, but retries are disabled during shutdown
- Persistent queue handles this correctly, creating false sense of coverage
- No metric emitted for "data lost during shutdown"

### Realistic Scenarios
1. **Kubernetes rolling deployment:** Pod receives SIGTERM, begins graceful shutdown. Brief network hiccup occurs. All queued telemetry is silently dropped.
2. **Cloud autoscaling:** Instance terminates during high load. Backend returns HTTP 503 temporarily. Thousands of queued spans/metrics are lost.
3. **Shutdown under load:** Any transient failure during drain = permanent data loss.

## Suggested Fix

Add shutdown error handling in `memoryQueue.onDone()` similar to persistent queue, or at minimum emit a clear metric/log indicating data was lost specifically due to shutdown timing.
