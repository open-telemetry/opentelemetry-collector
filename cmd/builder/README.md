# OpenTelemetry Collector Builder (ocb)

This program generates a custom OpenTelemetry Collector binary based on a given configuration.

## TL;DR

```console
$ go install go.opentelemetry.io/collector/cmd/builder@latest
$ cat > otelcol-builder.yaml <<EOF
dist:
  name: otelcol-custom
  description: Local OpenTelemetry Collector binary
  output_path: /tmp/dist
exporters:
  - gomod: github.com/open-telemetry/opentelemetry-collector-contrib/exporter/alibabacloudlogserviceexporter v0.99.0
  - gomod: go.opentelemetry.io/collector/exporter/debugexporter v0.99.0

receivers:
  - gomod: go.opentelemetry.io/collector/receiver/otlpreceiver v0.99.0

processors:
  - gomod: go.opentelemetry.io/collector/processor/batchprocessor v0.99.0

providers:
  - gomod: go.opentelemetry.io/collector/confmap/provider/fileprovider v0.99.0

converters:
  - gomod: go.opentelemetry.io/collector/confmap/converter/expandconverter v0.99.0
EOF
$ builder --config=otelcol-builder.yaml
$ cat > /tmp/otelcol.yaml <<EOF
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: localhost:4317

processors:
  batch:

exporters:
  debug:

service:
  pipelines:
    traces:
      receivers:
      - otlp
      processors:
      - batch
      exporters:
      - debug
EOF
$ /tmp/dist/otelcol-custom --config=/tmp/otelcol.yaml
```

## Installation

There are two supported ways to install the builder: via the official releases (recommended) and through `go install`.

### Official releases 

This is the recommended installation method. Download the binary for your respective platform under the ["Releases"](https://github.com/open-telemetry/opentelemetry-collector/releases?q=builder) page.

### `go install`

You need to have a `go` compiler in your PATH. Run the following command to install the latest version:

```
go install go.opentelemetry.io/collector/cmd/builder@latest
```

If installing through this method the binary will be called `builder`. Binaries installed through this method [will incorrectly show `dev` as their version](https://github.com/open-telemetry/opentelemetry-collector/issues/8691).

## Running

A build configuration file must be provided with the `--config` flag.
You will need to specify at least one module (extension, exporter, receiver, processor) to add to your distribution.
To build a default collector configuration, you can use [this](../otelcorecol/builder-config.yaml) build configuration.

```console
$ ocb --config=builder-config.yaml
```

Use `ocb --help` to learn about which flags are available.

## Debug

To keep the debug symbols in the resulting OpenTelemetry Collector binary, set the configuration property `debug_compilation` to true.

Then install `go-delve` and run OpenTelemetry Collector with `dlv` command as the following example:
```bash
# go install github.com/go-delve/delve/cmd/dlv@latest
# ~/go/bin/dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient --log exec .otel-collector-binary -- --config otel-collector-config.yaml
```
Finally, load the OpenTelemetry Collector as a project in the IDE, configure debug for Go

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
    debug_compilation: false # enabling this causes the builder to keep the debug symbols in the resulting binary. Optional.
exporters:
  - gomod: "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/alibabacloudlogserviceexporter v0.40.0" # the Go module for the component. Required.
    import: "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/alibabacloudlogserviceexporter" # the import path for the component. Optional.
    name: "alibabacloudlogserviceexporter" # package name to use in the generated sources. Optional.
    path: "./alibabacloudlogserviceexporter" # in case a local version should be used for the module, the path relative to the current dir, or a full path can be specified. Optional.
replaces:
  # a list of "replaces" directives that will be part of the resulting go.mod
  - github.com/open-telemetry/opentelemetry-collector-contrib/internal/common => github.com/open-telemetry/opentelemetry-collector-contrib/internal/common v0.40.0
```

## Steps

The builder has 3 steps:

* Generate: generates the golang source code
* Get modules: generates the go.mod file based on the imported modules in the generated golang source code
* Compilation: builds the OpenTelemetry Collector executable

Each step can be skipped independently: `--skip-generate`, `--skip-get-modules` and `--skip-compilation`.

For instance, a code generation step could execute

```console
ocb --skip-compilation --config=config.yaml
```
then commit the code in a git repo. A CI can sync the code and execute
```console
ocb --skip-generate --skip-get-modules --config=config.yaml
```
to only execute the compilation step.

### Strict versioning checks

The builder checks the relevant `go.mod`
file for the following things after `go get`ing all components and calling 
`go mod tidy`:

1. The `dist::otelcol_version` field in the build configuration must have 
   matching major and minor versions as the core library version calculated by 
   the Go toolchain, considering all components.  A mismatch could happen, for 
   example, when the builder or one of the components depends on a newer release 
   of the core collector library.
2. For each component in the build configuration, the major and minor versions 
   included in the `gomod` module specifier must match the one calculated by
   the Go toolchain, considering all components.  A mismatch could
   happen, for example, when the enclosing Go module uses a newer
   release of the core collector library.
   
The `--skip-strict-versioning` flag disables these versioning checks. 
This flag is available temporarily and 
**will be removed in a future minor version**.