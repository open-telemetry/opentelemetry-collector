# OpenTelemetry Collector Service

## How to provide configuration?

The `--config` flag accepts either a file path or values in the form of a config URI `"<scheme>:<opaque_data>"`.
Currently, the OpenTelemetry Collector supports the following providers `scheme`:
- [file](../confmap/provider/fileprovider/provider.go) - Reads configuration from a file. E.g. `file:path/to/config.yaml`.
- [env](../confmap/provider/envprovider/provider.go) - Reads configuration from an environment variable. E.g. `env:MY_CONFIG_IN_AN_ENVVAR`.
- [yaml](../confmap/provider/yamlprovider/provider.go) - Reads configuration from yaml bytes. E.g. `yaml:exporters::debug::verbosity: detailed`.
- [http](../confmap/provider/httpprovider/provider.go) - Reads configuration from a HTTP URI. E.g. `http://www.example.com`

For more technical details about how configuration is resolved you can read the [configuration resolving design](../confmap/README.md#configuration-resolving).

### Single Config Source

1. Simple local file:

    `./otelcorecol --config=examples/local/otel-config.yaml`

2. Simple local file using the new URI format:

    `./otelcorecol --config=file:examples/local/otel-config.yaml`

3. Config provided via an environment variable:

    `./otelcorecol --config=env:MY_CONFIG_IN_AN_ENVVAR`


### Multiple Config Sources

1. Merge a `otel-config.yaml` file with the content of an environment variable `MY_OTHER_CONFIG` and use the merged result as the config:
     
    `./otelcorecol --config=file:examples/local/otel-config.yaml --config=env:MY_OTHER_CONFIG`

2. Merge a `config.yaml` file with the content of a yaml bytes configuration (overwrites the `exporters::debug::verbosity` config) and use the content as the config:

    `./otelcorecol --config=file:examples/local/otel-config.yaml --config="yaml:exporters::debug::verbosity: normal"`

### Embedding other configuration providers

One configuration provider can also make references to other config providers, like the following:

```yaml
receivers:
  otlp:
    protocols:
      grpc:

exporters: ${file:otlp-exporter.yaml}

service:
  extensions: [ ]
  pipelines:
    traces:
      receivers:  [ otlp ]
      processors: [  ]
      exporters:  [ otlp ]
```

## How to override config properties?

The `--set` flag allows to set arbitrary config property. The `--set` values are merged into the final configuration
after all the sources specified by the `--config` are resolved and merged.

### The Format and Limitations of `--set`

#### Simple property

The `--set` option takes always one key/value pair, and it is used like this: `--set key=value`. The YAML equivalent of that is:

```yaml
key: value
```

#### Complex nested keys

Use dot (`.`) in the pair's name as key separator to reference nested map values. For example, `--set outer.inner=value` is translated into this:

```yaml
outer:
  inner: value
```

#### Multiple values

To set multiple values specify multiple --set flags, so `--set a=b --set c=d` becomes:

```yaml
a: b
c: d
```


#### Array values

Arrays can be expressed by enclosing values in `[]`. For example, `--set "key=[a, b, c]"` translates to:

```yaml
key:
  - a
  - b
  - c
```

#### Map values

Maps can be expressed by enclosing values in `{}`. For example, `"--set "key={a: c}"` translates to:

```yaml
key:
  a: c
```

#### Limitations

1. Does not support setting a key that contains a dot `.`.
2. Does not support setting a key that contains a equal sign `=`.
3. The configuration key separator inside the value part of the property is "::". For example `--set "name={a::b: c}"` is equivalent with `--set name.a.b=c`.

## How to check components available in a distribution

Use the sub command build-info. Below is an example:

```bash
   ./otelcorecol components
```
Sample output:

```yaml
buildinfo:
   command: otelcorecol
   description: Local OpenTelemetry Collector binary, testing only.
   version: 0.62.1-dev
receivers:
   - otlp
processors:
   - memory_limiter
exporters:
   - otlp
   - otlphttp
   - debug
extensions:
   - zpages
```

## How to validate configuration file and return all errors without running collector

```bash
   ./otelcorecol validate --config=file:examples/local/otel-config.yaml
```

## How to examine the final configuration after resolving, parsing and validating?

Use `print-config` in the default mode (`--mode=redacted`) and `--feature-gates=otelcol.printInitialConfig`:

```bash
   ./otelcorecol print-config --config=file:examples/local/otel-config.yaml
```

Note that by default the configuration will only print when it is
valid, and that sensitive information will be redacted.  To print a
potentially invalid configuration, use `--validate=false`.

## How to examine the final configuration including sensitive fields?

Use `print-config` with `--mode=unredacted` and `--feature-gates=otelcol.printInitialConfig`:

```bash
   ./otelcorecol print-config --mode=unredacted --config=file:examples/local/otel-config.yaml
```

## How to print the final configuration in JSON format?

Use `print-config` with `--format=json` and
`--feature-gates=otelcol.printInitialConfig`. Note that JSON format is
considered unstable.

```bash
   ./otelcorecol print-config --format=json --config=file:examples/local/otel-config.yaml
```

