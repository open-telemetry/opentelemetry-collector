// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepoRoot(t *testing.T) {
	t.Run("returns_repo_root_from_subdirectory", func(t *testing.T) {
		wd, err := os.Getwd()
		require.NoError(t, err)

		root, err := repoRoot(wd)
		require.NoError(t, err)
		assert.DirExists(t, root)
		assert.FileExists(t, filepath.Join(root, "go.mod"))
	})

	t.Run("error_for_nonexistent_directory", func(t *testing.T) {
		_, err := repoRoot("/nonexistent/path/that/does/not/exist")
		require.Error(t, err)
	})

	t.Run("error_outside_git_repo", func(t *testing.T) {
		tmp := t.TempDir()
		_, err := repoRoot(tmp)
		require.Error(t, err)
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

	t.Run("error_when_go_mod_missing", func(t *testing.T) {
		tmp := t.TempDir()
		subDir := filepath.Join(tmp, "sub")
		require.NoError(t, os.MkdirAll(subDir, 0o700))

		gitInit(t, tmp)

		_, err := RootPackage(subDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "go.mod")
	})

	t.Run("error_when_go_mod_has_no_module_directive", func(t *testing.T) {
		tmp := t.TempDir()
		subDir := filepath.Join(tmp, "sub")
		require.NoError(t, os.MkdirAll(subDir, 0o700))

		gitInit(t, tmp)
		require.NoError(t, os.WriteFile(
			filepath.Join(tmp, "go.mod"),
			[]byte("go 1.21\n"),
			0o600,
		))

		_, err := RootPackage(subDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "module directive not found")
	})

	t.Run("parses_module_from_synthetic_go_mod", func(t *testing.T) {
		tmp := t.TempDir()
		subDir := filepath.Join(tmp, "pkg", "foo")
		require.NoError(t, os.MkdirAll(subDir, 0o700))

		gitInit(t, tmp)
		require.NoError(t, os.WriteFile(
			filepath.Join(tmp, "go.mod"),
			[]byte("module example.com/my-project\n\ngo 1.21\n"),
			0o600,
		))

		pkg, err := RootPackage(subDir)
		require.NoError(t, err)
		assert.Equal(t, "example.com/my-project", pkg)
	})
}

func gitInit(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git init failed: %s", out)
}
