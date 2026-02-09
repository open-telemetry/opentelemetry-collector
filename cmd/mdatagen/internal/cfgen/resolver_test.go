// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfgen

import (
	"os"
	"path/filepath"
	"testing"

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
	require.Equal(t, schemaVersion, result.Schema)
	require.Equal(t, "go.opentelemetry.io/collector/v1.0.0/receiver/otlpreceiver", result.ID)
	require.Equal(t, "receiver/otlp", result.Title)
	require.Equal(t, "OTLP receiver configuration", result.Description)
	require.Equal(t, "object", result.Type)
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
	require.Equal(t, "object", result.Type)
	require.NotNil(t, result.Properties["config"])
	require.Equal(t, "string", result.Properties["config"].Type)
	require.Equal(t, "Target type description", result.Properties["config"].Description)
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
	require.Equal(t, "unknown_type", result.Properties["config"].GoType)
	require.Contains(t, result.Properties["config"].Comment, "Empty or unknown reference")
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
	require.Equal(t, "object", result.Type)
	require.NotNil(t, result.Properties["nested"])
	require.Equal(t, "object", result.Properties["nested"].Type)
	require.NotNil(t, result.Properties["nested"].Properties["field1"])
	require.Equal(t, "string", result.Properties["nested"].Properties["field1"].Type)
	require.NotNil(t, result.Properties["nested"].Properties["field2"])
	require.Equal(t, "integer", result.Properties["nested"].Properties["field2"].Type)
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
	require.Len(t, result.AllOf, 2)
	require.NotNil(t, result.AllOf[0].Properties["field1"])
	require.NotNil(t, result.AllOf[1].Properties["field2"])
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
	require.Equal(t, "array", result.Type)
	require.NotNil(t, result.Items)
	require.Equal(t, "object", result.Items.Type)
	require.NotNil(t, result.Items.Properties["name"])
}

func TestResolver_LoadExternalRef_Success(t *testing.T) {
	tempDir := t.TempDir()

	// Create external schema
	schemaPath := filepath.Join(tempDir, "main", "go.opentelemetry.io/collector/scraper/scraperhelper")
	require.NoError(t, os.MkdirAll(schemaPath, 0o750))
	schemaFile := filepath.Join(schemaPath, schemaFileName)
	schemaContent := `type: object
$defs:
  controller_config:
    type: object
    properties:
      timeout:
        type: string
`
	require.NoError(t, os.WriteFile(schemaFile, []byte(schemaContent), 0o600))

	resolver := &Resolver{
		pkgID:   "go.opentelemetry.io/collector/test/component",
		version: "main",
		class:   "receiver",
		name:    "test",
		loader:  NewLoader(tempDir),
	}

	result, err := resolver.loadExternalRef("go.opentelemetry.io/collector/scraper/scraperhelper.controller_config")
	require.NoError(t, err)
	require.Equal(t, "object", result.Type)
	require.NotNil(t, result.Properties["timeout"])
	require.Equal(t, "string", result.Properties["timeout"].Type)
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
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid reference path")
	require.Nil(t, result)
}

func TestResolver_LoadExternalRef_TypeNotFound(t *testing.T) {
	tempDir := t.TempDir()

	// Create external schema without the requested type
	schemaPath := filepath.Join(tempDir, "main", "go.opentelemetry.io/collector/scraper/scraperhelper")
	require.NoError(t, os.MkdirAll(schemaPath, 0o750))
	schemaFile := filepath.Join(schemaPath, schemaFileName)
	schemaContent := `type: object
$defs:
  other_type:
    type: string
`
	require.NoError(t, os.WriteFile(schemaFile, []byte(schemaContent), 0o600))

	resolver := &Resolver{
		pkgID:   "go.opentelemetry.io/collector/test/component",
		version: "main",
		class:   "receiver",
		name:    "test",
		loader:  NewLoader(tempDir),
	}

	result, err := resolver.loadExternalRef("go.opentelemetry.io/collector/scraper/scraperhelper.controller_config")
	require.Error(t, err)
	require.Contains(t, err.Error(), "type \"controller_config\" not found")
	require.Nil(t, result)
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
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestResolver_ResolveSchema_ExternalReference_Integration(t *testing.T) {
	tempDir := t.TempDir()

	// Create external schema
	schemaPath := filepath.Join(tempDir, "v1.0.0", "go.opentelemetry.io/collector/config/confighttp")
	require.NoError(t, os.MkdirAll(schemaPath, 0o750))
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
	require.NoError(t, os.WriteFile(schemaFile, []byte(schemaContent), 0o600))

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
	require.Equal(t, "object", result.Type)
	require.NotNil(t, result.Properties["http"])
	require.Equal(t, "object", result.Properties["http"].Type)
	require.NotNil(t, result.Properties["http"].Properties["endpoint"])
	require.Equal(t, "HTTP endpoint", result.Properties["http"].Properties["endpoint"].Description)
	require.NotNil(t, result.Properties["http"].Properties["timeout"])
	require.Equal(t, "Request timeout", result.Properties["http"].Properties["timeout"].Description)
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
	require.NotNil(t, result.Properties["timeout"])
	require.Equal(t, "string", result.Properties["timeout"].Type)
	require.Empty(t, result.Properties["timeout"].Format, "format should be cleared")
	require.Equal(t, `^([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$`, result.Properties["timeout"].Pattern)
	require.Contains(t, result.Properties["timeout"].Description, "duration format")
	require.Contains(t, result.Properties["timeout"].Description, "Request timeout")

	// Check interval field - should have pattern and auto-generated description hint
	require.NotNil(t, result.Properties["interval"])
	require.Equal(t, "string", result.Properties["interval"].Type)
	require.Empty(t, result.Properties["interval"].Format)
	require.Equal(t, `^([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$`, result.Properties["interval"].Pattern)
}
