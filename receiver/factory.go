// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receiver // import "go.opentelemetry.io/collector/receiver"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver/internal"
)

// Factory is a factory interface for receivers.
//
// This interface cannot be directly implemented. Implementations must
// use the NewReceiverFactory to implement it.
type Factory interface {
	component.Factory

	// CreateTracesReceiver creates a TracesReceiver based on this config.
	// If the receiver type does not support traces,
	// this function returns the error [pipeline.ErrSignalNotSupported].
	// Implementers can assume `nextConsumer` is never nil.
	CreateTracesReceiver(ctx context.Context, set Settings, cfg component.Config, nextConsumer consumer.Traces) (Traces, error)

	// TracesReceiverStability gets the stability level of the TracesReceiver.
	TracesReceiverStability() component.StabilityLevel

	// CreateMetricsReceiver creates a MetricsReceiver based on this config.
	// If the receiver type does not support metrics,
	// this function returns the error [pipeline.ErrSignalNotSupported].
	// Implementers can assume `nextConsumer` is never nil.
	CreateMetricsReceiver(ctx context.Context, set Settings, cfg component.Config, nextConsumer consumer.Metrics) (Metrics, error)

	// MetricsReceiverStability gets the stability level of the MetricsReceiver.
	MetricsReceiverStability() component.StabilityLevel

	// CreateLogsReceiver creates a LogsReceiver based on this config.
	// If the receiver type does not support logs,
	// this function returns the error [pipeline.ErrSignalNotSupported].
	// Implementers can assume `nextConsumer` is never nil.
	CreateLogsReceiver(ctx context.Context, set Settings, cfg component.Config, nextConsumer consumer.Logs) (Logs, error)

	// LogsReceiverStability gets the stability level of the LogsReceiver.
	LogsReceiverStability() component.StabilityLevel
}

// FactoryOption apply changes to ReceiverOptions.
type FactoryOption = internal.FactoryOption
