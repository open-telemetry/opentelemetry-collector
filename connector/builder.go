// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connector // import "go.opentelemetry.io/collector/connector"

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
)

// Builder is a helper struct that given a set of Configs and Factories helps with creating connectors.
type Builder struct {
	cfgs      map[component.ID]component.Config
	factories map[component.Type]Factory
}

// NewBuilder creates a new connector.Builder to help with creating components form a set of configs and factories.
func NewBuilder(cfgs map[component.ID]component.Config, factories map[component.Type]Factory) *Builder {
	return &Builder{cfgs: cfgs, factories: factories}
}

// CreateTracesToTraces creates a Traces connector based on the settings and config.
func (b *Builder) CreateTracesToTraces(ctx context.Context, set Settings, next consumer.Traces) (Traces, error) {
	if next == nil {
		return nil, errNilNextConsumer
	}
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("connector %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("connector factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.TracesToTracesStability())
	return f.CreateTracesToTraces(ctx, set, cfg, next)
}

// CreateTracesToMetrics creates a Traces connector based on the settings and config.
func (b *Builder) CreateTracesToMetrics(ctx context.Context, set Settings, next consumer.Metrics) (Traces, error) {
	if next == nil {
		return nil, errNilNextConsumer
	}
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("connector %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("connector factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.TracesToMetricsStability())
	return f.CreateTracesToMetrics(ctx, set, cfg, next)
}

// CreateTracesToLogs creates a Traces connector based on the settings and config.
func (b *Builder) CreateTracesToLogs(ctx context.Context, set Settings, next consumer.Logs) (Traces, error) {
	if next == nil {
		return nil, errNilNextConsumer
	}
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("connector %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("connector factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.TracesToLogsStability())
	return f.CreateTracesToLogs(ctx, set, cfg, next)
}

// CreateMetricsToTraces creates a Metrics connector based on the settings and config.
func (b *Builder) CreateMetricsToTraces(ctx context.Context, set Settings, next consumer.Traces) (Metrics, error) {
	if next == nil {
		return nil, errNilNextConsumer
	}
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("connector %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("connector factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.MetricsToTracesStability())
	return f.CreateMetricsToTraces(ctx, set, cfg, next)
}

// CreateMetricsToMetrics creates a Metrics connector based on the settings and config.
func (b *Builder) CreateMetricsToMetrics(ctx context.Context, set Settings, next consumer.Metrics) (Metrics, error) {
	if next == nil {
		return nil, errNilNextConsumer
	}
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("connector %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("connector factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.MetricsToMetricsStability())
	return f.CreateMetricsToMetrics(ctx, set, cfg, next)
}

// CreateMetricsToLogs creates a Metrics connector based on the settings and config.
func (b *Builder) CreateMetricsToLogs(ctx context.Context, set Settings, next consumer.Logs) (Metrics, error) {
	if next == nil {
		return nil, errNilNextConsumer
	}
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("connector %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("connector factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.MetricsToLogsStability())
	return f.CreateMetricsToLogs(ctx, set, cfg, next)
}

// CreateLogsToTraces creates a Logs connector based on the settings and config.
func (b *Builder) CreateLogsToTraces(ctx context.Context, set Settings, next consumer.Traces) (Logs, error) {
	if next == nil {
		return nil, errNilNextConsumer
	}
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("connector %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("connector factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.LogsToTracesStability())
	return f.CreateLogsToTraces(ctx, set, cfg, next)
}

// CreateLogsToMetrics creates a Logs connector based on the settings and config.
func (b *Builder) CreateLogsToMetrics(ctx context.Context, set Settings, next consumer.Metrics) (Logs, error) {
	if next == nil {
		return nil, errNilNextConsumer
	}
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("connector %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("connector factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.LogsToMetricsStability())
	return f.CreateLogsToMetrics(ctx, set, cfg, next)
}

// CreateLogsToLogs creates a Logs connector based on the settings and config.
func (b *Builder) CreateLogsToLogs(ctx context.Context, set Settings, next consumer.Logs) (Logs, error) {
	if next == nil {
		return nil, errNilNextConsumer
	}
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("connector %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("connector factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.LogsToLogsStability())
	return f.CreateLogsToLogs(ctx, set, cfg, next)
}

func (b *Builder) IsConfigured(componentID component.ID) bool {
	_, ok := b.cfgs[componentID]
	return ok
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
