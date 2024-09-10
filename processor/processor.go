// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processor // import "go.opentelemetry.io/collector/processor"

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/processor/internal"
)

// Traces is a processor that can consume traces.
type Traces = internal.Traces

// Metrics is a processor that can consume metrics.
type Metrics = internal.Metrics

// Logs is a processor that can consume logs.
type Logs = internal.Logs

// Settings is passed to Create* functions in Factory.
type Settings = internal.Settings

// Factory is Factory interface for processors.
//
// This interface cannot be directly implemented. Implementations must
// use the NewProcessorFactory to implement it.
type Factory = internal.Factory

// FactoryOption apply changes to Options.
type FactoryOption = internal.FactoryOption

// CreateTracesFunc is the equivalent of Factory.CreateTraces().
type CreateTracesFunc = internal.CreateTracesFunc

// CreateMetricsFunc is the equivalent of Factory.CreateMetrics().
type CreateMetricsFunc = internal.CreateMetricsFunc

// CreateLogsFunc is the equivalent of Factory.CreateLogs().
type CreateLogsFunc = internal.CreateLogsFunc

// WithTraces overrides the default "error not supported" implementation for CreateTraces and the default "undefined" stability level.
func WithTraces(createTraces CreateTracesFunc, sl component.StabilityLevel) FactoryOption {
	return internal.WithTraces(createTraces, sl)
}

// WithMetrics overrides the default "error not supported" implementation for CreateMetrics and the default "undefined" stability level.
func WithMetrics(createMetrics CreateMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return internal.WithMetrics(createMetrics, sl)
}

// WithLogs overrides the default "error not supported" implementation for CreateLogs and the default "undefined" stability level.
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
			return fMap, fmt.Errorf("duplicate processor factory %q", f.Type())
		}
		fMap[f.Type()] = f
	}
	return fMap, nil
}
