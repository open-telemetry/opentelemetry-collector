# OCB Hooks

Author: @braydonk

Collaborators: @jade-guiton-dd

AI USAGE DISCLOSURE: Some parts of the POC development leveraged Google Antigravity. The RFC contents are written entirely by me.

## Abstract

This RFC proposes a new feature in OCB that I am calling "hooks". It introduces a plugin interface for OCB which can be configured as hooks during OCB's process for generating and compiling a distribution. The hooks are Go binaries that accept a YAML file for configuration. As of now, there are 4 hooks available: Pre/Post Generate, and Pre/Post Compile. Each hook can be individually skipped, or both skipped when Generation/Compilation respectively are disabled.

## Proof of Concept

A proof of concept is [available on my Collector fork][POC].

## Example Configuration

The following is an example configuration that sets up a plugin that can run bash shell scripts, with pregenerate and prebuild setups:

```yaml
dist:
  name: otelcol-custom
  description: Custom Collector Distribution
  output_path: ./build
receivers:
  - gomod: go.opentelemetry.io/collector/receiver/otlpreceiver v0.156.0
exporters:
  - gomod: go.opentelemetry.io/collector/exporter/nopexporter v0.156.0
hooks:
  pre_generate:
    - plugin: ./scriptplugin
      path: ./scripts/pre_generate.sh
      args:
        - "--verbose"
  pre_build:
    - plugin: ./scriptplugin
      path: ./scripts/pre_build.sh
      env:
        BUILD_ENV: "staging"
```

A plugin can be referenced via a Go module URL in the `plugin` field. This can be a fully formed Go module URL (i.e. `github.com/me/my-plugin@v1.1.1`), an OCB-internal representation of a Go module string (i.e. `github.com/me/my-plugin v1.1.1`), or a local path.

## Defining a Plugin

An `ocbplugin` package will be provided for plugin authors. It will feature an interface for OCB plugins to implement, and provide a function to be called from the plugin's `main` function. The plugin itself only needs to implement the plugin interface and call `RunPlugin`, all other facilitation between OCB and the plugin is managed by OCB (running the subprocess with a config, interpreting the success/failure exit code).

See [the example implementation of a bash script execution plugin in the POC](https://github.com/braydonk/opentelemetry-collector/tree/ocb_plugin_experiment/cmd/builder/scriptplugin).

QUESTION: Should this package live under `cmd/builder`? It is a bit odd for a library module to live under a `cmd` directory, but there is no `pkg` directory in `opentelemetry-collector`.  
ANSWER: For at least the first iteration, due to it being directly related with the OCB binary we believe it makes sense for it to live in the binary's directory.

## Plugin Management

The procedure in OCB for one plugin is as follows:

1. `go install` the plugin using a configurable `GOPATH`; by default this will be `$HOME/.ocb` (or `$(pwd)/.ocb` if the user's home directory can't be resolved), and a different directory can be provided via `OCB_PLUGIN_DIR`.
1. After installing, run the plugin using `cmd/exec`, passing the plugin's config via a temporary file (stored in OS-respective `tmp` dir via `os.CreateTemp`). See [here](#why-pass-the-config-in-a-temporary-file) for an explanation about why a temporary file was chosen as the communication mechanism.
1. If the plugin fails (exits with code 1) fail the entire OCB process.

This procedure is repeated for every plugin discovered at every step. Due to the way `go install` is used, if a plugin is used repeatedly in a single OCB run it should be fetched from the Go build cache. If using a local Go plugin and iterating on the code, the build cache will miss and rebuild the plugin.

### Why pass the config in a temporary file?

It seems pretty logical to use `stdin` to pass this data along to the subprocess. We opted not to go this route to enable `stdin` to be used by OCB plugins to prompt for information; the OCB process's `stdin` is passed along to each plugin when they are executed to allow them to prompt specific additional information.

An example of this `stdin` prompting is [in the `scriptplugin` in the POC](https://github.com/braydonk/opentelemetry-collector/blob/a3f3b72045525424f1b9b4fdd595ac1ce0bee59a/cmd/builder/scriptplugin/script.go#L82-L94).

## OBI Plugin Example

Within the POC I also [implemented an example OBI plugin](https://github.com/braydonk/opentelemetry-collector/tree/ocb_plugin_experiment/cmd/builder/obiplugin) and an [example config using it](https://github.com/braydonk/opentelemetry-collector/blob/ocb_plugin_experiment/cmd/builder/obiplugin/example/ocb-obi-config.yaml) that mirrors the functionality of [the `prepare-obi.sh` script in the releases repo](https://github.com/open-telemetry/opentelemetry-collector-releases/blob/145e73e8663aff5ce978ea38cf0cbd4a97017141/scripts/prepare-obi.sh) instead working as an OCB `pre_generate` hook. Since this feature was inspired by the need to include the OBI receiver in a unique way within OCB, I decided to implement it as part of this POC. In reality, this hook can be created and managed by the OBI SIG to work whatever way makes sense for them, and they can instruct users to use the plugin remotely via the actual Go module URL when the plugin exists.

[POC]: https://github.com/open-telemetry/opentelemetry-collector/compare/main...braydonk:opentelemetry-collector:ocb_plugin_experiment
