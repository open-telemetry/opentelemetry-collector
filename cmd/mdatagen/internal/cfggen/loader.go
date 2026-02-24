// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/cfggen"

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
	"golang.org/x/tools/go/packages"
)

const (
	defaultVersion = "main"
	schemaFileName = "config.schema.yaml"
)

// ErrNotFound indicates a schema was not found by any source.
var ErrNotFound = errors.New("schema not found")

// Loader loads configuration metadata from various sources (file, HTTP).
type Loader interface {
	Load(ref Ref) (*ConfigMetadata, error)
}

type schemaLoader struct {
	cache      map[string]*ConfigMetadata
	cd         string
	rootDir    string
	httpClient *http.Client
}

// NewLoader creates a fully configured loader. Takes current component's directory to determine where to look for local schema files.
func NewLoader(cwd string) Loader {
	return &schemaLoader{
		cache:      make(map[string]*ConfigMetadata),
		cd:         cwd,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (sl *schemaLoader) Load(ref Ref) (*ConfigMetadata, error) {
	if cached, ok := sl.cache[ref.CacheKey()]; ok {
		return cached, nil
	}

	metadata, err := sl.load(ref)
	if err != nil {
		return nil, err
	}

	sl.cache[ref.CacheKey()] = metadata
	return metadata, nil
}

func (sl *schemaLoader) load(ref Ref) (*ConfigMetadata, error) {
	repoRoot, err := sl.repoRoot(sl.cd)
	if err != nil {
		return nil, err
	}

	if ref.isLocal() {
		var filePath string
		if strings.HasPrefix(ref.schemaID, "/") {
			filePath = filepath.Join(repoRoot, ref.SchemaID(), schemaFileName)
		} else {
			filePath = filepath.Join(sl.cd, ref.SchemaID(), schemaFileName)
		}
		return sl.loadFromFile(filePath)
	}

	return sl.loadFromHTTP(ref, filepath.Join(repoRoot, ".schemas"))
}

func (sl *schemaLoader) loadFromFile(filePath string) (*ConfigMetadata, error) {
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

func (sl *schemaLoader) loadFromHTTP(ref Ref, fileCacheDir string) (*ConfigMetadata, error) {
	version := sl.refVersion(&ref)
	filePath := filepath.Join(fileCacheDir, version, filepath.FromSlash(ref.SchemaID()), schemaFileName)
	// check fs cache first
	metadata, err := sl.loadFromFile(filePath)
	if err == nil {
		return metadata, nil
	}
	if !errors.Is(err, ErrNotFound) {
		log.Printf("warning: failed to load schema from file cache at %s: %v", filePath, err)
	}
	metadata, err = sl.tryLoad(ref, version)
	if err != nil {
		return nil, err
	}
	if err := sl.persistToFile(filePath, metadata); err != nil {
		log.Printf("warning: failed to persist schema to file cache at %s: %v", filePath, err)
	}
	return metadata, nil
}

func (sl *schemaLoader) tryLoad(ref Ref, version string) (*ConfigMetadata, error) {
	url, err := ref.URL(version)
	if err != nil {
		return nil, fmt.Errorf("failed to construct URL for %s: %w", ref.CacheKey(), err)
	}

	resp, err := sl.httpClient.Get(url)
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

func (sl *schemaLoader) persistToFile(filePath string, md *ConfigMetadata) error {
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

func (sl *schemaLoader) refVersion(ref *Ref) string {
	if v := ref.InlineVersion(); v != "" {
		return v
	}

	// Try to resolve version via packages.Load (respects replace directives)
	modulePath := ref.Module()
	if modulePath != "" {
		if version := sl.resolveModuleVersion(modulePath); version != "" {
			return version
		}
	}

	log.Printf("warning: could not resolve version for %s, falling back to %q", ref.CacheKey(), defaultVersion)
	return defaultVersion
}

func (sl *schemaLoader) resolveModuleVersion(importPath string) string {
	cfg := &packages.Config{
		Mode: packages.NeedModule,
		Dir:  sl.cd,
	}
	pkgs, err := packages.Load(cfg, importPath)
	if err != nil || len(pkgs) == 0 {
		return ""
	}

	pkg := pkgs[0]
	if pkg.Module == nil {
		return ""
	}

	return pkg.Module.Version
}

func (sl *schemaLoader) repoRoot(componentDir string) (string, error) {
	if sl.rootDir != "" {
		return sl.rootDir, nil
	}

	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = componentDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to determine repo root: %w", err)
	}
	sl.rootDir = strings.TrimSpace(string(output))

	return sl.rootDir, nil
}
