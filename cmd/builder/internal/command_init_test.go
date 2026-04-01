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

func TestRunInit(t *testing.T) {
	for _, tt := range []struct {
		name      string
		buildPath func(string) string

		wantErr string
	}{
		{
			name:      "without a path",
			buildPath: func(string) string { return "" },
			wantErr:   "argument must be a folder",
		},
		{
			name:      "with a relative path",
			buildPath: func(string) string { return "./tmp/init" },
			wantErr:   "",
		},
		{
			name:      "with an absolute path",
			buildPath: func(dir string) string { return dir },
			wantErr:   "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tmpdir := filepath.Join(t.TempDir(), "init")
			path := tt.buildPath(tmpdir)
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

func validateCollector(t *testing.T, path string) {
	require.FileExists(t, filepath.Join(path, ".gitignore"))
	require.FileExists(t, filepath.Join(path, "README.md"))
	require.FileExists(t, filepath.Join(path, "manifest.yaml"))
	require.FileExists(t, filepath.Join(path, "go.mod"))
	require.FileExists(t, filepath.Join(path, "go.sum"))
	require.FileExists(t, filepath.Join(path, "Makefile"))
	require.FileExists(t, filepath.Join(path, "config.yaml"))
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

	expected := "dist:\n" +
		"    description: Custom OpenTelemetry Collector\n" +
		"    name: myCollector\n" +
		"    output_path: ./build/collector\n" +
		"exporters:\n" +
		"    - gomod: go.opentelemetry.io/collector/exporter/otlpexporter " + builder.DefaultBetaOtelColVersion + "\n" +
		"receivers:\n" +
		"    - gomod: go.opentelemetry.io/collector/receiver/otlpreceiver " + builder.DefaultBetaOtelColVersion + "\n"

	assert.Equal(t, expected, string(content))
}
