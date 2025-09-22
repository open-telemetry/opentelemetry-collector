# Configuration merging

## Background

As part of issue [#8754](https://github.com/open-telemetry/opentelemetry-collector/issues/8754), a new feature gate has been introduced to support merging component lists instead of replacing them ([first PR](https://github.com/open-telemetry/opentelemetry-collector/pull/12097)). This enhancement enables configurations from multiple sources to be combined, preserving all defined components in the final configuration.

More information about this feature can be found in the [confmap README](https://github.com/open-telemetry/opentelemetry-collector/blob/d4539dd6b4e554e15066226fa975b156af7b1510/confmap/README.md#experimental-append-merging-strategy-for-lists).

The main motivation for this change was to allow users to define configuration fragments in different sources, and have them merged in such a way that all specified components are included under `service::pipeline` in the final configuration.

Previously, we relied on Koanf’s default merging strategy, which overrides static values and slices (such as strings, numbers, and lists). This behavior often resulted in configurations being unintentionally overwritten when merged from multiple sources.

This issue has been highlighted in several discussions and feature requests:
- https://github.com/open-telemetry/opentelemetry-collector/issues/8394
- https://github.com/open-telemetry/opentelemetry-collector/issues/8754
- https://github.com/open-telemetry/opentelemetry-collector/issues/10370

## Motivation and Scope

We’ve already implemented a feature gate and foundational logic that supports merging lists across configuration files. Currently, this logic is hardcoded to merge lists only for specific keys: receivers, exporters, and extensions. The relevant implementation can be found [here](https://github.com/open-telemetry/opentelemetry-collector/blob/main/confmap/internal/merge.go).

The current implementation lacks flexibility. Ideally, users should be able to specify which configuration paths should be merged, rather than relying on hardcoded defaults. 
This RFC proposes extending the existing functionality by introducing a user-configurable mechanism to define merge behavior.

This RFC builds on top of feedback gathered from the [original PR](https://github.com/open-telemetry/opentelemetry-collector/pull/12097).
More specifically, this RFC aims to:
1. Add an option to specify which configuration paths should be merged.
2. Introduce support for prepend and append operations when merging list values:
    - This is good to have for lists that rely on certain ordering, such as:
        - `processors`
        - `transformprocessor` statements

## Proposed approaches:

### Approach 1 (Recommended): Use yaml tags

The first approach relies on the concept of [yaml tags](https://tutorialreference.com/yaml/yaml-tags). We can specify a custom tag in our configuration files to indicate the lists we want to merge.

Consider following two configurations:

```yaml
#main.yaml
receivers:
    ...
exporters:
    ...
extensions:
    extension1:

service:
  extensions: [extension1]
  pipelines:
    logs:
      receivers: [...]
      exporters: [...]
```

```yaml
#extra.yaml
extensions:
    extension2:

service:
  extensions: !mode=append [extension2]
```

After running the collector with above configurations, the `service::extensions` path will get merged and final configuration will look like:

```yaml
#final.yaml
receivers:
    ...
exporters:
    ...
extensions:
    extension1:
    extension2:

service:
  extensions: [extension1, extension2]
  pipelines:
    logs:
      receivers: [...]
      exporters: [...]
```

This approach completely relies on the configuration files and doesn't introduce any command line option.
Internally, we will loop through the yaml tree and fetch the paths we want to merged based on user-defined tags, then merge the lists found at those paths.

Our "custom" yaml tag will be in the format of URI query parameters. For starters, we can support following options:
1. `mode`:
    - This setting will control the ordering of merged list.
    - The value will be either of `append` or `prepend`.
    - Default: `append`
2. `duplicates`:
    - This setting controls the duplication of elements in the final list. 
    - If the user wants to allow duplicates, they can simply turn this flag on.
    - Default: `false`
3. `recursive`:
    - This setting controls the merging of child elements of the given yaml path. 
    - This is useful if user wants to merge all the lists under a give sub-tree. 
        - For eg. merging all the lists under the `service` section.

#### Examples of first approach

1. _Merge the `service::extensions` list_:

```yaml
#main.yaml
receivers:
    ...
exporters:
    ...
extensions:
    extension1:

service:
  extensions: [extension1]
  pipelines:
    logs:
      receivers: [...]
      exporters: [...]
      
---

#extra.yaml
extensions:
    extension2:

service:
  extensions: !mode=append [extension2]
```

```yaml
#final.yaml
receivers:
    ...
exporters:
    ...
extensions:
    extension1:
    extension2:

service:
  extensions: [extension1, extension2]
  pipelines:
    logs:
      receivers: [...]
      exporters: [...]
```

2. _Merge all the lists under the `service::*` section_:


```yaml
#main.yaml
receivers:
    ...
exporters:
    ...
extensions:
    extension1:

service:
  extensions: [extension1]
  pipelines:
    logs:
      receivers: [...]
      exporters: [...]
      
---

#extra.yaml
extensions:
    extension2:
receiver:
    receiver2:

service: !mode=append&recursive=true
  extensions: [extension2]
  pipelines:
    logs:
        receivers: [receiver2]
```

```yaml
#final.yaml
receivers:
    ...
    receiver2:
exporters:
    ...
extensions:
    extension1:
    extension2:

service:
  extensions: [extension1, extension2]
  pipelines:
    logs:
      receivers: [..., receiver2]
      exporters: [...]
```

### Approach 2: URI params

The proposed approach will rely on concept of URI query parameters([_RFC 3986_](https://datatracker.ietf.org/doc/html/rfc3986#page-23)). Our configuration URIs already adhere to this syntax and we can extend it to support query params instead adding new CLI flags. 

For now, the new merging strategy is only enabled under `confmap.enableMergeAppendOption` gate. If user specifies the options and tries to run the collector without gate, we will merge as per default behaviour.

We will support new parameters to config URIs as follows:
1. `merge_paths`: A comma-separated list of glob patterns which will be used while config merging
    - This setting will control the paths user wants to merge from the given config.
    - Example: 
        - `otelcol --config main.yaml --config extra.yaml?merge_paths=service::extensions,service::**::receivers`
            - In this example, we will merge the list of extensions and receivers from pipeline, excluding lists in the rest of the config.
        - `otelcol --config main.yaml --config ext.yaml?merge_paths=service::extensions --config rec.yaml?merge_paths=service::**::receivers`
            - In this example, we will merge all list of extensions from `ext.yml` and list of receivers from `rec.yaml`, excluding lists in the rest of the config.
2. `merge_mode`: One of `prepend` or `append`.
    - This setting will control the ordering of merged list.

#### Examples of second approach

Here are some examples:

1. _Append to default mergeable components_:
```bash
otelcol --config=main.yaml --config=extra_components.yaml?merge_mode=append --feature-gates=confmap.enableMergeAppendOption
```

- After running above command, the final configuration will include:
    - Merged component(s) (`receivers`, `exporters` and `extensions`) from `extra_components.yaml`

2. _Specify exact paths for merging_:
```bash
otelcol \
  --config=main.yaml \
  --config=extra_extension.yaml?merge_mode=append&merge_paths=service::extensions \
  --config=extra_receiver.yaml?merge_mode=append&merge_paths=service::**::receivers \
  --feature-gates=confmap.enableMergeAppendOption
```

- After running above command, the final configuration will include:
    - Merged extension(s) from `extra_extension.yaml`
    - Merged receiver(s) from `extra_receiver.yaml`


3. _Prepend processors_:
```bash
otelcol --config=main.yaml --config=extra_processor.yaml?merge_mode=prepend&merge_paths=service::**::processors --feature-gates=confmap.enableMergeAppendOption
```

- After running above command, the final configuration will include:
    - Merged processor(s) from `extra_processor.yaml`, but prepend the existing list.

4. _Exclude a config file from lists merging process_:
```bash
otelcol --config=main.yaml --config=extra_components.yaml?merge_mode=append --config override_components.yaml --feature-gates=confmap.enableMergeAppendOption
```

- In the above command, we have no specified any options for `override_components.yaml`. Hence, it will override all the conflicting lists from previous configuration, which is the default behaviour.

## Open questions

- What to do if an invalid option is provided for `merge_mode` or `merge_paths`?
    - I can think of two possibilities:
        1. Error out.
        2. Log an error and merge the default way
- What to do if an invalid query param is provided in config URI?
    - In this case, I strongly feel that we should error out. 

## Extensibility 

This URI-based approach is highly extensible. In the future, it can enable advanced operations such as map overriding. Currently, it's impossible to do so.