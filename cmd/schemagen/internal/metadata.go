// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/schemagen/internal"

import (
	"os"
	"path"

	"go.yaml.in/yaml/v3"
)

type Metadata struct {
	Type   string `mapstructure:"type"`
	Status struct {
		Class string `mapstructure:"class"`
	} `mapstructure:"status"`
	Parent string `mapstructure:"parent"`
}

func ReadMetadata(dir string) (*Metadata, bool) {
	mdPath := path.Join(dir, "metadata.yaml")
	data, err := os.ReadFile(mdPath) // #nosec G304
	if err == nil {
		var m Metadata
		if err := yaml.Unmarshal(data, &m); err == nil {
			return &m, true
		}
	}
	return nil, false
}
