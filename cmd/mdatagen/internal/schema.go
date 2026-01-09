// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

// SchemaConfig holds configuration for YAML schema generation.
type SchemaConfig struct {
	// Enabled controls whether schema generation is enabled (default: false).
	Enabled bool `mapstructure:"enabled"`
}
