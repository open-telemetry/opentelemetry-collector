// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen // import "go.opentelemetry.io/collector/internal/schemagen"

import "fmt"

// goDurationPattern matches Go duration strings (e.g., "30s", "1h30m", "500ms")
const goDurationPattern = `^([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$`

// extendedTypes is the centralized registry of first-class type aliases that can be used as the "type" field
// in a metadata.yaml config schema. Each entry maps an alias name to the standard JSON Schema fields it expands to,
// together with any Go-specific annotations (GoType) needed for code generation.
//
// To add a new alias, add a single entry here. No other switch or case needs editing.
var extendedTypes = map[string]ConfigMetadata{
	// Integer aliases
	"rune":   {Type: "integer", GoType: "rune"},
	"byte":   {Type: "integer", GoType: "byte"},
	"uint":   {Type: "integer", GoType: "uint"},
	"int8":   {Type: "integer", GoType: "int8"},
	"uint8":  {Type: "integer", GoType: "uint8"},
	"int16":  {Type: "integer", GoType: "int16"},
	"uint16": {Type: "integer", GoType: "uint16"},
	"int32":  {Type: "integer", GoType: "int32"},
	"uint32": {Type: "integer", GoType: "uint32"},
	"int64":  {Type: "integer", GoType: "int64"},
	"uint64": {Type: "integer", GoType: "uint64"},

	// Number aliases
	"float32": {Type: "number", GoType: "float32"},
	"float64": {Type: "number", GoType: "float64"},

	// String-backed aliases using full import-path GoType convention
	"opaque_string": {Type: "string", GoType: "go.opentelemetry.io/collector/config/configopaque.String"},
	"id":            {Type: "string", GoType: "go.opentelemetry.io/collector/component.ID"},

	// duration and time
	"duration": {Type: "string", GoType: "time.Duration", Pattern: goDurationPattern},
	"time":     {Type: "string", GoType: "time.Time", Format: "date-time"},

	// opaque_map: Go uses configopaque.MapList; JSON gets a map[string]string
	"opaque_map": {
		Type:   "object",
		GoType: "go.opentelemetry.io/collector/config/configopaque.MapList",
		AdditionalProperties: &ConfigMetadata{
			Type: "string",
		},
	},
}

var standardJSONTypes = map[string]bool{
	"":        true, // empty is allowed (treated as "any" by the generator)
	"string":  true,
	"integer": true,
	"number":  true,
	"boolean": true,
	"object":  true,
	"array":   true,
	"null":    true,
}

func isStandardJSONType(typ string) bool {
	return standardJSONTypes[typ]
}

// expandExtendedType rewrites md.Type from an extended alias to the equivalent standard JSON Schema fields.
// It is a no-op when md.Type is already a standard JSON Schema type. An explicit x-customType on the node
// is never overwritten.
//
// Returns an actionable error for unknown aliases.
func expandExtendedType(md *ConfigMetadata) error {
	if isStandardJSONType(md.Type) {
		return nil
	}
	ext, ok := extendedTypes[md.Type]
	if !ok {
		return fmt.Errorf("unknown config type %q: not a JSON Schema type and not a known extended type alias (e.g. int64, duration, opaque_string, id, opaque_map)", md.Type)
	}

	md.Type = ext.Type

	if md.GoType == "" {
		md.GoType = ext.GoType
	}

	if md.Format == "" {
		md.Format = ext.Format
	}

	if md.Pattern == "" {
		md.Pattern = ext.Pattern
	}

	if md.Items == nil {
		md.Items = ext.Items
	}

	if md.AdditionalProperties == nil {
		md.AdditionalProperties = ext.AdditionalProperties
	}

	return nil
}
