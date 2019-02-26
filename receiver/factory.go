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

package receiver

import (
	"github.com/census-instrumentation/opencensus-service/processor"
	"github.com/spf13/viper"
)

// TraceReceiverFactory is an interface that builds a new TraceReceiver based on
// some viper.Viper configuration.
type TraceReceiverFactory interface {
	// Type gets the type of the TraceReceiver created by this factory.
	Type() string
	// NewFromViper takes a viper.Viper config and creates a new TraceReceiver which uses next as the
	// next TraceDataProcessor in the pipeline.
	NewFromViper(cfg *viper.Viper, next processor.TraceDataProcessor) (TraceReceiver, error)
	// DefaultConfig returns the default configuration for TraceReceivers
	// created by this factory.
	DefaultConfig() *viper.Viper
}

// MetricsReceiverFactory is an interface that builds a new MetricsReceiver based on
// some viper.Viper configuration.
type MetricsReceiverFactory interface {
	// Type gets the type of the MetricsReceiver created by this factory.
	Type() string
	// NewFromViper takes a viper.Viper config and creates a new MetricsReceiver which uses next as the
	// next MetricsDataProcessor in the pipeline.
	NewFromViper(cfg *viper.Viper, next processor.MetricsDataProcessor) (MetricsReceiver, error)
	// DefaultConfig returns the default configuration for MetricsReceivers
	// created by this factory.
	DefaultConfig() *viper.Viper
}
