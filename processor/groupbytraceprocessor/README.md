# Group by Trace processor

Supported pipeline types: traces
Status: in development

This processor collects all the spans from the same trace, waiting a 
pre-determined amount of time before releasing the trace to the next processor.
The expectation is that, generally, traces will be complete after the given time.

This processor should be used whenever a processor requires grouped traces to make decisions,
such as a tail-based sampler or a per-trace metrics processor.

The batch processor shouldn't be used before this processor, as this one will 
probably undo part (or much) of the work that the batch processor performs. It's
fine to have the batch processor to run right after this one, and every entry in the
batch will be a complete trace.

Please refer to [config.go](./config.go) for the config spec.

Examples:

```yaml
processors:
  groupbytrace:
  groupbytrace/2:
    wait_duration: 10s
    num_traces: 1000
```

## Configuration

Refer to [config.yaml](./testdata/config.yaml) for detailed examples on using the processor.

The `num_traces` property tells the processor what's the maximum number of traces to keep in the internal storage. A higher `num_traces` might incur in a higher memory usage.

The `wait_duration` property tells the processor for how long it should keep traces in the internal storage. Once a trace is kept for this duration, it's then released to the next consumer and removed from the internal storage. Spans from a trace that has been released will be kept for the entire duration again.

## Metrics

The following metrics are recorded by this processor:

* `otelcol_processor_groupbytrace_conf_num_traces` represents the maximum number of traces that can be kept by the internal storage. This value comes from the processor's configuration and will never change over the lifecycle of the processor.
* `otelcol_processor_groupbytrace_event_latency_bucket`, with the following `event` tag values:
  * `onBatchReceived` represents the number of batches the processor has received from the previous components
  * `onTraceExpired` represents the number of traces that finished waiting in memory for spans to arrive
  * `onTraceReleased` represents the number of traces that have been marked as released to the next component
  * `onTraceRemoved` represents the number of traces that have been marked for removal from the internal storage
* `otelcol_processor_groupbytrace_num_events_in_queue` representing the state of the internal queue. Ideally, this number would be close to zero, but might have temporary spikes if the storage is slow.
* `otelcol_processor_groupbytrace_num_traces_in_memory` representing the state of the internal trace storage, waiting for spans to arrive. It's common to have items in memory all the time if the processor has a continuous flow of data. The longer the `wait_duration`, the higher the amount of traces in memory should be, given enough traffic.
* `otelcol_processor_groupbytrace_spans_released` and `otelcol_processor_groupbytrace_traces_released` represent the number of spans and traces effectively released to the next component.
* `otelcol_processor_groupbytrace_traces_evicted` represents the number of traces that have been evicted from the internal storage due to capacity problems. Ideally, this should be zero, or very close to zero at all times. If you keep getting items evicted, increase the `num_traces`.
* `otelcol_processor_groupbytrace_incomplete_releases` represents the traces that have been marked as expired, but had been previously been removed. This might be the case when a span from a trace has been received in a batch while the trace existed in the in-memory storage, but has since been released/removed before the span could be added to the trace. This should always be very close to 0, and a high value might indicate a software bug.

A healthy system would have the same value for the metric `otelcol_processor_groupbytrace_spans_released` and for three events under `otelcol_processor_groupbytrace_event_latency_bucket`: `onTraceExpired`, `onTraceRemoved` and `onTraceReleased`.

The metric `otelcol_processor_groupbytrace_event_latency_bucket` is a bucket and shows how long each event took to be processed in miliseconds. In most cases, it should take less than 5ms for an event to be processed, but it might be the case where an event could take 10ms. Higher latencies are possible, but it should never really reach the last item, representing 1s. Events taking more than 1s are killed automatically, and if you have multiple items in this bucket, it might indicate a bug in the software.

Most metrics are updated when the events occur, except for the following ones, which are updated periodically:
* `otelcol_processor_groupbytrace_num_events_in_queue`
* `otelcol_processor_groupbytrace_num_traces_in_memory`
