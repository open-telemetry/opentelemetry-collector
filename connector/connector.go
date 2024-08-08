// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connector // import "go.opentelemetry.io/collector/connector"

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector/internal"
)

// A Traces connector acts as an exporter from a traces pipeline and a receiver
// to one or more traces, metrics, or logs pipelines.
// Traces feeds a consumer.Traces, consumer.Metrics, or consumer.Logs with data.
//
// Examples:
//   - Traces could be collected in one pipeline and routed to another traces pipeline
//     based on criteria such as attributes or other content of the trace. The second
//     pipeline can then process and export the trace to the appropriate backend.
//   - Traces could be summarized by a metrics connector that emits statistics describing
//     the number of traces observed.
//   - Traces could be analyzed by a logs connector that emits events when particular
//     criteria are met.
type Traces = internal.Traces

// A Metrics connector acts as an exporter from a metrics pipeline and a receiver
// to one or more traces, metrics, or logs pipelines.
// Metrics feeds a consumer.Traces, consumer.Metrics, or consumer.Logs with data.
//
// Examples:
//   - Latency between related data points could be modeled and emitted as traces.
//   - Metrics could be collected in one pipeline and routed to another metrics pipeline
//     based on criteria such as attributes or other content of the metric. The second
//     pipeline can then process and export the metric to the appropriate backend.
//   - Metrics could be analyzed by a logs connector that emits events when particular
//     criteria are met.
type Metrics = internal.Metrics

// A Logs connector acts as an exporter from a logs pipeline and a receiver
// to one or more traces, metrics, or logs pipelines.
// Logs feeds a consumer.Traces, consumer.Metrics, or consumer.Logs with data.
//
// Examples:
//   - Structured logs containing span information could be consumed and emitted as traces.
//   - Metrics could be extracted from structured logs that contain numeric data.
//   - Logs could be collected in one pipeline and routed to another logs pipeline
//     based on criteria such as attributes or other content of the log. The second
//     pipeline can then process and export the log to the appropriate backend.
type Logs = internal.Logs

// Settings configures Connector creators.
type Settings = internal.Settings

// Factory is factory interface for connectors.
//
// This interface cannot be directly implemented. Implementations must
// use the NewFactory to implement it.
type Factory = internal.Factory

// FactoryOption applies changes to Factory.
type FactoryOption = internal.FactoryOption

// NewFactory returns a Factory.
func NewFactory(cfgType component.Type, createDefaultConfig component.CreateDefaultConfigFunc, options ...FactoryOption) Factory {
	return internal.NewFactory(cfgType, createDefaultConfig, options...)
}

// CreateTracesToTracesFunc is the equivalent of Factory.CreateTracesToTraces().
type CreateTracesToTracesFunc = internal.CreateTracesToTracesFunc

// CreateTracesToMetricsFunc is the equivalent of Factory.CreateTracesToMetrics().
type CreateTracesToMetricsFunc = internal.CreateTracesToMetricsFunc

// CreateTracesToLogsFunc is the equivalent of Factory.CreateTracesToLogs().
type CreateTracesToLogsFunc = internal.CreateTracesToLogsFunc

// CreateMetricsToTracesFunc is the equivalent of Factory.CreateMetricsToTraces().
type CreateMetricsToTracesFunc = internal.CreateMetricsToTracesFunc

// CreateMetricsToMetricsFunc is the equivalent of Factory.CreateMetricsToTraces().
type CreateMetricsToMetricsFunc = internal.CreateMetricsToMetricsFunc

// CreateMetricsToLogsFunc is the equivalent of Factory.CreateMetricsToLogs().
type CreateMetricsToLogsFunc = internal.CreateMetricsToLogsFunc

// CreateLogsToTracesFunc is the equivalent of Factory.CreateLogsToTraces().
type CreateLogsToTracesFunc = internal.CreateLogsToTracesFunc

// CreateLogsToMetricsFunc is the equivalent of Factory.CreateLogsToMetrics().
type CreateLogsToMetricsFunc = internal.CreateLogsToMetricsFunc

// CreateLogsToLogsFunc is the equivalent of Factory.CreateLogsToLogs().
type CreateLogsToLogsFunc = internal.CreateLogsToLogsFunc

// WithTracesToTraces overrides the default "error not supported" implementation for WithTracesToTraces and the default "undefined" stability level.
func WithTracesToTraces(createTracesToTraces CreateTracesToTracesFunc, sl component.StabilityLevel) FactoryOption {
	return internal.WithTracesToTraces(createTracesToTraces, sl)
}

// WithTracesToMetrics overrides the default "error not supported" implementation for WithTracesToMetrics and the default "undefined" stability level.
func WithTracesToMetrics(createTracesToMetrics CreateTracesToMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return internal.WithTracesToMetrics(createTracesToMetrics, sl)
}

// WithTracesToLogs overrides the default "error not supported" implementation for WithTracesToLogs and the default "undefined" stability level.
func WithTracesToLogs(createTracesToLogs CreateTracesToLogsFunc, sl component.StabilityLevel) FactoryOption {
	return internal.WithTracesToLogs(createTracesToLogs, sl)
}

// WithMetricsToTraces overrides the default "error not supported" implementation for WithMetricsToTraces and the default "undefined" stability level.
func WithMetricsToTraces(createMetricsToTraces CreateMetricsToTracesFunc, sl component.StabilityLevel) FactoryOption {
	return internal.WithMetricsToTraces(createMetricsToTraces, sl)
}

// WithMetricsToMetrics overrides the default "error not supported" implementation for WithMetricsToMetrics and the default "undefined" stability level.
func WithMetricsToMetrics(createMetricsToMetrics CreateMetricsToMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return internal.WithMetricsToMetrics(createMetricsToMetrics, sl)
}

// WithMetricsToLogs overrides the default "error not supported" implementation for WithMetricsToLogs and the default "undefined" stability level.
func WithMetricsToLogs(createMetricsToLogs CreateMetricsToLogsFunc, sl component.StabilityLevel) FactoryOption {
	return internal.WithMetricsToLogs(createMetricsToLogs, sl)
}

// WithLogsToTraces overrides the default "error not supported" implementation for WithLogsToTraces and the default "undefined" stability level.
func WithLogsToTraces(createLogsToTraces CreateLogsToTracesFunc, sl component.StabilityLevel) FactoryOption {
	return internal.WithLogsToTraces(createLogsToTraces, sl)
}

// WithLogsToMetrics overrides the default "error not supported" implementation for WithLogsToMetrics and the default "undefined" stability level.
func WithLogsToMetrics(createLogsToMetrics CreateLogsToMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return internal.WithLogsToMetrics(createLogsToMetrics, sl)
}

// WithLogsToLogs overrides the default "error not supported" implementation for WithLogsToLogs and the default "undefined" stability level.
func WithLogsToLogs(createLogsToLogs CreateLogsToLogsFunc, sl component.StabilityLevel) FactoryOption {
	return internal.WithLogsToLogs(createLogsToLogs, sl)
}

// MakeFactoryMap takes a list of connector factories and returns a map with factory type as keys.
// It returns a non-nil error when there are factories with duplicate type.
func MakeFactoryMap(factories ...Factory) (map[component.Type]Factory, error) {
	fMap := map[component.Type]Factory{}
	for _, f := range factories {
		if _, ok := fMap[f.Type()]; ok {
			return fMap, fmt.Errorf("duplicate connector factory %q", f.Type())
		}
		fMap[f.Type()] = f
	}
	return fMap, nil
}
