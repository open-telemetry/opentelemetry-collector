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

func TestGenerateAndCompileDefault(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping the test on Windows, see https://github.com/open-telemetry/opentelemetry-collector/issues/5403")
	}
	cfg := NewDefaultConfig()
	cfg.Distribution.OutputPath = t.TempDir()

	// This test is dependent on the current file structure.
	// The goal is find the root of the repo so we can replace the root module.
	_, thisFile, _, _ := runtime.Caller(0)
	workspaceDir := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))))
	cfg.Replaces = append(cfg.Replaces, fmt.Sprintf("go.opentelemetry.io/collector => %s", workspaceDir))
	cfg.Replaces = append(cfg.Replaces, fmt.Sprintf("go.opentelemetry.io/collector/component => %s/component", workspaceDir))
	cfg.Replaces = append(cfg.Replaces, fmt.Sprintf("go.opentelemetry.io/collector/confmap => %s/confmap", workspaceDir))
	cfg.Replaces = append(cfg.Replaces, fmt.Sprintf("go.opentelemetry.io/collector/consumer => %s/consumer", workspaceDir))
	cfg.Replaces = append(cfg.Replaces, fmt.Sprintf("go.opentelemetry.io/collector/exporter/loggingexporter => %s/exporter/loggingexporter", workspaceDir))
	cfg.Replaces = append(cfg.Replaces, fmt.Sprintf("go.opentelemetry.io/collector/exporter/otlpexporter => %s/exporter/otlpexporter", workspaceDir))
	cfg.Replaces = append(cfg.Replaces, fmt.Sprintf("go.opentelemetry.io/collector/exporter/otlphttpexporter => %s/exporter/otlphttpexporter", workspaceDir))
	cfg.Replaces = append(cfg.Replaces, fmt.Sprintf("go.opentelemetry.io/collector/extension/ballastextension => %s/extension/ballastextension", workspaceDir))
	cfg.Replaces = append(cfg.Replaces, fmt.Sprintf("go.opentelemetry.io/collector/extension/zpagesextension => %s/extension/zpagesextension", workspaceDir))
	cfg.Replaces = append(cfg.Replaces, fmt.Sprintf("go.opentelemetry.io/collector/featuregate => %s/featuregate", workspaceDir))
	cfg.Replaces = append(cfg.Replaces, fmt.Sprintf("go.opentelemetry.io/collector/processor/batchprocessor => %s/processor/batchprocessor", workspaceDir))
	cfg.Replaces = append(cfg.Replaces, fmt.Sprintf("go.opentelemetry.io/collector/processor/memorylimiterprocessor => %s/processor/memorylimiterprocessor", workspaceDir))
	cfg.Replaces = append(cfg.Replaces, fmt.Sprintf("go.opentelemetry.io/collector/receiver/otlpreceiver => %s/receiver/otlpreceiver", workspaceDir))
	cfg.Replaces = append(cfg.Replaces, fmt.Sprintf("go.opentelemetry.io/collector/pdata => %s/pdata", workspaceDir))
	cfg.Replaces = append(cfg.Replaces, fmt.Sprintf("go.opentelemetry.io/collector/semconv => %s/semconv", workspaceDir))

	assert.NoError(t, cfg.Validate())
	assert.NoError(t, cfg.SetGoPath())
	require.NoError(t, GenerateAndCompile(cfg))

	// Sleep for 1 second to make sure all processes using the files are completed
	// (on Windows fail to delete temp dir otherwise).
	time.Sleep(1 * time.Second)
}
