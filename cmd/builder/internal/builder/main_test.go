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

func newInitializedConfig(t *testing.T) Config {
	cfg := NewDefaultConfig()
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
	cfg.Distribution.OutputPath = "/:invalid"
	err := Generate(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create output path")
}

func TestVersioning(t *testing.T) {
	tests := []struct {
		description string
		cfgBuilder  func() Config
		expectedErr error
	}{
		{
			description: "defaults",
			cfgBuilder: func() Config {
				cfg := newInitializedConfig(t)
				cfg.Distribution.Go = "go"
				return cfg
			},
			expectedErr: nil,
		},
		{
			description: "require otelcol",
			cfgBuilder: func() Config {
				cfg := newInitializedConfig(t)
				cfg.Distribution.Go = "go"
				cfg.Distribution.RequireOtelColModule = true
				return cfg
			},
			expectedErr: nil,
		},
		{
			description: "only gomod file, skip generate",
			cfgBuilder: func() Config {
				cfg := newInitializedConfig(t)
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
				cfg := newInitializedConfig(t)
				cfg.Distribution.OtelColVersion = "0.90.0"
				return cfg
			},
			expectedErr: ErrVersionMismatch,
		},
		{
			description: "old otel version without strict mode",
			cfgBuilder: func() Config {
				cfg := newInitializedConfig(t)
				cfg.Verbose = true
				cfg.Distribution.Go = "go"
				cfg.SkipStrictVersioning = true
				cfg.Distribution.OtelColVersion = "0.90.0"
				return cfg
			},
			expectedErr: nil,
		},
		{
			description: "invalid collector version",
			cfgBuilder: func() Config {
				cfg := newInitializedConfig(t)
				cfg.Distribution.OtelColVersion = "invalid"
				return cfg
			},
			expectedErr: ErrVersionMismatch,
		},
		{
			description: "invalid collector version without strict mode, only generate",
			cfgBuilder: func() Config {
				cfg := newInitializedConfig(t)
				cfg.Distribution.OtelColVersion = "invalid"
				cfg.SkipGetModules = true
				cfg.SkipCompilation = true
				cfg.SkipStrictVersioning = true
				return cfg
			},
			expectedErr: nil,
		},
		{
			description: "invalid collector version without strict mode",
			cfgBuilder: func() Config {
				cfg := newInitializedConfig(t)
				cfg.Distribution.OtelColVersion = "invalid"
				cfg.SkipStrictVersioning = true
				return cfg
			},
			expectedErr: errGoGetFailed,
		},
		{
			description: "old component version",
			cfgBuilder: func() Config {
				cfg := newInitializedConfig(t)
				cfg.Distribution.Go = "go"
				cfg.Exporters = []Module{
					{
						Import: "go.opentelemetry.io/collector/receiver/otlpreceiver",
						GoMod:  "go.opentelemetry.io/collector v0.96.0",
					},
				}
				return cfg
			},
			expectedErr: ErrVersionMismatch,
		},
		{
			description: "old component version without strict mode",
			cfgBuilder: func() Config {
				cfg := newInitializedConfig(t)
				cfg.Distribution.Go = "go"
				cfg.SkipStrictVersioning = true
				cfg.Exporters = []Module{
					{
						Import: "go.opentelemetry.io/collector/receiver/otlpreceiver",
						GoMod:  "go.opentelemetry.io/collector v0.96.0",
					},
				}
				return cfg
			},
			expectedErr: errCompileFailed,
		},
		{
			description: "invalid component version",
			cfgBuilder: func() Config {
				cfg := newInitializedConfig(t)
				cfg.Distribution.Go = "go"
				cfg.Exporters = []Module{
					{
						Import: "go.opentelemetry.io/collector/receiver/otlpreceiver",
						GoMod:  "go.opentelemetry.io/collector invalid",
					},
				}
				return cfg
			},
			expectedErr: errGoGetFailed,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			cfg := tc.cfgBuilder()
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
	// This test is dependent on the current file structure.
	// The goal is find the root of the repo so we can replace the root module.
	_, thisFile, _, _ := runtime.Caller(0)
	workspaceDir := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))))
	replaces := []string{fmt.Sprintf("go.opentelemetry.io/collector => %s", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/component => %s/component", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/config/confignet => %s/config/confignet", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/config/configtelemetry => %s/config/configtelemetry", workspaceDir),
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

	testCases := []struct {
		testCase   string
		cfgBuilder func(t *testing.T) Config
	}{
		{
			testCase: "Default Configuration Compilation",
			cfgBuilder: func(t *testing.T) Config {
				cfg := NewDefaultConfig()
				cfg.Distribution.OutputPath = t.TempDir()
				cfg.Replaces = append(cfg.Replaces, replaces...)
				return cfg
			},
		},
		{
			testCase: "LDFlags Compilation",
			cfgBuilder: func(t *testing.T) Config {
				cfg := NewDefaultConfig()
				cfg.Distribution.OutputPath = t.TempDir()
				cfg.Replaces = append(cfg.Replaces, replaces...)
				cfg.LDFlags = `-X "test.gitVersion=0743dc6c6411272b98494a9b32a63378e84c34da" -X "test.gitTag=local-testing" -X "test.goVersion=go version go1.20.7 darwin/amd64"`
				return cfg
			},
		},
		{
			testCase: "Debug Compilation",
			cfgBuilder: func(t *testing.T) Config {
				cfg := NewDefaultConfig()
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
				cfg := NewDefaultConfig()
				cfg.Distribution.OutputPath = t.TempDir()
				cfg.Replaces = append(cfg.Replaces, replaces...)
				cfg.Providers = &[]Module{}
				return cfg
			},
		},
		{
			testCase: "Pre-confmap factories",
			cfgBuilder: func(t *testing.T) Config {
				cfg := NewDefaultConfig()
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
				cfg := NewDefaultConfig()
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
