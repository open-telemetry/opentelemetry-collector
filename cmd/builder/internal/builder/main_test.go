// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var (
	goModTestFile = []byte(`// Copyright The OpenTelemetry Authors
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
)`)
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
				err := makeModule(tempDir, goModTestFile)
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
	// This test is dependent on the current file structure.
	// The goal is find the root of the repo so we can replace the root module.
	_, thisFile, _, _ := runtime.Caller(0)
	workspaceDir := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))))
	return []string{fmt.Sprintf("go.opentelemetry.io/collector => %s", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/component => %s/component", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/config/configauth => %s/config/configauth", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/config/configcompression => %s/config/configcompression", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/config/configgrpc => %s/config/configgrpc", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/config/confignet => %s/config/confignet", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/config/configopaque => %s/config/configopaque", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/config/configretry => %s/config/configretry", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/config/configtelemetry => %s/config/configtelemetry", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/config/configtls => %s/config/configtls", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/config/internal => %s/config/internal", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/confmap => %s/confmap", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/confmap/converter/expandconverter => %s/confmap/converter/expandconverter", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/confmap/provider/envprovider => %s/confmap/provider/envprovider", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/confmap/provider/fileprovider => %s/confmap/provider/fileprovider", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/confmap/provider/httpprovider => %s/confmap/provider/httpprovider", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/confmap/provider/httpsprovider => %s/confmap/provider/httpsprovider", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/confmap/provider/yamlprovider => %s/confmap/provider/yamlprovider", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/consumer => %s/consumer", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/connector => %s/connector", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/exporter => %s/exporter", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/exporter/debugexporter => %s/exporter/debugexporter", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/exporter/loggingexporter => %s/exporter/loggingexporter", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/exporter/nopexporter => %s/exporter/nopexporter", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/exporter/otlpexporter => %s/exporter/otlpexporter", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/exporter/otlphttpexporter => %s/exporter/otlphttpexporter", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/extension => %s/extension", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/extension/auth => %s/extension/auth", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/extension/ballastextension => %s/extension/ballastextension", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/extension/zpagesextension => %s/extension/zpagesextension", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/featuregate => %s/featuregate", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/processor => %s/processor", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/processor/batchprocessor => %s/processor/batchprocessor", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/processor/memorylimiterprocessor => %s/processor/memorylimiterprocessor", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/receiver => %s/receiver", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/receiver/nopreceiver => %s/receiver/nopreceiver", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/receiver/otlpreceiver => %s/receiver/otlpreceiver", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/otelcol => %s/otelcol", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/pdata => %s/pdata", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/pdata/testdata => %s/pdata/testdata", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/semconv => %s/semconv", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/service => %s/service", workspaceDir),
	}
}
