// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen // import "go.opentelemetry.io/collector/internal/schemagen"

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"go.opentelemetry.io/collector/confmap"
)

// ConfigMetadata is a single schema node parsed from metadata.yaml's `config` (and from
// any externally-loaded schema). It mirrors a subset of JSON Schema 2020-12, plus the
// project-specific x-* and go_struct extensions.
//
// It is NOT the type emitted to disk as a JSON Schema document; that is JSONSchemaDoc.
// The Defs field is kept here only to back the resolver's internal ref-lookup logic for
// schemas that carry their own nested $defs. Top-level $defs (built from a component's
// exported_configs) are owned by JSONSchemaDoc and emitted at write time. The json tag
// on Defs is "-" so a resolved node's internal defs are never serialized directly; the
// document's $defs go through JSONSchemaDoc instead.
type ConfigMetadata struct {
	Schema               string                     `mapstructure:"$schema,omitempty" json:"$schema,omitempty" yaml:"$schema,omitempty"`
	ID                   string                     `mapstructure:"$id,omitempty" json:"$id,omitempty" yaml:"$id,omitempty"`
	Title                string                     `mapstructure:"title,omitempty" json:"title,omitempty" yaml:"title,omitempty"`
	Description          string                     `mapstructure:"description,omitempty" json:"description,omitempty" yaml:"description,omitempty"`
	Comment              string                     `mapstructure:"$comment,omitempty" json:"$comment,omitempty" yaml:"$comment,omitempty"`
	Type                 string                     `mapstructure:"type,omitempty" json:"type,omitempty" yaml:"type,omitempty"`
	Ref                  string                     `mapstructure:"$ref,omitempty" json:"-" yaml:"$ref,omitempty"`
	Default              any                        `mapstructure:"default,omitempty" json:"default,omitempty" yaml:"default,omitempty"`
	Examples             []any                      `mapstructure:"examples,omitempty" json:"examples,omitempty" yaml:"examples,omitempty"`
	Deprecated           bool                       `mapstructure:"deprecated,omitempty" json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Enum                 []any                      `mapstructure:"enum,omitempty" json:"enum,omitempty" yaml:"enum,omitempty"`
	Const                any                        `mapstructure:"const,omitempty" json:"const,omitempty" yaml:"const,omitempty"`
	AllOf                []*ConfigMetadata          `mapstructure:"allOf,omitempty" json:"allOf,omitempty" yaml:"allOf,omitempty"`
	Properties           map[string]*ConfigMetadata `mapstructure:"properties,omitempty" json:"properties,omitempty" yaml:"properties,omitempty"`
	AdditionalProperties *ConfigMetadata            `mapstructure:"additionalProperties,omitempty" json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
	PatternProperties    map[string]*ConfigMetadata `mapstructure:"patternProperties,omitempty" json:"-" yaml:"patternProperties,omitempty"`
	Required             []string                   `mapstructure:"required,omitempty" json:"required,omitempty" yaml:"required,omitempty"`
	MinProperties        *int                       `mapstructure:"minProperties,omitempty" json:"minProperties,omitempty" yaml:"minProperties,omitempty"`
	MaxProperties        *int                       `mapstructure:"maxProperties,omitempty" json:"maxProperties,omitempty" yaml:"maxProperties,omitempty"`
	Items                *ConfigMetadata            `mapstructure:"items,omitempty" json:"items,omitempty" yaml:"items,omitempty"`
	MinItems             *int                       `mapstructure:"minItems,omitempty" json:"minItems,omitempty" yaml:"minItems,omitempty"`
	MaxItems             *int                       `mapstructure:"maxItems,omitempty" json:"maxItems,omitempty" yaml:"maxItems,omitempty"`
	UniqueItems          bool                       `mapstructure:"uniqueItems,omitempty" json:"uniqueItems,omitempty" yaml:"uniqueItems,omitempty"`
	MaxLength            *int                       `mapstructure:"maxLength,omitempty" json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	MinLength            *int                       `mapstructure:"minLength,omitempty" json:"minLength,omitempty" yaml:"minLength,omitempty"`
	Pattern              string                     `mapstructure:"pattern,omitempty" json:"pattern,omitempty" yaml:"pattern,omitempty"`
	Format               string                     `mapstructure:"format,omitempty" json:"format,omitempty" yaml:"format,omitempty"`
	ContentMediaType     string                     `mapstructure:"contentMediaType,omitempty" json:"contentMediaType,omitempty" yaml:"contentMediaType,omitempty"`
	ContentEncoding      string                     `mapstructure:"contentEncoding,omitempty" json:"contentEncoding,omitempty" yaml:"contentEncoding,omitempty"`
	ContentSchema        *ConfigMetadata            `mapstructure:"contentSchema,omitempty" json:"contentSchema,omitempty" yaml:"contentSchema,omitempty"`
	MultipleOf           *float64                   `mapstructure:"multipleOf,omitempty" json:"multipleOf,omitempty" yaml:"multipleOf,omitempty"`
	Maximum              *float64                   `mapstructure:"maximum,omitempty" json:"maximum,omitempty" yaml:"maximum,omitempty"`
	ExclusiveMaximum     *float64                   `mapstructure:"exclusiveMaximum,omitempty" json:"exclusiveMaximum,omitempty" yaml:"exclusiveMaximum,omitempty"`
	Minimum              *float64                   `mapstructure:"minimum,omitempty" json:"minimum,omitempty" yaml:"minimum,omitempty"`
	ExclusiveMinimum     *float64                   `mapstructure:"exclusiveMinimum,omitempty" json:"exclusiveMinimum,omitempty" yaml:"exclusiveMinimum,omitempty"`
	Defs                 map[string]*ConfigMetadata `mapstructure:"$defs,omitempty" json:"-" yaml:"$defs,omitempty"`
	// Additional custom fields
	GoStruct   GoStructConfig `mapstructure:"go_struct,omitempty" json:"-" yaml:"go_struct,omitempty"`
	GoType     string         `mapstructure:"x-customType,omitempty" json:"-" yaml:"x-customType,omitempty"`
	IsPointer  bool           `mapstructure:"x-pointer,omitempty" json:"-" yaml:"x-pointer,omitempty"`
	IsOptional bool           `mapstructure:"x-optional,omitempty" json:"-" yaml:"x-optional,omitempty"`
	Embed      bool           `mapstructure:"embed,omitempty" json:"-" yaml:"embed,omitempty"`
	// internal
	ResolvedFrom                string `mapstructure:"-" json:"-" yaml:"-"`
	EmbeddedName                string `mapstructure:"-" json:"-" yaml:"-"`
	AdditionalPropertiesAllowed *bool  `mapstructure:"-" json:"-" yaml:"-"`
	InternalOnly                bool   `mapstructure:"-" json:"-" yaml:"-"`
}

type Metadata struct {
	Config          *ConfigMetadata            `mapstructure:"config,omitempty" json:"config,omitempty" yaml:"config,omitempty"`
	ExportedConfigs map[string]*ConfigMetadata `mapstructure:"exported_configs,omitempty" json:"exported_configs,omitempty" yaml:"exported_configs,omitempty"`
}

type GoStructConfig struct {
	CustomValidator *CustomValidatorConfig `mapstructure:"custom_validator" json:"-" yaml:"custom_validator,omitempty"`
	Anonymous       bool                   `mapstructure:"anonymous" json:"-" yaml:"anonymous,omitempty"`
	IgnoreDefault   bool                   `mapstructure:"ignore_default" json:"-" yaml:"ignore_default,omitempty"`
	FieldName       string                 `mapstructure:"field_name" json:"-" yaml:"field_name,omitempty"`
}

type CustomValidatorConfig struct {
	Name string `mapstructure:"name,omitempty" json:"-" yaml:"name,omitempty"`
}

func (g *GoStructConfig) Unmarshal(parser *confmap.Conf) error {
	type goStructConfig GoStructConfig
	if err := parser.Unmarshal((*goStructConfig)(g), confmap.WithIgnoreUnused()); err != nil {
		return err
	}
	if !parser.IsSet("custom_validator") || g.CustomValidator != nil {
		return nil
	}
	sub, err := parser.Sub("custom_validator")
	if err != nil {
		return fmt.Errorf("invalid custom_validator: %w", err)
	}
	g.CustomValidator = &CustomValidatorConfig{}
	return sub.Unmarshal(g.CustomValidator)
}

// JSONSchemaDoc is the writer-side JSON Schema 2020-12 document produced by schemagen.
// It is a standalone type, not a wrapper around ConfigMetadata: a component's config
// root is always an object schema, so the document carries the object-level and
// identity fields it can actually emit plus the top-level $defs built from the
// component's exported_configs (and any internal definitions injected by mdatagen).
//
// Keeping it separate lets the on-disk JSON Schema evolve independently of the
// metadata.yaml config representation, which is free to drift away from JSON Schema
// shape. Standard encoding/json handles serialization; no custom MarshalJSON is needed.
//
// Field order mirrors the equivalent fields on ConfigMetadata so the serialized
// document is byte-identical to the prior embedded-node output: allOf precedes
// properties, and $defs is emitted last.
type JSONSchemaDoc struct {
	Schema      string `json:"$schema,omitempty"`
	ID          string `json:"$id,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Comment     string `json:"$comment,omitempty"`
	Type        string `json:"type,omitempty"`

	AllOf                []*ConfigMetadata          `json:"allOf,omitempty"`
	Properties           map[string]*ConfigMetadata `json:"properties,omitempty"`
	AdditionalProperties any                        `json:"additionalProperties,omitempty"`
	PatternProperties    map[string]*ConfigMetadata `json:"patternProperties,omitempty"`
	Required             []string                   `json:"required,omitempty"`
	MinProperties        *int                       `json:"minProperties,omitempty"`
	MaxProperties        *int                       `json:"maxProperties,omitempty"`

	Defs map[string]*ConfigMetadata `json:"$defs,omitempty"`
}

// NewJSONSchemaDoc projects a resolved schema node onto a writer-side JSON Schema
// document. It copies the document-level fields a config root can carry and the
// resolved $defs. The node's internal AdditionalPropertiesAllowed flag becomes a real
// additionalProperties boolean on the document. The source node is not retained.
func NewJSONSchemaDoc(root *ConfigMetadata) *JSONSchemaDoc {
	if root == nil {
		return &JSONSchemaDoc{}
	}
	doc := &JSONSchemaDoc{
		Schema:            root.Schema,
		ID:                root.ID,
		Title:             root.Title,
		Description:       root.Description,
		Comment:           root.Comment,
		Type:              root.Type,
		Properties:        root.Properties,
		PatternProperties: root.PatternProperties,
		Required:          root.Required,
		MinProperties:     root.MinProperties,
		MaxProperties:     root.MaxProperties,
		AllOf:             root.AllOf,
		Defs:              root.Defs,
	}
	if root.AdditionalPropertiesAllowed != nil {
		doc.AdditionalProperties = *root.AdditionalPropertiesAllowed
	} else if root.AdditionalProperties != nil {
		doc.AdditionalProperties = root.AdditionalProperties
	}
	return doc
}

// ToJSON serializes the JSON Schema document with indentation.
func (d *JSONSchemaDoc) ToJSON() ([]byte, error) {
	return json.MarshalIndent(d, "", "  ")
}

// ToJSON serializes the schema node with indentation. This is the entry point used
// by callers that operate on a bare ConfigMetadata, such as the combiner output
// path. The writer-side document path goes through JSONSchemaDoc, which additionally
// emits the top-level $defs.
func (md *ConfigMetadata) ToJSON() ([]byte, error) {
	return json.MarshalIndent(md, "", "  ")
}

// AsJSONSchema builds a writer-side document directly from this source Metadata without
// running the resolver. The document's $defs are the source exported_configs. Used where
// a document is needed without ref resolution.
func (md *Metadata) AsJSONSchema() *JSONSchemaDoc {
	if md == nil {
		return &JSONSchemaDoc{}
	}
	doc := NewJSONSchemaDoc(md.Config)
	doc.Defs = md.ExportedConfigs
	return doc
}

func (md *ConfigMetadata) MarshalJSON() ([]byte, error) {
	type alias ConfigMetadata

	if len(md.PatternProperties) == 0 && md.AdditionalPropertiesAllowed == nil {
		return json.Marshal((*alias)(md))
	}

	type withSpecialFields struct {
		*alias
		AdditionalProperties any                        `json:"additionalProperties,omitempty"`
		PatternProperties    map[string]*ConfigMetadata `json:"patternProperties,omitempty"`
	}

	out := withSpecialFields{
		alias:             (*alias)(md),
		PatternProperties: md.PatternProperties,
	}
	if md.AdditionalPropertiesAllowed != nil {
		out.AdditionalProperties = *md.AdditionalPropertiesAllowed
	}

	return json.Marshal(out)
}

func (md *ConfigMetadata) Validate() error {
	var errs error

	hasDefs := len(md.Defs) > 0
	hasConfigFields := len(md.Properties) > 0 || len(md.AllOf) > 0
	if md.Type != "object" && (md.Type != "" || !hasDefs || hasConfigFields) {
		errs = errors.Join(errs, fmt.Errorf("config type must be \"object\", got %q", md.Type))
	}
	if !hasDefs && !hasConfigFields {
		errs = errors.Join(errs, errors.New("config must not be empty"))
	}
	for name, prop := range md.Properties {
		if len(prop.Enum) > 0 && (prop.Type == "object" || prop.Type == "array") {
			errs = errors.Join(errs, fmt.Errorf("property %q: enum is not supported for type %q", name, prop.Type))
		}
	}
	return errs
}

// Clone returns a deep copy of the Metadata. The returned Metadata shares no pointers
// with the receiver, so callers downstream of Clone() can freely mutate the result
// without affecting the source.
func (md *Metadata) Clone() *Metadata {
	if md == nil {
		return nil
	}
	out := &Metadata{Config: md.Config.Clone()}
	if md.ExportedConfigs != nil {
		out.ExportedConfigs = make(map[string]*ConfigMetadata, len(md.ExportedConfigs))
		for k, v := range md.ExportedConfigs {
			out.ExportedConfigs[k] = v.Clone()
		}
	}
	return out
}

// Clone returns a deep copy of the ConfigMetadata node and every pointer-bearing field
// reachable from it (Properties, Items, AllOf, AdditionalProperties, ContentSchema,
// PatternProperties, Defs). Primitive slices are copied; numeric pointer fields
// (MinLength, MultipleOf, etc.) are shallow-shared since the resolver only reads them.
func (md *ConfigMetadata) Clone() *ConfigMetadata {
	if md == nil {
		return nil
	}
	out := *md
	if md.AllOf != nil {
		out.AllOf = make([]*ConfigMetadata, len(md.AllOf))
		for i, v := range md.AllOf {
			out.AllOf[i] = v.Clone()
		}
	}
	if md.Properties != nil {
		out.Properties = make(map[string]*ConfigMetadata, len(md.Properties))
		for k, v := range md.Properties {
			out.Properties[k] = v.Clone()
		}
	}
	if md.AdditionalProperties != nil {
		out.AdditionalProperties = md.AdditionalProperties.Clone()
	}
	if md.PatternProperties != nil {
		out.PatternProperties = make(map[string]*ConfigMetadata, len(md.PatternProperties))
		for k, v := range md.PatternProperties {
			out.PatternProperties[k] = v.Clone()
		}
	}
	if md.Items != nil {
		out.Items = md.Items.Clone()
	}
	if md.ContentSchema != nil {
		out.ContentSchema = md.ContentSchema.Clone()
	}
	if md.Defs != nil {
		out.Defs = make(map[string]*ConfigMetadata, len(md.Defs))
		for k, v := range md.Defs {
			out.Defs[k] = v.Clone()
		}
	}
	if md.Examples != nil {
		out.Examples = slices.Clone(md.Examples)
	}
	if md.Enum != nil {
		out.Enum = slices.Clone(md.Enum)
	}
	if md.Required != nil {
		out.Required = slices.Clone(md.Required)
	}
	return &out
}
