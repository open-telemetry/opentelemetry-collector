# Component Configuration Management Roadmap

## Motivation

The OpenTelemetry Collector ecosystem lacks a unified approach to configuration management, leading to several problems:

1. **Documentation Drift**: Go configuration structs and documentation exist independently and frequently diverge over time
2. **Inconsistent Developer Experience**: No standardized patterns for defining component configurations
3. **No config validation capabilities**: Lack of JSON schemas prevents autocompletion and validation in configuration editors

## Current state

- Go configuration structs in each component with validation implemented via custom code and defaults set in `setDefaultConfig` functions
- Manual documentation that often becomes outdated
- No standardized JSON schemas for configuration validation

## Desired state

**Goal**: Establish a single source of truth for component configuration that generates:
1. **Go configuration structs** with proper mapstructure tags, validation, and default values. 
2. **JSON schemas** for configuration validation and editor autocompletion
3. **Documentation** that stays automatically synchronized with implementation

## Previous and current approaches

### Past attempts

- [Previously available contrib configschema tool](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.102.0/cmd/configschema): Retired due to incompleteness, complexity and maintenance burden. It required dynamic analysis of Go code and pulling all dependencies.

- [PR #27003](https://github.com/open-telemetry/opentelemetry-collector-contrib/pull/27003): Failed due to trying to cover all corner-cases in the design phase instead of quickly iterating from a simpler approach.

- [PR #10694](https://github.com/open-telemetry/opentelemetry-collector/pull/10694): An attempt to generate config structs from the schema defined in metadata.yaml using github.com/atombender/go-jsonschema. It faced some limitations of the library. However, it was abandoned mostly due to a lack of involvement from the reviewers.

### Current initiatives

- [opentelemetry-collector-contrib/cmd/schemagen/](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/cmd/schemagen): Generates JSON schemas from Go structs with limited support for validation and default values. It uses AST parsing with module-aware loading of dependencies to handle shared libraries.

- [PR #14288](https://github.com/open-telemetry/opentelemetry-collector/pull/14288): Also uses AST parsing to generate JSON schemas from Go structs for the component configurations without using shared config support. Written as part of mdatagen tool.

Parsing Go code to generate schemas is inherently limited. Community consensus recommends reversing the process: generate Go code from schemas instead. There is already widely established practice in other ecosystems to generate go code and documentation for other parts of the OTel Collector.

## Suggested approach

### Overview

This RFC proposes an approach that transitions from the current Go-struct-first model to a schema-first configuration generation system:

1. **Bootstrap Phase**: Use existing `schemagen` tool to generate initial schema specifications
2. **Tool Development Phase**: Create new tooling that generates Go structs, JSON schemas, and documentation from YAML schema specifications 
3. **Migration Phase**: Migrate all components to the new schema-first approach

Use of the `schemagen` tool is dictated by the modularity of the Collector components. It allows generating schemas for shared libraries (e.g., scraperhelper) that can be referenced by individual components.

### Reasoning behind this approach

- **Explicit validation**: Schema specifications can explicitly capture validation rules and default config values that cannot be extracted from Go code
- **Rich documentation**: Schemas can include descriptions, examples, and constraints that enhance generated documentation
- **Simplified tooling**: Template-based code generation is more predictable than AST parsing

**Why YAML schema format for the source of truth?**
- **Human-readable**: Easier for component developers to author and maintain than JSON
- **Integration with existing infrastructure**: Natural extension of `metadata.yaml` approach used by `mdatagen` given that it already uses YAML to generate metrics builder configs
- **Extensibility**: YAML allows for custom fields to capture domain-specific configuration and provide escape-hatches to generate config fields that still require custom implementation, validation or default value setters.

### Example schema format

```yaml
config:
  allOf:
    - $ref: "go.opentelemetry.io/collector/scraper/scraperhelper#/$defs/ControllerConfig"
    - $ref: "#/$defs/MetricsBuilderConfig"
  properties:
    targets:
      type: array
      items:
        type: object
        properties:
          host:
            type: string
            description: "Target hostname or IP address"
          ping_count:
            type: integer
            description: "Number of pings to send"
            default: 3
          ping_interval:
            type: string
            format: duration
            x-customType: "time.Duration"
            description: "Interval between pings"
            default: "1s"
        required: ["host"]
```

`#/$defs/MetricsBuilderConfig` would be automatically generated by mdatagen with the same process used to generate the go structs and documentation today.

`go.opentelemetry.io/collector/scraper/scraperhelper#/$defs/ControllerConfig` would be generated by the new tool from the schema definition in the scraperhelper component.

#### Extensibility

The YAML schema specification can be extended with custom fields (e.g., `x-customType`) to capture domain-specific types and validation rules that are not natively supported in JSON schema. Additionally, we may introduce custom fields that generate fields that will produce references to structs or validation functions that require more complex logic and manual implementation.

### Roadmap

#### Phase 1: Bootstrap initial schemas

**Objective**: Use `schemagen` tool to generate initial schema specifications for all components

**Success Criteria:**
- YAML schemas generated for all components in core and contrib repositories 
- Setup CI check to ensure schemas remain up-to-date with Go structs

#### Phase 2: Implement new generation tool

**Objective**: Implement a new tool that takes YAML schema from the user and generates Go structs, combined JSON schema, and documentation per component.

**Success Criteria:**
- New tool generates Go structs that are API-compatible with existing implementations with the following features:
  - Parses YAML schema specifications
  - Generates Go configuration structs with proper validation
  - Produces JSON schemas for config validation
  - Creates synchronized documentation
- Generated JSON schemas pass validation tests with real collector configurations
- Generated documentation accurately reflects all configuration options
- Pilot components successfully replace hand-written implementations

If existing config structs don't follow the established naming patterns produced by the generated code, the implementation may allow breaking the Go API compatibility in favor of consistent Go API naming standards and long-term maintainability. However, the configuration file format MUST remain compatible for end users.

#### Phase 3: Migrate all components

**Objective**: Migrate all components to the new tool introduced in Phase 2

**Success Criteria:**
- All core and contrib components migrated to schema-first approach
- All new components use schema-first tooling by default
