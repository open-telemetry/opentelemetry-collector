# General Information

Extensions provide capabilities on top of the primary functionality of the
collector. Generally, extensions are used for implementing components that can
be added to the Collector, but which do not require direct access to telemetry
data and are not part of the pipelines (like receivers, processors or
exporters). Example extensions are: Health Check extension that responds to
health check requests or PProf extension that allows fetching Collector's
performance profile.

Supported service extensions (sorted alphabetically):

- [Health Check](healthcheckextension/README.md)
- [Performance Profiler](pprofextension/README.md)
- [zPages](zpagesextension/README.md)

The [contributors
repository](https://github.com/open-telemetry/opentelemetry-collector-contrib)
may have more extensions that can be added to custom builds of the Collector.

## Ordering Extensions

The order extensions are specified for the service is important as this is the
order in which each extension will be started and the reverse order in which they
will be shutdown. The ordering is determined in the `extensions` tag under the
`service` tag in the configuration file, example:

```yaml
service:
  # Extensions specified below are going to be loaded by the service in the
  # order given below, and shutdown on reverse order.
  extensions: [health_check, pprof, zpages]
```

# Extensions

## <a name="health_check"></a>Health Check
Health Check extension enables an HTTP url that can be probed to check the
status of the the OpenTelemetry Collector. This extension can be used as a
liveness and/or readiness probe on Kubernetes.

The following settings are required:

- `port` (default = 13133): What port to expose HTTP health information.

Example:

```yaml
extensions:
  health_check:
```

The full list of settings exposed for this exporter is documented [here](healthcheckextension/config.go)
with detailed sample configurations [here](healthcheckextension/testdata/config.yaml).

## <a name="pprof"></a>Performance Profiler

Performance Profiler extension enables the golang `net/http/pprof` endpoint.
This is typically used by developers to collect performance profiles and
investigate issues with the service.

The following settings are required:

- `endpoint` (default = localhost:1777): The endpoint in which the pprof will
be listening to.
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

The full list of settings exposed for this exporter are documented [here](pprofextension/config.go)
with detailed sample configurations [here](pprofextension/testdata/config.yaml).

## <a name="zpages"></a>zPages

Enables an extension that serves zPages, an HTTP endpoint that provides live
data for debugging different components that were properly instrumented for such.
All core exporters and receivers provide some zPages instrumentation.

The following settings are required:

- `endpoint` (default = localhost:55679): Specifies the HTTP endpoint that serves
zPages.

Example:

```yaml
extensions:
  zpages:
```

The full list of settings exposed for this exporter are documented [here](zpagesextension/config.go)
with detailed sample configurations [here](zpagesextension/testdata/config.yaml).
