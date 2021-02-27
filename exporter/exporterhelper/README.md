# Exporter Helper

This is a helper exporter that other exporters can depend on. It offers several
independent capabilities including:

- Export settings
- In-memory queuing
- Ability to retry an export operation

> :warning: This exporter should not be added to a service pipeline.

## Configuration

The below configuration options can be modified. The full list of settings
exposed for this helper exporter are documented [here](factory.go).

### Export Settings

These settings modify the behavior when a payload is exported.

- `resource_to_telemetry_conversion`
  - `enabled` (default = `false`): If `enabled` is `true`, all the resource
    attributes will be converted to metric labels.
- `timeout` (default = `5s`): Time to wait per individual attempt to export data
  to a backend. If the backend accepts the data or returns an error before the
  configured `timeout` then the `timeout` setting is ignored. This setting is
  completely independent of `max_elapsed_time` under `retry_on_failure.`

### Queuing

These settings modify in-memory queuing. It is recommended to keep queuing
enabled so data is not lost during export in case of error or timeout.

- `sending_queue`
  - `enabled` (default = `true`)
  - `num_consumers` (default = `10`): Number of consumers that dequeue batches;
    ignored if `enabled` is `false`.
  - `queue_size` (default = `5000`): Maximum number of batches kept in memory
    before data; ignored if `enabled` is `false`.
  User should calculate this as `num_seconds * requests_per_second` where:
    - `num_seconds` is the number of seconds to buffer in case of a backend outage.
    - `requests_per_second` is the average number of requests per seconds.

### Retry

These settings modify retry. It is recommended to keep retry enabled so data is
not lost during export in case of error or timeout. If out of order data is a
concern for the configured destination, disabling retry may be desirable.

> On permanent errors, data is dropped and not retried. On partial error, only
> non-exported data is retried.

- `retry_on_failure`
  - `enabled` (default = `true`)
  - `initial_interval` (default = `5s`): Time to wait after the first failure
    (which includes a `timeout`) before retrying; ignored if `enabled` is
    `false`.
  - `max_interval` (default = `30s`): Is the upper bound on backoff; ignored if
    `enabled` is `false`.
  - `max_elapsed_time` (default = `120s`): Is the maximum amount of time spent
    trying to export data; ignored if `enabled` is `false`. The setting is only
    checked during a retry backoff operation which means the total time between
    first attempting to send data to a destination and the time at which
    `retry_on_failure` stops retrying may be more than `max_elapsed_time`. This
    setting is completely independent of `timeout`.

Retry happens when data fails to be accepted by the configured destination.
This failure may be an error returned by the destination or the result of a
`timeout`. If retry is enabled then on the first retry attempt
`initial_interval` must pass before the retry is attempted. On subsequent
retries, an exponential backoff occurs with the upper bound on backoff being
defined by `max_interval`. On every retry backoff operation, including the
initial retry, `max_elapsed_time` is checked and if exceeded then retrying
stops. Because `max_elapsed_time` is only checked on a retry backoff operation,
the total time between first attempting to send data to a destination and the
time at which retry stops retrying may be more than `max_elapsed_time`.

<details>
<summary>
Example
</summary>

For example, let's assume a configured destination is completely unavailable
for three minutes and that `max_elapsed_time` is changed to `30s`. For
demonstration purposes, let's use the `max_interval` for the backoff every
time.

- The first export request will `timeout` (total time = 5s)
- `max_elapsed_time` in `retry_on_failure` is checked and evaluates false
- `initial_interval` for `retry_on_failure` passes and then data is sent again
  (total time = 10s)
- The second export request will `timeout` (total time = 15s)
- `max_elapsed_time` in `retry_on_failure` is checked and evaluates false
- An exponential backoff occurs which can be a maximum of `max_interval`
  (assuming max, total time = 45s)
- The third export request will `timeout` (total time = 50s)
- `max_elapsed_time` in `retry_on_failure` is checked and evaluates to true;
  retry stops (total time is greater than `max_elapsed_time`)
</details>
