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

package component

import (
	"context"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
)

// Exporter defines functions that all exporters must implement.
type Exporter interface {
	Component
}

// TraceExporterBase defines a common interface for TraceExporter and TraceExporterOld
type TraceExporterBase interface {
	Exporter
}

// TraceExporterOld is a TraceExporter that can consume old-style traces.
type TraceExporterOld interface {
	consumer.TraceConsumerOld
	TraceExporterBase
}

// TraceExporter is a TraceExporter that can consume new-style traces.
type TraceExporter interface {
	consumer.TraceConsumer
	TraceExporterBase
}

// MetricsExporterBase defines a common interface for MetricsExporter and MetricsExporterOld
type MetricsExporterBase interface {
	Exporter
}

// MetricsExporterOld is a TraceExporter that can consume old-style metrics.
type MetricsExporterOld interface {
	consumer.MetricsConsumerOld
	MetricsExporterBase
}

// MetricsExporter is a TraceExporter that can consume new-style metrics.
type MetricsExporter interface {
	consumer.MetricsConsumer
	MetricsExporterBase
}

// LogsExporter is a LogsConsumer that is also an Exporter.
type LogsExporter interface {
	Exporter
	consumer.LogsConsumer
}

// ExporterFactoryBase defines the common functions for all exporter factories.
type ExporterFactoryBase interface {
	Factory

	// CreateDefaultConfig creates the default configuration for the Exporter.
	// This method can be called multiple times depending on the pipeline
	// configuration and should not cause side-effects that prevent the creation
	// of multiple instances of the Exporter.
	// The object returned by this method needs to pass the checks implemented by
	// 'configcheck.ValidateConfig'. It is recommended to have such check in the
	// tests of any implementation of the Factory interface.
	CreateDefaultConfig() configmodels.Exporter
}

// ExporterFactoryOld can create TraceExporterOld and MetricsExporterOld.
type ExporterFactoryOld interface {
	ExporterFactoryBase

	// CreateTraceExporter creates a trace exporter based on this config.
	CreateTraceExporter(logger *zap.Logger, cfg configmodels.Exporter) (TraceExporterOld, error)

	// CreateMetricsExporter creates a metrics exporter based on this config.
	CreateMetricsExporter(logger *zap.Logger, cfg configmodels.Exporter) (MetricsExporterOld, error)
}

// ExporterCreateParams is passed to Create*Exporter functions.
type ExporterCreateParams struct {
	// Logger that the factory can use during creation and can pass to the created
	// component to be used later as well.
	Logger *zap.Logger
}

// ExporterFactory can create TraceExporter and MetricsExporter. This is the
// new factory type that can create new style exporters.
type ExporterFactory interface {
	ExporterFactoryBase

	// CreateTraceExporter creates a trace exporter based on this config.
	// If the exporter type does not support tracing or if the config is not valid
	// error will be returned instead.
	CreateTraceExporter(ctx context.Context, params ExporterCreateParams,
		cfg configmodels.Exporter) (TraceExporter, error)

	// CreateMetricsExporter creates a metrics exporter based on this config.
	// If the exporter type does not support metrics or if the config is not valid
	// error will be returned instead.
	CreateMetricsExporter(ctx context.Context, params ExporterCreateParams,
		cfg configmodels.Exporter) (MetricsExporter, error)
}

// LogsExporterFactory can create a LogsExporter.
type LogsExporterFactory interface {
	ExporterFactoryBase

	// CreateLogsExporter creates an exporter based on the config.
	// If the exporter type does not support logs or if the config is not valid
	// error will be returned instead.
	CreateLogsExporter(
		ctx context.Context,
		params ExporterCreateParams,
		cfg configmodels.Exporter,
	) (LogsExporter, error)
}
