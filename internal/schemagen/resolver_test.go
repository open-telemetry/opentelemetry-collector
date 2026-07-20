// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLoader is a test helper that returns pre-configured schemas keyed by cache key.
type mockLoader struct {
	schemas map[string]*ConfigsMetadata
}

func (m *mockLoader) Load(ref Ref) (*ConfigsMetadata, error) {
	cacheKey := ref.CacheKey()
	if md, ok := m.schemas[cacheKey]; ok {
		return md, nil
	}
	return nil, fmt.Errorf("schema not found for ref: %s", cacheKey)
}

// nilResultLoader returns (nil, nil) for any ref.
type nilResultLoader struct{}

func (n *nilResultLoader) Load(_ Ref) (*ConfigsMetadata, error) { return nil, nil }

func TestResolver_ResolveSchema_BasicMetadata(t *testing.T) {
	resolver := &Resolver{
		loader: NewLoader(""),
	}

	src := &ConfigsMetadata{Config: &ConfigMetadata{
		Description: "OTLP receiver configuration",
		Type:        "object",
	}}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	require.Equal(t, "OTLP receiver configuration", result.Config.Description)
	require.Equal(t, "object", result.Config.Type)
}

func TestResolver_ResolveSchema_InternalReference(t *testing.T) {
	resolver := &Resolver{
		loader: NewLoader(""),
	}

	src := &ConfigsMetadata{
		Config: &ConfigMetadata{
			Type: "object",
			Properties: map[string]*ConfigMetadata{
				"config": {
					Ref: "target_type",
				},
			},
		},
		ExportedConfigs: map[string]*ConfigMetadata{
			"target_type": {
				Type:        "string",
				Description: "Target type description",
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	require.Equal(t, "object", result.Config.Type)
	require.NotNil(t, result.Config.Properties["config"])
	require.Equal(t, "string", result.Config.Properties["config"].Type)
	require.Equal(t, "Target type description", result.Config.Properties["config"].Description)
}

func TestResolver_ResolveSchema_UnknownInternalReference(t *testing.T) {
	resolver := &Resolver{
		loader: NewLoader(""),
	}

	src := &ConfigsMetadata{Config: &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"config": {
				Ref: "unknown_type",
			},
		},
	}}

	// Should use "any" type because the internal reference doesn't exist
	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	require.Empty(t, result.Config.Properties["config"].Type)
}

func TestResolver_ResolveSchema_NestedStructures(t *testing.T) {
	resolver := &Resolver{
		loader: NewLoader(""),
	}

	src := &ConfigsMetadata{Config: &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"nested": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"field1": {Type: "string"},
					"field2": {Type: "integer"},
				},
			},
		},
	}}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	nested := result.Config.Properties["nested"]
	require.NotNil(t, nested)
	require.Equal(t, "object", nested.Type)
	require.Equal(t, "string", nested.Properties["field1"].Type)
	require.Equal(t, "integer", nested.Properties["field2"].Type)
}

func TestResolver_ResolveSchema_ArrayItems(t *testing.T) {
	resolver := &Resolver{
		loader: NewLoader(""),
	}

	src := &ConfigsMetadata{Config: &ConfigMetadata{
		Type: "array",
		Items: &ConfigMetadata{
			Type: "object",
			Properties: map[string]*ConfigMetadata{
				"name": {Type: "string"},
			},
		},
	}}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	require.Equal(t, "array", result.Config.Type)
	require.NotNil(t, result.Config.Items)
	require.Equal(t, "object", result.Config.Items.Type)
	require.NotNil(t, result.Config.Items.Properties["name"])
}

func TestResolver_LoadExternalRef_Success(t *testing.T) {
	ml := &mockLoader{
		schemas: map[string]*ConfigsMetadata{
			"go.opentelemetry.io/collector/scraper/scraperhelper.controller_config": {
				ExportedConfigs: map[string]*ConfigMetadata{
					"controller_config": {
						Type: "object",
						Properties: map[string]*ConfigMetadata{
							"timeout": {Type: "string"},
						},
					},
				},
			},
		},
	}

	resolver := &Resolver{
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
		loader: NewLoader(""),
	}

	ref := NewRef("invalid/path/without/namespace")
	result, err := resolver.loadExternalRef(ref)
	require.Error(t, err)
	require.Nil(t, result)
}

func TestResolver_LoadExternalRef_TypeNotFound(t *testing.T) {
	ml := &mockLoader{
		schemas: map[string]*ConfigsMetadata{
			"go.opentelemetry.io/collector/scraper/scraperhelper.controller_config": {
				ExportedConfigs: map[string]*ConfigMetadata{
					"other_type": {Type: "string"},
				},
			},
		},
	}

	resolver := &Resolver{
		loader: ml,
	}

	ref := NewRef("go.opentelemetry.io/collector/scraper/scraperhelper.controller_config")
	result, err := resolver.loadExternalRef(ref)
	require.Error(t, err)
	require.Contains(t, err.Error(), "type \"controller_config\" not found")
	require.Nil(t, result)
}

func TestResolver_LoadExternalRef_NilResult(t *testing.T) {
	ml := &nilResultLoader{}
	resolver := &Resolver{
		loader: ml,
	}

	ref := NewRef("go.opentelemetry.io/collector/config/confighttp.client_config")
	result, err := resolver.loadExternalRef(ref)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no loader could resolve external reference")
	require.Nil(t, result)
}

func TestResolver_LoadExternalRef_InternalResolutionError(t *testing.T) {
	brokenSchema := &ConfigsMetadata{
		ExportedConfigs: map[string]*ConfigMetadata{
			"config": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"field": {
						Ref: "/",
					},
				},
			},
		},
	}

	ml := &mockLoader{
		schemas: map[string]*ConfigsMetadata{
			"go.opentelemetry.io/collector/config/confighttp.config": brokenSchema,
		},
	}

	resolver := &Resolver{
		loader: ml,
	}

	ref := NewRef("go.opentelemetry.io/collector/config/confighttp.config")
	result, err := resolver.loadExternalRef(ref)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to resolve internal references in external schema")
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
			result := r.IsExternal()
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestNewResolver(t *testing.T) {
	dir := t.TempDir()
	r := NewResolver(dir)
	require.NotNil(t, r)
	require.NotNil(t, r.loader)
}

func TestResolver_ResolveSchema_ExternalReference_Integration(t *testing.T) {
	confighttpSchema := &ConfigsMetadata{
		ExportedConfigs: map[string]*ConfigMetadata{
			"client_config": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"endpoint": {Type: "string", Description: "HTTP endpoint"},
					"timeout":  {Type: "string", Description: "Request timeout"},
				},
			},
		},
	}

	ml := &mockLoader{
		schemas: map[string]*ConfigsMetadata{
			"go.opentelemetry.io/collector/config/confighttp.client_config": confighttpSchema,
		},
	}

	resolver := &Resolver{
		loader: ml,
	}

	src := &ConfigsMetadata{Config: &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"http": {
				Ref: "go.opentelemetry.io/collector/config/confighttp.client_config",
			},
		},
	}}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	http := result.Config.Properties["http"]
	require.NotNil(t, http)
	require.Equal(t, "object", http.Type)
	require.Equal(t, "HTTP endpoint", http.Properties["endpoint"].Description)
	require.Equal(t, "Request timeout", http.Properties["timeout"].Description)
}

func TestResolver_ResolveSchema_DurationFormat(t *testing.T) {
	resolver := &Resolver{
		loader: NewLoader(""),
	}

	src := &ConfigsMetadata{Config: &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"timeout": {
				Type:        "duration",
				Description: "Request timeout",
			},
			"interval": {
				Type: "duration",
			},
		},
	}}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)

	timeout := result.Config.Properties["timeout"]
	require.Equal(t, "string", timeout.Type)
	require.Empty(t, timeout.Format, "format should be cleared")
	require.Equal(t, "time.Duration", timeout.GoType)
	require.Equal(t, `^([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$`, timeout.Pattern)
	require.Equal(t, "Request timeout", timeout.Description)

	interval := result.Config.Properties["interval"]
	require.Equal(t, "string", interval.Type)
	require.Empty(t, interval.Format)
	require.Equal(t, "time.Duration", interval.GoType)
	require.Equal(t, `^([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$`, interval.Pattern)
}

func TestResolver_ResolveSchema_OriginConvertsLocalRefToExternal(t *testing.T) {
	configauthSchema := &ConfigsMetadata{
		ExportedConfigs: map[string]*ConfigMetadata{
			"config": {
				Type:        "object",
				Description: "Auth configuration",
				Properties: map[string]*ConfigMetadata{
					"token": {Type: "string"},
				},
			},
		},
	}

	confighttpSchema := &ConfigsMetadata{
		ExportedConfigs: map[string]*ConfigMetadata{
			"client_config": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"endpoint": {Type: "string"},
					"auth": {
						Ref: "/config/configauth.config",
					},
				},
			},
		},
	}

	ml := &mockLoader{
		schemas: map[string]*ConfigsMetadata{
			"go.opentelemetry.io/collector/config/confighttp.client_config": confighttpSchema,
			"go.opentelemetry.io/collector/config/configauth.config":        configauthSchema,
		},
	}

	resolver := &Resolver{
		loader: ml,
	}

	src := &ConfigsMetadata{Config: &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"http": {
				Ref: "go.opentelemetry.io/collector/config/confighttp.client_config",
			},
		},
	}}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	http := result.Config.Properties["http"]
	require.NotNil(t, http)
	require.Equal(t, "object", http.Type)
	require.Equal(t, "string", http.Properties["endpoint"].Type)
	auth := http.Properties["auth"]
	require.NotNil(t, auth)
	require.Equal(t, "object", auth.Type)
	require.Equal(t, "Auth configuration", auth.Description)
	require.NotNil(t, auth.Properties["token"])
}

func TestResolver_ResolveSchema_NestedOriginPropagation(t *testing.T) {
	// Schema A (remote) → local ref → Schema B (also remote) → local ref → Schema C
	schemaC := &ConfigsMetadata{
		ExportedConfigs: map[string]*ConfigMetadata{
			"tls_config": {
				Type:        "object",
				Description: "TLS configuration",
			},
		},
	}

	schemaB := &ConfigsMetadata{
		ExportedConfigs: map[string]*ConfigMetadata{
			"auth_config": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"tls": {
						Ref: "/config/configtls.tls_config",
					},
				},
			},
		},
	}

	schemaA := &ConfigsMetadata{
		ExportedConfigs: map[string]*ConfigMetadata{
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
		schemas: map[string]*ConfigsMetadata{
			"go.opentelemetry.io/collector/config/confighttp.client_config": schemaA,
			"go.opentelemetry.io/collector/config/configauth.auth_config":   schemaB,
			"go.opentelemetry.io/collector/config/configtls.tls_config":     schemaC,
		},
	}

	resolver := &Resolver{
		loader: ml,
	}

	src := &ConfigsMetadata{Config: &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"http": {
				Ref: "go.opentelemetry.io/collector/config/confighttp.client_config",
			},
		},
	}}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	auth := result.Config.Properties["http"].Properties["auth"]
	require.NotNil(t, auth)
	require.Equal(t, "object", auth.Type)
	tls := auth.Properties["tls"]
	require.NotNil(t, tls)
	require.Equal(t, "object", tls.Type)
	require.Equal(t, "TLS configuration", tls.Description)
}

func TestResolver_ResolveSchema_RelativeRefWithOrigin(t *testing.T) {
	metadataSchema := &ConfigsMetadata{
		ExportedConfigs: map[string]*ConfigMetadata{
			"metrics_config": {
				Type:        "object",
				Description: "Metrics configuration",
			},
		},
	}

	confighttpSchema := &ConfigsMetadata{
		ExportedConfigs: map[string]*ConfigMetadata{
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
		schemas: map[string]*ConfigsMetadata{
			"go.opentelemetry.io/collector/config/confighttp.client_config":                    confighttpSchema,
			"go.opentelemetry.io/collector/config/confighttp/internal/metadata.metrics_config": metadataSchema,
		},
	}

	resolver := &Resolver{
		loader: ml,
	}

	src := &ConfigsMetadata{Config: &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"http": {
				Ref: "go.opentelemetry.io/collector/config/confighttp.client_config",
			},
		},
	}}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	metrics := result.Config.Properties["http"].Properties["metrics"]
	require.NotNil(t, metrics)
	require.Equal(t, "object", metrics.Type)
	require.Equal(t, "Metrics configuration", metrics.Description)
}

func TestResolver_ResolveSchema_ParentRelativeRefWithOrigin(t *testing.T) {
	configtlsSchema := &ConfigsMetadata{
		ExportedConfigs: map[string]*ConfigMetadata{
			"tls_config": {
				Type:        "object",
				Description: "TLS settings",
			},
		},
	}

	confighttpSchema := &ConfigsMetadata{
		ExportedConfigs: map[string]*ConfigMetadata{
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
		schemas: map[string]*ConfigsMetadata{
			"go.opentelemetry.io/collector/config/confighttp.client_config": confighttpSchema,
			"go.opentelemetry.io/collector/config/configtls.tls_config":     configtlsSchema,
		},
	}

	resolver := &Resolver{
		loader: ml,
	}

	src := &ConfigsMetadata{Config: &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"http": {
				Ref: "go.opentelemetry.io/collector/config/confighttp.client_config",
			},
		},
	}}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	tls := result.Config.Properties["http"].Properties["tls"]
	require.NotNil(t, tls)
	require.Equal(t, "object", tls.Type)
	require.Equal(t, "TLS settings", tls.Description)
}

func TestResolver_ResolveSchema_LoaderError(t *testing.T) {
	ml := &mockLoader{schemas: map[string]*ConfigsMetadata{}}

	resolver := &Resolver{
		loader: ml,
	}

	src := &ConfigsMetadata{Config: &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"http": {
				Ref: "go.opentelemetry.io/collector/config/confighttp.client_config",
			},
		},
	}}

	result, err := resolver.ResolveSchema(src)
	require.Error(t, err)
	require.Nil(t, result)
}

func TestResolver_ResolveRef_InvalidRefFormat(t *testing.T) {
	resolver := &Resolver{
		loader: NewLoader(""),
	}

	src := &ConfigsMetadata{Config: &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"bad": {
				Ref: "/",
			},
		},
	}}
	_, err := resolver.ResolveSchema(src)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid reference format")
}

func TestResolver_ResolveSchema_PointerFields(t *testing.T) {
	resolver := &Resolver{
		loader: NewLoader(""),
	}

	minItems := 1
	maxItems := 10
	src := &ConfigsMetadata{Config: &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"tags": {
				Type:     "array",
				Items:    &ConfigMetadata{Type: "string"},
				MinItems: &minItems,
				MaxItems: &maxItems,
			},
		},
	}}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	tags := result.Config.Properties["tags"]
	require.NotNil(t, tags)
	require.Equal(t, "array", tags.Type)
	require.NotNil(t, tags.Items)
	require.Equal(t, "string", tags.Items.Type)
}

func TestResolver_ResolveSchema_PreservesIntAndFloatPointers(t *testing.T) {
	// Regression test: *int and *float64 pointer fields must be copied by the resolver.
	resolver := &Resolver{
		loader: NewLoader(""),
	}

	minLen, maxLen := 1, 255
	minItems, maxItems := 2, 10
	minProps, maxProps := 1, 5
	minimum, maximum := 0.5, 100.0
	exclMin, exclMax := 0.0, 101.0

	src := &ConfigsMetadata{Config: &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"name": {
				Type:      "string",
				MinLength: &minLen,
				MaxLength: &maxLen,
			},
			"tags": {
				Type:     "array",
				MinItems: &minItems,
				MaxItems: &maxItems,
			},
			"meta": {
				Type:          "object",
				MinProperties: &minProps,
				MaxProperties: &maxProps,
			},
			"score": {
				Type:             "number",
				Minimum:          &minimum,
				Maximum:          &maximum,
				ExclusiveMinimum: &exclMin,
				ExclusiveMaximum: &exclMax,
			},
		},
	}}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)

	name := result.Config.Properties["name"]
	require.NotNil(t, name.MinLength)
	require.Equal(t, minLen, *name.MinLength)
	require.NotNil(t, name.MaxLength)
	require.Equal(t, maxLen, *name.MaxLength)

	tags := result.Config.Properties["tags"]
	require.NotNil(t, tags.MinItems)
	require.Equal(t, minItems, *tags.MinItems)
	require.NotNil(t, tags.MaxItems)
	require.Equal(t, maxItems, *tags.MaxItems)

	meta := result.Config.Properties["meta"]
	require.NotNil(t, meta.MinProperties)
	require.Equal(t, minProps, *meta.MinProperties)
	require.NotNil(t, meta.MaxProperties)
	require.Equal(t, maxProps, *meta.MaxProperties)

	score := result.Config.Properties["score"]
	require.NotNil(t, score.Minimum)
	require.InEpsilon(t, minimum, *score.Minimum, 1e-9)
	require.NotNil(t, score.Maximum)
	require.InEpsilon(t, maximum, *score.Maximum, 1e-9)
	require.NotNil(t, score.ExclusiveMinimum)
	require.InDelta(t, exclMin, *score.ExclusiveMinimum, 1e-9)
	require.NotNil(t, score.ExclusiveMaximum)
	require.InEpsilon(t, exclMax, *score.ExclusiveMaximum, 1e-9)
}

func TestResolver_ResolveSchema_DecoratesPropNames(t *testing.T) {
	resolver := &Resolver{
		loader: NewLoader(""),
	}

	src := &ConfigsMetadata{Config: &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"endpoint": {Type: "string"},
			"storage": {
				Type:   "string",
				GoType: "go.opentelemetry.io/collector/component.ID",
			},
		},
	}}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)

	require.Equal(t, "endpoint", result.Config.Properties["endpoint"].GoStruct.FieldName)
	require.Equal(t, "storage", result.Config.Properties["storage"].GoStruct.FieldName)
}

// ---------------------------------------------------------------------------
// Extended type alias tests
// ---------------------------------------------------------------------------

func TestResolver_ResolveSchema_ExtendedTypes_InProperties(t *testing.T) {
	resolver := &Resolver{
		loader: NewLoader(""),
	}

	src := &ConfigsMetadata{Config: &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"count":     {Type: "int64"},
			"ratio":     {Type: "float32"},
			"timeout":   {Type: "duration"},
			"ts":        {Type: "time"},
			"token":     {Type: "opaque_string"},
			"component": {Type: "id"},
		},
	}}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)

	count := result.Config.Properties["count"]
	require.NotNil(t, count)
	assert.Equal(t, "integer", count.Type)
	assert.Equal(t, "int64", count.GoType)

	ratio := result.Config.Properties["ratio"]
	require.NotNil(t, ratio)
	assert.Equal(t, "number", ratio.Type)
	assert.Equal(t, "float32", ratio.GoType)

	timeout := result.Config.Properties["timeout"]
	require.NotNil(t, timeout)
	assert.Equal(t, "string", timeout.Type)
	assert.Equal(t, "time.Duration", timeout.GoType)
	assert.Equal(t, goDurationPattern, timeout.Pattern)
	assert.Empty(t, timeout.Format)

	ts := result.Config.Properties["ts"]
	require.NotNil(t, ts)
	assert.Equal(t, "string", ts.Type)
	assert.Equal(t, "time.Time", ts.GoType)

	token := result.Config.Properties["token"]
	require.NotNil(t, token)
	assert.Equal(t, "string", token.Type)
	assert.Equal(t, "go.opentelemetry.io/collector/config/configopaque.String", token.GoType)

	comp := result.Config.Properties["component"]
	require.NotNil(t, comp)
	assert.Equal(t, "string", comp.Type)
	assert.Equal(t, "go.opentelemetry.io/collector/component.ID", comp.GoType)
}

func TestResolver_ResolveSchema_ExtendedType_InArrayItems(t *testing.T) {
	resolver := &Resolver{
		loader: NewLoader(""),
	}

	src := &ConfigsMetadata{Config: &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"ids": {
				Type:  "array",
				Items: &ConfigMetadata{Type: "id"},
			},
		},
	}}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	ids := result.Config.Properties["ids"]
	require.NotNil(t, ids)
	assert.Equal(t, "array", ids.Type)
	require.NotNil(t, ids.Items)
	assert.Equal(t, "string", ids.Items.Type)
	assert.Equal(t, "go.opentelemetry.io/collector/component.ID", ids.Items.GoType)
}

func TestResolver_ResolveSchema_ExtendedType_InAdditionalProperties(t *testing.T) {
	resolver := &Resolver{
		loader: NewLoader(""),
	}

	src := &ConfigsMetadata{Config: &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"secrets": {
				Type:                 "object",
				AdditionalProperties: &ConfigMetadata{Type: "opaque_string"},
			},
		},
	}}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	secrets := result.Config.Properties["secrets"]
	require.NotNil(t, secrets)
	require.NotNil(t, secrets.AdditionalProperties)
	assert.Equal(t, "string", secrets.AdditionalProperties.Type)
	assert.Equal(t, "go.opentelemetry.io/collector/config/configopaque.String", secrets.AdditionalProperties.GoType)
}

func TestResolver_ResolveSchema_ExtendedType_InDefs_ViaRef(t *testing.T) {
	resolver := &Resolver{
		loader: NewLoader(""),
	}

	src := &ConfigsMetadata{
		Config: &ConfigMetadata{
			Type: "object",
			Properties: map[string]*ConfigMetadata{
				"count": {Ref: "counter"},
			},
		},
		ExportedConfigs: map[string]*ConfigMetadata{
			"counter": {Type: "int64", InternalOnly: true},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	count := result.Config.Properties["count"]
	require.NotNil(t, count)
	assert.Equal(t, "integer", count.Type)
	assert.Equal(t, "int64", count.GoType)
}

func TestResolver_ResolveSchema_ExtendedType_OpaqueMap(t *testing.T) {
	resolver := &Resolver{
		loader: NewLoader(""),
	}

	src := &ConfigsMetadata{Config: &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"headers": {Type: "opaque_map"},
		},
	}}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	headers := result.Config.Properties["headers"]
	require.NotNil(t, headers)
	assert.Equal(t, "object", headers.Type)
	assert.Equal(t, "go.opentelemetry.io/collector/config/configopaque.MapList", headers.GoType)
	require.NotNil(t, headers.AdditionalProperties)
	assert.Equal(t, "string", headers.AdditionalProperties.Type)
}

func TestResolver_ResolveSchema_ExtendedType_UnknownAlias_Error(t *testing.T) {
	resolver := &Resolver{
		loader: NewLoader(""),
	}

	src := &ConfigsMetadata{Config: &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"bad": {Type: "not_a_type"},
		},
	}}

	_, err := resolver.ResolveSchema(src)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not_a_type")
}
