// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfgen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolver_ResolveSchema_BasicMetadata(t *testing.T) {
	resolver := &Resolver{
		pkgID:   "go.opentelemetry.io/collector/receiver/otlpreceiver",
		version: "v1.0.0",
		class:   "receiver",
		name:    "otlp",
		loader:  NewLoader(""),
	}

	src := &ConfigMetadata{
		Description: "OTLP receiver configuration",
		Type:        "object",
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	assert.Equal(t, schemaVersion, result.Schema)
	assert.Equal(t, "go.opentelemetry.io/collector/v1.0.0/receiver/otlpreceiver", result.ID)
	assert.Equal(t, "receiver/otlp", result.Title)
	assert.Equal(t, "OTLP receiver configuration", result.Description)
	assert.Equal(t, "object", result.Type)
}

func TestResolver_ResolveSchema_InternalReference(t *testing.T) {
	resolver := &Resolver{
		pkgID:   "go.opentelemetry.io/collector/test/component",
		version: "v1.0.0",
		class:   "receiver",
		name:    "test",
		loader:  NewLoader(""),
	}

	src := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"config": {
				Ref: "target_type",
			},
		},
		Defs: map[string]*ConfigMetadata{
			"target_type": {
				Type:        "string",
				Description: "Target type description",
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	assert.Equal(t, "object", result.Type)
	assert.NotNil(t, result.Properties["config"])
	assert.Equal(t, "string", result.Properties["config"].Type)
	assert.Equal(t, "Target type description", result.Properties["config"].Description)
}

func TestResolver_ResolveSchema_UnknownInternalReference(t *testing.T) {
	resolver := &Resolver{
		pkgID:   "go.opentelemetry.io/collector/test/component",
		version: "v1.0.0",
		class:   "receiver",
		name:    "test",
		loader:  NewLoader(""),
	}

	src := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"config": {
				Ref: "unknown_type",
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	// Should fallback to "any" type
	assert.Equal(t, "unknown_type", result.Properties["config"].GoType)
	assert.Contains(t, result.Properties["config"].Comment, "Empty or unknown reference")
}

func TestResolver_ResolveSchema_NestedStructures(t *testing.T) {
	resolver := &Resolver{
		pkgID:   "go.opentelemetry.io/collector/test/component",
		version: "v1.0.0",
		class:   "receiver",
		name:    "test",
		loader:  NewLoader(""),
	}

	src := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"nested": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"field1": {
						Type: "string",
					},
					"field2": {
						Type: "integer",
					},
				},
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	assert.Equal(t, "object", result.Type)
	assert.NotNil(t, result.Properties["nested"])
	assert.Equal(t, "object", result.Properties["nested"].Type)
	assert.NotNil(t, result.Properties["nested"].Properties["field1"])
	assert.Equal(t, "string", result.Properties["nested"].Properties["field1"].Type)
	assert.NotNil(t, result.Properties["nested"].Properties["field2"])
	assert.Equal(t, "integer", result.Properties["nested"].Properties["field2"].Type)
}

func TestResolver_ResolveSchema_AllOf(t *testing.T) {
	resolver := &Resolver{
		pkgID:   "go.opentelemetry.io/collector/test/component",
		version: "v1.0.0",
		class:   "receiver",
		name:    "test",
		loader:  NewLoader(""),
	}

	src := &ConfigMetadata{
		Type: "object",
		AllOf: []*ConfigMetadata{
			{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"field1": {Type: "string"},
				},
			},
			{
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"field2": {Type: "integer"},
				},
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	assert.Len(t, result.AllOf, 2)
	assert.NotNil(t, result.AllOf[0].Properties["field1"])
	assert.NotNil(t, result.AllOf[1].Properties["field2"])
}

func TestResolver_ResolveSchema_ArrayItems(t *testing.T) {
	resolver := &Resolver{
		pkgID:   "go.opentelemetry.io/collector/test/component",
		version: "v1.0.0",
		class:   "receiver",
		name:    "test",
		loader:  NewLoader(""),
	}

	src := &ConfigMetadata{
		Type: "array",
		Items: &ConfigMetadata{
			Type: "object",
			Properties: map[string]*ConfigMetadata{
				"name": {Type: "string"},
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	assert.Equal(t, "array", result.Type)
	assert.NotNil(t, result.Items)
	assert.Equal(t, "object", result.Items.Type)
	assert.NotNil(t, result.Items.Properties["name"])
}

func TestResolver_LoadExternalRef_Success(t *testing.T) {
	tempDir := t.TempDir()

	// Create external schema
	schemaPath := filepath.Join(tempDir, "main", "go.opentelemetry.io/collector/scraper/scraperhelper")
	require.NoError(t, os.MkdirAll(schemaPath, 0o755))
	schemaFile := filepath.Join(schemaPath, schemaFileName)
	schemaContent := `type: object
$defs:
  controller_config:
    type: object
    properties:
      timeout:
        type: string
`
	require.NoError(t, os.WriteFile(schemaFile, []byte(schemaContent), 0o644))

	resolver := &Resolver{
		pkgID:   "go.opentelemetry.io/collector/test/component",
		version: "main",
		class:   "receiver",
		name:    "test",
		loader:  NewLoader(tempDir),
	}

	result, err := resolver.loadExternalRef("go.opentelemetry.io/collector/scraper/scraperhelper.controller_config")
	require.NoError(t, err)
	assert.Equal(t, "object", result.Type)
	assert.NotNil(t, result.Properties["timeout"])
	assert.Equal(t, "string", result.Properties["timeout"].Type)
}

func TestResolver_LoadExternalRef_InvalidPath(t *testing.T) {
	resolver := &Resolver{
		pkgID:   "go.opentelemetry.io/collector/test/component",
		version: "main",
		class:   "receiver",
		name:    "test",
		loader:  NewLoader(""),
	}

	result, err := resolver.loadExternalRef("invalid/path/without/namespace")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid reference path")
	assert.Nil(t, result)
}

func TestResolver_LoadExternalRef_TypeNotFound(t *testing.T) {
	tempDir := t.TempDir()

	// Create external schema without the requested type
	schemaPath := filepath.Join(tempDir, "main", "go.opentelemetry.io/collector/scraper/scraperhelper")
	require.NoError(t, os.MkdirAll(schemaPath, 0o755))
	schemaFile := filepath.Join(schemaPath, schemaFileName)
	schemaContent := `type: object
$defs:
  other_type:
    type: string
`
	require.NoError(t, os.WriteFile(schemaFile, []byte(schemaContent), 0o644))

	resolver := &Resolver{
		pkgID:   "go.opentelemetry.io/collector/test/component",
		version: "main",
		class:   "receiver",
		name:    "test",
		loader:  NewLoader(tempDir),
	}

	result, err := resolver.loadExternalRef("go.opentelemetry.io/collector/scraper/scraperhelper.controller_config")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "type \"controller_config\" not found")
	assert.Nil(t, result)
}

func TestResolver_IsExternalRef(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		expected bool
	}{
		{
			name:     "collector external ref",
			ref:      "go.opentelemetry.io/collector/scraper/scraperhelper.controller_config",
			expected: true,
		},
		{
			name:     "contrib external ref",
			ref:      "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/mysqlreceiver.config",
			expected: true,
		},
		{
			name:     "relative path",
			ref:      "./internal/metadata.metrics_builder_config",
			expected: true,
		},
		{
			name:     "internal ref - simple name",
			ref:      "target_type",
			expected: false,
		},
		{
			name:     "internal ref - json pointer",
			ref:      "#/$defs/target_type",
			expected: false,
		},
		{
			name:     "unsupported namespace",
			ref:      "github.com/example/custom.config",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isExternalRef(tt.ref)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolver_ResolveSchema_ExternalReference_Integration(t *testing.T) {
	tempDir := t.TempDir()

	// Create external schema
	schemaPath := filepath.Join(tempDir, "v1.0.0", "go.opentelemetry.io/collector/config/confighttp")
	require.NoError(t, os.MkdirAll(schemaPath, 0o755))
	schemaFile := filepath.Join(schemaPath, schemaFileName)
	schemaContent := `type: object
$defs:
  client_config:
    type: object
    properties:
      endpoint:
        type: string
        description: "HTTP endpoint"
      timeout:
        type: string
        description: "Request timeout"
`
	require.NoError(t, os.WriteFile(schemaFile, []byte(schemaContent), 0o644))

	resolver := &Resolver{
		pkgID:   "go.opentelemetry.io/collector/receiver/otlpreceiver",
		version: "v1.0.0",
		class:   "receiver",
		name:    "otlp",
		loader:  NewLoader(tempDir),
	}

	src := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"http": {
				Ref: "go.opentelemetry.io/collector/config/confighttp.client_config",
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	assert.Equal(t, "object", result.Type)
	assert.NotNil(t, result.Properties["http"])
	assert.Equal(t, "object", result.Properties["http"].Type)
	assert.NotNil(t, result.Properties["http"].Properties["endpoint"])
	assert.Equal(t, "HTTP endpoint", result.Properties["http"].Properties["endpoint"].Description)
	assert.NotNil(t, result.Properties["http"].Properties["timeout"])
	assert.Equal(t, "Request timeout", result.Properties["http"].Properties["timeout"].Description)
}

func TestResolver_ResolveSchema_DurationFormat(t *testing.T) {
	resolver := &Resolver{
		pkgID:   "go.opentelemetry.io/collector/test/component",
		version: "v1.0.0",
		class:   "receiver",
		name:    "test",
		loader:  NewLoader(""),
	}

	src := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"timeout": {
				Type:        "string",
				Format:      "duration",
				Description: "Request timeout",
			},
			"interval": {
				Type:   "string",
				Format: "duration",
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)

	// Check timeout field - should have pattern instead of format
	assert.NotNil(t, result.Properties["timeout"])
	assert.Equal(t, "string", result.Properties["timeout"].Type)
	assert.Empty(t, result.Properties["timeout"].Format, "format should be cleared")
	assert.Equal(t, `^([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$`, result.Properties["timeout"].Pattern)
	assert.Contains(t, result.Properties["timeout"].Description, "duration format")
	assert.Contains(t, result.Properties["timeout"].Description, "Request timeout")

	// Check interval field - should have pattern and auto-generated description hint
	assert.NotNil(t, result.Properties["interval"])
	assert.Equal(t, "string", result.Properties["interval"].Type)
	assert.Empty(t, result.Properties["interval"].Format)
	assert.Equal(t, `^([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$`, result.Properties["interval"].Pattern)
}
