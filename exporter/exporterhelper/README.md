# Exporter Helper

This package provides reusable implementations of common capabilities for exporters.
Currently, this includes queuing, batching, timeouts, and retries.

## Configuration

The following configuration options can be modified:

### Retry on Failure

- `retry_on_failure`
  - `enabled` (default = true)
  - `initial_interval` (default = 5s): Time to wait after the first failure before retrying; ignored if `enabled` is `false`
  - `max_interval` (default = 30s): Is the upper bound on backoff; ignored if `enabled` is `false`
  - `max_elapsed_time` (default = 300s): Is the maximum amount of time spent trying to send a batch; ignored if `enabled` is `false`. If set to 0, the retries are never stopped.
  - `multiplier` (default = 1.5): Factor by which the retry interval is multiplied on each attempt; ignored if `enabled` is `false`

### Sending Queue

- `sending_queue`
  - `enabled` (default = true)
  - `num_consumers` (default = 10): Number of consumers that dequeue batches; ignored if `enabled` is `false`
  - `wait_for_result` (default = false): determines if incoming requests are blocked until the request is processed or not.
  - `block_on_overflow` (default = false): If true, blocks the request until the queue has space otherwise rejects the data immediately; ignored if `enabled` is `false`
  - `sizer` (default = requests): How the queue and batching is measured. Available options: 
    - `requests`: number of incoming batches of metrics, logs, traces (the most performant option);
    - `items`: number of the smallest parts of each signal (spans, metric data points, log records);
    - `bytes`: the size of serialized data in bytes (the least performant option).
  - `queue_size` (default = 1000): Maximum size the queue can accept. Measured in units defined by `sizer`
  - `batch`: see below.

#### Sending queue batch settings

Batch settings are available in the sending queue. Batching is disabled, by default. To enable default
batch settings, use `batch: {}`. When `batch` is defined, the settings are:

- `flush_timeout` (default = 200 ms): time after which a batch will be sent regardless of its size. Must be a non-zero value;
- `min_size` (default = 8192): the minimum size of a batch; should be less than or equal to the `sending_queue::queue_size` if `sending_queue::batch::sizer` matches `sending_queue::sizer`.
- `max_size` (default = 0): the maximum size of a batch, enables batch splitting. The maximum size of a batch should be greater than or equal to the minimum size of a batch. If set to zero, there is no maximum size;
- `sizer`: see below.

The `batch::sizer` field is given special treatment because the queue itself also defines a `sizer`. This field supports using different size limits for the queue and batch-related logic. 

If the `batch::sizer` field is not set, it takes its value from the parent structure. 

If `sending_queue::sizer` is not set, `batch::sizer` defaults to `items`. 

Available `batch::sizer` options:

- `items`: number of the smallest parts of each signal (spans, metric data points, log records);
- `bytes`: the size of serialized data in bytes (the least performant option).
### Timeout

- `timeout` (default = 5s): Time to wait per individual attempt to send data to a backend

The `initial_interval`, `max_interval`, `max_elapsed_time`, and `timeout` options accept 
[duration strings](https://pkg.go.dev/time#ParseDuration),
valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".

### Persistent Queue

To use the persistent queue, the following setting needs to be set:

- `sending_queue`
  - `storage` (default = none): When set, enables persistence and uses the component specified as a storage extension for the persistent queue.
    There is no in-memory queue when set.

The maximum number of batches stored to disk can be controlled using `sending_queue.queue_size` parameter (which,
similarly as for in-memory buffering, defaults to 1000 batches).

When persistent queue is enabled, the batches are being buffered using the provided storage extension - [filestorage] is a popular and safe choice. If the collector instance is killed while having some items in the persistent queue, on restart the items will be picked and the exporting is continued.

**Context Propagation**: Request context (including client metadata and span context) is preserved when using persistent queues. However, context set by Auth extensions is **not** propagated through the persistent queue. Auth extension context is ignored when data is persisted to disk, which means authentication/authorization information will not be available when the persisted data is processed.

```
                                                              ┌─Consumer #1─┐
                                                              │    ┌───┐    │
                              ──────Deleted──────        ┌───►│    │ 1 │    ├───► Success
        Waiting in channel    x           x     x        │    │    └───┘    │
        for consumer ───┐     x           x     x        │    │             │
                        │     x           x     x        │    └─────────────┘
                        ▼     x           x     x        │
┌─────────────────────────────────────────x─────x───┐    │    ┌─Consumer #2─┐
│                             x           x     x   │    │    │    ┌───┐    │
│     ┌───┐     ┌───┐ ┌───┐ ┌─x─┐ ┌───┐ ┌─x─┐ ┌─x─┐ │    │    │    │ 2 │    ├───► Permanent -> X
│ n+1 │ n │ ... │ 6 │ │ 5 │ │ 4 │ │ 3 │ │ 2 │ │ 1 │ ├────┼───►│    └───┘    │      failure
│     └───┘     └───┘ └───┘ └───┘ └───┘ └───┘ └───┘ │    │    │             │
│                                                   │    │    └─────────────┘
└───────────────────────────────────────────────────┘    │
   ▲              ▲     ▲           ▲                    │    ┌─Consumer #3─┐
   │              │     │           │                    │    │    ┌───┐    │
   │              │     │           │                    │    │    │ 3 │    ├───► (in progress)
 write          read    └─────┬─────┘                    ├───►│    └───┘    │
 index          index         │                          │    │             │
                              │                          │    └─────────────┘
                              │                          │
                          currently                      │    ┌─Consumer #4─┐
                          dispatched                     │    │    ┌───┐    │     Temporary
                                                         └───►│    │ 4 │    ├───►  failure
                                                              │    └───┘    │         │
                                                              │             │         │
                                                              └─────────────┘         │
                                                                     ▲                │
                                                                     └── Retry ───────┤
                                                                                      │
                                                                                      │
                                                   X  ◄────── Retry limit exceeded ───┘
```

Example:

```
receivers:
  otlp:
    protocols:
      grpc:
exporters:
  otlp:
    endpoint: <ENDPOINT>
    sending_queue:
      storage: file_storage/otc
extensions:
  file_storage/otc:
    directory: /var/lib/storage/otc
    timeout: 10s
service:
  extensions: [file_storage]
  pipelines:
    metrics:
      receivers: [otlp]
      exporters: [otlp]
    logs:
      receivers: [otlp]
      exporters: [otlp]
    traces:
      receivers: [otlp]
      exporters: [otlp]

```

[filestorage]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/extension/storage/filestorage
