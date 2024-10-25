// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"go.opentelemetry.io/collector/cmd/builder/internal/config"
)

func TestParseModules(t *testing.T) {
	// prepare
	cfg := Config{
		Extensions: []Module{{
			GoMod: "github.com/org/repo v0.1.2",
		}},
	}

	// test
	err := cfg.ParseModules()
	require.NoError(t, err)

	// verify
	assert.Equal(t, "github.com/org/repo v0.1.2", cfg.Extensions[0].GoMod)
	assert.Equal(t, "github.com/org/repo", cfg.Extensions[0].Import)
	assert.Equal(t, "repo", cfg.Extensions[0].Name)
}

func TestRelativePath(t *testing.T) {
	// prepare
	cfg := Config{
		Extensions: []Module{{
			GoMod: "some-module",
			Path:  "./some-module",
		}},
	}

	// test
	err := cfg.ParseModules()
	require.NoError(t, err)

	// verify
	cwd, err := os.Getwd()
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(cfg.Extensions[0].Path, cwd))
}

func TestModuleFromCore(t *testing.T) {
	// prepare
	cfg := Config{
		Extensions: []Module{ // see issue-12
			{
				Import: "go.opentelemetry.io/collector/receiver/otlpreceiver",
				GoMod:  "go.opentelemetry.io/collector v0.0.0",
			},
			{
				Import: "go.opentelemetry.io/collector/receiver/otlpreceiver",
				GoMod:  "go.opentelemetry.io/collector v0.0.0",
			},
		},
	}

	// test
	err := cfg.ParseModules()
	require.NoError(t, err)

	// verify
	assert.True(t, strings.HasPrefix(cfg.Extensions[0].Name, "otlpreceiver"))
}

func TestMissingModule(t *testing.T) {
	type invalidModuleTest struct {
		cfg Config
		err error
	}
	// prepare
	configurations := []invalidModuleTest{
		{
			cfg: Config{
				Logger: zap.NewNop(),
				Providers: &[]Module{{
					Import: "invalid",
				}},
			},
			err: ErrMissingGoMod,
		},
		{
			cfg: Config{
				Logger: zap.NewNop(),
				Extensions: []Module{{
					Import: "invalid",
				}},
			},
			err: ErrMissingGoMod,
		},
		{
			cfg: Config{
				Logger: zap.NewNop(),
				Receivers: []Module{{
					Import: "invalid",
				}},
			},
			err: ErrMissingGoMod,
		},
		{
			cfg: Config{
				Logger: zap.NewNop(),
				Exporters: []Module{{
					Import: "invali",
				}},
			},
			err: ErrMissingGoMod,
		},
		{
			cfg: Config{
				Logger: zap.NewNop(),
				Processors: []Module{{
					Import: "invalid",
				}},
			},
			err: ErrMissingGoMod,
		},
		{
			cfg: Config{
				Logger: zap.NewNop(),
				Connectors: []Module{{
					Import: "invalid",
				}},
			},
			err: ErrMissingGoMod,
		},
	}

	for _, test := range configurations {
		assert.ErrorIs(t, test.cfg.Validate(), test.err)
	}
}

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()
	require.NoError(t, cfg.ParseModules())
	assert.NoError(t, cfg.Validate())
	assert.NoError(t, cfg.SetGoPath())
	require.NoError(t, cfg.Validate())
	assert.False(t, cfg.Distribution.DebugCompilation)
	assert.Empty(t, cfg.Distribution.BuildTags)
}

func TestNewBuiltinConfig(t *testing.T) {
	k := koanf.New(".")

	require.NoError(t, k.Load(config.DefaultProvider(), yaml.Parser()))

	cfg := Config{Logger: zaptest.NewLogger(t)}

	require.NoError(t, k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{Tag: "mapstructure"}))
	assert.NoError(t, cfg.ParseModules())
	assert.NoError(t, cfg.Validate())
	assert.NoError(t, cfg.SetGoPath())

	// Unlike the config initialized in NewDefaultConfig(), we expect
	// the builtin default to be practically useful, so there must be
	// a set of modules present.
	assert.NotEmpty(t, cfg.Receivers)
	assert.NotEmpty(t, cfg.Exporters)
	assert.NotEmpty(t, cfg.Extensions)
	assert.NotEmpty(t, cfg.Processors)
}

func TestSkipGoValidation(t *testing.T) {
	cfg := Config{
		Distribution: Distribution{
			Go: "invalid/go/binary/path",
		},
		SkipCompilation: true,
		SkipGetModules:  true,
	}
	assert.NoError(t, cfg.Validate())
	assert.NoError(t, cfg.SetGoPath())
}

func TestSkipGoInitialization(t *testing.T) {
	cfg := Config{
		SkipCompilation: true,
		SkipGetModules:  true,
	}
	assert.NoError(t, cfg.Validate())
	assert.NoError(t, cfg.SetGoPath())
	assert.Zero(t, cfg.Distribution.Go)
}

func TestBuildTagConfig(t *testing.T) {
	cfg := Config{
		Distribution: Distribution{
			BuildTags: "customTag",
		},
		SkipCompilation: true,
		SkipGetModules:  true,
	}
	require.NoError(t, cfg.Validate())
	assert.Equal(t, "customTag", cfg.Distribution.BuildTags)
}

func TestDebugOptionSetConfig(t *testing.T) {
	cfg := Config{
		Distribution: Distribution{
			DebugCompilation: true,
		},
		SkipCompilation: true,
		SkipGetModules:  true,
	}
	require.NoError(t, cfg.Validate())
	assert.True(t, cfg.Distribution.DebugCompilation)
}

func TestRequireOtelColModule(t *testing.T) {
	tests := []struct {
		Version                      string
		ExpectedRequireOtelColModule bool
	}{
		{
			Version:                      "0.85.0",
			ExpectedRequireOtelColModule: false,
		},
		{
			Version:                      "0.86.0",
			ExpectedRequireOtelColModule: true,
		},
		{
			Version:                      "0.86.1",
			ExpectedRequireOtelColModule: true,
		},
		{
			Version:                      "1.0.0",
			ExpectedRequireOtelColModule: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Version, func(t *testing.T) {
			cfg := NewDefaultConfig()
			cfg.Distribution.OtelColVersion = tt.Version
			require.NoError(t, cfg.SetBackwardsCompatibility())
			assert.Equal(t, tt.ExpectedRequireOtelColModule, cfg.Distribution.RequireOtelColModule)
		})
	}
}

func TestConfmapFactoryVersions(t *testing.T) {
	testCases := []struct {
		version   string
		supported bool
		err       bool
	}{
		{
			version:   "x.0.0",
			supported: false,
			err:       true,
		},
		{
			version:   "0.x.0",
			supported: false,
			err:       true,
		},
		{
			version:   "0.0.0",
			supported: false,
		},
		{
			version:   "0.98.0",
			supported: false,
		},
		{
			version:   "0.98.1",
			supported: false,
		},
		{
			version:   "0.99.0",
			supported: true,
		},
		{
			version:   "0.99.7",
			supported: true,
		},
		{
			version:   "0.100.0",
			supported: true,
		},
		{
			version:   "0.100.1",
			supported: true,
		},
		{
			version:   "1.0",
			supported: true,
		},
		{
			version:   "1.0.0",
			supported: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.version, func(t *testing.T) {
			cfg := NewDefaultConfig()
			cfg.Distribution.OtelColVersion = tt.version
			if !tt.err {
				require.NoError(t, cfg.SetBackwardsCompatibility())
				assert.Equal(t, tt.supported, cfg.Distribution.SupportsConfmapFactories)
			} else {
				require.Error(t, cfg.SetBackwardsCompatibility())
			}
		})
	}
}

func TestAddsDefaultProviders(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Providers = nil
	require.NoError(t, cfg.ParseModules())
	assert.Len(t, *cfg.Providers, 5)
}

func TestSkipsNilFieldValidation(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Providers = nil
	assert.NoError(t, cfg.Validate())
}

func TestGetGoVersionFromBuildInfo(t *testing.T) {
	tests := []struct {
		name          string
		mockBuildInfo *debug.BuildInfo
		expected      string
		expectError   bool
	}{
		{
			name: "successful read",
			mockBuildInfo: &debug.BuildInfo{
				GoVersion: "go1.16",
			},
			expected:    "1.16",
			expectError: false,
		},
		{
			name:          "failed read",
			mockBuildInfo: nil,
			expected:      "",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock debug.ReadBuildInfo
			mockReadBuildInfo := func() (*debug.BuildInfo, bool) {
				if tt.mockBuildInfo == nil {
					return nil, false
				}
				return tt.mockBuildInfo, true
			}
			old := readBuildInfo
			readBuildInfo = mockReadBuildInfo
			defer func() { readBuildInfo = old }()

			version, err := getGoVersionFromBuildInfo()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, version)
			}
		})
	}
}

const readmePath = "../../README.md"
const tarballPath = "../../test/test-tarball.tar.gz"

func TestDownloadGoBinary(t *testing.T) {
	tests := []struct {
		name           string
		version        string
		serverResponse string
		serverStatus   int
		expectError    bool
	}{
		{
			name:           "successful download",
			version:        "1.16.5",
			serverResponse: "fake go binary content",
			serverStatus:   http.StatusOK,
			expectError:    false,
		},
		{
			name:           "failed download with 404",
			version:        "1.16.5",
			serverResponse: "not found",
			serverStatus:   http.StatusNotFound,
			expectError:    true,
		},
		{
			name:           "failed to create gzip reader",
			version:        "1.16.5",
			serverResponse: "invalid gzip content",
			serverStatus:   http.StatusOK,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tarballLocation := fmt.Sprintf("/go%s.%s-%s.tar.gz", tt.version, runtime.GOOS, runtime.GOARCH)
			// Create a mock HTTP server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tmp := r.URL.Path
				if tmp == tarballLocation {
					w.WriteHeader(tt.serverStatus)
					if tt.serverStatus == http.StatusOK {
						var fileContent []byte
						var err error
						if tt.serverResponse == "invalid gzip content" {
							fileContent, err = os.ReadFile(readmePath)
						} else {
							fileContent, err = os.ReadFile(tarballPath)
						}
						if err != nil {
							t.Fatalf("failed to read test tarball: %v", err)
						}
						if _, err := w.Write(fileContent); err != nil {
							t.Fatalf("failed to write server response: %v", err)
						}
					} else {
						if _, err := w.Write([]byte(tt.serverResponse)); err != nil {
							t.Fatalf("failed to write server response: %v", err)
						}
					}
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			// Override the goVersionPackageURL with the mock server URL
			goVersionPackageURL = server.URL

			// Call the function under test
			err := downloadGoBinary(tt.version)

			// Check for expected error
			if (err != nil) != tt.expectError {
				t.Fatalf("expected error: %v, got: %v", tt.expectError, err)
			}

			// If no error is expected, check if the file was downloaded correctly
			if !tt.expectError {
				tempDir := os.TempDir()
				if _, err := os.Stat(tempDir); os.IsNotExist(err) {
					t.Fatalf("expected directory %s to exist", tempDir)
				}
			}
		})
	}
}

func TestSetGoPathWithDownload(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on windows")
	}
	cfg := NewDefaultConfig()
	require.NoError(t, cfg.ParseModules())
	assert.NoError(t, cfg.Validate())
	// Save the original PATH
	originalPath := os.Getenv("PATH")

	// Mock the PATH to exclude any references to go
	paths := strings.Split(originalPath, string(os.PathListSeparator))
	var newPaths []string
	for _, p := range paths {
		if !strings.Contains(p, "go") {
			newPaths = append(newPaths, p)
		}
	}
	mockedPath := strings.Join(newPaths, string(os.PathListSeparator))
	t.Setenv("PATH", mockedPath)

	assert.NoError(t, cfg.SetGoPath())
	assert.NoError(t, removeGoTempDir())
}
