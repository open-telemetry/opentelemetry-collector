// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/builder/internal"

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
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

func validateCollector(t *testing.T, path string) {
	require.FileExists(t, filepath.Join(path, ".gitignore"))
	require.FileExists(t, filepath.Join(path, "README.md"))
	require.FileExists(t, filepath.Join(path, "manifest.yaml"))
	require.FileExists(t, filepath.Join(path, "go.mod"))
	require.FileExists(t, filepath.Join(path, "go.sum"))
	require.FileExists(t, filepath.Join(path, "Makefile"))
	require.FileExists(t, filepath.Join(path, "config.yaml"))

	k := koanf.New(".")
	err := k.Load(file.Provider(filepath.Join(path, "manifest.yaml")), yaml.Parser())
	require.NoError(t, err)

	cfg := builder.Config{}
	err = k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{
		Tag: "mapstructure",
	})
	require.NoError(t, err)

	assert.Equal(t, distributionName, cfg.Distribution.Name)
	assert.Equal(t, defaultDescription, cfg.Distribution.Description)
}
