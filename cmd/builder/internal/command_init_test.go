// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/builder/internal"

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/cmd/builder/internal/builder"
	"go.opentelemetry.io/collector/cmd/builder/internal/fromconfig"
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

			err := run(path, initOptions{})

			if tt.wantErr == "" {
				require.NoError(t, err)
				validateCollector(t, path)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestBuildManifest(t *testing.T) {
	dir := t.TempDir()
	meta := metadata{
		Name:        "myCollector",
		Description: defaultDescription,
		BetaVersion: builder.DefaultBetaOtelColVersion,
	}
	require.NoError(t, buildManifest(dir, meta, nil))

	content, err := os.ReadFile(filepath.Join(dir, "manifest.yaml")) //nolint:gosec // G304: path is test-controlled
	require.NoError(t, err)

	expected := `dist:
    description: Custom OpenTelemetry Collector
    name: myCollector
    output_path: ./build/collector
exporters:
    - gomod: go.opentelemetry.io/collector/exporter/otlpexporter ` + builder.DefaultBetaOtelColVersion + `
receivers:
    - gomod: go.opentelemetry.io/collector/receiver/otlpreceiver ` + builder.DefaultBetaOtelColVersion + `
`

	assert.Equal(t, expected, string(content))
}

const (
	corePrefix    = "go.opentelemetry.io/collector/"
	contribPrefix = "github.com/open-telemetry/opentelemetry-collector-contrib/"

	fromConfigExample = `receivers:
  otlp:
  hostmetrics:
  prometheus_simple:
processors:
  batch:
exporters:
  debug:
extensions:
  file_storage:
  myvendor:
service:
  extensions: [file_storage]
`
)

// fakeResolver resolves components against a fixed set of published modules.
func fakeResolver() *fromconfig.Resolver {
	published := map[string][]string{
		corePrefix + "receiver/otlpreceiver":                {"v0.1.0"},
		corePrefix + "processor/batchprocessor":             {"v0.1.0"},
		corePrefix + "exporter/debugexporter":               {"v0.1.0"},
		contribPrefix + "receiver/hostmetricsreceiver":      {"v0.1.0"},
		contribPrefix + "receiver/simpleprometheusreceiver": {"v0.1.0"},
		contribPrefix + "extension/storage/filestorage":     {"v0.1.0"},
	}
	return &fromconfig.Resolver{
		Release: "v0.1.0",
		List: func(_ context.Context, modules []string) ([]fromconfig.ListedModule, error) {
			out := make([]fromconfig.ListedModule, 0, len(modules))
			for _, m := range modules {
				out = append(out, fromconfig.ListedModule{Path: m, Versions: published[m]})
			}
			return out, nil
		},
		Versions: func(context.Context, string) (fromconfig.Versions, error) {
			return fromconfig.Versions{Beta: "v0.1.0", Stable: "v1.0.0"}, nil
		},
	}
}

func TestRunInitFromConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "collector-config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(fromConfigExample), 0o600))
	path := filepath.Join(t.TempDir(), distributionName)

	var stderr bytes.Buffer
	require.NoError(t, run(path, initOptions{fromConfig: configPath, resolver: fakeResolver(), stderr: &stderr}))
	validateCollector(t, path)

	manifest, err := os.ReadFile(filepath.Join(path, "manifest.yaml")) //nolint:gosec // G304: path is test-controlled
	require.NoError(t, err)
	assert.Equal(t, `dist:
    description: Custom OpenTelemetry Collector
    name: test-distro
    output_path: ./build/collector
exporters:
    - gomod: `+corePrefix+`exporter/debugexporter v0.1.0
extensions:
    - gomod: `+contribPrefix+`extension/storage/filestorage v0.1.0
processors:
    - gomod: `+corePrefix+`processor/batchprocessor v0.1.0
receivers:
    - gomod: `+contribPrefix+`receiver/hostmetricsreceiver v0.1.0
    - gomod: `+corePrefix+`receiver/otlpreceiver v0.1.0
    - gomod: `+contribPrefix+`receiver/simpleprometheusreceiver v0.1.0

# The following components are referenced by `+configPath+` but could
# not be matched to a Go module in the opentelemetry-collector or
# opentelemetry-collector-contrib repositories. Fill in the module path and
# version for each one and uncomment it, or remove it from your Collector
# configuration.
#
# extensions:
#     - gomod: <module path> <version> # myvendor
`, string(manifest))

	// The configuration is copied instead of the static template.
	config, err := os.ReadFile(filepath.Join(path, "config.yaml")) //nolint:gosec // G304: path is test-controlled
	require.NoError(t, err)
	assert.Equal(t, fromConfigExample, string(config))

	assert.Contains(t, stderr.String(), "ocb: resolved receiver otlp \u2192 "+corePrefix+"receiver/otlpreceiver v0.1.0\n")
	assert.Contains(t, stderr.String(), `ocb: warning: no Go module found for extension "myvendor"; add its gomod to manifest.yaml manually`+"\n")
	assert.Contains(t, stderr.String(), "ocb: resolved 6 of 7 components referenced by "+configPath+"\n")
}

func TestRunInitFromConfigAllResolved(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("receivers:\n  otlp:\n"), 0o600))
	path := filepath.Join(t.TempDir(), distributionName)

	require.NoError(t, run(path, initOptions{fromConfig: configPath, resolver: fakeResolver()}))

	manifest, err := os.ReadFile(filepath.Join(path, "manifest.yaml")) //nolint:gosec // G304: path is test-controlled
	require.NoError(t, err)
	assert.NotContains(t, string(manifest), "#")
}

func TestRunInitFromConfigErrors(t *testing.T) {
	dir := t.TempDir()
	badConfig := filepath.Join(dir, "bad.yaml")
	require.NoError(t, os.WriteFile(badConfig, []byte("receivers: ${env:FOO}\n"), 0o600))

	for _, tt := range []struct {
		name       string
		fromConfig string
		wantErr    string
	}{
		{"missing file", filepath.Join(dir, "missing.yaml"), "failed reading configuration"},
		{"unresolvable section", badConfig, "must be a mapping"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), distributionName)
			err := run(path, initOptions{fromConfig: tt.fromConfig, resolver: fakeResolver()})
			require.ErrorContains(t, err, tt.wantErr)
			// Nothing is written on failure.
			assert.NoDirExists(t, path)
		})
	}
}

func TestInitCommandFromConfigFlag(t *testing.T) {
	flag := initCommand().Flags().Lookup("from-config")
	require.NotNil(t, flag)
	assert.Empty(t, flag.DefValue)
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
