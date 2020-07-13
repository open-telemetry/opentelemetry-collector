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

Refer to [config.yaml](./testdata/config.yaml) for detailed
examples on using the processor.
