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
	SizerType        SchemaType = "sizer"
)

// extendedTypes is the centralized registry of first-class type aliases that can be used as the "type" field
// in a metadata.yaml config schema. Each entry maps an alias name to the standard JSON Schema fields it expands to,
// together with any Go-specific annotations (go_struct.type) needed for code generation.
//
// To add a new alias, add a single entry here. No other switch or case needs editing.
var extendedTypes = map[SchemaType]ConfigMetadata{
	FloatType:  {Type: Float32Type},
	DoubleType: {Type: Float64Type},

	// String-backed aliases using full import-path go_struct.type convention
	OpaqueStringType: {Type: StringType, GoStruct: GoStructConfig{Type: "go.opentelemetry.io/collector/config/configopaque.String"}},
	ComponentIDType:  {Type: StringType, GoStruct: GoStructConfig{Type: "go.opentelemetry.io/collector/component.ID"}},

	// duration and time
	DurationType: {Type: StringType, GoStruct: GoStructConfig{Type: "time.Duration"}, Pattern: goDurationPattern},
	TimeType:     {Type: StringType, GoStruct: GoStructConfig{Type: "time.Time"}, Format: "date-time"},

	// opaque_map: Go uses configopaque.MapList; JSON gets a map[string]string
	OpaqueMapType: {
		Type:     MapType,
		GoStruct: GoStructConfig{Type: "go.opentelemetry.io/collector/config/configopaque.MapList"},
		Values: &ConfigMetadata{
			Type: StringType,
		},
	},

	// sizer: Go uses requests.SizerType; JSON gets a string
	SizerType: {
		Type: StringType,
		GoStruct: GoStructConfig{
			Type: "go.opentelemetry.io/collector/exporter/exporterhelper/internal/request.SizerType",
		},
	},
}

// expandExtendedType rewrites md.Type from an extended alias to the equivalent standard JSON Schema fields.
// It is a no-op when md.Type is already a standard JSON Schema type. An explicit go_struct.type on the node
// is never overwritten.
func expandExtendedType(md *ConfigMetadata) error {
	ext, ok := extendedTypes[md.Type]
	if !ok { // not an extended type
		return nil
	}

	md.Type = ext.Type

	if md.GoStruct.Type == "" {
		md.GoStruct.Type = ext.GoStruct.Type
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
