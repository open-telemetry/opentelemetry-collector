# OpenTelemetry Collector Batch Processor Deadlock Investigation

## Problem Statement

The OpenTelemetry Collector batch processor experiences a deadlock when the following conditions are met:
1. The `useExporterHelper` feature gate is enabled
2. The `propagateErrors` feature gate is enabled 
3. The batch processor uses `WaitForResult=true` (to propagate errors back to the receiver)
4. Multiple concurrent requests are processed simultaneously

This deadlock manifests as the collector hanging indefinitely, with goroutines blocked waiting for completion signals that never arrive.

## Technical Context

### Components Involved
- **Batch Processor**: Buffers telemetry data before sending to exporters
- **Exporter Helper**: Provides common exporter functionality including retry, queue, and batching
- **Queue Batch**: Internal batching mechanism within exporter helper
- **Partition Batcher**: Manages request batching and worker pool coordination
- **Memory Queue**: Handles request queuing and completion tracking
- **Worker Pool**: Executes export operations with limited concurrency

### Feature Gates Impact
- `useExporterHelper`: Enables new batching logic in exporter helper instead of batch processor
- `propagateErrors`: Enables error propagation from exporters back to receivers via `WaitForResult`

## Root Cause Analysis

### Investigation Timeline

1. **Initial Hypothesis - MergeSplitErr Bug**: Suspected bug in error merging logic
   - **Status**: RULED OUT - No requests go through this path in our test cases

2. **Reference Counting Bug**: Suspected incorrect OnDone call counting in partition batcher
   - **Status**: RULED OUT - Logging confirms all OnDone calls are made correctly

3. **Worker Pool Token Management**: Identified bug in `workerPool.execute()`
   - **Status**: FIXED - Token was returned before work completed, causing premature worker availability
   - **Impact**: Partial fix - Simple tests now pass, but stress tests still deadlock

4. **Current Focus**: Complex coordination issue under high concurrent load

### Confirmed Facts

✅ **Working Correctly**:
- Reference counting in partition batcher
- OnDone callback execution 
- Basic worker pool operation (simple test passes)
- Error propagation mechanism
- waitingDone channel communication

❌ **Still Problematic**:
- High concurrency scenarios with many requests
- Stress tests with >100 concurrent requests
- Complex batching scenarios with worker pool exhaustion

### Technical Details

#### Worker Pool Fix Applied
```go
// BEFORE (buggy)
func (wp *workerPool) execute(fn func()) {
    <-wp.tokens  // Acquire token
    defer func() { wp.tokens <- struct{}{} }()  // Return token immediately
    go fn()      // Start work asynchronously - token returned before work completes!
}

// AFTER (fixed)
func (wp *workerPool) execute(fn func()) {
    <-wp.tokens  // Acquire token
    go func() {
        defer func() { wp.tokens <- struct{}{} }()  // Return token after work completes
        fn()     // Do the work
    }()
}
```

#### Test Results Summary
- **Simple Test (10 requests, sequential)**: ✅ PASSES
- **Stress Test (1000 requests, concurrent)**: ❌ DEADLOCKS
- **Race Detector**: No race conditions detected, but deadlock persists

## Current Hypotheses

### Hypothesis 1: Coordination Timing Issue
The deadlock may occur due to subtle timing issues in the coordination between:
- Batch flushing triggers
- Worker pool token management
- waitingDone completion signals
- Request queuing/dequeuing

### Hypothesis 2: Resource Exhaustion Pattern
Under high load, the following sequence may cause deadlock:
1. All worker pool tokens are consumed
2. Batched requests are waiting for workers
3. New requests keep arriving and queuing
4. Some completion signal gets lost or delayed
5. System reaches a state where no progress can be made

### Hypothesis 3: Channel/Goroutine Leak
There may be goroutines or channels that aren't being properly cleaned up under high concurrency, leading to resource exhaustion.

## Next Steps

### Immediate Actions
1. **Reduce Test Complexity**: Create intermediate stress tests between simple (10 requests) and full stress (1000 requests) to find the breaking point
2. **Add More Instrumentation**: Add detailed logging to worker pool token acquisition/release
3. **Profile Resource Usage**: Use `go tool pprof` to analyze goroutine and memory usage during deadlock
4. **Timeout Analysis**: Add timeouts at different levels to identify where the system gets stuck

### Diagnostic Commands
```bash
# Run with goroutine profiling
go test -v -run "TestPartitionBatcherWaitForResultDeadlock" -cpuprofile cpu.prof -memprofile mem.prof

# Analyze goroutine dump during deadlock
kill -SIGABRT <process_id>  # While test is hanging
```

### Potential Solutions
1. **Improve Worker Pool**: Add better token management or increase pool size dynamically
2. **Add Circuit Breaker**: Prevent system overload by rejecting requests when resources are exhausted
3. **Optimize Batching Logic**: Reduce coordination complexity between components
4. **Add Deadlock Detection**: Implement timeouts and recovery mechanisms

## Risk Assessment

- **High**: Production systems using both feature gates will experience complete hangs
- **Medium**: Performance degradation even when not deadlocking due to coordination overhead
- **Low**: Memory leaks from goroutine accumulation

## Success Criteria

A successful fix should:
1. Pass all existing unit tests
2. Handle 1000+ concurrent requests without deadlocking
3. Maintain error propagation functionality
4. Show no resource leaks under stress testing
5. Pass with `-race` detector enabled
