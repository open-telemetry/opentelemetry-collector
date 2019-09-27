# Extensions
*Note* This documentation is still in progress. For any questions, please reach
out in the [OpenTelemetry Gitter](https://gitter.im/open-telemetry/opentelemetry-service)
or refer to the [issues page](https://github.com/open-telemetry/opentelemetry-collector/issues).

Supported service extensions (sorted alphabetically):
- [Health Check](#health_check)
- [Performance Profiler](#pprof)
- [zPages](#zpages)

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

## <a name="health_check"></a>Health Check
Health Check extension enables an HTTP url that can be probed to check the
status of the the OpenTelemetry Collector. The only configuration setting is the
port in which the endpoint is going to be available, the default port is 13133.

This extension can be used as kubernetes liveness and readiness probe.

Configuration:
```yaml

extensions:
  # Configures the health_check extension to expose an HTTP endpoint with the
  # service status.
  health_check:
    # Specifies the port in which the HTTP endpoint is going to be opened. The
    # default value is 13133.
    port: 13133
```

## <a name="pprof"></a>Performance Profiler
Performance Profiler extension enables the golang `net/http/pprof` endpoint.
This is typically used by developers to collect performance profiles and
investigate issues with the service.

Configuration:
```yaml

extensions:
  # Configures the pprof (Performance Profiler) extension to expose an HTTP
  # endpoint that can be used by the golang tool pprof to collect profiles.
  # The default values are listed below.
  pprof:
    # The endpoint in which the pprof will be listening to. Use localhost:<port>
    # to make it available only locally, or ":<port>" to make it available on
    # all network interfaces.
    endpoint: localhost:1777
    # Fraction of blocking events that are profiled. A value <= 0 disables
    # profiling. See https://golang.org/pkg/runtime/#SetBlockProfileRate for details.
    block_profile_fraction: 0
    # Fraction of mutex contention events that are profiled. A value <= 0
    # disables profiling. See https://golang.org/pkg/runtime/#SetMutexProfileFraction
    # for details.
    mutex_profile_fraction: 0
```

## <a name="zpages"></a>zPages
Enables an extension that serves zPages, an HTTP endpoint that provides live
data for debugging different components that were properly instrumented for such.
All core exporters and receivers provide some zPage instrumentation.

Configuration:
```yaml
extensions:
  # Configures the zPages extension to expose an HTTP endpoint that can be used
  # to debug components of the service.
  zpages:
    # Specifies the HTTP endpoint is going to be opened to serve zPages.
    # Use localhost:<port> to make it available only locally, or ":<port>" to
    # make it available on all network interfaces.
    # The default value is listed below.
    endpoint: localhost:55679
```
