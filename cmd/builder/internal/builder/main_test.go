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

var (
	replaceModules = []string{
		"",
		"/component",
		"/config/configauth",
		"/config/configcompression",
		"/config/configgrpc",
		"/config/confighttp",
		"/config/confignet",
		"/config/configopaque",
		"/config/configretry",
		"/config/configtelemetry",
		"/config/configtls",
		"/config/internal",
		"/confmap",
		"/confmap/converter/expandconverter",
		"/confmap/provider/envprovider",
		"/confmap/provider/fileprovider",
		"/confmap/provider/httpprovider",
		"/confmap/provider/httpsprovider",
		"/confmap/provider/yamlprovider",
		"/consumer",
		"/connector",
		"/exporter",
		"/exporter/debugexporter",
		"/exporter/nopexporter",
		"/exporter/otlpexporter",
		"/exporter/otlphttpexporter",
		"/extension",
		"/extension/auth",
		"/extension/zpagesextension",
		"/featuregate",
		"/processor",
		"/processor/batchprocessor",
		"/processor/memorylimiterprocessor",
		"/receiver",
		"/receiver/nopreceiver",
		"/receiver/otlpreceiver",
		"/otelcol",
		"/pdata",
		"/pdata/testdata",
		"/semconv",
		"/service",
	}
)

func newTestConfig() Config {
	cfg := NewDefaultConfig()
	cfg.downloadModules.wait = 0
	cfg.downloadModules.numRetries = 1
	return cfg
}

func newInitializedConfig(t *testing.T) Config {
	cfg := newTestConfig()
	// Validate and ParseModules will be called before the config is
	// given to Generate.
	assert.NoError(t, cfg.Validate())
	assert.NoError(t, cfg.SetBackwardsCompatibility())
	assert.NoError(t, cfg.ParseModules())

	return cfg
}

func TestGenerateDefault(t *testing.T) {
	require.NoError(t, Generate(newInitializedConfig(t)))
}

func TestGenerateInvalidOutputPath(t *testing.T) {
	cfg := newInitializedConfig(t)
	cfg.Distribution.OutputPath = "/:invalid"
	err := Generate(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create output path")
}

func TestVersioning(t *testing.T) {
	replaces := generateReplaces()
	tests := []struct {
		description string
		cfgBuilder  func() Config
		expectedErr error
	}{
		{
			description: "defaults",
			cfgBuilder: func() Config {
				cfg := newTestConfig()
				cfg.Distribution.Go = "go"
				cfg.Replaces = append(cfg.Replaces, replaces...)
				return cfg
			},
			expectedErr: nil,
		},
		{
			description: "require otelcol",
			cfgBuilder: func() Config {
				cfg := newTestConfig()
				cfg.Distribution.Go = "go"
				cfg.Distribution.RequireOtelColModule = true
				cfg.Replaces = append(cfg.Replaces, replaces...)
				return cfg
			},
			expectedErr: nil,
		},
		{
			description: "only gomod file, skip generate",
			cfgBuilder: func() Config {
				cfg := newTestConfig()
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
			description: "old otel version",
			cfgBuilder: func() Config {
				cfg := newTestConfig()
				cfg.Verbose = true
				cfg.Distribution.Go = "go"
				cfg.Distribution.OtelColVersion = "0.97.0"
				cfg.Distribution.RequireOtelColModule = true
				var err error
				cfg.Exporters, err = parseModules([]Module{
					{
						GoMod: "go.opentelemetry.io/collector/exporter/otlpexporter v0.97.0",
					},
				})
				require.NoError(t, err)
				cfg.Receivers, err = parseModules([]Module{
					{
						GoMod: "go.opentelemetry.io/collector/receiver/otlpreceiver v0.97.0",
					},
				})
				require.NoError(t, err)
				providers, err := parseModules([]Module{
					{
						GoMod: "go.opentelemetry.io/collector/confmap/provider/envprovider v0.97.0",
					},
				})
				require.NoError(t, err)
				cfg.Providers = &providers
				return cfg
			},
			expectedErr: nil,
		},
		{
			description: "old component version",
			cfgBuilder: func() Config {
				cfg := newTestConfig()
				cfg.Distribution.Go = "go"
				cfg.Exporters = []Module{
					{
						GoMod: "go.opentelemetry.io/collector/exporter/otlpexporter v0.97.0",
					},
				}
				cfg.Providers = &[]Module{}
				cfg.Replaces = append(cfg.Replaces, replaces...)
				return cfg
			},
			expectedErr: nil,
		},
		{
			description: "old component version without strict mode",
			cfgBuilder: func() Config {
				cfg := newTestConfig()
				cfg.Distribution.Go = "go"
				cfg.SkipStrictVersioning = true
				cfg.Exporters = []Module{
					{
						GoMod: "go.opentelemetry.io/collector/exporter/otlpexporter v0.97.0",
					},
				}
				cfg.Providers = &[]Module{}
				cfg.Replaces = append(cfg.Replaces, replaces...)
				return cfg
			},
			expectedErr: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			cfg := tc.cfgBuilder()
			require.NoError(t, cfg.SetBackwardsCompatibility())
			require.NoError(t, cfg.Validate())
			require.NoError(t, cfg.ParseModules())
			err := GenerateAndCompile(cfg)
			require.ErrorIs(t, err, tc.expectedErr)
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
		testCase   string
		cfgBuilder func(t *testing.T) Config
	}{
		{
			testCase: "Default Configuration Compilation",
			cfgBuilder: func(t *testing.T) Config {
				cfg := newTestConfig()
				cfg.Distribution.OutputPath = t.TempDir()
				cfg.Replaces = append(cfg.Replaces, replaces...)
				return cfg
			},
		},
		{
			testCase: "LDFlags Compilation",
			cfgBuilder: func(t *testing.T) Config {
				cfg := newTestConfig()
				cfg.Distribution.OutputPath = t.TempDir()
				cfg.Replaces = append(cfg.Replaces, replaces...)
				cfg.LDFlags = `-X "test.gitVersion=0743dc6c6411272b98494a9b32a63378e84c34da" -X "test.gitTag=local-testing" -X "test.goVersion=go version go1.20.7 darwin/amd64"`
				return cfg
			},
		},
		{
			testCase: "Debug Compilation",
			cfgBuilder: func(t *testing.T) Config {
				cfg := newTestConfig()
				cfg.Distribution.OutputPath = t.TempDir()
				cfg.Replaces = append(cfg.Replaces, replaces...)
				cfg.Logger = zap.NewNop()
				cfg.Distribution.DebugCompilation = true
				return cfg
			},
		},
		{
			testCase: "No providers",
			cfgBuilder: func(t *testing.T) Config {
				cfg := newTestConfig()
				cfg.Distribution.OutputPath = t.TempDir()
				cfg.Replaces = append(cfg.Replaces, replaces...)
				cfg.Providers = &[]Module{}
				return cfg
			},
		},
		{
			testCase: "Pre-confmap factories",
			cfgBuilder: func(t *testing.T) Config {
				cfg := newTestConfig()
				cfg.Distribution.OutputPath = t.TempDir()
				cfg.Replaces = append(cfg.Replaces, replaces...)
				cfg.Distribution.OtelColVersion = "0.98.0"
				cfg.SkipStrictVersioning = true
				return cfg
			},
		},
		{
			testCase: "With confmap factories",
			cfgBuilder: func(t *testing.T) Config {
				cfg := newTestConfig()
				cfg.Distribution.OutputPath = t.TempDir()
				cfg.Replaces = append(cfg.Replaces, replaces...)
				cfg.Distribution.OtelColVersion = "0.99.0"
				cfg.SkipStrictVersioning = true
				return cfg
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.testCase, func(t *testing.T) {
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
	cfg := NewDefaultConfig()
	cfg.Distribution.Go = "go"
	cfg.Distribution.OutputPath = dir
	// Use a deliberately nonexistent version to simulate an unreleased
	// version of the package. Not strictly necessary since this test
	// will catch gaps in the replace statements before a release is in
	// progress.
	cfg.Distribution.OtelColVersion = "1.9999.9999"
	cfg.Replaces = append(cfg.Replaces, generateReplaces()...)
	// Configure all components that we want to use elsewhere in these tests.
	// This ensures the resulting go.mod file has maximum coverage of modules
	// that exist in the Core repository.
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
	})
	require.NoError(t, err)
	cfg.Receivers, err = parseModules([]Module{
		{
			GoMod: "go.opentelemetry.io/collector/receiver/nopreceiver v1.9999.9999",
		},
		{
			GoMod: "go.opentelemetry.io/collector/receiver/otlpreceiver v1.9999.9999",
		},
	})
	require.NoError(t, err)
	cfg.Extensions, err = parseModules([]Module{
		{
			GoMod: "go.opentelemetry.io/collector/extension/zpagesextension v1.9999.9999",
		},
	})
	require.NoError(t, err)
	cfg.Processors, err = parseModules([]Module{
		{
			GoMod: "go.opentelemetry.io/collector/processor/batchprocessor v1.9999.9999",
		},
		{
			GoMod: "go.opentelemetry.io/collector/processor/memorylimiterprocessor v1.9999.9999",
		},
	})
	require.NoError(t, err)

	require.NoError(t, cfg.SetBackwardsCompatibility())
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
	// #nosec G304 We control this path and generate the file inside, so we can assume it is safe.
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
		if err = os.Mkdir(dir, 0750); err != nil {
			return fmt.Errorf("failed to create output path: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to create output path: %w", err)
	}

	err := os.WriteFile(filepath.Clean(filepath.Join(dir, "go.mod")), fileContents, 0600)
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
