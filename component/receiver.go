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

// Deprecated: [v0.67.0] use receiver.Traces.
type TracesReceiver interface {
	Component
}

// Deprecated: [v0.67.0] use receiver.Metrics.
type MetricsReceiver interface {
	Component
}

// Deprecated: [v0.67.0] use receiver.Logs.
type LogsReceiver interface {
	Component
}

// Deprecated: [v0.67.0] use receiver.CreateSettings.
type ReceiverCreateSettings struct {
	// ID returns the ID of the component that will be created.
	ID ID

	TelemetrySettings

	// BuildInfo can be used by components for informational purposes.
	BuildInfo BuildInfo
}

// Deprecated: [v0.67.0] use receivrer.Factory.
type ReceiverFactory interface {
	Factory

	// CreateTracesReceiver creates a TracesReceiver based on this config.
	// If the receiver type does not support tracing or if the config is not valid
	// an error will be returned instead.
	CreateTracesReceiver(ctx context.Context, set ReceiverCreateSettings, cfg Config, nextConsumer consumer.Traces) (TracesReceiver, error)

	// TracesReceiverStability gets the stability level of the TracesReceiver.
	TracesReceiverStability() StabilityLevel

	// CreateMetricsReceiver creates a MetricsReceiver based on this config.
	// If the receiver type does not support metrics or if the config is not valid
	// an error will be returned instead.
	CreateMetricsReceiver(ctx context.Context, set ReceiverCreateSettings, cfg Config, nextConsumer consumer.Metrics) (MetricsReceiver, error)

	// MetricsReceiverStability gets the stability level of the MetricsReceiver.
	MetricsReceiverStability() StabilityLevel

	// CreateLogsReceiver creates a LogsReceiver based on this config.
	// If the receiver type does not support the data type or if the config is not valid
	// an error will be returned instead.
	CreateLogsReceiver(ctx context.Context, set ReceiverCreateSettings, cfg Config, nextConsumer consumer.Logs) (LogsReceiver, error)

	// LogsReceiverStability gets the stability level of the LogsReceiver.
	LogsReceiverStability() StabilityLevel
}
