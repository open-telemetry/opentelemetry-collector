// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

const centralConfigFileName = ".mdatagen.yaml"

// CentralConfig is the repository-level configuration for mdatagen.
type CentralConfig struct {
	Exclusions []ComponentExclusion `yaml:"exclusions"`
}

type ComponentExclusion struct {
	// Component Go import path.
	Component     string                 `yaml:"component"`
	FeatureGates  []FeatureGateExclusion `yaml:"feature_gates"`
	LifecycleTest Toggle                 `yaml:"lifecycle_test"`
	ShutdownTest  Toggle                 `yaml:"shutdown_test"`
	Goleak        Toggle                 `yaml:"goleak"`
}

type FeatureGateExclusion struct {
	// Name is the feature gate ID.
	Name       string `yaml:"name"`
	Validation Toggle `yaml:"validation"`
}

// Toggle enables or disables a check; a nil Enabled keeps the default (enabled).
type Toggle struct {
	Enabled *bool `yaml:"enabled"`
}

func (t Toggle) disabled() bool {
	return t.Enabled != nil && !*t.Enabled
}

// applyTo folds the central config into md. Call it after md.PackageName is set
// and before md.Validate so feature-gate exemptions apply during validation.
func (c *CentralConfig) applyTo(md *Metadata) {
	for _, cc := range c.Exclusions {
		if cc.Component != md.PackageName {
			continue
		}
		md.Tests.SkipLifecycle = md.Tests.SkipLifecycle || cc.LifecycleTest.disabled()
		md.Tests.SkipShutdown = md.Tests.SkipShutdown || cc.ShutdownTest.disabled()
		md.Tests.GoLeak.Skip = md.Tests.GoLeak.Skip || cc.Goleak.disabled()
		for _, fg := range cc.FeatureGates {
			if !fg.Validation.disabled() {
				continue
			}
			for i := range md.FeatureGates {
				if string(md.FeatureGates[i].ID) == fg.Name {
					md.FeatureGates[i].SkipStrictValidation = true
				}
			}
		}
	}
}

// loadCentralConfig loads the nearest .mdatagen.yaml at or above startDir, or an
// empty config if none exists.
func loadCentralConfig(startDir string) (*CentralConfig, error) {
	path, err := findCentralConfig(startDir)
	if err != nil {
		return nil, err
	}
	if path == "" {
		return &CentralConfig{}, nil
	}

	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("failed reading central config %v: %w", path, err)
	}

	cfg := &CentralConfig{}
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	if err := dec.Decode(cfg); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("failed parsing central config %v: %w", path, err)
	}
	return cfg, nil
}

// findCentralConfig returns the nearest .mdatagen.yaml at or above startDir,
// bounded by the repository root (.git). Empty string if none.
func findCentralConfig(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, centralConfigFileName)
		switch _, statErr := os.Stat(candidate); {
		case statErr == nil:
			return candidate, nil
		case !errors.Is(statErr, fs.ErrNotExist):
			return "", fmt.Errorf("failed to stat %v: %w", candidate, statErr)
		}

		// Stop at the repo root so discovery never escapes into e.g. the home directory.
		if _, statErr := os.Stat(filepath.Join(dir, ".git")); statErr == nil {
			return "", nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}
