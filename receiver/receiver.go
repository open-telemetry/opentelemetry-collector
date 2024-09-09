// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receiver // import "go.opentelemetry.io/collector/receiver"

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver/internal"
)

// Traces receiver receives traces.
// Its purpose is to translate data from any format to the collector's internal trace format.
// TracesReceiver feeds a consumer.Traces with data.
//
// For example, it could be Zipkin data source which translates Zipkin spans into ptrace.Traces.
type Traces = internal.Traces

// Metrics receiver receives metrics.
// Its purpose is to translate data from any format to the collector's internal metrics format.
// MetricsReceiver feeds a consumer.Metrics with data.
//
// For example, it could be Prometheus data source which translates Prometheus metrics into pmetric.Metrics.
type Metrics = internal.Metrics

// Logs receiver receives logs.
// Its purpose is to translate data from any format to the collector's internal logs data format.
// LogsReceiver feeds a consumer.Logs with data.
//
// For example, it could be a receiver that reads syslogs and convert them into plog.Logs.
type Logs = internal.Logs

// Settings configures Receiver creators.
type Settings = internal.Settings

// Factory is factory interface for receivers.
//
// This interface cannot be directly implemented. Implementations must
// use the NewReceiverFactory to implement it.
type Factory = internal.Factory

// FactoryOption apply changes to ReceiverOptions.
type FactoryOption = internal.FactoryOption

// CreateTracesFunc is the equivalent of Factory.CreateTraces.
type CreateTracesFunc = internal.CreateTracesFunc

// CreateMetricsFunc is the equivalent of Factory.CreateMetrics.
type CreateMetricsFunc = internal.CreateMetricsFunc

// CreateLogsFunc is the equivalent of ReceiverFactory.CreateLogsReceiver().
type CreateLogsFunc = internal.CreateLogsFunc

// WithTraces overrides the default "error not supported" implementation for CreateTracesReceiver and the default "undefined" stability level.
func WithTraces(createTracesReceiver CreateTracesFunc, sl component.StabilityLevel) FactoryOption {
	return internal.WithTraces(createTracesReceiver, sl)
}

// WithMetrics overrides the default "error not supported" implementation for CreateMetricsReceiver and the default "undefined" stability level.
func WithMetrics(createMetricsReceiver CreateMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return internal.WithMetrics(createMetricsReceiver, sl)
}

// WithLogs overrides the default "error not supported" implementation for CreateLogsReceiver and the default "undefined" stability level.
func WithLogs(createLogsReceiver CreateLogsFunc, sl component.StabilityLevel) FactoryOption {
	return internal.WithLogs(createLogsReceiver, sl)
}

// NewFactory returns a Factory.
func NewFactory(cfgType component.Type, createDefaultConfig component.CreateDefaultConfigFunc, options ...FactoryOption) Factory {
	return internal.NewFactory(cfgType, createDefaultConfig, options...)
}

// MakeFactoryMap takes a list of receiver factories and returns a map with factory type as keys.
// It returns a non-nil error when there are factories with duplicate type.
func MakeFactoryMap(factories ...Factory) (map[component.Type]Factory, error) {
	fMap := map[component.Type]Factory{}
	for _, f := range factories {
		if _, ok := fMap[f.Type()]; ok {
			return fMap, fmt.Errorf("duplicate receiver factory %q", f.Type())
		}
		fMap[f.Type()] = f
	}
	return fMap, nil
}
