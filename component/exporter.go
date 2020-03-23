// Copyright  OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
)

// Exporter defines functions that trace and metric exporters must implement.
type Exporter interface {
	Component
}

// TraceExporterOld is a TraceConsumer that is also an Exporter.
type TraceExporterOld interface {
	consumer.TraceConsumerOld
	Exporter
}

// TraceExporter is an TraceConsumer that is also an Exporter.
type TraceExporter interface {
	consumer.TraceConsumer
	Exporter
}

// MetricsExporterOld is a MetricsConsumer that is also an Exporter.
type MetricsExporterOld interface {
	consumer.MetricsConsumerOld
	Exporter
}

// MetricsExporter is a MetricsConsumer that is also an Exporter.
type MetricsExporter interface {
	consumer.MetricsConsumer
	Exporter
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

// ExporterCreateParams is passed to ExporterFactory.Create* functions.
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
