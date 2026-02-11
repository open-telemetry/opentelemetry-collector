// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/cfgen"

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go.yaml.in/yaml/v3"
)

const (
	defaultVersion = "main"
	schemaFileName = "config.schema.yaml"
)

// ErrNotFound indicates a schema was not found by any source.
var ErrNotFound = errors.New("schema not found")

// Loader loads configuration metadata from various sources (file, HTTP).
// It automatically handles caching, version fallback, and persistence.
type Loader interface {
	Load(ref Ref, version string) (*ConfigMetadata, error)
}

// NewLoader creates a fully configured loader.
// If schemasDir is provided, enables file source and automatic persistence of HTTP fetches.
func NewLoader(schemasDir string) Loader {
	return &loader{
		cache:      make(map[string]*ConfigMetadata),
		schemasDir: schemasDir,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// loader is the concrete implementation of Loader.
type loader struct {
	cache      map[string]*ConfigMetadata
	schemasDir string
	httpClient *http.Client
}

func (l *loader) Load(ref Ref, version string) (*ConfigMetadata, error) {
	// 1. Check cache
	key := cacheKey(ref, version)
	if cached, ok := l.cache[key]; ok {
		return cached, nil
	}

	// 2. Try loading (with version fallback)
	metadata, err := l.loadWithFallback(ref, version)
	if err != nil {
		return nil, err
	}

	// 3. Cache the result
	l.cache[key] = metadata

	return metadata, nil
}

// loadWithFallback tries to load the requested version, falling back to "main" if not found.
func (l *loader) loadWithFallback(ref Ref, version string) (*ConfigMetadata, error) {
	metadata, err := l.tryLoad(ref, version)
	if err == nil {
		return metadata, nil
	}

	if version == defaultVersion || !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	metadata, err = l.tryLoad(ref, defaultVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to load version %s (fallback to main also failed: %w)", version, err)
	}

	return metadata, nil
}

// tryLoad attempts to load from file, then HTTP if file not found.
func (l *loader) tryLoad(ref Ref, version string) (*ConfigMetadata, error) {
	if l.schemasDir != "" {
		metadata, err := l.loadFromFile(ref, version)
		if err == nil {
			return metadata, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return nil, err // Real error (parse error, permission denied, etc.)
		}
	}

	metadata, err := l.loadFromHTTP(ref, version)
	if err != nil {
		return nil, err
	}

	if l.schemasDir != "" {
		_ = l.persistToFile(ref, version, metadata)
	}

	return metadata, nil
}

// loadFromFile loads a schema from the local filesystem.
func (l *loader) loadFromFile(ref Ref, version string) (*ConfigMetadata, error) {
	filePath := filepath.Join(l.schemasDir, version, filepath.FromSlash(ref.PkgPath()), schemaFileName)

	body, err := os.ReadFile(filePath) // #nosec G304
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to read schema from %s: %w", filePath, err)
	}

	var metadata ConfigMetadata
	if err := yaml.Unmarshal(body, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse schema from %s: %w", filePath, err)
	}

	return &metadata, nil
}

// loadFromHTTP fetches a schema from HTTP.
func (l *loader) loadFromHTTP(ref Ref, version string) (*ConfigMetadata, error) {
	url := getRefURL(ref, version)

	resp, err := l.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch schema from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch schema from %s: HTTP %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body from %s: %w", url, err)
	}

	var metadata ConfigMetadata
	if err := yaml.Unmarshal(body, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse schema from %s: %w", url, err)
	}

	return &metadata, nil
}

// persistToFile saves a schema to the filesystem.
func (l *loader) persistToFile(ref Ref, version string, md *ConfigMetadata) error {
	filePath := filepath.Join(l.schemasDir, version, filepath.FromSlash(ref.PkgPath()), schemaFileName)

	if err := os.MkdirAll(filepath.Dir(filePath), 0o750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := yaml.Marshal(md)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// cacheKey generates a cache key from ref and version.
func cacheKey(ref Ref, version string) string {
	return ref.PkgPath() + "@" + version
}
