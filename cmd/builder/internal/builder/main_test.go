// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package builder

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

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

func TestGenerateAndCompile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping the test on Windows, see https://github.com/open-telemetry/opentelemetry-collector/issues/5403")
	}
	// This test is dependent on the current file structure.
	// The goal is find the root of the repo so we can replace the root module.
	_, thisFile, _, _ := runtime.Caller(0)
	workspaceDir := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))))
	replaces := []string{fmt.Sprintf("go.opentelemetry.io/collector => %s", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/component => %s/component", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/confmap => %s/confmap", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/consumer => %s/consumer", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/exporter => %s/exporter", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/exporter/loggingexporter => %s/exporter/loggingexporter", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/exporter/otlpexporter => %s/exporter/otlpexporter", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/exporter/otlphttpexporter => %s/exporter/otlphttpexporter", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/extension/ballastextension => %s/extension/ballastextension", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/extension/zpagesextension => %s/extension/zpagesextension", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/featuregate => %s/featuregate", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/processor/batchprocessor => %s/processor/batchprocessor", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/processor/memorylimiterprocessor => %s/processor/memorylimiterprocessor", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/receiver => %s/receiver", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/receiver/otlpreceiver => %s/receiver/otlpreceiver", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/pdata => %s/pdata", workspaceDir),
		fmt.Sprintf("go.opentelemetry.io/collector/semconv => %s/semconv", workspaceDir),
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
				cfg.LDFlags = `-X "test.gitVersion=0743dc6c6411272b98494a9b32a63378e84c34da" -X "test.gitTag=local-testing" -X "test.goVersion=go version go1.19.4 darwin/amd64"`
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

	// Sleep for 1 second to make sure all processes using the files are completed
	// (on Windows fail to delete temp dir otherwise).
	time.Sleep(1 * time.Second)
}
