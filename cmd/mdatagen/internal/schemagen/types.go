// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/schemagen"

// Schema represents a JSON Schema document (draft-07).
type Schema struct {
	Schema               string             `yaml:"$schema,omitempty"`
	ID                   string             `yaml:"$id,omitempty"`
	Title                string             `yaml:"title,omitempty"`
	Description          string             `yaml:"description,omitempty"`
	Type                 string             `yaml:"type,omitempty"`
	Properties           map[string]*Schema `yaml:"properties,omitempty"`
	Required             []string           `yaml:"required,omitempty"`
	AdditionalProperties *Schema            `yaml:"additionalProperties,omitempty"`
	Items                *Schema            `yaml:"items,omitempty"`
	Enum                 []any              `yaml:"enum,omitempty"`
	Default              any                `yaml:"default,omitempty"`
	Ref                  string             `yaml:"$ref,omitempty"`
	Definitions          map[string]*Schema `yaml:"definitions,omitempty"`
	OneOf                []*Schema          `yaml:"oneOf,omitempty"`
	AnyOf                []*Schema          `yaml:"anyOf,omitempty"`
	AllOf                []*Schema          `yaml:"allOf,omitempty"`
	Format               string             `yaml:"format,omitempty"`
	Minimum              *float64           `yaml:"minimum,omitempty"`
	Maximum              *float64           `yaml:"maximum,omitempty"`
	MinLength            *int               `yaml:"minLength,omitempty"`
	MaxLength            *int               `yaml:"maxLength,omitempty"`
	Pattern              string             `yaml:"pattern,omitempty"`
	MinItems             *int               `yaml:"minItems,omitempty"`
	MaxItems             *int               `yaml:"maxItems,omitempty"`
}

// FieldInfo holds metadata about a struct field for schema generation.
type FieldInfo struct {
	Name        string      // Go field name
	JSONName    string      // Name in JSON/YAML (from mapstructure or json tag)
	Type        string      // Go type string representation
	Description string      // Doc comment for the field
	Default     any         // Default value if any
	Embedded    bool        // Whether this is an embedded struct
	Fields      []FieldInfo // Nested fields for struct types
}

// StructInfo holds metadata about a Go struct for schema generation.
type StructInfo struct {
	Name        string      // Struct name
	Package     string      // Package path
	Description string      // Doc comment for the struct
	Fields      []FieldInfo // Struct fields
}
