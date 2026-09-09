// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/schemagen/internal"

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// ApplyOverlayToYAML deep-merges the overlay YAML file into the provided schema bytes (expected to be YAML) and returns the merged result as YAML.
// It is a no-op when no overlay is configured, returning the input bytes unchanged.
func ApplyOverlayToYAML(schemaBytes []byte, cfg *Config) ([]byte, error) {
	if cfg.ComponentOverride == nil || cfg.ComponentOverride.OverlayFile == "" {
		return schemaBytes, nil
	}

	path, err := resolveOverlayPath(cfg)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		return nil, fmt.Errorf("overlay file %s: %w", path, err)
	}

	var overlay map[string]any
	if err := yaml.Unmarshal(data, &overlay); err != nil {
		return nil, fmt.Errorf("overlay file %s: %w", path, err)
	}

	var base map[string]any
	if err := yaml.Unmarshal(schemaBytes, &base); err != nil {
		return nil, fmt.Errorf("base schema: %w", err)
	}

	merged := deepMerge(base, overlay)
	return marshalYAML(merged)
}

// ApplyOverlayToJSON deep-merges the overlay YAML file into the provided schema bytes (expected to be JSON) and returns the merged result as JSON.
// It is a no-op when no overlay is configured, returning the input bytes unchanged.
func ApplyOverlayToJSON(schemaBytes []byte, cfg *Config) ([]byte, error) {
	if cfg.ComponentOverride == nil || cfg.ComponentOverride.OverlayFile == "" {
		return schemaBytes, nil
	}

	path, err := resolveOverlayPath(cfg)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		return nil, fmt.Errorf("overlay file %s: %w", path, err)
	}

	var overlay map[string]any
	if err := yaml.Unmarshal(data, &overlay); err != nil {
		return nil, fmt.Errorf("overlay file %s: %w", path, err)
	}

	var base map[string]any
	if err := json.Unmarshal(schemaBytes, &base); err != nil {
		return nil, fmt.Errorf("base schema: %w", err)
	}

	merged := deepMerge(base, overlay)
	return json.MarshalIndent(merged, "", "  ")
}

func marshalYAML(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func deepMerge(dst, src map[string]any) map[string]any {
	if dst == nil {
		dst = make(map[string]any)
	}
	for k, sv := range src {
		dv, exists := dst[k]
		if !exists {
			dst[k] = sv
			continue
		}
		dm, dIsMap := dv.(map[string]any)
		sm, sIsMap := sv.(map[string]any)
		if dIsMap && sIsMap {
			dst[k] = deepMerge(dm, sm)
		} else {
			dst[k] = sv
		}
	}
	return dst
}

func resolveOverlayPath(cfg *Config) (string, error) {
	overlayFile := cfg.ComponentOverride.OverlayFile

	if filepath.IsAbs(overlayFile) {
		if fileExists(overlayFile) {
			return overlayFile, nil
		}
		return "", fmt.Errorf("overlay file not found: %s", overlayFile)
	}

	// Try relative to component dir first.
	if cfg.DirPath != "" {
		candidate := filepath.Join(cfg.DirPath, overlayFile)
		if fileExists(candidate) {
			return candidate, nil
		}
	}

	// Fall back to settings dir (repo root).
	if cfg.SettingsDir != "" {
		candidate := filepath.Join(cfg.SettingsDir, overlayFile)
		if fileExists(candidate) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("overlay file not found: %s (tried relative to %s and %s)", overlayFile, cfg.DirPath, cfg.SettingsDir)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
