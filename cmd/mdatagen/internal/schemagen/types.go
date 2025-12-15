// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/schemagen"

// Schema represents a JSON Schema document (draft-07).
type Schema struct {
	Schema               string             `json:"$schema,omitempty"`
	ID                   string             `json:"$id,omitempty"`
	Title                string             `json:"title,omitempty"`
	Description          string             `json:"description,omitempty"`
	Type                 string             `json:"type,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	Required             []string           `json:"required,omitempty"`
	AdditionalProperties *Schema            `json:"additionalProperties,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	Enum                 []any              `json:"enum,omitempty"`
	Default              any                `json:"default,omitempty"`
	Ref                  string             `json:"$ref,omitempty"`
	Definitions          map[string]*Schema `json:"definitions,omitempty"`
	OneOf                []*Schema          `json:"oneOf,omitempty"`
	AnyOf                []*Schema          `json:"anyOf,omitempty"`
	AllOf                []*Schema          `json:"allOf,omitempty"`
	Format               string             `json:"format,omitempty"`
	Minimum              *float64           `json:"minimum,omitempty"`
	Maximum              *float64           `json:"maximum,omitempty"`
	MinLength            *int               `json:"minLength,omitempty"`
	MaxLength            *int               `json:"maxLength,omitempty"`
	Pattern              string             `json:"pattern,omitempty"`
	MinItems             *int               `json:"minItems,omitempty"`
	MaxItems             *int               `json:"maxItems,omitempty"`
}

// FieldInfo holds metadata about a struct field for schema generation.
type FieldInfo struct {
	Name        string      // Go field name
	JSONName    string      // Name in JSON/YAML (from mapstructure or json tag)
	Type        string      // Go type string representation
	Description string      // Doc comment for the field
	Required    bool        // Whether the field is required
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
