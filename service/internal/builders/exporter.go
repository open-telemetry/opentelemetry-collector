// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builders // import "go.opentelemetry.io/collector/service/internal/builders"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/xexporter"
	"go.opentelemetry.io/collector/pipeline"
)

// ExporterBuilder is a helper struct that given a set of Configs and Factories helps with creating exporters.
type ExporterBuilder struct {
	cfgs      map[component.ID]component.Config
	factories map[component.Type]exporter.Factory
}

// NewExporter creates a new ExporterBuilder to help with creating components form a set of configs and factories.
func NewExporter(cfgs map[component.ID]component.Config, factories map[component.Type]exporter.Factory) *ExporterBuilder {
	return &ExporterBuilder{cfgs: cfgs, factories: factories}
}

// CreateTraces creates a Traces exporter based on the settings and config.
func (b *ExporterBuilder) CreateTraces(ctx context.Context, set exporter.Settings) (exporter.Traces, error) {
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("exporter %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("exporter factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.TracesStability())
	return f.CreateTraces(ctx, set, cfg)
}

// CreateMetrics creates a Metrics exporter based on the settings and config.
func (b *ExporterBuilder) CreateMetrics(ctx context.Context, set exporter.Settings) (exporter.Metrics, error) {
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("exporter %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("exporter factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.MetricsStability())
	return f.CreateMetrics(ctx, set, cfg)
}

// CreateLogs creates a Logs exporter based on the settings and config.
func (b *ExporterBuilder) CreateLogs(ctx context.Context, set exporter.Settings) (exporter.Logs, error) {
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("exporter %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("exporter factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.LogsStability())
	return f.CreateLogs(ctx, set, cfg)
}

// CreateProfiles creates a Profiles exporter based on the settings and config.
func (b *ExporterBuilder) CreateProfiles(ctx context.Context, set exporter.Settings) (xexporter.Profiles, error) {
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("exporter %q is not configured", set.ID)
	}

	expFact, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("exporter factory not available for: %q", set.ID)
	}

	f, ok := expFact.(xexporter.Factory)
	if !ok {
		return nil, pipeline.ErrSignalNotSupported
	}

	logStabilityLevel(set.Logger, f.ProfilesStability())
	return f.CreateProfiles(ctx, set, cfg)
}

func (b *ExporterBuilder) Factory(componentType component.Type) component.Factory {
	return b.factories[componentType]
}

// NewNopExporterConfigsAndFactories returns a configuration and factories that allows building a new nop exporter.
func NewNopExporterConfigsAndFactories() (map[component.ID]component.Config, map[component.Type]exporter.Factory) {
	nopFactory := exportertest.NewNopFactory()
	configs := map[component.ID]component.Config{
		component.NewID(NopType): nopFactory.CreateDefaultConfig(),
	}
	factories := map[component.Type]exporter.Factory{
		NopType: nopFactory,
	}

	return configs, factories
}
