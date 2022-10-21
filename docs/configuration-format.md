# OpenTelemetry Collector Configuration Format

The effective configuration that defines runtime behavior of the collector is calculated from 2 sources: the Default configuration which is hard-coded and the User-defined configuration which supplied by the user via various configuration sources. Default configuration defines reasonable defaults that most users will likely want to use. User-defined configuration can override the defaults and specify additional configuration. The effective configuration is calculated by merging Default and User-defined configurations (see merging rules in a separate section below).

The top-level entities in the configuration file are receivers, processors, exporters, pipeline (more about each below in separate sections). Here is the top-level structure of the configuration:

```yaml
receivers:
  # map of receivers
processors:
  # map of processors
exporters:
  # map of exporters
extensions:
  # map of extensions
pipelines:
  # map of pipelines
```

## Receivers

Each receiver has a name which is defined by the key of the mapping under receivers top level-key. The name should be in the form `type[/name]`, where type is the type of the receiver (e.g. `otlp`), and `name` suffix is optionally appended after a forward slash to ensure uniqueness of the full name of the receiver. It is not allowed to define more than one receiver with the same name.

If more than one receiver is defined that listens on the same port it is treated as a fatal configuration error (note: it is valid for 2 receivers to listen on the same port number if they bind to different network interfaces).

More than one pipeline can be associated with the same receiver definition. In that case the data from that receiver is copied (fanned out) to all associated pipelines. 
The definition of receiver is specific for each type however at the minimum it will include a port number and the network interface address to bind to (defaults to `127.0.0.1`).

There are 2 possible configuration structure for a receiver. First structure is for receivers that support only one protocol:

```yaml
receivers:
  <receiver name>:
    endpoint: <network interface and port to bind to, address:port>
    enabled: <boolean, defaults to true>
    # other key/value pairs as needed by specific receiver type
```

For receivers that support more than one protocol the structure is the following:

```yaml
receivers:
  <receiver name>:
    protocols:
      <protocol name 1>: # key is string, protocol name, unique
        endpoint: <network interface and port to bind to, address:port>
        enabled: <boolean, defaults to true>
        # other key/value pairs as needed by specific receiver type
      <protocol name 2>:
        # settings for protocol 2
      ...
      <protocol name N>:
        # settings for protocol N
```

There can be one or more `protocols` mappings, for example:

```yaml
receivers:
  jaeger/external:
    protocols:
      thrift-tchannel:
        endpoint: "127.0.0.1:14267”
      thrift-http:
        endpoint: "127.0.0.1:14268”
```

## Processors

The configuration structure of a top-level processor is the following:

```yaml
processors:
  <processor name>: # key is string, unique name of processor
    enabled: <boolean, defaults to true>
    # other key/value pairs as needed by specific processor type
```

The name should be in the form `type[/name]`, where `type` is the type of the processor (e.g. `batching`), and `name` suffix is optionally appended after a forward slash to ensure uniqueness of the full name. It is not allowed to define more than one processor with the same name.

## Exporters

Each exporter has a name which is defined by the key of the mapping under exporters top level-key. The name should be in the form `type[/name]`, where type is the type of the exporter (e.g. `otlp`), and `name` suffix is optionally appended after a forward slash to ensure uniqueness of the full name of the exporter. It is not allowed to define more than one exporter with the same name.

Multiple pipelines can be associated with the same exporter. When this happens the data processed by these pipelines is directed to that exporter for further delivery.
The definition of exporter is specific for each type however at the minimum it will include an endpoint string (which normally includes the destination address and port).
The structure of exporter configuration is the following:

```yaml
<exporter name>: # key is string, unique name of exporter
  endpoint: <network interface and port to bind to, address:port>
  enabled: <boolean, defaults to true>
  # other key/value pairs as needed by specific exporter type
```

The format and interpretation of `endpoint` is exporter specific but typically is a string in the form `address:port`.

## Pipelines

The effective configuration specifies one or more Pipelines. A pipeline defines how data is received (via `receivers`), processed (via `processors`) and exported (via `exporters`).

A pipeline can be one of the 3 types: `traces`, `metrics`, or `logs`. For a pipeline, `receivers` and `exporters` must be present for pipeline to be complete.

At least one complete pipeline must be defined otherwise the configuration is not valid.

Each pipeline has a name that must be unique within the set of pipelines of the same type (note: it is allowed to have same-named pipelines if they are of different types).
Note that the configuration may specify receivers, processors and exporters which are not used (referenced) in any pipeline. This makes those receivers, processors and exporters inactive.

Here is the configuration structure for a pipeline:

```yaml
<pipeline name>: # key is string, unique name of pipeline
  receivers: [receiver-name-1, receiver-name-2, ...]
  processors: [processor-name-1, processor-name-2, ...]
  exporters: [exporter-name-1, exporter-name-2, ...]
```

The name should be in the form `type[/name]`, where type is the input type of the pipeline (either traces, metrics, or logs), and name suffix is optionally appended after a forward slash to ensure uniqueness of the full name of the pipeline. It is not allowed to define more than one pipeline with the same name.

## Pipeline Processors

Pipeline `processors` define what processors will be performed on data after it is received. Each element in ordered-processors list references a processor that is defined in the `processors` section. The order in which processors are listed in `processors` is significant.

## Enabling/Disabling Sections

Each section of configuration can be enabled or disable via `enabled` key. If the effective value of enabled key is false, the entire section with all of its key/value pairs and all child sections is ignored as if it did not exist in the effective configuration.

If `enabled` key is not present in the section the section is assumed to be enabled (the default value for `enabled` key is `false`).

The Default configuration contains several defined sections with `enabled:false` and other key/value pairs defined that specify the default behavior for that entity. This allows the user to easily enable the entire section by simply specifying `enabled:true` for the corresponding section in the user-defined configuration, this also makes the other key/value pairs effective (and they can overridden like any key/value pair the Default).

## Merging Rules

Default and User-defined configurations are merged at startup and form the Effective configuration that defines the runtime behavior. Merging is performed by recursively applying the following rules to each section:

| Default | User-defined | Effective |
| --| --| -- |
| `processors:`<br/>`  batch:`<br/>`    wait: 30`<br/>`    enabled: true` | `receivers:`<br/>`  jaeger:`<br/>`    protocol:`<br/>`      http:` | `processors:`<br/>`  batch:`<br/>`    wait: 30`<br/>`    enabled: true`<br/>`receivers:`<br/>`  jaeger:`<br/>`    protocol:`<br/>`      http:`<br/> |

If a mapping is present in both Default and User-defined then the effective configuration will contain a mapping of key/values that is merged recursively according to these rules:

For key/value present in either of Default or User-defined the effective configuration will contain that key/value. Example:

| Default | User-defined | Effective |
| --| --| -- |
| `type: tags` <br/> `enabled: false`| `type: tags` <br/> `overwrite: true` | `type: tags` <br/> `overwrite: true` <br/> `enabled: false`|


For key/value present in both Default and User-defined the effective configuration will contain that key/value from User-defined and the key/value from Default will be ignored. Example:

| Default | User-defined | Effective |
| --| --| -- |
| `type: tags` <br/> `overwrite: false` <br/> `enabled: false`| `type: tags` <br/> `overwrite: true` <br/> `block: 5` | `type: tags` <br/> `overwrite: true` <br/> `enabled: false`<br/> `block: 5`|

## Reference

Original document: https://docs.google.com/document/d/1NeheFG7DmcUYo_h2vLtNRlia9x5wOJMlV4QKEK05FhQ/edit#
