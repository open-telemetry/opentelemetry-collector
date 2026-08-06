# OCB Hooks

Author: @braydonk

AI USAGE DISCLOSURE: Some parts of the POC development leveraged Google Antigravity. The RFC contents are written entirely by me.

## Abstract

This RFC proposes a new feature in OCB that I am calling "hooks". It introduces a plugin interface for OCB plugins managed via [Hashicorp's Go Plugin framework][Hashicorp Go Plugin] which can be configured as hooks during OCB's process for generating and compiling a distribution. As of now, there are 4 hooks available: Pre/Post Generate, and Pre/Post Compile. Each hook can be individually skipped, or both skipped when Generation/Compilation respectively are disabled.

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
  plugins:
    - name: scriptplugin
      module: go.opentelemetry.io/cmd/builder/scriptplugin
      version: v0.157.0
  pre_generate:
    - plugin: scriptplugin
      path: ./scripts/pre_generate.sh
      args:
        - "--verbose"
  pre_build:
    - plugin: scriptplugin
      path: ./scripts/pre_build.sh
      env:
        BUILD_ENV: "staging"
```

A set of plugin sources are configured, which can be installed remotely via `module`/`version`, or locally via `path`. Those plugins can then be referenced by name in the different hook configurations, along with a generic configuration that will be passed to the hook.

## Defining a Plugin

An `ocbplugin` package will be provided for plugin authors. It will feature an interface for OCB plugins to implement, as well as helpers to facilitate their plugin host setup, which automatically configures the required handshake that OCB will expect. 

See the [Hashicorp Go Plugin][Hashicorp Go Plugin] library for more information on what these plugins generally look like, and [the example implementation of a bash script execution plugin in the POC](https://github.com/braydonk/opentelemetry-collector/tree/ocb_plugin_experiment/cmd/builder/scriptplugin).

OPEN QUESTION: Should this package live under `cmd/builder`? It is a bit odd for a library module to live under a `cmd` directory, but there is no `pkg` directory in `opentelemetry-collector`.

## Plugin Management

The procedure in OCB is as follows:

1. If any hooks are configured, go through the list of plugins and install them. By default the plugins will be installed into `$HOME/.ocb` (or `$(pwd)/.ocb` if user's home directory can't be resolved). Users can configure a plugin directory via the `OCB_PLUGIN_DIR` environment variable. Unless the `--reinstall-hooks` flag is provided, if any plugins are already detected as being installed it can reuse them.
1. Hooks are run in their respective places. The first time a hook is requested, OCB will start it as a subprocess and manage an RPC client to it for the rest of the generation/compilation process. If any plugin is incapable of being started for any reason, such as requesting an unsupported hook action, failing the handshake, or just not serving RPCs properly at all, it will fail the full OCB process.
1. Hooks will reuse the same plugin subprocess for each time the plugin is configured. After OCB is finished, all subprocesses are cleaned up.

## OBI Plugin Example

Within the POC I also [implemented an example OBI plugin](https://github.com/braydonk/opentelemetry-collector/tree/ocb_plugin_experiment/cmd/builder/obiplugin) and an [example config using it](https://github.com/braydonk/opentelemetry-collector/blob/ocb_plugin_experiment/cmd/builder/obiplugin/example/ocb-obi-config.yaml) that mirrors the functionality of [the `prepare-obi.sh` script in the releases repo](https://github.com/open-telemetry/opentelemetry-collector-releases/blob/145e73e8663aff5ce978ea38cf0cbd4a97017141/scripts/prepare-obi.sh) instead working as an OCB pre_generate hook. Since this feature was inspired by the need to include the OBI receiver in a unique way within OCB, I decided to implement it as part of this POC. In reality, this hook can be created and managed by the OBI SIG to work whatever way makes sense for them, and they can instruct users to use the plugin remotely via the `module` and `version`, rather than the `path` source like I have in the example config.

[Hashicorp Go Plugin]: https://github.com/hashicorp/go-plugin
[POC]: https://github.com/open-telemetry/opentelemetry-collector/compare/main...braydonk:opentelemetry-collector:ocb_plugin_experiment
