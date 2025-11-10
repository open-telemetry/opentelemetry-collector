# OpenTelemetry Collector Builder (ocb)

This program generates a custom OpenTelemetry Collector binary based on a given configuration.

## TL;DR

```console
$ go install go.opentelemetry.io/collector/cmd/builder@v0.129.0
$ cat > otelcol-builder.yaml <<EOF
dist:
  name: otelcol-custom
  description: Local OpenTelemetry Collector binary
  output_path: /tmp/dist
exporters:
  - gomod: github.com/open-telemetry/opentelemetry-collector-contrib/exporter/alibabacloudlogserviceexporter v0.129.0
  - gomod: go.opentelemetry.io/collector/exporter/debugexporter v0.129.0

receivers:
  - gomod: go.opentelemetry.io/collector/receiver/otlpreceiver v0.129.0

processors:
  - gomod: go.opentelemetry.io/collector/processor/batchprocessor v0.129.0

providers:
  - gomod: go.opentelemetry.io/collector/confmap/provider/envprovider v1.35.0
  - gomod: go.opentelemetry.io/collector/confmap/provider/fileprovider v1.35.0
  - gomod: go.opentelemetry.io/collector/confmap/provider/httpprovider v1.35.0
  - gomod: go.opentelemetry.io/collector/confmap/provider/httpsprovider v1.35.0
  - gomod: go.opentelemetry.io/collector/confmap/provider/yamlprovider v1.35.0
EOF
$ builder --config=otelcol-builder.yaml
$ cat > /tmp/otelcol.yaml <<EOF
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: localhost:4317

exporters:
  debug:

service:
  pipelines:
    traces:
      receivers:
      - otlp
      exporters:
      - debug
EOF
$ /tmp/dist/otelcol-custom --config=/tmp/otelcol.yaml
```

## Installation

There are three supported ways to install the builder:
1. Via official release Docker images (recommended)
2. Via official release binaries (recommended)
3. Through `go install` (not recommended)

### Official release Docker image

You will find the official docker images at [DockerHub](https://hub.docker.com/r/otel/opentelemetry-collector-builder).

Pull the image via tagged version number (e.g. `0.129.0`) or 'latest'. You may also specify platform, although Docker will handle this automatically as it is a multi-platform build.

```console
docker pull otel/opentelemetry-collector-builder:latest
```

The included builder configuration file/manifest should be replaced by mounting a file from your local filesystem to the docker container; the default location is `/build/builder-config.yaml`. If you mount a file at a different location inside the container, your `builder.config.yaml` must be specified as a command line argument to ocb. Additionally, the output folder must also be mounted from your local system to the docker container. This output directory must be specified in your `builder-config.yaml` file as it cannot be set via the command-line arguments.

Assuming you are running this image in your working directory, have a `builder-config.yaml` file located in this folder, the `dist.output_path` item inside your `builder-config.yaml` is set to `./otelcol-dev`, and you wish to output the binary/go module files to a folder named `output`, the command would look as follows:

```console
docker run -v "$(pwd)/builder-config.yaml:/build/builder-config.yaml" -v "$(pwd)/output:/build/otelcol-dev" otel/opentelemetry-collector-builder:latest --config=/build/builder-config.yaml
```

Please note that a `--config` flag must be passed to specify your custom manifest.yaml/builder-config.yaml file regardless of where you mount it inside the container, otherwise a default config is used that cannot be changed.

Additional arguments may be passed to ocb on the command line as specified below, but if you wish to do this, you must make sure to pass the `--config` argument, as this is specified as an additional `CMD`, not an entrypoint.

### Official release binaries

This is the recommended installation method for the binary. Download the binary for your respective platform from the ["Releases"](https://github.com/open-telemetry/opentelemetry-collector-releases/releases?q=cmd/builder) page.

### `go install`

You need to have a `go` compiler in your PATH. Run the following command to install the latest version:

```console
go install go.opentelemetry.io/collector/cmd/builder@latest
```

If installing through this method the binary will be called `builder`.

In order to successfully generate and build a collector using ocb, you must use [compatible Go version](../../README.md#compatibility).

## Running

A build configuration file must be provided with the `--config` flag.
You will need to specify at least one module (extension, exporter, receiver, processor) to add to your distribution.
To build a default collector configuration, you can use [this](../otelcorecol/builder-config.yaml) build configuration.

```console
ocb --config=builder-config.yaml
```

Use `ocb --help` to learn about which flags are available.

## Debug

### Debug symbols

By default, the LDflags are set to `-s -w`, which strips debugging symbols to produce a smaller OpenTelemetry Collector binary. To retain debugging symbols and DWARF debugging data in the binary, override the LDflags as shown:

```console
ocb --ldflags="" --config=builder-config.yaml.
```

### Debugging with Delve

To ensure the code being executed matches the written code exactly, debugging symbols must be preserved, and compiler inlining and optimizations disabled. You can achieve this in two ways:

1. Set the configuration property `debug_compilation` to true.
2. Manually override the ldflags and gcflags `ocb --ldflags="" --gcflags="all=-N -l" --config=builder-config.yaml.`

Then install `go-delve` and run OpenTelemetry Collector with `dlv` command as the following example:

```bash
# go install github.com/go-delve/delve/cmd/dlv@latest
# ~/go/bin/dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient --log exec .otel-collector-binary -- --config otel-collector-config.yaml
```

Finally, load the OpenTelemetry Collector as a project in the IDE, configure debug for Go

## Configuration

The configuration file is composed of two main parts: `dist` and module types. All `dist` options can be specified via command line flags:

```console
ocb --config=config.yaml
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
    output_path: /tmp/otelcol-distributionNNN # the path to write the output (sources and binary). Optional.
    version: "1.0.0" # the version for your custom OpenTelemetry Collector. Optional.
    go: "/usr/bin/go" # which Go binary to use to compile the generated sources. Optional.
    debug_compilation: false # enabling this causes the builder to keep the debug symbols in the resulting binary. Optional.
exporters:
  - gomod: "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/alibabacloudlogserviceexporter v0.129.0" # the Go module for the component. Required.
    import: "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/alibabacloudlogserviceexporter" # the import path for the component. Optional.
    name: "alibabacloudlogserviceexporter" # package name to use in the generated sources. Optional.
    path: "./alibabacloudlogserviceexporter" # in case a local version should be used for the module, the path relative to the current dir, or a full path can be specified. Optional.
replaces:
  # a list of "replaces" directives that will be part of the resulting go.mod
  - github.com/open-telemetry/opentelemetry-collector-contrib/internal/common => github.com/open-telemetry/opentelemetry-collector-contrib/internal/common v0.128.0
```

The builder also allows setting the scheme to use as the default URI scheme via `conf_resolver.default_uri_scheme`:

```yaml
conf_resolver:
   default_uri_scheme: "env"
```

This tells the builder to produce a Collector that uses the `env` scheme when expanding configuration that does not
provide a scheme, such as `${HOST}` (instead of doing `${env:HOST}`).

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

### Cgo disabled by default

By default, the OpenTelemetry Collector binary is built with `CGO_ENABLED=0` in accordance with
how the official OpenTelemetry Collector releases are built. This can be overridden by adding
the following configuration option to the `dist` section of the builder configuration file:

```yaml
dist:
  cgo_enabled: true
```
