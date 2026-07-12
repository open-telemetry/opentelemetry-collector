// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen // import "go.opentelemetry.io/collector/internal/schemagen"

import (
	"encoding/json"
	"maps"
	"slices"

	"github.com/google/jsonschema-go/jsonschema"
)

const schemaVersion = "https://json-schema.org/draft/2020-12/schema"

// JSONSchema models a JSON Schema Draft 2020-12 schema.
type JSONSchema = jsonschema.Schema

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
	if md.Default != nil {
		if raw, err := json.Marshal(md.Default); err == nil {
			jsonSchema.Default = raw
		}
	}
	jsonSchema.Deprecated = md.Deprecated
	jsonSchema.Enum = md.Enum
	jsonSchema.Required = md.Required
	jsonSchema.Pattern = md.Pattern
	jsonSchema.Format = md.Format

	if md.MinProperties != nil {
		jsonSchema.MinProperties = md.MinProperties
	}
	if md.MaxProperties != nil {
		jsonSchema.MaxProperties = md.MaxProperties
	}
	if md.MinItems != nil {
		jsonSchema.MinItems = md.MinItems
	}
	if md.MaxItems != nil {
		jsonSchema.MaxItems = md.MaxItems
	}
	if md.MinLength != nil {
		jsonSchema.MinLength = md.MinLength
	}
	if md.MaxLength != nil {
		jsonSchema.MaxLength = md.MaxLength
	}
	jsonSchema.UniqueItems = md.UniqueItems

	if md.Minimum != nil {
		jsonSchema.Minimum = md.Minimum
	}
	if md.ExclusiveMinimum != nil {
		jsonSchema.ExclusiveMinimum = md.ExclusiveMinimum
	}
	if md.Maximum != nil {
		jsonSchema.Maximum = md.Maximum
	}
	if md.ExclusiveMaximum != nil {
		jsonSchema.ExclusiveMaximum = md.ExclusiveMaximum
	}

	if md.Items != nil {
		jsonSchema.Items = convertMetadataToJSONSchema(md.Items, &JSONSchema{})
	}
	if md.AdditionalProperties != nil {
		jsonSchema.AdditionalProperties = convertMetadataToJSONSchema(md.AdditionalProperties, &JSONSchema{})
	}

	return jsonSchema
}
