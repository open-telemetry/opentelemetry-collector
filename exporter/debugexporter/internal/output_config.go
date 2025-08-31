// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/debugexporter/internal"

type Attributes struct {
	Enabled bool     `mapstructure:"enabled"`
	Include []string `mapstructure:"include"`
	Exclude []string `mapstructure:"exclude"`
}

type RecordOutputConfig struct {
	Enabled    bool `mapstructure:"enabled"`
	Attributes `mapstructure:"attributes"`
}

type ScopeOutputConfig struct {
	Enabled    bool `mapstructure:"enabled"`
	Attributes `mapstructure:"attributes"`
}

type ResourceOutputConfig struct {
	Enabled                bool       `mapstructure:"enabled"`
	AttributesOutputConfig Attributes `mapstructure:"attributes"`
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
			Attributes: Attributes{
				Enabled: true,
			},
		},
		Resource: ResourceOutputConfig{
			Enabled: true,
			AttributesOutputConfig: Attributes{
				Enabled: true,
			},
		},
		Scope: ScopeOutputConfig{
			Enabled: true,
			Attributes: Attributes{
				Enabled: true,
			},
		},
	}
}
