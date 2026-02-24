// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolver_ResolveSchema_BasicMetadata(t *testing.T) {
	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/receiver/otlpreceiver",
		class:  "receiver",
		name:   "otlp",
		loader: NewLoader(""),
	}

	src := &ConfigMetadata{
		Description: "OTLP receiver configuration",
		Type:        "object",
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	require.Equal(t, schemaVersion, result.Schema)
	require.Equal(t, "go.opentelemetry.io/collector/receiver/otlpreceiver", result.ID)
	require.Equal(t, "receiver/otlp", result.Title)
	require.Equal(t, "OTLP receiver configuration", result.Description)
	require.Equal(t, "object", result.Type)
}

func TestResolver_ResolveSchema_InternalReference(t *testing.T) {
	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: NewLoader(""),
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
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: NewLoader(""),
	}

	src := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"config": {
				Ref: "unknown_type",
			},
		},
	}

	// Should use "any" type because the internal reference doesn't exist
	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	require.Nil(t, result.Properties["config"].Type)
}

func TestResolver_ResolveSchema_NestedStructures(t *testing.T) {
	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: NewLoader(""),
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
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: NewLoader(""),
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
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: NewLoader(""),
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
	ml := &mockLoader{
		schemas: map[string]*ConfigMetadata{
			"go.opentelemetry.io/collector/scraper/scraperhelper.controller_config": {
				Type: "object",
				Defs: map[string]*ConfigMetadata{
					"controller_config": {
						Type: "object",
						Properties: map[string]*ConfigMetadata{
							"timeout": {
								Type: "string",
							},
						},
					},
				},
			},
		},
	}

	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: ml,
	}

	ref := NewRef("go.opentelemetry.io/collector/scraper/scraperhelper.controller_config")
	result, err := resolver.loadExternalRef(ref)
	require.NoError(t, err)
	require.Equal(t, "object", result.Type)
	require.NotNil(t, result.Properties["timeout"])
	require.Equal(t, "string", result.Properties["timeout"].Type)
}

func TestResolver_LoadExternalRef_InvalidPath(t *testing.T) {
	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: NewLoader(""),
	}

	ref := NewRef("invalid/path/without/namespace")
	result, err := resolver.loadExternalRef(ref)
	require.Error(t, err)
	require.Nil(t, result)
}

func TestResolver_LoadExternalRef_TypeNotFound(t *testing.T) {
	ml := &mockLoader{
		schemas: map[string]*ConfigMetadata{
			"go.opentelemetry.io/collector/scraper/scraperhelper.controller_config": {
				Type: "object",
				Defs: map[string]*ConfigMetadata{
					"other_type": {
						Type: "string",
					},
				},
			},
		},
	}

	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: ml,
	}

	ref := NewRef("go.opentelemetry.io/collector/scraper/scraperhelper.controller_config")
	result, err := resolver.loadExternalRef(ref)
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
			name:     "collector external ref - known namespace",
			ref:      "go.opentelemetry.io/collector/scraper/scraperhelper.controller_config",
			expected: true,
		},
		{
			name:     "contrib external ref - known namespace",
			ref:      "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/mysqlreceiver.config",
			expected: true,
		},
		{
			name:     "relative path - local not external",
			ref:      "./internal/metadata.metrics_builder_config",
			expected: false,
		},
		{
			name:     "internal ref - simple name",
			ref:      "target_type",
			expected: false,
		},
		{
			name:     "unsupported namespace - still external (not internal/local)",
			ref:      "github.com/example/custom.config",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRef(tt.ref)
			result := r.isExternal()
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestResolver_ResolveSchema_ExternalReference_Integration(t *testing.T) {
	// Use mockLoader instead of real file loading to avoid repo root dependency
	confighttpSchema := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"client_config": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"endpoint": {
						Type:        "string",
						Description: "HTTP endpoint",
					},
					"timeout": {
						Type:        "string",
						Description: "Request timeout",
					},
				},
			},
		},
	}

	ml := &mockLoader{
		schemas: map[string]*ConfigMetadata{
			"go.opentelemetry.io/collector/config/confighttp.client_config": confighttpSchema,
		},
	}

	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/receiver/otlpreceiver",
		class:  "receiver",
		name:   "otlp",
		loader: ml,
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
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: NewLoader(""),
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

// mockLoader is a test helper that returns pre-configured schemas keyed by cache key.
type mockLoader struct {
	schemas map[string]*ConfigMetadata
}

func (m *mockLoader) Load(ref Ref) (*ConfigMetadata, error) {
	cacheKey := ref.CacheKey()
	if md, ok := m.schemas[cacheKey]; ok {
		return md, nil
	}
	return nil, fmt.Errorf("schema not found for ref: %s", cacheKey)
}

func TestResolver_ResolveSchema_OriginConvertsLocalRefToExternal(t *testing.T) {
	// confighttp schema contains a local absolute ref to /config/configauth.config
	// When loaded as an external ref from the collector namespace, the local ref
	// should be converted to go.opentelemetry.io/collector/config/configauth.config
	configauthSchema := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"config": {
				Type:        "object",
				Description: "Auth configuration",
				Properties: map[string]*ConfigMetadata{
					"token": {Type: "string"},
				},
			},
		},
	}

	confighttpSchema := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"client_config": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"endpoint": {Type: "string"},
					"auth": {
						// This is the key: a local absolute ref inside an externally-loaded schema
						Ref: "/config/configauth.config",
					},
				},
			},
		},
	}

	ml := &mockLoader{
		schemas: map[string]*ConfigMetadata{
			// When loading the reference to confighttp.client_config, we get the whole schema with all defs
			"go.opentelemetry.io/collector/config/confighttp.client_config": confighttpSchema,
			// When resolving the local ref /config/configauth.config -> go.opentelemetry.io/collector/config/configauth.config
			"go.opentelemetry.io/collector/config/configauth.config": configauthSchema,
		},
	}

	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/receiver/otlpreceiver",
		class:  "receiver",
		name:   "otlp",
		loader: ml,
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
	require.NotNil(t, result.Properties["http"])
	require.Equal(t, "object", result.Properties["http"].Type)
	require.NotNil(t, result.Properties["http"].Properties["endpoint"])
	require.Equal(t, "string", result.Properties["http"].Properties["endpoint"].Type)
	// The auth property should have been resolved through origin-aware conversion
	require.NotNil(t, result.Properties["http"].Properties["auth"])
	require.Equal(t, "object", result.Properties["http"].Properties["auth"].Type)
	require.Equal(t, "Auth configuration", result.Properties["http"].Properties["auth"].Description)
	require.NotNil(t, result.Properties["http"].Properties["auth"].Properties["token"])
}

func TestResolver_ResolveSchema_LocalRefWithOriginConversion(t *testing.T) {
	// When a local ref is encountered in an externally-loaded schema, it should be converted
	// using the origin namespace
	configauthSchema := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"config": {
				Type:        "object",
				Description: "Auth config",
			},
		},
	}

	confighttpSchema := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"client_config": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"auth": {
						// This is a local absolute ref inside an externally-loaded schema
						Ref: "/config/configauth.config",
					},
				},
			},
		},
	}

	ml := &mockLoader{
		schemas: map[string]*ConfigMetadata{
			"go.opentelemetry.io/collector/config/confighttp.client_config": confighttpSchema,
			// After origin conversion, /config/configauth.config becomes:
			"go.opentelemetry.io/collector/config/configauth.config": configauthSchema,
		},
	}

	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/receiver/otlpreceiver",
		class:  "receiver",
		name:   "otlp",
		loader: ml,
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
	require.NotNil(t, result.Properties["http"])
	require.Equal(t, "object", result.Properties["http"].Type)
	// auth should be resolved through origin-aware conversion
	require.NotNil(t, result.Properties["http"].Properties["auth"])
	require.Equal(t, "object", result.Properties["http"].Properties["auth"].Type)
	require.Equal(t, "Auth config", result.Properties["http"].Properties["auth"].Description)
}

func TestResolver_ResolveSchema_NestedOriginPropagation(t *testing.T) {
	// Schema A (remote) → local ref → Schema B (also remote) → local ref → Schema C
	// Verify the origin propagates through all levels.

	schemaC := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"tls_config": {
				Type:        "object",
				Description: "TLS configuration",
			},
		},
	}

	schemaB := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"auth_config": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"tls": {
						// Nested local ref — should also be converted using origin
						Ref: "/config/configtls.tls_config",
					},
				},
			},
		},
	}

	schemaA := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"client_config": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"auth": {
						Ref: "/config/configauth.auth_config",
					},
				},
			},
		},
	}

	ml := &mockLoader{
		schemas: map[string]*ConfigMetadata{
			"go.opentelemetry.io/collector/config/confighttp.client_config": schemaA,
			"go.opentelemetry.io/collector/config/configauth.auth_config":   schemaB,
			"go.opentelemetry.io/collector/config/configtls.tls_config":     schemaC,
		},
	}

	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/receiver/otlpreceiver",
		class:  "receiver",
		name:   "otlp",
		loader: ml,
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
	require.NotNil(t, result.Properties["http"])
	// auth should be resolved through origin-aware conversion
	auth := result.Properties["http"].Properties["auth"]
	require.NotNil(t, auth)
	require.Equal(t, "object", auth.Type)
	// tls inside auth should also be resolved through origin propagation
	tls := auth.Properties["tls"]
	require.NotNil(t, tls)
	require.Equal(t, "object", tls.Type)
	require.Equal(t, "TLS configuration", tls.Description)
}

func TestResolver_ResolveSchema_RelativeRefWithOrigin(t *testing.T) {
	// A relative ref like ./internal/metadata.metrics_config inside a schema loaded
	// from go.opentelemetry.io/collector/config/confighttp should resolve to
	// go.opentelemetry.io/collector/config/confighttp/internal/metadata.metrics_config
	metadataSchema := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"metrics_config": {
				Type:        "object",
				Description: "Metrics configuration",
			},
		},
	}

	confighttpSchema := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"client_config": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"metrics": {
						Ref: "./internal/metadata.metrics_config",
					},
				},
			},
		},
	}

	ml := &mockLoader{
		schemas: map[string]*ConfigMetadata{
			"go.opentelemetry.io/collector/config/confighttp.client_config": confighttpSchema,
			// ./internal/metadata resolved against origin go.opentelemetry.io/collector/config/confighttp:
			"go.opentelemetry.io/collector/config/confighttp/internal/metadata.metrics_config": metadataSchema,
		},
	}

	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/receiver/otlpreceiver",
		class:  "receiver",
		name:   "otlp",
		loader: ml,
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
	require.NotNil(t, result.Properties["http"])
	metrics := result.Properties["http"].Properties["metrics"]
	require.NotNil(t, metrics)
	require.Equal(t, "object", metrics.Type)
	require.Equal(t, "Metrics configuration", metrics.Description)
}

func TestResolver_ResolveSchema_ParentRelativeRefWithOrigin(t *testing.T) {
	// A parent-relative ref like ../configtls.tls_config inside a schema loaded
	// from go.opentelemetry.io/collector/config/confighttp should resolve to
	// go.opentelemetry.io/collector/config/configtls.tls_config
	configtlsSchema := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"tls_config": {
				Type:        "object",
				Description: "TLS settings",
			},
		},
	}

	confighttpSchema := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"client_config": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"tls": {
						Ref: "../configtls.tls_config",
					},
				},
			},
		},
	}

	ml := &mockLoader{
		schemas: map[string]*ConfigMetadata{
			"go.opentelemetry.io/collector/config/confighttp.client_config": confighttpSchema,
			// ../configtls resolved against parent of go.opentelemetry.io/collector/config/confighttp:
			"go.opentelemetry.io/collector/config/configtls.tls_config": configtlsSchema,
		},
	}

	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/receiver/otlpreceiver",
		class:  "receiver",
		name:   "otlp",
		loader: ml,
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
	require.NotNil(t, result.Properties["http"])
	tls := result.Properties["http"].Properties["tls"]
	require.NotNil(t, tls)
	require.Equal(t, "object", tls.Type)
	require.Equal(t, "TLS settings", tls.Description)
}

func TestNewResolver(t *testing.T) {
	dir := t.TempDir()
	r := NewResolver("go.opentelemetry.io/collector/receiver/otlp", "receiver", "otlp", dir)
	require.NotNil(t, r)
	require.Equal(t, "go.opentelemetry.io/collector/receiver/otlp", r.pkgID)
	require.Equal(t, "receiver", r.class)
	require.Equal(t, "otlp", r.name)
	require.NotNil(t, r.loader)
}

func TestResolver_ResolveSchema_UnknownNamespaceFallback(t *testing.T) {
	// An external ref with an unsupported namespace should fall back to "any" type
	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: NewLoader(""),
	}

	src := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"custom": {
				Ref: "github.com/example/custom.config",
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	require.NotNil(t, result.Properties["custom"])
	require.Equal(t, "github.com/example/custom.config", result.Properties["custom"].GoType)
	require.Contains(t, result.Properties["custom"].Comment, "any")
}

func TestResolver_ResolveSchema_LoaderError(t *testing.T) {
	ml := &mockLoader{schemas: map[string]*ConfigMetadata{}}

	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: ml,
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
	require.Error(t, err)
	require.Nil(t, result)
}

func TestResolver_ResolveSchema_ContentSchema(t *testing.T) {
	// Test that ContentSchema (*ConfigMetadata pointer field) is recursively resolved
	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: NewLoader(""),
	}

	src := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"body": {
				Type:             "string",
				ContentMediaType: "application/json",
				ContentSchema: &ConfigMetadata{
					Type: "object",
					Properties: map[string]*ConfigMetadata{
						"name": {Type: "string"},
					},
				},
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	require.NotNil(t, result.Properties["body"])
	require.NotNil(t, result.Properties["body"].ContentSchema)
	require.Equal(t, "object", result.Properties["body"].ContentSchema.Type)
}
