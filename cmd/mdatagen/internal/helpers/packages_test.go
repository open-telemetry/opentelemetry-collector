// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootModuleDir(t *testing.T) {
	t.Run("finds_go_mod_in_parent", func(t *testing.T) {
		tmp := t.TempDir()
		subDir := filepath.Join(tmp, "sub")
		require.NoError(t, os.MkdirAll(subDir, 0o700))

		require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module example.com/root\n"), 0o600))

		dir, err := rootModuleDir(subDir)
		require.NoError(t, err)
		assert.Equal(t, tmp, dir)
	})

	t.Run("finds_go_mod_in_intermediate_directory", func(t *testing.T) {
		tmp := t.TempDir()
		projectDir := filepath.Join(tmp, "project")
		componentDir := filepath.Join(projectDir, "receiver", "foo")
		require.NoError(t, os.MkdirAll(componentDir, 0o700))

		// No go.mod at tmp root; go.mod is in project/ subdirectory
		require.NoError(t, os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module example.com/project\n"), 0o600))

		dir, err := rootModuleDir(componentDir)
		require.NoError(t, err)
		assert.Equal(t, projectDir, dir)
	})

	t.Run("prefers_highest_go_mod", func(t *testing.T) {
		tmp := t.TempDir()
		componentDir := filepath.Join(tmp, "project", "receiver", "foo")
		require.NoError(t, os.MkdirAll(componentDir, 0o700))

		// go.mod at both levels
		require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module example.com/root\n"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(tmp, "project", "go.mod"), []byte("module example.com/project\n"), 0o600))

		dir, err := rootModuleDir(componentDir)
		require.NoError(t, err)
		assert.Equal(t, tmp, dir)
	})

	t.Run("error_when_no_go_mod_found", func(t *testing.T) {
		tmp := t.TempDir()
		subDir := filepath.Join(tmp, "sub")
		require.NoError(t, os.MkdirAll(subDir, 0o700))

		_, err := rootModuleDir(subDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no go.mod found")
	})
}

func TestRootPackage(t *testing.T) {
	t.Run("returns_correct_module_path", func(t *testing.T) {
		wd, err := os.Getwd()
		require.NoError(t, err)

		pkg, err := RootPackage(wd)
		require.NoError(t, err)
		assert.Equal(t, "go.opentelemetry.io/collector", pkg)
	})

	t.Run("error_for_nonexistent_directory", func(t *testing.T) {
		_, err := RootPackage("/nonexistent/path/that/does/not/exist")
		require.Error(t, err)
	})

	t.Run("resolves_module_from_synthetic_go_mod", func(t *testing.T) {
		tmp := t.TempDir()
		subDir := filepath.Join(tmp, "pkg", "foo")
		require.NoError(t, os.MkdirAll(subDir, 0o700))

		require.NoError(t, os.WriteFile(
			filepath.Join(tmp, "go.mod"),
			[]byte("module example.com/my-project\n\ngo 1.21\n"),
			0o600,
		))

		pkg, err := RootPackage(subDir)
		require.NoError(t, err)
		assert.Equal(t, "example.com/my-project", pkg)
	})

	t.Run("monorepo_go_mod_not_at_top_level", func(t *testing.T) {
		tmp := t.TempDir()
		projectDir := filepath.Join(tmp, "collector")
		componentDir := filepath.Join(projectDir, "receiver", "foo")
		require.NoError(t, os.MkdirAll(componentDir, 0o700))

		// No go.mod at top level; only in collector/ subdirectory
		require.NoError(t, os.WriteFile(
			filepath.Join(projectDir, "go.mod"),
			[]byte("module go.opentelemetry.io/collector\n\ngo 1.21\n"),
			0o600,
		))

		pkg, err := RootPackage(componentDir)
		require.NoError(t, err)
		assert.Equal(t, "go.opentelemetry.io/collector", pkg)
	})
}
