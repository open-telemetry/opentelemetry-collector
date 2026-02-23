// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoader_Load_Cache(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewLoader(tempDir).(*schemaLoader)

	ref := *NewRef("go.opentelemetry.io/collector/scraper/scraperhelper.controller_config", "")

	// Pre-populate cache
	expected := &ConfigMetadata{Title: "cached"}
	loader.cache[ref.CacheKey()] = expected

	// Load should return cached value
	result, err := loader.Load(ref)
	require.NoError(t, err)
	require.Equal(t, expected, result)
}

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
	loader := NewLoader(tempDir).(*schemaLoader)
	ref := *NewRef("go.opentelemetry.io/collector/test/path.config", "")

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
	loader := NewLoader(tempDir).(*schemaLoader)
	ref := *NewRef("go.opentelemetry.io/collector/test/path.config", "")

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
	loader := NewLoader(tempDir).(*schemaLoader)
	ref := *NewRef("go.opentelemetry.io/collector/test/path.config", "")

	result, err := loader.loadFromHTTP(ref, filepath.Join(tempDir, ".schemas"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "HTTP 500")
	require.Nil(t, result)
}

func TestLoader_TryLoad_WithVersion(t *testing.T) {
	// Create test HTTP server that returns different content for different versions
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if strings.Contains(r.URL.Path, "main") {
			_, _ = w.Write([]byte(`title: "Main Version Schema"`))
		} else {
			_, _ = w.Write([]byte(`title: "Versioned Schema"`))
		}
	}))
	defer server.Close()

	originalURL := namespaceToURL["go.opentelemetry.io/collector"]
	namespaceToURL["go.opentelemetry.io/collector"] = server.URL
	defer func() { namespaceToURL["go.opentelemetry.io/collector"] = originalURL }()

	tempDir := t.TempDir()
	loader := NewLoader(tempDir).(*schemaLoader)
	ref := *NewRef("go.opentelemetry.io/collector/test/path.config", "")

	// Try to load with version
	result, err := loader.tryLoad(ref, "v1.0.0")
	require.NoError(t, err)
	require.Equal(t, "Versioned Schema", result.Title)
}

func TestLoader_TryLoad_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	originalURL := namespaceToURL["go.opentelemetry.io/collector"]
	namespaceToURL["go.opentelemetry.io/collector"] = server.URL
	defer func() { namespaceToURL["go.opentelemetry.io/collector"] = originalURL }()

	tempDir := t.TempDir()
	loader := NewLoader(tempDir).(*schemaLoader)
	ref := *NewRef("go.opentelemetry.io/collector/nonexistent/path.config", "")

	// tryLoad should return ErrNotFound
	result, err := loader.tryLoad(ref, "v1.0.0")
	require.ErrorIs(t, err, ErrNotFound)
	require.Nil(t, result)
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

	// Verify content
	content, err := os.ReadFile(filePath) // #nosec G304
	require.NoError(t, err)
	require.Contains(t, string(content), "Persisted Schema")
	require.Contains(t, string(content), "Test persistence")
}

func TestLoader_Load_CacheInteraction(t *testing.T) {
	tempDir := t.TempDir()

	// Create a loader and manually populate its cache
	loader := NewLoader(tempDir).(*schemaLoader)
	ref := *NewRef("go.opentelemetry.io/collector/test/path.config", "")

	expected := &ConfigMetadata{Title: "Pre-cached Schema"}
	loader.cache[ref.CacheKey()] = expected

	// Load should return the cached value
	result, err := loader.Load(ref)
	require.NoError(t, err)
	require.Equal(t, expected, result)
	require.Same(t, expected, result)
}

func TestLoader_TryLoad_InvalidYAML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`invalid: yaml: content:`))
	}))
	defer server.Close()

	originalURL := namespaceToURL["go.opentelemetry.io/collector"]
	namespaceToURL["go.opentelemetry.io/collector"] = server.URL
	defer func() { namespaceToURL["go.opentelemetry.io/collector"] = originalURL }()

	tempDir := t.TempDir()
	loader := NewLoader(tempDir).(*schemaLoader)
	ref := *NewRef("go.opentelemetry.io/collector/test/path.config", "")

	result, err := loader.tryLoad(ref, "main")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse schema")
	require.Nil(t, result)
}

func TestLoader_Integration_MemoryCachePeristence(t *testing.T) {
	tempDir := t.TempDir()

	// Create a schemaLoader directly with a pre-populated cache
	loader := &schemaLoader{
		cache:      make(map[string]*ConfigMetadata),
		cd:         tempDir,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}

	ref := *NewRef("go.opentelemetry.io/collector/test/path.config", "")
	expected := &ConfigMetadata{Title: "Integration Test"}

	// Pre-populate cache
	loader.cache[ref.CacheKey()] = expected

	// First load - from cache
	result1, err := loader.Load(ref)
	require.NoError(t, err)
	require.Equal(t, "Integration Test", result1.Title)

	// Second load - should still come from cache
	result2, err := loader.Load(ref)
	require.NoError(t, err)
	require.Equal(t, "Integration Test", result2.Title)

	// Verify it's the same instance (from cache)
	require.Same(t, result1, result2)
}
