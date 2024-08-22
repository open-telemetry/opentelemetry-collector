// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/mod/modfile"
)

const modulePrefix = "go.opentelemetry.io/collector"

var (
	replaceModules = []string{
		"",
		"/component",
		"/component/componentprofiles",
		"/component/componentstatus",
		"/client",
		"/config/configauth",
		"/config/configcompression",
		"/config/configgrpc",
		"/config/confighttp",
		"/config/confignet",
		"/config/configopaque",
		"/config/configretry",
		"/config/configtelemetry",
		"/config/configtls",
		"/config/internal",
		"/confmap",
		"/confmap/provider/envprovider",
		"/confmap/provider/fileprovider",
		"/confmap/provider/httpprovider",
		"/confmap/provider/httpsprovider",
		"/confmap/provider/yamlprovider",
		"/consumer",
		"/consumer/consumerprofiles",
		"/consumer/consumertest",
		"/connector",
		"/exporter",
		"/exporter/debugexporter",
		"/exporter/nopexporter",
		"/exporter/otlpexporter",
		"/exporter/otlphttpexporter",
		"/extension",
		"/extension/auth",
		"/extension/zpagesextension",
		"/featuregate",
		"/internal/globalgates",
		"/processor",
		"/processor/batchprocessor",
		"/processor/memorylimiterprocessor",
		"/receiver",
		"/receiver/nopreceiver",
		"/receiver/otlpreceiver",
		"/otelcol",
		"/pdata",
		"/pdata/testdata",
		"/pdata/pprofile",
		"/semconv",
		"/service",
	}
)

func newTestConfig() Config {
	cfg := NewDefaultConfig()
	cfg.downloadModules.wait = 0
	cfg.downloadModules.numRetries = 1
	return cfg
}

func newInitializedConfig(t *testing.T) Config {
	cfg := newTestConfig()
	// Validate and ParseModules will be called before the config is
	// given to Generate.
	assert.NoError(t, cfg.Validate())
	assert.NoError(t, cfg.SetBackwardsCompatibility())
	assert.NoError(t, cfg.ParseModules())

	return cfg
}

func TestVersioning(t *testing.T) {
	replaces := generateReplaces()
	tests := []struct {
		description string
		cfgBuilder  func() Config
		expectedErr error
	}{
		{
			description: "defaults",
			cfgBuilder: func() Config {
				cfg := newTestConfig()
				cfg.Distribution.Go = "go"
				cfg.Replaces = append(cfg.Replaces, replaces...)
				return cfg
			},
			expectedErr: nil,
		},
		{
			description: "require otelcol",
			cfgBuilder: func() Config {
				cfg := newTestConfig()
				cfg.Distribution.Go = "go"
				cfg.Distribution.RequireOtelColModule = true
				cfg.Replaces = append(cfg.Replaces, replaces...)
				return cfg
			},
			expectedErr: nil,
		},
		{
			description: "only gomod file, skip generate",
			cfgBuilder: func() Config {
				cfg := newTestConfig()
				tempDir := t.TempDir()
				err := makeModule(tempDir, []byte(goModTestFile))
				require.NoError(t, err)
				cfg.Distribution.OutputPath = tempDir
				cfg.SkipGenerate = true
				cfg.Distribution.Go = "go"
				return cfg
			},
			expectedErr: ErrDepNotFound,
		},
		{
			description: "old otel version",
			cfgBuilder: func() Config {
				cfg := newTestConfig()
				cfg.Verbose = true
				cfg.Distribution.Go = "go"
				cfg.Distribution.OtelColVersion = "0.97.0"
				cfg.Distribution.RequireOtelColModule = true
				var err error
				cfg.Exporters, err = parseModules([]Module{
					{
						GoMod: "go.opentelemetry.io/collector/exporter/otlpexporter v0.97.0",
					},
				})
				require.NoError(t, err)
				cfg.Receivers, err = parseModules([]Module{
					{
						GoMod: "go.opentelemetry.io/collector/receiver/otlpreceiver v0.97.0",
					},
				})
				require.NoError(t, err)
				providers, err := parseModules([]Module{
					{
						GoMod: "go.opentelemetry.io/collector/confmap/provider/envprovider v0.97.0",
					},
				})
				require.NoError(t, err)
				cfg.Providers = &providers
				return cfg
			},
			expectedErr: nil,
		},
		{
			description: "old component version",
			cfgBuilder: func() Config {
				cfg := newTestConfig()
				cfg.Distribution.Go = "go"
				cfg.Exporters = []Module{
					{
						GoMod: "go.opentelemetry.io/collector/exporter/otlpexporter v0.97.0",
					},
				}
				cfg.Providers = &[]Module{}
				cfg.Replaces = append(cfg.Replaces, replaces...)
				return cfg
			},
			expectedErr: nil,
		},
		{
			description: "old component version without strict mode",
			cfgBuilder: func() Config {
				cfg := newTestConfig()
				cfg.Distribution.Go = "go"
				cfg.SkipStrictVersioning = true
				cfg.Exporters = []Module{
					{
						GoMod: "go.opentelemetry.io/collector/exporter/otlpexporter v0.97.0",
					},
				}
				cfg.Providers = &[]Module{}
				cfg.Replaces = append(cfg.Replaces, replaces...)
				return cfg
			},
			expectedErr: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			cfg := tc.cfgBuilder()
			require.NoError(t, cfg.SetBackwardsCompatibility())
			require.NoError(t, cfg.Validate())
			require.NoError(t, cfg.ParseModules())
			err := GenerateAndCompile(cfg)
			require.ErrorIs(t, err, tc.expectedErr)
		})
	}
}

func TestSkipGenerate(t *testing.T) {
	cfg := newInitializedConfig(t)
	cfg.Distribution.OutputPath = t.TempDir()
	cfg.SkipGenerate = true
	err := Generate(cfg)
	require.NoError(t, err)
	outputFile, err := os.Open(filepath.Clean(cfg.Distribution.OutputPath))
	defer func() {
		require.NoError(t, outputFile.Close())
	}()
	require.NoError(t, err)
	_, err = outputFile.Readdirnames(1)
	require.ErrorIs(t, err, io.EOF, "skip generate should leave output directory empty")
}

func TestGenerateAndCompile(t *testing.T) {
	replaces := generateReplaces()
	type testDesc struct {
		testCase    string
		cfgBuilder  func(t *testing.T) Config
		verifyFiles func(t *testing.T, dir string)
		expectedErr string
	}
	testCases := []testDesc{
		{
			testCase: "Default Configuration Compilation",
			cfgBuilder: func(t *testing.T) Config {
				cfg := newTestConfig()
				err := cfg.SetBackwardsCompatibility()
				require.NoError(t, err)
				cfg.Distribution.OutputPath = t.TempDir()
				cfg.Replaces = append(cfg.Replaces, replaces...)
				return cfg
			},
		}, {
			testCase: "Skip New Gomod Configuration Compilation",
			cfgBuilder: func(t *testing.T) Config {
				cfg := newTestConfig()
				err := cfg.SetBackwardsCompatibility()
				require.NoError(t, err)
				cfg.Receivers = append(cfg.Receivers,
					Module{
						GoMod: "go.opentelemetry.io/collector/receiver/otlpreceiver v0.106.0",
					},
				)
				cfg.Exporters = append(cfg.Exporters,
					Module{
						GoMod: "go.opentelemetry.io/collector/exporter/otlpexporter v0.106.0",
					},
				)
				tempDir := t.TempDir()
				err = makeModule(tempDir, []byte(goModTestFile))
				require.NoError(t, err)
				cfg.Distribution.OutputPath = filepath.Clean(filepath.Join(tempDir, "output"))
				cfg.Replaces = nil
				cfg.Excludes = nil
				cfg.SkipNewGoModule = true
				return cfg
			},
			verifyFiles: func(t *testing.T, dir string) {
				assert.FileExists(t, filepath.Clean(filepath.Join(dir, mainTemplate.Name())))
				assert.NoFileExists(t, filepath.Clean(filepath.Join(dir, "go.mod")))
			},
		},
		{
			testCase: "Generate Only",
			cfgBuilder: func(t *testing.T) Config {
				cfg := newInitializedConfig(t)
				cfg.SkipCompilation = true
				cfg.SkipGetModules = true
				return cfg
			},
		},
		{
			testCase: "Skip Everything",
			cfgBuilder: func(_ *testing.T) Config {
				cfg := NewDefaultConfig()
				cfg.Replaces = nil
				cfg.Excludes = nil
				cfg.SkipCompilation = true
				cfg.SkipGenerate = true
				cfg.SkipGetModules = true
				cfg.SkipNewGoModule = true
				return cfg
			},
			verifyFiles: func(t *testing.T, dir string) {
				// gosec linting error: G304 Potential file inclusion via variable
				// we are setting the dir
				outputFile, err := os.Open(dir) //nolint:gosec
				defer func() {
					require.NoError(t, outputFile.Close())
				}()
				require.NoError(t, err)
				_, err = outputFile.Readdirnames(1)
				require.ErrorIs(t, err, io.EOF, "skip generate should leave output directory empty")
			},
		},
		{
			testCase: "LDFlags Compilation",
			cfgBuilder: func(t *testing.T) Config {
				cfg := newTestConfig()
				err := cfg.SetBackwardsCompatibility()
				require.NoError(t, err)
				cfg.Distribution.OutputPath = t.TempDir()
				cfg.Replaces = append(cfg.Replaces, replaces...)
				cfg.LDFlags = `-X "test.gitVersion=0743dc6c6411272b98494a9b32a63378e84c34da" -X "test.gitTag=local-testing" -X "test.goVersion=go version go1.20.7 darwin/amd64"`
				return cfg
			},
		},
		{
			testCase: "Debug Compilation",
			cfgBuilder: func(t *testing.T) Config {
				cfg := newTestConfig()
				err := cfg.SetBackwardsCompatibility()
				require.NoError(t, err)
				cfg.Distribution.OutputPath = t.TempDir()
				cfg.Replaces = append(cfg.Replaces, replaces...)
				cfg.Logger = zap.NewNop()
				cfg.Distribution.DebugCompilation = true
				return cfg
			},
		},
		{
			testCase: "No providers",
			cfgBuilder: func(t *testing.T) Config {
				cfg := newTestConfig()
				err := cfg.SetBackwardsCompatibility()
				require.NoError(t, err)
				cfg.Distribution.OutputPath = t.TempDir()
				cfg.Replaces = append(cfg.Replaces, replaces...)
				cfg.Providers = &[]Module{}
				return cfg
			},
		},
		{
			testCase: "Pre-confmap factories",
			cfgBuilder: func(t *testing.T) Config {
				cfg := newTestConfig()
				err := cfg.SetBackwardsCompatibility()
				require.NoError(t, err)
				cfg.Distribution.OutputPath = t.TempDir()
				cfg.Replaces = append(cfg.Replaces, replaces...)
				cfg.Distribution.OtelColVersion = "0.98.0"
				cfg.SkipStrictVersioning = true
				return cfg
			},
		},
		{
			testCase: "With confmap factories",
			cfgBuilder: func(t *testing.T) Config {
				cfg := newTestConfig()
				err := cfg.SetBackwardsCompatibility()
				require.NoError(t, err)
				cfg.Distribution.OutputPath = t.TempDir()
				cfg.Replaces = append(cfg.Replaces, replaces...)
				cfg.Distribution.OtelColVersion = "0.99.0"
				cfg.SkipStrictVersioning = true
				return cfg
			},
		},
		{
			testCase: "ConfResolverDefaultURIScheme set",
			cfgBuilder: func(t *testing.T) Config {
				cfg := newTestConfig()
				err := cfg.SetBackwardsCompatibility()
				require.NoError(t, err)
				cfg.ConfResolver = ConfResolver{
					DefaultURIScheme: "env",
				}
				cfg.Distribution.OutputPath = t.TempDir()
				cfg.Replaces = append(cfg.Replaces, replaces...)
				return cfg
			},
		},
		{
			testCase: "Invalid Output Path",
			cfgBuilder: func(t *testing.T) Config {
				cfg := newInitializedConfig(t)
				cfg.Distribution.OutputPath = ":/invalid"
				return cfg
			},
			expectedErr: "failed to create output path",
		},
		{
			testCase: "Malformed Receiver",
			cfgBuilder: func(t *testing.T) Config {
				cfg := NewDefaultConfig()
				cfg.Receivers = append(cfg.Receivers,
					Module{
						Name:  "missing version",
						GoMod: "go.opentelemetry.io/collector/cmd/builder/unittests",
					},
				)
				tempDir := t.TempDir()
				err := makeModule(tempDir, []byte(goModTestFile))
				require.NoError(t, err)
				cfg.Distribution.OutputPath = filepath.Clean(filepath.Join(tempDir, "output"))
				cfg.Replaces = nil
				cfg.Excludes = nil
				cfg.SkipNewGoModule = true
				return cfg
			},
			expectedErr: "ill-formatted modspec",
		},
	}

	// file permissions don't work the same on windows systems, so this test always passes.
	if runtime.GOOS != "windows" {
		testCases = append(testCases, testDesc{
			testCase: "No Dir Permissions",
			cfgBuilder: func(t *testing.T) Config {
				cfg := newTestConfig()
				err := cfg.SetBackwardsCompatibility()
				require.NoError(t, err)
				cfg.Distribution.OutputPath = t.TempDir()
				assert.NoError(t, os.Chmod(cfg.Distribution.OutputPath, 0400))
				cfg.Replaces = append(cfg.Replaces, replaces...)
				return cfg
			},
			expectedErr: "failed to generate source file",
		})
	}

	for _, tt := range testCases {
		t.Run(tt.testCase, func(t *testing.T) {
			cfg := tt.cfgBuilder(t)
			assert.NoError(t, cfg.Validate())
			assert.NoError(t, cfg.SetGoPath())
			assert.NoError(t, cfg.ParseModules())
			err := GenerateAndCompile(cfg)
			if len(tt.expectedErr) == 0 {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.expectedErr)
			}
			if tt.verifyFiles != nil {
				tt.verifyFiles(t, cfg.Distribution.OutputPath)
			}
		})
	}
}

func TestGetModules(t *testing.T) {
	testCases := []struct {
		description string
		cfgBuilder  func(t *testing.T) Config
		expectedErr string
	}{
		{
			description: "Skip New Gomod Success",
			cfgBuilder: func(t *testing.T) Config {
				cfg := newTestConfig()
				cfg.Distribution.Go = "go"
				tempDir := t.TempDir()
				require.NoError(t, makeModule(tempDir, []byte(goModTestFile)))
				outputDir := filepath.Clean(filepath.Join(tempDir, "output"))
				cfg.Distribution.OutputPath = outputDir
				cfg.Replaces = nil
				cfg.Excludes = nil
				cfg.SkipNewGoModule = true
				return cfg
			},
		},
		{
			description: "Core Version Mismatch",
			cfgBuilder: func(t *testing.T) Config {
				cfg := newTestConfig()
				cfg.Distribution.Go = "go"
				cfg.Distribution.OtelColVersion = "0.100.0"
				tempDir := t.TempDir()
				require.NoError(t, makeModule(tempDir, []byte(goModTestFile)))
				outputDir := filepath.Clean(filepath.Join(tempDir, "output"))
				cfg.Distribution.OutputPath = outputDir
				cfg.Replaces = nil
				cfg.Excludes = nil
				cfg.SkipNewGoModule = true
				return cfg
			},
			expectedErr: ErrVersionMismatch.Error(),
		},
		{
			description: "No Go Distribution",
			cfgBuilder: func(_ *testing.T) Config {
				cfg := NewDefaultConfig()
				cfg.downloadModules.wait = 0
				return cfg
			},
			expectedErr: "failed to update go.mod",
		},
		{
			description: "Invalid Dependency",
			cfgBuilder: func(t *testing.T) Config {
				cfg := NewDefaultConfig()
				cfg.downloadModules.wait = 0
				cfg.Distribution.Go = "go"
				tempDir := t.TempDir()
				require.NoError(t, makeModule(tempDir, []byte(invalidDependencyGoMod)))
				outputDir := filepath.Clean(filepath.Join(tempDir, "output"))
				cfg.Distribution.OutputPath = outputDir
				cfg.Replaces = nil
				cfg.Excludes = nil
				cfg.SkipNewGoModule = true
				return cfg
			},
			expectedErr: "failed to update go.mod",
		},
		{
			description: "Malformed Go Mod",
			cfgBuilder: func(t *testing.T) Config {
				cfg := NewDefaultConfig()
				cfg.downloadModules.wait = 0
				cfg.Distribution.Go = "go"
				tempDir := t.TempDir()
				require.NoError(t, makeModule(tempDir, []byte(malformedGoMod)))
				outputDir := filepath.Clean(filepath.Join(tempDir, "output"))
				cfg.Distribution.OutputPath = outputDir
				cfg.Replaces = nil
				cfg.Excludes = nil
				cfg.SkipNewGoModule = true
				return cfg
			},
			expectedErr: "go subcommand failed with args '[mod edit -print]'",
		},
		{
			description: "Receiver Version Mismatch - Configured Lower",
			cfgBuilder: func(t *testing.T) Config {
				cfg := NewDefaultConfig()
				cfg.Distribution.Go = "go"
				cfg.Receivers = append(cfg.Receivers,
					Module{
						GoMod: "go.opentelemetry.io/collector/receiver/otlpreceiver v0.105.0",
					},
				)
				tempDir := t.TempDir()
				err := makeModule(tempDir, []byte(goModTestFile))
				require.NoError(t, err)
				cfg.Distribution.OutputPath = filepath.Clean(filepath.Join(tempDir, "output"))
				cfg.Replaces = nil
				cfg.Excludes = nil
				cfg.SkipNewGoModule = true
				return cfg
			},
			expectedErr: ErrVersionMismatch.Error(),
		},
		{
			description: "Receiver Version Mismatch - Configured Higher",
			cfgBuilder: func(t *testing.T) Config {
				cfg := NewDefaultConfig()
				cfg.Distribution.Go = "go"
				cfg.Receivers = append(cfg.Receivers,
					Module{
						GoMod: "go.opentelemetry.io/collector/receiver/otlpreceiver v0.106.1",
					},
				)
				tempDir := t.TempDir()
				err := makeModule(tempDir, []byte(goModTestFile))
				require.NoError(t, err)
				cfg.Distribution.OutputPath = filepath.Clean(filepath.Join(tempDir, "output"))
				cfg.Replaces = nil
				cfg.Excludes = nil
				cfg.SkipNewGoModule = true
				return cfg
			},
		},
		{
			description: "Exporter Not in Gomod",
			cfgBuilder: func(t *testing.T) Config {
				cfg := NewDefaultConfig()
				cfg.Distribution.Go = "go"
				cfg.Exporters = append(cfg.Exporters,
					Module{
						GoMod: "go.opentelemetry.io/collector/exporter/otlpexporter v0.106.0",
					},
				)
				tempDir := t.TempDir()
				err := makeModule(tempDir, []byte(goModTestFile))
				require.NoError(t, err)
				cfg.Distribution.OutputPath = filepath.Clean(filepath.Join(tempDir, "output"))
				cfg.Replaces = nil
				cfg.Excludes = nil
				cfg.SkipNewGoModule = true
				return cfg
			},
		},
		{
			description: "Receiver Nonexistent Version",
			cfgBuilder: func(t *testing.T) Config {
				cfg := NewDefaultConfig()
				cfg.Distribution.Go = "go"
				cfg.Receivers = append(cfg.Receivers,
					Module{
						GoMod: "go.opentelemetry.io/collector/receiver/otlpreceiver v0.106.2",
					},
				)
				tempDir := t.TempDir()
				err := makeModule(tempDir, []byte(goModTestFile))
				require.NoError(t, err)
				cfg.Distribution.OutputPath = filepath.Clean(filepath.Join(tempDir, "output"))
				cfg.Replaces = nil
				cfg.Excludes = nil
				cfg.SkipNewGoModule = true
				return cfg
			},
			expectedErr: "failed to update go.mod",
		},
		{
			description: "Receiver In Current Module",
			cfgBuilder: func(t *testing.T) Config {
				cfg := NewDefaultConfig()
				cfg.Distribution.Go = "go"
				cfg.Receivers = append(cfg.Receivers,
					Module{
						GoMod: "go.opentelemetry.io/collector/cmd/builder/unittests v0.0.0",
					},
				)
				tempDir := t.TempDir()
				err := makeModule(tempDir, []byte(goModTestFile))
				require.NoError(t, err)
				cfg.Distribution.OutputPath = filepath.Clean(filepath.Join(tempDir, "output"))
				cfg.Replaces = nil
				cfg.Excludes = nil
				cfg.SkipNewGoModule = true
				return cfg
			},
			expectedErr: "failed to update go.mod",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			cfg := tc.cfgBuilder(t)
			require.NoError(t, cfg.SetBackwardsCompatibility())
			require.NoError(t, cfg.Validate())
			require.NoError(t, cfg.ParseModules())
			// GenerateAndCompile calls GetModules().  We want to call Generate()
			// first so our dependencies stay in the gomod after go mod tidy.
			err := GenerateAndCompile(cfg)
			if len(tc.expectedErr) == 0 {
				if !assert.NoError(t, err) {
					mf, mvm, readErr := cfg.readGoModFile()
					t.Log("go mod file", mf, mvm, readErr)
				}
				return
			}
			assert.ErrorContains(t, err, tc.expectedErr)
		})
	}
}

// Test that the go.mod files that other tests in this file
// may generate have all their modules covered by our
// "replace" statements created in `generateReplaces`.
//
// An incomplete set of replace statements in these tests
// may cause them to fail during the release process, when
// the local version of modules in the release branch is
// not yet available on the Go package repository.
// Unless the replace statements route all modules to the
// local copy, `go get` will try to fetch the unreleased
// version remotely and some tests will fail.
func TestReplaceStatementsAreComplete(t *testing.T) {
	workspaceDir := getWorkspaceDir()
	replaceMods := map[string]bool{}

	for _, suffix := range replaceModules {
		replaceMods[modulePrefix+suffix] = false
	}

	for _, mod := range replaceModules {
		verifyGoMod(t, workspaceDir+mod, replaceMods)
	}

	var err error
	dir := t.TempDir()
	cfg := NewDefaultConfig()
	cfg.Distribution.Go = "go"
	cfg.Distribution.OutputPath = dir
	// Use a deliberately nonexistent version to simulate an unreleased
	// version of the package. Not strictly necessary since this test
	// will catch gaps in the replace statements before a release is in
	// progress.
	cfg.Distribution.OtelColVersion = "1.9999.9999"
	cfg.Replaces = append(cfg.Replaces, generateReplaces()...)
	// Configure all components that we want to use elsewhere in these tests.
	// This ensures the resulting go.mod file has maximum coverage of modules
	// that exist in the Core repository.
	cfg.Exporters, err = parseModules([]Module{
		{
			GoMod: "go.opentelemetry.io/collector/exporter/debugexporter v1.9999.9999",
		},
		{
			GoMod: "go.opentelemetry.io/collector/exporter/nopexporter v1.9999.9999",
		},
		{
			GoMod: "go.opentelemetry.io/collector/exporter/otlpexporter v1.9999.9999",
		},
		{
			GoMod: "go.opentelemetry.io/collector/exporter/otlphttpexporter v1.9999.9999",
		},
	})
	require.NoError(t, err)
	cfg.Receivers, err = parseModules([]Module{
		{
			GoMod: "go.opentelemetry.io/collector/receiver/nopreceiver v1.9999.9999",
		},
		{
			GoMod: "go.opentelemetry.io/collector/receiver/otlpreceiver v1.9999.9999",
		},
	})
	require.NoError(t, err)
	cfg.Extensions, err = parseModules([]Module{
		{
			GoMod: "go.opentelemetry.io/collector/extension/zpagesextension v1.9999.9999",
		},
	})
	require.NoError(t, err)
	cfg.Processors, err = parseModules([]Module{
		{
			GoMod: "go.opentelemetry.io/collector/processor/batchprocessor v1.9999.9999",
		},
		{
			GoMod: "go.opentelemetry.io/collector/processor/memorylimiterprocessor v1.9999.9999",
		},
	})
	require.NoError(t, err)

	require.NoError(t, cfg.SetBackwardsCompatibility())
	require.NoError(t, cfg.Validate())
	require.NoError(t, cfg.ParseModules())
	err = GenerateAndCompile(cfg)
	require.NoError(t, err)

	verifyGoMod(t, dir, replaceMods)

	for k, v := range replaceMods {
		assert.Truef(t, v, "Module not used: %s", k)
	}
}

func verifyGoMod(t *testing.T, dir string, replaceMods map[string]bool) {
	gomodpath := path.Join(dir, "go.mod")
	// #nosec G304 We control this path and generate the file inside, so we can assume it is safe.
	gomod, err := os.ReadFile(gomodpath)
	require.NoError(t, err)

	mod, err := modfile.Parse(gomodpath, gomod, nil)
	require.NoError(t, err)

	for _, req := range mod.Require {
		if !strings.HasPrefix(req.Mod.Path, modulePrefix) {
			continue
		}

		_, ok := replaceMods[req.Mod.Path]
		assert.Truef(t, ok, "Module missing from replace statements list: %s", req.Mod.Path)

		replaceMods[req.Mod.Path] = true
	}
}

func makeModule(dir string, fileContents []byte) error {
	// if the file does not exist, try to create it
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.Mkdir(dir, 0750); err != nil {
			return fmt.Errorf("failed to create output path: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to create output path: %w", err)
	}

	err := os.WriteFile(filepath.Clean(filepath.Join(dir, "go.mod")), fileContents, 0600)
	if err != nil {
		return fmt.Errorf("failed to write go.mod file: %w", err)
	}
	return nil
}

func generateReplaces() []string {
	workspaceDir := getWorkspaceDir()
	modules := replaceModules
	replaces := make([]string, len(modules))

	for i, mod := range modules {
		replaces[i] = fmt.Sprintf("%s%s => %s%s", modulePrefix, mod, workspaceDir, mod)
	}

	return replaces
}

func getWorkspaceDir() string {
	// This is dependent on the current file structure.
	// The goal is find the root of the repo so we can replace the root module.
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))))
}
