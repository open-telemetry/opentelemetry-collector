# OpenTelemetry Collector builder

This program generates a custom OpenTelemetry Collector binary based on a given configuration.

## TL;DR
```console
$ GO111MODULE=on go get github.com/open-telemetry/opentelemetry-collector-builder
$ cat > ~/.otelcol-builder.yaml <<EOF
exporters:
  - gomod: "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/alibabacloudlogserviceexporter v0.26.0"
EOF
$ opentelemetry-collector-builder --output-path=/tmp/dist
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

Download the binary for your respective platform under the ["Releases"](https://github.com/open-telemetry/opentelemetry-collector-builder/releases/latest) page.

## Running

A configuration file isn't strictly required, but the final artifact won't be different than a regular OpenTelemetry Collector. You probably want to specify at least one module (extension, exporter, receiver, processor) to add to your distribution. You can specify them via a configuration file. When no `--config` flag is provided with the location for the configuration file, `${HOME}/.otelcol-builder.yaml` will be used, if available.

```console
$ opentelemetry-collector-builder --config config.yaml
```

Use `opentelemetry-collector-builder --help` to learn about which flags are available.

## Configuration

The configuration file is composed of two main parts: `dist` and module types. All `dist` options can be specified via command line flags:

```console
$ opentelemetry-collector-builder --name="my-otelcol"
```

The module types are specified at the top-level, and might be: `extensions`, `exporters`, `receivers` and `processors`. They all accept a list of components, and each component is required to have at least the `gomod` entry. When not specified, the `import` value is inferred from the `gomod`. When not specified, the `name` is inferred from the `import`.

The `import` might specify a more specific path than what is specified in the `gomod`. For instance, your Go module might be `gitlab.com/myorg/myrepo` and the `import` might be `gitlab.com/myorg/myrepo/myexporter`.

The `name` will typically be omitted, except when multiple components have the same name. In such case, set a unique name for each module.

Optionally, a list of `go mod` replace entries can be provided, in case custom overrides are needed. This is typically necessary when a processor or some of its transitive dependencies have dependency problems.

```yaml
dist:
    module: github.com/open-telemetry/opentelemetry-collector-builder # the module name for the new distribution, following Go mod conventions. Optional, but recommended.
    name: otelcol-custom # the binary name. Optional.
    description: "Custom OpenTelemetry Collector distribution" # a long name for the application. Optional.
    include_core: true # whether the core components should be included in the distribution. Optional.
    otelcol_version: "0.26.0" # the OpenTelemetry Collector version to use as base for the distribution. Optional.
    output_path: /tmp/otelcol-distributionNNN # the path to write the output (sources and binary). Optional.
    version: "1.0.0" # the version for your custom OpenTelemetry Collector. Optional.
    go: "/usr/bin/go" # which Go binary to use to compile the generated sources. Optional.
exporters:
  - gomod: "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/alibabacloudlogserviceexporter v0.26.0" # the Go module for the component. Required.
    import: "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/alibabacloudlogserviceexporter" # the import path for the component. Optional.
    name: "alibabacloudlogserviceexporter" # package name to use in the generated sources. Optional.
    path: "./alibabacloudlogserviceexporter" # in case a local version should be used for the module, the path relative to the current dir, or a full path can be specified. Optional.
replaces:
  # a list of "replaces" directives that will be part of the resulting go.mod
  - github.com/open-telemetry/opentelemetry-collector-contrib/internal/common => github.com/open-telemetry/opentelemetry-collector-contrib/internal/common v0.26.0
```
