// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestCombineCollectorSchema_LayoutAndValidation(t *testing.T) {
	t.Parallel()

	schema, err := CombineCollectorSchema(CollectorSchemaParts{
		Receivers: []CollectorComponentSchema{
			{
				Type: "otlp",
				Schema: &JSONSchema{
					Type: "object",
					Properties: map[string]*JSONSchema{
						"endpoint": {Type: "string"},
					},
					Required: []string{"endpoint"},
				},
			},
		},
		Processors: []CollectorComponentSchema{
			{
				Type: "batch",
				Schema: &JSONSchema{
					Type: "object",
					Properties: map[string]*JSONSchema{
						"timeout": {Type: "string"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
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
				Schema: &JSONSchema{
					Type: "object",
					Properties: map[string]*JSONSchema{
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

func TestCombineCollectorSchema_NullComponentBodyIsAccepted(t *testing.T) {
	t.Parallel()

	schema, err := CombineCollectorSchema(CollectorSchemaParts{
		Receivers: []CollectorComponentSchema{
			{
				Type: "otlp",
				Schema: &JSONSchema{
					Type: "object",
					Properties: map[string]*JSONSchema{
						"endpoint": {Type: "string"},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	compiled := compileSchema(t, schema)

	require.NoError(t, compiled.Validate(unmarshalJSON(t, `{
		"receivers": {"otlp": null}
	}`)))

	require.NoError(t, compiled.Validate(unmarshalYAML(t, "receivers:\n  otlp:\n")))

	require.NoError(t, compiled.Validate(unmarshalJSON(t, `{
		"receivers": {"otlp/secondary": null}
	}`)))

	require.NoError(t, compiled.Validate(unmarshalJSON(t, `{
		"receivers": {"otlp": {"endpoint": "localhost:4317"}}
	}`)))
}

func TestCombineCollectorSchema_NullBranchDoesNotSwallowTypeErrors(t *testing.T) {
	t.Parallel()

	schema, err := CombineCollectorSchema(CollectorSchemaParts{
		Receivers: []CollectorComponentSchema{
			{
				Type: "otlp",
				Schema: &JSONSchema{
					Type: "object",
					Properties: map[string]*JSONSchema{
						"endpoint": {Type: "string"},
					},
					Required: []string{"endpoint"},
				},
			},
		},
	})
	require.NoError(t, err)

	compiled := compileSchema(t, schema)

	for name, config := range map[string]string{
		"scalar body":         `{"receivers": {"otlp": 5}}`,
		"string body":         `{"receivers": {"otlp": "enabled"}}`,
		"array body":          `{"receivers": {"otlp": []}}`,
		"wrong property type": `{"receivers": {"otlp": {"endpoint": 4317}}}`,
		"missing required":    `{"receivers": {"otlp": {}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			require.Error(t, compiled.Validate(unmarshalJSON(t, config)))
		})
	}
}

func TestCombineCollectorSchema_NilSchemaIsNotWrapped(t *testing.T) {
	t.Parallel()

	schema, err := CombineCollectorSchema(CollectorSchemaParts{
		Exporters: []CollectorComponentSchema{
			{Type: "debug"},
		},
	})
	require.NoError(t, err)

	debug := schema.Properties[string(CollectorSectionExporters)].PatternProperties[collectorComponentPattern("debug")]
	require.NotNil(t, debug)
	require.Empty(t, debug.AnyOf)

	data, err := debug.MarshalJSON()
	require.NoError(t, err)
	require.JSONEq(t, `true`, string(data))

	compiled := compileSchema(t, schema)
	require.NoError(t, compiled.Validate(unmarshalJSON(t, `{"exporters": {"debug": null}}`)))
	require.NoError(t, compiled.Validate(unmarshalJSON(t, `{"exporters": {"debug": {"verbosity": "detailed"}}}`)))
}

func TestCombineCollectorSchema_NilSchemaDeprecatedKeepsMarker(t *testing.T) {
	t.Parallel()

	schema, err := CombineCollectorSchema(CollectorSchemaParts{
		Extensions: []CollectorComponentSchema{
			{Type: "health_check", DeprecatedType: "healthcheck"},
		},
	})
	require.NoError(t, err)

	extensions := schema.Properties[string(CollectorSectionExtensions)]
	deprecatedSchema := extensions.PatternProperties[collectorComponentPattern("healthcheck")]
	require.NotNil(t, deprecatedSchema)
	require.True(t, deprecatedSchema.Deprecated)
	require.Empty(t, deprecatedSchema.AnyOf)

	data, err := deprecatedSchema.MarshalJSON()
	require.NoError(t, err)
	require.JSONEq(t, `{"deprecated": true}`, string(data))

	compiled := compileSchema(t, schema)
	require.NoError(t, compiled.Validate(unmarshalJSON(t, `{"extensions": {"healthcheck": null}}`)))
	require.NoError(t, compiled.Validate(unmarshalJSON(t, `{"extensions": {"healthcheck/1": {"endpoint": "x"}}}`)))
}

func TestCombineCollectorSchema_ServiceSection(t *testing.T) {
	t.Parallel()

	schema, err := CombineCollectorSchema(CollectorSchemaParts{
		Receivers:  []CollectorComponentSchema{{Type: "otlp"}},
		Processors: []CollectorComponentSchema{{Type: "batch"}},
		Exporters:  []CollectorComponentSchema{{Type: "debug"}},
		Connectors: []CollectorComponentSchema{{Type: "forward"}},
		Extensions: []CollectorComponentSchema{{Type: "health_check", DeprecatedType: "healthcheck"}},
	})
	require.NoError(t, err)

	compiled := compileSchema(t, schema)

	require.NoError(t, compiled.Validate(unmarshalYAML(t, `
extensions:
  health_check:
service:
  extensions: [health_check, healthcheck/legacy]
  telemetry:
    logs:
      level: debug
  pipelines:
    traces:
      receivers: [otlp, otlp/secondary]
      processors: [batch]
      exporters: [forward]
    traces/downstream:
      receivers: [forward]
      exporters: [debug]
    metrics:
      receivers: [otlp]
      exporters: [debug]
    logs/a:
      receivers: [otlp]
      exporters: [debug]
    profiles:
      receivers: [otlp]
      exporters: [debug]
`)))

	for name, config := range map[string]string{
		"unknown receiver":          `{"service": {"pipelines": {"traces": {"receivers": ["jaeger"], "exporters": ["debug"]}}}}`,
		"unknown processor":         `{"service": {"pipelines": {"traces": {"receivers": ["otlp"], "processors": ["filter"], "exporters": ["debug"]}}}}`,
		"unknown exporter":          `{"service": {"pipelines": {"traces": {"receivers": ["otlp"], "exporters": ["otlphttp"]}}}}`,
		"connector as processor":    `{"service": {"pipelines": {"traces": {"receivers": ["otlp"], "processors": ["forward"], "exporters": ["debug"]}}}}`,
		"exporter as receiver":      `{"service": {"pipelines": {"traces": {"receivers": ["debug"], "exporters": ["debug"]}}}}`,
		"unknown extension":         `{"service": {"extensions": ["zpages"]}}`,
		"unknown signal":            `{"service": {"pipelines": {"spans": {"receivers": ["otlp"], "exporters": ["debug"]}}}}`,
		"missing receivers":         `{"service": {"pipelines": {"traces": {"exporters": ["debug"]}}}}`,
		"missing exporters":         `{"service": {"pipelines": {"traces": {"receivers": ["otlp"]}}}}`,
		"empty receivers":           `{"service": {"pipelines": {"traces": {"receivers": [], "exporters": ["debug"]}}}}`,
		"empty exporters":           `{"service": {"pipelines": {"traces": {"receivers": ["otlp"], "exporters": []}}}}`,
		"duplicate processor":       `{"service": {"pipelines": {"traces": {"receivers": ["otlp"], "processors": ["batch", "batch"], "exporters": ["debug"]}}}}`,
		"unknown pipeline key":      `{"service": {"pipelines": {"traces": {"receivers": ["otlp"], "exporters": ["debug"], "connectors": []}}}}`,
		"unknown service key":       `{"service": {"pipeline": {}}}`,
		"non-string reference":      `{"service": {"pipelines": {"traces": {"receivers": [1], "exporters": ["debug"]}}}}`,
		"empty instance name":       `{"service": {"pipelines": {"traces": {"receivers": ["otlp/"], "exporters": ["debug"]}}}}`,
		"type prefix without slash": `{"service": {"pipelines": {"traces": {"receivers": ["otlpx"], "exporters": ["debug"]}}}}`,
		"extensions not an array":   `{"service": {"extensions": "health_check"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Error(t, compiled.Validate(unmarshalJSON(t, config)))
		})
	}
}

func TestCombineCollectorSchema_ServiceSectionWithoutComponents(t *testing.T) {
	t.Parallel()

	schema, err := CombineCollectorSchema(CollectorSchemaParts{})
	require.NoError(t, err)

	compiled := compileSchema(t, schema)

	require.NoError(t, compiled.Validate(unmarshalJSON(t, `{"service": {}}`)))
	require.NoError(t, compiled.Validate(unmarshalJSON(t, `{"service": {"extensions": [], "pipelines": {}}}`)))
	require.Error(t, compiled.Validate(unmarshalJSON(t, `{"service": {"extensions": ["zpages"]}}`)))
	require.Error(t, compiled.Validate(unmarshalJSON(t, `{"service": {"pipelines": {"traces": {"receivers": ["otlp"], "exporters": ["debug"]}}}}`)))
}

func TestCombineCollectorSchema_ServiceSectionLayout(t *testing.T) {
	t.Parallel()

	schema, err := CombineCollectorSchema(CollectorSchemaParts{
		Receivers:  []CollectorComponentSchema{{Type: "otlp"}},
		Exporters:  []CollectorComponentSchema{{Type: "debug"}},
		Connectors: []CollectorComponentSchema{{Type: "forward"}},
	})
	require.NoError(t, err)

	service := schema.Properties[string(CollectorSectionService)]
	pipeline := service.Properties["pipelines"].PatternProperties[collectorIdentifierPattern(defaultPipelineSignals)]
	require.NotNil(t, pipeline)
	require.Equal(t, "^(?:forward|otlp)(?:/.+)?$", pipeline.Properties["receivers"].Items.Pattern)
	require.Equal(t, "^(?:debug|forward)(?:/.+)?$", pipeline.Properties["exporters"].Items.Pattern)
	require.Equal(t, &JSONSchema{Not: &JSONSchema{}}, pipeline.Properties["processors"].Items)

	data, err := service.Properties["telemetry"].MarshalJSON()
	require.NoError(t, err)
	require.NotContains(t, string(data), "properties")
}

func TestCollectorIdentifierPattern(t *testing.T) {
	t.Parallel()

	t.Run("sorted and deduplicated", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "^(?:debug|otlp)(?:/.+)?$", collectorIdentifierPattern([]string{"otlp", "debug", "otlp"}))
		require.Equal(t, collectorIdentifierPattern([]string{"a", "b"}), collectorIdentifierPattern([]string{"b", "a"}))
	})

	t.Run("does not mutate input", func(t *testing.T) {
		t.Parallel()
		types := []string{"b", "a"}
		collectorIdentifierPattern(types)
		require.Equal(t, []string{"b", "a"}, types)
	})

	t.Run("empty matches nothing", func(t *testing.T) {
		t.Parallel()
		pattern := regexp.MustCompile(collectorIdentifierPattern(nil))
		for _, candidate := range []string{"", "/", "/name", "otlp", "otlp/name"} {
			require.False(t, pattern.MatchString(candidate), candidate)
		}
	})
}

func compileSchema(t *testing.T, schema *JSONSchema) *jsonschema.Schema {
	t.Helper()

	data, err := schema.MarshalJSON()
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

func unmarshalYAML(t *testing.T, data string) any {
	t.Helper()

	var value any
	require.NoError(t, yaml.Unmarshal([]byte(data), &value))

	return value
}
