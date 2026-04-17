// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDocType(t *testing.T) {
	tests := []struct {
		name     string
		md       *ConfigMetadata
		expected string
	}{
		{name: "nil", md: nil, expected: "any"},
		{name: "string", md: &ConfigMetadata{Type: "string"}, expected: "string"},
		{name: "string with enum", md: &ConfigMetadata{Type: "string", Enum: []any{"a", "b"}}, expected: "string (one of: a, b)"},
		{name: "duration", md: &ConfigMetadata{Type: "string", Format: "duration"}, expected: "duration"},
		{name: "datetime", md: &ConfigMetadata{Type: "string", Format: "date-time"}, expected: "datetime"},
		{name: "integer", md: &ConfigMetadata{Type: "integer"}, expected: "int"},
		{name: "number", md: &ConfigMetadata{Type: "number"}, expected: "float"},
		{name: "boolean", md: &ConfigMetadata{Type: "boolean"}, expected: "bool"},
		{name: "array no items", md: &ConfigMetadata{Type: "array"}, expected: "[]any"},
		{name: "array string items", md: &ConfigMetadata{Type: "array", Items: &ConfigMetadata{Type: "string"}}, expected: "[]string"},
		{name: "object plain", md: &ConfigMetadata{Type: "object"}, expected: "object"},
		{name: "object with additionalProperties", md: &ConfigMetadata{Type: "object", AdditionalProperties: &ConfigMetadata{Type: "string"}}, expected: "map[string]string"},
		{name: "unknown type with ref", md: &ConfigMetadata{Ref: "somewhere"}, expected: "object"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, DocType(tt.md))
		})
	}
}

func TestDocDefault(t *testing.T) {
	tests := []struct {
		name     string
		md       *ConfigMetadata
		expected string
	}{
		{name: "nil", md: nil, expected: ""},
		{name: "no default", md: &ConfigMetadata{Type: "string"}, expected: ""},
		{name: "string default", md: &ConfigMetadata{Type: "string", Default: "localhost:4317"}, expected: "localhost:4317"},
		{name: "bool default true", md: &ConfigMetadata{Type: "boolean", Default: true}, expected: "true"},
		{name: "int default", md: &ConfigMetadata{Type: "integer", Default: 42}, expected: "42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, DocDefault(tt.md))
		})
	}
}

func TestExtractPropDocs(t *testing.T) {
	cfg := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"endpoint": {
				Type:        "string",
				Description: "The endpoint to connect to.",
				Default:     "localhost:4317",
			},
			"timeout": {
				Type:        "string",
				Format:      "duration",
				Description: "Timeout for requests.",
				Default:     "10s",
			},
			"retries": {
				Type:        "integer",
				Description: "Number of retries.",
				Deprecated:  true,
			},
		},
		Required: []string{"endpoint"},
	}

	docs := ExtractPropDocs(cfg)
	require.Len(t, docs, 3)

	// Results are sorted by name.
	require.Equal(t, "endpoint", docs[0].Name)
	require.Equal(t, "string", docs[0].Type)
	require.Equal(t, "localhost:4317", docs[0].Default)
	require.True(t, docs[0].Required)
	require.False(t, docs[0].Deprecated)

	require.Equal(t, "retries", docs[1].Name)
	require.Equal(t, "int", docs[1].Type)
	require.False(t, docs[1].Required)
	require.True(t, docs[1].Deprecated)

	require.Equal(t, "timeout", docs[2].Name)
	require.Equal(t, "duration", docs[2].Type)
	require.Equal(t, "10s", docs[2].Default)
	require.False(t, docs[2].Required)
}

func TestExtractPropDocs_Nil(t *testing.T) {
	require.Nil(t, ExtractPropDocs(nil))
}

func TestExtractPropDocs_Empty(t *testing.T) {
	docs := ExtractPropDocs(&ConfigMetadata{Type: "object"})
	require.Empty(t, docs)
}
