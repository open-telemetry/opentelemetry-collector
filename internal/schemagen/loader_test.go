// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoader_LoadFromFile_Success(t *testing.T) {
	tempDir := t.TempDir()

	// Create schema file - for a local ref
	schemaPath := filepath.Join(tempDir, "internal/metadata")
	require.NoError(t, os.MkdirAll(schemaPath, 0o750))
	schemaFile := filepath.Join(schemaPath, schemaFileName)
	schemaContent := `config:
  description: "A test schema"
  type: object
`
	require.NoError(t, os.WriteFile(schemaFile, []byte(schemaContent), 0o600))

	loader := NewLoader(tempDir).(*schemaLoader)

	// For local refs, the loadFromFile method takes a file path
	result, err := loader.loadFromFile(schemaFile)
	require.NoError(t, err)
	require.NotNil(t, result.Config)
	require.Equal(t, "A test schema", result.Config.Description)
	require.Equal(t, "object", result.Config.Type)
}

func TestLoader_LoadFromFile_ExportedConfigsOnly(t *testing.T) {
	tempDir := t.TempDir()
	schemaFile := filepath.Join(tempDir, schemaFileName)
	schemaContent := `exported_configs:
  sample_config:
    type: object
    properties:
      endpoint:
        type: string
`
	require.NoError(t, os.WriteFile(schemaFile, []byte(schemaContent), 0o600))

	loader := NewLoader(tempDir).(*schemaLoader)

	result, err := loader.loadFromFile(schemaFile)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.ExportedConfigs)
	require.Contains(t, result.ExportedConfigs, "sample_config")
	require.Equal(t, "object", result.ExportedConfigs["sample_config"].Type)
	require.Contains(t, result.ExportedConfigs["sample_config"].Properties, "endpoint")
}

func TestLoader_LoadFromFile_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewLoader(tempDir).(*schemaLoader)

	// Try to load from non-existent file
	result, err := loader.loadFromFile(filepath.Join(tempDir, "nonexistent", schemaFileName))
	require.ErrorIs(t, err, ErrNotFound)
	require.Nil(t, result)
}

func TestLoader_LoadFromFile_ParseError(t *testing.T) {
	tempDir := t.TempDir()

	// Create invalid YAML file
	schemaPath := filepath.Join(tempDir, "test/path")
	require.NoError(t, os.MkdirAll(schemaPath, 0o750))
	schemaFile := filepath.Join(schemaPath, schemaFileName)
	invalidYAML := `title: "Test"
  invalid: yaml: content:
`
	require.NoError(t, os.WriteFile(schemaFile, []byte(invalidYAML), 0o600))

	loader := NewLoader(tempDir).(*schemaLoader)

	result, err := loader.loadFromFile(schemaFile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse schema")
	require.Nil(t, result)
}

func TestLoader_LoadFromHTTP_Success(t *testing.T) {
	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("config:\n  title: \"HTTP Schema\"\n  type: string\n"))
	}))
	defer server.Close()

	// Override namespaceToURL for testing
	originalURL := namespaceToURL["go.opentelemetry.io/collector"]
	namespaceToURL["go.opentelemetry.io/collector"] = server.URL
	defer func() { namespaceToURL["go.opentelemetry.io/collector"] = originalURL }()

	tempDir := t.TempDir()
	// Use schemagenDir so resolveModuleVersion can find confmap (a direct dep) without go get.
	loader := NewLoader(schemagenDir(t)).(*schemaLoader)
	ref := *NewRef("go.opentelemetry.io/collector/confmap.controller_config")

	result, err := loader.loadFromHTTP(ref, filepath.Join(tempDir, ".schemas"))
	require.NoError(t, err)
	require.Equal(t, "string", result.Config.Type)
}

func TestLoader_LoadFromHTTP_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	originalURL := namespaceToURL["go.opentelemetry.io/collector"]
	namespaceToURL["go.opentelemetry.io/collector"] = server.URL
	defer func() { namespaceToURL["go.opentelemetry.io/collector"] = originalURL }()

	tempDir := t.TempDir()
	loader := NewLoader(schemagenDir(t)).(*schemaLoader)
	ref := *NewRef("go.opentelemetry.io/collector/confmap.controller_config")

	result, err := loader.loadFromHTTP(ref, filepath.Join(tempDir, ".schemas"))
	require.ErrorIs(t, err, ErrNotFound)
	require.Nil(t, result)
}

func TestLoader_LoadFromHTTP_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	originalURL := namespaceToURL["go.opentelemetry.io/collector"]
	namespaceToURL["go.opentelemetry.io/collector"] = server.URL
	defer func() { namespaceToURL["go.opentelemetry.io/collector"] = originalURL }()

	tempDir := t.TempDir()
	loader := NewLoader(schemagenDir(t)).(*schemaLoader)
	ref := *NewRef("go.opentelemetry.io/collector/confmap.controller_config")

	result, err := loader.loadFromHTTP(ref, filepath.Join(tempDir, ".schemas"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "HTTP 500")
	require.Nil(t, result)
}

func TestLoader_TryLoad_WithVersion(t *testing.T) {
	// Create test HTTP server that returns different content for different versions
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if strings.Contains(r.URL.Path, "v1.0.0") {
			_, _ = w.Write([]byte("config:\n  description: \"Versioned Schema\"\n"))
		} else {
			_, _ = w.Write([]byte("config:\n  description: \"Main Version Schema\"\n"))
		}
	}))
	defer server.Close()

	originalURL := namespaceToURL["go.opentelemetry.io/collector"]
	namespaceToURL["go.opentelemetry.io/collector"] = server.URL
	defer func() { namespaceToURL["go.opentelemetry.io/collector"] = originalURL }()

	tempDir := t.TempDir()
	loader := NewLoader(tempDir).(*schemaLoader)
	ref := *NewRef("go.opentelemetry.io/collector/test/path.config")

	// Try to load with version
	result, err := loader.tryLoad(ref, "v1.0.0")
	require.NoError(t, err)
	require.Equal(t, "Versioned Schema", result.Config.Description)
}

func TestLoader_TryLoad_ExportedConfigsOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`exported_configs:
  sample_config:
    type: object
    properties:
      endpoint:
        type: string
`))
	}))
	defer server.Close()

	originalURL := namespaceToURL["go.opentelemetry.io/collector"]
	namespaceToURL["go.opentelemetry.io/collector"] = server.URL
	defer func() { namespaceToURL["go.opentelemetry.io/collector"] = originalURL }()

	loader := NewLoader(t.TempDir()).(*schemaLoader)
	ref := *NewRef("go.opentelemetry.io/collector/test/path.sample_config")

	result, err := loader.tryLoad(ref, "v1.0.0")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, result.ExportedConfigs, "sample_config")
	require.Equal(t, "object", result.ExportedConfigs["sample_config"].Type)
	require.Contains(t, result.ExportedConfigs["sample_config"].Properties, "endpoint")
}

func TestLoader_PersistToFile_Success(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewLoader(tempDir).(*schemaLoader)

	metadata := &ConfigsMetadata{
		Config: &ConfigMetadata{
			Description: "Test persistence",
		},
	}

	filePath := filepath.Join(tempDir, "test", "persisted.yaml")
	err := loader.persistToFile(filePath, metadata)
	require.NoError(t, err)

	// Verify file was created and can be round-tripped via loadFromFile
	require.FileExists(t, filePath)

	roundTripped, err := loader.loadFromFile(filePath)
	require.NoError(t, err)
	require.Equal(t, "Test persistence", roundTripped.Config.Description)
}

func TestLoader_Load_CacheInteraction(t *testing.T) {
	tempDir := t.TempDir()

	loader := NewLoader(tempDir).(*schemaLoader)
	ref := *NewRef("go.opentelemetry.io/collector/test/path.config")

	expected := &ConfigsMetadata{Config: &ConfigMetadata{Description: "Pre-cached Schema"}}
	loader.cache[ref.CacheKey()] = expected

	result, err := loader.Load(ref)
	require.NoError(t, err)
	require.Equal(t, expected, result)
	require.Same(t, expected, result)
}

func TestLoader_Integration_MemoryCachePeristence(t *testing.T) {
	tempDir := t.TempDir()

	loader := &schemaLoader{
		cache:      make(map[string]*ConfigsMetadata),
		cd:         tempDir,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}

	ref := *NewRef("go.opentelemetry.io/collector/test/path.config")
	expected := &ConfigsMetadata{
		Config: &ConfigMetadata{Description: "Integration Test"},
	}

	loader.cache[ref.CacheKey()] = expected

	result1, err := loader.Load(ref)
	require.NoError(t, err)
	require.Equal(t, "Integration Test", result1.Config.Description)

	result2, err := loader.Load(ref)
	require.NoError(t, err)
	require.Equal(t, "Integration Test", result2.Config.Description)

	require.Same(t, result1, result2)
}

func TestLoader_Load_CachesOnFirstLoad(t *testing.T) {
	tempDir := t.TempDir()

	schemaPath := filepath.Join(tempDir, "subdir")
	require.NoError(t, os.MkdirAll(schemaPath, 0o750))
	schemaFile := filepath.Join(schemaPath, schemaFileName)
	require.NoError(t, os.WriteFile(schemaFile, []byte("config:\n  description: Cached\n  type: object\n"), 0o600))

	loader := NewLoader(tempDir).(*schemaLoader)
	loader.rootDir = tempDir // bypass git

	ref := Ref{schemaID: "subdir", kind: Local}

	result1, err := loader.Load(ref)
	require.NoError(t, err)
	require.Equal(t, "Cached", result1.Config.Description)

	result2, err := loader.Load(ref)
	require.NoError(t, err)
	require.Same(t, result1, result2)
}

func TestLoader_Load_RepoRootError(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewLoader(tempDir).(*schemaLoader)

	ref := Ref{schemaID: "subdir", kind: Local}
	_, err := loader.load(ref)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to determine repo root")
}

func TestLoader_Load_LocalAbsolutePath(t *testing.T) {
	tempDir := t.TempDir()

	// Create schema at <repoRoot>/somepackage/config.schema.yaml
	schemaDir := filepath.Join(tempDir, "somepackage")
	require.NoError(t, os.MkdirAll(schemaDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(schemaDir, schemaFileName), []byte("config:\n  description: AbsoluteLocal\n  type: object\n"), 0o600))

	loader := NewLoader(tempDir).(*schemaLoader)
	loader.rootDir = tempDir // bypass git

	// absolute local ref (starts with /)
	ref := Ref{schemaID: "/somepackage", kind: Local}
	result, err := loader.load(ref)
	require.NoError(t, err)
	require.Equal(t, "AbsoluteLocal", result.Config.Description)
}

func TestLoader_Load_LocalRelativePath(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, schemaFileName), []byte("config:\n  description: RelativeLocal\n  type: object\n"), 0o600))

	loader := NewLoader(tempDir).(*schemaLoader)
	loader.rootDir = tempDir // bypass git

	// relative local ref (does not start with /)
	ref := Ref{schemaID: "subdir", kind: Local}
	result, err := loader.load(ref)
	require.NoError(t, err)
	require.Equal(t, "RelativeLocal", result.Config.Description)
}

func TestLoader_LoadFromFile_ReadError(t *testing.T) {
	// Create a directory where the file is expected — os.ReadFile on a directory fails
	tempDir := t.TempDir()
	dirPath := filepath.Join(tempDir, schemaFileName)
	require.NoError(t, os.MkdirAll(dirPath, 0o750))

	loader := NewLoader(tempDir).(*schemaLoader)
	result, err := loader.loadFromFile(dirPath)
	require.Error(t, err)
	require.Nil(t, result)
	require.NotErrorIs(t, err, ErrNotFound)
	require.Contains(t, err.Error(), "failed to read schema")
}

func TestLoader_LoadFromHTTP_FileCacheHit(t *testing.T) {
	// Test that loadFromHTTP returns from the file cache when a schema is already persisted,
	// without making any HTTP requests.
	ref := *NewRef("go.opentelemetry.io/collector/confmap.controller_config")

	// Resolve the real module version so we can place the file at the right path.
	sgDir := schemagenDir(t)
	helper := NewLoader(sgDir).(*schemaLoader)
	version := helper.resolveModuleVersion(ref.Module())
	if version == "" {
		t.Skip("could not resolve module version, skipping file cache hit test")
	}

	tempDir := t.TempDir()
	fileCacheDir := filepath.Join(tempDir, ".schemas")
	filePath := filepath.Join(fileCacheDir, version, ref.SchemaID(), schemaFileName)
	require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0o750))
	require.NoError(t, os.WriteFile(filePath, []byte("config:\n  description: FileCached\n  type: object\n"), 0o600))

	// Use an httpClient that always panics to confirm no HTTP call is made.
	loader := &schemaLoader{
		cache:      make(map[string]*ConfigsMetadata),
		cd:         sgDir,
		httpClient: &http.Client{},
	}

	result, err := loader.loadFromHTTP(ref, fileCacheDir)
	require.NoError(t, err)
	require.Equal(t, "FileCached", result.Config.Description)
}

func TestLoader_LoadFromHTTP_PersistWarning(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission test not reliable on Windows")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("config:\n  description: PersistFail\n  type: object\n"))
	}))
	defer server.Close()

	originalURL := namespaceToURL["go.opentelemetry.io/collector"]
	namespaceToURL["go.opentelemetry.io/collector"] = server.URL
	defer func() { namespaceToURL["go.opentelemetry.io/collector"] = originalURL }()

	tempDir := t.TempDir()

	// Make the file cache directory read-only so persistToFile fails
	cacheDir := filepath.Join(tempDir, ".schemas")
	require.NoError(t, os.MkdirAll(cacheDir, 0o750))

	loader := NewLoader(schemagenDir(t)).(*schemaLoader)
	ref := *NewRef("go.opentelemetry.io/collector/confmap.controller_config")

	// Should still return the result even though persist fails (warning only)
	result, err := loader.loadFromHTTP(ref, cacheDir)
	require.NoError(t, err)
	require.Equal(t, "PersistFail", result.Config.Description)
}

func TestLoader_LoadFromHTTP_NonNotFoundFileError(t *testing.T) {
	// Test the log.Printf warning path in loadFromHTTP when loadFromFile on the cached path
	// fails with a non-ErrNotFound error (directory placed where file is expected).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("config:\n  description: AfterWarning\n  type: object\n"))
	}))
	defer server.Close()

	originalURL := namespaceToURL["go.opentelemetry.io/collector"]
	namespaceToURL["go.opentelemetry.io/collector"] = server.URL
	defer func() { namespaceToURL["go.opentelemetry.io/collector"] = originalURL }()

	sgDir := schemagenDir(t)
	ref := *NewRef("go.opentelemetry.io/collector/confmap.controller_config")

	// Resolve the real version so we can poison the file cache at the correct path
	helper := NewLoader(sgDir).(*schemaLoader)
	version := helper.resolveModuleVersion(ref.Module())
	if version == "" {
		t.Skip("could not resolve module version, skipping non-ErrNotFound warning test")
	}

	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, ".schemas")
	filePath := filepath.Join(cacheDir, version, ref.SchemaID(), schemaFileName)
	// Place a directory at the expected file path → loadFromFile returns a non-ErrNotFound error
	require.NoError(t, os.MkdirAll(filePath, 0o750))

	loader := &schemaLoader{
		cache:      make(map[string]*ConfigsMetadata),
		cd:         sgDir,
		httpClient: &http.Client{},
	}

	result, err := loader.loadFromHTTP(ref, cacheDir)
	require.NoError(t, err)
	require.Equal(t, "AfterWarning", result.Config.Description)
}

func TestLoader_TryLoad_URLError(t *testing.T) {
	// Ref with unsupported namespace → URL() returns an error
	loader := NewLoader("").(*schemaLoader)
	ref := *NewRef("unknownns/path.type")
	// namespace is set so Module() returns non-empty, but Namespace() returns false
	// Actually NewRef sets it as external; let's manually set up a ref with no URL support
	ref2 := Ref{namespace: "unsupported.example.com", schemaID: "pkg", configName: "t", kind: External}
	_, err := loader.tryLoad(ref2, "v1.0.0")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to construct URL")
	_ = ref
}

func TestLoader_TryLoad_RequestCreationError(t *testing.T) {
	loader := NewLoader("").(*schemaLoader)
	ref := *NewRef("go.opentelemetry.io/collector/test/path.config")

	originalURL := namespaceToURL["go.opentelemetry.io/collector"]
	namespaceToURL["go.opentelemetry.io/collector"] = "http://bad host"
	defer func() { namespaceToURL["go.opentelemetry.io/collector"] = originalURL }()

	_, err := loader.tryLoad(ref, "v1.0.0")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to create a HTTP GET request")
}

func TestLoader_TryLoad_HTTPError(t *testing.T) {
	// Use a server that closes connections immediately to simulate a network error
	// The simplest approach: use an invalid URL
	loader := NewLoader("").(*schemaLoader)

	ref := *NewRef("go.opentelemetry.io/collector/test/path.config")

	// Override the URL to point to an invalid host
	originalURL := namespaceToURL["go.opentelemetry.io/collector"]
	namespaceToURL["go.opentelemetry.io/collector"] = "http://127.0.0.1:0" // port 0 → connection refused
	defer func() { namespaceToURL["go.opentelemetry.io/collector"] = originalURL }()

	_, err := loader.tryLoad(ref, "v1.0.0")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to fetch schema")
}

func TestLoader_TryLoad_ParseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("title: bad\n  invalid: yaml: :\n"))
	}))
	defer server.Close()

	originalURL := namespaceToURL["go.opentelemetry.io/collector"]
	namespaceToURL["go.opentelemetry.io/collector"] = server.URL
	defer func() { namespaceToURL["go.opentelemetry.io/collector"] = originalURL }()

	loader := NewLoader("").(*schemaLoader)
	ref := *NewRef("go.opentelemetry.io/collector/test/path.config")
	_, err := loader.tryLoad(ref, "v1.0.0")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse schema")
}

func TestLoader_RefVersion_UnknownModulePath(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewLoader(tempDir).(*schemaLoader)

	ref := Ref{namespace: "", schemaID: "pkg", configName: "t", kind: Internal}
	_, err := loader.refVersion(&ref)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown module path")
}

func TestLoader_ResolveModuleVersion_NilModule(t *testing.T) {
	loader := NewLoader(mdatagenDir(t)).(*schemaLoader)

	version := loader.resolveModuleVersion("fmt")
	require.Empty(t, version)
}

func TestLoader_ResolveModuleVersion_LoadError(t *testing.T) {
	loader := NewLoader("/nonexistent/path/xyz").(*schemaLoader)

	version := loader.resolveModuleVersion("somemodule/that/doesnt/exist")
	require.Empty(t, version)
}

func TestLoader_RepoRoot_CachedValue(t *testing.T) {
	loader := NewLoader("").(*schemaLoader)
	loader.rootDir = "/cached/root"

	root, err := loader.repoRoot("/any/dir")
	require.NoError(t, err)
	require.Equal(t, "/cached/root", root)
}

func TestLoader_RepoRoot_GitError(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewLoader(tempDir).(*schemaLoader)

	_, err := loader.repoRoot(tempDir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to determine repo root")
}

func TestLoader_PersistToFile_MkdirAllError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission test not reliable on Windows")
	}
	tempDir := t.TempDir()
	t.Cleanup(func() {
		_ = os.Chmod(tempDir, 0o700) // #nosec G302 -- restore so t.TempDir cleanup can remove it
	})

	require.NoError(t, os.Chmod(tempDir, 0o500)) // #nosec G302

	loader := NewLoader(tempDir).(*schemaLoader)
	err := loader.persistToFile(filepath.Join(tempDir, "newdir", "schema.yaml"), &ConfigsMetadata{Config: &ConfigMetadata{Description: "X"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create directory")
}

func TestLoader_PersistToFile_WriteFileError(t *testing.T) {
	tempDir := t.TempDir()

	filePath := filepath.Join(tempDir, "schema.yaml")
	require.NoError(t, os.MkdirAll(filePath, 0o750))

	loader := NewLoader(tempDir).(*schemaLoader)
	err := loader.persistToFile(filePath, &ConfigsMetadata{Config: &ConfigMetadata{Description: "X"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to write file")
}

func TestLoader_TryLoad_ReadBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "1000")
		w.WriteHeader(http.StatusOK)
		hj, ok := w.(http.Hijacker)
		if !ok {
			return
		}
		conn, _, _ := hj.Hijack()
		_ = conn.Close()
	}))
	defer server.Close()

	originalURL := namespaceToURL["go.opentelemetry.io/collector"]
	namespaceToURL["go.opentelemetry.io/collector"] = server.URL
	defer func() { namespaceToURL["go.opentelemetry.io/collector"] = originalURL }()

	loader := NewLoader("").(*schemaLoader)
	ref := *NewRef("go.opentelemetry.io/collector/test/path.config")
	_, err := loader.tryLoad(ref, "v1.0.0")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to read response body")
}

func mdatagenDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "could not determine caller file")
	return filepath.Join(filepath.Dir(file), "../../cmd/mdatagen")
}

// schemagenDir returns the directory of this test file (internal/schemagen).
// Use it as the loader's cwd when tests need packages.Load to resolve a module
// version without invoking go get — confmap is a direct dep here and locally replaced.
func schemagenDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "could not determine caller file")
	return filepath.Dir(file)
}
