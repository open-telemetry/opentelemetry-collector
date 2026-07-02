// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen // import "go.opentelemetry.io/collector/internal/schemagen"

import (
	"encoding/json"
	"maps"
	"slices"
	"strconv"
)

const schemaVersion = "https://json-schema.org/draft/2020-12/schema"

// JSONSchema models a JSON Schema Draft 2020-12 schema.
type JSONSchema struct {
	Schema               string                 `json:"$schema,omitempty"`
	ID                   string                 `json:"$id,omitempty"`
	Title                string                 `json:"title,omitempty"`
	Description          string                 `json:"description,omitempty"`
	Type                 any                    `json:"type,omitempty"`
	Default              any                    `json:"default,omitempty"`
	AllOf                []*JSONSchema          `json:"allOf,omitempty"`
	Ref                  string                 `json:"$ref,omitempty"`
	Items                *JSONSchema            `json:"items,omitempty"`
	AdditionalProperties *JSONSchema            `json:"additionalProperties,omitempty"`
	Properties           map[string]*JSONSchema `json:"properties,omitempty"`
	PatternProperties    map[string]*JSONSchema `json:"patternProperties,omitempty"`
	Enum                 []any                  `json:"enum,omitempty"`
	Maximum              json.Number            `json:"maximum,omitempty"`
	ExclusiveMaximum     json.Number            `json:"exclusiveMaximum,omitempty"`
	Minimum              json.Number            `json:"minimum,omitempty"`
	ExclusiveMinimum     json.Number            `json:"exclusiveMinimum,omitempty"`
	MaxLength            *uint64                `json:"maxLength,omitempty"`
	MinLength            *uint64                `json:"minLength,omitempty"`
	Pattern              string                 `json:"pattern,omitempty"`
	MaxItems             *uint64                `json:"maxItems,omitempty"`
	MinItems             *uint64                `json:"minItems,omitempty"`
	UniqueItems          bool                   `json:"uniqueItems,omitempty"`
	MaxProperties        *uint64                `json:"maxProperties,omitempty"`
	MinProperties        *uint64                `json:"minProperties,omitempty"`
	Required             []string               `json:"required,omitempty"`
	Format               string                 `json:"format,omitempty"`
	Deprecated           bool                   `json:"deprecated,omitempty"`
	Defs                 map[string]*JSONSchema `json:"$defs,omitempty"`
	Boolean              *bool                  `json:"-"`
}

func (s JSONSchema) MarshalJSON() ([]byte, error) {
	if s.Boolean != nil {
		return json.Marshal(*s.Boolean)
	}

	type plainJSONSchema JSONSchema
	return json.Marshal(plainJSONSchema(s))
}

func FromMetadata(id, title string, md *ConfigsMetadata) *JSONSchema {
	jsonSchema := &JSONSchema{
		Schema: schemaVersion,
		ID:     id,
		Title:  title,
	}

	defs := make(map[string]*JSONSchema)
	for name, config := range md.ExportedConfigs {
		if !config.InternalOnly {
			defs[name] = convertMetadataToJSONSchema(config, &JSONSchema{})
		}
	}
	if len(defs) > 0 {
		jsonSchema.Defs = defs
	}

	if md.Config != nil {
		convertMetadataToJSONSchema(md.Config, jsonSchema)
	}

	return jsonSchema
}

func convertMetadataToJSONSchema(md *ConfigMetadata, jsonSchema *JSONSchema) *JSONSchema {
	if md.Properties != nil {
		properties := make(map[string]*JSONSchema)
		allOf := make([]*JSONSchema, 0)
		for _, propName := range slices.Sorted(maps.Keys(md.Properties)) {
			propMd := md.Properties[propName]
			if propMd.Embed {
				allOf = append(allOf, convertMetadataToJSONSchema(propMd, &JSONSchema{}))
				continue
			}
			properties[propName] = convertMetadataToJSONSchema(propMd, &JSONSchema{})
		}
		if len(allOf) > 0 {
			jsonSchema.AllOf = allOf
		}
		if len(properties) > 0 {
			jsonSchema.Properties = properties
		}
	}

	if md.Type != "" {
		jsonSchema.Type = md.Type
	}
	jsonSchema.Description = md.Description
	jsonSchema.Default = md.Default
	jsonSchema.Deprecated = md.Deprecated
	jsonSchema.Enum = md.Enum
	jsonSchema.Required = md.Required
	jsonSchema.Pattern = md.Pattern
	jsonSchema.Format = md.Format

	if md.MinProperties != nil {
		v := uint64(*md.MinProperties)
		jsonSchema.MinProperties = &v
	}
	if md.MaxProperties != nil {
		v := uint64(*md.MaxProperties)
		jsonSchema.MaxProperties = &v
	}
	if md.MinItems != nil {
		v := uint64(*md.MinItems)
		jsonSchema.MinItems = &v
	}
	if md.MaxItems != nil {
		v := uint64(*md.MaxItems)
		jsonSchema.MaxItems = &v
	}
	if md.MinLength != nil {
		v := uint64(*md.MinLength)
		jsonSchema.MinLength = &v
	}
	if md.MaxLength != nil {
		v := uint64(*md.MaxLength)
		jsonSchema.MaxLength = &v
	}
	jsonSchema.UniqueItems = md.UniqueItems

	if md.Minimum != nil {
		jsonSchema.Minimum = json.Number(strconv.FormatFloat(*md.Minimum, 'f', -1, 64))
	}
	if md.ExclusiveMinimum != nil {
		jsonSchema.ExclusiveMinimum = json.Number(strconv.FormatFloat(*md.ExclusiveMinimum, 'f', -1, 64))
	}
	if md.Maximum != nil {
		jsonSchema.Maximum = json.Number(strconv.FormatFloat(*md.Maximum, 'f', -1, 64))
	}
	if md.ExclusiveMaximum != nil {
		jsonSchema.ExclusiveMaximum = json.Number(strconv.FormatFloat(*md.ExclusiveMaximum, 'f', -1, 64))
	}

	if md.Items != nil {
		jsonSchema.Items = convertMetadataToJSONSchema(md.Items, &JSONSchema{})
	}
	if md.AdditionalProperties != nil {
		jsonSchema.AdditionalProperties = convertMetadataToJSONSchema(md.AdditionalProperties, &JSONSchema{})
	}

	return jsonSchema
}
