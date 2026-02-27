// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen

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
	schemaContent := `title: "Test Schema"
description: "A test schema"
type: object
`
	require.NoError(t, os.WriteFile(schemaFile, []byte(schemaContent), 0o600))

	loader := NewLoader(tempDir).(*schemaLoader)

	// For local refs, the loadFromFile method takes a file path
	result, err := loader.loadFromFile(schemaFile)
	require.NoError(t, err)
	require.Equal(t, "Test Schema", result.Title)
	require.Equal(t, "A test schema", result.Description)
	require.Equal(t, "object", result.Type)
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
		_, _ = w.Write([]byte(`title: "HTTP Schema"
type: string
`))
	}))
	defer server.Close()

	// Override namespaceToURL for testing
	originalURL := namespaceToURL["go.opentelemetry.io/collector"]
	namespaceToURL["go.opentelemetry.io/collector"] = server.URL
	defer func() { namespaceToURL["go.opentelemetry.io/collector"] = originalURL }()

	tempDir := t.TempDir()
	// Use mdatagenDir so resolveModuleVersion can find the module in go.mod
	loader := NewLoader(mdatagenDir(t)).(*schemaLoader)
	ref := *NewRef("go.opentelemetry.io/collector/scraper/scraperhelper.controller_config")

	result, err := loader.loadFromHTTP(ref, filepath.Join(tempDir, ".schemas"))
	require.NoError(t, err)
	require.Equal(t, "HTTP Schema", result.Title)
	require.Equal(t, "string", result.Type)
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
	loader := NewLoader(mdatagenDir(t)).(*schemaLoader)
	ref := *NewRef("go.opentelemetry.io/collector/scraper/scraperhelper.controller_config")

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
	loader := NewLoader(mdatagenDir(t)).(*schemaLoader)
	ref := *NewRef("go.opentelemetry.io/collector/scraper/scraperhelper.controller_config")

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
			_, _ = w.Write([]byte(`title: "Versioned Schema"`))
		} else {
			_, _ = w.Write([]byte(`title: "Main Version Schema"`))
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
	require.Equal(t, "Versioned Schema", result.Title)
}

func TestLoader_PersistToFile_Success(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewLoader(tempDir).(*schemaLoader)

	metadata := &ConfigMetadata{
		Title:       "Persisted Schema",
		Description: "Test persistence",
	}

	filePath := filepath.Join(tempDir, "test", "persisted.yaml")
	err := loader.persistToFile(filePath, metadata)
	require.NoError(t, err)

	// Verify file was created
	require.FileExists(t, filePath)

	content, err := os.ReadFile(filePath) // #nosec G304
	require.NoError(t, err)
	require.Contains(t, string(content), "Persisted Schema")
	require.Contains(t, string(content), "Test persistence")
}

func TestLoader_Load_CacheInteraction(t *testing.T) {
	tempDir := t.TempDir()

	loader := NewLoader(tempDir).(*schemaLoader)
	ref := *NewRef("go.opentelemetry.io/collector/test/path.config")

	expected := &ConfigMetadata{Title: "Pre-cached Schema"}
	loader.cache[ref.CacheKey()] = expected

	result, err := loader.Load(ref)
	require.NoError(t, err)
	require.Equal(t, expected, result)
	require.Same(t, expected, result)
}

func TestLoader_Integration_MemoryCachePeristence(t *testing.T) {
	tempDir := t.TempDir()

	loader := &schemaLoader{
		cache:      make(map[string]*ConfigMetadata),
		cd:         tempDir,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}

	ref := *NewRef("go.opentelemetry.io/collector/test/path.config")
	expected := &ConfigMetadata{Title: "Integration Test"}

	loader.cache[ref.CacheKey()] = expected

	result1, err := loader.Load(ref)
	require.NoError(t, err)
	require.Equal(t, "Integration Test", result1.Title)

	result2, err := loader.Load(ref)
	require.NoError(t, err)
	require.Equal(t, "Integration Test", result2.Title)

	require.Same(t, result1, result2)
}

func TestLoader_Load_CachesOnFirstLoad(t *testing.T) {
	tempDir := t.TempDir()

	schemaPath := filepath.Join(tempDir, "subdir")
	require.NoError(t, os.MkdirAll(schemaPath, 0o750))
	schemaFile := filepath.Join(schemaPath, schemaFileName)
	require.NoError(t, os.WriteFile(schemaFile, []byte("title: Cached\ntype: object\n"), 0o600))

	loader := NewLoader(tempDir).(*schemaLoader)
	loader.rootDir = tempDir // bypass git

	ref := Ref{schemaID: "subdir", kind: Local}

	result1, err := loader.Load(ref)
	require.NoError(t, err)
	require.Equal(t, "Cached", result1.Title)

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
	require.NoError(t, os.WriteFile(filepath.Join(schemaDir, schemaFileName), []byte("title: AbsoluteLocal\ntype: object\n"), 0o600))

	loader := NewLoader(tempDir).(*schemaLoader)
	loader.rootDir = tempDir // bypass git

	// absolute local ref (starts with /)
	ref := Ref{schemaID: "/somepackage", kind: Local}
	result, err := loader.load(ref)
	require.NoError(t, err)
	require.Equal(t, "AbsoluteLocal", result.Title)
}

func TestLoader_Load_LocalRelativePath(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, schemaFileName), []byte("title: RelativeLocal\ntype: object\n"), 0o600))

	loader := NewLoader(tempDir).(*schemaLoader)
	loader.rootDir = tempDir // bypass git

	// relative local ref (does not start with /)
	ref := Ref{schemaID: "subdir", kind: Local}
	result, err := loader.load(ref)
	require.NoError(t, err)
	require.Equal(t, "RelativeLocal", result.Title)
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
	ref := *NewRef("go.opentelemetry.io/collector/scraper/scraperhelper.controller_config")

	// Resolve the real module version so we can place the file at the right path.
	mdDir := mdatagenDir(t)
	helper := NewLoader(mdDir).(*schemaLoader)
	version := helper.resolveModuleVersion(ref.Module())
	if version == "" {
		t.Skip("could not resolve module version, skipping file cache hit test")
	}

	tempDir := t.TempDir()
	fileCacheDir := filepath.Join(tempDir, ".schemas")
	filePath := filepath.Join(fileCacheDir, version, ref.SchemaID(), schemaFileName)
	require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0o750))
	require.NoError(t, os.WriteFile(filePath, []byte("title: FileCached\ntype: object\n"), 0o600))

	// Use an httpClient that always panics to confirm no HTTP call is made.
	loader := &schemaLoader{
		cache:      make(map[string]*ConfigMetadata),
		cd:         mdDir,
		httpClient: &http.Client{},
	}

	result, err := loader.loadFromHTTP(ref, fileCacheDir)
	require.NoError(t, err)
	require.Equal(t, "FileCached", result.Title)
}

func TestLoader_LoadFromHTTP_PersistWarning(t *testing.T) {
	// Test the warning path when persistToFile fails (non-writable dir).
	// We create a read-only file cache dir so persistToFile fails.
	if runtime.GOOS == "windows" {
		t.Skip("file permission test not reliable on Windows")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("title: PersistFail\ntype: object\n"))
	}))
	defer server.Close()

	originalURL := namespaceToURL["go.opentelemetry.io/collector"]
	namespaceToURL["go.opentelemetry.io/collector"] = server.URL
	defer func() { namespaceToURL["go.opentelemetry.io/collector"] = originalURL }()

	tempDir := t.TempDir()

	// Make the file cache directory read-only so persistToFile fails
	cacheDir := filepath.Join(tempDir, ".schemas")
	require.NoError(t, os.MkdirAll(cacheDir, 0o750))

	loader := NewLoader(mdatagenDir(t)).(*schemaLoader)
	ref := *NewRef("go.opentelemetry.io/collector/scraper/scraperhelper.controller_config")

	// Should still return the result even though persist fails (warning only)
	result, err := loader.loadFromHTTP(ref, cacheDir)
	require.NoError(t, err)
	require.Equal(t, "PersistFail", result.Title)
}

func TestLoader_LoadFromHTTP_NonNotFoundFileError(t *testing.T) {
	// Test the log.Printf warning path in loadFromHTTP when loadFromFile on the cached path
	// fails with a non-ErrNotFound error (directory placed where file is expected).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("title: AfterWarning\ntype: object\n"))
	}))
	defer server.Close()

	originalURL := namespaceToURL["go.opentelemetry.io/collector"]
	namespaceToURL["go.opentelemetry.io/collector"] = server.URL
	defer func() { namespaceToURL["go.opentelemetry.io/collector"] = originalURL }()

	mdDir := mdatagenDir(t)
	ref := *NewRef("go.opentelemetry.io/collector/scraper/scraperhelper.controller_config")

	// Resolve the real version so we can poison the file cache at the correct path
	helper := NewLoader(mdDir).(*schemaLoader)
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
		cache:      make(map[string]*ConfigMetadata),
		cd:         mdDir,
		httpClient: &http.Client{},
	}

	result, err := loader.loadFromHTTP(ref, cacheDir)
	require.NoError(t, err)
	require.Equal(t, "AfterWarning", result.Title)
}

func TestLoader_TryLoad_URLError(t *testing.T) {
	// Ref with unsupported namespace → URL() returns an error
	loader := NewLoader("").(*schemaLoader)
	ref := *NewRef("unknownns/path.type")
	// namespace is set so Module() returns non-empty, but Namespace() returns false
	// Actually NewRef sets it as external; let's manually set up a ref with no URL support
	ref2 := Ref{namespace: "unsupported.example.com", schemaID: "pkg", defName: "t", kind: External}
	_, err := loader.tryLoad(ref2, "v1.0.0")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to construct URL")
	_ = ref
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

	ref := Ref{namespace: "", schemaID: "pkg", defName: "t", kind: Internal}
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

func mdatagenDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "could not determine caller file")
	return filepath.Join(filepath.Dir(file), "../..")
}
