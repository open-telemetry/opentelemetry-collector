// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builders // import "go.opentelemetry.io/collector/service/internal/builders"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/connector/xconnector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
)

func errDataTypes(id component.ID, from, to pipeline.Signal) error {
	return fmt.Errorf("connector %q cannot connect from %s to %s: %w", id, from, to, pipeline.ErrSignalNotSupported)
}

// ConnectorBuilder is a helper struct that given a set of Configs and Factories helps with creating connectors.
type ConnectorBuilder struct {
	cfgs      map[component.ID]component.Config
	factories map[component.Type]connector.Factory
}

// NewConnector creates a new ConnectorBuilder to help with creating components form a set of configs and factories.
func NewConnector(cfgs map[component.ID]component.Config, factories map[component.Type]connector.Factory) *ConnectorBuilder {
	return &ConnectorBuilder{cfgs: cfgs, factories: factories}
}

// CreateTracesToTraces creates a Traces connector based on the settings and config.
func (b *ConnectorBuilder) CreateTracesToTraces(ctx context.Context, set connector.Settings, next consumer.Traces) (connector.Traces, error) {
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
func (b *ConnectorBuilder) CreateTracesToMetrics(ctx context.Context, set connector.Settings, next consumer.Metrics) (connector.Traces, error) {
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
func (b *ConnectorBuilder) CreateTracesToLogs(ctx context.Context, set connector.Settings, next consumer.Logs) (connector.Traces, error) {
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

// CreateTracesToProfiles creates a Traces connector based on the settings and config.
func (b *ConnectorBuilder) CreateTracesToProfiles(ctx context.Context, set connector.Settings, next xconsumer.Profiles) (connector.Traces, error) {
	if next == nil {
		return nil, errNilNextConsumer
	}
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("connector %q is not configured", set.ID)
	}

	connFact, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("connector factory not available for: %q", set.ID)
	}

	f, ok := connFact.(xconnector.Factory)
	if !ok {
		return nil, errDataTypes(set.ID, pipeline.SignalTraces, xpipeline.SignalProfiles)
	}

	logStabilityLevel(set.Logger, f.TracesToProfilesStability())
	return f.CreateTracesToProfiles(ctx, set, cfg, next)
}

// CreateMetricsToTraces creates a Metrics connector based on the settings and config.
func (b *ConnectorBuilder) CreateMetricsToTraces(ctx context.Context, set connector.Settings, next consumer.Traces) (connector.Metrics, error) {
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
func (b *ConnectorBuilder) CreateMetricsToMetrics(ctx context.Context, set connector.Settings, next consumer.Metrics) (connector.Metrics, error) {
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
func (b *ConnectorBuilder) CreateMetricsToLogs(ctx context.Context, set connector.Settings, next consumer.Logs) (connector.Metrics, error) {
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

// CreateMetricsToProfiles creates a Metrics connector based on the settings and config.
func (b *ConnectorBuilder) CreateMetricsToProfiles(ctx context.Context, set connector.Settings, next xconsumer.Profiles) (connector.Metrics, error) {
	if next == nil {
		return nil, errNilNextConsumer
	}
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("connector %q is not configured", set.ID)
	}

	connFact, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("connector factory not available for: %q", set.ID)
	}

	f, ok := connFact.(xconnector.Factory)
	if !ok {
		return nil, errDataTypes(set.ID, pipeline.SignalMetrics, xpipeline.SignalProfiles)
	}

	logStabilityLevel(set.Logger, f.MetricsToProfilesStability())
	return f.CreateMetricsToProfiles(ctx, set, cfg, next)
}

// CreateLogsToTraces creates a Logs connector based on the settings and config.
func (b *ConnectorBuilder) CreateLogsToTraces(ctx context.Context, set connector.Settings, next consumer.Traces) (connector.Logs, error) {
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
func (b *ConnectorBuilder) CreateLogsToMetrics(ctx context.Context, set connector.Settings, next consumer.Metrics) (connector.Logs, error) {
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
func (b *ConnectorBuilder) CreateLogsToLogs(ctx context.Context, set connector.Settings, next consumer.Logs) (connector.Logs, error) {
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

// CreateLogsToProfiles creates a Logs connector based on the settings and config.
func (b *ConnectorBuilder) CreateLogsToProfiles(ctx context.Context, set connector.Settings, next xconsumer.Profiles) (connector.Logs, error) {
	if next == nil {
		return nil, errNilNextConsumer
	}
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("connector %q is not configured", set.ID)
	}

	connFact, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("connector factory not available for: %q", set.ID)
	}

	f, ok := connFact.(xconnector.Factory)
	if !ok {
		return nil, errDataTypes(set.ID, pipeline.SignalLogs, xpipeline.SignalProfiles)
	}

	logStabilityLevel(set.Logger, f.LogsToProfilesStability())
	return f.CreateLogsToProfiles(ctx, set, cfg, next)
}

// CreateProfilesToTraces creates a Profiles connector based on the settings and config.
func (b *ConnectorBuilder) CreateProfilesToTraces(ctx context.Context, set connector.Settings, next consumer.Traces) (xconnector.Profiles, error) {
	if next == nil {
		return nil, errNilNextConsumer
	}
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("connector %q is not configured", set.ID)
	}

	connFact, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("connector factory not available for: %q", set.ID)
	}

	f, ok := connFact.(xconnector.Factory)
	if !ok {
		return nil, errDataTypes(set.ID, xpipeline.SignalProfiles, pipeline.SignalTraces)
	}

	logStabilityLevel(set.Logger, f.ProfilesToTracesStability())
	return f.CreateProfilesToTraces(ctx, set, cfg, next)
}

// CreateProfilesToMetrics creates a Profiles connector based on the settings and config.
func (b *ConnectorBuilder) CreateProfilesToMetrics(ctx context.Context, set connector.Settings, next consumer.Metrics) (xconnector.Profiles, error) {
	if next == nil {
		return nil, errNilNextConsumer
	}
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("connector %q is not configured", set.ID)
	}

	connFact, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("connector factory not available for: %q", set.ID)
	}

	f, ok := connFact.(xconnector.Factory)
	if !ok {
		return nil, errDataTypes(set.ID, xpipeline.SignalProfiles, pipeline.SignalMetrics)
	}

	logStabilityLevel(set.Logger, f.ProfilesToMetricsStability())
	return f.CreateProfilesToMetrics(ctx, set, cfg, next)
}

// CreateProfilesToLogs creates a Profiles connector based on the settings and config.
func (b *ConnectorBuilder) CreateProfilesToLogs(ctx context.Context, set connector.Settings, next consumer.Logs) (xconnector.Profiles, error) {
	if next == nil {
		return nil, errNilNextConsumer
	}
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("connector %q is not configured", set.ID)
	}

	connFact, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("connector factory not available for: %q", set.ID)
	}

	f, ok := connFact.(xconnector.Factory)
	if !ok {
		return nil, errDataTypes(set.ID, xpipeline.SignalProfiles, pipeline.SignalLogs)
	}

	logStabilityLevel(set.Logger, f.ProfilesToLogsStability())
	return f.CreateProfilesToLogs(ctx, set, cfg, next)
}

// CreateProfilesToProfiles creates a Profiles connector based on the settings and config.
func (b *ConnectorBuilder) CreateProfilesToProfiles(ctx context.Context, set connector.Settings, next xconsumer.Profiles) (xconnector.Profiles, error) {
	if next == nil {
		return nil, errNilNextConsumer
	}
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("connector %q is not configured", set.ID)
	}

	connFact, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("connector factory not available for: %q", set.ID)
	}

	f, ok := connFact.(xconnector.Factory)
	if !ok {
		return nil, errDataTypes(set.ID, xpipeline.SignalProfiles, xpipeline.SignalProfiles)
	}

	logStabilityLevel(set.Logger, f.ProfilesToProfilesStability())
	return f.CreateProfilesToProfiles(ctx, set, cfg, next)
}

func (b *ConnectorBuilder) IsConfigured(componentID component.ID) bool {
	_, ok := b.cfgs[componentID]
	return ok
}

func (b *ConnectorBuilder) Factory(componentType component.Type) component.Factory {
	return b.factories[componentType]
}

// NewNopConnectorConfigsAndFactories returns a configuration and factories that allows building a new nop connector.
func NewNopConnectorConfigsAndFactories() (map[component.ID]component.Config, map[component.Type]connector.Factory) {
	nopFactory := connectortest.NewNopFactory()
	// Use a different ID than receivertest and exportertest to avoid ambiguous
	// configuration scenarios. Ambiguous IDs are detected in the 'otelcol' package,
	// but lower level packages such as 'service' assume that IDs are disambiguated.
	connID := component.NewIDWithName(NopType, "conn")

	configs := map[component.ID]component.Config{
		connID: nopFactory.CreateDefaultConfig(),
	}
	factories := map[component.Type]connector.Factory{
		NopType: nopFactory,
	}

	return configs, factories
}
