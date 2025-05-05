# Generating example output

This document describes how to generate the example output used in the [README](./README.md)'s [Verbosity levels](./README.md#verbosity-levels) section.

1. Prepare the configuration of the Collector.

   ```yaml
   exporters:
     debug/basic:
       verbosity: basic
     debug/normal:
       verbosity: normal
     debug/detailed:
       verbosity: detailed

   receivers:
     otlp:
       protocols:
         grpc:

   service:
     pipelines:
       traces:
         exporters:
           - debug/basic
           - debug/normal
           - debug/detailed
         receivers:
           - otlp
   ```

2. Run the Collector (download latest version from <https://github.com/open-telemetry/opentelemetry-collector-releases/releases>).

   ```console
   otelcol --config config.yaml
   ```

3. Run the `telemetrygen` tool (install latest version with `go install github.com/open-telemetry/opentelemetry-collector-contrib/cmd/telemetrygen@latest`).

   ```console
   telemetrygen traces --otlp-insecure
   ```
