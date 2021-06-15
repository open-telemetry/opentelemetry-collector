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
  - `queue_size` (default = 5000): Maximum number of batches kept in memory or on disk (for persistent storage) before dropping; ignored if `enabled` is `false`
  User should calculate this as `num_seconds * requests_per_second` where:
    - `num_seconds` is the number of seconds to buffer in case of a backend outage
    - `requests_per_second` is the average number of requests per seconds.
  - `persistent_enabled` (default = false): When set, enables persistence via a file extension
- `resource_to_telemetry_conversion`
  - `enabled` (default = false): If `enabled` is `true`, all the resource attributes will be converted to metric labels by default.
- `timeout` (default = 5s): Time to wait per individual attempt to send data to a backend.

The full list of settings exposed for this helper exporter are documented [here](factory.go).

### Persistent Queue

**Status: under development**

When `persistent_enabled` is set, the queue is being buffered to disk by the 
[file storage extension](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/extension/storage/filestorage). 
It currently can be enabled only in OpenTelemetry Collector Contrib.

This has some limitations currently. The items that have been passed to consumer for the actual exporting 
are removed from persistent queue. In effect, in case of a sudden shutdown, they might be lost.

```
                                                   ┌─Consumer #1─┐
                                                   │    ┌───┐    │
                         ┌──Deleted──┐        ┌───►│    │ 1 │    ├───► Success
                         X     x     x        │    │    └───┘    │
                         x     x     x        │    │             │
                         x     x     x        │    └─────────────┘
                         x     x     x        │
 ┌─────────Persistent queue────x─────x───┐    │    ┌─Consumer #2─┐
 │                       x     x     x   │    │    │    ┌───┐    │
 │     ┌───┐     ┌───┐ ┌─x─┐ ┌─x─┐ ┌─x─┐ │    │    │    │ 2 │    ├───► Permanent -> X
 │ n+1 │ n │ ... │ 4 │ │ 3 │ │ 2 │ │ 1 │ ├────┼───►│    └───┘    │      failure
 │     └───┘     └───┘ └───┘ └───┘ └───┘ │    │    │             │
 │                                       │    │    └─────────────┘
 └───────────────────────────────────────┘    │
    ▲              ▲                          │    ┌─Consumer #3─┐
    │              │                          │    │    ┌───┐    │     Temporary
    │              │                          └───►│    │ 3 │    ├───►  failure
  write          read                              │    └───┘    │
  index          index                             │             │         │
    ▲                                              └─────────────┘         │
    │                                                     ▲                │
    │                                                     └── Retry ───────┤
    │                                                                      │
    │                                                                      │
    └───────────────────────── Requeuing ◄────────── Retry limit exceeded ─┘
```