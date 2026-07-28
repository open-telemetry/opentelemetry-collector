# schemagen

`schemagen` is a tiny utility that walks a Go configuration file and emits
JSON Schema that mirrors the exported structs.

Script is only for temporary use. The aim is to facilitate the process of introducing schemas for
Open Telemetry Collector components. Ultimately, the configuration schemas would be maintained manually,
and the generation process would be reversed – Go config files from schemas.

> **Status:** In Development.

## Modes

Script can generate schemas for components and packages. There are two distinct modes of operation:
`component` and `package`. Schemagen automatically detects the mode based on `metadata.yaml` file content.

When the `-p` flag targets a package outside the current directory, schemagen resolves the target
package's source directory and reads its own `metadata.yaml` for mode detection. This works for any
package reachable from the `go.mod` of the directory you run schemagen from — including packages
from other repositories that are present in the module cache as dependencies. If the package is not
in the local module graph, auto-detection falls back to the default mode; use the `-m` flag to
override it explicitly in that case. The generated schema is always written to the local output
folder, not to the resolved package directory.

### Component mode

The `component` mode is aimed at generating configurations for individual components like receivers, processors,
exporters, and connectors.

In component mode, schema is generated in file named `config.schema.<ext>` to the chosen output folder
(by default it's input directory). You can optionally specify the root struct to use for schema generation
using the `-c` flag; if not provided, the tool defaults to using `Config`.

### Package mode

The `package` mode, on the other hand, is designed for generating schemas for entire packages. In this mode,
you provide the path to a directory containing multiple Go files that collectively define the configuration
for a package. This can be used to generate schemas for shared libraries or common configuration structures used across
multiple components.

In package mode every exported struct becomes a definition under `$defs` and the output file name is
`config.schema.<ext>`.

## Usage

Run schemagen via the `schemagen` Make target:

```bash
make schemagen SRC=path/to/your/package/dir
```

You can pass any CLI flags through the `FLAGS` variable:

```bash
make schemagen SRC=exporter/otlpexporter/ FLAGS="-o=$(pwd)/schemas -t=json -c=CustomConfig"
```

You can also go to specific component folder and run `make schemagen` directly.

## CLI options

| Flag | Default                                                                | Purpose                                                       |
|------|------------------------------------------------------------------------|---------------------------------------------------------------|
| `-c` | `Config` in component mode                                             | Explicitly set the struct that should become the root schema. |
| `-o` | Given directory (or `outputFolder` from `.schemagen.yaml` if provided) | Folder that will receive the generated `*.schema.<ext>` file. |
| `-t` | `yaml`                                                                 | Output format. Accepts `yaml`, `yml`, or `json`.              |
| `-r` | `false`                                                                | Resolve `$ref` entries inline (see [Ref resolution](#ref-resolution)). |
| `-p` | `.`                                                                    | Go package pattern passed to `packages.Load`. Use a fully-qualified import path (e.g. `go.opentelemetry.io/collector/receiver/otlpreceiver`) to target a specific package instead of the current directory. When non-default, schemagen resolves the target package's source directory and reads its `metadata.yaml` for mode detection. Requires the package to be reachable from the local `go.mod`. |
| `-m` | *(auto-detected)*                                                      | Override the run mode: `component` or `package`. Required when using `-p` with a package that is not in the local module graph, since `metadata.yaml` cannot be located automatically in that case. |

The schema `$id` is derived from the Go package import path and the `$title` is the package name with the current
run mode (`component`/`package`) appended.

## Settings file

The script looks for a `.schemagen.yaml` file starting from the current working directory and walking up to
the repository root. Although settings file is optional it extends significantly capabilities a `schemagen`.
A practical example:

```yaml
namespace: github.com/open-telemetry/opentelemetry-collector-contrib
mappings:
  time:
    Duration:
      schemaType: string
      format: duration
allowedRefs:
  - go.opentelemetry.io/collector
  - github.com/open-telemetry/opentelemetry-collector-contrib
componentOverrides:
  receiver/named_pipe:
    configName: 'NamedPipeConfig'
  receiver/file_log:
    configName: 'FileLogConfig'
  receiver/prometheus:
    overlayFile: receiver/prometheusreceiver/config.schema.overlay.yaml
```

- `namespace` corresponds to the Go package import path for modules from current repository.
  It is used to resolve references to other types within the same repository.
- `mappings` tell schemagen how to treat specific selector expressions as
  primitive schema fields. Each mapping converts the Go type into a scalar
  `schemaType` and can also set the JSON Schema `format`. The original
  Go type shows up under the `x-customType` extension so consumers can still
  see the source information.
- `allowedRefs` lists repositories that schemagen can make references to when
  generating `$ref` pointers. This is useful when your configuration structs
  embed types from other repositories that also have schemagen-generated
  schemas. If repo is not listed here, schemagen will use type `any`.
- `componentOverrides` allow per-component customization. Supported fields:
  - `configName` — overrides the root struct name (default: `Config`).
  - `factoryMaps` — expands factory-keyed map fields (see below).
  - `overlayFile` — path to a hand-curated YAML file deep-merged into the generated schema (see [Overlay files](#overlay-files)).

## Overlay files

Some configuration fields cannot be described by Go types alone — for example, a field that holds
an arbitrary Prometheus scrape config expressed as raw YAML. Overlay files let you inject
descriptions, constraints, or additional properties into the generated schema without modifying the
generator.

Set `overlayFile` in `.schemagen.yaml` for the target component to a path relative to the
repository root (or an absolute path):

```yaml
componentOverrides:
  receiver/prometheus:
    overlayFile: receiver/prometheusreceiver/config.schema.overlay.yaml
```

The file must be valid YAML that mirrors the structure of the generated schema. After generation,
schemagen deep-merges the overlay into the schema: map keys are merged recursively, scalar values
replace the generated ones. This means you can add or override `description`, `title`, `examples`,
or any other JSON Schema keyword on a per-field basis while leaving the rest of the schema intact.

```yaml
# config.schema.overlay.yaml — only the keys you want to change
properties:
  prom_config:
    description: "Prometheus scrape configuration, as defined by the Prometheus documentation."
    properties:
      scrape_configs:
        description: "List of scrape configurations."
```

The overlay is applied to both YAML and JSON output formats.

## Ref resolution

By default schemagen emits `$ref` pointers for types defined outside the current
package. When the `-r` flag is set, it instead walks every `$ref` in the output
and replaces it with the actual type definition, producing a fully self-contained
schema with no remaining references.

```bash
make schemagen SRC=receiver/icmpcheckreceiver FLAGS="-r -t=json"
```

Resolution supports three ref formats produced by the parser:

| Form | Example | Resolved as |
|------|---------|-------------|
| Bare local name | `inner_type` | Looked up in the schema's own `$defs` |
| Absolute-local path | `/receiver/otlpreceiver.Config` | Prefixed with `namespace` from `.schemagen.yaml` |
| Fully-qualified path | `github.com/some/pkg.Type` | Parsed from the referenced Go package |

Each package is loaded at most once; subsequent refs to the same package reuse
the cached result. References that cannot be resolved (missing type, unlisted
package) are dropped from the output with a log warning rather than aborting.

In `component` mode the `$defs` section is removed after resolution because all
types have been inlined.

> **Note:** Ref resolution runs `packages.Load` with full syntax and module
> information for every distinct external package. For components with many
> cross-package refs this can be slow.

## Generated schema highlights

- Structs become `type: object` definitions; nested structs are hoisted into
  `#/$defs/<TypeName>` and referenced where needed.
- Maps translate into `additionalProperties` with the schema generated for the
  map value. Slices keep their element type under `items`.
- Pointers compile down to the schema of the pointed type and carry an
  `x-pointer` marker for tooling.
- Anonymous embedded structs are added via `allOf` with the referenced `$defs`.
- Struct fields must expose a `mapstructure` tag; the tag value becomes the
  property name so only documented configuration knobs make it into the schema.
- Selectors that match entries under `mappings` produce
  primitive properties that retain the original Go type under `x-customType`.
- Optional wrapper types (e.g. `Optional[FooConfig]`) set the `x-optional`
  marker so downstream tooling can highlight nullable fields.

You can see end-to-end examples in `cmd/schemagen/internal/testdata`, e.g.
`SimpleConfig.go` → `simple_config.schema.json` or the more involved
`ComplexTypeFieldConfig.go`.

## Limitations

The current parser intentionally skips or errors on:

- Default values
- Validation
- Channels
- Function-typed fields
- Generic type parameters
- Interface fields without concrete type information
- Standard JSON Schema `required` arrays (beyond the `x-optional` marker)
- Exported fields without `mapstructure` tags (ignored)
