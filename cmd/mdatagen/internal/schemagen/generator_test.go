// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestSchemaGenerator_GenerateSchema(t *testing.T) {
	testDir := filepath.Join("..", "samplereceiver")
	outputDir := t.TempDir()

	analyzer := NewPackageAnalyzer(testDir)
	generator := NewSchemaGenerator(outputDir, analyzer)

	// Pass empty string to test auto-detection of config type
	err := generator.GenerateSchema("receiver", "sample", "")
	require.NoError(t, err)

	// Check that schema file was created
	schemaPath := filepath.Join(outputDir, "config_schema.json")
	_, err = os.Stat(schemaPath)
	require.NoError(t, err, "schema file was not created")

	// Read the generated schema
	generatedData, err := os.ReadFile(schemaPath) //#nosec G304 -- test file path
	require.NoError(t, err)

	// Read the expected schema from testdata
	expectedPath := filepath.Join("testdata", "config_schema.json")
	expectedData, err := os.ReadFile(expectedPath) //#nosec G304 -- test file path
	require.NoError(t, err)

	// Compare JSON content (ignoring formatting differences)
	assert.JSONEq(t, string(expectedData), string(generatedData))
}

func TestParseTag(t *testing.T) {
	tests := []struct {
		name           string
		tag            string
		expectedName   string
		expectedSquash bool
	}{
		{
			name:           "simple name",
			tag:            `mapstructure:"endpoint"`,
			expectedName:   "endpoint",
			expectedSquash: false,
		},
		{
			name:           "skip field with dash",
			tag:            `mapstructure:"-"`,
			expectedName:   "-",
			expectedSquash: false,
		},
		{
			name:           "squash tag",
			tag:            `mapstructure:",squash"`,
			expectedName:   "",
			expectedSquash: true,
		},
		{
			name:           "name with squash",
			tag:            `mapstructure:"config,squash"`,
			expectedName:   "config",
			expectedSquash: true,
		},
		{
			name:           "empty mapstructure",
			tag:            `json:"foo"`,
			expectedName:   "",
			expectedSquash: false,
		},
		{
			name:           "omitempty option",
			tag:            `mapstructure:"field,omitempty"`,
			expectedName:   "field",
			expectedSquash: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			name, squash := parseTag(tc.tag)
			assert.Equal(t, tc.expectedName, name, "unexpected name")
			assert.Equal(t, tc.expectedSquash, squash, "unexpected squash")
		})
	}
}

func TestSetSchemaType(t *testing.T) {
	g := &SchemaGenerator{}

	tests := []struct {
		goType       string
		expectedType string
		format       string
		pattern      string
	}{
		{"string", "string", "", ""},
		{"bool", "boolean", "", ""},
		{"int", "integer", "", ""},
		{"int64", "integer", "", ""},
		{"float64", "number", "", ""},
		{"time.Duration", "string", "", `^(0|[-+]?((\d+(\.\d*)?|\.\d+)(ns|us|µs|μs|ms|s|m|h))+)$`},
		{"[]string", "array", "", ""},
		{"map[string]string", "object", "", ""},
		{"go.opentelemetry.io/collector/config/configopaque.String", "string", "", ""},
		{"go.opentelemetry.io/collector/config/configoptional.Optional[string]", "string", "", ""},
		{"go.opentelemetry.io/collector/config/configoptional.Optional[int]", "integer", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.goType, func(t *testing.T) {
			schema := &Schema{}
			g.setSchemaType(schema, tc.goType)
			if schema.Type != tc.expectedType {
				t.Errorf("expected type %q, got %q", tc.expectedType, schema.Type)
			}
			if tc.format != "" && schema.Format != tc.format {
				t.Errorf("expected format %q, got %q", tc.format, schema.Format)
			}
			if tc.pattern != "" && schema.Pattern != tc.pattern {
				t.Errorf("expected pattern %q, got %q", tc.pattern, schema.Pattern)
			}
		})
	}
}

func TestSchemaValidation(t *testing.T) {
	// Load the JSON schema
	schemaPath := filepath.Join("testdata", "config_schema.json")
	schemaData, err := os.ReadFile(schemaPath)
	require.NoError(t, err, "failed to read schema file")

	// Parse the schema JSON
	var schemaDoc any
	err = json.Unmarshal(schemaData, &schemaDoc)
	require.NoError(t, err, "failed to parse schema JSON")

	// Compile the schema
	compiler := jsonschema.NewCompiler()
	err = compiler.AddResource("config_schema.json", schemaDoc)
	require.NoError(t, err, "failed to add schema resource")

	schema, err := compiler.Compile("config_schema.json")
	require.NoError(t, err, "failed to compile schema")

	tests := []struct {
		name        string
		configFile  string
		expectValid bool
	}{
		{
			name:        "valid configuration",
			configFile:  "samplereceiver_config.yaml",
			expectValid: true,
		},
		{
			name:        "invalid configuration with type mismatches",
			configFile:  "samplereceiver_invalid_config.yaml",
			expectValid: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Load the YAML config
			configPath := filepath.Join("testdata", tc.configFile)
			configData, err := os.ReadFile(configPath)
			require.NoError(t, err, "failed to read config file")

			// Parse YAML to interface{}
			var config any
			err = yaml.Unmarshal(configData, &config)
			require.NoError(t, err, "failed to parse YAML")

			// Convert to JSON-compatible format via round-trip
			jsonBytes, err := json.Marshal(config)
			require.NoError(t, err, "failed to marshal config to JSON")
			err = json.Unmarshal(jsonBytes, &config)
			require.NoError(t, err, "failed to unmarshal JSON")

			// Validate against schema
			validationErr := schema.Validate(config)

			if tc.expectValid {
				assert.NoError(t, validationErr, "expected config to be valid")
			} else {
				assert.Error(t, validationErr, "expected config to be invalid")
				t.Logf("Validation errors (expected): %v", validationErr)
			}
		})
	}
}
