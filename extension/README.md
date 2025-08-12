# General Information

Extensions provide capabilities on top of the primary functionality of the
collector. Generally, extensions are used for implementing components that can
be added to the Collector, but which do not require direct access to telemetry
data and are not part of the pipelines (like receivers, processors or
exporters). Example extensions are: Memory Limiter extension that prevents
out of memory situations or zPages extension that provides live data for 
debugging different components.

Supported service extensions (sorted alphabetically):

- [Memory Limiter](memorylimiterextension/README.md)
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
  extensions: [extension1, extension2]
```
