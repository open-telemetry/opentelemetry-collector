// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapGoType_BasicTypes(t *testing.T) {
	tests := []struct {
		name     string
		metadata *ConfigMetadata
		propName string
		expected string
	}{
		{
			name:     "string type",
			metadata: &ConfigMetadata{Type: "string"},
			propName: "field",
			expected: "string",
		},
		{
			name:     "integer type",
			metadata: &ConfigMetadata{Type: "integer"},
			propName: "field",
			expected: "int",
		},
		{
			name:     "number type",
			metadata: &ConfigMetadata{Type: "number"},
			propName: "field",
			expected: "float64",
		},
		{
			name:     "boolean type",
			metadata: &ConfigMetadata{Type: "boolean"},
			propName: "field",
			expected: "bool",
		},
		{
			name:     "empty type defaults to any",
			metadata: &ConfigMetadata{Type: ""},
			propName: "field",
			expected: "any",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MapGoType(tt.metadata, tt.propName, "", "")
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestMapGoType_FormattedStrings(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		expected string
	}{
		{
			name:     "date-time format",
			format:   "date-time",
			expected: "time.Time",
		},
		{
			name:     "duration format",
			format:   "duration",
			expected: "time.Duration",
		},
		{
			name:     "no format",
			format:   "",
			expected: "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := &ConfigMetadata{
				Type:   "string",
				Format: tt.format,
			}
			result, err := MapGoType(md, "field", "", "")
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestMapGoType_Arrays(t *testing.T) {
	compPkg := "go.opentelemetry.io/collector/cmd/mdatagen/internal/samplescraper"

	tests := []struct {
		name     string
		metadata *ConfigMetadata
		expected string
	}{
		{
			name: "array with string items",
			metadata: &ConfigMetadata{
				Type:  "array",
				Items: &ConfigMetadata{Type: "string"},
			},
			expected: "[]string",
		},
		{
			name: "array with int items",
			metadata: &ConfigMetadata{
				Type:  "array",
				Items: &ConfigMetadata{Type: "integer"},
			},
			expected: "[]int",
		},
		{
			name: "array with ref items",
			metadata: &ConfigMetadata{
				Type:  "array",
				Items: &ConfigMetadata{Ref: "./internal/metadata.custom_type"},
			},
			expected: "[]metadata.CustomType",
		},
		{
			name: "array with nested object items ",
			metadata: &ConfigMetadata{
				Type: "array",
				Items: &ConfigMetadata{
					Type: "object",
					Properties: map[string]*ConfigMetadata{
						"name": {Type: "string"},
					},
				},
			},
			expected: "[]FieldItem",
		},
		{
			name: "array without items defaults to any",
			metadata: &ConfigMetadata{
				Type:  "array",
				Items: nil,
			},
			expected: "[]any",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MapGoType(tt.metadata, "field", "", compPkg)
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestMapGoType_Objects(t *testing.T) {
	compPkg := "go.opentelemetry.io/collector/cmd/mdatagen/internal/samplescraper"

	tests := []struct {
		name     string
		metadata *ConfigMetadata
		propName string
		expected string
	}{
		{
			name: "object with additionalProperties string",
			metadata: &ConfigMetadata{
				Type:                 "object",
				AdditionalProperties: &ConfigMetadata{Type: "string"},
			},
			propName: "field",
			expected: "map[string]string",
		},
		{
			name: "object with additionalProperties int",
			metadata: &ConfigMetadata{
				Type:                 "object",
				AdditionalProperties: &ConfigMetadata{Type: "integer"},
			},
			propName: "field",
			expected: "map[string]int",
		},
		{
			name: "object with additionalProperties ref",
			metadata: &ConfigMetadata{
				Type:                 "object",
				AdditionalProperties: &ConfigMetadata{Ref: "./internal/metadata.custom_type"},
			},
			propName: "field",
			expected: "map[string]metadata.CustomType",
		},
		{
			name: "object without additionalProperties or properties",
			metadata: &ConfigMetadata{
				Type: "object",
			},
			propName: "field",
			expected: "map[string]any",
		},
		{
			name: "object with properties",
			metadata: &ConfigMetadata{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"name": {Type: "string"},
				},
			},
			propName: "my_config",
			expected: "MyConfig",
		},
		{
			name: "map of arrays of objects",
			metadata: &ConfigMetadata{
				Type: "object",
				AdditionalProperties: &ConfigMetadata{
					Type: "array",
					Items: &ConfigMetadata{
						Type: "object",
						Properties: map[string]*ConfigMetadata{
							"id": {Type: "integer"},
						},
					},
				},
			},
			propName: "field",
			expected: "map[string][]FieldItem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MapGoType(tt.metadata, tt.propName, "", compPkg)
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestMapGoType_CustomTypes(t *testing.T) {
	tests := []struct {
		name     string
		metadata *ConfigMetadata
		expected string
	}{
		{
			name: "custom type with basic type",
			metadata: &ConfigMetadata{
				Type:   "string",
				GoType: "rune",
			},
			expected: "rune",
		},
		{
			name: "custom type with external package",
			metadata: &ConfigMetadata{
				Type:   "object",
				GoType: "github.com/example/pkg.CustomType",
			},
			expected: "pkg.CustomType",
		},
		{
			name: "custom type without package",
			metadata: &ConfigMetadata{
				Type:   "object",
				GoType: "my_custom_type",
			},
			expected: "MyCustomType",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MapGoType(tt.metadata, "field", "", "")
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestMapGoType_References(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		expected string
	}{
		{
			name:     "internal reference",
			ref:      "my_type",
			expected: "MyType",
		},
		{
			name:     "external reference",
			ref:      "go.opentelemetry.io/collector/component.Config",
			expected: "component.Config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := &ConfigMetadata{
				Ref: tt.ref,
			}
			result, err := MapGoType(md, "field", "", "")
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestMapGoType_Modifiers(t *testing.T) {
	tests := []struct {
		name     string
		metadata *ConfigMetadata
		expected string
	}{
		{
			name: "optional type",
			metadata: &ConfigMetadata{
				Type:       "string",
				IsOptional: true,
			},
			expected: "configoptional.Optional[string]",
		},
		{
			name: "pointer type",
			metadata: &ConfigMetadata{
				Type:      "string",
				IsPointer: true,
			},
			expected: "*string",
		},
		{
			name: "optional pointer",
			metadata: &ConfigMetadata{
				Type:       "string",
				IsOptional: true,
				IsPointer:  true,
			},
			expected: "configoptional.Optional[*string]",
		},
		{
			name: "array of pointers",
			metadata: &ConfigMetadata{
				Type: "array",
				Items: &ConfigMetadata{
					Type:      "string",
					IsPointer: true,
				},
			},
			expected: "[]*string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MapGoType(tt.metadata, "field", "", "")
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestMapGoType_NilInput(t *testing.T) {
	_, err := MapGoType(nil, "field", "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil ConfigMetadata")
}

func TestMapGoType_UnsupportedType(t *testing.T) {
	md := &ConfigMetadata{
		Type: "unsupported_type",
	}
	_, err := MapGoType(md, "field", "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported type")
}

func TestExtractImports_BasicTypes(t *testing.T) {
	tests := []struct {
		name     string
		metadata *ConfigMetadata
		expected []string
	}{
		{
			name:     "no imports for basic types",
			metadata: &ConfigMetadata{Type: "string"},
			expected: []string{},
		},
		{
			name: "time import for date-time format",
			metadata: &ConfigMetadata{
				Type:   "string",
				Format: "date-time",
			},
			expected: []string{"time"},
		},
		{
			name: "time import for duration format",
			metadata: &ConfigMetadata{
				Type:   "string",
				Format: "duration",
			},
			expected: []string{"time"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractImports(tt.metadata, "", "")
			require.NoError(t, err)
			require.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestExtractImports_CustomTypes(t *testing.T) {
	tests := []struct {
		name     string
		metadata *ConfigMetadata
		expected []string
	}{
		{
			name: "external custom type",
			metadata: &ConfigMetadata{
				Type:   "object",
				GoType: "github.com/example/pkg.CustomType",
			},
			expected: []string{"github.com/example/pkg"},
		},
		{
			name: "external reference",
			metadata: &ConfigMetadata{
				Ref: "go.opentelemetry.io/collector/component.Config",
			},
			expected: []string{"go.opentelemetry.io/collector/component"},
		},
		{
			name: "no import for internal reference",
			metadata: &ConfigMetadata{
				Ref: "my_type",
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractImports(tt.metadata, "", "")
			require.NoError(t, err)
			require.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestExtractImports_LocalRef(t *testing.T) {
	rootPkg := "go.opentelemetry.io/collector"
	compPkg := "go.opentelemetry.io/collector/scraper/scraperhelper"

	tests := []struct {
		name     string
		metadata *ConfigMetadata
		expected []string
	}{
		{
			name: "local absolute reference",
			metadata: &ConfigMetadata{
				Ref: "/config/confighttp.client_config",
			},
			expected: []string{"go.opentelemetry.io/collector/config/confighttp"},
		},
		{
			name: "local relative reference",
			metadata: &ConfigMetadata{
				Ref: "./internal/metadata.custom_type",
			},
			expected: []string{"go.opentelemetry.io/collector/scraper/scraperhelper/internal/metadata"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractImports(tt.metadata, rootPkg, compPkg)
			require.NoError(t, err)
			require.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestExtractImports_Optional(t *testing.T) {
	md := &ConfigMetadata{
		Type:       "string",
		IsOptional: true,
	}
	result, err := ExtractImports(md, "", "")
	require.NoError(t, err)
	require.Contains(t, result, "go.opentelemetry.io/collector/config/configoptional")
}

func TestExtractImports_Nested(t *testing.T) {
	md := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"timeout": {
				Type:   "string",
				Format: "duration",
			},
			"nested": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"timestamp": {
						Type:   "string",
						Format: "date-time",
					},
				},
			},
		},
	}
	result, err := ExtractImports(md, "", "")
	require.NoError(t, err)
	require.Contains(t, result, "time")
}

func TestExtractImports_AllOf(t *testing.T) {
	md := &ConfigMetadata{
		Type: "object",
		AllOf: []*ConfigMetadata{
			{
				Type:   "string",
				Format: "duration",
			},
		},
	}
	result, err := ExtractImports(md, "", "")
	require.NoError(t, err)
	require.Contains(t, result, "time")
}

func TestExtractImports_ArrayItems(t *testing.T) {
	md := &ConfigMetadata{
		Type: "array",
		Items: &ConfigMetadata{
			Type:   "string",
			Format: "date-time",
		},
	}
	result, err := ExtractImports(md, "", "")
	require.NoError(t, err)
	require.Contains(t, result, "time")
}

func TestExtractImports_AdditionalProperties(t *testing.T) {
	md := &ConfigMetadata{
		Type: "object",
		AdditionalProperties: &ConfigMetadata{
			Type:   "string",
			Format: "duration",
		},
	}
	result, err := ExtractImports(md, "", "")
	require.NoError(t, err)
	require.Contains(t, result, "time")
}

func TestExtractImports_Defs(t *testing.T) {
	md := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"CustomType": {
				Type:   "string",
				Format: "date-time",
			},
		},
	}
	result, err := ExtractImports(md, "", "")
	require.NoError(t, err)
	require.Contains(t, result, "time")
}

func TestExtractImports_NilInput(t *testing.T) {
	result, err := ExtractImports(nil, "", "")
	require.NoError(t, err)
	require.Nil(t, result)
}

func TestFormatTypeName_InternalReferences(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		expected string
	}{
		{
			name:     "simple name",
			ref:      "my_type",
			expected: "MyType",
		},
		{
			name:     "snake case",
			ref:      "my_custom_type",
			expected: "MyCustomType",
		},
		{
			name:     "already formatted",
			ref:      "MyType",
			expected: "MyType",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FormatTypeName(tt.ref, "", "")
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatTypeName_ExternalReferences(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		expected string
	}{
		{
			name:     "full package path",
			ref:      "go.opentelemetry.io/collector/component.Config",
			expected: "component.Config",
		},
		{
			name:     "nested package",
			ref:      "github.com/example/pkg/subpkg.Type",
			expected: "subpkg.Type",
		},
		{
			name:     "type name needs formatting",
			ref:      "github.com/example/pkg.my_type",
			expected: "pkg.MyType",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FormatTypeName(tt.ref, "", "")
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatTypeName_LocalReferences(t *testing.T) {
	rootPkg := "go.opentelemetry.io/collector"
	compPkg := "go.opentelemetry.io/collector/scraper/scraperhelper"

	tests := []struct {
		name     string
		ref      string
		expected string
	}{
		{
			name:     "local absolute",
			ref:      "/config/confighttp.client_config",
			expected: "confighttp.ClientConfig",
		},
		{
			name:     "local relative",
			ref:      "./internal/metadata.metrics_builder",
			expected: "metadata.MetricsBuilder",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FormatTypeName(tt.ref, rootPkg, compPkg)
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatTypeName_InvalidInput(t *testing.T) {
	tests := []struct {
		name string
		ref  string
	}{
		{
			name: "empty type name after dot",
			ref:  "github.com/example/pkg.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FormatTypeName(tt.ref, "", "")
			require.Error(t, err)
		})
	}
}

func TestExtractDefs_Basic(t *testing.T) {
	md := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"CustomType": {
				Type: "string",
			},
			"AnotherType": {
				Type: "integer",
			},
		},
	}

	result := ExtractDefs(md)
	require.Len(t, result, 2)
	require.Contains(t, result, "CustomType")
	require.Contains(t, result, "AnotherType")
}

func TestExtractDefs_NestedDefs(t *testing.T) {
	md := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"OuterType": {
				Type: "object",
				Defs: map[string]*ConfigMetadata{
					"InnerType": {
						Type: "string",
					},
				},
			},
		},
	}

	result := ExtractDefs(md)
	require.Len(t, result, 2)
	require.Contains(t, result, "OuterType")
	require.Contains(t, result, "InnerType")
}

func TestExtractDefs_EmbeddedObjects(t *testing.T) {
	md := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"config": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"name": {Type: "string"},
				},
			},
		},
	}

	result := ExtractDefs(md)
	require.Len(t, result, 1)
	require.Contains(t, result, "config")
	require.Equal(t, "object", result["config"].Type)
}

func TestExtractDefs_ArrayItems(t *testing.T) {
	md := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"servers": {
				Type: "array",
				Items: &ConfigMetadata{
					Type: "object",
					Properties: map[string]*ConfigMetadata{
						"host": {Type: "string"},
					},
				},
			},
		},
	}

	result := ExtractDefs(md)
	require.Len(t, result, 1)
	require.Contains(t, result, "servers_item")
	require.Equal(t, "object", result["servers_item"].Type)
}

func TestExtractDefs_NilInput(t *testing.T) {
	result := ExtractDefs(nil)
	require.Empty(t, result)
}

func TestExtractDefs_EmptyInput(t *testing.T) {
	md := &ConfigMetadata{
		Type: "object",
	}
	result := ExtractDefs(md)
	require.Empty(t, result)
}

func TestNewCfgFns_ExtractImports(t *testing.T) {
	fns := NewCfgFns("go.opentelemetry.io/collector", "go.opentelemetry.io/collector/comp")

	extractImports := fns["extractImports"].(func(*ConfigMetadata) []string)

	// nil input returns nil
	require.Nil(t, extractImports(nil))

	// valid input returns imports
	md := &ConfigMetadata{Type: "string", Format: "duration"}
	result := extractImports(md)
	require.Contains(t, result, "time")

	// input with unresolvable GoType: collectImports swallows the error, returns empty slice
	errMd := &ConfigMetadata{GoType: "github.com/pkg."}
	result = extractImports(errMd)
	require.Empty(t, result)
}

func TestNewCfgFns_ExtractDefs(t *testing.T) {
	fns := NewCfgFns("", "")

	extractDefs := fns["extractDefs"].(func(*ConfigMetadata) map[string]*ConfigMetadata)

	// nil input returns nil
	require.Nil(t, extractDefs(nil))

	// valid input
	md := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{"MyType": {Type: "string"}},
	}
	result := extractDefs(md)
	require.Contains(t, result, "MyType")
}

func TestNewCfgFns_MapGoType(t *testing.T) {
	fns := NewCfgFns("", "")

	mapGoType := fns["mapGoType"].(func(*ConfigMetadata, string) string)

	// nil input returns "any"
	require.Equal(t, "any", mapGoType(nil, "field"))

	// valid input
	require.Equal(t, "string", mapGoType(&ConfigMetadata{Type: "string"}, "field"))
}

func TestNewCfgFns_PublicType(t *testing.T) {
	fns := NewCfgFns("", "")

	publicType := fns["publicType"].(func(string) string)

	require.Equal(t, "MyType", publicType("my_type"))
	require.Equal(t, "component.Config", publicType("go.opentelemetry.io/collector/component.Config"))
}

func TestWithCfgFns(t *testing.T) {
	base := map[string]any{"existing": "value"}
	result := WithCfgFns(base, "", "")

	require.Equal(t, "value", result["existing"])
	require.Contains(t, result, "mapGoType")
	require.Contains(t, result, "extractImports")
	require.Contains(t, result, "extractDefs")
	require.Contains(t, result, "publicType")
}

func TestResolveGoType_CustomTypeFormatError(t *testing.T) {
	// GoType with invalid empty type name after dot triggers FormatTypeName error
	md := &ConfigMetadata{GoType: "github.com/pkg."}
	_, err := MapGoType(md, "field", "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to format custom type")
}

func TestResolveGoType_RefFormatError(t *testing.T) {
	// Ref with invalid empty type name after dot triggers FormatTypeName error
	md := &ConfigMetadata{Ref: "github.com/pkg."}
	_, err := MapGoType(md, "field", "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to format reference type")
}

func TestResolveGoType_ArrayItemError(t *testing.T) {
	// Array whose item type fails to resolve
	md := &ConfigMetadata{
		Type:  "array",
		Items: &ConfigMetadata{Type: "unsupported_array_item"},
	}
	_, err := MapGoType(md, "field", "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to map array item type")
}

func TestResolveGoType_AdditionalPropertiesError(t *testing.T) {
	// Object with additionalProperties whose type fails to resolve
	md := &ConfigMetadata{
		Type:                 "object",
		AdditionalProperties: &ConfigMetadata{Type: "unsupported_value"},
	}
	_, err := MapGoType(md, "field", "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to map additionalProperties type")
}

func TestResolveGoType_EmbeddedObjectNameError(t *testing.T) {
	// Object with properties but propName that cannot be formatted as an identifier
	md := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"x": {Type: "string"},
		},
	}
	_, err := MapGoType(md, "", "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to format embedded object type name")
}

func TestExtractImports_PropError(t *testing.T) {
	// A property with an invalid GoType propagates the error through collectImports
	md := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"bad": {GoType: "github.com/pkg.", Type: "object"},
		},
	}
	// collectImports swallows ResolveGoTypeRef errors (err == nil check), so no error expected;
	// this exercises the properties loop path
	_, err := ExtractImports(md, "", "")
	require.NoError(t, err)
}

func TestExtractImports_ItemsPath(t *testing.T) {
	md := &ConfigMetadata{
		Type: "array",
		Items: &ConfigMetadata{
			Type:       "string",
			IsOptional: true,
		},
	}
	result, err := ExtractImports(md, "", "")
	require.NoError(t, err)
	require.Contains(t, result, "go.opentelemetry.io/collector/config/configoptional")
}

func TestExtractImports_DefsPath(t *testing.T) {
	md := &ConfigMetadata{
		Defs: map[string]*ConfigMetadata{
			"T": {
				Type:       "string",
				IsOptional: true,
			},
		},
	}
	result, err := ExtractImports(md, "", "")
	require.NoError(t, err)
	require.Contains(t, result, "go.opentelemetry.io/collector/config/configoptional")
}

func TestExtractImports_ContentSchema(t *testing.T) {
	md := &ConfigMetadata{
		ContentSchema: &ConfigMetadata{
			Type:   "string",
			Format: "duration",
		},
	}
	result, err := ExtractImports(md, "", "")
	require.NoError(t, err)
	require.Contains(t, result, "time")
}

func TestExtractValidators(t *testing.T) {
	tests := []struct {
		name     string
		metadata *ConfigMetadata
		expected []Validator
	}{
		{
			name: "no validators",
			metadata: &ConfigMetadata{
				Type: "string",
			},
			expected: []Validator{},
		},
		{
			name:     "nil config",
			metadata: nil,
			expected: []Validator{},
		},
		{
			name: "required string field",
			metadata: &ConfigMetadata{
				Type:     "object",
				Required: []string{"name"},
				Properties: map[string]*ConfigMetadata{
					"name": {Type: "string"},
				},
			},
			expected: []Validator{
				{
					FieldName:  "name",
					FieldType:  "string",
					IsRequired: true,
					IsOptional: false,
					IsPointer:  false,
					IsCallable: false,
					IsIterable: false,
				},
			},
		},
		{
			name: "required optional field",
			metadata: &ConfigMetadata{
				Type:     "object",
				Required: []string{"name"},
				Properties: map[string]*ConfigMetadata{
					"name": {Type: "string", IsOptional: true},
				},
			},
			expected: []Validator{
				{
					FieldName:  "name",
					FieldType:  "string",
					IsRequired: true,
					IsOptional: true,
					IsPointer:  false,
					IsCallable: false,
					IsIterable: false,
				},
			},
		},
		{
			name: "required pointer field",
			metadata: &ConfigMetadata{
				Type:     "object",
				Required: []string{"name"},
				Properties: map[string]*ConfigMetadata{
					"name": {Type: "string", IsPointer: true},
				},
			},
			expected: []Validator{
				{
					FieldName:  "name",
					FieldType:  "string",
					IsRequired: true,
					IsOptional: false,
					IsPointer:  true,
					IsCallable: false,
					IsIterable: false,
				},
			},
		},
		{
			name: "embedded object with required child emits IsCallable",
			metadata: &ConfigMetadata{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"server": {
						Type:     "object",
						Required: []string{"host"},
						Properties: map[string]*ConfigMetadata{
							"host": {Type: "string"},
						},
					},
				},
			},
			expected: []Validator{
				{
					FieldName:  "server",
					FieldType:  "object",
					IsRequired: false,
					IsPointer:  false,
					IsOptional: false,
					IsCallable: true,
					IsIterable: false,
				},
			},
		},
		{
			name: "embedded object without required children emits nothing",
			metadata: &ConfigMetadata{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"server": {
						Type: "object",
						Properties: map[string]*ConfigMetadata{
							"host": {Type: "string"},
						},
					},
				},
			},
			expected: []Validator{},
		},
		{
			name: "map with required child in value schema emits IsCallable with IsIterable",
			metadata: &ConfigMetadata{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"tags": {
						Type: "object",
						AdditionalProperties: &ConfigMetadata{
							Type:     "object",
							Required: []string{"value"},
							Properties: map[string]*ConfigMetadata{
								"value": {Type: "string"},
							},
						},
					},
				},
			},
			expected: []Validator{
				{
					FieldName:  "tags",
					FieldType:  "object",
					IsRequired: false,
					IsPointer:  false,
					IsOptional: false,
					IsCallable: true,
					IsIterable: true,
				},
			},
		},
		{
			name: "array with required child in items schema emits IsCallable with IsIterable",
			metadata: &ConfigMetadata{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"endpoints": {
						Type: "array",
						Items: &ConfigMetadata{
							Type:     "object",
							Required: []string{"url"},
							Properties: map[string]*ConfigMetadata{
								"url": {Type: "string"},
							},
						},
					},
				},
			},
			expected: []Validator{
				{
					FieldName:  "endpoints",
					FieldType:  "array",
					IsRequired: false,
					IsPointer:  false,
					IsOptional: false,
					IsCallable: true,
					IsIterable: true,
				},
			},
		},
		{
			name: "required field that also has children emits both validators",
			metadata: &ConfigMetadata{
				Type:     "object",
				Required: []string{"server"},
				Properties: map[string]*ConfigMetadata{
					"server": {
						Type:     "object",
						Required: []string{"host"},
						Properties: map[string]*ConfigMetadata{
							"host": {Type: "string"},
						},
					},
				},
			},
			expected: []Validator{
				{
					FieldName:  "server",
					FieldType:  "object",
					IsRequired: true,
					IsPointer:  false,
					IsOptional: false,
					IsCallable: false,
					IsIterable: false,
				},
				{
					FieldName:  "server",
					FieldType:  "object",
					IsRequired: false,
					IsPointer:  false,
					IsOptional: false,
					IsCallable: true,
					IsIterable: false,
				},
			},
		},
		{
			name: "array with no items emits nothing",
			metadata: &ConfigMetadata{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"ids": {
						Type:  "array",
						Items: nil,
					},
				},
			},
			expected: []Validator{},
		},
		{
			name: "object with additionalProperties but no required children emits nothing",
			metadata: &ConfigMetadata{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"labels": {
						Type: "object",
						AdditionalProperties: &ConfigMetadata{
							Type: "string",
						},
					},
				},
			},
			expected: []Validator{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := ExtractValidators(test.metadata, test.metadata)
			require.Equal(t, test.expected, result)
		})
	}
}

func TestExtractValidators_PropRef(t *testing.T) {
	root := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"server": {
				Ref: "server_config",
			},
		},
	}
	serverConfig := &ConfigMetadata{
		Required: []string{"host"},
		Properties: map[string]*ConfigMetadata{
			"host": {Type: "string"},
		},
	}

	propWithRef := &ConfigMetadata{
		Type:       "object",
		Ref:        "server_config",
		Required:   serverConfig.Required,
		Properties: serverConfig.Properties,
	}
	root.Properties["server"] = propWithRef

	result := ExtractValidators(root, root)
	require.Len(t, result, 1)
	require.Equal(t, Validator{
		FieldName:  "server",
		FieldType:  "ref",
		IsRequired: false,
		IsCallable: true,
		IsIterable: false,
	}, result[0])
}

func TestExtractValidators_InternalRefFromDefs(t *testing.T) {
	root := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"server_config": {
				Type:     "object",
				Required: []string{"host"},
				Properties: map[string]*ConfigMetadata{
					"host": {Type: "string"},
				},
			},
		},
	}
	md := &ConfigMetadata{
		Ref: "server_config",
	}

	result := ExtractValidators(root, md)
	require.Len(t, result, 1)
	require.Equal(t, Validator{
		FieldName:  "server_config",
		FieldType:  "ref",
		IsRequired: false,
		IsPointer:  false,
		IsOptional: false,
		IsCallable: true,
		IsIterable: false,
	}, result[0])
}

func TestExtractValidators_InternalRefFromDefs_NoValidators(t *testing.T) {
	root := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"plain_config": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"host": {Type: "string"},
				},
			},
		},
	}
	md := &ConfigMetadata{
		Ref: "plain_config",
	}

	result := ExtractValidators(root, md)
	require.Empty(t, result)
}

func TestExtractValidators_InternalRefFromDefs_RefNotFound(t *testing.T) {
	root := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{},
	}
	md := &ConfigMetadata{
		Ref: "missing_def",
	}

	result := ExtractValidators(root, md)
	require.Empty(t, result)
}

func TestExtractValidators_AllOf_WithRef(t *testing.T) {
	root := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"base_config": {
				Type:     "object",
				Required: []string{"endpoint"},
				Properties: map[string]*ConfigMetadata{
					"endpoint": {Type: "string"},
				},
			},
		},
	}
	md := &ConfigMetadata{
		Type: "object",
		AllOf: []*ConfigMetadata{
			{Ref: "base_config"},
		},
	}

	result := ExtractValidators(root, md)
	require.Len(t, result, 1)
	require.Equal(t, Validator{
		FieldName:  "base_config",
		FieldType:  "ref",
		IsRequired: false,
		IsPointer:  false,
		IsOptional: false,
		IsCallable: true,
		IsIterable: false,
	}, result[0])
}

func TestExtractValidators_AllOf_NoRef(t *testing.T) {
	root := &ConfigMetadata{
		Type: "object",
	}
	md := &ConfigMetadata{
		Type: "object",
		AllOf: []*ConfigMetadata{
			{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"host": {Type: "string"},
				},
			},
		},
	}

	result := ExtractValidators(root, md)
	require.Empty(t, result)
}

func TestExtractValidators_AllOf_RefWithNoValidators(t *testing.T) {
	// An allOf entry whose Ref resolves to a schema with no required fields
	// produces no validator.
	root := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"empty_base": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"host": {Type: "string"},
				},
			},
		},
	}
	md := &ConfigMetadata{
		Type: "object",
		AllOf: []*ConfigMetadata{
			{Ref: "empty_base"},
		},
	}

	result := ExtractValidators(root, md)
	require.Empty(t, result)
}

func TestNewCfgFns_ExtractValidators(t *testing.T) {
	fns := NewCfgFns("", "")
	extractValidators := fns["extractValidators"].(func(*ConfigMetadata, *ConfigMetadata) []Validator)

	require.Nil(t, extractValidators(nil, nil))

	md := &ConfigMetadata{
		Type:     "object",
		Required: []string{"name"},
		Properties: map[string]*ConfigMetadata{
			"name": {Type: "string"},
		},
	}
	result := extractValidators(md, md)
	require.Len(t, result, 1)
	require.Equal(t, "name", result[0].FieldName)
	require.True(t, result[0].IsRequired)
}
