// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receiver // import "go.opentelemetry.io/collector/receiver"

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
)

// Builder receiver is a helper struct that given a set of Configs and Factories helps with creating receivers.
type Builder struct {
	cfgs      map[component.ID]component.Config
	factories map[component.Type]Factory
}

// NewBuilder creates a new receiver.Builder to help with creating components form a set of configs and factories.
func NewBuilder(cfgs map[component.ID]component.Config, factories map[component.Type]Factory) *Builder {
	return &Builder{cfgs: cfgs, factories: factories}
}

// CreateTraces creates a Traces receiver based on the settings and config.
func (b *Builder) CreateTraces(ctx context.Context, set Settings, next consumer.Traces) (Traces, error) {
	if next == nil {
		return nil, errNilNextConsumer
	}
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("receiver %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("receiver factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.TracesReceiverStability())
	return f.CreateTracesReceiver(ctx, set, cfg, next)
}

// CreateMetrics creates a Metrics receiver based on the settings and config.
func (b *Builder) CreateMetrics(ctx context.Context, set Settings, next consumer.Metrics) (Metrics, error) {
	if next == nil {
		return nil, errNilNextConsumer
	}
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("receiver %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("receiver factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.MetricsReceiverStability())
	return f.CreateMetricsReceiver(ctx, set, cfg, next)
}

// CreateLogs creates a Logs receiver based on the settings and config.
func (b *Builder) CreateLogs(ctx context.Context, set Settings, next consumer.Logs) (Logs, error) {
	if next == nil {
		return nil, errNilNextConsumer
	}
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("receiver %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("receiver factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.LogsReceiverStability())
	return f.CreateLogsReceiver(ctx, set, cfg, next)
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
