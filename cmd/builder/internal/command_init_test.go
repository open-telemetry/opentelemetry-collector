// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/builder/internal"

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/cmd/builder/internal/builder"
)

func TestInitCommand(t *testing.T) {
	cmd := initCommand()

	assert.NotNil(t, cmd)
	assert.IsType(t, &cobra.Command{}, cmd)
	assert.Equal(t, "init", cmd.Use)
}

const distributionName = "test-distro"

func TestRunInit(t *testing.T) {
	for _, tt := range []struct {
		name      string
		buildPath func(*testing.T) string

		wantErr string
	}{
		{
			name:      "without a path",
			buildPath: func(*testing.T) string { return "" },
			wantErr:   "argument must be a folder",
		},
		{
			name:      "with a relative path",
			buildPath: func(*testing.T) string { return "./" + distributionName },
			wantErr:   "",
		},
		{
			name:      "with an absolute path",
			buildPath: func(t *testing.T) string { return filepath.Join(t.TempDir(), distributionName) },
			wantErr:   "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.buildPath(t)
			t.Cleanup(func() {
				os.RemoveAll(path)
			})

			err := run(path)

			if tt.wantErr == "" {
				require.NoError(t, err)
				validateCollector(t, path)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestBuildManifest(t *testing.T) {
	dir := t.TempDir()
	meta := metadata{
		Name:        "myCollector",
		Description: defaultDescription,
		BetaVersion: builder.DefaultBetaOtelColVersion,
	}
	require.NoError(t, buildManifest(dir, meta))

	content, err := os.ReadFile(filepath.Join(dir, "manifest.yaml")) //nolint:gosec // G304: path is test-controlled
	require.NoError(t, err)

	expected := `dist:
    description: Custom OpenTelemetry Collector
    name: myCollector
    output_path: ./build/collector
exporters:
    - gomod: go.opentelemetry.io/collector/exporter/otlpexporter ` + builder.DefaultBetaOtelColVersion + `
receivers:
    - gomod: go.opentelemetry.io/collector/receiver/otlpreceiver ` + builder.DefaultBetaOtelColVersion + `
`

	assert.Equal(t, expected, string(content))
}

func validateCollector(t *testing.T, path string) {
	require.FileExists(t, filepath.Join(path, ".gitignore"))
	require.FileExists(t, filepath.Join(path, "README.md"))
	require.FileExists(t, filepath.Join(path, "manifest.yaml"))
	require.FileExists(t, filepath.Join(path, "go.mod"))
	require.FileExists(t, filepath.Join(path, "go.sum"))
	require.FileExists(t, filepath.Join(path, "Makefile"))
	require.FileExists(t, filepath.Join(path, "config.yaml"))
}
