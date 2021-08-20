// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filterprocessor

import (
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/internal/processor/filterconfig"
	"go.opentelemetry.io/collector/internal/processor/filtermetric"
)

// Config defines configuration for Resource processor.
type Config struct {
	config.ProcessorSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct

	Metrics MetricFilters `mapstructure:"metrics"`

	Logs LogFilters `mapstructure:"logs"`
}

// MetricFilters filters by Metric properties.
type MetricFilters struct {
	// Include match properties describe metrics that should be included in the Collector Service pipeline,
	// all other metrics should be dropped from further processing.
	// If both Include and Exclude are specified, Include filtering occurs first.
	Include *filtermetric.MatchProperties `mapstructure:"include"`

	// Exclude match properties describe metrics that should be excluded from the Collector Service pipeline,
	// all other metrics should be included.
	// If both Include and Exclude are specified, Include filtering occurs first.
	Exclude *filtermetric.MatchProperties `mapstructure:"exclude"`
}

// LogFilters filters by Log properties.
type LogFilters struct {
	// ResourceAttributes defines a list of possible resource attributes to match logs against.
	// A match occurs if any resource attribute matches at least one expression in this given list.
	ResourceAttributes []filterconfig.Attribute `mapstructure:"resource_attributes"`
}

var _ config.Processor = (*Config)(nil)

// Validate checks if the processor configuration is valid
func (cfg *Config) Validate() error {
	return nil
}
