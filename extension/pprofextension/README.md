# Performance Profiler

Performance Profiler extension enables the golang `net/http/pprof` endpoint.
This is typically used by developers to collect performance profiles and
investigate issues with the service.

The following settings are required:

- `endpoint` (default = localhost:1777): The endpoint in which the pprof will
be listening to. Use localhost:<port> to make it available only locally, or
":<port>" to make it available on all network interfaces.
- `block_profile_fraction` (default = 0): Fraction of blocking events that
are profiled. A value <= 0 disables profiling. See
https://golang.org/pkg/runtime/#SetBlockProfileRate for details.
- `mutex_profile_fraction` (default = 0): Fraction of mutex contention
events that are profiled. A value <= 0 disables profiling. See
https://golang.org/pkg/runtime/#SetMutexProfileFraction for details.

The following settings can be optionally configured:

- `save_to_file`: File name to save the CPU profile to. The profiling starts when the
Collector starts and is saved to the file when the Collector is terminated.

Example:
```yaml

extensions:
  pprof:
```

The full list of settings exposed for this exporter are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).
