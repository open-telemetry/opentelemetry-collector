// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zpagesextension // import "go.opentelemetry.io/collector/extension/zpagesextension"

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
)

// Config has the configuration for the extension enabling the zPages extension.
type Config struct {
	ServerConfig confighttp.ServerConfig `mapstructure:",squash"`

	Expvar ExpvarConfig `mapstructure:"expvar"`

	// SupportBundle configures the support bundle endpoint.
	SupportBundle SupportBundleConfig `mapstructure:"support_bundle"`
	// prevent unkeyed literal initialization
	_ struct{}
}

// ExpvarConfig has the configuration for the expvar service.
type ExpvarConfig struct {
	// Enabled indicates whether to enable expvar service.
	// (default = false)
	Enabled bool `mapstructure:"enabled"`
	// prevent unkeyed literal initialization
	_ struct{}
}

// SupportBundleConfig has the configuration for the support bundle endpoint.
type SupportBundleConfig struct {
	// Enabled indicates whether to serve support bundles at /debug/v1/supportbundle.
	// (default = false)
	Enabled bool `mapstructure:"enabled"`
	// ArchiveName names the archive. The download file name starts with it, and
	// it is the name of the top directory in the archive.
	// (default = otelcol-support-bundle)
	ArchiveName string `mapstructure:"archive_name"`
	// Gatherers enables or disables each gatherer. A request can select only
	// enabled gatherers.
	Gatherers GatherersConfig `mapstructure:"gatherers"`
	// prevent unkeyed literal initialization
	_ struct{}
}

// GatherersConfig has the configuration for each support bundle gatherer.
type GatherersConfig struct {
	// PProf collects point-in-time heap and goroutine profiles.
	PProf GathererConfig `mapstructure:"pprof"`
	// prevent unkeyed literal initialization
	_ struct{}
}

// GathererConfig has the configuration that every gatherer shares.
type GathererConfig struct {
	// Enabled indicates whether the gatherer can run.
	// (default = true)
	Enabled bool `mapstructure:"enabled"`
	// prevent unkeyed literal initialization
	_ struct{}
}

var _ component.Config = (*Config)(nil)

// Validate checks if the support bundle configuration is valid.
func (cfg *SupportBundleConfig) Validate() error {
	if !cfg.Enabled {
		return nil
	}
	// The archive must have exactly one top directory.
	if !fs.ValidPath(cfg.ArchiveName) || cfg.ArchiveName == "." || strings.ContainsAny(cfg.ArchiveName, `/\`) {
		return fmt.Errorf("\"archive_name\" must be a valid file name with no path separators, got %q", cfg.ArchiveName)
	}
	return nil
}

// Validate checks if the extension configuration is valid
func (cfg *Config) Validate() error {
	if cfg.ServerConfig.NetAddr.Endpoint == "" {
		return errors.New("\"endpoint\" is required when using the \"zpages\" extension")
	}
	return nil
}
