// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package create

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCommand(t *testing.T) {
	cmd := CreateCommand()

	assert.NotNil(t, cmd)
	assert.IsType(t, &cobra.Command{}, cmd)
	assert.Equal(t, "create", cmd.Use)
}

func TestRunCreateExporter(t *testing.T) {
	for _, tt := range []struct {
		name      string
		buildPath func(string) string
		signals   []string
		wantErr   string
	}{
		{
			name:      "empty directory name",
			buildPath: func(string) string { return "" },
			signals:   []string{"traces", "metrics", "logs"},
			wantErr:   "directory name cannot be empty",
		},
		{
			name:      "with relative path",
			buildPath: func(string) string { return "./tmp/myexporter" },
			signals:   []string{"traces", "metrics", "logs"},
			wantErr:   "",
		},
		{
			name:      "with absolute path",
			buildPath: func(dir string) string { return dir },
			signals:   []string{"traces", "metrics", "logs"},
			wantErr:   "",
		},
		{
			name:      "with only traces signal",
			buildPath: func(dir string) string { return dir },
			signals:   []string{"traces"},
			wantErr:   "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tmpdir := filepath.Join(t.TempDir(), "myexporter")
			path := tt.buildPath(tmpdir)
			err := runCreateExporter("my", tt.signals, path)

			if tt.wantErr == "" {
				require.NoError(t, err)
				validateExporter(t, filepath.Join(path, "myexporter"))
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func validateExporter(t *testing.T, path string) {
	t.Helper()

	require.FileExists(t, filepath.Join(path, ".gitignore"))
	require.FileExists(t, filepath.Join(path, "README.md"))
	require.FileExists(t, filepath.Join(path, "go.mod"))
	require.FileExists(t, filepath.Join(path, "Makefile"))
	require.FileExists(t, filepath.Join(path, "metadata.yaml"))
	require.FileExists(t, filepath.Join(path, "config.go"))
	require.FileExists(t, filepath.Join(path, "factory.go"))
	require.FileExists(t, filepath.Join(path, "exporter.go"))

	data, err := os.ReadFile(filepath.Join(path, "metadata.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "class: exporter")
	assert.Contains(t, string(data), "type: my")
}
