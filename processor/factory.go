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

// TraceProcessorFactory is an interface that builds a new TraceProcessor based on
// some viper.Viper configuration.
type TraceProcessorFactory interface {
	// Type gets the type of the TraceProcessor created by this factory.
	Type() string
	// NewFromViper takes a viper.Viper config and creates a new TraceProcessor which uses next as
	// the next TraceProcessor in the pipeline.
	NewFromViper(cfg *viper.Viper, next TraceProcessor) (TraceProcessor, error)
	// DefaultConfig returns the default configuration for TraceProcessors
	// created by this factory.
	DefaultConfig() *viper.Viper
}

// MetricsProcessorFactory is an interface that builds a new MetricsProcessor based on
// some viper.Viper configuration.
type MetricsProcessorFactory interface {
	// Type gets the type of the MetricsProcessor created by this factory.
	Type() string
	// NewFromViper takes a viper.Viper config and creates a new MetricsProcessor which uses next as
	// the next MetricsProcessor in the pipeline.
	NewFromViper(cfg *viper.Viper, next MetricsProcessor) (MetricsProcessor, error)
	// DefaultConfig returns the default configuration for MetricsProcessors
	// created by this factory.
	DefaultConfig() *viper.Viper
}
