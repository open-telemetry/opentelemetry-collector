// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			wantErr: `config type must be "object":, got "string"`,
		},
		{
			name: "type is empty string",
			md: &ConfigMetadata{
				Type: "",
				Properties: map[string]*ConfigMetadata{
					"endpoint": {Type: "string"},
				},
			},
			wantErr: `config type must be "object":, got ""`,
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
				`config type must be "object":, got "string"`,
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
				`config type must be "object":, got "array"`,
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
		typeVal any
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
		{
			name:    "type as array of strings (union type) - not supported",
			typeVal: []any{"object", "null"},
			wantErr: true, // Current implementation doesn't handle union types, treats as invalid
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
