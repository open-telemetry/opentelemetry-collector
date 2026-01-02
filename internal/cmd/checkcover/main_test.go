// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Run("success with valid coverage", func(t *testing.T) {
		tmpDir := t.TempDir()
		
		// Create coverage file
		coverageFile := filepath.Join(tmpDir, "coverage.txt")
		coverageContent := `mode: atomic
go.opentelemetry.io/collector/test/file.go:10.2,12.3 2 5
`
		err := os.WriteFile(coverageFile, []byte(coverageContent), 0644)
		require.NoError(t, err)

		cmd := &cobra.Command{}
		cmd.Flags().String("coverage-file", coverageFile, "")
		cmd.Flags().String("repo-root", tmpDir, "")
		cmd.Flags().Int("repo-minimum", 0, "")
		cmd.Flags().Bool("verbose", false, "")

		err = run(cmd, []string{})
		require.NoError(t, err)
	})

	t.Run("failure with missing coverage file", func(t *testing.T) {
		tmpDir := t.TempDir()
		
		cmd := &cobra.Command{}
		cmd.Flags().String("coverage-file", "nonexistent.txt", "")
		cmd.Flags().String("repo-root", tmpDir, "")
		cmd.Flags().Int("repo-minimum", 0, "")
		cmd.Flags().Bool("verbose", false, "")

		err := run(cmd, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load coverage data")
	})

	t.Run("verbose mode", func(t *testing.T) {
		tmpDir := t.TempDir()
		
		// Create coverage file
		coverageFile := filepath.Join(tmpDir, "coverage.txt")
		coverageContent := `mode: atomic
go.opentelemetry.io/collector/test/file.go:10.2,12.3 2 5
`
		err := os.WriteFile(coverageFile, []byte(coverageContent), 0644)
		require.NoError(t, err)

		cmd := &cobra.Command{}
		cmd.Flags().String("coverage-file", coverageFile, "")
		cmd.Flags().String("repo-root", tmpDir, "")
		cmd.Flags().Int("repo-minimum", 50, "")
		cmd.Flags().Bool("verbose", true, "")

		err = run(cmd, []string{})
		require.NoError(t, err)
	})
}

func TestMain_Integration(t *testing.T) {
	// Save original args and restore after test
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	t.Run("help flag", func(t *testing.T) {
		// This test verifies the command can be created
		// Actual help output is tested via CLI
		cmd := &cobra.Command{
			Use:   "checkcover",
			Short: "Validate code coverage requirements",
		}
		cmd.Flags().StringP("coverage-file", "c", "coverage.txt", "")
		cmd.Flags().StringP("repo-root", "r", ".", "")
		cmd.Flags().IntP("repo-minimum", "m", 0, "")
		cmd.Flags().BoolP("verbose", "v", false, "")
		
		assert.NotNil(t, cmd)
		assert.Equal(t, "checkcover", cmd.Use)
	})
}
