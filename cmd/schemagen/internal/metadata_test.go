// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolvePackageDir(t *testing.T) {
	moduleDir := setupResolvePackageDirModule(t)

	tests := []struct {
		name    string
		dir     string
		pattern string
		wantDir string
	}{
		{
			name:    "module import path",
			dir:     moduleDir,
			pattern: "example.com/schemagen/receiver/foo",
			wantDir: filepath.Join(moduleDir, "receiver", "foo"),
		},
		{
			name:    "relative package pattern",
			dir:     moduleDir,
			pattern: "./receiver/foo",
			wantDir: filepath.Join(moduleDir, "receiver", "foo"),
		},
		{
			name:    "pattern is resolved relative to dir",
			dir:     filepath.Join(moduleDir, "cmd", "tool"),
			pattern: "../../receiver/foo",
			wantDir: filepath.Join(moduleDir, "receiver", "foo"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ResolvePackageDir(tt.dir, tt.pattern)
			require.True(t, ok)
			require.Equal(t, resolvePath(t, tt.wantDir), resolvePath(t, got))
		})
	}
}

func TestResolvePackageDirReturnsFalse(t *testing.T) {
	moduleDir := setupResolvePackageDirModule(t)

	tests := []struct {
		name    string
		dir     string
		pattern string
	}{
		{
			name:    "unresolvable package",
			dir:     moduleDir,
			pattern: "./missing",
		},
		{
			name:    "no non-test Go source files",
			dir:     moduleDir,
			pattern: "./testonly",
		},
		{
			name:    "missing base directory",
			dir:     filepath.Join(moduleDir, "missing"),
			pattern: ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ResolvePackageDir(tt.dir, tt.pattern)
			require.False(t, ok)
			require.Empty(t, got)
		})
	}
}

func setupResolvePackageDirModule(t *testing.T) string {
	t.Helper()

	moduleDir := t.TempDir()
	writeResolvePackageDirFile(t, moduleDir, "go.mod", "module example.com/schemagen\n\ngo 1.20\n")
	writeResolvePackageDirFile(t, moduleDir, "root.go", "package schemagen\n")
	writeResolvePackageDirFile(t, moduleDir, filepath.Join("receiver", "foo", "config.go"), "package foo\n")
	writeResolvePackageDirFile(t, moduleDir, filepath.Join("testonly", "config_test.go"), "package testonly\n")
	require.NoError(t, os.MkdirAll(filepath.Join(moduleDir, "cmd", "tool"), 0o700))

	return moduleDir
}

func writeResolvePackageDirFile(t *testing.T, root, name, data string) {
	t.Helper()

	path := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(data), 0o600))
}

func resolvePath(t *testing.T, path string) string {
	t.Helper()

	resolved, err := filepath.EvalSymlinks(path)
	require.NoError(t, err)
	return resolved
}
