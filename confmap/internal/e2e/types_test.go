// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2etest

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
)

type TargetField string

const (
	TargetFieldInt          TargetField = "int_field"
	TargetFieldString       TargetField = "string_field"
	TargetFieldBool         TargetField = "bool_field"
	TargetFieldInlineString TargetField = "inline_string_field"
	TargetFieldSlice        TargetField = "slice_field"
)

type Test struct {
	value        string
	targetField  TargetField
	expected     any
	resolveErr   string
	unmarshalErr string
}

type targetConfig[T any] struct {
	Field T `mapstructure:"field"`
}

func NewResolver(tb testing.TB, path string) *confmap.Resolver {
	resolver, err := confmap.NewResolver(confmap.ResolverSettings{
		URIs: []string{filepath.Join("testdata", path)},
		ProviderFactories: []confmap.ProviderFactory{
			fileprovider.NewFactory(),
			envprovider.NewFactory(),
		},
		DefaultScheme: "env",
	})
	require.NoError(tb, err)
	return resolver
}

func AssertExpectedMatch[T any](t *testing.T, tt Test, conf *confmap.Conf, cfg *targetConfig[T]) {
	err := conf.Unmarshal(cfg)
	if tt.unmarshalErr != "" {
		require.ErrorContains(t, err, tt.unmarshalErr)
		return
	}
	require.NoError(t, err)
	require.Equal(t, tt.expected, cfg.Field)
}

func AssertResolvesTo(t *testing.T, resolver *confmap.Resolver, tt Test) {
	conf, err := resolver.Resolve(context.Background())
	if tt.resolveErr != "" {
		require.ErrorContains(t, err, tt.resolveErr)
		return
	}
	require.NoError(t, err)

	switch tt.targetField {
	case TargetFieldInt:
		var cfg targetConfig[int]
		AssertExpectedMatch(t, tt, conf, &cfg)
	case TargetFieldString, TargetFieldInlineString:
		var cfg targetConfig[string]
		AssertExpectedMatch(t, tt, conf, &cfg)
	case TargetFieldBool:
		var cfg targetConfig[bool]
		AssertExpectedMatch(t, tt, conf, &cfg)
	case TargetFieldSlice:
		var cfg targetConfig[[]any]
		AssertExpectedMatch(t, tt, conf, &cfg)
	default:
		t.Fatalf("unexpected target field %q", tt.targetField)
	}
}

func TestStrictTypeCasting(t *testing.T) {
	t.Setenv("ENV_VALUE", "testreceiver")

	values := []Test{
		{
			value:       "123",
			targetField: TargetFieldInt,
			expected:    123,
		},
		{
			value:       "123",
			targetField: TargetFieldString,
			expected:    "123",
		},
		{
			value:       "123",
			targetField: TargetFieldInlineString,
			expected:    "inline field with 123 expansion",
		},
		{
			value:       "0123",
			targetField: TargetFieldInt,
			expected:    83,
		},
		{
			value:       "0123",
			targetField: TargetFieldString,
			expected:    "0123",
		},
		{
			value:       "0123",
			targetField: TargetFieldInlineString,
			expected:    "inline field with 0123 expansion",
		},
		{
			value:       "0xdeadbeef",
			targetField: TargetFieldInt,
			expected:    3735928559,
		},
		{
			value:       "0xdeadbeef",
			targetField: TargetFieldString,
			expected:    "0xdeadbeef",
		},
		{
			value:       "0xdeadbeef",
			targetField: TargetFieldInlineString,
			expected:    "inline field with 0xdeadbeef expansion",
		},
		{
			value:       "\"0123\"",
			targetField: TargetFieldString,
			expected:    "\"0123\"",
		},
		{
			value:        "\"0123\"",
			targetField:  TargetFieldInt,
			unmarshalErr: "'field' expected type 'int', got unconvertible type 'string'",
		},
		{
			value:       "\"0123\"",
			targetField: TargetFieldInlineString,
			expected:    "inline field with \"0123\" expansion",
		},
		{
			value:       "!!str 0123",
			targetField: TargetFieldString,
			expected:    "!!str 0123",
		},
		{
			value:       "!!str 0123",
			targetField: TargetFieldInlineString,
			expected:    "inline field with !!str 0123 expansion",
		},
		{
			value:        "t",
			targetField:  TargetFieldBool,
			unmarshalErr: "'field' expected type 'bool', got unconvertible type 'string'",
		},
		{
			value:        "23",
			targetField:  TargetFieldBool,
			unmarshalErr: "'field' expected type 'bool', got unconvertible type 'int'",
		},
		{
			value:       "{\"field\": 123}",
			targetField: TargetFieldInlineString,
			expected:    "inline field with {\"field\": 123} expansion",
		},
		{
			value:       "1111:1111:1111:1111:1111::",
			targetField: TargetFieldInlineString,
			expected:    "inline field with 1111:1111:1111:1111:1111:: expansion",
		},
		{
			value:       "1111:1111:1111:1111:1111::",
			targetField: TargetFieldString,
			expected:    "1111:1111:1111:1111:1111::",
		},
		{
			value:       "2006-01-02T15:04:05Z07:00",
			targetField: TargetFieldString,
			expected:    "2006-01-02T15:04:05Z07:00",
		},
		{
			value:       "2006-01-02T15:04:05Z07:00",
			targetField: TargetFieldInlineString,
			expected:    "inline field with 2006-01-02T15:04:05Z07:00 expansion",
		},
		{
			value:       "2023-03-20T03:17:55.432328Z",
			targetField: TargetFieldString,
			expected:    "2023-03-20T03:17:55.432328Z",
		},
		{
			value:       "2023-03-20T03:17:55.432328Z",
			targetField: TargetFieldInlineString,
			expected:    "inline field with 2023-03-20T03:17:55.432328Z expansion",
		},
		// issue 10787
		{
			value:       "true # comment with a ${env:hello.world} reference",
			targetField: TargetFieldBool,
			expected:    true,
		},
		{
			value:        "true # comment with a ${env:hello.world} reference",
			targetField:  TargetFieldString,
			unmarshalErr: `expected type 'string', got unconvertible type 'bool'`,
		},
		{
			value:       "true # comment with a ${env:hello.world} reference",
			targetField: TargetFieldInlineString,
			resolveErr:  `environment variable "hello.world" has invalid name`,
		},
		// issue 10759
		{
			value:       `["a",`,
			targetField: TargetFieldString,
			expected:    `["a",`,
		},
		{
			value:       `["a",`,
			targetField: TargetFieldInlineString,
			expected:    `inline field with ["a", expansion`,
		},
		// issue 10799
		{
			value:       `[filelog,windowseventlog/application]`,
			targetField: TargetFieldSlice,
			expected:    []any{"filelog", "windowseventlog/application"},
		},
		{
			value:       `[filelog,windowseventlog/application]`,
			targetField: TargetFieldString,
			expected:    "[filelog,windowseventlog/application]",
		},
		{
			value:       `[filelog,windowseventlog/application]`,
			targetField: TargetFieldInlineString,
			expected:    "inline field with [filelog,windowseventlog/application] expansion",
		},
		{
			value:       "$$ENV",
			targetField: TargetFieldString,
			expected:    "$ENV",
		},
		{
			value:       "$$ENV",
			targetField: TargetFieldInlineString,
			expected:    "inline field with $ENV expansion",
		},
		{
			value:       "$${ENV}",
			targetField: TargetFieldString,
			expected:    "${ENV}",
		},
		{
			value:       "$${ENV}",
			targetField: TargetFieldInlineString,
			expected:    "inline field with ${ENV} expansion",
		},
		{
			value:       "$${env:ENV}",
			targetField: TargetFieldString,
			expected:    "${env:ENV}",
		},
		{
			value:       "$${env:ENV}",
			targetField: TargetFieldInlineString,
			expected:    "inline field with ${env:ENV} expansion",
		},
		{
			value:       `[filelog,${env:ENV_VALUE}]`,
			targetField: TargetFieldString,
			expected:    "[filelog,testreceiver]",
		},
		{
			value:       `[filelog,${ENV_VALUE}]`,
			targetField: TargetFieldString,
			expected:    "[filelog,testreceiver]",
		},
		{
			value:       `["filelog","$${env:ENV_VALUE}"]`,
			targetField: TargetFieldString,
			expected:    `["filelog","${env:ENV_VALUE}"]`,
		},
		{
			value:       `["filelog","$${ENV_VALUE}"]`,
			targetField: TargetFieldString,
			expected:    `["filelog","${ENV_VALUE}"]`,
		},
		{
			value:       `["filelog","$$ENV_VALUE"]`,
			targetField: TargetFieldString,
			expected:    `["filelog","$ENV_VALUE"]`,
		},
		{
			value:       `["filelog","$ENV_VALUE"]`,
			targetField: TargetFieldString,
			expected:    `["filelog","$ENV_VALUE"]`,
		},
	}

	for _, tt := range values {
		t.Run(tt.value+"/"+string(tt.targetField)+"/"+"direct", func(t *testing.T) {
			testFile := "types_expand.yaml"
			if tt.targetField == TargetFieldInlineString {
				testFile = "types_expand_inline.yaml"
			}
			resolver := NewResolver(t, testFile)
			t.Setenv("ENV", tt.value)
			AssertResolvesTo(t, resolver, tt)
		})

		t.Run(tt.value+"/"+string(tt.targetField)+"/"+"indirect", func(t *testing.T) {
			testFile := "types_expand.yaml"
			if tt.targetField == TargetFieldInlineString {
				testFile = "types_expand_inline.yaml"
			}

			resolver := NewResolver(t, testFile)
			t.Setenv("ENV", "${env:ENV2}")
			t.Setenv("ENV2", tt.value)
			AssertResolvesTo(t, resolver, tt)
		})
	}
}

func TestRecursiveInlineString(t *testing.T) {
	values := []Test{
		{
			value:       "123",
			targetField: TargetFieldString,
			expected:    "The value The value 123 is wrapped is wrapped",
		},
		{
			value:       "123",
			targetField: TargetFieldInlineString,
			expected:    "inline field with The value The value 123 is wrapped is wrapped expansion",
		},
		{
			value:       "opentelemetry",
			targetField: TargetFieldString,
			expected:    "The value The value opentelemetry is wrapped is wrapped",
		},
		{
			value:       "opentelemetry",
			targetField: TargetFieldInlineString,
			expected:    "inline field with The value The value opentelemetry is wrapped is wrapped expansion",
		},
	}

	for _, tt := range values {
		t.Run(tt.value+"/"+string(tt.targetField), func(t *testing.T) {
			testFile := "types_expand.yaml"
			if tt.targetField == TargetFieldInlineString {
				testFile = "types_expand_inline.yaml"
			}

			resolver := NewResolver(t, testFile)
			t.Setenv("ENV", "The value ${env:ENV2} is wrapped")
			t.Setenv("ENV2", "The value ${env:ENV3} is wrapped")
			t.Setenv("ENV3", tt.value)
			AssertResolvesTo(t, resolver, tt)
		})
	}
}

func TestRecursiveMaps(t *testing.T) {
	value := "{value: 123}"
	resolver := NewResolver(t, "types_expand.yaml")
	t.Setenv("ENV", `{env: "${env:ENV2}", inline: "inline ${env:ENV2}"}`)
	t.Setenv("ENV2", `{env2: "${env:ENV3}"}`)
	t.Setenv("ENV3", value)
	conf, err := resolver.Resolve(context.Background())
	require.NoError(t, err)

	type Value struct {
		Value int `mapstructure:"value"`
	}
	type ENV2 struct {
		Env2 Value `mapstructure:"env2"`
	}
	type ENV struct {
		Env    ENV2   `mapstructure:"env"`
		Inline string `mapstructure:"inline"`
	}
	type Target struct {
		Field ENV `mapstructure:"field"`
	}

	var cfg Target
	err = conf.Unmarshal(&cfg)
	require.NoError(t, err)
	require.Equal(t,
		Target{Field: ENV{
			Env: ENV2{
				Env2: Value{
					Value: 123,
				},
			},
			Inline: "inline {env2: \"{value: 123}\"}",
		}},
		cfg,
	)

	confStr, err := resolver.Resolve(context.Background())
	require.NoError(t, err)
	var cfgStr targetConfig[string]
	err = confStr.Unmarshal(&cfgStr)
	require.NoError(t, err)
	require.Equal(t, `{env: "{env2: "{value: 123}"}", inline: "inline {env2: "{value: 123}"}"}`,
		cfgStr.Field,
	)
}

// Test that comments with invalid ${env:...} references do not prevent configuration from loading.
func TestIssue10787(t *testing.T) {
	resolver := NewResolver(t, "issue-10787-main.yaml")
	conf, err := resolver.Resolve(context.Background())
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"exporters": map[string]any{
			"debug": map[string]any{
				"verbosity": "detailed",
			},
		},
		"processors": map[string]any{
			"memory_limiter": nil,
		},
		"receivers": map[string]any{
			"otlp": map[string]any{
				"protocols": map[string]any{
					"grpc": map[string]any{
						"endpoint": "0.0.0.0:4317",
					},
					"http": map[string]any{
						"endpoint": "0.0.0.0:4318",
					},
				},
			},
		},
		"service": map[string]any{
			"pipelines": map[string]any{
				"traces": map[string]any{
					"exporters":  []any{"debug"},
					"processors": []any{"memory_limiter"},
					"receivers":  []any{"otlp"},
				},
			},
			"telemetry": map[string]any{
				"metrics": map[string]any{
					"level": "detailed",
				},
			},
		},
	}, conf.ToStringMap(),
	)
}

func TestStructMappingIssue10787(t *testing.T) {
	resolver := NewResolver(t, "types_expand.yaml")
	t.Setenv("ENV", `# this is a comment
debug:
  verbosity: detailed`)
	conf, err := resolver.Resolve(context.Background())
	require.NoError(t, err)

	type Debug struct {
		Verbosity string `mapstructure:"verbosity"`
	}
	type Exporters struct {
		Debug Debug `mapstructure:"debug"`
	}
	type Target struct {
		Field Exporters `mapstructure:"field"`
	}

	var cfg Target
	err = conf.Unmarshal(&cfg)
	require.NoError(t, err)
	require.Equal(t,
		Target{Field: Exporters{
			Debug: Debug{
				Verbosity: "detailed",
			},
		}},
		cfg,
	)

	confStr, err := resolver.Resolve(context.Background())
	require.NoError(t, err)
	var cfgStr targetConfig[string]
	err = confStr.Unmarshal(&cfgStr)
	require.NoError(t, err)
	require.Equal(t, `# this is a comment
debug:
  verbosity: detailed`,
		cfgStr.Field,
	)
}

func TestStructMappingIssue10787_ExpandComment(t *testing.T) {
	resolver := NewResolver(t, "types_expand.yaml")
	t.Setenv("EXPAND_ME", "an expanded env var")
	t.Setenv("ENV", `# this is a comment with ${EXPAND_ME}
debug:
  verbosity: detailed`)
	conf, err := resolver.Resolve(context.Background())
	require.NoError(t, err)

	type Debug struct {
		Verbosity string `mapstructure:"verbosity"`
	}
	type Exporters struct {
		Debug Debug `mapstructure:"debug"`
	}
	type Target struct {
		Field Exporters `mapstructure:"field"`
	}

	var cfg Target
	err = conf.Unmarshal(&cfg)
	require.NoError(t, err)
	require.Equal(t,
		Target{Field: Exporters{
			Debug: Debug{
				Verbosity: "detailed",
			},
		}},
		cfg,
	)

	confStr, err := resolver.Resolve(context.Background())
	require.NoError(t, err)
	var cfgStr targetConfig[string]
	err = confStr.Unmarshal(&cfgStr)
	require.NoError(t, err)
	require.Equal(t, `# this is a comment with an expanded env var
debug:
  verbosity: detailed`,
		cfgStr.Field,
	)
}

func TestIndirectSliceEnvVar(t *testing.T) {
	// This replicates the situation in https://github.com/open-telemetry/opentelemetry-collector/issues/10799
	// where a configuration file is loaded that contains a reference to a slice of strings in an environment variable.
	t.Setenv("BASE_FOLDER", "testdata")
	t.Setenv("OTEL_LOGS_RECEIVER", "[nop, otlp]")
	t.Setenv("OTEL_LOGS_EXPORTER", "[otlp, nop]")
	resolver := NewResolver(t, "indirect-slice-env-var-main.yaml")
	conf, err := resolver.Resolve(context.Background())
	require.NoError(t, err)

	type CollectorConf struct {
		Exporters struct {
			OTLP struct {
				Endpoint string `mapstructure:"endpoint"`
			} `mapstructure:"otlp"`
			Nop struct{} `mapstructure:"nop"`
		} `mapstructure:"exporters"`
		Receivers struct {
			OTLP struct {
				Protocols struct {
					GRPC struct{} `mapstructure:"grpc"`
				} `mapstructure:"protocols"`
			} `mapstructure:"otlp"`
			Nop struct{} `mapstructure:"nop"`
		} `mapstructure:"receivers"`
		Service struct {
			Pipelines struct {
				Logs struct {
					Exporters []string `mapstructure:"exporters"`
					Receivers []string `mapstructure:"receivers"`
				} `mapstructure:"logs"`
			} `mapstructure:"pipelines"`
		} `mapstructure:"service"`
	}

	var collectorConf CollectorConf
	err = conf.Unmarshal(&collectorConf)
	require.NoError(t, err)
	assert.Equal(t, "localhost:4317", collectorConf.Exporters.OTLP.Endpoint)
	assert.Equal(t, []string{"nop", "otlp"}, collectorConf.Service.Pipelines.Logs.Receivers)
	assert.Equal(t, []string{"otlp", "nop"}, collectorConf.Service.Pipelines.Logs.Exporters)
}

func TestIssue10937_MapType(t *testing.T) {
	t.Setenv("VALUE", "1234")

	resolver := NewResolver(t, "types_map.yaml")
	conf, err := resolver.Resolve(context.Background())
	require.NoError(t, err)

	var cfg targetConfig[map[string]configopaque.String]
	err = conf.Unmarshal(&cfg)
	require.NoError(t, err)
	require.Equal(t, map[string]configopaque.String{"key": "1234"}, cfg.Field)
}

func TestIssue10937_ArrayType(t *testing.T) {
	t.Setenv("VALUE", "1234")

	resolver := NewResolver(t, "types_slice.yaml")
	conf, err := resolver.Resolve(context.Background())
	require.NoError(t, err)

	var cfgStrSlice targetConfig[[]string]
	err = conf.Unmarshal(&cfgStrSlice)
	require.NoError(t, err)
	require.Equal(t, []string{"1234"}, cfgStrSlice.Field)

	var cfgStrArray targetConfig[[1]string]
	err = conf.Unmarshal(&cfgStrArray)
	require.NoError(t, err)
	require.Equal(t, [1]string{"1234"}, cfgStrArray.Field)

	var cfgAnySlice targetConfig[[]any]
	err = conf.Unmarshal(&cfgAnySlice)
	require.NoError(t, err)
	require.Equal(t, []any{1234}, cfgAnySlice.Field)

	var cfgAnyArray targetConfig[[1]any]
	err = conf.Unmarshal(&cfgAnyArray)
	require.NoError(t, err)
	require.Equal(t, [1]any{1234}, cfgAnyArray.Field)
}

func TestIssue10937_ComplexType(t *testing.T) {
	t.Setenv("VALUE", "1234")

	resolver := NewResolver(t, "types_complex.yaml")
	conf, err := resolver.Resolve(context.Background())
	require.NoError(t, err)

	var cfgStringy targetConfig[[]map[string][]string]
	err = conf.Unmarshal(&cfgStringy)
	require.NoError(t, err)
	require.Equal(t, []map[string][]string{{"key": {"1234"}}}, cfgStringy.Field)

	var cfgNotStringy targetConfig[[]map[string][]any]
	err = conf.Unmarshal(&cfgNotStringy)
	require.NoError(t, err)
	require.Equal(t, []map[string][]any{{"key": {1234}}}, cfgNotStringy.Field)
}

func TestIssue10949_UnsetVar(t *testing.T) {
	t.Setenv("ENV", "")
	resolver := NewResolver(t, "types_expand.yaml")
	conf, err := resolver.Resolve(context.Background())
	require.NoError(t, err)

	var cfg targetConfig[int]
	err = conf.Unmarshal(&cfg)
	require.NoError(t, err)
	require.Equal(t, 0, cfg.Field)
}
