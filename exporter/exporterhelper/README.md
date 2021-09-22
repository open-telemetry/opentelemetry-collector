# Exporter Helper

This is a helper exporter that other exporters can depend on. Today, it
primarily offers queued retries  and resource attributes to metric labels conversion.

> :warning: This exporter should not be added to a service pipeline.

## Configuration

The following configuration options can be modified:

- `retry_on_failure`
  - `enabled` (default = true)
  - `initial_interval` (default = 5s): Time to wait after the first failure before retrying; ignored if `enabled` is `false`
  - `max_interval` (default = 30s): Is the upper bound on backoff; ignored if `enabled` is `false`
  - `max_elapsed_time` (default = 120s): Is the maximum amount of time spent trying to send a batch; ignored if `enabled` is `false`
- `sending_queue`
  - `enabled` (default = true)
  - `num_consumers` (default = 10): Number of consumers that dequeue batches; ignored if `enabled` is `false`
  - `queue_size` (default = 5000): Maximum number of batches kept in memory before dropping; ignored if `enabled` is `false`
  User should calculate this as `num_seconds * requests_per_second` where:
    - `num_seconds` is the number of seconds to buffer in case of a backend outage
    - `requests_per_second` is the average number of requests per seconds.
- `resource_to_telemetry_conversion`
  - `enabled` (default = false): If `enabled` is `true`, all the resource attributes will be converted to metric labels by default.
- `timeout` (default = 5s): Time to wait per individual attempt to send data to a backend.

The full list of settings exposed for this helper exporter are documented [here](factory.go).

### Persistent Queue

**Status: under development**

> :warning: The capability is under development and currently can be enabled only in OpenTelemetry
> Collector Contrib with `enable_unstable` build tag set. 

With this build tag set, additional configuration option can be enabled:

- `sending_queue`
  - `persistent_storage_enabled` (default = false): When set, enables persistence via a file storage extension
    (note, `enable_unstable` build tag needs to be enabled first, see below for more details)

The maximum number of batches stored to disk can be controlled using `sending_queue.queue_size` parameter (which,
similarly as for in-memory buffering, defaults to 5000 batches).

When `persistent_storage_enabled` is set to true, the queue is being buffered to disk using 
[file storage extension](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/extension/storage/filestorage).
If collector instance is killed while having some items in the persistent queue, on restart the items are being picked and
the exporting is continued.

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
   ▲                          │                          │    └─────────────┘
   │                          │                          │
   │                      currently                      │    ┌─Consumer #4─┐
   │                      dispatched                     │    │    ┌───┐    │     Temporary
   │                                                     └───►│    │ 4 │    ├───►  failure
   │                                                          │    └───┘    │         │
   │                                                          │             │         │
   │                                                          └─────────────┘         │
   │                                                                 ▲                │
   │                                                                 └── Retry ───────┤
   │                                                                                  │
   │                                                                                  │
   └────────────────────────────────────── Requeuing  ◄────── Retry limit exceeded ───┘
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
      persistent_storage_enabled: true
extensions:
  file_storage:
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