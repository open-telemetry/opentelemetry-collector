# Exporter Helper

This is a helper exporter that other exporters can depend on. Today, it primarily offers queued retry capabilities.

> :warning: This exporter should not be added to a service pipeline.

## Configuration

The following configuration options can be modified:

- `retry_on_failure`
  - `enabled` (default = true)
  - `initial_interval` (default = 5s): Time to wait after the first failure before retrying; ignored if `enabled` is `false`
  - `max_interval` (default = 30s): Is the upper bound on backoff; ignored if `enabled` is `false`
  - `max_elapsed_time` (default = 300s): Is the maximum amount of time spent trying to send a batch; ignored if `enabled` is `false`. If set to 0, the retries are never stopped.
- `sending_queue`
  - `enabled` (default = true)
  - `num_consumers` (default = 10): Number of consumers that dequeue batches; ignored if `enabled` is `false`
  - `queue_size` (default = 1000): Maximum number of batches kept in memory before dropping; ignored if `enabled` is `false`
  User should calculate this as `num_seconds * requests_per_second / requests_per_batch` where:
    - `num_seconds` is the number of seconds to buffer in case of a backend outage
    - `requests_per_second` is the average number of requests per seconds
    - `requests_per_batch` is the average number of requests per batch (if 
      [the batch processor](https://github.com/open-telemetry/opentelemetry-collector/tree/main/processor/batchprocessor)
      is used, the metric `send_batch_size` can be used for estimation)
- `timeout` (default = 5s): Time to wait per individual attempt to send data to a backend
- `short_timeout_policy` (default: sustain): The policy toward handling of short-timeout requests. See below.

The `initial_interval`, `max_interval`, `max_elapsed_time`, and `timeout` options accept 
[duration strings](https://pkg.go.dev/time#ParseDuration),
valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".

### Timeout

The Timeout sender imposes a context deadline on the Export request.

It is common, in the OpenTelemetry Collector, for receiver components
to associate a context deadline with a request.  Typically, gRPC
receivers propagate the timeout from a preceding exporter or
OpenTelemetry SDK, so in many cases a request context will already
have an associated deadline when the Timeout sender is called.

If there is an existing context deadline associated with the request,
by default the Timeout sender will not extend the deadline.

If the associated context deadline is shorter than the `timeout`
configuration, the `short_timeout_policy` value determines behavior:

- **sustain**: The default behavior, this allows a shorter-than-configured timeout to remain in effect.
- **ignore**: Optional behavior: this allows ignoring the short timeout; requests will be issued with the full configured `timeout` setting, irrespective of the caller's deadline.
- **abort**: Optional behavior: this allows failing requests that callers will abort before the full configured `timeout` setting.

If the `timeout` is zero, a new deadline will not be added to the
export context; however, by default, incoming context deadlines remain
in effect.  To disable timeouts while ignoring the caller's deadline,
set `timeout` to `0` and `short_timeout_policy` to `ignore`.

### Persistent Queue

To use the persistent queue, the following setting needs to be set:

- `sending_queue`
  - `storage` (default = none): When set, enables persistence and uses the component specified as a storage extension for the persistent queue.
    There is no in-memory queue when set.

The maximum number of batches stored to disk can be controlled using `sending_queue.queue_size` parameter (which,
similarly as for in-memory buffering, defaults to 1000 batches).

When persistent queue is enabled, the batches are being buffered using the provided storage extension - [filestorage] is a popular and safe choice. If the collector instance is killed while having some items in the persistent queue, on restart the items will be picked and the exporting is continued.

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
[alpha]: https://github.com/open-telemetry/opentelemetry-collector#alpha
