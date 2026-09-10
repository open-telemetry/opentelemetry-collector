// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen // import "go.opentelemetry.io/collector/internal/schemagen"

// goDurationPattern matches Go duration strings (e.g., "30s", "1h30m", "500ms")
const goDurationPattern = `^([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$`

// Extended type alias constants — these are the user-facing names accepted in the type: field
// of metadata.yaml. Each one expands to a primitive SchemaType during schema resolution.
const (
	FloatType        SchemaType = "float"
	DoubleType       SchemaType = "double"
	DurationType     SchemaType = "duration"
	TimeType         SchemaType = "time"
	OpaqueStringType SchemaType = "opaque_string"
	ComponentIDType  SchemaType = "component_id"
	OpaqueMapType    SchemaType = "opaque_map"
)

// extendedTypes is the centralized registry of first-class type aliases that can be used as the "type" field
// in a metadata.yaml config schema. Each entry maps an alias name to the standard JSON Schema fields it expands to,
// together with any Go-specific annotations (GoType) needed for code generation.
//
// To add a new alias, add a single entry here. No other switch or case needs editing.
var extendedTypes = map[SchemaType]ConfigMetadata{
	FloatType:  {Type: Float32Type},
	DoubleType: {Type: Float64Type},

	// String-backed aliases using full import-path GoType convention
	OpaqueStringType: {Type: StringType, GoType: "go.opentelemetry.io/collector/config/configopaque.String"},
	ComponentIDType:  {Type: StringType, GoType: "go.opentelemetry.io/collector/component.ID"},

	// duration and time
	DurationType: {Type: StringType, GoType: "time.Duration", Pattern: goDurationPattern},
	TimeType:     {Type: StringType, GoType: "time.Time", Format: "date-time"},

	// opaque_map: Go uses configopaque.MapList; JSON gets a map[string]string
	OpaqueMapType: {
		Type:   MapType,
		GoType: "go.opentelemetry.io/collector/config/configopaque.MapList",
		Values: &ConfigMetadata{
			Type: StringType,
		},
	},
}

// expandExtendedType rewrites md.Type from an extended alias to the equivalent standard JSON Schema fields.
// It is a no-op when md.Type is already a standard JSON Schema type. An explicit x-customType on the node
// is never overwritten.
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
