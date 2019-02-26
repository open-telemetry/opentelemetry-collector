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

package exporter

import (
	"github.com/spf13/viper"

	"github.com/census-instrumentation/opencensus-service/processor"
)

// TraceDataExporter composes TraceDataProcessor with some additional
// exporter-specific functions. This helps the service core to identify which
// TraceDataProcessors are Exporters and which are internal processing
// components, so that better validation of pipelines can be done.
type TraceDataExporter interface {
	processor.TraceDataProcessor

	// ExportFormat gets the name of the format in which this exporter sends its data.
	ExportFormat() string
}

// TraceDataExporterFactory is an interface that builds a new TraceDataExporter based on
// some viper.Viper configuration.
type TraceDataExporterFactory interface {
	// Type gets the type of the TraceDataExporter created by this factory.
	Type() string
	// NewFromViper takes a viper.Viper config and creates a new TraceDataExporter.
	NewFromViper(cfg *viper.Viper) (TraceDataExporter, error)
	// DefaultConfig returns the default configuration for TraceDataExporter
	// created by this factory.
	DefaultConfig() *viper.Viper
}

// MetricsDataExporter composes MetricsDataProcessor with some additional
// exporter-specific functions. This helps the service core to identify which
// MetricsDataProcessors are Exporters and which are internal processing
// components, so that better validation of pipelines can be done.
type MetricsDataExporter interface {
	processor.MetricsDataProcessor

	// ExportFormat gets the name of the format in which this exporter sends its data.
	ExportFormat() string
}

// MetricsDataExporterFactory is an interface that builds a new MetricsDataExporter based on
// some viper.Viper configuration.
type MetricsDataExporterFactory interface {
	// Type gets the type of the MetricsDataExporter created by this factory.
	Type() string
	// NewFromViper takes a viper.Viper config and creates a new MetricsDataExporter.
	NewFromViper(cfg *viper.Viper) (MetricsDataExporter, error)
	// DefaultConfig returns the default configuration for MetricsDataExporter
	// created by this factory.
	DefaultConfig() *viper.Viper
}
