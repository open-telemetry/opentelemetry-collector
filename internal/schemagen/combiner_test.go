// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen

import (
	"encoding/json"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/require"
)

func TestCombineCollectorSchema_LayoutAndValidation(t *testing.T) {
	t.Parallel()

	schema, err := CombineCollectorSchema(CollectorSchemaParts{
		Receivers: []CollectorComponentSchema{
			{
				Type: "otlp",
				Schema: &ConfigMetadata{
					Type: "object",
					Properties: map[string]*ConfigMetadata{
						"endpoint": {Type: "string"},
					},
					Required: []string{"endpoint"},
				},
			},
		},
		Processors: []CollectorComponentSchema{
			{
				Type: "batch",
				Schema: &ConfigMetadata{
					Type: "object",
					Properties: map[string]*ConfigMetadata{
						"timeout": {Type: "string"},
					},
				},
			},
		},
		Service: &ConfigMetadata{
			Type: "object",
			Properties: map[string]*ConfigMetadata{
				"pipelines": {Type: "object"},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, schemaVersion, schema.Schema)
	require.Contains(t, schema.Properties, "receivers")
	require.Contains(t, schema.Properties, "processors")
	require.Contains(t, schema.Properties, "exporters")
	require.Contains(t, schema.Properties, "connectors")
	require.Contains(t, schema.Properties, "extensions")
	require.Contains(t, schema.Properties, "service")

	compiled := compileSchema(t, schema)

	require.NoError(t, compiled.Validate(unmarshalJSON(t, `{
		"receivers": {
			"otlp": {"endpoint": "localhost:4317"},
			"otlp/secondary": {"endpoint": "localhost:4318"}
		},
		"processors": {
			"batch": {"timeout": "5s"}
		},
		"service": {
			"pipelines": {}
		}
	}`)))

	err = compiled.Validate(unmarshalJSON(t, `{
		"receivers": {
			"otlp": {}
		}
	}`))
	require.Error(t, err)

	err = compiled.Validate(unmarshalJSON(t, `{
		"receivers": {
			"prometheus": {}
		}
	}`))
	require.Error(t, err)
}

func TestCombineCollectorSchema_DeprecatedTypeAndMissingSchema(t *testing.T) {
	t.Parallel()

	schema, err := CombineCollectorSchema(CollectorSchemaParts{
		Exporters: []CollectorComponentSchema{
			{
				Type:           "otlp_http",
				DeprecatedType: "otlphttp",
				Schema: &ConfigMetadata{
					Type: "object",
					Properties: map[string]*ConfigMetadata{
						"endpoint": {Type: "string"},
					},
					Required: []string{"endpoint"},
				},
			},
			{
				Type: "debug",
			},
		},
	})
	require.NoError(t, err)

	exporters := schema.Properties[string(CollectorSectionExporters)]
	require.NotNil(t, exporters)
	require.Len(t, exporters.PatternProperties, 3)
	require.True(t, exporters.PatternProperties[collectorComponentPattern("otlphttp")].Deprecated)
	require.Empty(t, exporters.PatternProperties[collectorComponentPattern("debug")].Type)

	compiled := compileSchema(t, schema)

	require.NoError(t, compiled.Validate(unmarshalJSON(t, `{
		"exporters": {
			"otlp_http": {"endpoint": "https://example.test"},
			"otlphttp/legacy": {"endpoint": "https://example.test"},
			"debug/custom": {"verbosity": "detailed"}
		}
	}`)))

	err = compiled.Validate(unmarshalJSON(t, `{
		"exporters": {
			"nope": {}
		}
	}`))
	require.Error(t, err)
}

func TestCombineCollectorSchema_DuplicateIdentifiers(t *testing.T) {
	t.Parallel()

	_, err := CombineCollectorSchema(CollectorSchemaParts{
		Receivers: []CollectorComponentSchema{
			{Type: "otlp"},
			{Type: "otlp"},
		},
	})
	require.ErrorContains(t, err, `duplicate component identifier "otlp"`)

	_, err = CombineCollectorSchema(CollectorSchemaParts{
		Exporters: []CollectorComponentSchema{
			{Type: "otlp_http", DeprecatedType: "otlphttp"},
			{Type: "otlphttp"},
		},
	})
	require.ErrorContains(t, err, `duplicate component identifier "otlphttp"`)
}

func TestCombineCollectorSchema_EmptyComponentType(t *testing.T) {
	t.Parallel()

	_, err := CombineCollectorSchema(CollectorSchemaParts{
		Connectors: []CollectorComponentSchema{
			{Type: ""},
		},
	})
	require.ErrorContains(t, err, "component type must not be empty")
}

func TestCombineCollectorSchema_DeprecatedTypeDuplicatesMainType(t *testing.T) {
	t.Parallel()

	_, err := CombineCollectorSchema(CollectorSchemaParts{
		Extensions: []CollectorComponentSchema{
			{Type: "health_check"},
			{Type: "healthcheck", DeprecatedType: "health_check"},
		},
	})
	require.ErrorContains(t, err, `duplicate component identifier "health_check"`)
}

func compileSchema(t *testing.T, schema *ConfigMetadata) *jsonschema.Schema {
	t.Helper()

	data, err := schema.ToJSON()
	require.NoError(t, err)

	compiler := jsonschema.NewCompiler()
	var doc any
	require.NoError(t, json.Unmarshal(data, &doc))
	require.NoError(t, compiler.AddResource("schema.json", doc))

	compiled, err := compiler.Compile("schema.json")
	require.NoError(t, err)

	return compiled
}

func unmarshalJSON(t *testing.T, data string) any {
	t.Helper()

	var value any
	require.NoError(t, json.Unmarshal([]byte(data), &value))

	return value
}

func TestConfigMetadata_MarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		metadata *ConfigMetadata
		expected map[string]any
	}{
		{
			name: "with pattern properties",
			metadata: &ConfigMetadata{
				Type: "object",
				PatternProperties: map[string]*ConfigMetadata{
					"^test": {Type: "string"},
				},
			},
			expected: map[string]any{
				"type": "object",
				"patternProperties": map[string]any{
					"^test": map[string]any{"type": "string"},
				},
			},
		},
		{
			name: "with additional properties allowed false",
			metadata: &ConfigMetadata{
				Type:                        "object",
				AdditionalPropertiesAllowed: boolPtr(false),
			},
			expected: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
			},
		},
		{
			name: "with additional properties allowed true",
			metadata: &ConfigMetadata{
				Type:                        "object",
				AdditionalPropertiesAllowed: boolPtr(true),
			},
			expected: map[string]any{
				"type":                 "object",
				"additionalProperties": true,
			},
		},
		{
			name: "with both pattern properties and additional properties",
			metadata: &ConfigMetadata{
				Type: "object",
				PatternProperties: map[string]*ConfigMetadata{
					"^foo": {Type: "number"},
				},
				AdditionalPropertiesAllowed: boolPtr(false),
			},
			expected: map[string]any{
				"type": "object",
				"patternProperties": map[string]any{
					"^foo": map[string]any{"type": "number"},
				},
				"additionalProperties": false,
			},
		},
		{
			name: "without pattern properties or additional properties",
			metadata: &ConfigMetadata{
				Type: "string",
			},
			expected: map[string]any{
				"type": "string",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.metadata)
			require.NoError(t, err)

			var actual map[string]any
			require.NoError(t, json.Unmarshal(data, &actual))

			for key, expectedValue := range tt.expected {
				actualValue, ok := actual[key]
				require.True(t, ok, "expected key %q not found in output", key)
				require.Equal(t, expectedValue, actualValue, "value mismatch for key %q", key)
			}
		})
	}
}
