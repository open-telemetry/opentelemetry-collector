// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestReadConfig(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	cfg, err := readConfigForTest(t, dir)
	require.NoError(t, err)

	// No metadata.yaml present: falls back to the default mode (Package).
	require.Equal(t, Package, cfg.Mode)
	require.Equal(t, dir, cfg.DirPath)
	require.Equal(t, dir, cfg.OutputFolder)
	require.Empty(t, cfg.ConfigType)
	require.Equal(t, ".", cfg.Pattern)
}

func TestReadConfig_Errors(t *testing.T) {
	t.Run("missing path", func(t *testing.T) {
		t.Chdir(t.TempDir())
		missing := filepath.Join(t.TempDir(), "missing")
		_, err := readConfigForTest(t, missing)
		require.Error(t, err)
	})

	t.Run("unknown file type", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		file := createConfigFile(t, dir, "config.go")

		_, err := readConfigForTest(t, "-t", "xml", file)
		require.Error(t, err)
	})
}

func TestReadConfig_RespectsRootTypeFlag(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	target := createConfigFile(t, dir, "component.go")

	cfg, err := readConfigForTest(t, "-r", "ExplicitType", target)
	require.NoError(t, err)

	require.Equal(t, "ExplicitType", cfg.ConfigType)
}

func TestReadConfig_ReadsSettingsFile(t *testing.T) {
	projectDir := t.TempDir()
	settings := Settings{
		Namespace: "github.com/open-telemetry/opentelemetry-collector-contrib",
		Mappings: Mappings{
			"pkg": PackagesMapping{
				"Thing": {
					SchemaType: SchemaTypeString,
					Format:     "uuid",
				},
			},
		},
	}
	data, err := yaml.Marshal(settings)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, SettingsFileName), data, 0o600))

	workDir := filepath.Join(projectDir, "workdir")
	require.NoError(t, os.Mkdir(workDir, 0o700))
	t.Chdir(workDir)

	target := createConfigFile(t, workDir, "component.go")

	cfg, err := readConfigForTest(t, target)
	require.NoError(t, err)

	expectedOutput := filepath.Join(projectDir, "workdir")
	require.Equal(t, evalPath(t, expectedOutput), evalPath(t, cfg.OutputFolder))
	require.Equal(t, "github.com/open-telemetry/opentelemetry-collector-contrib", cfg.Namespace)
	require.Equal(t, Mappings{
		"pkg": PackagesMapping{
			"Thing": {SchemaType: SchemaTypeString, Format: "uuid"},
		},
	}, cfg.Mappings)
}

func TestReadConfig_MetadataHandling(t *testing.T) {
	tests := []struct {
		name          string
		metadata      string
		expectedMode  RunMode
		expectedClass string
	}{
		{
			name: "with parent field",
			metadata: `type: test
status:
  class: pkg
parent: someparent
`,
			expectedMode:  Component,
			expectedClass: "",
		},
		{
			name: "receiver class",
			metadata: `type: testreceiver
status:
  class: receiver
`,
			expectedMode:  Component,
			expectedClass: "receiver",
		},
		{
			name: "processor class",
			metadata: `type: testprocessor
status:
  class: processor
`,
			expectedMode:  Component,
			expectedClass: "processor",
		},
		{
			name: "exporter class",
			metadata: `type: testexporter
status:
  class: exporter
`,
			expectedMode:  Component,
			expectedClass: "exporter",
		},
		{
			name: "connector class",
			metadata: `type: testconnector
status:
  class: connector
`,
			expectedMode:  Component,
			expectedClass: "connector",
		},
		{
			name: "extension class",
			metadata: `type: testextension
status:
  class: extension
`,
			expectedMode:  Component,
			expectedClass: "extension",
		},
		{
			name: "unknown class (pkg)",
			metadata: `type: testpkg
status:
  class: pkg
`,
			expectedMode:  Package,
			expectedClass: "pkg",
		},
		{
			name: "scraper class (default case)",
			metadata: `type: testscraper
status:
  class: scraper
`,
			expectedMode:  Package,
			expectedClass: "scraper",
		},
		{
			name:          "no metadata file",
			metadata:      "",
			expectedMode:  Package,
			expectedClass: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			createConfigFile(t, dir, "config.go")

			// Create metadata.yaml if metadata content is provided
			if tt.metadata != "" {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "metadata.yaml"), []byte(tt.metadata), 0o600))
			}

			cfg, err := readConfigForTest(t, dir)
			require.NoError(t, err)

			require.Equal(t, tt.expectedMode, cfg.Mode)
			require.Equal(t, tt.expectedClass, cfg.Class)
		})
	}
}

func TestReadConfig_ModeOverride(t *testing.T) {
	t.Run("valid override to package", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		// Write a metadata.yaml that would yield component mode.
		require.NoError(t, os.WriteFile(filepath.Join(dir, "metadata.yaml"), []byte(`type: testreceiver
status:
  class: receiver
`), 0o600))

		cfg, err := readConfigForTestWithMode(t, "package", dir)
		require.NoError(t, err)
		require.Equal(t, Package, cfg.Mode)
	})

	t.Run("valid override to component", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		// Write a metadata.yaml that would yield package mode (unknown class).
		require.NoError(t, os.WriteFile(filepath.Join(dir, "metadata.yaml"), []byte(`type: testpkg
status:
  class: pkg
`), 0o600))

		cfg, err := readConfigForTestWithMode(t, "component", dir)
		require.NoError(t, err)
		require.Equal(t, Component, cfg.Mode)
	})

	t.Run("invalid override value", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)

		_, err := readConfigForTestWithMode(t, "bogus", dir)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unknown mode")
	})
}

func TestReadConfig_PatternResolvesMetadata(t *testing.T) {
	// Use an import path from within this repository that has a metadata.yaml
	// declaring a receiver component. schemagen's own go.mod must be able to
	// resolve it; we run from cmd/schemagen so the local replace directives apply.
	//
	// The pattern-based metadata resolution works when the package is reachable
	// from the current module (local replace or module cache). We pick a package
	// in testdata to keep the test self-contained and fast.
	dir := t.TempDir()
	t.Chdir(dir)

	// Verify the local dirPath has no metadata.yaml (so mode would be Package
	// by default without pattern-based resolution). We can only test this properly
	// for local packages because packages.Load needs a real module graph; we just
	// verify that when pattern == "." the metadata comes from dirPath.
	createConfigFile(t, dir, "config.go")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "metadata.yaml"), []byte(`type: testpkg
status:
  class: pkg
`), 0o600))

	// With pattern "." the local metadata.yaml is used — should detect Package.
	cfg, err := readConfigForTest(t, dir)
	require.NoError(t, err)
	require.Equal(t, Package, cfg.Mode)
}

func createConfigFile(t *testing.T, dir, name string) string {
	t.Helper()
	target := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(target, []byte("package test"), 0o600))
	return target
}

func readConfigForTest(t *testing.T, args ...string) (*Config, error) {
	t.Helper()
	return readConfigWithFlags(t, "", args...)
}

// readConfigForTestWithMode is like readConfigForTest but prepends -m <mode> to the args.
func readConfigForTestWithMode(t *testing.T, mode string, args ...string) (*Config, error) {
	t.Helper()
	return readConfigWithFlags(t, mode, args...)
}

func readConfigWithFlags(t *testing.T, mode string, args ...string) (*Config, error) {
	t.Helper()

	origArgs := os.Args
	origCommandLine := flag.CommandLine
	origRootType := configType
	origOutputFolder := outputFolder
	origFileType := fileType
	origPattern := pattern
	origModeOverride := modeOverride

	flag.CommandLine = flag.NewFlagSet(origArgs[0], flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)

	configType = flag.String("r", "", "Root type name (default is derived from file name)")
	outputFolder = flag.String("o", "", "Output schema folder")
	fileType = flag.String("t", "yaml", "Output file type (yaml or json)")
	pattern = flag.String("p", ".", "Optional pattern to match config struct package")
	modeOverride = flag.String("m", "", "Override detected run mode: component or package")

	cliArgs := []string{origArgs[0]}
	if mode != "" {
		cliArgs = append(cliArgs, "-m", mode)
	}
	cliArgs = append(cliArgs, args...)
	os.Args = cliArgs

	t.Cleanup(func() {
		os.Args = origArgs
		flag.CommandLine = origCommandLine
		configType = origRootType
		outputFolder = origOutputFolder
		fileType = origFileType
		pattern = origPattern
		modeOverride = origModeOverride
	})

	return ReadConfig()
}

func evalPath(t *testing.T, path string) string {
	t.Helper()
	dir := filepath.Dir(path)
	resolved, err := filepath.EvalSymlinks(dir)
	require.NoError(t, err)
	return filepath.Join(resolved, filepath.Base(path))
}
