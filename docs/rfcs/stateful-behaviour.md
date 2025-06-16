# Simplifying stateful behaviour

## Motivation

Currently, the collector operates in a stateless mode by default, with stateful components storing offsets in memory. Ideally, stateful components should persist their state during shutdown if a storage extension is available. 

This was the case previously with `filelog` receiver, but it was reverted. Refer [historical context](#historical-context) for more details on this.

At present, enabling stateful behavior involves a somewhat lengthy process:

1. Adding a filestorage entry to the extensions stanza in the configuration.

```yaml
# main.yaml
receivers:
  ...
processors:
  ...
exporters:
  ...
extensions:
  file_storage: 		<--- HERE

service:
  ...
```

2. Including filestorage under service::extensions.

```yaml
# main.yaml
receivers:
  ...
processors:
  ...
exporters:
  ...
extensions:
  file_storage:

service:
  extensions: [file_storage]     <--- HERE
  ...
```

3. Adding storage: file_storage/xyz to individual components.

```yaml
# main.yaml
receivers:
  filelog:
    storage: file_storage    <--- HERE	
processors:
  ...
exporters:
  ...
extensions:
  file_storage:

service:
  extensions: [file_storage]
```

It would be beneficial to simplify this process by introducing a feature gate or simple single configuration option(s). With this approach, users could enable stateful mode with a single setting, and the necessary steps would be handled automatically, achieving the same effect as the manual steps described.

### Historical context

A few years ago, filelog receiver automatically _hooked_ a storage extension when it was specified in the piepline. There was no need to explicitly mention `storage: file_storage` in the stanza confiugration. 
That changed was [reverted](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/10915) due to following downsides:
1. It was not possible to specify multiple storage extensions and use a preferred one for the receiver.
2. User had no option to opt out of statefulness.

This RFC addresses above mentioned downsides.

## Scope of this RFC

In this RFC, I'll only use `filelogreceiver` and `filestorage` extension for demonstration. We can enable this feature for any stateful receivers if we want to in future.

## Explanation

This RFC proposes to solve above mentioned limitations using a [config converter](https://github.com/open-telemetry/opentelemetry-collector/tree/main/confmap#converter).
We can automate the 3-step manual process by using a converter (let's call it `statefulconverter` for the purpose of this RFC), in a following way:

1. We can create a new converter to inject the `filestorage` extension into the configuration provided by the user.
2. We can then loop through the config and inject `storage: file_storage` into the receiver config, if not set.

All of the above steps can be controlled via a single command line option, making it easier for us to enable statefulness without changing the code or introducing side effects.

Here’s the POC [changeset](https://github.com/open-telemetry/opentelemetry-collector-contrib/compare/main...VihasMakwana:opentelemetry-collector-contrib:stateful-flag) implementing `statefulconverter` and updating `filelogreceiver`. It is not ready for review, but give you a rough idea of how we'll update the config.

## User experience

In default mode, the user experience will be unaffected. If they want to enable statefulness, it will be simplified by using following command:

```bash
otelcontribcol_darwin_arm64 --feature-gates=stateful --config otel.yaml
```

> [!NOTE] 
> Users will not change configuration files in any way. Everything will be taken care of by the converters. 

## Internal details

Ideally, we should introduce a new command-line flag and use a feature gate to control its functionality.
We can add a boolean in [ConverterSettings](https://github.com/open-telemetry/opentelemetry-collector/blob/3ef58fda95de1fa1d27f1d43c9ef92193bac0b2d/confmap/converter.go#L13-L22) and set it to `false` by default to avoid any breaking changes.
The statefulconverter will be enabled conditionally based on the value of this boolean.

> [!NOTE] 
> By default, offsets are stored in memory. The main goal of this RFC is to simplify enabling statefulness.

### Changes required for this feature

1. The stateful converter implementation and the feature gate will live in `opentelemetry-collector-contrib` repository.
2. The changes in core can be divded into two parts:
   a. In the first part, we can just add a boolean to [ConverterSettings](https://github.com/open-telemetry/opentelemetry-collector/blob/3ef58fda95de1fa1d27f1d43c9ef92193bac0b2d/confmap/converter.go#L13-L22).
   b. Finally, we can add a new command line flag to update the boolean.

## Examples using the feature gate

1. Use default directory to store offsets:

```yaml
# config.yaml
receivers:
  filelog:
    include: [/var/log/*.log]
exporters:
  debug:
service:
  logs:
    receivers: [filelog]
    exporters: [debug]
```

  - _Command_
```bash
    otelcontribcol_darwin_arm64 --feature-gates=stateful --config config.yaml
```

2. Use custom directory to store offsets

```yaml
# config.yaml
receivers:
  filelog:
    include: [/var/log/*.log]
exporters:
  debug:
extensions:
  file_storage/_stateful:
    directory: NEW_DIRECTORY
service:
  logs:
    receivers: [filelog]
    exporters: [debug]
```

  - _Command_
```bash
    otelcontribcol_darwin_arm64 --feature-gates=stateful --config config.yaml
```

3. Use mutliple receivers

```yaml
# config.yaml
receivers:
  filelog:
    include: [/var/log/*.log]
  filelog/syslog:
    include: [/var/log/syslog]
exporters:
  debug:
service:
  logs:
    receivers: [filelog]
    exporters: [debug]
```

  - _Command_
```bash
    otelcontribcol_darwin_arm64 --feature-gates=stateful --config config.yaml
```

4. Opt out of statefulness for receiver(s), with flag enabled

```yaml
# config.yaml
receivers:
  filelog:
    include: [/var/log/*.log]
    storage: null               ## IMPORTANT
  filelog/syslog:
    include: [/var/log/syslog]
exporters:
  debug:
service:
  logs:
    receivers: [filelog]
    exporters: [debug]
```

  - _Command_
```bash
    otelcontribcol_darwin_arm64 --feature-gates=stateful --config config.yaml
```

5. Overwrite default storage extension for receiver(s)

```yaml
# config.yaml
receivers:
  filelog:
    include: [/var/log/*.log]
    storage: file_storage
  filelog/syslog:
    include: [/var/log/syslog]
exporters:
  debug:
extensions:
  file_storage:
    directory: DIR
service:
  logs:
    receivers: [filelog]
    exporters: [debug]
```

  - _Command_
```bash
    otelcontribcol_darwin_arm64 --feature-gates=stateful --config config.yaml
```