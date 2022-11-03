# OpenTelemetry Collector Builder (ocb)
[![CI](https://github.com/open-telemetry/opentelemetry-collector-builder/actions/workflows/go.yaml/badge.svg)](https://github.com/open-telemetry/opentelemetry-collector-builder/actions/workflows/go.yaml?query=branch%3Amain)

This program generates a custom OpenTelemetry Collector binary based on a given configuration.

## TL;DR

```console
$ GO111MODULE=on go install go.opentelemetry.io/collector/cmd/builder@latest
$ cat > otelcol-builder.yaml <<EOF
exporters:
  - gomod: github.com/open-telemetry/opentelemetry-collector-contrib/exporter/alibabacloudlogserviceexporter v0.64.0
  - gomod: go.opentelemetry.io/collector/exporter/loggingexporter v0.64.0

receivers:
  - gomod: go.opentelemetry.io/collector/receiver/otlpreceiver v0.64.0

processors:
  - gomod: go.opentelemetry.io/collector/processor/batchprocessor v0.64.0
EOF
$ builder --config=otelcol-builder.yaml --output-path=/tmp/dist
$ cat > /tmp/otelcol.yaml <<EOF
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: localhost:4317

processors:
  batch:

exporters:
  logging:

service:
  pipelines:
    traces:
      receivers:
      - otlp
      processors:
      - batch
      exporters:
      - logging
EOF
$ /tmp/dist/otelcol-custom --config=/tmp/otelcol.yaml
```

## Installation

Download the binary for your respective platform under the ["Releases"](https://github.com/open-telemetry/opentelemetry-collector/releases/latest) page.
If install an official release build, the binary is named `ocb`, but if you installed by using `go install`, it will be called `builder`.

## Running

A build configuration file must be provided with the `--config` flag.
You will need to specify at least one module (extension, exporter, receiver, processor) to add to your distribution.
To build a default collector configuration, you can use [this](../otelcorecol/builder-config.yaml) build configuration.

```console
$ ocb --config=builder-config.yaml
```

Use `ocb --help` to learn about which flags are available.

## Configuration

The configuration file is composed of two main parts: `dist` and module types. All `dist` options can be specified via command line flags:

```console
$ ocb --config=config.yaml --name="my-otelcol"
```

The module types are specified at the top-level, and might be: `extensions`, `exporters`, `receivers` and `processors`. They all accept a list of components, and each component is required to have at least the `gomod` entry. When not specified, the `import` value is inferred from the `gomod`. When not specified, the `name` is inferred from the `import`.

The `import` might specify a more specific path than what is specified in the `gomod`. For instance, your Go module might be `gitlab.com/myorg/myrepo` and the `import` might be `gitlab.com/myorg/myrepo/myexporter`.

The `name` will typically be omitted, except when multiple components have the same name. In such case, set a unique name for each module.

Optionally, a list of `go mod` replace entries can be provided, in case custom overrides are needed. This is typically necessary when a processor or some of its transitive dependencies have dependency problems.

```yaml
dist:
    module: github.com/open-telemetry/opentelemetry-collector # the module name for the new distribution, following Go mod conventions. Optional, but recommended.
    name: otelcol-custom # the binary name. Optional.
    description: "Custom OpenTelemetry Collector distribution" # a long name for the application. Optional.
    otelcol_version: "0.40.0" # the OpenTelemetry Collector version to use as base for the distribution. Optional.
    output_path: /tmp/otelcol-distributionNNN # the path to write the output (sources and binary). Optional.
    version: "1.0.0" # the version for your custom OpenTelemetry Collector. Optional.
    go: "/usr/bin/go" # which Go binary to use to compile the generated sources. Optional.
exporters:
  - gomod: "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/alibabacloudlogserviceexporter v0.40.0" # the Go module for the component. Required.
    import: "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/alibabacloudlogserviceexporter" # the import path for the component. Optional.
    name: "alibabacloudlogserviceexporter" # package name to use in the generated sources. Optional.
    path: "./alibabacloudlogserviceexporter" # in case a local version should be used for the module, the path relative to the current dir, or a full path can be specified. Optional.
replaces:
  # a list of "replaces" directives that will be part of the resulting go.mod
  - github.com/open-telemetry/opentelemetry-collector-contrib/internal/common => github.com/open-telemetry/opentelemetry-collector-contrib/internal/common v0.40.0
```
