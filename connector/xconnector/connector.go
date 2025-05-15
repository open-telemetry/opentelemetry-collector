// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconnector // import "go.opentelemetry.io/collector/connector/xconnector"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/internal"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
)

type Factory interface {
	connector.Factory

	CreateTracesToProfiles(ctx context.Context, set connector.Settings, cfg component.Config, next xconsumer.Profiles) (connector.Traces, error)
	CreateMetricsToProfiles(ctx context.Context, set connector.Settings, cfg component.Config, next xconsumer.Profiles) (connector.Metrics, error)
	CreateLogsToProfiles(ctx context.Context, set connector.Settings, cfg component.Config, next xconsumer.Profiles) (connector.Logs, error)

	TracesToProfilesStability() component.StabilityLevel
	MetricsToProfilesStability() component.StabilityLevel
	LogsToProfilesStability() component.StabilityLevel

	CreateProfilesToProfiles(ctx context.Context, set connector.Settings, cfg component.Config, next xconsumer.Profiles) (Profiles, error)
	CreateProfilesToTraces(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Traces) (Profiles, error)
	CreateProfilesToMetrics(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Metrics) (Profiles, error)
	CreateProfilesToLogs(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Logs) (Profiles, error)

	ProfilesToProfilesStability() component.StabilityLevel
	ProfilesToTracesStability() component.StabilityLevel
	ProfilesToMetricsStability() component.StabilityLevel
	ProfilesToLogsStability() component.StabilityLevel
}

// A Profiles connector acts as an exporter from a profiles pipeline and a receiver
// to one or more traces, metrics, logs, or profiles pipelines.
// Profiles feeds a consumer.Traces, consumer.Metrics, consumer.Logs, or xconsumer.Profiles with data.
//
// Examples:
//   - Profiles could be collected in one pipeline and routed to another profiles pipeline
//     based on criteria such as attributes or other content of the profile. The second
//     pipeline can then process and export the profile to the appropriate backend.
//   - Profiles could be summarized by a metrics connector that emits statistics describing
//     the number of profiles observed.
//   - Profiles could be analyzed by a logs connector that emits events when particular
//     criteria are met.
type Profiles interface {
	component.Component
	xconsumer.Profiles
}

// CreateTracesToProfilesFunc is the equivalent of Factory.CreateTracesToProfiles().
type CreateTracesToProfilesFunc func(context.Context, connector.Settings, component.Config, xconsumer.Profiles) (connector.Traces, error)

// CreateMetricsToProfilesFunc is the equivalent of Factory.CreateMetricsToProfiles().
type CreateMetricsToProfilesFunc func(context.Context, connector.Settings, component.Config, xconsumer.Profiles) (connector.Metrics, error)

// CreateLogsToProfilesFunc is the equivalent of Factory.CreateLogsToProfiles().
type CreateLogsToProfilesFunc func(context.Context, connector.Settings, component.Config, xconsumer.Profiles) (connector.Logs, error)

// CreateProfilesToProfilesFunc is the equivalent of Factory.CreateProfilesToProfiles().
type CreateProfilesToProfilesFunc func(context.Context, connector.Settings, component.Config, xconsumer.Profiles) (Profiles, error)

// CreateProfilesToTracesFunc is the equivalent of Factory.CreateProfilesToTraces().
type CreateProfilesToTracesFunc func(context.Context, connector.Settings, component.Config, consumer.Traces) (Profiles, error)

// CreateProfilesToMetricsFunc is the equivalent of Factory.CreateProfilesToMetrics().
type CreateProfilesToMetricsFunc func(context.Context, connector.Settings, component.Config, consumer.Metrics) (Profiles, error)

// CreateProfilesToLogsFunc is the equivalent of Factory.CreateProfilesToLogs().
type CreateProfilesToLogsFunc func(context.Context, connector.Settings, component.Config, consumer.Logs) (Profiles, error)

// FactoryOption apply changes to ReceiverOptions.
type FactoryOption interface {
	// applyOption applies the option.
	applyOption(o *factoryOpts)
}

// factoryOptionFunc is an ReceiverFactoryOption created through a function.
type factoryOptionFunc func(*factoryOpts)

func (f factoryOptionFunc) applyOption(o *factoryOpts) {
	f(o)
}

type factoryOpts struct {
	opts []connector.FactoryOption

	*factory
}

// WithTracesToTraces overrides the default "error not supported" implementation for WithTracesToTraces and the default "undefined" stability level.
func WithTracesToTraces(createTracesToTraces connector.CreateTracesToTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.opts = append(o.opts, connector.WithTracesToTraces(createTracesToTraces, sl))
	})
}

// WithTracesToMetrics overrides the default "error not supported" implementation for WithTracesToMetrics and the default "undefined" stability level.
func WithTracesToMetrics(createTracesToMetrics connector.CreateTracesToMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.opts = append(o.opts, connector.WithTracesToMetrics(createTracesToMetrics, sl))
	})
}

// WithTracesToLogs overrides the default "error not supported" implementation for WithTracesToLogs and the default "undefined" stability level.
func WithTracesToLogs(createTracesToLogs connector.CreateTracesToLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.opts = append(o.opts, connector.WithTracesToLogs(createTracesToLogs, sl))
	})
}

// WithMetricsToTraces overrides the default "error not supported" implementation for WithMetricsToTraces and the default "undefined" stability level.
func WithMetricsToTraces(createMetricsToTraces connector.CreateMetricsToTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.opts = append(o.opts, connector.WithMetricsToTraces(createMetricsToTraces, sl))
	})
}

// WithMetricsToMetrics overrides the default "error not supported" implementation for WithMetricsToMetrics and the default "undefined" stability level.
func WithMetricsToMetrics(createMetricsToMetrics connector.CreateMetricsToMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.opts = append(o.opts, connector.WithMetricsToMetrics(createMetricsToMetrics, sl))
	})
}

// WithMetricsToLogs overrides the default "error not supported" implementation for WithMetricsToLogs and the default "undefined" stability level.
func WithMetricsToLogs(createMetricsToLogs connector.CreateMetricsToLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.opts = append(o.opts, connector.WithMetricsToLogs(createMetricsToLogs, sl))
	})
}

// WithLogsToTraces overrides the default "error not supported" implementation for WithLogsToTraces and the default "undefined" stability level.
func WithLogsToTraces(createLogsToTraces connector.CreateLogsToTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.opts = append(o.opts, connector.WithLogsToTraces(createLogsToTraces, sl))
	})
}

// WithLogsToMetrics overrides the default "error not supported" implementation for WithLogsToMetrics and the default "undefined" stability level.
func WithLogsToMetrics(createLogsToMetrics connector.CreateLogsToMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.opts = append(o.opts, connector.WithLogsToMetrics(createLogsToMetrics, sl))
	})
}

// WithLogsToLogs overrides the default "error not supported" implementation for WithLogsToLogs and the default "undefined" stability level.
func WithLogsToLogs(createLogsToLogs connector.CreateLogsToLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.opts = append(o.opts, connector.WithLogsToLogs(createLogsToLogs, sl))
	})
}

// WithTracesToProfiles overrides the default "error not supported" implementation for WithTracesToProfiles and the default "undefined" stability level.
func WithTracesToProfiles(createTracesToProfiles CreateTracesToProfilesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.tracesToProfilesStabilityLevel = sl
		o.createTracesToProfilesFunc = createTracesToProfiles
	})
}

// WithMetricsToProfiles overrides the default "error not supported" implementation for WithMetricsToProfiles and the default "undefined" stability level.
func WithMetricsToProfiles(createMetricsToProfiles CreateMetricsToProfilesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.metricsToProfilesStabilityLevel = sl
		o.createMetricsToProfilesFunc = createMetricsToProfiles
	})
}

// WithLogsToProfiles overrides the default "error not supported" implementation for WithLogsToProfiles and the default "undefined" stability level.
func WithLogsToProfiles(createLogsToProfiles CreateLogsToProfilesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.logsToProfilesStabilityLevel = sl
		o.createLogsToProfilesFunc = createLogsToProfiles
	})
}

// WithProfilesToProfiles overrides the default "error not supported" implementation for WithProfilesToProfiles and the default "undefined" stability level.
func WithProfilesToProfiles(createProfilesToProfiles CreateProfilesToProfilesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.profilesToProfilesStabilityLevel = sl
		o.createProfilesToProfilesFunc = createProfilesToProfiles
	})
}

// WithProfilesToTraces overrides the default "error not supported" implementation for WithProfilesToTraces and the default "undefined" stability level.
func WithProfilesToTraces(createProfilesToTraces CreateProfilesToTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.profilesToTracesStabilityLevel = sl
		o.createProfilesToTracesFunc = createProfilesToTraces
	})
}

// WithProfilesToMetrics overrides the default "error not supported" implementation for WithProfilesToMetrics and the default "undefined" stability level.
func WithProfilesToMetrics(createProfilesToMetrics CreateProfilesToMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.profilesToMetricsStabilityLevel = sl
		o.createProfilesToMetricsFunc = createProfilesToMetrics
	})
}

// WithProfilesToLogs overrides the default "error not supported" implementation for WithProfilesToLogs and the default "undefined" stability level.
func WithProfilesToLogs(createProfilesToLogs CreateProfilesToLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.profilesToLogsStabilityLevel = sl
		o.createProfilesToLogsFunc = createProfilesToLogs
	})
}

// factory implements the Factory interface.
type factory struct {
	connector.Factory

	createTracesToProfilesFunc  CreateTracesToProfilesFunc
	createMetricsToProfilesFunc CreateMetricsToProfilesFunc
	createLogsToProfilesFunc    CreateLogsToProfilesFunc

	createProfilesToProfilesFunc CreateProfilesToProfilesFunc
	createProfilesToTracesFunc   CreateProfilesToTracesFunc
	createProfilesToMetricsFunc  CreateProfilesToMetricsFunc
	createProfilesToLogsFunc     CreateProfilesToLogsFunc

	tracesToProfilesStabilityLevel  component.StabilityLevel
	metricsToProfilesStabilityLevel component.StabilityLevel
	logsToProfilesStabilityLevel    component.StabilityLevel

	profilesToProfilesStabilityLevel component.StabilityLevel
	profilesToTracesStabilityLevel   component.StabilityLevel
	profilesToMetricsStabilityLevel  component.StabilityLevel
	profilesToLogsStabilityLevel     component.StabilityLevel
}

func (f *factory) TracesToProfilesStability() component.StabilityLevel {
	return f.tracesToProfilesStabilityLevel
}

func (f *factory) MetricsToProfilesStability() component.StabilityLevel {
	return f.metricsToProfilesStabilityLevel
}

func (f *factory) LogsToProfilesStability() component.StabilityLevel {
	return f.logsToProfilesStabilityLevel
}

func (f *factory) ProfilesToProfilesStability() component.StabilityLevel {
	return f.profilesToProfilesStabilityLevel
}

func (f *factory) ProfilesToTracesStability() component.StabilityLevel {
	return f.profilesToTracesStabilityLevel
}

func (f *factory) ProfilesToMetricsStability() component.StabilityLevel {
	return f.profilesToMetricsStabilityLevel
}

func (f *factory) ProfilesToLogsStability() component.StabilityLevel {
	return f.profilesToLogsStabilityLevel
}

func (f *factory) CreateTracesToProfiles(ctx context.Context, set connector.Settings, cfg component.Config, next xconsumer.Profiles) (connector.Traces, error) {
	if f.createTracesToProfilesFunc == nil {
		return nil, internal.ErrDataTypes(set.ID, pipeline.SignalTraces, xpipeline.SignalProfiles)
	}
	return f.createTracesToProfilesFunc(ctx, set, cfg, next)
}

func (f *factory) CreateMetricsToProfiles(ctx context.Context, set connector.Settings, cfg component.Config, next xconsumer.Profiles) (connector.Metrics, error) {
	if f.createMetricsToProfilesFunc == nil {
		return nil, internal.ErrDataTypes(set.ID, pipeline.SignalMetrics, xpipeline.SignalProfiles)
	}
	return f.createMetricsToProfilesFunc(ctx, set, cfg, next)
}

func (f *factory) CreateLogsToProfiles(ctx context.Context, set connector.Settings, cfg component.Config, next xconsumer.Profiles) (connector.Logs, error) {
	if f.createLogsToProfilesFunc == nil {
		return nil, internal.ErrDataTypes(set.ID, pipeline.SignalLogs, xpipeline.SignalProfiles)
	}
	return f.createLogsToProfilesFunc(ctx, set, cfg, next)
}

func (f *factory) CreateProfilesToProfiles(ctx context.Context, set connector.Settings, cfg component.Config, next xconsumer.Profiles) (Profiles, error) {
	if f.createProfilesToProfilesFunc == nil {
		return nil, internal.ErrDataTypes(set.ID, xpipeline.SignalProfiles, xpipeline.SignalProfiles)
	}
	return f.createProfilesToProfilesFunc(ctx, set, cfg, next)
}

func (f *factory) CreateProfilesToTraces(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Traces) (Profiles, error) {
	if f.createProfilesToTracesFunc == nil {
		return nil, internal.ErrDataTypes(set.ID, xpipeline.SignalProfiles, pipeline.SignalTraces)
	}
	return f.createProfilesToTracesFunc(ctx, set, cfg, next)
}

func (f *factory) CreateProfilesToMetrics(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Metrics) (Profiles, error) {
	if f.createProfilesToMetricsFunc == nil {
		return nil, internal.ErrDataTypes(set.ID, xpipeline.SignalProfiles, pipeline.SignalMetrics)
	}
	return f.createProfilesToMetricsFunc(ctx, set, cfg, next)
}

func (f *factory) CreateProfilesToLogs(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Logs) (Profiles, error) {
	if f.createProfilesToLogsFunc == nil {
		return nil, internal.ErrDataTypes(set.ID, xpipeline.SignalProfiles, pipeline.SignalLogs)
	}
	return f.createProfilesToLogsFunc(ctx, set, cfg, next)
}

// NewFactory returns a Factory.
func NewFactory(cfgType component.Type, createDefaultConfig component.CreateDefaultConfigFunc, options ...FactoryOption) Factory {
	opts := factoryOpts{factory: &factory{}}
	for _, opt := range options {
		opt.applyOption(&opts)
	}
	opts.Factory = connector.NewFactory(cfgType, createDefaultConfig, opts.opts...)
	return opts.factory
}
