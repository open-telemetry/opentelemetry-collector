// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/debugexporter/internal"

type AttributesOutputConfig struct {
	Enabled bool     `mapstructure:"enabled"`
	Include []string `mapstructure:"include"`
	Exclude []string `mapstructure:"exclude"`
}

type RecordOutputConfig struct {
	Enabled                bool `mapstructure:"enabled"`
	AttributesOutputConfig `mapstructure:"attributes"`
}

type ScopeOutputConfig struct {
	Enabled                bool `mapstructure:"enabled"`
	AttributesOutputConfig `mapstructure:"attributes"`
}

type ResourceOutputConfig struct {
	Enabled                bool                   `mapstructure:"enabled"`
	AttributesOutputConfig AttributesOutputConfig `mapstructure:"attributes"`
}

type OutputConfig struct {
	Record   RecordOutputConfig   `mapstructure:"record"`
	Scope    ScopeOutputConfig    `mapstructure:"scope"`
	Resource ResourceOutputConfig `mapstructure:"resource"`
}

func NewDefaultOutputConfig() OutputConfig {
	return OutputConfig{
		Record: RecordOutputConfig{
			Enabled: true,
			AttributesOutputConfig: AttributesOutputConfig{
				Enabled: true,
			},
		},
		Resource: ResourceOutputConfig{
			Enabled: true,
			AttributesOutputConfig: AttributesOutputConfig{
				Enabled: true,
			},
		},
		Scope: ScopeOutputConfig{
			Enabled: true,
			AttributesOutputConfig: AttributesOutputConfig{
				Enabled: true,
			},
		},
	}
}
