// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

// SchemaConfig holds configuration for YAML schema generation.
type SchemaConfig struct {
	// Enabled controls whether schema generation is enabled (default: false).
	Enabled bool `mapstructure:"enabled"`

	// ConfigType specifies the config type to generate schema for.
	// Can be a simple type name (e.g., "Config") for local package,
	// or a fully qualified type (e.g., "go.opentelemetry.io/collector/pkg.Config")
	// for external packages. If empty, auto-detection is used.
	ConfigType string `mapstructure:"config_type"`
}
