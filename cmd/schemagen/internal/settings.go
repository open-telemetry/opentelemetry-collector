// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/schemagen/internal"

import (
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

const (
	SettingsFileName = ".schemagen.yaml"
)

type (
	Settings struct {
		Namespace          string             `yaml:"namespace"`
		Mappings           Mappings           `yaml:"mappings"`
		ComponentOverrides ComponentOverrides `yaml:"componentOverrides"`
		AllowedRefs        []string           `yaml:"allowedRefs"`
	}
	Mappings           map[string]PackagesMapping
	PackagesMapping    map[string]TypeDesc
	ComponentOverrides map[string]ComponentOverride
	ComponentOverride  struct {
		ConfigName  string           `yaml:"configName"`
		FactoryMaps []FactoryMapSpec `yaml:"factoryMaps,omitempty"`
		OverlayFile string           `yaml:"overlayFile,omitempty"`
	}
	FactoryMapSpec struct {
		// Property is the JSON-schema key under the root config object where the
		// synthetic object is inserted. Example: "scrapers".
		Property string `yaml:"property"`
		// FactoriesVar is the name of the package-level var holding the factory map literal.
		FactoriesVar string `yaml:"factoriesVar"`
		// Description is attached to the synthetic property.
		Description string `yaml:"description,omitempty"`
		// KeyFromMetadataVar is the var name in each factory's internal/metadata package
		// that contains the discriminator key (default: "Type").
		KeyFromMetadataVar string `yaml:"keyFromMetadataVar,omitempty"`
	}
	TypeDesc struct {
		SchemaType     SchemaType `yaml:"schemaType"`
		Format         string     `yaml:"format"`
		SkipAnnotation bool       `yaml:"skipAnnotation"`
	}
)

// ReadSettingsFile searches upward for a .schemagen.yaml and returns the parsed
// Settings together with the directory in which the file was found.
func ReadSettingsFile() (*Settings, string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, "", false
	}

	for {
		candidate := filepath.Join(dir, SettingsFileName)
		data, err := os.ReadFile(candidate) // #nosec G304
		if err == nil {
			var s Settings
			if err := yaml.Unmarshal(data, &s); err == nil {
				fmt.Println("Settings file read from: ", candidate)
				return &s, dir, true
			}
			fmt.Println("Warning: failed to parse config file:", candidate)
			return nil, "", false
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return nil, "", false
}
