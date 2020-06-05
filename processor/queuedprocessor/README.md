# Queued Retry Processor

Supported pipeline types: traces

The queued retry processor uses a bounded queue to relay batches from the receiver
or previous processor to the next processor. Received data is enqueued immediately
if the queue is not full. At the same time, the processor has one or more workers
which consume the data in the queue by sending them to the next processor or exporter.
If relaying the data to the next processor or exporter in the pipeline fails, the
processor retries after some backoff delay depending on the configuration.

Some important things to keep in mind with the queued_retry processor:
- Given that if the queue is full the data will be dropped, it is important to size
the queue appropriately.
- Since the queue is based on batches and batch sizes are environment specific, it may
not be easy to understand how much memory a queue will consume.
- Finally, the queue size is dependent on the deployment model of the collector. The
agent deployment model typically has a smaller queue than the collector deployment
model.

It is highly recommended to configure the queued_retry processor on every collector
as it minimizes the likelihood of data being dropped due to delays in processing or
issues exporting the data. This processor should be the last processor defined in
the pipeline because issues that require retry are typically due to exporting.

Please refer to [config.go](./config.go) for the config spec.

The following configuration options can be modified:
- `backoff_delay` (default = 5s): Time interval to wait before retrying
- `num_workers` (default = 10): Number of workers that dequeue batches
- `queue_size` (default = 5000): Maximum number of batches kept in memory before data
is dropped
- `retry_on_failure` (default = true): Whether to retry on failure or give up and drop

Examples:

```yaml
processors:
  queued_retry/example:
    backoff_delay: 5s
    num_workers: 2
    queue_size: 10
    retry_on_failure: true
```

Refer to [config.yaml](./testdata/config.yaml) for detailed
examples on using the processor.
