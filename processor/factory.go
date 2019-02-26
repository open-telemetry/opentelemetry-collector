// Copyright 2019, OpenCensus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package processor

import "github.com/spf13/viper"

// TraceDataProcessorFactory is an interface that builds a new TraceDataProcessor based on
// some viper.Viper configuration.
type TraceDataProcessorFactory interface {
	// Type gets the type of the TraceDataProcessor created by this factory.
	Type() string
	// NewFromViper takes a viper.Viper config and creates a new TraceDataProcessor which uses next as
	// the next TraceDataProcessor in the pipeline.
	NewFromViper(cfg *viper.Viper, next TraceDataProcessor) (TraceDataProcessor, error)
	// DefaultConfig returns the default configuration for TraceDataProcessors
	// created by this factory.
	DefaultConfig() *viper.Viper
}

// MetricsDataProcessorFactory is an interface that builds a new MetricsDataProcessor based on
// some viper.Viper configuration.
type MetricsDataProcessorFactory interface {
	// Type gets the type of the MetricsDataProcessor created by this factory.
	Type() string
	// NewFromViper takes a viper.Viper config and creates a new MetricsDataProcessor which uses next as
	// the next MetricsDataProcessor in the pipeline.
	NewFromViper(cfg *viper.Viper, next MetricsDataProcessor) (MetricsDataProcessor, error)
	// DefaultConfig returns the default configuration for MetricsDataProcessors
	// created by this factory.
	DefaultConfig() *viper.Viper
}
