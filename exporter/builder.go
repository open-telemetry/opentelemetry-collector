// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporter // import "go.opentelemetry.io/collector/exporter"

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
)

// Builder exporter is a helper struct that given a set of Configs and Factories helps with creating exporters.
type Builder struct {
	cfgs      map[component.ID]component.Config
	factories map[component.Type]Factory
}

// NewBuilder creates a new exporter.Builder to help with creating components form a set of configs and factories.
func NewBuilder(cfgs map[component.ID]component.Config, factories map[component.Type]Factory) *Builder {
	return &Builder{cfgs: cfgs, factories: factories}
}

// CreateTraces creates a Traces exporter based on the settings and config.
func (b *Builder) CreateTraces(ctx context.Context, set Settings) (Traces, error) {
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("exporter %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("exporter factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.TracesExporterStability())
	return f.CreateTracesExporter(ctx, set, cfg)
}

// CreateMetrics creates a Metrics exporter based on the settings and config.
func (b *Builder) CreateMetrics(ctx context.Context, set Settings) (Metrics, error) {
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("exporter %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("exporter factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.MetricsExporterStability())
	return f.CreateMetricsExporter(ctx, set, cfg)
}

// CreateLogs creates a Logs exporter based on the settings and config.
func (b *Builder) CreateLogs(ctx context.Context, set Settings) (Logs, error) {
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("exporter %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("exporter factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.LogsExporterStability())
	return f.CreateLogsExporter(ctx, set, cfg)
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
