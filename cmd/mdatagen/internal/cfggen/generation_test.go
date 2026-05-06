// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen

import (
	"maps"
	"slices"
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
		goType   string
		expected string
	}{
		{
			name:     "date-time format",
			goType:   "time.Time",
			expected: "time.Time",
		},
		{
			name:     "duration format",
			goType:   "time.Duration",
			expected: "time.Duration",
		},
		{
			name:     "no format",
			goType:   "",
			expected: "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := &ConfigMetadata{
				Type:   "string",
				GoType: tt.goType,
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
				Items: &ConfigMetadata{ResolvedFrom: "./internal/metadata.custom_type"},
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
				AdditionalProperties: &ConfigMetadata{ResolvedFrom: "./internal/metadata.custom_type"},
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
				ResolvedFrom: tt.ref,
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
				GoType: "time.Time",
			},
			expected: []string{"time"},
		},
		{
			name: "time import for duration format",
			metadata: &ConfigMetadata{
				Type:   "string",
				GoType: "time.Duration",
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
				ResolvedFrom: "go.opentelemetry.io/collector/component.Config",
			},
			expected: []string{"go.opentelemetry.io/collector/component"},
		},
		{
			name: "no import for internal reference",
			metadata: &ConfigMetadata{
				ResolvedFrom: "my_type",
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
				ResolvedFrom: "/config/confighttp.client_config",
			},
			expected: []string{"go.opentelemetry.io/collector/config/confighttp"},
		},
		{
			name: "local relative reference",
			metadata: &ConfigMetadata{
				ResolvedFrom: "./internal/metadata.custom_type",
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

func TestExtractImports_ResolvedReferenceIncludesDefaultOverrideImports(t *testing.T) {
	md := &ConfigMetadata{
		Type:         "object",
		ResolvedFrom: "go.opentelemetry.io/collector/scraper/scraperhelper.ControllerConfig",
		Properties: map[string]*ConfigMetadata{
			"timeout": {
				Type:   "string",
				GoType: "time.Duration",
			},
		},
		Default: defaultValue(map[string]any{"timeout": "30s"}),
	}

	result, err := ExtractImports(md, "", "")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"go.opentelemetry.io/collector/scraper/scraperhelper", "time"}, result)
}

func TestCollectCustomDefaultImports(t *testing.T) {
	tests := []struct {
		name         string
		metadata     *ConfigMetadata
		defaultValue any
		expected     []string
	}{
		{
			name:         "nil metadata",
			defaultValue: map[string]any{"timeout": "30s"},
		},
		{
			name: "ignored default",
			metadata: &ConfigMetadata{
				Type:     "object",
				GoStruct: GoStructConfig{IgnoreDefault: true},
				Properties: map[string]*ConfigMetadata{
					"timeout": {Type: "string", GoType: "time.Duration"},
				},
			},
			defaultValue: map[string]any{"timeout": "30s"},
		},
		{
			name: "map schema with additional properties does not inspect entries",
			metadata: &ConfigMetadata{
				Type:                 "object",
				AdditionalProperties: &ConfigMetadata{Type: "string", GoType: "time.Duration"},
			},
			defaultValue: map[string]any{"timeout": "30s"},
		},
		{
			name: "missing property is ignored",
			metadata: &ConfigMetadata{
				Type:       "object",
				Properties: map[string]*ConfigMetadata{},
			},
			defaultValue: map[string]any{"timeout": "30s"},
		},
		{
			name: "map default imports overridden property types",
			metadata: &ConfigMetadata{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"timeout": {Type: "string", GoType: "time.Duration"},
				},
			},
			defaultValue: map[string]any{"timeout": "30s"},
			expected:     []string{"time"},
		},
		{
			name: "array default imports object item property types",
			metadata: &ConfigMetadata{
				Type: "array",
				Items: &ConfigMetadata{
					Type: "object",
					Properties: map[string]*ConfigMetadata{
						"timestamp": {Type: "string", GoType: "time.Time"},
					},
				},
			},
			defaultValue: []any{map[string]any{"timestamp": "2026-04-30T00:00:00Z"}},
			expected:     []string{"time"},
		},
		{
			name: "array default with non-object item does not inspect entries",
			metadata: &ConfigMetadata{
				Type:  "array",
				Items: &ConfigMetadata{Type: "string", GoType: "time.Duration"},
			},
			defaultValue: []any{"30s"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imports := map[string]bool{}
			require.NoError(t, collectCustomDefaultImports(tt.metadata, tt.defaultValue, imports, "", ""))
			require.ElementsMatch(t, tt.expected, slices.Collect(maps.Keys(imports)))
		})
	}
}

func TestExtractImports_InternalResolvedReferenceIncludesNestedImports(t *testing.T) {
	md := &ConfigMetadata{
		Type:         "object",
		ResolvedFrom: "plain_config",
		Properties: map[string]*ConfigMetadata{
			"timeout": {
				Type:   "string",
				GoType: "time.Duration",
			},
		},
	}

	result, err := ExtractImports(md, "", "")
	require.NoError(t, err)
	require.Equal(t, []string{"time"}, result)
}

func TestExtractImports_ResolvedReferenceOptional(t *testing.T) {
	md := &ConfigMetadata{
		Type:         "object",
		ResolvedFrom: "go.opentelemetry.io/collector/config/confighttp.ClientConfig",
		IsOptional:   true,
	}

	result, err := ExtractImports(md, "", "")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{
		"go.opentelemetry.io/collector/config/confighttp",
		"go.opentelemetry.io/collector/config/configoptional",
	}, result)
}

func TestExtractImports_Nested(t *testing.T) {
	md := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"timeout": {
				Type:   "string",
				GoType: "time.Duration",
			},
			"nested": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"timestamp": {
						Type:   "string",
						GoType: "time.Time",
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
				GoType: "time.Duration",
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
			GoType: "time.Time",
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
			GoType: "time.Duration",
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
				GoType: "time.Time",
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

func TestExtractDefs_MapValueObject(t *testing.T) {
	md := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"labels": {
				Type: "object",
				AdditionalProperties: &ConfigMetadata{
					Type: "object",
					Properties: map[string]*ConfigMetadata{
						"name": {Type: "string"},
					},
				},
			},
			"custom": {
				Type:   "object",
				GoType: "github.com/example/pkg.Custom",
				Properties: map[string]*ConfigMetadata{
					"ignored": {Type: "string"},
				},
			},
		},
	}

	result := ExtractDefs(md)
	require.Len(t, result, 1)
	require.Contains(t, result, "labels")
	require.Same(t, md.Properties["labels"].AdditionalProperties, result["labels"])
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

func TestExtractDefs_InternalResolvedReference(t *testing.T) {
	md := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"config": {
				Type:         "object",
				ResolvedFrom: "plain_config",
				Defs: map[string]*ConfigMetadata{
					"nested_def": {Type: "string"},
				},
				Properties: map[string]*ConfigMetadata{
					"nested": {
						Type: "object",
						Properties: map[string]*ConfigMetadata{
							"name": {Type: "string"},
						},
					},
				},
			},
		},
	}

	result := ExtractDefs(md)
	require.Len(t, result, 3)
	require.Same(t, md.Properties["config"], result["plain_config"])
	require.Contains(t, result, "nested")
	require.Contains(t, result, "nested_def")
}

func TestExtractDefs_SkipsExternalResolvedReference(t *testing.T) {
	md := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"config": {
				Type:         "object",
				ResolvedFrom: "go.opentelemetry.io/collector/config/confighttp.ClientConfig",
				Defs: map[string]*ConfigMetadata{
					"nested_def": {Type: "string"},
				},
				Properties: map[string]*ConfigMetadata{
					"nested": {
						Type: "object",
						Properties: map[string]*ConfigMetadata{
							"name": {Type: "string"},
						},
					},
				},
			},
		},
	}

	result := ExtractDefs(md)
	require.Empty(t, result)
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

func TestExtractDefs_AllOfInternalRef(t *testing.T) {
	// Mirrors the schema:
	//   type: object
	//   $defs:
	//     embedded_type:
	//       type: object
	//       properties:
	//         field: {type: string}
	//   allOf:
	//     - $ref: embedded_type
	//
	// After resolver runs, the allOf entry carries ResolvedFrom: "embedded_type".
	// ExtractDefs must emit "embedded_type" so that EmbeddedType struct is generated.
	embeddedType := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"field": {Type: "string"},
		},
		ResolvedFrom: "embedded_type",
	}
	md := &ConfigMetadata{
		Type:  "object",
		AllOf: []*ConfigMetadata{embeddedType},
	}

	result := ExtractDefs(md)
	require.Len(t, result, 1)
	require.Contains(t, result, "embedded_type")
	require.Same(t, embeddedType, result["embedded_type"])
}

func TestExtractDefs_AllOfMultipleInternalRefs(t *testing.T) {
	// Two allOf entries, each resolving from a distinct internal definition.
	typeA := &ConfigMetadata{
		Type:         "object",
		ResolvedFrom: "type_a",
		Properties:   map[string]*ConfigMetadata{"x": {Type: "string"}},
	}
	typeB := &ConfigMetadata{
		Type:         "object",
		ResolvedFrom: "type_b",
		Properties:   map[string]*ConfigMetadata{"y": {Type: "integer"}},
	}
	md := &ConfigMetadata{
		Type:  "object",
		AllOf: []*ConfigMetadata{typeA, typeB},
	}

	result := ExtractDefs(md)
	require.Len(t, result, 2)
	require.Same(t, typeA, result["type_a"])
	require.Same(t, typeB, result["type_b"])
}

func TestExtractDefs_AllOfExternalRefSkipped(t *testing.T) {
	// An allOf entry that resolved from an external reference must NOT be emitted
	// (there is nothing to generate for it locally).
	md := &ConfigMetadata{
		Type: "object",
		AllOf: []*ConfigMetadata{
			{
				Type:         "object",
				ResolvedFrom: "go.opentelemetry.io/collector/scraper/scraperhelper.ControllerConfig",
				Properties:   map[string]*ConfigMetadata{"timeout": {Type: "string"}},
			},
		},
	}

	result := ExtractDefs(md)
	require.Empty(t, result)
}

func TestExtractDefs_AllOfInternalRefWithNestedProperties(t *testing.T) {
	// An allOf-referenced type that itself contains an inline nested object must
	// also emit the nested type definition.
	embedded := &ConfigMetadata{
		Type:         "object",
		ResolvedFrom: "embedded_type",
		Properties: map[string]*ConfigMetadata{
			"nested": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"value": {Type: "string"},
				},
			},
		},
	}
	md := &ConfigMetadata{
		Type:  "object",
		AllOf: []*ConfigMetadata{embedded},
	}

	result := ExtractDefs(md)
	require.Contains(t, result, "embedded_type")
	require.Contains(t, result, "nested")
}

func TestNewCfgFns_ExtractImports(t *testing.T) {
	fns := NewCfgFns("go.opentelemetry.io/collector", "go.opentelemetry.io/collector/comp")

	extractImports := fns["extractImports"].(func(*ConfigMetadata) []string)

	// nil input returns nil
	require.Nil(t, extractImports(nil))

	// valid input returns imports
	md := &ConfigMetadata{Type: "string", GoType: "time.Duration"}
	result := extractImports(md)
	require.Contains(t, result, "time")

	// unresolvable GoType: extractImports panics
	errMd := &ConfigMetadata{GoType: "github.com/pkg."}
	require.Panics(t, func() { extractImports(errMd) })
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
	require.Panics(t, func() {
		mapGoType(&ConfigMetadata{GoType: "github.com/pkg."}, "field")
	})
}

func TestNewCfgFns_PublicType(t *testing.T) {
	fns := NewCfgFns("", "")

	publicType := fns["publicType"].(func(string) string)

	require.Equal(t, "MyType", publicType("my_type"))
	require.Equal(t, "component.Config", publicType("go.opentelemetry.io/collector/component.Config"))
	require.Panics(t, func() {
		publicType("github.com/pkg.")
	})
}

func TestNewCfgFns_EmbeddedName(t *testing.T) {
	fns := NewCfgFns("", "")
	embeddedName := fns["embeddedName"].(func(*ConfigMetadata) string)

	require.Equal(t, "MyType", embeddedName(&ConfigMetadata{EmbeddedName: "my_type"}))
	require.Equal(t, "MyType", embeddedName(&ConfigMetadata{ResolvedFrom: "my_type"}))
	require.Panics(t, func() { embeddedName(&ConfigMetadata{}) })
}

func TestCamelVar(t *testing.T) {
	require.Equal(t, "controllerConfig",
		CamelVar("go.opentelemetry.io/collector/scraper/scraperhelper.controller_config"))
	require.Equal(t, "plainConfig", CamelVar("plain_config"))
	require.Panics(t, func() { CamelVar("") })
}

func TestNewCfgFns_CamelVar(t *testing.T) {
	fns := NewCfgFns("", "")
	camelVar := fns["camelVar"].(func(string) string)
	require.Equal(t, "controllerConfig",
		camelVar("go.opentelemetry.io/collector/scraper/scraperhelper.controller_config"))
	require.Panics(t, func() { camelVar("") })
}

func TestWithCfgFns(t *testing.T) {
	base := map[string]any{"existing": "value"}
	result := WithCfgFns(base, "", "")

	require.Equal(t, "value", result["existing"])
	require.Contains(t, result, "mapGoType")
	require.Contains(t, result, "extractImports")
	require.Contains(t, result, "extractDefs")
	require.Contains(t, result, "formatDefaultValue")
	require.Contains(t, result, "mapCustomDefaults")
	require.Contains(t, result, "hasDefaultValue")
	require.Contains(t, result, "publicType")
	require.Contains(t, result, "camelVar")
}

func TestResolveGoType_CustomTypeFormatError(t *testing.T) {
	// GoType with invalid empty type name after dot triggers FormatTypeName error
	md := &ConfigMetadata{GoType: "github.com/pkg."}
	_, err := MapGoType(md, "field", "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to format custom type")
}

func TestResolveGoType_RefFormatError(t *testing.T) {
	// ResolvedFrom with invalid empty type name after dot triggers FormatTypeName error
	md := &ConfigMetadata{ResolvedFrom: "github.com/pkg."}
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
	md := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"bad": {GoType: "github.com/pkg.", Type: "object"},
		},
	}
	_, err := ExtractImports(md, "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to resolve import for custom type")
}

func TestExtractImports_GoTypeError(t *testing.T) {
	md := &ConfigMetadata{GoType: "github.com/pkg."}
	_, err := ExtractImports(md, "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to resolve import for custom type")
}

func TestExtractImports_ResolvedFromError(t *testing.T) {
	md := &ConfigMetadata{ResolvedFrom: "github.com/pkg."}
	_, err := ExtractImports(md, "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to resolve import for reference")
}

func TestExtractImports_ItemsError(t *testing.T) {
	md := &ConfigMetadata{
		Type:  "array",
		Items: &ConfigMetadata{GoType: "github.com/pkg."},
	}
	_, err := ExtractImports(md, "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to resolve import for custom type")
}

func TestExtractImports_AllOfError(t *testing.T) {
	md := &ConfigMetadata{
		Type:  "object",
		AllOf: []*ConfigMetadata{{GoType: "github.com/pkg."}},
	}
	_, err := ExtractImports(md, "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to resolve import for custom type")
}

func TestExtractImports_DefsError(t *testing.T) {
	md := &ConfigMetadata{
		Defs: map[string]*ConfigMetadata{
			"bad": {GoType: "github.com/pkg."},
		},
	}
	_, err := ExtractImports(md, "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to resolve import for custom type")
}

func TestExtractImports_AdditionalPropertiesError(t *testing.T) {
	md := &ConfigMetadata{
		Type:                 "object",
		AdditionalProperties: &ConfigMetadata{GoType: "github.com/pkg."},
	}
	_, err := ExtractImports(md, "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to resolve import for custom type")
}

func TestExtractImports_ContentSchemaError(t *testing.T) {
	md := &ConfigMetadata{
		ContentSchema: &ConfigMetadata{GoType: "github.com/pkg."},
	}
	_, err := ExtractImports(md, "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to resolve import for custom type")
}

func TestExtractImports_ExternalResolvedFromDefaultsError(t *testing.T) {
	// collectImports on the prop itself fails (line 294-295 in collectCustomDefaultImports)
	md := &ConfigMetadata{
		ResolvedFrom: "go.opentelemetry.io/collector/scraper/scraperhelper.ControllerConfig",
		Properties: map[string]*ConfigMetadata{
			"timeout": {GoType: "github.com/pkg."},
		},
		Default: defaultValue(map[string]any{"timeout": "30s"}),
	}
	_, err := ExtractImports(md, "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to resolve import for custom type")
}

func TestExtractImports_ExternalResolvedFromNestedDefaultsError(t *testing.T) {
	// "nested" has an external ResolvedFrom with no Default, so collectImports(nested) succeeds:
	// it short-circuits after processing nested.Default=nil without descending into nested.Properties.
	// collectCustomDefaultImports(nested, {"bad": "x"}) then walks the override value and calls
	// collectImports on nested.Properties["bad"] which has an invalid GoType — this error propagates
	// back to the caller at line 297-298 in the outer collectCustomDefaultImports.
	md := &ConfigMetadata{
		ResolvedFrom: "go.opentelemetry.io/collector/scraper/scraperhelper.ControllerConfig",
		Properties: map[string]*ConfigMetadata{
			"nested": {
				ResolvedFrom: "go.opentelemetry.io/collector/config/confighttp.ClientConfig",
				Properties: map[string]*ConfigMetadata{
					"bad": {GoType: "github.com/pkg."},
				},
			},
		},
		Default: defaultValue(map[string]any{
			"nested": map[string]any{"bad": "value"},
		}),
	}
	_, err := ExtractImports(md, "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to resolve import for custom type")
}

func TestExtractImports_ExternalResolvedFromArrayDefaultsError(t *testing.T) {
	// collectCustomDefaultImports over array items fails (line 306-307)
	md := &ConfigMetadata{
		ResolvedFrom: "go.opentelemetry.io/collector/scraper/scraperhelper.ControllerConfig",
		Type:         "array",
		Items: &ConfigMetadata{
			Type: "object",
			Properties: map[string]*ConfigMetadata{
				"bad": {GoType: "github.com/pkg."},
			},
		},
		Default: defaultValue([]any{map[string]any{"bad": "value"}}),
	}
	_, err := ExtractImports(md, "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to resolve import for custom type")
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
			GoType: "time.Duration",
		},
	}
	result, err := ExtractImports(md, "", "")
	require.NoError(t, err)
	require.Contains(t, result, "time")
}

func TestExtractImports_Pattern(t *testing.T) {
	tests := []struct {
		name     string
		metadata *ConfigMetadata
	}{
		{
			name:     "direct pattern field",
			metadata: &ConfigMetadata{Type: "string", Pattern: `^[a-z]+$`},
		},
		{
			name: "pattern in nested property",
			metadata: &ConfigMetadata{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"name": {Type: "string", Pattern: `^[a-z]+$`},
				},
			},
		},
		{
			name: "pattern in array items",
			metadata: &ConfigMetadata{
				Type:  "array",
				Items: &ConfigMetadata{Type: "string", Pattern: `^[a-z]+$`},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractImports(tt.metadata, "", "")
			require.NoError(t, err)
			require.Contains(t, result, "regexp")
		})
	}
}

func TestExtractImports_NoPatternNoRegexpImport(t *testing.T) {
	md := &ConfigMetadata{Type: "string"}
	result, err := ExtractImports(md, "", "")
	require.NoError(t, err)
	require.NotContains(t, result, "regexp")
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
					IsOptional: false,
					IsPointer:  false,
					Rules:      ValidationRules{Required: true},
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
					IsOptional: true,
					IsPointer:  false,
					Rules:      ValidationRules{Required: true},
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
					IsOptional: false,
					IsPointer:  true,
					Rules:      ValidationRules{Required: true},
				},
			},
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
			result := ExtractValidators(test.metadata)
			require.Equal(t, test.expected, result)
		})
	}
}

func TestExtractValidators_StringValidators(t *testing.T) {
	maxLen := 64
	minLen := 3

	tests := []struct {
		name     string
		metadata *ConfigMetadata
		expected []Validator
	}{
		{
			name: "maxLength only",
			metadata: &ConfigMetadata{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"name": {Type: "string", MaxLength: &maxLen},
				},
			},
			expected: []Validator{
				{
					FieldName: "name",
					FieldType: "string",
					Rules:     ValidationRules{MaxLength: &maxLen},
				},
			},
		},
		{
			name: "minLength only",
			metadata: &ConfigMetadata{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"name": {Type: "string", MinLength: &minLen},
				},
			},
			expected: []Validator{
				{
					FieldName: "name",
					FieldType: "string",
					Rules:     ValidationRules{MinLength: &minLen},
				},
			},
		},
		{
			name: "pattern only",
			metadata: &ConfigMetadata{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"name": {Type: "string", Pattern: `^[a-z]+$`},
				},
			},
			expected: []Validator{
				{
					FieldName: "name",
					FieldType: "string",
					Rules:     ValidationRules{Pattern: Ptr(`^[a-z]+$`)},
				},
			},
		},
		{
			name: "all string validators without required",
			metadata: &ConfigMetadata{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"name": {Type: "string", MinLength: &minLen, MaxLength: &maxLen, Pattern: `^[a-z]+$`},
				},
			},
			expected: []Validator{
				{
					FieldName: "name",
					FieldType: "string",
					Rules:     ValidationRules{MinLength: &minLen, MaxLength: &maxLen, Pattern: Ptr(`^[a-z]+$`)},
				},
			},
		},
		{
			name: "required combined with string validators",
			metadata: &ConfigMetadata{
				Type:     "object",
				Required: []string{"name"},
				Properties: map[string]*ConfigMetadata{
					"name": {Type: "string", MinLength: &minLen, MaxLength: &maxLen, Pattern: `^[a-z]+$`},
				},
			},
			expected: []Validator{
				{
					FieldName: "name",
					FieldType: "string",
					Rules:     ValidationRules{Required: true, MinLength: &minLen, MaxLength: &maxLen, Pattern: Ptr(`^[a-z]+$`)},
				},
			},
		},
		{
			name: "nil minLength and maxLength produces no validator",
			metadata: &ConfigMetadata{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"name": {Type: "string"},
				},
			},
			expected: []Validator{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractValidators(tt.metadata)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractValidators_InternalRefFromDefs_NoValidators(t *testing.T) {
	md := &ConfigMetadata{
		Ref: "plain_config",
	}

	result := ExtractValidators(md)
	require.Empty(t, result)
}

func TestExtractValidators_InternalRefFromDefs_RefNotFound(t *testing.T) {
	md := &ConfigMetadata{
		Ref: "missing_def",
	}

	result := ExtractValidators(md)
	require.Empty(t, result)
}

func TestExtractValidators_AllOf_NoRef(t *testing.T) {
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

	result := ExtractValidators(md)
	require.Empty(t, result)
}

func TestExtractValidators_AllOf_RefWithNoValidators(t *testing.T) {
	md := &ConfigMetadata{
		Type: "object",
		AllOf: []*ConfigMetadata{
			{Ref: "empty_base"},
		},
	}

	result := ExtractValidators(md)
	require.Empty(t, result)
}

func TestExtractValidators_CustomValidator(t *testing.T) {
	nonNil := new(CustomValidatorConfig)
	md := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"http_client": {
				Ref:       "/config/confighttp.client_config",
				IsPointer: true,
				GoStruct:  GoStructConfig{CustomValidator: nonNil},
			},
		},
	}

	result := ExtractValidators(md)
	require.Len(t, result, 1)
	require.Equal(t, "http_client", result[0].FieldName)
	require.Equal(t, "validateHTTPClient", result[0].CustomValidator)
	require.Empty(t, result[0].Rules)
	require.True(t, result[0].IsPointer)
}

func TestExtractValidators_RequiredAndCustomValidator(t *testing.T) {
	nonNil := new(CustomValidatorConfig)
	md := &ConfigMetadata{
		Type:     "object",
		Required: []string{"http_client"},
		Properties: map[string]*ConfigMetadata{
			"http_client": {
				Ref:       "/config/confighttp.client_config",
				IsPointer: true,
				GoStruct:  GoStructConfig{CustomValidator: nonNil},
			},
		},
	}

	result := ExtractValidators(md)
	require.Len(t, result, 2)

	// First validator is the required check
	require.Equal(t, "http_client", result[0].FieldName)
	require.NotNil(t, result[0].Rules.Required)
	require.True(t, result[0].Rules.Required)
	require.Empty(t, result[0].CustomValidator)

	// Second validator is the custom validator
	require.Equal(t, "http_client", result[1].FieldName)
	require.Empty(t, result[1].Rules)
	require.Equal(t, "validateHTTPClient", result[1].CustomValidator)
}

func TestExtractValidators_RootCustomValidatorLast(t *testing.T) {
	md := &ConfigMetadata{
		Type:     "object",
		Required: []string{"http_client"},
		GoStruct: GoStructConfig{CustomValidator: &CustomValidatorConfig{Name: "validateConfig"}},
		Properties: map[string]*ConfigMetadata{
			"http_client": {
				Ref:       "/config/confighttp.client_config",
				IsPointer: true,
			},
		},
	}

	result := ExtractValidators(md)
	require.Len(t, result, 2)
	require.NotNil(t, result[1].Rules.Required)
	require.True(t, result[1].Rules.Required)
	require.Empty(t, result[1].CustomValidator)
	require.Equal(t, ".", result[0].FieldName)
	require.Equal(t, "validateConfig", result[0].CustomValidator)
}

func TestExtractValidators_NoCustomValidator(t *testing.T) {
	md := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"endpoint": {
				Type:     "string",
				GoStruct: GoStructConfig{}, // CustomValidator is nil
			},
		},
	}

	result := ExtractValidators(md)
	require.Empty(t, result)
}

func TestValidationRules_HasValueRule(t *testing.T) {
	tests := []struct {
		name     string
		rules    ValidationRules
		expected bool
	}{
		{
			name:     "empty validators map",
			rules:    ValidationRules{},
			expected: false,
		},
		{
			name:     "only required",
			rules:    ValidationRules{Required: true},
			expected: false,
		},
		{
			name:     "maxLength",
			rules:    ValidationRules{MaxLength: Ptr(64)},
			expected: true,
		},
		{
			name:     "minLength",
			rules:    ValidationRules{MinLength: Ptr(1)},
			expected: true,
		},
		{
			name:     "pattern",
			rules:    ValidationRules{Pattern: Ptr(`^[a-z]+$`)},
			expected: true,
		},
		{
			name:     "required and pattern",
			rules:    ValidationRules{Required: true, Pattern: Ptr(`^[a-z]+$`)},
			expected: true,
		},
		{
			name:     "nil validators map",
			rules:    ValidationRules{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.rules.HasValueRule())
		})
	}
}

func TestResolveType(t *testing.T) {
	tests := []struct {
		name     string
		metadata *ConfigMetadata
		expected string
	}{
		{
			name:     "reference",
			metadata: &ConfigMetadata{ResolvedFrom: "go.opentelemetry.io/collector/config/confighttp.ClientConfig"},
			expected: "ref",
		},
		{
			name:     "date time",
			metadata: &ConfigMetadata{Type: "string", GoType: "time.Time"},
			expected: "datetime",
		},
		{
			name:     "duration",
			metadata: &ConfigMetadata{Type: "string", GoType: "time.Duration"},
			expected: "duration",
		},
		{
			name: "map",
			metadata: &ConfigMetadata{
				Type:                 "object",
				AdditionalProperties: &ConfigMetadata{Type: "string"},
			},
			expected: "map",
		},
		{
			name:     "plain type",
			metadata: &ConfigMetadata{Type: "boolean"},
			expected: "boolean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, resolveType(tt.metadata))
		})
	}
}

func TestNewCfgFns_ExtractValidators(t *testing.T) {
	fns := NewCfgFns("", "")
	extractValidators := fns["extractValidators"].(func(*ConfigMetadata) []Validator)

	require.Nil(t, extractValidators(nil))

	md := &ConfigMetadata{
		Type:     "object",
		Required: []string{"name"},
		Properties: map[string]*ConfigMetadata{
			"name": {Type: "string"},
		},
	}
	result := extractValidators(md)
	require.Len(t, result, 1)
	require.Equal(t, "name", result[0].FieldName)
	require.NotNil(t, result[0].Rules.Required)
	require.True(t, result[0].Rules.Required)
}

func Ptr[T any](v T) *T {
	return &v
}

func TestFormatDefaultValue_ScalarDefaults(t *testing.T) {
	tests := []struct {
		name         string
		schema       *ConfigMetadata
		propName     string
		defaultValue any
		expected     string
	}{
		{
			name:         "string",
			schema:       &ConfigMetadata{Type: "string"},
			propName:     "endpoint",
			defaultValue: defaultValue("http://localhost:8080"),
			expected:     `"http://localhost:8080"`,
		},
		{
			name:         "duration",
			schema:       &ConfigMetadata{Type: "string", GoType: "time.Duration"},
			propName:     "timeout",
			defaultValue: defaultValue("30s"),
			expected:     "30*time.Second",
		},
		{
			name:         "duration zero int",
			schema:       &ConfigMetadata{Type: "string", GoType: "time.Duration"},
			propName:     "timeout",
			defaultValue: defaultValue(0),
			expected:     "0",
		},
		{
			name:         "duration zero float",
			schema:       &ConfigMetadata{Type: "string", GoType: "time.Duration"},
			propName:     "timeout",
			defaultValue: defaultValue(float64(0)),
			expected:     "0",
		},
		{
			name:         "optional duration",
			schema:       &ConfigMetadata{Type: "string", GoType: "time.Duration", IsOptional: true},
			propName:     "interval",
			defaultValue: defaultValue("10s"),
			expected:     "configoptional.Some(10*time.Second)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, FormatDefaultValue(tt.schema, tt.propName, tt.defaultValue, "", ""))
		})
	}
}

func TestRenderDurationExpr_InvalidInputs(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{
			name:  "invalid duration string",
			value: "not-a-duration",
		},
		{
			name:  "unsupported type",
			value: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, ok := renderDurationExpr(tt.value)
			require.False(t, ok)
			require.Empty(t, expr)
		})
	}
}

func TestFormatDefaultValue_MapDefault(t *testing.T) {
	md := &ConfigMetadata{
		Type:                 "object",
		AdditionalProperties: &ConfigMetadata{Type: "string"},
	}

	require.Equal(t, `map[string]string{"env": "prod"}`, FormatDefaultValue(md, "labels", defaultValue(map[string]any{"env": "prod"}), "", ""))
}

func TestFormatDefaultValue_OptionalObjectDefault(t *testing.T) {
	md := &ConfigMetadata{
		Type:       "object",
		IsOptional: true,
		Properties: map[string]*ConfigMetadata{
			"endpoint": {Type: "string", Default: defaultValue("localhost")},
		},
	}

	require.Equal(t, "configoptional.Default(NewDefaultClient())", FormatDefaultValue(md, "client", defaultValue(map[string]any{"endpoint": "localhost"}), "", ""))
}

func TestFormatDefaultValue_PointerArrayOfObjects(t *testing.T) {
	md := &ConfigMetadata{
		Type:      "array",
		IsPointer: true,
		Items: &ConfigMetadata{
			Type: "object",
			Properties: map[string]*ConfigMetadata{
				"url": {Type: "string", Default: defaultValue("http://example.com")},
			},
		},
	}

	require.Equal(t, "&[]TargetsItem{NewDefaultTargetsItem()}", FormatDefaultValue(md, "targets", defaultValue([]any{map[string]any{}}), "", ""))
}

func TestFormatDefaultValue_NilDefault(t *testing.T) {
	tests := []struct {
		name     string
		metadata *ConfigMetadata
		expected string
	}{
		{
			name:     "plain",
			metadata: &ConfigMetadata{Type: "string"},
			expected: "",
		},
		{
			name:     "pointer",
			metadata: &ConfigMetadata{Type: "string", IsPointer: true},
			expected: "nil",
		},
		{
			name:     "optional",
			metadata: &ConfigMetadata{Type: "string", IsOptional: true},
			expected: "configoptional.None[string]()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, FormatDefaultValue(tt.metadata, "endpoint", defaultValue(nil), "", ""))
		})
	}
}

func TestFormatBaseValue(t *testing.T) {
	md := &ConfigMetadata{
		Type:       "string",
		IsPointer:  true,
		IsOptional: true,
	}

	require.Equal(t, `"localhost"`, FormatBaseValue(md, "endpoint", defaultValue("localhost"), "", ""))
	require.Empty(t, FormatBaseValue(
		&ConfigMetadata{Type: "string", GoStruct: GoStructConfig{IgnoreDefault: true}},
		"endpoint",
		defaultValue("localhost"),
		"",
		"",
	))
}

func TestFormatDefaultValue_UnsetDefault(t *testing.T) {
	require.Empty(t, FormatDefaultValue(&ConfigMetadata{Type: "integer"}, "port", nil, "", ""))
}

func TestFormatDefaultValue_Panics(t *testing.T) {
	tests := []struct {
		name         string
		metadata     *ConfigMetadata
		defaultValue any
	}{
		{
			name: "invalid reference",
			metadata: &ConfigMetadata{
				ResolvedFrom: "github.com/pkg.",
				Default:      defaultValue(map[string]any{}),
			},
			defaultValue: defaultValue(map[string]any{}),
		},
		{
			name: "invalid array default value",
			metadata: &ConfigMetadata{
				Type:  "array",
				Items: &ConfigMetadata{Type: "string"},
			},
			defaultValue: defaultValue("localhost"),
		},
		{
			name: "invalid array item type",
			metadata: &ConfigMetadata{
				Type:  "array",
				Items: &ConfigMetadata{Type: "unknown"},
			},
			defaultValue: defaultValue([]any{"localhost"}),
		},
		{
			name: "invalid map default value",
			metadata: &ConfigMetadata{
				Type:                 "object",
				AdditionalProperties: &ConfigMetadata{Type: "string"},
			},
			defaultValue: defaultValue("localhost"),
		},
		{
			name: "invalid map value type",
			metadata: &ConfigMetadata{
				Type:                 "object",
				AdditionalProperties: &ConfigMetadata{Type: "unknown"},
			},
			defaultValue: defaultValue(map[string]any{"endpoint": "localhost"}),
		},
		{
			name:         "invalid duration default value",
			metadata:     &ConfigMetadata{Type: "string", GoType: "time.Duration"},
			defaultValue: defaultValue("invalid-duration"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Panics(t, func() {
				FormatDefaultValue(tt.metadata, "endpoint", tt.defaultValue, "", "")
			})
		})
	}
}

func TestWrapDefaultValue(t *testing.T) {
	tests := []struct {
		name     string
		metadata *ConfigMetadata
		expected string
	}{
		{
			name:     "plain value",
			metadata: &ConfigMetadata{},
			expected: "defaultValue",
		},
		{
			name:     "pointer",
			metadata: &ConfigMetadata{IsPointer: true},
			expected: "&defaultValue",
		},
		{
			name:     "optional",
			metadata: &ConfigMetadata{IsOptional: true},
			expected: "configoptional.Some(defaultValue)",
		},
		{
			name:     "pointer optional",
			metadata: &ConfigMetadata{IsPointer: true, IsOptional: true},
			expected: "&defaultValue",
		},
		{
			name: "optional object",
			metadata: &ConfigMetadata{
				Type:       "object",
				IsOptional: true,
				Properties: map[string]*ConfigMetadata{
					"name": {Type: "string"},
				},
			},
			expected: "configoptional.Default(defaultValue)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, WrapDefaultValue(tt.metadata, "defaultValue"))
		})
	}
}

func TestFormatDefaultValue_ResolvedReferenceWithDefaults(t *testing.T) {
	md := &ConfigMetadata{
		Type:         "object",
		ResolvedFrom: "go.opentelemetry.io/collector/config/confighttp.ClientConfig",
		Properties: map[string]*ConfigMetadata{
			"timeout": {Type: "string", GoType: "time.Duration", Default: defaultValue("30s")},
		},
	}

	require.Equal(t,
		"confighttp.NewDefaultClientConfig()",
		FormatDefaultValue(md, "client", defaultValue(map[string]any{"timeout": "30s"}), "go.opentelemetry.io/collector", "go.opentelemetry.io/collector/cmd/mdatagen/internal/samplescraper"),
	)
}

func TestFormatDefaultValue_ResolvedReferenceWithoutDefaults(t *testing.T) {
	// An external ref with no property defaults must not generate a NewDefault... call.
	md := &ConfigMetadata{
		Type:         "object",
		ResolvedFrom: "go.opentelemetry.io/collector/config/confighttp.ClientConfig",
	}

	require.Empty(t,
		FormatDefaultValue(md, "client", defaultValue(map[string]any{}), "go.opentelemetry.io/collector", "go.opentelemetry.io/collector/cmd/mdatagen/internal/samplescraper"),
	)
}

func TestHasDefaultValue(t *testing.T) {
	require.False(t, hasDefaultValue(&ConfigMetadata{Type: "object"}))
	require.True(t, hasDefaultValue(&ConfigMetadata{Type: "string", Default: defaultValue("value")}))
	require.True(t, hasDefaultValue(&ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"timeout": {Type: "string", GoType: "time.Duration", Default: defaultValue("30s")},
		},
	}))
	require.True(t, hasDefaultValue(&ConfigMetadata{
		Type: "object",
		AllOf: []*ConfigMetadata{
			{Type: "object", Default: defaultValue(map[string]any{"enabled": true})},
		},
	}))
	require.False(t, hasDefaultValue(&ConfigMetadata{
		Type:     "string",
		Default:  defaultValue("value"),
		GoStruct: GoStructConfig{IgnoreDefault: true},
	}))
	// External ref without any property defaults must not be treated as having defaults.
	require.False(t, hasDefaultValue(&ConfigMetadata{
		Type:         "object",
		ResolvedFrom: "go.opentelemetry.io/collector/config/confighttp.ClientConfig",
	}))
}

func TestMapCustomDefaults_NestedObjectOverrides(t *testing.T) {
	md := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"host": {Type: "string"},
			"port": {Type: "integer"},
		},
	}

	exprs := MapCustomDefaults(md, defaultValue(map[string]any{
		"host": "localhost",
		"port": float64(9090),
	}), "", "")

	require.ElementsMatch(t, []string{
		`.Host = "localhost"`,
		`.Port = 9090`,
	}, exprs)
}

func TestMapCustomDefaults_ArrayOfObjectsOverrides(t *testing.T) {
	md := &ConfigMetadata{
		Type: "array",
		Items: &ConfigMetadata{
			Type: "object",
			Properties: map[string]*ConfigMetadata{
				"url": {Type: "string"},
			},
		},
	}

	exprs := MapCustomDefaults(md, defaultValue([]any{
		map[string]any{"url": "http://example.com"},
	}), "", "")

	require.Equal(t, []string{`[0].Url = "http://example.com"`}, exprs)
}

func TestMapCustomDefaults_Panics(t *testing.T) {
	tests := []struct {
		name         string
		metadata     *ConfigMetadata
		defaultValue any
	}{
		{
			name: "missing property",
			metadata: &ConfigMetadata{
				Type:       "object",
				Properties: map[string]*ConfigMetadata{},
			},
			defaultValue: defaultValue(map[string]any{"missing": "value"}),
		},
		{
			name: "map of structs",
			metadata: &ConfigMetadata{
				Type:                 "object",
				AdditionalProperties: &ConfigMetadata{Type: "object"},
			},
			defaultValue: defaultValue(map[string]any{"entry": map[string]any{}}),
		},
		{
			name: "array without object items",
			metadata: &ConfigMetadata{
				Type:  "array",
				Items: &ConfigMetadata{Type: "string"},
			},
			defaultValue: defaultValue([]any{"value"}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Panics(t, func() {
				MapCustomDefaults(tt.metadata, tt.defaultValue, "", "")
			})
		})
	}
}

func TestMapCustomDefaults_EmptyInput(t *testing.T) {
	require.Empty(t, MapCustomDefaults(&ConfigMetadata{Type: "string"}, nil, "", ""))
}

func TestMapCustomDefaults_IgnoreDefault(t *testing.T) {
	md := &ConfigMetadata{
		Type:     "object",
		GoStruct: GoStructConfig{IgnoreDefault: true},
		Properties: map[string]*ConfigMetadata{
			"host": {Type: "string"},
		},
	}

	require.Empty(t, MapCustomDefaults(md, defaultValue(map[string]any{"host": "localhost"}), "", ""))
}

func TestNewCfgFns_DefaultHelpers(t *testing.T) {
	fns := NewCfgFns("", "")

	formatDefaultValue := fns["formatDefaultValue"].(func(*ConfigMetadata, string, any) string)
	formatBaseValue := fns["formatBaseValue"].(func(*ConfigMetadata, string, any) string)
	mapCustomDefaults := fns["mapCustomDefaults"].(func(*ConfigMetadata, any) []string)
	hasDefaultValue := fns["hasDefaultValue"].(func(*ConfigMetadata) bool)

	require.Equal(t, `"localhost"`, formatDefaultValue(&ConfigMetadata{Type: "string"}, "endpoint", defaultValue("localhost")))
	require.Equal(t, `"localhost"`, formatBaseValue(
		&ConfigMetadata{Type: "string", IsPointer: true, IsOptional: true},
		"endpoint",
		defaultValue("localhost"),
	))
	require.Equal(t, []string{`[0].Url = "http://example.com"`}, mapCustomDefaults(
		&ConfigMetadata{
			Type: "array",
			Items: &ConfigMetadata{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"url": {Type: "string"},
				},
			},
		},
		defaultValue([]any{map[string]any{"url": "http://example.com"}}),
	))
	require.True(t, hasDefaultValue(&ConfigMetadata{Type: "string", Default: defaultValue("localhost")}))
	require.False(t, hasDefaultValue(&ConfigMetadata{Type: "string"}))
}
