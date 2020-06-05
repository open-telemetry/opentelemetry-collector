# Batch Processor

Supported pipeline types: metric, traces

The batch processor accepts spans or metrics and places them into batches.
Batching helps better compress the data and reduce the number of outgoing 
connections required to transmit the data. This processor supports both size and
time based batching.

It is highly recommended to configure the batch processor on every collector.
The batch processor should be defined in the pipeline after the `memory_limiter`
as well as any sampling processors. This is because batching should happen after
any data drops such as sampling.

Please refer to [config.go](./config.go) for the config spec.

The following configuration options can be modified:
- `send_batch_size` (default = 8192): Number of spans or metrics after which a
batch will be sent.
- `timeout` (default = 200ms): Time duration after which a batch will be sent
regardless of size.

Examples:

```yaml
processors:
  batch:
  batch/2:
    send_batch_size: 10000
    timeout: 10s
```

Refer to [config.yaml](./testdata/config.yaml) for detailed
examples on using the processor.
