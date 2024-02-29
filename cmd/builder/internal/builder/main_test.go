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
)

func TestGenerateDefault(t *testing.T) {
	require.NoError(t, Generate(NewDefaultConfig()))
}

func TestGenerateInvalidCollectorVersion(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Distribution.OtelColVersion = "invalid"
	err := Generate(cfg)
	require.NoError(t, err)
}

func TestGenerateInvalidOutputPath(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Distribution.OutputPath = "/:invalid"
	err := Generate(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create output path")
}

func TestSkipGenerate(t *testing.T) {
	cfg := NewDefaultConfig()
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
		fmt.Sprintf("go.opentelemetry.io/collector/receiver/otlpreceiver => %s/receiver/otlpreceiver", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/otelcol => %s/otelcol", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/pdata => %s/pdata", workspaceDir),
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
				cfg.Distribution.DebugCompilation = true
				return cfg
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.testCase, func(t *testing.T) {
			cfg := tt.cfgBuilder(t)
			assert.NoError(t, cfg.Validate())
			assert.NoError(t, cfg.SetGoPath())
			require.NoError(t, GenerateAndCompile(cfg))
		})
	}
}
