// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/schemagen"

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// jsonSchemaVersion is the JSON Schema draft version used.
	jsonSchemaVersion = "https://json-schema.org/draft/2020-12/schema"

	// durationPattern is a regex pattern matching Go's time.ParseDuration format.
	// Matches: "0", "10s", "1.5h", "1h30m", "-5m", etc.
	// Valid units: ns, us (or µs), ms, s, m, h
	// Uses actual Unicode characters for µ (micro sign U+00B5 and Greek mu U+03BC)
	durationPattern = `^(0|[-+]?((\d+(\.\d*)?|\.\d+)(ns|us|µs|μs|ms|s|m|h))+)$`
)

// SchemaGenerator generates JSON schemas from Go struct information.
type SchemaGenerator struct {
	outputDir string
	analyzer  *PackageAnalyzer
}

// NewSchemaGenerator creates a new SchemaGenerator.
func NewSchemaGenerator(outputDir string, analyzer *PackageAnalyzer) *SchemaGenerator {
	return &SchemaGenerator{
		outputDir: outputDir,
		analyzer:  analyzer,
	}
}

// GenerateSchema generates a YAML schema for the component's config.
// configTypeName is the name of the config type (e.g., "Config").
// configPkgPath is the package path where the config is defined (empty for local package).
func (g *SchemaGenerator) GenerateSchema(componentKind, componentName, configTypeName, configPkgPath string) error {
	structInfo, err := g.analyzer.analyzeConfig(configTypeName, configPkgPath)
	if err != nil {
		return fmt.Errorf("failed to analyze config: %w", err)
	}

	schema := g.structToSchema(structInfo, componentKind, componentName)

	// Ensure output directory exists
	if err = os.MkdirAll(g.outputDir, 0o700); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write schema to file
	outputPath := filepath.Join(g.outputDir, "config_schema.yaml")
	data, err := yaml.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write schema: %w", err)
	}

	return nil
}

// structToSchema converts a StructInfo to a JSON Schema.
func (g *SchemaGenerator) structToSchema(info *StructInfo, componentKind, componentName string) *Schema {
	schema := &Schema{
		Schema:      jsonSchemaVersion,
		Title:       fmt.Sprintf("%s %s configuration", componentName, componentKind),
		Description: info.Description,
		Type:        "object",
		Properties:  make(map[string]*Schema),
	}

	for _, field := range info.Fields {
		propSchema := g.fieldToSchema(&field)
		if propSchema != nil {
			if field.Embedded {
				// For embedded structs, merge properties into parent
				maps.Copy(schema.Properties, propSchema.Properties)
			} else {
				schema.Properties[field.JSONName] = propSchema
			}
		}
	}

	return schema
}

// fieldToSchema converts a FieldInfo to a JSON Schema property.
func (g *SchemaGenerator) fieldToSchema(field *FieldInfo) *Schema {
	schema := &Schema{
		Description: field.Description,
	}

	// Handle embedded structs
	if field.Embedded && len(field.Fields) > 0 {
		schema.Type = "object"
		schema.Properties = make(map[string]*Schema)
		for _, f := range field.Fields {
			propSchema := g.fieldToSchema(&f)
			if propSchema != nil {
				if f.Embedded {
					// For embedded/squashed fields, merge their properties into parent
					maps.Copy(schema.Properties, propSchema.Properties)
				} else {
					schema.Properties[f.JSONName] = propSchema
				}
			}
		}
		return schema
	}

	// Convert Go type to JSON Schema type
	g.setSchemaType(schema, field.Type)

	// Handle nested struct fields
	if len(field.Fields) > 0 {
		if schema.Type == "object" {
			schema.Properties = make(map[string]*Schema)
			for _, f := range field.Fields {
				propSchema := g.fieldToSchema(&f)
				if propSchema != nil {
					if f.Embedded {
						// For embedded/squashed fields, merge their properties into parent
						maps.Copy(schema.Properties, propSchema.Properties)
					} else {
						schema.Properties[f.JSONName] = propSchema
					}
				}
			}
		} else if schema.Type == "array" && schema.Items != nil && schema.Items.Type == "object" {
			// For arrays of structs, populate items properties
			schema.Items.Properties = make(map[string]*Schema)
			for _, f := range field.Fields {
				propSchema := g.fieldToSchema(&f)
				if propSchema != nil {
					if f.Embedded {
						// For embedded/squashed fields, merge their properties into parent
						maps.Copy(schema.Items.Properties, propSchema.Properties)
					} else {
						schema.Items.Properties[f.JSONName] = propSchema
					}
				}
			}
		}
	}

	return schema
}

// setSchemaType sets the JSON Schema type based on Go type.
func (g *SchemaGenerator) setSchemaType(schema *Schema, goType string) {
	// Remove package prefix for easier matching
	typeName := goType
	if idx := strings.LastIndex(typeName, "."); idx != -1 {
		typeName = typeName[idx+1:]
	}

	// Handle pointer types
	if after, found := strings.CutPrefix(goType, "*"); found {
		g.setSchemaType(schema, after)
		return
	}

	// Handle opaque string types (e.g., configopaque.String) - treat as string
	if strings.HasSuffix(typeName, "String") && strings.Contains(goType, "opaque") {
		schema.Type = "string"
		return
	}

	// Handle Optional[T] generic types - unwrap the inner type
	if strings.Contains(goType, ".Optional[") {
		innerType := extractOptionalInnerType(goType)
		if innerType != "" {
			g.setSchemaType(schema, innerType)
		}
		return
	}

	// Handle slice types
	if after, found := strings.CutPrefix(goType, "[]"); found {
		schema.Type = "array"
		itemSchema := &Schema{}
		g.setSchemaType(itemSchema, after)
		schema.Items = itemSchema
		return
	}

	// Handle map types
	if strings.HasPrefix(goType, "map[") {
		schema.Type = "object"
		// Extract value type from map[string]ValueType
		if _, valueType, found := strings.Cut(goType, "]"); found && valueType != "" {
			addProps := &Schema{}
			g.setSchemaType(addProps, valueType)
			schema.AdditionalProperties = addProps
		}
		return
	}

	// Handle common types
	switch typeName {
	case "string":
		schema.Type = "string"
	case "bool":
		schema.Type = "boolean"
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		schema.Type = "integer"
	case "float32", "float64":
		schema.Type = "number"
	case "Duration":
		// time.Duration uses Go's duration format (e.g., "1h30m10s"), not ISO 8601
		schema.Type = "string"
		schema.Pattern = durationPattern
	case "Time":
		schema.Type = "string"
		schema.Format = "date-time"
	case "ID", "Type":
		// component.ID and component.Type are represented as strings in config
		schema.Type = "string"
	case "interface{}", "any":
		// No specific type constraint
		schema.Type = ""
	default:
		// Default to object for complex types
		schema.Type = "object"
	}
}

// extractOptionalInnerType extracts the inner type from configoptional.Optional[T].
func extractOptionalInnerType(goType string) string {
	// Find the start of Optional[
	start := strings.Index(goType, "Optional[")
	if start == -1 {
		return ""
	}
	start += len("Optional[")

	// Find matching closing bracket
	depth := 1
	for i := start; i < len(goType); i++ {
		switch goType[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return goType[start:i]
			}
		}
	}
	return ""
}
