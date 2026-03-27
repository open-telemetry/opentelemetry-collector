// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package create // import "go.opentelemetry.io/collector/cmd/builder/internal/create"

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

	data, err := os.ReadFile(filepath.Join(path, "metadata.yaml")) // #nosec G304 -- test path controlled
	require.NoError(t, err)
	assert.Contains(t, string(data), "class: exporter")
	assert.Contains(t, string(data), "type: my")
}
