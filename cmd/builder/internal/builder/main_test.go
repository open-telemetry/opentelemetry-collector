// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builder

import (
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

func TestGenerateInvalidOutputPath(t *testing.T) {
	cfg := newInitializedConfig(t)
	cfg.Distribution.OutputPath = ":/invalid"
	err := Generate(cfg)
	require.ErrorContains(t, err, "failed to create output path")
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
	cfg.Exporters, err = parseModules([]Module{
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
	cfg.Receivers, err = parseModules([]Module{
		{
			GoMod: "go.opentelemetry.io/collector/receiver/nopreceiver v1.9999.9999",
		},
		{
			GoMod: "go.opentelemetry.io/collector/receiver/otlpreceiver v1.9999.9999",
		},
	}, usedNames)
	require.NoError(t, err)
	cfg.Extensions, err = parseModules([]Module{
		{
			GoMod: "go.opentelemetry.io/collector/extension/zpagesextension v1.9999.9999",
		},
	}, usedNames)
	require.NoError(t, err)
	cfg.Processors, err = parseModules([]Module{
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
	//nolint:gosec // #nosec G304 We control this path and generate the file inside, so we can assume it is safe.
	gomod, err := os.ReadFile(gomodpath)
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
