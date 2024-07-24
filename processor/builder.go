// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processor // import "go.opentelemetry.io/collector/processor"

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
)

// Builder processor is a helper struct that given a set of Configs and Factories helps with creating processors.
type Builder struct {
	cfgs      map[component.ID]component.Config
	factories map[component.Type]Factory
}

// NewBuilder creates a new processor.Builder to help with creating components form a set of configs and factories.
func NewBuilder(cfgs map[component.ID]component.Config, factories map[component.Type]Factory) *Builder {
	return &Builder{cfgs: cfgs, factories: factories}
}

// CreateTraces creates a Traces processor based on the settings and config.
func (b *Builder) CreateTraces(ctx context.Context, set Settings, next consumer.Traces) (Traces, error) {
	if next == nil {
		return nil, errNilNextConsumer
	}
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("processor %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("processor factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.TracesProcessorStability())
	return f.CreateTracesProcessor(ctx, set, cfg, next)
}

// CreateMetrics creates a Metrics processor based on the settings and config.
func (b *Builder) CreateMetrics(ctx context.Context, set Settings, next consumer.Metrics) (Metrics, error) {
	if next == nil {
		return nil, errNilNextConsumer
	}
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("processor %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("processor factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.MetricsProcessorStability())
	return f.CreateMetricsProcessor(ctx, set, cfg, next)
}

// CreateLogs creates a Logs processor based on the settings and config.
func (b *Builder) CreateLogs(ctx context.Context, set Settings, next consumer.Logs) (Logs, error) {
	if next == nil {
		return nil, errNilNextConsumer
	}
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("processor %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("processor factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.LogsProcessorStability())
	return f.CreateLogsProcessor(ctx, set, cfg, next)
}

func (b *Builder) Factory(componentType component.Type) component.Factory {
	return b.factories[componentType]
}

// logStabilityLevel logs the stability level of a component. The log level is set to info for
// undefined, unmaintained, deprecated and development. The log level is set to debug
// for alpha, beta and stable.
func logStabilityLevel(logger *zap.Logger, sl component.StabilityLevel) {
	if sl >= component.StabilityLevelAlpha {
		logger.Debug(sl.LogMessage())
	} else {
		logger.Info(sl.LogMessage())
	}
}
