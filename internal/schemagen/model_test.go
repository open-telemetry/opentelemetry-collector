// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"

	"go.opentelemetry.io/collector/confmap"
)

func defaultValue(value any) any {
	return value
}

// toSchemaJSON converts a single config metadata node into its JSON Schema
// representation and marshals it with indentation.
func toSchemaJSON(t *testing.T, md *ConfigMetadata) []byte {
	t.Helper()
	schema := FromMetadata("test", "test", &ConfigsMetadata{Config: md})
	data, err := json.MarshalIndent(schema, "", "  ")
	require.NoError(t, err)
	return data
}

func TestConfigMetadata_ToJSON(t *testing.T) {
	md := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"endpoint": {Type: "string", Description: "The endpoint"},
		},
	}

	data := toSchemaJSON(t, md)
	assert.Contains(t, string(data), `"$schema"`)
	assert.Contains(t, string(data), `"endpoint"`)
	assert.Contains(t, string(data), `"The endpoint"`)
}

func TestConfigMetadata_MarshalJSON_NoSpecialFieldsUsesStructLayout(t *testing.T) {
	t.Parallel()

	metadata := &ConfigMetadata{
		Description: "Example schema",
		Type:        "object",
		Properties: map[string]*ConfigMetadata{
			"value": {Type: "string"},
		},
		Required: []string{"value"},
	}

	type alias ConfigMetadata

	expected, err := json.Marshal((*alias)(metadata))
	require.NoError(t, err)

	actual, err := json.Marshal(metadata)
	require.NoError(t, err)

	require.JSONEq(t, string(expected), string(actual))
}

func TestConfigMetadata_UnmarshalYAMLDefaultValue(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want any
	}{
		{
			name: "absent default",
			yaml: `
type: string
`,
			want: nil,
		},
		{
			name: "scalar",
			yaml: `
type: string
default: localhost
`,
			want: "localhost",
		},
		{
			name: "map",
			yaml: `
type: object
default:
  enabled: true
  label: prod
`,
			want: map[string]any{
				"enabled": true,
				"label":   "prod",
			},
		},
		{
			name: "list",
			yaml: `
type: array
default:
  - one
  - two
`,
			want: []any{"one", "two"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var md ConfigMetadata
			require.NoError(t, yaml.Unmarshal([]byte(tt.yaml), &md))

			require.Equal(t, tt.want, md.Default)
		})
	}
}

func TestConfigMetadata_ToJSONDefaultValue(t *testing.T) {
	absent := &ConfigMetadata{Type: "string"}

	jsonData := toSchemaJSON(t, absent)
	require.NotContains(t, string(jsonData), `"default"`)

	withDefault := &ConfigMetadata{Type: "string", Default: defaultValue("localhost")}

	jsonData = toSchemaJSON(t, withDefault)
	require.Contains(t, string(jsonData), `"default": "localhost"`)
}

func TestConfigMetadata_UnmarshalConfMapDefaultValue(t *testing.T) {
	parser := confmap.NewFromStringMap(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"endpoint": map[string]any{
				"type":    "string",
				"default": nil,
			},
			"headers": map[string]any{
				"type": "object",
				"default": map[string]any{
					"env": "prod",
				},
			},
		},
	})

	var md ConfigMetadata
	require.NoError(t, parser.Unmarshal(&md))

	require.Nil(t, md.Properties["endpoint"].Default)
	require.Equal(t, map[string]any{"env": "prod"}, md.Properties["headers"].Default)
}

func TestConfigMetadata_UnmarshalConfMapError(t *testing.T) {
	parser := confmap.NewFromStringMap(map[string]any{
		"type":       "object",
		"properties": "invalid",
	})

	var md ConfigMetadata
	require.Error(t, parser.Unmarshal(&md))
}

func TestConfigMetadata_Validate_Valid(t *testing.T) {
	tests := []struct {
		name string
		md   *ConfigMetadata
	}{
		{
			name: "valid with properties",
			md: &ConfigMetadata{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"endpoint": {Type: "string"},
				},
			},
		},
		{
			name: "valid with multiple properties",
			md: &ConfigMetadata{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"endpoint": {Type: "string"},
					"timeout":  {Type: "string", GoType: "time.Duration"},
					"port":     {Type: "integer"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.md.Validate()
			assert.NoError(t, err)
		})
	}
}

func TestConfigsMetadata_Validate_InvalidType(t *testing.T) {
	tests := []struct {
		name    string
		md      *ConfigsMetadata
		wantErr string
	}{
		{
			name: "type is string instead of object",
			md: &ConfigsMetadata{Config: &ConfigMetadata{
				Type: "string",
				Properties: map[string]*ConfigMetadata{
					"endpoint": {Type: "string"},
				},
			}},
			wantErr: `config type must be "object", got "string"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.md.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestConfigMetadata_Validate_EmptyConfig(t *testing.T) {
	md := &ConfigMetadata{Type: ""}
	err := md.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config must specify at least one property")
}

func TestConfigMetadata_Validate_NilMetadata(t *testing.T) {
	var md *ConfigMetadata
	// The current implementation panics on nil receiver
	// This test documents that behavior
	assert.Panics(t, func() {
		_ = md.Validate()
	}, "Validate() should panic when called on nil ConfigMetadata")
}

func TestGoStructConfig_Unmarshal(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]any
		want  GoStructConfig
	}{
		{
			name:  "custom_validator present with empty map",
			input: map[string]any{"custom_validator": map[string]any{}},
			want:  GoStructConfig{CustomValidator: &CustomValidatorConfig{}},
		},
		{
			name:  "custom_validator present with nil value",
			input: map[string]any{"custom_validator": nil},
			want:  GoStructConfig{CustomValidator: &CustomValidatorConfig{}},
		},
		{
			name:  "custom_validator absent",
			input: map[string]any{},
			want:  GoStructConfig{},
		},
		{
			name: "go_struct fields decode through mapstructure",
			input: map[string]any{
				"anonymous":      true,
				"ignore_default": true,
				"custom_validator": map[string]any{
					"name": "validateConfig",
				},
			},
			want: GoStructConfig{
				Anonymous:       true,
				IgnoreDefault:   true,
				CustomValidator: &CustomValidatorConfig{Name: "validateConfig"},
			},
		},
		{
			name:  "unrelated keys only",
			input: map[string]any{"something_else": true},
			want:  GoStructConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := confmap.NewFromStringMap(tt.input)
			var g GoStructConfig
			err := g.Unmarshal(parser)
			require.NoError(t, err)
			assert.Equal(t, tt.want, g)
		})
	}
}

func TestGoStructConfig_UnmarshalError(t *testing.T) {
	parser := confmap.NewFromStringMap(map[string]any{
		"anonymous": "not-a-bool",
	})

	var g GoStructConfig
	require.Error(t, g.Unmarshal(parser))
}

func TestConfigMetadata_Validate_EnumOnComplexType(t *testing.T) {
	tests := []struct {
		name    string
		md      *ConfigMetadata
		wantErr string
	}{
		{
			name: "enum on object",
			md: &ConfigMetadata{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"nested": {Type: "object", Enum: []any{"a"}},
				},
			},
			wantErr: `property "nested" is invalid: enum is not supported for type "object"`,
		},
		{
			name: "enum on array",
			md: &ConfigMetadata{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"items": {Type: "array", Enum: []any{"a"}},
				},
			},
			wantErr: `property "items" is invalid: enum is not supported for type "array"`,
		},
		{
			name: "enum on string is valid",
			md: &ConfigMetadata{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"level": {Type: "string", Enum: []any{"a", "b"}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.md.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigMetadata_Validate_NestedContainers(t *testing.T) {
	tests := []struct {
		name    string
		md      *ConfigMetadata
		wantErr string
	}{
		{
			name: "invalid additionalProperties",
			md: &ConfigMetadata{
				Type:                 "object",
				AdditionalProperties: &ConfigMetadata{Type: "object", Enum: []any{"a"}},
			},
			wantErr: `enum is not supported for type "object"`,
		},
		{
			name: "valid additionalProperties",
			md: &ConfigMetadata{
				Type:                 "object",
				AdditionalProperties: &ConfigMetadata{Type: "string"},
			},
		},
		{
			name: "invalid items",
			md: &ConfigMetadata{
				Type:  "array",
				Items: &ConfigMetadata{Type: "array", Enum: []any{"a"}},
			},
			wantErr: `enum is not supported for type "array"`,
		},
		{
			name: "valid items",
			md: &ConfigMetadata{
				Type:  "array",
				Items: &ConfigMetadata{Type: "string"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.md.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigsMetadata_Validate(t *testing.T) {
	tests := []struct {
		name    string
		md      *ConfigsMetadata
		wantErr string
	}{
		{
			name: "nil config and exported_configs",
			md:   &ConfigsMetadata{},
		},
		{
			name: "empty config type is allowed with properties",
			md: &ConfigsMetadata{Config: &ConfigMetadata{
				Type: "",
				Properties: map[string]*ConfigMetadata{
					"endpoint": {Type: "string"},
				},
			}},
		},
		{
			name: "config validation error propagates",
			md: &ConfigsMetadata{Config: &ConfigMetadata{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"nested": {Type: "object", Enum: []any{"a"}},
				},
			}},
			wantErr: `enum is not supported for type "object"`,
		},
		{
			name: "empty exported_configs section",
			md:   &ConfigsMetadata{ExportedConfigs: map[string]*ConfigMetadata{}},
			// a non-nil but empty map is required to be non-empty
			wantErr: "empty exported_configs section",
		},
		{
			name: "valid exported_configs",
			md: &ConfigsMetadata{ExportedConfigs: map[string]*ConfigMetadata{
				"sample": {
					Type: "object",
					Properties: map[string]*ConfigMetadata{
						"endpoint": {Type: "string"},
					},
				},
			}},
		},
		{
			name: "exported_configs validation error propagates",
			md: &ConfigsMetadata{ExportedConfigs: map[string]*ConfigMetadata{
				"sample": {
					Type: "object",
					Properties: map[string]*ConfigMetadata{
						"nested": {Type: "array", Enum: []any{"a"}},
					},
				},
			}},
			wantErr: `enum is not supported for type "array"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.md.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigMetadata_Clone(t *testing.T) {
	minProps := 1
	maximum := 10.5
	orig := &ConfigMetadata{
		Description:   "root",
		Type:          "object",
		Default:       map[string]any{"nested": []any{"a", "b"}, "flag": true},
		Enum:          []any{"x", "y"},
		Required:      []string{"endpoint"},
		MinProperties: &minProps,
		Maximum:       &maximum,
		UniqueItems:   true,
		Deprecated:    true,
		IsPointer:     true,
		IsOptional:    true,
		Embed:         true,
		InternalOnly:  true,
		GoType:        "time.Duration",
		Pattern:       "^a$",
		Format:        "duration",
		Properties: map[string]*ConfigMetadata{
			"endpoint": {Type: "string", Description: "the endpoint"},
		},
		AdditionalProperties: &ConfigMetadata{Type: "string"},
		Items:                &ConfigMetadata{Type: "integer"},
		GoStruct: GoStructConfig{
			Anonymous:       true,
			IgnoreDefault:   true,
			FieldName:       "Endpoint",
			CustomValidator: &CustomValidatorConfig{Name: "validate"},
		},
	}

	clone := orig.Clone()

	require.NotSame(t, orig, clone)
	assert.Equal(t, orig, clone)

	// Mutating the clone must not affect the original (deep copy).
	clone.Description = "changed"
	clone.Properties["endpoint"].Type = "integer"
	clone.Required[0] = "other"
	clone.Enum[0] = "z"
	*clone.MinProperties = 99
	clone.Default.(map[string]any)["flag"] = false
	clone.Default.(map[string]any)["nested"].([]any)[0] = "changed"
	clone.GoStruct.CustomValidator.Name = "other"
	clone.AdditionalProperties.Type = "changed"
	clone.Items.Type = "changed"

	assert.Equal(t, "root", orig.Description)
	assert.Equal(t, "string", orig.Properties["endpoint"].Type)
	assert.Equal(t, "endpoint", orig.Required[0])
	assert.Equal(t, "x", orig.Enum[0])
	assert.Equal(t, 1, *orig.MinProperties)
	assert.Equal(t, true, orig.Default.(map[string]any)["flag"])
	assert.Equal(t, "a", orig.Default.(map[string]any)["nested"].([]any)[0])
	assert.Equal(t, "validate", orig.GoStruct.CustomValidator.Name)
	assert.Equal(t, "string", orig.AdditionalProperties.Type)
	assert.Equal(t, "integer", orig.Items.Type)
}

func TestConfigMetadata_Clone_Nil(t *testing.T) {
	var md *ConfigMetadata
	assert.Nil(t, md.Clone())
}

func TestConfigsMetadata_Clone(t *testing.T) {
	orig := &ConfigsMetadata{
		Config: &ConfigMetadata{
			Type: "object",
			Properties: map[string]*ConfigMetadata{
				"endpoint": {Type: "string"},
			},
		},
		ExportedConfigs: map[string]*ConfigMetadata{
			"sample": {Type: "object", Properties: map[string]*ConfigMetadata{"a": {Type: "string"}}},
		},
	}

	clone := orig.Clone()
	require.NotSame(t, orig, clone)
	assert.Equal(t, orig, clone)

	clone.Config.Type = "changed"
	clone.ExportedConfigs["sample"].Type = "changed"
	assert.Equal(t, "object", orig.Config.Type)
	assert.Equal(t, "object", orig.ExportedConfigs["sample"].Type)
}

func TestConfigsMetadata_Clone_Nil(t *testing.T) {
	var md *ConfigsMetadata
	assert.Nil(t, md.Clone())
}

func TestConfigMetadata_MergeFrom(t *testing.T) {
	t.Run("nil other is a no-op", func(t *testing.T) {
		md := &ConfigMetadata{Type: "string"}
		md.MergeFrom(nil)
		assert.Equal(t, "string", md.Type)
	})

	t.Run("existing fields are preserved", func(t *testing.T) {
		md := &ConfigMetadata{
			Description: "mine",
			Type:        "string",
			Enum:        []any{"keep"},
			Required:    []string{"keep"},
			Properties: map[string]*ConfigMetadata{
				"shared": {Type: "integer"},
			},
		}
		other := &ConfigMetadata{
			Description: "theirs",
			Type:        "object",
			Enum:        []any{"drop"},
			Required:    []string{"drop"},
			Properties: map[string]*ConfigMetadata{
				"shared": {Type: "string"},
				"extra":  {Type: "boolean"},
			},
		}

		md.MergeFrom(other)

		assert.Equal(t, "mine", md.Description)
		assert.Equal(t, "string", md.Type)
		assert.Equal(t, []any{"keep"}, md.Enum)
		assert.Equal(t, []string{"keep"}, md.Required)
		// shared property is kept, missing one is merged in
		assert.Equal(t, "integer", md.Properties["shared"].Type)
		require.Contains(t, md.Properties, "extra")
		assert.Equal(t, "boolean", md.Properties["extra"].Type)
	})

	t.Run("missing fields are filled from other", func(t *testing.T) {
		md := &ConfigMetadata{}
		other := &ConfigMetadata{
			Description: "theirs",
			Deprecated:  true,
			UniqueItems: true,
			IsPointer:   true,
			IsOptional:  true,
			Embed:       true,
			GoStruct: GoStructConfig{
				Anonymous:     true,
				IgnoreDefault: true,
				FieldName:     "Field",
			},
		}

		md.MergeFrom(other)

		assert.Equal(t, "theirs", md.Description)
		assert.True(t, md.Deprecated)
		assert.True(t, md.UniqueItems)
		assert.True(t, md.IsPointer)
		assert.True(t, md.IsOptional)
		assert.True(t, md.Embed)
		assert.True(t, md.GoStruct.Anonymous)
		assert.True(t, md.GoStruct.IgnoreDefault)
		assert.Equal(t, "Field", md.GoStruct.FieldName)
	})
}
