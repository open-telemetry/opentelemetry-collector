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

// TracesExporter is a Exporter that can consume traces.
type TracesExporter interface {
	Exporter
	consumer.TracesConsumer
}

// MetricsExporter is an Exporter that can consume metrics.
type MetricsExporter interface {
	Exporter
	consumer.MetricsConsumer
}

// LogsExporter is an Exporter that can consume logs.
type LogsExporter interface {
	Exporter
	consumer.LogsConsumer
}

// ExporterCreateParams is passed to Create*Exporter functions.
type ExporterCreateParams struct {
	// Logger that the factory can use during creation and can pass to the created
	// component to be used later as well.
	Logger *zap.Logger

	// ApplicationStartInfo can be used by components for informational purposes
	ApplicationStartInfo ApplicationStartInfo
}

// ExporterFactory can create TracesExporter and MetricsExporter. This is the
// new factory type that can create new style exporters.
type ExporterFactory interface {
	Factory

	// CreateDefaultConfig creates the default configuration for the Exporter.
	// This method can be called multiple times depending on the pipeline
	// configuration and should not cause side-effects that prevent the creation
	// of multiple instances of the Exporter.
	// The object returned by this method needs to pass the checks implemented by
	// 'configcheck.ValidateConfig'. It is recommended to have such check in the
	// tests of any implementation of the Factory interface.
	CreateDefaultConfig() configmodels.Exporter

	// CreateTracesExporter creates a trace exporter based on this config.
	// If the exporter type does not support tracing or if the config is not valid
	// error will be returned instead.
	CreateTracesExporter(
		ctx context.Context,
		params ExporterCreateParams,
		cfg configmodels.Exporter,
	) (TracesExporter, error)

	// CreateMetricsExporter creates a metrics exporter based on this config.
	// If the exporter type does not support metrics or if the config is not valid
	// error will be returned instead.
	CreateMetricsExporter(
		ctx context.Context,
		params ExporterCreateParams,
		cfg configmodels.Exporter,
	) (MetricsExporter, error)

	// CreateLogsExporter creates an exporter based on the config.
	// If the exporter type does not support logs or if the config is not valid
	// error will be returned instead.
	CreateLogsExporter(
		ctx context.Context,
		params ExporterCreateParams,
		cfg configmodels.Exporter,
	) (LogsExporter, error)
}
