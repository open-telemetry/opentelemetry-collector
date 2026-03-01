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
  type: object
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
            default: 4
          ping_interval:
            type: string
			description: "Interval between pings"
            format: duration
            default: 300ms
          ping_timeout:
            type: string
			description: "Timeout of ping request"
            format: duration
            default: 4s
      required: [ host ]
      minItems: 1
    custom_field:
      go:
        custom: ['type', 'default', 'validate']
  required: [ targets ]
  allOf:
    - $ref: 'go.opentelemetry.io/collector/scraper/scraperhelper.controller_config'
    - $ref: './internal/metadata.metrics_builder_config' 
```

#### Schema specification details

- **format** of config schema follows roughly JSON Schema specification (https://json-schema.org/)
- **special types** like time.Duration or time.Time can be represented as strings with specific formats (e.g., "duration", "date-time").
- **dependencies** are specified with `$ref` attribute to reference other schema definitions, either from external packages or internal definitions.
  To identify references we need full package paths (for external) or use relative path (for internal):
    - `go.opentelemetry.io/collector/scraper/scraperhelper.controller_config` would be automatically generated by mdatagen with the same process used to generate the go structs and documentation today.
    - `./internal/metadata.metrics_builder_config` would be generated by `mdatagen` from metrics definition in the scraperhelper component.
    - dependencies is from external packages not covered with schemas will be replaced by `any` type with actual type annotation.
- **default values** can be specified using the `default` attribute.
- **validation** is possible using standard JSON Schema attributes, see [docs](https://json-schema.org/draft/2020-12/json-schema-validation).

#### Extensibility

The YAML schema specification can be extended with custom annotations to capture domain-specific types and validation rules that are not natively supported in JSON schema. Additionally, we may introduce custom fields that generate fields that will produce references to structs or validation functions that require more complex logic and manual implementation.

One proposal is to have `go` annotation that could contain the following sub-attributes:
- type [string] - allows specifying a custom Go type for the field in a format `<package>.<type_name>`.
- pointer [bool] - if set to true, the generated field will be a pointer to the specified type.
- optional [bool] - if set to true, the generated field will be wrapped in Optional[...] generic type.
- custom [list of strings] - allows specifying custom code snippets to be injected into the generated code.
  The possible values are: `type`, `default`, `validate`.
  Example:
  ```yaml
  some_optional_field:
   type: string
   go:
     type: com.github/custom/custompkg.CustomType
     pointer: true
     optional: true
     custom: ['validate']
  ``` 
  
### Generated Go code example

See below for an example of the generated Go code from the above schema:

```go
package icmpcheckreceiver

import (
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/scraper/scraperhelper"
	"go.uber.org/multierr"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/icmpcheckreceiver/internal/metadata"
)

type Config struct {
	scraperhelper.ControllerConfig `mapstructure:",squash"`
	metadata.MetricsBuilderConfig  `mapstructure:",squash"`
	Targets                        []TargetsItemConfig `mapstructure:"targets"`
	CustomField                    CustomFieldType     `mapstructure:"custom_field"`
}

// subtype for targets item generated from schema
type TargetsItemConfig struct {
	Host         string        `mapstructure:"host"`
	PingCount    int           `mapstructure:"ping_count"`
	PingInterval time.Duration `mapstructure:"ping_interval"`
	PingTimeout  time.Duration `mapstructure:"ping_timeout"`
}

func (c *Config) Validate() error {
	var err error

	if len(c.Targets) == 0 {
		return multierr.Append(err, errMissingTarget)
	}

	for _, target := range c.Targets {
		if target.Host == "" {
			err = multierr.Append(err, errMissingTargetHost)
		}
	}
	
	err = multierr.Append(err, ValidateCustomField(c.CustomField))

	return err
}

func createDefaultConfig() component.Config {
	return &Config{
		// default values derived from schema
		CustomField: DefaultCustomFieldValue(),
	}
}

```

#### Generated code details

- The generated `config.go` file will define Go structs that mirror the configuration schema. It's capable
  of handling nested objects and arrays and creating subtypes with auto-generated names.
- The `Validate` method will include validation logic based on the schema (e.g., required fields).
- A `createDefaultConfig` function will be generated to provide default values for the configuration.

#### Customization

Because schema defined field with custom type, validation and default value called `custom_field`, generated code will refer to placeholders that must be implemented in component's Go package. 
This creates an implementation hook for component developers to add the custom logic.

In the above example:
1. `CustomFieldType` - a Go type for a `custom_field`
2. `ValidateCustomField` - a function that provides validation of `custom_field` values
3. `DefaultCustomFieldValue` - a function that returns default value(s) for `custom_field`

### Generated JSON Schema example

This file will be a direct translation of the `config` section from `metadata.yaml` into JSON Schema format.

Example:
```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/icmpcheckreceiver",
  "title": "receiver/icmpcheck component",
  "$defs": {
    "metadata.metric_config": {
      "description": "MetricConfig provides common config for a particular metric.",
      "type": "object",
      "properties": {
        "enabled": {
          "type": "boolean"
        }
      }
    },
    "metadata.metrics_builder_config": {
      "description": "MetricsBuilderConfig is a configuration for icmpcheckreceiver metrics builder.",
      "type": "object",
      "properties": {
        "metrics": {
          "$ref": "#/$defs/metadata.MetricsConfig"
        },
        "resource_attributes": {
          "$ref": "#/$defs/metadata.ResourceAttributesConfig"
        }
      }
    },
    "metadata.metrics_config": {
      "description": "MetricsConfig provides config for icmpcheckreceiver metrics.",
      "type": "object",
      "properties": { ... }
    },
    "metadata.resource_attribute_config": {
      "description": "ResourceAttributeConfig provides common config for a particular resource attribute.",
      "type": "object",
      "properties": { ... }
    },
    "metadata.resource_attributes_config": { ... },
    "scraperhelper.controller_config": {
      "description": "ControllerConfig defines common settings for a scraper controller configuration. Scraper controller receivers can embed this struct, instead of receiver.Settings, and extend it with more fields if needed.",
      "type": "object",
      "properties": {
        "collection_interval": {
          "description": "CollectionInterval sets how frequently the scraper should be called and used as the context timeout to ensure that scrapers don't exceed the interval.",
          "type": "string",
          "pattern": "^(?:[0-9]+(\\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$",
          "default": "60s"
        },
        "initial_delay": {
          "description": "InitialDelay sets the initial start delay for the scraper, any non positive value is assumed to be immediately.",
          "type": "string",
          "pattern": "^(?:[0-9]+(\\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$",
          "default": "1s"
        },
        "timeout": {
          "description": "Timeout is an optional value used to set scraper's context deadline.",
          "type": "string",
          "pattern": "^(?:[0-9]+(\\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$",
          "default": "0s"
        }
      }
    }
  },
  "type": "object",
  "properties": {
    "targets": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "host": {
            "type": "string"
          },
          "ping_count": {
            "type": "integer",
            "default": 4
          },
          "ping_interval": {
            "type": "string",
            "pattern": "^(?:[0-9]+(\\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$",
            "default": "300ms"
          },
          "ping_timeout": {
            "type": "string",
            "pattern": "^(?:[0-9]+(\\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$",
            "default": "4s"
          }
        },
        "required": [
          "host"
        ],
        "minItems": 1
      },
      "required": [
        "targets"
      ]
    },
    "custom_field": {
      "type": {} // any
    }
  },
  "allOf": [
    {
      "$ref": "#/$defs/scraperhelper.controller_config"
    },
    {
      "$ref": "#/$defs/metadata.metrics_builder_config"
    }
  ]
}
```

#### Generated JSON Schema details

- The `config.schema.json` file will be a direct representation of the `config` section from `metadata.yaml`,
  with proper handling of `$ref` references to include definitions from external packages and internal definitions
  in a single JSON Schema file.
- Each referenced schema will be embedded inline.
- The `$id` field will be set to the component's import path for easy identification.
- The custom types will be represented as `any` type in JSON Schema.


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

#### Phase 4: Extend OCB

**Objective**: Use OCB to bind all components schemas to single schema that describes entire collector config file

**Success Criteria:**
- Generated schema for collector config file includes all components
- Schema can be used by internal and external tools for validation and autocompletion
