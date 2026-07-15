// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/schemagen/internal"

import (
	"os"
	"path"
	"path/filepath"

	"go.yaml.in/yaml/v3"
	"golang.org/x/tools/go/packages"
)

type Metadata struct {
	Type   string `mapstructure:"type"`
	Status struct {
		Class string `mapstructure:"class"`
	} `mapstructure:"status"`
	Parent string `mapstructure:"parent"`
}

// ResolvePackageDir returns the on-disk source directory of the package matched by pattern,
// resolved relative to dir.
//
// Returns false if the directory cannot be determined (pattern unresolvable, no Go source files, etc.),
// in which case the caller should fall back to dir.
func ResolvePackageDir(dir, pattern string) (string, bool) {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedFiles | packages.NeedModule,
		Dir:  dir,
	}, pattern)
	if err != nil || len(pkgs) == 0 || len(pkgs[0].GoFiles) == 0 {
		return "", false
	}
	return filepath.Dir(pkgs[0].GoFiles[0]), true
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
