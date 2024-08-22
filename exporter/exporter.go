// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporter // import "go.opentelemetry.io/collector/exporter"

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/internal"
)

// Traces is an exporter that can consume traces.
type Traces = internal.Traces

// Metrics is an exporter that can consume metrics.
type Metrics = internal.Metrics

// Logs is an exporter that can consume logs.
type Logs = internal.Logs

// Settings configures exporter creators.
type Settings = internal.Settings

// Factory is factory interface for exporters.
//
// This interface cannot be directly implemented. Implementations must
// use the NewFactory to implement it.
type Factory = internal.Factory

// FactoryOption apply changes to Factory.
type FactoryOption = internal.FactoryOption

// CreateTracesFunc is the equivalent of Factory.CreateTraces.
type CreateTracesFunc = internal.CreateTracesFunc

// CreateMetricsFunc is the equivalent of Factory.CreateMetrics.
type CreateMetricsFunc = internal.CreateMetricsFunc

// CreateLogsFunc is the equivalent of Factory.CreateLogs.
type CreateLogsFunc = internal.CreateLogsFunc

// WithTraces overrides the default "error not supported" implementation for CreateTracesExporter and the default "undefined" stability level.
func WithTraces(createTraces CreateTracesFunc, sl component.StabilityLevel) FactoryOption {
	return internal.WithTraces(createTraces, sl)
}

// WithMetrics overrides the default "error not supported" implementation for CreateMetricsExporter and the default "undefined" stability level.
func WithMetrics(createMetrics CreateMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return internal.WithMetrics(createMetrics, sl)
}

// WithLogs overrides the default "error not supported" implementation for CreateLogsExporter and the default "undefined" stability level.
func WithLogs(createLogs CreateLogsFunc, sl component.StabilityLevel) FactoryOption {
	return internal.WithLogs(createLogs, sl)
}

// NewFactory returns a Factory.
func NewFactory(cfgType component.Type, createDefaultConfig component.CreateDefaultConfigFunc, options ...FactoryOption) Factory {
	return internal.NewFactory(cfgType, createDefaultConfig, options...)
}

// MakeFactoryMap takes a list of factories and returns a map with Factory type as keys.
// It returns a non-nil error when there are factories with duplicate type.
func MakeFactoryMap(factories ...Factory) (map[component.Type]Factory, error) {
	fMap := map[component.Type]Factory{}
	for _, f := range factories {
		if _, ok := fMap[f.Type()]; ok {
			return fMap, fmt.Errorf("duplicate exporter factory %q", f.Type())
		}
		fMap[f.Type()] = f
	}
	return fMap, nil
}
