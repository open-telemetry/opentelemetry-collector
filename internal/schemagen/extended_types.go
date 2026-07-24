// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen // import "go.opentelemetry.io/collector/internal/schemagen"

// goDurationPattern matches Go duration strings (e.g., "30s", "1h30m", "500ms")
const goDurationPattern = `^([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$`

// extendedTypes is the centralized registry of first-class type aliases that can be used as the "type" field
// in a metadata.yaml config schema. Each entry maps an alias name to the standard JSON Schema fields it expands to,
// together with any Go-specific annotations (GoType) needed for code generation.
//
// To add a new alias, add a single entry here. No other switch or case needs editing.
var extendedTypes = map[string]ConfigMetadata{
	"float":  {Type: "float32"},
	"double": {Type: "float64"},

	// String-backed aliases using full import-path GoType convention
	"opaque_string": {Type: "string", GoType: "go.opentelemetry.io/collector/config/configopaque.String"},
	"component_id":  {Type: "string", GoType: "go.opentelemetry.io/collector/component.ID"},

	// duration and time
	"duration": {Type: "string", GoType: "time.Duration", Pattern: goDurationPattern},
	"time":     {Type: "string", GoType: "time.Time", Format: "date-time"},

	// opaque_map: Go uses configopaque.MapList; JSON gets a map[string]string
	"opaque_map": {
		Type:   "map",
		GoType: "go.opentelemetry.io/collector/config/configopaque.MapList",
		Values: &ConfigMetadata{
			Type: "string",
		},
	},
}

// expandExtendedType rewrites md.Type from an extended alias to the equivalent standard JSON Schema fields.
// It is a no-op when md.Type is already a standard JSON Schema type. An explicit x-customType on the node
// is never overwritten.
//
// Returns an actionable error for unknown aliases.
func expandExtendedType(md *ConfigMetadata) error {
	ext, ok := extendedTypes[md.Type]
	if !ok { // not an extended type
		return nil
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

	if md.Values == nil {
		md.Values = ext.Values
	}

	return nil
}
