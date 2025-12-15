// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	generatedData, err := os.ReadFile(schemaPath)
	require.NoError(t, err)

	// Read the expected schema from testdata
	expectedPath := filepath.Join("testdata", "config_schema.json")
	expectedData, err := os.ReadFile(expectedPath)
	require.NoError(t, err)

	// Parse both schemas to compare them (to ignore formatting differences)
	var generatedSchema, expectedSchema Schema
	err = json.Unmarshal(generatedData, &generatedSchema)
	require.NoError(t, err)
	err = json.Unmarshal(expectedData, &expectedSchema)
	require.NoError(t, err)

	// Re-marshal both schemas with consistent formatting for comparison
	generatedJSON, err := json.MarshalIndent(generatedSchema, "", "  ")
	require.NoError(t, err)
	expectedJSON, err := json.MarshalIndent(expectedSchema, "", "  ")
	require.NoError(t, err)

	assert.Equal(t, string(expectedJSON), string(generatedJSON))
}

func TestSetSchemaType(t *testing.T) {
	g := &SchemaGenerator{}

	tests := []struct {
		goType       string
		expectedType string
		format       string
	}{
		{"string", "string", ""},
		{"bool", "boolean", ""},
		{"int", "integer", ""},
		{"int64", "integer", ""},
		{"float64", "number", ""},
		{"time.Duration", "string", "duration"},
		{"[]string", "array", ""},
		{"map[string]string", "object", ""},
		{"go.opentelemetry.io/collector/config/configopaque.String", "string", ""},
		{"go.opentelemetry.io/collector/config/configoptional.Optional[string]", "string", ""},
		{"go.opentelemetry.io/collector/config/configoptional.Optional[int]", "integer", ""},
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
		})
	}
}
