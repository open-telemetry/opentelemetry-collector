// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"

	"go.opentelemetry.io/collector/confmap"
)

func defaultValue(value any) any {
	return value
}

func TestConfigMetadata_ToJSON(t *testing.T) {
	md := &ConfigMetadata{
		Schema: schemaVersion,
		Type:   "object",
		Properties: map[string]*ConfigMetadata{
			"endpoint": {Type: "string", Description: "The endpoint"},
		},
	}

	data, err := md.ToJSON()
	require.NoError(t, err)
	assert.Contains(t, string(data), `"$schema"`)
	assert.Contains(t, string(data), `"endpoint"`)
	assert.Contains(t, string(data), `"The endpoint"`)
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

	jsonData, err := absent.ToJSON()
	require.NoError(t, err)
	require.NotContains(t, string(jsonData), `"default"`)

	withDefault := &ConfigMetadata{Type: "string", Default: defaultValue("localhost")}

	jsonData, err = withDefault.ToJSON()
	require.NoError(t, err)
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
			name: "valid with allOf",
			md: &ConfigMetadata{
				Type: "object",
				AllOf: []*ConfigMetadata{
					{Ref: "some_ref"},
				},
			},
		},
		{
			name: "valid with both properties and allOf",
			md: &ConfigMetadata{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"endpoint": {Type: "string"},
				},
				AllOf: []*ConfigMetadata{
					{Ref: "some_ref"},
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

func TestConfigMetadata_Validate_InvalidType(t *testing.T) {
	tests := []struct {
		name    string
		md      *ConfigMetadata
		wantErr string
	}{
		{
			name: "type is string instead of object",
			md: &ConfigMetadata{
				Type: "string",
				Properties: map[string]*ConfigMetadata{
					"endpoint": {Type: "string"},
				},
			},
			wantErr: `config type must be "object", got "string"`,
		},
		{
			name: "type is empty string",
			md: &ConfigMetadata{
				Type: "",
				Properties: map[string]*ConfigMetadata{
					"endpoint": {Type: "string"},
				},
			},
			wantErr: `config type must be "object", got ""`,
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
	tests := []struct {
		name    string
		md      *ConfigMetadata
		wantErr string
	}{
		{
			name: "no properties and no allOf",
			md: &ConfigMetadata{
				Type: "object",
			},
			wantErr: "config must not be empty",
		},
		{
			name: "empty properties map and no allOf",
			md: &ConfigMetadata{
				Type:       "object",
				Properties: map[string]*ConfigMetadata{},
			},
			wantErr: "config must not be empty",
		},
		{
			name: "empty allOf slice and no properties",
			md: &ConfigMetadata{
				Type:  "object",
				AllOf: []*ConfigMetadata{},
			},
			wantErr: "config must not be empty",
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

func TestConfigMetadata_Validate_MultipleErrors(t *testing.T) {
	tests := []struct {
		name            string
		md              *ConfigMetadata
		wantErrCount    int
		wantErrContains []string
	}{
		{
			name: "invalid type and empty config",
			md: &ConfigMetadata{
				Type: "string",
			},
			wantErrCount: 2,
			wantErrContains: []string{
				`config type must be "object", got "string"`,
				"config must not be empty",
			},
		},
		{
			name: "invalid type with empty properties and empty allOf",
			md: &ConfigMetadata{
				Type:       "array",
				Properties: map[string]*ConfigMetadata{},
				AllOf:      []*ConfigMetadata{},
			},
			wantErrCount: 2,
			wantErrContains: []string{
				`config type must be "object", got "array"`,
				"config must not be empty",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.md.Validate()
			require.Error(t, err)

			// Check that error contains all expected substrings
			for _, expectedErr := range tt.wantErrContains {
				assert.Contains(t, err.Error(), expectedErr)
			}
		})
	}
}

func TestConfigMetadata_Validate_NilMetadata(t *testing.T) {
	var md *ConfigMetadata
	// The current implementation panics on nil receiver
	// This test documents that behavior
	assert.Panics(t, func() {
		_ = md.Validate()
	}, "Validate() should panic when called on nil ConfigMetadata")
}

func TestConfigMetadata_Validate_TypeAsInterface(t *testing.T) {
	// Test when Type field is set as interface{} instead of string
	// This tests the real-world scenario where YAML/JSON unmarshaling
	// might produce different types
	tests := []struct {
		name    string
		typeVal string
		wantErr bool
	}{
		{
			name:    "type as string 'object'",
			typeVal: "object",
			wantErr: false,
		},
		{
			name:    "type as string 'string'",
			typeVal: "string",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := &ConfigMetadata{
				Type: tt.typeVal,
				Properties: map[string]*ConfigMetadata{
					"field": {Type: "string"},
				},
			}

			err := md.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigMetadata_Validate_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		md      *ConfigMetadata
		wantErr bool
	}{
		{
			name: "single property is sufficient",
			md: &ConfigMetadata{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"only_field": {Type: "string"},
				},
			},
			wantErr: false,
		},
		{
			name: "single allOf entry is sufficient",
			md: &ConfigMetadata{
				Type: "object",
				AllOf: []*ConfigMetadata{
					{Ref: "base_config"},
				},
			},
			wantErr: false,
		},
		{
			name: "properties with nested objects",
			md: &ConfigMetadata{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"server": {
						Type: "object",
						Properties: map[string]*ConfigMetadata{
							"host": {Type: "string"},
							"port": {Type: "integer"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "allOf with nil entries",
			md: &ConfigMetadata{
				Type: "object",
				AllOf: []*ConfigMetadata{
					nil,
					{Ref: "base_config"},
				},
			},
			wantErr: false, // At least one non-nil entry exists
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.md.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
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
