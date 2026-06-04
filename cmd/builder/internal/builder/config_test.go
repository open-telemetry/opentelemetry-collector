// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"go.opentelemetry.io/collector/cmd/builder/internal/config"
)

func TestAliases(t *testing.T) {
	// prepare
	cfg := Config{
		Extensions: []Module{
			{
				GoMod: "github.com/org/repo/impl v0.1.2",
			},
			{
				GoMod: "github.com/org/repo2/impl v0.1.2",
			},
			{
				GoMod: "github.com/org/repo3/impl v0.1.2",
			},
		},
		Receivers: []Module{
			{
				GoMod: "github.com/org/repo v0.1.2",
			},
			{
				GoMod: "github.com/org2/repo v0.1.2",
			},
			{
				GoMod: "github.com/org/repo4/impl v0.1.2",
			},
		},
		Exporters: []Module{
			{
				GoMod: "github.com/another/module v0.1.2",
			},
			{
				GoMod: "github.com/org/repo5/impl v0.1.2",
			},
		},
		Processors: []Module{
			{
				GoMod: "github.com/another/module2 v0.1.2",
			},
			{
				GoMod: "github.com/another2/module v0.1.2",
			},
		},
		Connectors: []Module{
			{
				GoMod: "github.com/another/module3 v0.1.2",
			},
			{
				GoMod: "github.com/another2/module4 v0.1.2",
			},
			{
				GoMod: "github.com/another3/module v0.1.2",
			},
		},
		Telemetry: Module{
			GoMod: "github.com/another3/module v0.1.2",
		},
	}

	// test
	err := cfg.ParseModules()
	require.NoError(t, err)

	// verify
	assert.Equal(t, "github.com/org/repo/impl v0.1.2", cfg.Extensions[0].GoMod)
	assert.Equal(t, "github.com/org/repo/impl", cfg.Extensions[0].Import)
	assert.Equal(t, "impl", cfg.Extensions[0].Name)

	assert.Equal(t, "github.com/org/repo2/impl v0.1.2", cfg.Extensions[1].GoMod)
	assert.Equal(t, "github.com/org/repo2/impl", cfg.Extensions[1].Import)
	assert.Equal(t, "impl2", cfg.Extensions[1].Name)

	assert.Equal(t, "github.com/org/repo3/impl v0.1.2", cfg.Extensions[2].GoMod)
	assert.Equal(t, "github.com/org/repo3/impl", cfg.Extensions[2].Import)
	assert.Equal(t, "impl3", cfg.Extensions[2].Name)

	assert.Equal(t, "github.com/org/repo v0.1.2", cfg.Receivers[0].GoMod)
	assert.Equal(t, "github.com/org/repo", cfg.Receivers[0].Import)
	assert.Equal(t, "repo", cfg.Receivers[0].Name)

	assert.Equal(t, "github.com/org2/repo v0.1.2", cfg.Receivers[1].GoMod)
	assert.Equal(t, "github.com/org2/repo", cfg.Receivers[1].Import)
	assert.Equal(t, "repo2", cfg.Receivers[1].Name)

	assert.Equal(t, "github.com/org/repo4/impl v0.1.2", cfg.Receivers[2].GoMod)
	assert.Equal(t, "github.com/org/repo4/impl", cfg.Receivers[2].Import)
	assert.Equal(t, "impl4", cfg.Receivers[2].Name)

	assert.Equal(t, "github.com/another/module v0.1.2", cfg.Exporters[0].GoMod)
	assert.Equal(t, "github.com/another/module", cfg.Exporters[0].Import)
	assert.Equal(t, "module", cfg.Exporters[0].Name)

	assert.Equal(t, "github.com/org/repo5/impl v0.1.2", cfg.Exporters[1].GoMod)
	assert.Equal(t, "github.com/org/repo5/impl", cfg.Exporters[1].Import)
	assert.Equal(t, "impl5", cfg.Exporters[1].Name)

	assert.Equal(t, "github.com/another/module2 v0.1.2", cfg.Processors[0].GoMod)
	assert.Equal(t, "github.com/another/module2", cfg.Processors[0].Import)
	assert.Equal(t, "module2", cfg.Processors[0].Name)

	assert.Equal(t, "github.com/another2/module v0.1.2", cfg.Processors[1].GoMod)
	assert.Equal(t, "github.com/another2/module", cfg.Processors[1].Import)
	assert.Equal(t, "module3", cfg.Processors[1].Name)

	assert.Equal(t, "github.com/another/module3 v0.1.2", cfg.Connectors[0].GoMod)
	assert.Equal(t, "github.com/another/module3", cfg.Connectors[0].Import)
	assert.Equal(t, "module32", cfg.Connectors[0].Name)

	assert.Equal(t, "github.com/another2/module4 v0.1.2", cfg.Connectors[1].GoMod)
	assert.Equal(t, "github.com/another2/module4", cfg.Connectors[1].Import)
	assert.Equal(t, "module4", cfg.Connectors[1].Name)

	assert.Equal(t, "github.com/another3/module v0.1.2", cfg.Connectors[2].GoMod)
	assert.Equal(t, "github.com/another3/module", cfg.Connectors[2].Import)
	assert.Equal(t, "module5", cfg.Connectors[2].Name)

	assert.Equal(t, "github.com/another3/module v0.1.2", cfg.Telemetry.GoMod)
	assert.Equal(t, "github.com/another3/module", cfg.Telemetry.Import)
	assert.Equal(t, "module6", cfg.Telemetry.Name)
}

func TestParseModules(t *testing.T) {
	// prepare
	cfg := Config{
		Extensions: []Module{{
			GoMod: "github.com/org/repo v0.1.2",
		}},
	}

	// test
	err := cfg.ParseModules()
	require.NoError(t, err)

	// verify
	assert.Equal(t, "github.com/org/repo v0.1.2", cfg.Extensions[0].GoMod)
	assert.Equal(t, "github.com/org/repo", cfg.Extensions[0].Import)
	assert.Equal(t, "repo", cfg.Extensions[0].Name)
}

func TestInvalidConverter(t *testing.T) {
	// Create a Config instance with invalid Converters
	config := &Config{
		ConfmapConverters: []Module{
			{
				Path: "./invalid/module/path", // Invalid module path to trigger an error
			},
		},
	}

	// Call the method and expect an error
	err := config.ParseModules()
	require.Error(t, err, "expected an error when parsing invalid modules")
}

func TestAbsoluteReplacePaths(t *testing.T) {
	// prepare
	cfg := Config{
		Distribution: Distribution{
			UseAbsoluteReplacePaths: true,
		},
		Extensions: []Module{{
			GoMod: "some-module",
			Path:  "./templates",
		}},
	}

	// test
	err := cfg.ParseModules()
	require.NoError(t, err)

	// verify
	cwd, err := os.Getwd()
	require.NoError(t, err)
	normalizedCwd := filepath.ToSlash(cwd)
	assert.True(t, strings.HasPrefix(cfg.Extensions[0].Path, normalizedCwd), "expected path %q to have prefix %q", cfg.Extensions[0].Path, normalizedCwd)
}

func TestModuleFromCore(t *testing.T) {
	// prepare
	cfg := Config{
		Extensions: []Module{ // see issue-12
			{
				Import: "go.opentelemetry.io/collector/receiver/otlpreceiver",
				GoMod:  "go.opentelemetry.io/collector v0.0.0",
			},
			{
				Import: "go.opentelemetry.io/collector/receiver/otlpreceiver",
				GoMod:  "go.opentelemetry.io/collector v0.0.0",
			},
		},
	}

	// test
	err := cfg.ParseModules()
	require.NoError(t, err)

	// verify
	assert.True(t, strings.HasPrefix(cfg.Extensions[0].Name, "otlpreceiver"))
}

func TestMissingModule(t *testing.T) {
	type invalidModuleTest struct {
		cfg Config
		err error
	}
	// prepare
	configurations := []invalidModuleTest{
		{
			cfg: Config{
				Logger: zap.NewNop(),
				ConfmapProviders: []Module{{
					Import: "invalid",
				}},
			},
			err: errMissingGoMod,
		},
		{
			cfg: Config{
				Logger: zap.NewNop(),
				Extensions: []Module{{
					Import: "invalid",
				}},
			},
			err: errMissingGoMod,
		},
		{
			cfg: Config{
				Logger: zap.NewNop(),
				Receivers: []Module{{
					Import: "invalid",
				}},
			},
			err: errMissingGoMod,
		},
		{
			cfg: Config{
				Logger: zap.NewNop(),
				Exporters: []Module{{
					Import: "invalid",
				}},
			},
			err: errMissingGoMod,
		},
		{
			cfg: Config{
				Logger: zap.NewNop(),
				Processors: []Module{{
					Import: "invalid",
				}},
			},
			err: errMissingGoMod,
		},
		{
			cfg: Config{
				Logger: zap.NewNop(),
				Connectors: []Module{{
					Import: "invalid",
				}},
			},
			err: errMissingGoMod,
		},
		{
			cfg: Config{
				Logger: zap.NewNop(),
				ConfmapConverters: []Module{{
					Import: "invalid",
				}},
			},
			err: errMissingGoMod,
		},
		{
			cfg: Config{
				Logger: zap.NewNop(),
				Telemetry: Module{
					Import: "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry",
				},
			},
			err: errMissingGoMod,
		},
	}

	for _, test := range configurations {
		assert.ErrorIs(t, test.cfg.Validate(), test.err)
	}
}

func TestNewDefaultConfig(t *testing.T) {
	cfg, err := NewDefaultConfig()
	require.NoError(t, err)
	assert.Empty(t, cfg.Telemetry.GoMod)
	require.NoError(t, cfg.Validate())
	require.NoError(t, cfg.SetGoPath())
	require.NoError(t, cfg.ParseModules())
	assert.NotEmpty(t, cfg.Telemetry.GoMod)
	assert.False(t, cfg.Distribution.DebugCompilation)
	assert.Empty(t, cfg.Distribution.BuildTags)
	assert.False(t, cfg.LDSet)
	assert.Empty(t, cfg.LDFlags)
	assert.False(t, cfg.GCSet)
	assert.Empty(t, cfg.GCFlags)
}

func TestNewBuiltinConfig(t *testing.T) {
	k := koanf.New(".")

	require.NoError(t, k.Load(config.DefaultProvider(), yaml.Parser()))

	cfg := Config{Logger: zaptest.NewLogger(t)}

	require.NoError(t, k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{Tag: "mapstructure"}))
	require.NoError(t, cfg.Validate())
	require.NoError(t, cfg.SetGoPath())
	require.NoError(t, cfg.ParseModules())

	// Unlike the config initialized in NewDefaultConfig(), we expect
	// the builtin default to be practically useful, so there must be
	// a set of modules present.
	assert.NotEmpty(t, cfg.Receivers)
	assert.NotEmpty(t, cfg.Exporters)
	assert.NotEmpty(t, cfg.Extensions)
	assert.NotEmpty(t, cfg.Processors)
}

func TestSkipGoValidation(t *testing.T) {
	cfg := Config{
		Distribution: Distribution{
			Go: "invalid/go/binary/path",
		},
		SkipCompilation: true,
		SkipGetModules:  true,
	}
	assert.NoError(t, cfg.Validate())
	assert.NoError(t, cfg.SetGoPath())
}

func TestSkipGoInitialization(t *testing.T) {
	cfg := Config{
		SkipCompilation: true,
		SkipGetModules:  true,
	}
	assert.NoError(t, cfg.Validate())
	assert.NoError(t, cfg.SetGoPath())
	assert.Empty(t, cfg.Distribution.Go)
}

func TestBuildTagConfig(t *testing.T) {
	cfg := Config{
		Distribution: Distribution{
			BuildTags: "customTag",
		},
		SkipCompilation: true,
		SkipGetModules:  true,
	}
	require.NoError(t, cfg.Validate())
	assert.Equal(t, "customTag", cfg.Distribution.BuildTags)
}

func TestDebugOptionSetConfig(t *testing.T) {
	cfg := Config{
		Distribution: Distribution{
			DebugCompilation: true,
		},
		SkipCompilation: true,
		SkipGetModules:  true,
	}
	require.NoError(t, cfg.Validate())
	assert.True(t, cfg.Distribution.DebugCompilation)
}

func TestAddsDefaultProviders(t *testing.T) {
	cfg, err := NewDefaultConfig()
	require.NoError(t, err)
	require.NoError(t, cfg.ParseModules())
	assert.Len(t, cfg.ConfmapProviders, 5)
}

func TestSkipsNilFieldValidation(t *testing.T) {
	cfg, err := NewDefaultConfig()
	require.NoError(t, err)
	cfg.ConfmapProviders = nil
	cfg.ConfmapConverters = nil
	assert.NoError(t, cfg.Validate())
}

func TestParseModulesDefaultRelativePath(t *testing.T) {
	cfg := Config{
		Distribution: Distribution{
			OutputPath: "./output",
		},
		Extensions: []Module{{
			GoMod: "some-module",
			Path:  "./templates",
		}},
	}

	err := cfg.ParseModules()
	require.NoError(t, err)

	assert.Equal(t, "../templates", cfg.Extensions[0].Path)
}

func TestParseModulesAbsoluteReplacePathsFlag(t *testing.T) {
	cfg := Config{
		Distribution: Distribution{
			OutputPath:              "./output",
			UseAbsoluteReplacePaths: true,
		},
		Extensions: []Module{{
			GoMod: "some-module",
			Path:  "./templates",
		}},
	}

	err := cfg.ParseModules()
	require.NoError(t, err)

	cwd, err := os.Getwd()
	require.NoError(t, err)
	expectedPath := filepath.ToSlash(filepath.Join(cwd, "templates"))
	assert.Equal(t, expectedPath, cfg.Extensions[0].Path)
}

func TestIsEmpty(t *testing.T) {
	for _, tt := range []struct {
		name  string
		value any
		want  bool
	}{
		{"nil", nil, true},
		{"empty map", map[string]any{}, true},
		{"non-empty map", map[string]any{"k": "v"}, false},
		{"nil slice", []string(nil), true},
		{"empty slice", []string{}, true},
		{"non-empty slice", []string{"a"}, false},
		{"empty string", "", true},
		{"non-empty string", "x", false},
		{"false", false, true},
		{"true", true, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isEmpty(tt.value))
		})
	}
}

func TestEncodeValue(t *testing.T) {
	type inner struct {
		Name string `mapstructure:"name"`
	}

	t.Run("struct", func(t *testing.T) {
		result := encodeValue(reflect.ValueOf(inner{Name: "foo"}))
		assert.Equal(t, map[string]any{"name": "foo"}, result)
	})

	t.Run("nil pointer", func(t *testing.T) {
		result := encodeValue(reflect.ValueOf((*inner)(nil)))
		assert.Nil(t, result)
	})

	t.Run("pointer to struct", func(t *testing.T) {
		result := encodeValue(reflect.ValueOf(&inner{Name: "foo"}))
		assert.Equal(t, map[string]any{"name": "foo"}, result)
	})

	t.Run("nil slice", func(t *testing.T) {
		result := encodeValue(reflect.ValueOf([]inner(nil)))
		assert.Nil(t, result)
	})

	t.Run("empty non-nil slice", func(t *testing.T) {
		result := encodeValue(reflect.ValueOf([]inner{}))
		assert.Equal(t, []any{}, result)
	})

	t.Run("slice of structs", func(t *testing.T) {
		result := encodeValue(reflect.ValueOf([]inner{{Name: "foo"}, {Name: "bar"}}))
		assert.Equal(t, []any{
			map[string]any{"name": "foo"},
			map[string]any{"name": "bar"},
		}, result)
	})

	t.Run("string", func(t *testing.T) {
		assert.Equal(t, "hello", encodeValue(reflect.ValueOf("hello")))
	})

	t.Run("bool", func(t *testing.T) {
		assert.Equal(t, true, encodeValue(reflect.ValueOf(true)))
	})
}

func TestStructToMap(t *testing.T) {
	type nested struct {
		Value string `mapstructure:"value,omitempty"`
	}
	type testStruct struct {
		NoTag   string
		Skip    string   `mapstructure:"-"`
		Present string   `mapstructure:"present"`
		Zero    string   `mapstructure:"zero"`
		Omit    string   `mapstructure:"omit,omitempty"`
		Nested  nested   `mapstructure:"nested,omitempty"`
		List    []string `mapstructure:"list,omitempty"`
	}

	t.Run("unexported fields are skipped", func(t *testing.T) {
		type withUnexported struct {
			exported   string `mapstructure:"exported"`
			unexported string `mapstructure:"unexported"`
		}
		m := structToMap(withUnexported{exported: "x", unexported: "y"})
		assert.NotContains(t, m, "exported")
		assert.NotContains(t, m, "unexported")
	})

	t.Run("field with no tag is skipped", func(t *testing.T) {
		m := structToMap(testStruct{NoTag: "x"})
		assert.NotContains(t, m, "NoTag")
	})

	t.Run("field with dash tag is skipped", func(t *testing.T) {
		m := structToMap(testStruct{Skip: "x"})
		assert.NotContains(t, m, "-")
		assert.NotContains(t, m, "Skip")
	})

	t.Run("regular field is included", func(t *testing.T) {
		m := structToMap(testStruct{Present: "hello"})
		assert.Equal(t, "hello", m["present"])
	})

	t.Run("field without omitempty is included when zero", func(t *testing.T) {
		m := structToMap(testStruct{})
		assert.Contains(t, m, "zero")
		assert.Empty(t, m["zero"])
	})

	t.Run("omitempty zero value is omitted", func(t *testing.T) {
		m := structToMap(testStruct{})
		assert.NotContains(t, m, "omit")
	})

	t.Run("omitempty non-zero value is included", func(t *testing.T) {
		m := structToMap(testStruct{Omit: "x"})
		assert.Equal(t, "x", m["omit"])
	})

	t.Run("omitempty empty nested struct is omitted", func(t *testing.T) {
		m := structToMap(testStruct{})
		assert.NotContains(t, m, "nested")
	})

	t.Run("omitempty non-empty nested struct is included", func(t *testing.T) {
		m := structToMap(testStruct{Nested: nested{Value: "x"}})
		require.Contains(t, m, "nested")
		assert.Equal(t, "x", m["nested"].(map[string]any)["value"])
	})

	t.Run("omitempty empty slice is omitted", func(t *testing.T) {
		m := structToMap(testStruct{})
		assert.NotContains(t, m, "list")
	})

	t.Run("omitempty non-empty slice is included", func(t *testing.T) {
		m := structToMap(testStruct{List: []string{"a", "b"}})
		assert.Equal(t, []any{"a", "b"}, m["list"])
	})
}

func TestMarshalYAML(t *testing.T) {
	cfg := Config{
		Distribution: Distribution{
			Name:        "myCollector",
			Description: "Custom OpenTelemetry Collector",
			OutputPath:  "./build/collector",
		},
		Exporters: []Module{
			{GoMod: "go.opentelemetry.io/collector/exporter/otlpexporter " + DefaultBetaOtelColVersion},
		},
		Receivers: []Module{
			{GoMod: "go.opentelemetry.io/collector/receiver/otlpreceiver " + DefaultBetaOtelColVersion},
		},
	}

	out, err := cfg.MarshalYAML()
	require.NoError(t, err)

	m, ok := out.(map[string]any)
	require.True(t, ok)

	dist, ok := m["dist"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "myCollector", dist["name"])
	assert.Equal(t, "Custom OpenTelemetry Collector", dist["description"])
	assert.Equal(t, "./build/collector", dist["output_path"])

	assert.Contains(t, m, "exporters")
	assert.Contains(t, m, "receivers")

	assert.NotContains(t, m, "Logger")
	assert.NotContains(t, m, "extensions")
	assert.NotContains(t, m, "processors")
	assert.NotContains(t, m, "connectors")
	assert.NotContains(t, m, "providers")
	assert.NotContains(t, m, "converters")
	assert.NotContains(t, m, "replaces")
	assert.NotContains(t, m, "excludes")
	assert.NotContains(t, m, "conf_resolver")
	assert.NotContains(t, m, "telemetry")
}
