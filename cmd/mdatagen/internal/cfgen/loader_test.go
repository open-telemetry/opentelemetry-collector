// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfgen

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoader_Load_Cache(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewLoader(tempDir).(*loader)

	ref := Ref{
		Namespace: "go.opentelemetry.io/collector",
		Path:      "scraper/scraperhelper",
		Type:      "controller_config",
	}

	// Pre-populate cache
	expected := &ConfigMetadata{Title: "cached"}
	loader.cache[cacheKey(ref, "v1.0.0")] = expected

	// Load should return cached value
	result, err := loader.Load(ref, "v1.0.0")
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestLoader_LoadFromFile_Success(t *testing.T) {
	tempDir := t.TempDir()

	// Create schema file
	schemaPath := filepath.Join(tempDir, "main", "go.opentelemetry.io/collector/scraper/scraperhelper")
	require.NoError(t, os.MkdirAll(schemaPath, 0o755))
	schemaFile := filepath.Join(schemaPath, schemaFileName)
	schemaContent := `title: "Test Schema"
description: "A test schema"
type: object
`
	require.NoError(t, os.WriteFile(schemaFile, []byte(schemaContent), 0o644))

	loader := NewLoader(tempDir).(*loader)
	ref := Ref{
		Namespace: "go.opentelemetry.io/collector",
		Path:      "scraper/scraperhelper",
		Type:      "controller_config",
	}

	result, err := loader.loadFromFile(ref, "main")
	require.NoError(t, err)
	assert.Equal(t, "Test Schema", result.Title)
	assert.Equal(t, "A test schema", result.Description)
	assert.Equal(t, "object", result.Type)
}

func TestLoader_LoadFromFile_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewLoader(tempDir).(*loader)

	ref := Ref{
		Namespace: "go.opentelemetry.io/collector",
		Path:      "nonexistent/path",
		Type:      "config",
	}

	result, err := loader.loadFromFile(ref, "main")
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Nil(t, result)
}

func TestLoader_LoadFromFile_ParseError(t *testing.T) {
	tempDir := t.TempDir()

	// Create invalid YAML file
	schemaPath := filepath.Join(tempDir, "main", "go.opentelemetry.io/collector/test/path")
	require.NoError(t, os.MkdirAll(schemaPath, 0o755))
	schemaFile := filepath.Join(schemaPath, schemaFileName)
	invalidYAML := `title: "Test"
  invalid: yaml: content:
`
	require.NoError(t, os.WriteFile(schemaFile, []byte(invalidYAML), 0o644))

	loader := NewLoader(tempDir).(*loader)
	ref := Ref{
		Namespace: "go.opentelemetry.io/collector",
		Path:      "test/path",
		Type:      "config",
	}

	result, err := loader.loadFromFile(ref, "main")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse schema")
	assert.Nil(t, result)
}

func TestLoader_LoadFromHTTP_Success(t *testing.T) {
	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`title: "HTTP Schema"
type: string
`))
	}))
	defer server.Close()

	// Override namespaceToURL for testing
	originalURL := namespaceToURL["go.opentelemetry.io/collector"]
	namespaceToURL["go.opentelemetry.io/collector"] = server.URL
	defer func() { namespaceToURL["go.opentelemetry.io/collector"] = originalURL }()

	loader := NewLoader("").(*loader)
	ref := Ref{
		Namespace: "go.opentelemetry.io/collector",
		Path:      "test/path",
		Type:      "config",
	}

	result, err := loader.loadFromHTTP(ref, "main")
	require.NoError(t, err)
	assert.Equal(t, "HTTP Schema", result.Title)
	assert.Equal(t, "string", result.Type)
}

func TestLoader_LoadFromHTTP_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	originalURL := namespaceToURL["go.opentelemetry.io/collector"]
	namespaceToURL["go.opentelemetry.io/collector"] = server.URL
	defer func() { namespaceToURL["go.opentelemetry.io/collector"] = originalURL }()

	loader := NewLoader("").(*loader)
	ref := Ref{
		Namespace: "go.opentelemetry.io/collector",
		Path:      "test/path",
		Type:      "config",
	}

	result, err := loader.loadFromHTTP(ref, "main")
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Nil(t, result)
}

func TestLoader_LoadFromHTTP_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	originalURL := namespaceToURL["go.opentelemetry.io/collector"]
	namespaceToURL["go.opentelemetry.io/collector"] = server.URL
	defer func() { namespaceToURL["go.opentelemetry.io/collector"] = originalURL }()

	loader := NewLoader("").(*loader)
	ref := Ref{
		Namespace: "go.opentelemetry.io/collector",
		Path:      "test/path",
		Type:      "config",
	}

	result, err := loader.loadFromHTTP(ref, "main")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 500")
	assert.Nil(t, result)
}

func TestLoader_LoadWithFallback_Success(t *testing.T) {
	tempDir := t.TempDir()

	// Create schema file only for "main" version
	schemaPath := filepath.Join(tempDir, "main", "go.opentelemetry.io/collector/test/path")
	require.NoError(t, os.MkdirAll(schemaPath, 0o755))
	schemaFile := filepath.Join(schemaPath, schemaFileName)
	require.NoError(t, os.WriteFile(schemaFile, []byte(`title: "Fallback Schema"`), 0o644))

	loader := NewLoader(tempDir).(*loader)
	ref := Ref{
		Namespace: "go.opentelemetry.io/collector",
		Path:      "test/path",
		Type:      "config",
	}

	// Try to load v1.0.0, should fallback to main
	result, err := loader.loadWithFallback(ref, "v1.0.0")
	require.NoError(t, err)
	assert.Equal(t, "Fallback Schema", result.Title)
}

func TestLoader_LoadWithFallback_MainVersionFails(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewLoader(tempDir).(*loader)

	ref := Ref{
		Namespace: "go.opentelemetry.io/collector",
		Path:      "nonexistent/path",
		Type:      "config",
	}

	// Both v1.0.0 and main should fail
	result, err := loader.loadWithFallback(ref, "v1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fallback to main also failed")
	assert.Nil(t, result)
}

func TestLoader_PersistToFile_Success(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewLoader(tempDir).(*loader)

	ref := Ref{
		Namespace: "go.opentelemetry.io/collector",
		Path:      "test/path",
		Type:      "config",
	}

	metadata := &ConfigMetadata{
		Title:       "Persisted Schema",
		Description: "Test persistence",
	}

	err := loader.persistToFile(ref, "v1.0.0", metadata)
	require.NoError(t, err)

	// Verify file was created
	filePath := filepath.Join(tempDir, "v1.0.0", ref.PkgPath(), schemaFileName)
	assert.FileExists(t, filePath)

	// Verify content
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Persisted Schema")
	assert.Contains(t, string(content), "Test persistence")
}

func TestLoader_TryLoad_FileToHTTPFallback(t *testing.T) {
	tempDir := t.TempDir()

	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`title: "HTTP Fallback"`))
	}))
	defer server.Close()

	originalURL := namespaceToURL["go.opentelemetry.io/collector"]
	namespaceToURL["go.opentelemetry.io/collector"] = server.URL
	defer func() { namespaceToURL["go.opentelemetry.io/collector"] = originalURL }()

	loader := NewLoader(tempDir).(*loader)
	ref := Ref{
		Namespace: "go.opentelemetry.io/collector",
		Path:      "test/path",
		Type:      "config",
	}

	// File doesn't exist, should fall back to HTTP
	result, err := loader.tryLoad(ref, "main")
	require.NoError(t, err)
	assert.Equal(t, "HTTP Fallback", result.Title)

	// Verify it was persisted to file
	filePath := filepath.Join(tempDir, "main", ref.PkgPath(), schemaFileName)
	assert.FileExists(t, filePath)
}

func TestLoader_LoadFromHTTP_InvalidYAML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid: yaml: content:`))
	}))
	defer server.Close()

	originalURL := namespaceToURL["go.opentelemetry.io/collector"]
	namespaceToURL["go.opentelemetry.io/collector"] = server.URL
	defer func() { namespaceToURL["go.opentelemetry.io/collector"] = originalURL }()

	loader := NewLoader("").(*loader)
	ref := Ref{
		Namespace: "go.opentelemetry.io/collector",
		Path:      "test/path",
		Type:      "config",
	}

	result, err := loader.loadFromHTTP(ref, "main")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse schema")
	assert.Nil(t, result)
}

func TestLoader_Integration_CacheAfterLoad(t *testing.T) {
	tempDir := t.TempDir()

	// Create schema file
	schemaPath := filepath.Join(tempDir, "main", "go.opentelemetry.io/collector/test/path")
	require.NoError(t, os.MkdirAll(schemaPath, 0o755))
	schemaFile := filepath.Join(schemaPath, schemaFileName)
	require.NoError(t, os.WriteFile(schemaFile, []byte(`title: "Integration Test"`), 0o644))

	loader := NewLoader(tempDir)
	ref := Ref{
		Namespace: "go.opentelemetry.io/collector",
		Path:      "test/path",
		Type:      "config",
	}

	// First load - from file
	result1, err := loader.Load(ref, "main")
	require.NoError(t, err)
	assert.Equal(t, "Integration Test", result1.Title)

	// Delete the file
	require.NoError(t, os.Remove(schemaFile))

	// Second load - should come from cache, not fail
	result2, err := loader.Load(ref, "main")
	require.NoError(t, err)
	assert.Equal(t, "Integration Test", result2.Title)

	// Verify it's the same instance (from cache)
	assert.Same(t, result1, result2)
}
