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

// Deprecated: [v0.68.0] use processor.Traces.
type TracesProcessor interface {
	Component
	consumer.Traces
}

// Deprecated: [v0.68.0] use processor.Metrics.
type MetricsProcessor interface {
	Component
	consumer.Metrics
}

// Deprecated: [v0.68.0] use processor.Logs.
type LogsProcessor interface {
	Component
	consumer.Logs
}

// Deprecated: [v0.68.0] use processor.CreateSettings.
type ProcessorCreateSettings struct {
	// ID returns the ID of the component that will be created.
	ID ID

	TelemetrySettings

	// BuildInfo can be used by components for informational purposes
	BuildInfo BuildInfo
}

// Deprecated: [v0.68.0] use processor.Factory.
type ProcessorFactory interface {
	Factory

	// CreateTracesProcessor creates a TracesProcessor based on this config.
	// If the processor type does not support tracing or if the config is not valid,
	// an error will be returned instead.
	CreateTracesProcessor(ctx context.Context, set ProcessorCreateSettings, cfg Config, nextConsumer consumer.Traces) (TracesProcessor, error)

	// TracesProcessorStability gets the stability level of the TracesProcessor.
	TracesProcessorStability() StabilityLevel

	// CreateMetricsProcessor creates a MetricsProcessor based on this config.
	// If the processor type does not support metrics or if the config is not valid,
	// an error will be returned instead.
	CreateMetricsProcessor(ctx context.Context, set ProcessorCreateSettings, cfg Config, nextConsumer consumer.Metrics) (MetricsProcessor, error)

	// MetricsProcessorStability gets the stability level of the MetricsProcessor.
	MetricsProcessorStability() StabilityLevel

	// CreateLogsProcessor creates a LogsProcessor based on the config.
	// If the processor type does not support logs or if the config is not valid,
	// an error will be returned instead.
	CreateLogsProcessor(ctx context.Context, set ProcessorCreateSettings, cfg Config, nextConsumer consumer.Logs) (LogsProcessor, error)

	// LogsProcessorStability gets the stability level of the LogsProcessor.
	LogsProcessorStability() StabilityLevel
}
