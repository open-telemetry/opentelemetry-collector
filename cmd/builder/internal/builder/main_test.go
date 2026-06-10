// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/mod/modfile"

	"go.opentelemetry.io/collector/internal/schemagen"
)

const (
	goModTestFile = `// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
module go.opentelemetry.io/collector/cmd/builder/internal/tester
go 1.20
require (
	go.opentelemetry.io/collector/component v0.96.0
	go.opentelemetry.io/collector/connector v0.94.1
	go.opentelemetry.io/collector/exporter v0.94.1
	go.opentelemetry.io/collector/extension v0.94.1
	go.opentelemetry.io/collector/otelcol v0.94.1
	go.opentelemetry.io/collector/processor v0.94.1
	go.opentelemetry.io/collector/receiver v0.94.1
	go.opentelemetry.io/collector v0.96.0
)`
	modulePrefix = "go.opentelemetry.io/collector"
)

var replaceModules = []string{
	"",
	"/component",
	"/component/componentstatus",
	"/component/componenttest",
	"/client",
	"/config/configauth",
	"/config/configcompression",
	"/config/configgrpc",
	"/config/confighttp",
	"/config/configmiddleware",
	"/config/confignet",
	"/config/configopaque",
	"/config/configoptional",
	"/config/configretry",
	"/config/configtelemetry",
	"/config/configtls",
	"/confmap",
	"/confmap/xconfmap",
	"/confmap/provider/envprovider",
	"/confmap/provider/fileprovider",
	"/confmap/provider/httpprovider",
	"/confmap/provider/httpsprovider",
	"/confmap/provider/yamlprovider",
	"/consumer",
	"/consumer/consumererror",
	"/consumer/consumererror/xconsumererror",
	"/consumer/xconsumer",
	"/consumer/consumertest",
	"/connector",
	"/connector/connectortest",
	"/connector/xconnector",
	"/exporter",
	"/exporter/debugexporter",
	"/exporter/xexporter",
	"/exporter/exportertest",
	"/exporter/exporterhelper",
	"/exporter/exporterhelper/xexporterhelper",
	"/exporter/nopexporter",
	"/exporter/otlpexporter",
	"/exporter/otlphttpexporter",
	"/extension",
	"/extension/extensionauth",
	"/extension/extensionauth/extensionauthtest",
	"/extension/extensioncapabilities",
	"/extension/extensionmiddleware",
	"/extension/extensionmiddleware/extensionmiddlewaretest",
	"/extension/extensiontest",
	"/extension/zpagesextension",
	"/extension/xextension",
	"/featuregate",
	"/internal/componentalias",
	"/internal/memorylimiter",
	"/internal/fanoutconsumer",
	"/internal/sharedcomponent",
	"/internal/telemetry",
	"/internal/testutil",
	"/otelcol",
	"/pdata",
	"/pdata/testdata",
	"/pdata/pprofile",
	"/pdata/xpdata",
	"/pipeline",
	"/pipeline/xpipeline",
	"/processor",
	"/processor/processortest",
	"/processor/batchprocessor",
	"/processor/memorylimiterprocessor",
	"/processor/processorhelper",
	"/processor/processorhelper/xprocessorhelper",
	"/processor/xprocessor",
	"/receiver",
	"/receiver/nopreceiver",
	"/receiver/otlpreceiver",
	"/receiver/receivertest",
	"/receiver/receiverhelper",
	"/receiver/xreceiver",
	"/service",
	"/service/hostcapabilities",
	"/service/telemetry/telemetrytest",
}

func newTestConfig(tb testing.TB) *Config {
	cfg, err := NewDefaultConfig()
	require.NoError(tb, err)
	cfg.downloadModules.wait = 0
	cfg.downloadModules.numRetries = 1
	return cfg
}

func newInitializedConfig(t *testing.T) *Config {
	cfg := newTestConfig(t)
	// Validate and ParseModules will be called before the config is
	// given to Generate.
	assert.NoError(t, cfg.Validate())
	assert.NoError(t, cfg.ParseModules())

	return cfg
}

func TestGenerateDefault(t *testing.T) {
	require.NoError(t, Generate(newInitializedConfig(t)))
}

func TestGenerateWritesEmbeddedSchemaStub(t *testing.T) {
	cfg := newInitializedConfig(t)
	cfg.Distribution.OutputPath = t.TempDir()

	require.NoError(t, Generate(cfg))

	schemaSource, err := os.ReadFile(filepath.Join(cfg.Distribution.OutputPath, embeddedSchemaFileName))
	require.NoError(t, err)
	require.Contains(t, string(schemaSource), "var embeddedConfigSchema = []byte(nil)")

	mainSource, err := os.ReadFile(filepath.Join(cfg.Distribution.OutputPath, "main.go"))
	require.NoError(t, err)
	require.Contains(t, string(mainSource), "ConfigSchema: embeddedConfigSchema,")
}

func TestCompileWrapper(t *testing.T) {
	cfg := newInitializedConfig(t)
	cfg.SkipCompilation = true

	require.NoError(t, Compile(cfg))
}

func TestGetModulesWrapper(t *testing.T) {
	cfg := newInitializedConfig(t)
	cfg.SkipGetModules = true

	require.NoError(t, GetModules(cfg))
}

func TestGenerateInvalidOutputPath(t *testing.T) {
	cfg := newInitializedConfig(t)
	cfg.Distribution.OutputPath = ":/invalid"
	err := Generate(cfg)
	require.ErrorContains(t, err, "failed to create output path")
}

func TestGenerateEmbeddedSchemaStubWriteError(t *testing.T) {
	cfg := newInitializedConfig(t)
	cfg.Distribution.OutputPath = t.TempDir()

	deps := defaultBuilderDeps()
	deps.writeEmbeddedSchemaSourceFile = func(string, []byte) error { return assert.AnError }

	err := generateWithDeps(cfg, deps)
	require.ErrorContains(t, err, `failed to generate source file "schema.go"`)
}

func TestOutputBinaryName(t *testing.T) {
	for _, tt := range []struct {
		name string
		goos string
		want string
	}{
		{
			name: "on windows",
			goos: "windows",
			want: "otelcorecol.exe",
		},
		{
			name: "on other OSes",
			goos: "linux",
			want: "otelcorecol",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GOOS", tt.goos)
			assert.Equal(t, tt.want, outputBinaryName("otelcorecol"))
			assert.Equal(t, "otelcorecol.exe", outputBinaryName("otelcorecol.exe"))
		})
	}
}

func TestVersioning(t *testing.T) {
	replaces := generateReplaces()
	tests := []struct {
		name        string
		cfgBuilder  func() *Config
		expectedErr error
	}{
		{
			name: "defaults",
			cfgBuilder: func() *Config {
				cfg := newTestConfig(t)
				cfg.Distribution.Go = "go"
				cfg.Replaces = append(cfg.Replaces, replaces...)
				return cfg
			},
			expectedErr: nil,
		},
		{
			name: "only gomod file, skip generate",
			cfgBuilder: func() *Config {
				cfg := newTestConfig(t)
				tempDir := t.TempDir()
				err := makeModule(tempDir, []byte(goModTestFile))
				require.NoError(t, err)
				cfg.Distribution.OutputPath = tempDir
				cfg.SkipGenerate = true
				cfg.Distribution.Go = "go"
				return cfg
			},
			expectedErr: ErrDepNotFound,
		},
		{
			name: "old component version",
			cfgBuilder: func() *Config {
				cfg := newTestConfig(t)
				cfg.Distribution.Go = "go"
				cfg.Exporters = []Module{
					{
						GoMod: "go.opentelemetry.io/collector/exporter/otlpexporter v0.112.0",
					},
				}
				cfg.ConfmapProviders = []Module{}
				cfg.Replaces = append(cfg.Replaces, replaces...)
				return cfg
			},
			expectedErr: nil,
		},
		{
			name: "old component version without strict mode",
			cfgBuilder: func() *Config {
				cfg := newTestConfig(t)
				cfg.Distribution.Go = "go"
				cfg.SkipStrictVersioning = true
				cfg.Exporters = []Module{
					{
						GoMod: "go.opentelemetry.io/collector/exporter/otlpexporter v0.112.0",
					},
				}
				cfg.ConfmapProviders = []Module{}
				cfg.Replaces = append(cfg.Replaces, replaces...)
				return cfg
			},
			expectedErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// X25519 curves are not supported when GODEBUG=fips140=only is set, so we
			// detect if it is and conditionally also add the tlsmklem=0 flag to disable
			// these curves. See: https://pkg.go.dev/crypto/tls#Config.CurvePreferences
			if strings.Contains(os.Getenv("GODEBUG"), "fips140=only") {
				t.Setenv("GODEBUG", os.Getenv("GODEBUG")+",tlsmlkem=0")
			}

			cfg := tt.cfgBuilder()
			require.NoError(t, cfg.Validate())
			require.NoError(t, cfg.ParseModules())
			err := GenerateAndCompile(cfg)
			require.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

func TestSkipGenerate(t *testing.T) {
	cfg := newInitializedConfig(t)
	cfg.Distribution.OutputPath = t.TempDir()
	cfg.SkipGenerate = true
	err := Generate(cfg)
	require.NoError(t, err)
	outputFile, err := os.Open(cfg.Distribution.OutputPath)
	defer func() {
		require.NoError(t, outputFile.Close())
	}()
	require.NoError(t, err)
	_, err = outputFile.Readdirnames(1)
	require.ErrorIs(t, err, io.EOF, "skip generate should leave output directory empty")
}

func TestSkipGetModules(t *testing.T) {
	cfg := newInitializedConfig(t)
	cfg.Distribution.OutputPath = t.TempDir()
	cfg.SkipGetModules = true
	err := Generate(cfg)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(cfg.Distribution.OutputPath, "go.mod"))
	require.ErrorIs(t, err, os.ErrNotExist, "go.mod should not be generated when skip-get-modules is enabled")

	_, err = os.Stat(filepath.Join(cfg.Distribution.OutputPath, "main.go"))
	require.NoError(t, err, "main.go should be generated even when skip-get-modules is enabled")
}

func TestGenerateAndCompile(t *testing.T) {
	replaces := generateReplaces()
	testCases := []struct {
		name       string
		cfgBuilder func(t *testing.T) *Config
	}{
		{
			name: "Default Configuration Compilation",
			cfgBuilder: func(t *testing.T) *Config {
				cfg := newTestConfig(t)
				cfg.Distribution.OutputPath = t.TempDir()
				cfg.Replaces = append(cfg.Replaces, replaces...)
				return cfg
			},
		},
		{
			name: "LDFlags Compilation",
			cfgBuilder: func(t *testing.T) *Config {
				cfg := newTestConfig(t)
				cfg.Distribution.OutputPath = t.TempDir()
				cfg.Replaces = append(cfg.Replaces, replaces...)
				cfg.LDSet = true
				cfg.LDFlags = `-X "test.gitVersion=0743dc6c6411272b98494a9b32a63378e84c34da" -X "test.gitTag=local-testing" -X "test.goVersion=go version go1.20.7 darwin/amd64"`
				return cfg
			},
		},
		{
			name: "GCFlags Compilation",
			cfgBuilder: func(t *testing.T) *Config {
				cfg := newTestConfig(t)
				cfg.Distribution.OutputPath = t.TempDir()
				cfg.Replaces = append(cfg.Replaces, replaces...)
				cfg.GCSet = true
				cfg.GCFlags = `all=-N -l`
				return cfg
			},
		},
		{
			name: "Build Tags Compilation",
			cfgBuilder: func(t *testing.T) *Config {
				cfg := newTestConfig(t)
				cfg.Distribution.OutputPath = t.TempDir()
				cfg.Replaces = append(cfg.Replaces, replaces...)
				cfg.Distribution.BuildTags = "customTag"
				return cfg
			},
		},
		{
			name: "Debug Compilation",
			cfgBuilder: func(t *testing.T) *Config {
				cfg := newTestConfig(t)
				cfg.Distribution.OutputPath = t.TempDir()
				cfg.Replaces = append(cfg.Replaces, replaces...)
				cfg.Logger = zap.NewNop()
				cfg.Distribution.DebugCompilation = true
				return cfg
			},
		},
		{
			name: "No providers",
			cfgBuilder: func(t *testing.T) *Config {
				cfg := newTestConfig(t)
				cfg.Distribution.OutputPath = t.TempDir()
				cfg.Replaces = append(cfg.Replaces, replaces...)
				cfg.ConfmapProviders = []Module{}
				return cfg
			},
		},
		{
			name: "With confmap factories",
			cfgBuilder: func(t *testing.T) *Config {
				cfg := newTestConfig(t)
				cfg.Distribution.OutputPath = t.TempDir()
				cfg.Replaces = append(cfg.Replaces, replaces...)
				cfg.SkipStrictVersioning = true
				return cfg
			},
		},
		{
			name: "ConfResolverDefaultURIScheme set",
			cfgBuilder: func(t *testing.T) *Config {
				cfg := newTestConfig(t)
				cfg.ConfResolver = ConfResolver{
					DefaultURIScheme: "env",
				}
				cfg.Distribution.OutputPath = t.TempDir()
				cfg.Replaces = append(cfg.Replaces, replaces...)
				return cfg
			},
		},
		{
			name: "CGoEnabled set to true",
			cfgBuilder: func(t *testing.T) *Config {
				cfg := newTestConfig(t)
				cfg.Distribution.OutputPath = t.TempDir()
				cfg.Replaces = append(cfg.Replaces, replaces...)
				cfg.Distribution.CGoEnabled = true
				return cfg
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfgBuilder(t)
			assert.NoError(t, cfg.Validate())
			assert.NoError(t, cfg.SetGoPath())
			assert.NoError(t, cfg.ParseModules())
			require.NoError(t, GenerateAndCompile(cfg))
		})
	}
}

func TestGetModulesErrorPaths(t *testing.T) {
	t.Run("skip get modules skips schema generation", func(t *testing.T) {
		cfg := newInitializedConfig(t)
		cfg.SkipGetModules = true
		cfg.Distribution.OutputPath = t.TempDir()

		deps := defaultBuilderDeps()
		deps.writeEmbeddedSchemaSourceFile = func(string, []byte) error {
			return assert.AnError
		}

		require.NoError(t, getModulesWithDeps(cfg, deps))
	})

	t.Run("mod tidy error", func(t *testing.T) {
		cfg := newInitializedConfig(t)
		cfg.Distribution.OutputPath = t.TempDir()

		deps := defaultBuilderDeps()
		deps.runGoCommand = func(*Config, ...string) ([]byte, error) { return nil, assert.AnError }

		err := getModulesWithDeps(cfg, deps)
		require.ErrorContains(t, err, "failed to update go.mod")
	})

	t.Run("skip strict download error", func(t *testing.T) {
		cfg := newInitializedConfig(t)
		cfg.SkipStrictVersioning = true
		cfg.Distribution.OutputPath = t.TempDir()

		deps := defaultBuilderDeps()
		deps.runGoCommand = func(*Config, ...string) ([]byte, error) { return nil, nil }
		deps.downloadModules = func(*Config) error { return assert.AnError }

		err := getModulesWithDeps(cfg, deps)
		require.ErrorIs(t, err, assert.AnError)
	})

	t.Run("read go.mod error", func(t *testing.T) {
		cfg := newInitializedConfig(t)
		cfg.Distribution.OutputPath = t.TempDir()

		deps := defaultBuilderDeps()
		deps.runGoCommand = func(*Config, ...string) ([]byte, error) { return nil, nil }
		deps.readGoModFile = func(*Config) (string, map[string]string, error) {
			return "", nil, assert.AnError
		}

		err := getModulesWithDeps(cfg, deps)
		require.ErrorIs(t, err, assert.AnError)
	})

	t.Run("strict download error", func(t *testing.T) {
		cfg := newInitializedConfig(t)
		cfg.Distribution.OutputPath = t.TempDir()
		dependencyVersions := map[string]string{
			otelcolPath: DefaultBetaOtelColVersion,
		}
		for _, mod := range cfg.allComponents() {
			module, version, _ := strings.Cut(mod.GoMod, " ")
			dependencyVersions[module] = version
		}

		deps := defaultBuilderDeps()
		deps.runGoCommand = func(*Config, ...string) ([]byte, error) { return nil, nil }
		deps.readGoModFile = func(*Config) (string, map[string]string, error) {
			return "example.com/dist", dependencyVersions, nil
		}
		deps.downloadModules = func(*Config) error { return assert.AnError }

		err := getModulesWithDeps(cfg, deps)
		require.ErrorIs(t, err, assert.AnError)
	})
}

func TestBuildEmbeddedSchema(t *testing.T) {
	t.Parallel()

	receiverDir := writeSchemaModule(t, componentSchemaMetadata{Type: "otlp", DeprecatedType: "otlp_legacy"}, map[string]any{
		"type": "object",
		"properties": map[string]any{
			"endpoint": map[string]any{"type": "string"},
		},
		"required": []string{"endpoint"},
	})
	exporterDir := writeSchemaModule(t, componentSchemaMetadata{Type: "debug"}, map[string]any{
		"type": "object",
		"patternProperties": map[string]any{
			"^verbosity$": map[string]any{"type": "string"},
		},
	})
	processorDir := writeSchemaModule(t, componentSchemaMetadata{Type: "batch"}, map[string]any{
		"type": "object",
	})
	connectorDir := writeSchemaModule(t, componentSchemaMetadata{Type: "forward"}, map[string]any{
		"type": "object",
	})
	extensionDir := writeSchemaModule(t, componentSchemaMetadata{Type: "health_check"}, map[string]any{
		"type": "object",
	})
	cfg := newTestConfig(t)
	cfg.Receivers = []Module{{GoMod: "example.com/receiver v0.0.1"}}
	cfg.Exporters = []Module{{GoMod: "example.com/exporter v0.0.1"}}
	cfg.Processors = []Module{{GoMod: "example.com/processor v0.0.1"}}
	cfg.Connectors = []Module{{GoMod: "example.com/connector v0.0.1"}}
	cfg.Extensions = []Module{{GoMod: "example.com/extension v0.0.1"}}

	schemaBytes, err := buildEmbeddedSchema(cfg, func(modulePath string) (string, error) {
		switch modulePath {
		case "example.com/receiver":
			return receiverDir, nil
		case "example.com/exporter":
			return exporterDir, nil
		case "example.com/processor":
			return processorDir, nil
		case "example.com/connector":
			return connectorDir, nil
		case "example.com/extension":
			return extensionDir, nil
		default:
			return "", fmt.Errorf("unexpected module path %q", modulePath)
		}
	})
	require.NoError(t, err)
	require.NotNil(t, schemaBytes)

	var schema map[string]any
	require.NoError(t, json.Unmarshal(schemaBytes, &schema))

	properties := schema["properties"].(map[string]any)

	receivers := properties["receivers"].(map[string]any)
	receiverPatterns := receivers["patternProperties"].(map[string]any)
	require.Contains(t, receiverPatterns, "^otlp(?:/.+)?$")
	require.Contains(t, receiverPatterns, "^otlp_legacy(?:/.+)?$")
	require.Equal(t, true, receiverPatterns["^otlp_legacy(?:/.+)?$"].(map[string]any)["deprecated"])

	exporters := properties["exporters"].(map[string]any)
	exporterPatterns := exporters["patternProperties"].(map[string]any)
	debugSchema := exporterPatterns["^debug(?:/.+)?$"].(map[string]any)
	require.Contains(t, debugSchema, "patternProperties")

	processors := properties["processors"].(map[string]any)
	require.Contains(t, processors["patternProperties"].(map[string]any), "^batch(?:/.+)?$")

	connectors := properties["connectors"].(map[string]any)
	require.Contains(t, connectors["patternProperties"].(map[string]any), "^forward(?:/.+)?$")

	extensions := properties["extensions"].(map[string]any)
	require.Contains(t, extensions["patternProperties"].(map[string]any), "^health_check(?:/.+)?$")
}

func TestBuildEmbeddedSchemaWithoutAvailableSchemas(t *testing.T) {
	t.Parallel()

	componentDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(componentDir, "metadata.yaml"), []byte("type: otlp\n"), 0o600))

	cfg := newTestConfig(t)
	cfg.Receivers = []Module{{GoMod: "example.com/receiver v0.0.1"}}

	schemaBytes, err := buildEmbeddedSchema(cfg, func(modulePath string) (string, error) {
		require.Equal(t, "example.com/receiver", modulePath)
		return componentDir, nil
	})
	require.NoError(t, err)
	require.Nil(t, schemaBytes)
}

func TestWriteEmbeddedSchemaSourceFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, writeEmbeddedSchemaSourceFile(dir, []byte(`{"type":"object"}`)))

	// #nosec G304 -- dir is a test temporary directory
	schemaSource, err := os.ReadFile(filepath.Join(dir, embeddedSchemaFileName))
	require.NoError(t, err)
	require.Contains(t, string(schemaSource), `var embeddedConfigSchema = []byte("{\"type\":\"object\"}")`)
}

func TestWriteEmbeddedSchemaSourceFileError(t *testing.T) {
	t.Parallel()

	err := writeEmbeddedSchemaSourceFile(filepath.Join(t.TempDir(), "missing"), []byte(`{}`))
	require.Error(t, err)
}

func TestLoadCollectorComponentSchemaErrors(t *testing.T) {
	t.Parallel()

	t.Run("missing metadata", func(t *testing.T) {
		_, _, err := loadCollectorComponentSchema(t.TempDir())
		require.ErrorContains(t, err, "failed to read metadata.yaml")
	})

	t.Run("invalid metadata", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "metadata.yaml"), []byte("{"), 0o600))
		_, _, err := loadCollectorComponentSchema(dir)
		require.ErrorContains(t, err, "failed to parse metadata.yaml")
	})

	t.Run("missing type", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "metadata.yaml"), []byte("display_name: test\n"), 0o600))
		_, _, err := loadCollectorComponentSchema(dir)
		require.ErrorContains(t, err, "metadata.yaml missing component type")
	})

	t.Run("invalid schema json", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "metadata.yaml"), []byte("type: otlp\n"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "config.schema.json"), []byte("{"), 0o600))
		_, _, err := loadCollectorComponentSchema(dir)
		require.ErrorContains(t, err, "failed to parse config.schema.json")
	})

	t.Run("yaml fallback", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "metadata.yaml"), []byte("type: otlp\n"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "config.schema.yaml"), []byte("type: object\nproperties:\n  endpoint:\n    type: string\n"), 0o600))
		schema, ok, err := loadCollectorComponentSchema(dir)
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, "object", schema.Schema.Type)
		require.Equal(t, "string", schema.Schema.Properties["endpoint"].Type)
	})

	t.Run("yml fallback", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "metadata.yaml"), []byte("type: otlp\n"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "config.schema.yml"), []byte("type: object\n"), 0o600))
		schema, ok, err := loadCollectorComponentSchema(dir)
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, "object", schema.Schema.Type)
	})

	t.Run("schema read error", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "metadata.yaml"), []byte("type: otlp\n"), 0o600))
		require.NoError(t, os.Mkdir(filepath.Join(dir, "config.schema.json"), 0o700))
		_, _, err := loadCollectorComponentSchema(dir)
		require.ErrorContains(t, err, "failed to read config.schema.json")
	})
}

func TestParseComponentConfigSchemaError(t *testing.T) {
	t.Parallel()

	_, err := parseComponentConfigSchema([]byte("{"))
	require.Error(t, err)
}

func TestBuildEmbeddedSchemaResolveModuleDirError(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)
	cfg.Receivers = []Module{{GoMod: "example.com/receiver v0.0.1"}}

	_, err := buildEmbeddedSchema(cfg, func(modulePath string) (string, error) {
		require.Equal(t, "example.com/receiver", modulePath)
		return "", assert.AnError
	})
	require.ErrorContains(t, err, `failed to resolve module directory for "example.com/receiver"`)
}

func TestBuildEmbeddedSchemaLoadCollectorComponentSchemaError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := newTestConfig(t)
	cfg.Receivers = []Module{{GoMod: "example.com/receiver v0.0.1"}}

	_, err := buildEmbeddedSchema(cfg, func(string) (string, error) {
		return dir, nil
	})
	require.ErrorContains(t, err, `failed to load component schema for "example.com/receiver"`)
}

func TestBuildEmbeddedSchemaUnsupportedSection(t *testing.T) {
	componentDir := writeSchemaModule(t, componentSchemaMetadata{Type: "otlp"}, map[string]any{
		"type": "object",
	})

	cfg := newTestConfig(t)
	cfg.Telemetry = Module{GoMod: "example.com/telemetry v0.0.1"}

	deps := defaultBuilderDeps()
	deps.selectedComponentSchemaRefs = func(*Config) []componentSchemaRef {
		return []componentSchemaRef{{
			section: schemagen.CollectorSectionService,
			module:  cfg.Telemetry,
		}}
	}

	_, err := buildEmbeddedSchemaWithDeps(cfg, deps, func(string) (string, error) {
		return componentDir, nil
	})
	require.ErrorContains(t, err, `unsupported collector section "service"`)
}

func TestBuildEmbeddedSchemaCombineError(t *testing.T) {
	t.Parallel()

	firstDir := writeSchemaModule(t, componentSchemaMetadata{Type: "otlp"}, map[string]any{"type": "object"})
	secondDir := writeSchemaModule(t, componentSchemaMetadata{Type: "otlp"}, map[string]any{"type": "object"})
	cfg := newTestConfig(t)
	cfg.Receivers = []Module{
		{GoMod: "example.com/receiver1 v0.0.1"},
		{GoMod: "example.com/receiver2 v0.0.1"},
	}

	_, err := buildEmbeddedSchema(cfg, func(modulePath string) (string, error) {
		if modulePath == "example.com/receiver1" {
			return firstDir, nil
		}
		return secondDir, nil
	})
	require.ErrorContains(t, err, "failed to combine collector schema")
}

func TestGenerateEmbeddedSchema(t *testing.T) {
	t.Parallel()

	componentDir := writeSchemaGoModule(t, "example.com/receiver", componentSchemaMetadata{Type: "otlp"}, map[string]any{
		"type": "object",
		"properties": map[string]any{
			"endpoint": map[string]any{"type": "string"},
		},
	})
	outputDir := writeDistributionModule(t, []string{
		"require example.com/receiver v0.0.1",
		"replace example.com/receiver => " + componentDir,
	})

	cfg := newTestConfig(t)
	cfg.Distribution.OutputPath = outputDir
	cfg.Distribution.Go = "go"
	cfg.Receivers = []Module{{GoMod: "example.com/receiver v0.0.1"}}

	require.NoError(t, generateEmbeddedSchemaWithDeps(cfg, defaultBuilderDeps()))

	// #nosec G304 -- outputDir is a test temporary directory
	schemaSource, err := os.ReadFile(filepath.Join(outputDir, embeddedSchemaFileName))
	require.NoError(t, err)
	require.Contains(t, string(schemaSource), "endpoint")
}

func TestGenerateEmbeddedSchemaGoListError(t *testing.T) {
	t.Parallel()

	outputDir := writeDistributionModule(t, nil)
	cfg := newTestConfig(t)
	cfg.Distribution.OutputPath = outputDir
	cfg.Distribution.Go = "go"
	cfg.Receivers = []Module{{GoMod: "example.com/missing v0.0.1"}}

	err := generateEmbeddedSchemaWithDeps(cfg, defaultBuilderDeps())
	require.Error(t, err)
}

func TestGenerateEmbeddedSchemaWriteError(t *testing.T) {
	t.Parallel()

	componentDir := writeSchemaGoModule(t, "example.com/receiver", componentSchemaMetadata{Type: "otlp"}, map[string]any{
		"type": "object",
	})
	outputDir := writeDistributionModule(t, []string{
		"require example.com/receiver v0.0.1",
		"replace example.com/receiver => " + componentDir,
	})
	// #nosec G302 -- intentionally set restrictive permissions to test write error
	require.NoError(t, os.Chmod(outputDir, 0o500))
	defer func() {
		// #nosec G302 -- restoring permissions after test
		_ = os.Chmod(outputDir, 0o700)
	}()

	cfg := newTestConfig(t)
	cfg.Distribution.OutputPath = outputDir
	cfg.Distribution.Go = "go"
	cfg.Receivers = []Module{{GoMod: "example.com/receiver v0.0.1"}}

	deps := defaultBuilderDeps()
	deps.writeEmbeddedSchemaSourceFile = func(string, []byte) error { return assert.AnError }

	err := generateEmbeddedSchemaWithDeps(cfg, deps)
	require.ErrorContains(t, err, "failed to write embedded schema source")
}

// Test that the go.mod files that other tests in this file
// may generate have all their modules covered by our
// "replace" statements created in `generateReplaces`.
//
// An incomplete set of replace statements in these tests
// may cause them to fail during the release process, when
// the local version of modules in the release branch is
// not yet available on the Go package repository.
// Unless the replace statements route all modules to the
// local copy, `go get` will try to fetch the unreleased
// version remotely and some tests will fail.
func TestReplaceStatementsAreComplete(t *testing.T) {
	workspaceDir := getWorkspaceDir()
	replaceMods := map[string]bool{}

	for _, suffix := range replaceModules {
		replaceMods[modulePrefix+suffix] = false
	}

	for _, mod := range replaceModules {
		verifyGoMod(t, workspaceDir+mod, replaceMods)
	}

	var err error
	dir := t.TempDir()
	cfg, err := NewDefaultConfig()
	require.NoError(t, err)
	cfg.Distribution.Go = "go"
	cfg.Distribution.OutputPath = dir
	cfg.Replaces = append(cfg.Replaces, generateReplaces()...)
	// Configure all components that we want to use elsewhere in these tests.
	// This ensures the resulting go.mod file has maximum coverage of modules
	// that exist in the Core repository.
	usedNames := make(map[string]int)
	cfg.Exporters, err = cfg.parseModules([]Module{
		{
			GoMod: "go.opentelemetry.io/collector/exporter/debugexporter v1.9999.9999",
		},
		{
			GoMod: "go.opentelemetry.io/collector/exporter/nopexporter v1.9999.9999",
		},
		{
			GoMod: "go.opentelemetry.io/collector/exporter/otlpexporter v1.9999.9999",
		},
		{
			GoMod: "go.opentelemetry.io/collector/exporter/otlphttpexporter v1.9999.9999",
		},
	}, usedNames)
	require.NoError(t, err)
	cfg.Receivers, err = cfg.parseModules([]Module{
		{
			GoMod: "go.opentelemetry.io/collector/receiver/nopreceiver v1.9999.9999",
		},
		{
			GoMod: "go.opentelemetry.io/collector/receiver/otlpreceiver v1.9999.9999",
		},
	}, usedNames)
	require.NoError(t, err)
	cfg.Extensions, err = cfg.parseModules([]Module{
		{
			GoMod: "go.opentelemetry.io/collector/extension/zpagesextension v1.9999.9999",
		},
	}, usedNames)
	require.NoError(t, err)
	cfg.Processors, err = cfg.parseModules([]Module{
		{
			GoMod: "go.opentelemetry.io/collector/processor/batchprocessor v1.9999.9999",
		},
		{
			GoMod: "go.opentelemetry.io/collector/processor/memorylimiterprocessor v1.9999.9999",
		},
	}, usedNames)
	require.NoError(t, err)

	require.NoError(t, cfg.Validate())
	require.NoError(t, cfg.ParseModules())
	err = GenerateAndCompile(cfg)
	require.NoError(t, err)

	verifyGoMod(t, dir, replaceMods)

	for k, v := range replaceMods {
		assert.Truef(t, v, "Module not used: %s", k)
	}
}

func verifyGoMod(t *testing.T, dir string, replaceMods map[string]bool) {
	gomodpath := path.Join(dir, "go.mod")
	gomod, err := os.ReadFile(filepath.Clean(gomodpath))
	require.NoError(t, err)

	mod, err := modfile.Parse(gomodpath, gomod, nil)
	require.NoError(t, err)

	for _, req := range mod.Require {
		if !strings.HasPrefix(req.Mod.Path, modulePrefix) {
			continue
		}

		_, ok := replaceMods[req.Mod.Path]
		assert.Truef(t, ok, "Module missing from replace statements list: %s", req.Mod.Path)

		replaceMods[req.Mod.Path] = true
	}
}

func makeModule(dir string, fileContents []byte) error {
	// if the file does not exist, try to create it
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.Mkdir(dir, 0o750); err != nil {
			return fmt.Errorf("failed to create output path: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to create output path: %w", err)
	}

	err := os.WriteFile(filepath.Clean(filepath.Join(dir, "go.mod")), fileContents, 0o600)
	if err != nil {
		return fmt.Errorf("failed to write go.mod file: %w", err)
	}
	return nil
}

func generateReplaces() []string {
	workspaceDir := getWorkspaceDir()
	modules := replaceModules
	replaces := make([]string, len(modules))

	for i, mod := range modules {
		replaces[i] = fmt.Sprintf("%s%s => %s%s", modulePrefix, mod, workspaceDir, mod)
	}

	return replaces
}

func getWorkspaceDir() string {
	// This is dependent on the current file structure.
	// The goal is find the root of the repo so we can replace the root module.
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))))
}

func writeSchemaModule(t *testing.T, metadata componentSchemaMetadata, schema map[string]any) string {
	t.Helper()

	dir := t.TempDir()
	metadataBytes := fmt.Sprintf("type: %s\n", metadata.Type)
	if metadata.DeprecatedType != "" {
		metadataBytes += fmt.Sprintf("deprecated_type: %s\n", metadata.DeprecatedType)
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "metadata.yaml"), []byte(metadataBytes), 0o600))

	schemaBytes, err := json.Marshal(schema)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.schema.json"), schemaBytes, 0o600))

	return dir
}

func writeSchemaGoModule(t *testing.T, modulePath string, metadata componentSchemaMetadata, schema map[string]any) string {
	t.Helper()

	dir := writeSchemaModule(t, metadata, schema)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(fmt.Sprintf("module %s\n\ngo 1.25.0\n", modulePath)), 0o600))
	return dir
}

func writeDistributionModule(t *testing.T, directives []string) string {
	t.Helper()

	dir := t.TempDir()
	goMod := "module example.com/testdist\n\ngo 1.25.0\n"
	if len(directives) > 0 {
		goMod += "\n" + strings.Join(directives, "\n") + "\n"
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))
	return dir
}
