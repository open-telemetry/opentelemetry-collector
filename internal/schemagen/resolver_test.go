// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen

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

func TestResolver_ResolveSchema_InternalReferencePreservesInlineValidationOverrides(t *testing.T) {
	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: NewLoader(""),
	}

	minimum, aliasMaximum, fieldMaximum := 1.0, 65535.0, 10000.0
	src := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"port": {
				Ref:     "port_number",
				Maximum: &fieldMaximum,
			},
		},
		Defs: map[string]*ConfigMetadata{
			"port_number": {
				Type:    "integer",
				GoType:  "int32",
				Minimum: &minimum,
				Maximum: &aliasMaximum,
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)

	port := result.Properties["port"]
	require.NotNil(t, port)
	require.Equal(t, "integer", port.Type)
	require.Equal(t, "port_number", port.ResolvedFrom)
	require.Empty(t, port.GoType)
	require.NotNil(t, port.Minimum)
	require.InEpsilon(t, minimum, *port.Minimum, 1e-9)
	require.NotNil(t, port.Maximum)
	require.InEpsilon(t, fieldMaximum, *port.Maximum, 1e-9)

	def := result.Defs["port_number"]
	require.NotNil(t, def)
	require.Equal(t, "int32", def.GoType)
	require.NotNil(t, def.Maximum)
	require.InEpsilon(t, aliasMaximum, *def.Maximum, 1e-9)
}

func TestResolver_ResolveSchema_DefsOnlyPreservesDefs(t *testing.T) {
	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/pkg",
		class:  "pkg",
		name:   "testpkg",
		loader: NewLoader(""),
	}

	src := &ConfigMetadata{
		Defs: map[string]*ConfigMetadata{
			"sample_config": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"endpoint": {Type: "string"},
				},
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	require.Empty(t, result.Type)
	require.Empty(t, result.Properties)
	require.Contains(t, result.Defs, "sample_config")
	require.Equal(t, "object", result.Defs["sample_config"].Type)
	require.Contains(t, result.Defs["sample_config"].Properties, "endpoint")
}

func TestResolver_ResolveSchema_PreservesInlineDefsWithProperties(t *testing.T) {
	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: NewLoader(""),
	}

	src := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"endpoint": {Type: "string"},
		},
		Defs: map[string]*ConfigMetadata{
			"sample_config": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"host_name": {Type: "string"},
				},
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	require.Contains(t, result.Properties, "endpoint")
	require.Contains(t, result.Defs, "sample_config")
	require.Equal(t, "object", result.Defs["sample_config"].Type)
	require.Contains(t, result.Defs["sample_config"].Properties, "host_name")
}

func TestResolver_ResolveSchema_DropsInternalOnlyDefs(t *testing.T) {
	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: NewLoader(""),
	}

	src := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"endpoint": {Type: "string"},
		},
		Defs: map[string]*ConfigMetadata{
			"metrics_builder_config": {
				Type:         "object",
				InternalOnly: true,
				Properties: map[string]*ConfigMetadata{
					"metrics": {Type: "object"},
				},
			},
			"sample_config": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"host_name": {Type: "string"},
				},
			},
			"exported_metrics_config": {
				Ref: "metrics_builder_config",
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	require.Contains(t, result.Properties, "endpoint")
	require.NotContains(t, result.Defs, "metrics_builder_config")
	require.Contains(t, result.Defs, "sample_config")
	require.Contains(t, result.Defs, "exported_metrics_config")
	require.Contains(t, result.Defs["exported_metrics_config"].Properties, "metrics")
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
	require.Empty(t, result.Properties["config"].Type)
}

func TestResolver_ResolveSchema_AliasChain_InternalToExternal(t *testing.T) {
	externalSchema := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"controller_config": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"timeout": {
						Type:   "string",
						GoType: "time.Duration",
					},
				},
			},
		},
	}

	ml := &mockLoader{
		schemas: map[string]*ConfigMetadata{
			"go.opentelemetry.io/collector/scraper/scraperhelper.controller_config": externalSchema,
		},
	}

	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/cmd/mdatagen/internal/samplescraper",
		class:  "scraper",
		name:   "sample",
		loader: ml,
	}

	src := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			// alias: internal name → external type
			"helper": {
				Ref: "go.opentelemetry.io/collector/scraper/scraperhelper.controller_config",
			},
		},
		Properties: map[string]*ConfigMetadata{
			"controller_config": {
				Ref:     "helper",
				Embed:   true,
				Default: map[string]any{"timeout": "30s"},
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)

	require.Len(t, result.AllOf, 1)
	embedded := result.AllOf[0]

	require.Equal(t, "go.opentelemetry.io/collector/scraper/scraperhelper.controller_config", embedded.ResolvedFrom)
	require.NotNil(t, embedded.Properties["timeout"])
	require.Equal(t, "time.Duration", embedded.Properties["timeout"].GoType)
	require.Equal(t, map[string]any{"timeout": "30s"}, embedded.Default)
}

func TestResolver_ResolveSchema_AliasChain_PreservesCustomExtensions(t *testing.T) {
	externalSchema := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"base": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"level": {Type: "integer"},
				},
			},
		},
	}

	ml := &mockLoader{
		schemas: map[string]*ConfigMetadata{
			"go.opentelemetry.io/collector/config/configbase.base": externalSchema,
		},
	}

	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: ml,
	}

	src := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"alias": {Ref: "go.opentelemetry.io/collector/config/configbase.base"},
		},
		Properties: map[string]*ConfigMetadata{
			"cfg": {
				Ref:         "alias",
				Description: "overridden description",
				Default:     map[string]any{"level": 5},
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)

	cfg := result.Properties["cfg"]
	require.NotNil(t, cfg)
	require.Equal(t, "go.opentelemetry.io/collector/config/configbase.base", cfg.ResolvedFrom)
	require.Equal(t, "overridden description", cfg.Description)
	require.Equal(t, map[string]any{"level": 5}, cfg.Default)
	require.NotNil(t, cfg.Properties["level"])
}

func TestResolver_ResolveSchema_InternalAliasChain(t *testing.T) {
	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: &mockLoader{schemas: map[string]*ConfigMetadata{}},
	}

	src := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"alias": {Ref: "base"},
			"base": {
				Type:        "object",
				Description: "Base configuration",
				Properties: map[string]*ConfigMetadata{
					"endpoint": {Type: "string"},
				},
			},
		},
		Properties: map[string]*ConfigMetadata{
			"cfg": {
				Ref: "alias",
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)

	cfg := result.Properties["cfg"]
	require.NotNil(t, cfg)
	require.Equal(t, "base", cfg.ResolvedFrom)
	require.Equal(t, "object", cfg.Type)
	require.Equal(t, "Base configuration", cfg.Description)
	require.Contains(t, cfg.Properties, "endpoint")
}

func TestResolver_ResolveSchema_DefsOnlyLocalAliasWithSameDefName(t *testing.T) {
	controllerSchema := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"controller_config": {
				Type:        "object",
				Description: "Controller configuration",
				Properties: map[string]*ConfigMetadata{
					"collection_interval": {Type: "string"},
				},
			},
		},
	}

	resolver := &Resolver{
		pkgID: "go.opentelemetry.io/collector/scraper/scraperhelper",
		class: "pkg",
		name:  "scraperhelper",
		loader: &mockLoader{schemas: map[string]*ConfigMetadata{
			"./internal/controller.controller_config": controllerSchema,
		}},
	}

	src := &ConfigMetadata{
		Defs: map[string]*ConfigMetadata{
			"controller_config": {
				Ref: "./internal/controller.controller_config",
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)

	controller := result.Defs["controller_config"]
	require.NotNil(t, controller)
	require.Equal(t, "./internal/controller.controller_config", controller.ResolvedFrom)
	require.Equal(t, "object", controller.Type)
	require.Equal(t, "Controller configuration", controller.Description)
	require.Contains(t, controller.Properties, "collection_interval")
}

func TestResolver_ResolveSchema_ExternalRefDoesNotUseRootDefWithSameName(t *testing.T) {
	externalSchema := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"shared_config": {
				Type:        "object",
				Description: "External shared config",
				Properties: map[string]*ConfigMetadata{
					"external": {Type: "string"},
				},
			},
		},
	}

	ml := &mockLoader{
		schemas: map[string]*ConfigMetadata{
			"go.opentelemetry.io/collector/config/shared.shared_config": externalSchema,
		},
	}

	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: ml,
	}

	src := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"shared_config": {
				Type:        "object",
				Description: "Local shared config",
				Properties: map[string]*ConfigMetadata{
					"local": {Type: "string"},
				},
			},
		},
		Properties: map[string]*ConfigMetadata{
			"cfg": {
				Ref: "go.opentelemetry.io/collector/config/shared.shared_config",
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)

	cfg := result.Properties["cfg"]
	require.NotNil(t, cfg)
	require.Equal(t, "go.opentelemetry.io/collector/config/shared.shared_config", cfg.ResolvedFrom)
	require.Equal(t, "External shared config", cfg.Description)
	require.Contains(t, cfg.Properties, "external")
	require.NotContains(t, cfg.Properties, "local")
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

func TestResolver_ResolveSchema_EmbeddedProperties(t *testing.T) {
	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: NewLoader(""),
	}

	src := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"anonymous_config": {
				Ref:      "anonymous_config",
				Embed:    true,
				GoStruct: GoStructConfig{Anonymous: true},
			},
			"named_config": {
				Ref:   "named_config",
				Embed: true,
			},
			"regular": {
				Type: "string",
			},
		},
		Defs: map[string]*ConfigMetadata{
			"anonymous_config": {Type: "object"},
			"named_config":     {Type: "object"},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	require.Len(t, result.Properties, 1)
	require.Contains(t, result.Properties, "regular")
	require.Len(t, result.AllOf, 2)

	named := findSchema(t, result.AllOf, func(md *ConfigMetadata) bool {
		return md.ResolvedFrom == "named_config"
	})
	require.NotNil(t, named)
	require.True(t, named.Embed)
	require.Equal(t, "named_config", named.EmbeddedName)

	anonymous := findSchema(t, result.AllOf, func(md *ConfigMetadata) bool {
		return md.ResolvedFrom == "anonymous_config"
	})
	require.NotNil(t, anonymous)
	require.True(t, anonymous.Embed)
	require.True(t, anonymous.GoStruct.Anonymous)
	require.Empty(t, anonymous.EmbeddedName)
}

func TestResolver_ResolveSchema_EmbeddedReferencePreservesExtensions(t *testing.T) {
	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: NewLoader(""),
	}

	src := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"base": {
				Ref:         "base_config",
				Embed:       true,
				GoStruct:    GoStructConfig{Anonymous: true},
				Description: "Embedded base config",
			},
		},
		Defs: map[string]*ConfigMetadata{
			"base_config": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"endpoint": {Type: "string"},
				},
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	require.Empty(t, result.Properties)
	require.Len(t, result.AllOf, 1)

	embedded := result.AllOf[0]
	require.Equal(t, "base_config", embedded.ResolvedFrom)
	require.True(t, embedded.Embed)
	require.True(t, embedded.GoStruct.Anonymous)
	require.Empty(t, embedded.EmbeddedName)
	require.Equal(t, "Embedded base config", embedded.Description)
	require.Contains(t, embedded.Properties, "endpoint")
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
			result := r.IsExternal()
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

	// Check timeout field - format should be cleared, GoType and Pattern set
	require.NotNil(t, result.Properties["timeout"])
	require.Equal(t, "string", result.Properties["timeout"].Type)
	require.Empty(t, result.Properties["timeout"].Format, "format should be cleared")
	require.Equal(t, "time.Duration", result.Properties["timeout"].GoType)
	require.Equal(t, `^([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$`, result.Properties["timeout"].Pattern)
	require.Equal(t, "Request timeout", result.Properties["timeout"].Description)

	// Check interval field - format should be cleared, GoType and Pattern set
	require.NotNil(t, result.Properties["interval"])
	require.Equal(t, "string", result.Properties["interval"].Type)
	require.Empty(t, result.Properties["interval"].Format)
	require.Equal(t, "time.Duration", result.Properties["interval"].GoType)
	require.Equal(t, `^([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$`, result.Properties["interval"].Pattern)
}

// mockLoader is a test helper that returns pre-configured schemas keyed by cache key.
type mockLoader struct {
	schemas map[string]*ConfigMetadata
}

func findSchema(t *testing.T, schemas []*ConfigMetadata, match func(*ConfigMetadata) bool) *ConfigMetadata {
	t.Helper()
	for _, md := range schemas {
		if match(md) {
			return md
		}
	}
	require.FailNow(t, "schema not found")
	return nil
}

func (m *mockLoader) Load(ref Ref) (*Metadata, error) {
	cacheKey := ref.CacheKey()
	if md, ok := m.schemas[cacheKey]; ok {
		return &Metadata{Config: md}, nil
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
			"go.opentelemetry.io/collector/config/confighttp.client_config":                    confighttpSchema,
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
			"go.opentelemetry.io/collector/config/configtls.tls_config":     configtlsSchema,
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

func TestResolver_ResolveRef_InvalidRefFormat(t *testing.T) {
	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: NewLoader(""),
	}

	src := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"bad": {
				Ref: "/",
			},
		},
	}
	_, err := resolver.ResolveSchema(src)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid reference format")
}

func TestResolver_LoadExternalRef_NilResult(t *testing.T) {
	ml := &nilResultLoader{}
	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: ml,
	}

	ref := NewRef("go.opentelemetry.io/collector/config/confighttp.client_config")
	result, err := resolver.loadExternalRef(ref)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no loader could resolve external reference")
	require.Nil(t, result)
}

// nilResultLoader returns (nil, nil) for any ref.
type nilResultLoader struct{}

func (n *nilResultLoader) Load(_ Ref) (*Metadata, error) { return nil, nil }

func TestResolver_LoadExternalRef_InternalResolutionError(t *testing.T) {
	brokenSchema := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
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
		schemas: map[string]*ConfigMetadata{
			"go.opentelemetry.io/collector/config/confighttp.config": brokenSchema,
		},
	}

	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: ml,
	}

	ref := NewRef("go.opentelemetry.io/collector/config/confighttp.config")
	result, err := resolver.loadExternalRef(ref)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to resolve internal references in external schema")
	require.Nil(t, result)
}

func TestResolver_ResolveSchema_LocalRef(t *testing.T) {
	localSchema := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"target": {
				Type:        "object",
				Description: "Local target",
			},
		},
	}

	ml := &mockLoader{
		schemas: map[string]*ConfigMetadata{
			"/config/localtype.target": localSchema,
		},
	}

	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: ml,
	}

	src := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"local": {
				Ref: "/config/localtype.target",
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	require.NotNil(t, result.Properties["local"])
	require.Equal(t, "object", result.Properties["local"].Type)
	require.Equal(t, "Local target", result.Properties["local"].Description)
}

func TestResolver_ResolveSchema_LocalRefUsesRootDef(t *testing.T) {
	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: &mockLoader{schemas: map[string]*ConfigMetadata{}},
	}

	src := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"resource_attributes_config": {
				Type:        "object",
				Description: "Injected resource attributes config",
				Properties: map[string]*ConfigMetadata{
					"service.name": {
						Type: "string",
					},
				},
			},
		},
		Properties: map[string]*ConfigMetadata{
			"resource_attributes": {
				Ref: "/config/metadata.resource_attributes_config",
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)

	resourceAttributes := result.Properties["resource_attributes"]
	require.Equal(t, "object", resourceAttributes.Type)
	require.Equal(t, "Injected resource attributes config", resourceAttributes.Description)
	require.Contains(t, resourceAttributes.Properties, "service.name")
	require.Equal(t, "string", resourceAttributes.Properties["service.name"].Type)
}

func TestResolver_ResolveSchema_MapValueError(t *testing.T) {
	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: &mockLoader{schemas: map[string]*ConfigMetadata{}},
	}

	src := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"field": {
				// Ref to an unknown external schema → loader returns error
				Ref: "go.opentelemetry.io/collector/missing/pkg.config",
			},
		},
	}

	_, err := resolver.ResolveSchema(src)
	require.Error(t, err)
}

func TestResolver_ResolveSchema_AllOfError(t *testing.T) {
	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: &mockLoader{schemas: map[string]*ConfigMetadata{}},
	}

	src := &ConfigMetadata{
		Type: "object",
		AllOf: []*ConfigMetadata{
			{
				Ref: "go.opentelemetry.io/collector/missing/pkg.config",
			},
		},
	}

	_, err := resolver.ResolveSchema(src)
	require.Error(t, err)
}

func TestResolver_ResolveSchema_PtrFieldError(t *testing.T) {
	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: &mockLoader{schemas: map[string]*ConfigMetadata{}},
	}

	src := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"body": {
				Type: "string",
				ContentSchema: &ConfigMetadata{
					Ref: "/",
				},
			},
		},
	}

	_, err := resolver.ResolveSchema(src)
	require.Error(t, err)
}

func TestResolver_ResolveSchema_PointerFields(t *testing.T) {
	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: NewLoader(""),
	}

	minItems := 1
	maxItems := 10
	src := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"tags": {
				Type:     "array",
				Items:    &ConfigMetadata{Type: "string"},
				MinItems: &minItems,
				MaxItems: &maxItems,
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	require.NotNil(t, result.Properties["tags"])
	require.Equal(t, "array", result.Properties["tags"].Type)
	require.NotNil(t, result.Properties["tags"].Items)
	require.Equal(t, "string", result.Properties["tags"].Items.Type)
}

func TestResolver_ResolveSchema_ContentSchema(t *testing.T) {
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

func TestResolver_ResolveSchema_PreservesCustomExtensions(t *testing.T) {
	// When a node has both a $ref and custom extensions (GoType, IsPointer,
	// IsOptional, Description, Default, Enum), the custom extensions should
	// be preserved after resolution instead of being overwritten by the
	// resolved schema's values.

	targetSchema := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"duration_type": {
				Type:        "string",
				Description: "A generic duration type",
			},
		},
	}

	ml := &mockLoader{
		schemas: map[string]*ConfigMetadata{
			"go.opentelemetry.io/collector/config/configbase.duration_type": targetSchema,
		},
	}

	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: ml,
	}

	src := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"timeout": {
				Ref:         "go.opentelemetry.io/collector/config/configbase.duration_type",
				GoType:      "time.Duration",
				IsPointer:   true,
				IsOptional:  true,
				Description: "Request timeout for the endpoint",
				Default:     defaultValue("30s"),
				Enum:        []any{"10s", "30s", "60s"},
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	require.NotNil(t, result.Properties["timeout"])

	timeout := result.Properties["timeout"]
	// GoType should be preserved from the referencing node
	require.Equal(t, "time.Duration", timeout.GoType)
	// IsPointer should be preserved
	require.True(t, timeout.IsPointer)
	// IsOptional should be preserved
	require.True(t, timeout.IsOptional)
	// Description should come from the referencing node, not the target
	require.Equal(t, "Request timeout for the endpoint", timeout.Description)
	// Default should be preserved
	require.Equal(t, "30s", timeout.Default)
	// Enum should be preserved
	require.Equal(t, []any{"10s", "30s", "60s"}, timeout.Enum)
	// Type should come from the resolved schema
	require.Equal(t, "string", timeout.Type)
}

func TestResolver_ResolveSchema_RefWithoutCustomExtensions(t *testing.T) {
	// When a node has a $ref but NO custom extensions, the resolved schema's
	// values should be used as-is (no overriding).

	targetSchema := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"base_config": {
				Type:        "object",
				Description: "Base configuration from the target schema",
				GoType:      "BaseConfig",
			},
		},
	}

	ml := &mockLoader{
		schemas: map[string]*ConfigMetadata{
			"go.opentelemetry.io/collector/config/configbase.base_config": targetSchema,
		},
	}

	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: ml,
	}

	src := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"base": {
				Ref: "go.opentelemetry.io/collector/config/configbase.base_config",
				// No custom extensions set
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)
	require.NotNil(t, result.Properties["base"])

	base := result.Properties["base"]
	// Values should come from the resolved target
	require.Equal(t, "object", base.Type)
	require.Equal(t, "Base configuration from the target schema", base.Description)
	require.Equal(t, "BaseConfig", base.GoType)
}

func TestResolver_ResolveSchema_DecoratesPropNames(t *testing.T) {
	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: NewLoader(""),
	}

	src := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"endpoint": {Type: "string"},
			"storage": {
				Type:   "string",
				GoType: "go.opentelemetry.io/collector/component.ID",
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)

	require.Equal(t, "endpoint", result.Properties["endpoint"].GoStruct.FieldName)
	require.Equal(t, "storage", result.Properties["storage"].GoStruct.FieldName)
}

func TestResolver_ResolveSchema_PreservesIntAndFloatPointers(t *testing.T) {
	// Regression test: *int and *float64 pointer fields must be copied by the resolver.
	// Previously, only *ConfigMetadata pointers were handled; all other pointer types
	// were silently dropped, causing MinLength, MaxLength, Minimum, Maximum, etc. to be nil.
	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: NewLoader(""),
	}

	minLen, maxLen := 1, 255
	minItems, maxItems := 2, 10
	minProps, maxProps := 1, 5
	minimum, maximum := 0.5, 100.0
	exclMin, exclMax := 0.0, 101.0
	multipleOf := 0.5

	src := &ConfigMetadata{
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
				MultipleOf:       &multipleOf,
			},
		},
	}

	result, err := resolver.ResolveSchema(src)
	require.NoError(t, err)

	name := result.Properties["name"]
	require.NotNil(t, name.MinLength)
	require.Equal(t, minLen, *name.MinLength)
	require.NotNil(t, name.MaxLength)
	require.Equal(t, maxLen, *name.MaxLength)

	tags := result.Properties["tags"]
	require.NotNil(t, tags.MinItems)
	require.Equal(t, minItems, *tags.MinItems)
	require.NotNil(t, tags.MaxItems)
	require.Equal(t, maxItems, *tags.MaxItems)

	meta := result.Properties["meta"]
	require.NotNil(t, meta.MinProperties)
	require.Equal(t, minProps, *meta.MinProperties)
	require.NotNil(t, meta.MaxProperties)
	require.Equal(t, maxProps, *meta.MaxProperties)

	score := result.Properties["score"]
	require.NotNil(t, score.Minimum)
	require.InEpsilon(t, minimum, *score.Minimum, 1e-9)
	require.NotNil(t, score.Maximum)
	require.InEpsilon(t, maximum, *score.Maximum, 1e-9)
	require.NotNil(t, score.ExclusiveMinimum)
	require.InDelta(t, exclMin, *score.ExclusiveMinimum, 1e-9)
	require.NotNil(t, score.ExclusiveMaximum)
	require.InEpsilon(t, exclMax, *score.ExclusiveMaximum, 1e-9)
	require.NotNil(t, score.MultipleOf)
	require.InEpsilon(t, multipleOf, *score.MultipleOf, 1e-9)
}

func TestResolver_Resolve_DoesNotMutateSourceMetadata(t *testing.T) {
	// Regression for the writer-side type split: Resolve must treat its *Metadata
	// argument as read-only. The pre-fix shallow-copy root left every nested
	// *ConfigMetadata pointer in Properties / Items / Defs / ExportedConfigs
	// aliased with the caller's tree, so the "any"-fallback in resolveRef leaked
	// GoType and Comment writes back into the source.
	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: &mockLoader{schemas: map[string]*ConfigMetadata{}},
	}

	// Node that the "any" fallback in resolveRef would mutate.
	anyFallbackNode := &ConfigMetadata{Ref: "github.com/example/unknown.type"}

	src := &Metadata{
		Config: &ConfigMetadata{
			Type: "object",
			Properties: map[string]*ConfigMetadata{
				"outer": {
					Type: "object",
					Properties: map[string]*ConfigMetadata{
						"field": anyFallbackNode,
					},
				},
				"items_field": {
					Type: "array",
					Items: &ConfigMetadata{
						Type: "object",
						Properties: map[string]*ConfigMetadata{
							"name": {Type: "string"},
						},
					},
				},
			},
			Defs: map[string]*ConfigMetadata{
				"local_alias": {
					Type:        "string",
					Description: "shared local def",
				},
			},
		},
		ExportedConfigs: map[string]*ConfigMetadata{
			"exp_config": {
				Type:        "object",
				Description: "shared exported config",
				Properties: map[string]*ConfigMetadata{
					"knob": {Type: "string"},
				},
			},
		},
	}

	snapshot := src.Clone()
	_, err := resolver.Resolve(src)
	require.NoError(t, err)

	// Direct check on the node the "any" fallback would have mutated.
	require.Empty(t, anyFallbackNode.GoType, "Resolve must not mutate caller's nested ref node (GoType)")
	require.Empty(t, anyFallbackNode.Comment, "Resolve must not mutate caller's nested ref node (Comment)")
	// Full deep-equality check across the whole source tree.
	require.Equal(t, snapshot, src, "Resolve must not mutate caller's source Metadata")
}

func TestResolver_LoadExternalRef_DoesNotMutateLoaderResult(t *testing.T) {
	// Loaders cache and reuse *Metadata across calls, so loadExternalRef must own
	// the tree it walks. Otherwise mutations performed by the recursive
	// resolveSchema (the "any"-fallback in resolveRef, future per-node writes)
	// would persist into the cached schema and contaminate subsequent loads.
	external := &ConfigMetadata{
		Type: "object",
		Defs: map[string]*ConfigMetadata{
			"config": {
				Type:        "object",
				Description: "external cached config",
				Properties: map[string]*ConfigMetadata{
					"endpoint": {Type: "string"},
					"nested": {
						Type: "object",
						Properties: map[string]*ConfigMetadata{
							"x": {Type: "integer"},
						},
					},
					"timing": {
						Type:   "string",
						Format: "duration",
					},
					"items_field": {
						Type:  "array",
						Items: &ConfigMetadata{Type: "string"},
					},
				},
			},
		},
	}
	ml := &mockLoader{
		schemas: map[string]*ConfigMetadata{
			"go.opentelemetry.io/collector/foo/bar.config": external,
		},
	}
	snapshot := external.Clone()

	resolver := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: ml,
	}
	ref := NewRef("go.opentelemetry.io/collector/foo/bar.config")

	// Call twice: a second hit on the cached schema would surface any first-call
	// mutation that escaped the deep clone.
	_, err := resolver.loadExternalRef(ref)
	require.NoError(t, err)
	_, err = resolver.loadExternalRef(ref)
	require.NoError(t, err)

	require.Equal(t, snapshot, external, "loadExternalRef must not mutate the loader's returned schema")
}

func TestConfigMetadata_Clone_DeepCopiesNestedPointers(t *testing.T) {
	// Verifies the property Resolve relies on: a cloned tree shares no pointers
	// with the source for the fields the resolver walks (Properties, Items,
	// AllOf, AdditionalProperties, ContentSchema, PatternProperties, Defs).
	src := &ConfigMetadata{
		Type:        "object",
		Description: "root",
		Properties: map[string]*ConfigMetadata{
			"a": {Type: "string"},
		},
		Items:                &ConfigMetadata{Type: "integer"},
		AllOf:                []*ConfigMetadata{{Ref: "x"}},
		AdditionalProperties: &ConfigMetadata{Type: "string"},
		ContentSchema:        &ConfigMetadata{Type: "object"},
		PatternProperties: map[string]*ConfigMetadata{
			"^p": {Type: "string"},
		},
		Defs: map[string]*ConfigMetadata{
			"d": {Type: "object"},
		},
		Required: []string{"a"},
		Enum:     []any{"x", "y"},
	}

	cloned := src.Clone()
	require.Equal(t, src, cloned)

	require.NotSame(t, src, cloned)
	require.NotSame(t, src.Properties["a"], cloned.Properties["a"])
	require.NotSame(t, src.Items, cloned.Items)
	require.NotSame(t, src.AllOf[0], cloned.AllOf[0])
	require.NotSame(t, src.AdditionalProperties, cloned.AdditionalProperties)
	require.NotSame(t, src.ContentSchema, cloned.ContentSchema)
	require.NotSame(t, src.PatternProperties["^p"], cloned.PatternProperties["^p"])
	require.NotSame(t, src.Defs["d"], cloned.Defs["d"])

	// Mutating the clone must not affect the source.
	cloned.Properties["a"].Type = "mutated"
	require.Equal(t, "string", src.Properties["a"].Type)
}

func TestMetadata_Clone_HandlesNil(t *testing.T) {
	var nilMd *Metadata
	require.Nil(t, nilMd.Clone())

	var nilCM *ConfigMetadata
	require.Nil(t, nilCM.Clone())

	empty := &Metadata{}
	cloned := empty.Clone()
	require.NotNil(t, cloned)
	require.Nil(t, cloned.Config)
	require.Nil(t, cloned.ExportedConfigs)
}

func TestJSONSchemaDoc_MarshalJSON_EmitsDefsWithMarshalJSONInner(t *testing.T) {
	// JSONSchemaDoc embeds *ConfigMetadata, which now defines its own MarshalJSON
	// (for PatternProperties / AdditionalPropertiesAllowed). Without an explicit
	// JSONSchemaDoc.MarshalJSON, the promoted inner method would marshal only the
	// inner node and silently drop the outer $defs.
	doc := &JSONSchemaDoc{
		ConfigMetadata: &ConfigMetadata{
			Schema: schemaVersion,
			Type:   "object",
			Properties: map[string]*ConfigMetadata{
				"endpoint": {Type: "string"},
			},
			// Trigger the inner MarshalJSON's "special fields" branch.
			AdditionalPropertiesAllowed: boolPtr(false),
		},
		Defs: map[string]*ConfigMetadata{
			"sample_config": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"endpoint": {Type: "string"},
				},
			},
		},
	}

	data, err := doc.ToJSON()
	require.NoError(t, err)
	require.Contains(t, string(data), `"$defs"`)
	require.Contains(t, string(data), `"sample_config"`)
	require.Contains(t, string(data), `"additionalProperties": false`)
	require.Contains(t, string(data), `"endpoint"`)
}

func TestJSONSchemaDoc_MarshalJSON_DefsOnlyWithoutInner(t *testing.T) {
	doc := &JSONSchemaDoc{
		Defs: map[string]*ConfigMetadata{
			"sample": {Type: "string"},
		},
	}
	data, err := doc.ToJSON()
	require.NoError(t, err)
	require.Contains(t, string(data), `"$defs"`)
	require.Contains(t, string(data), `"sample"`)
}

func TestJSONSchemaDoc_MarshalJSON_EmptyDoc(t *testing.T) {
	doc := &JSONSchemaDoc{}
	data, err := doc.ToJSON()
	require.NoError(t, err)
	require.Equal(t, "{}", string(data))
}

func TestResolver_Resolve_NilSource(t *testing.T) {
	r := &Resolver{loader: NewLoader("")}
	_, err := r.Resolve(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil metadata")
}

func TestResolver_Resolve_MergesExportedConfigsIntoDefs(t *testing.T) {
	r := &Resolver{
		pkgID:  "go.opentelemetry.io/collector/test/component",
		class:  "receiver",
		name:   "test",
		loader: &mockLoader{schemas: map[string]*ConfigMetadata{}},
	}
	src := &Metadata{
		Config: &ConfigMetadata{
			Type: "object",
			Properties: map[string]*ConfigMetadata{
				"shared": {Ref: "shared_config"},
			},
		},
		ExportedConfigs: map[string]*ConfigMetadata{
			"shared_config": {
				Type: "object",
				Properties: map[string]*ConfigMetadata{
					"endpoint": {Type: "string"},
				},
			},
		},
	}

	doc, err := r.Resolve(src)
	require.NoError(t, err)
	require.NotNil(t, doc.ConfigMetadata)
	require.Equal(t, "object", doc.Type)
	require.Contains(t, doc.Defs, "shared_config")
	require.Equal(t, "object", doc.Properties["shared"].Type)
	require.Contains(t, doc.Properties["shared"].Properties, "endpoint")
	// $defs must live on the outer doc, not on the inner ConfigMetadata.
	require.Nil(t, doc.ConfigMetadata.Defs)
}
