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

package component // import "go.opentelemetry.io/collector/component"

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
)

// Deprecated: [v0.67.0] use exporter.Traces.
type TracesExporter interface {
	Component
	consumer.Traces
}

// Deprecated: [v0.67.0] use exporter.Metrics.
type MetricsExporter interface {
	Component
	consumer.Metrics
}

// Deprecated: [v0.67.0] use exporter.Logs.
type LogsExporter interface {
	Component
	consumer.Logs
}

// Deprecated: [v0.67.0] use exporter.CreateSettings.
type ExporterCreateSettings struct {
	// ID returns the ID of the component that will be created.
	ID ID

	TelemetrySettings

	// BuildInfo can be used by components for informational purposes
	BuildInfo BuildInfo
}

// Deprecated: [v0.67.0] use exporter.Factory.
type ExporterFactory interface {
	Factory

	// CreateTracesExporter creates a TracesExporter based on this config.
	// If the exporter type does not support tracing or if the config is not valid,
	// an error will be returned instead.
	CreateTracesExporter(ctx context.Context, set ExporterCreateSettings, cfg Config) (TracesExporter, error)

	// TracesExporterStability gets the stability level of the TracesExporter.
	TracesExporterStability() StabilityLevel

	// CreateMetricsExporter creates a MetricsExporter based on this config.
	// If the exporter type does not support metrics or if the config is not valid,
	// an error will be returned instead.
	CreateMetricsExporter(ctx context.Context, set ExporterCreateSettings, cfg Config) (MetricsExporter, error)

	// MetricsExporterStability gets the stability level of the MetricsExporter.
	MetricsExporterStability() StabilityLevel

	// CreateLogsExporter creates a LogsExporter based on the config.
	// If the exporter type does not support logs or if the config is not valid,
	// an error will be returned instead.
	CreateLogsExporter(ctx context.Context, set ExporterCreateSettings, cfg Config) (LogsExporter, error)

	// LogsExporterStability gets the stability level of the LogsExporter.
	LogsExporterStability() StabilityLevel
}
