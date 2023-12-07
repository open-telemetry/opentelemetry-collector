// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config // import "go.opentelemetry.io/collector/cmd/builder/internal/config"

import (
	"embed"

	"github.com/knadh/koanf/providers/fs"
	"github.com/knadh/koanf/v2"
)

//go:embed *.yaml
var configs embed.FS

// DefaultProvider returns a koanf.Provider that provides the default build
// configuration file. This is the same configuration that otelcorecol is
// built with.
func DefaultProvider() koanf.Provider {
	return fs.Provider(configs, "default.yaml")
}
