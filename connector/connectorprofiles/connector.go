// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connectorprofiles // import "go.opentelemetry.io/collector/connector/connectorprofiles"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentprofiles"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/internal"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
	"go.opentelemetry.io/collector/pipeline"
)

type Factory interface {
	connector.Factory

	CreateTracesToProfiles(ctx context.Context, set connector.Settings, cfg component.Config, next consumerprofiles.Profiles) (connector.Traces, error)
	CreateMetricsToProfiles(ctx context.Context, set connector.Settings, cfg component.Config, next consumerprofiles.Profiles) (connector.Metrics, error)
	CreateLogsToProfiles(ctx context.Context, set connector.Settings, cfg component.Config, next consumerprofiles.Profiles) (connector.Logs, error)

	TracesToProfilesStability() component.StabilityLevel
	MetricsToProfilesStability() component.StabilityLevel
	LogsToProfilesStability() component.StabilityLevel

	CreateProfilesToProfiles(ctx context.Context, set connector.Settings, cfg component.Config, next consumerprofiles.Profiles) (Profiles, error)
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
// Profiles feeds a consumer.Traces, consumer.Metrics, consumer.Logs, or consumerprofiles.Profiles with data.
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
	consumerprofiles.Profiles
}

// CreateTracesToProfilesFunc is the equivalent of Factory.CreateTracesToProfiles().
type CreateTracesToProfilesFunc func(context.Context, connector.Settings, component.Config, consumerprofiles.Profiles) (connector.Traces, error)

// CreateTracesToProfiles implements Factory.CreateTracesToProfiles().
func (f CreateTracesToProfilesFunc) CreateTracesToProfiles(ctx context.Context, set connector.Settings, cfg component.Config, next consumerprofiles.Profiles) (connector.Traces, error) {
	if f == nil {
		return nil, internal.ErrDataTypes(set.ID, pipeline.SignalTraces, componentprofiles.SignalProfiles)
	}
	return f(ctx, set, cfg, next)
}

// CreateMetricsToProfilesFunc is the equivalent of Factory.CreateMetricsToProfiles().
type CreateMetricsToProfilesFunc func(context.Context, connector.Settings, component.Config, consumerprofiles.Profiles) (connector.Metrics, error)

// CreateMetricsToProfiles implements Factory.CreateMetricsToProfiles().
func (f CreateMetricsToProfilesFunc) CreateMetricsToProfiles(ctx context.Context, set connector.Settings, cfg component.Config, next consumerprofiles.Profiles) (connector.Metrics, error) {
	if f == nil {
		return nil, internal.ErrDataTypes(set.ID, pipeline.SignalMetrics, componentprofiles.SignalProfiles)
	}
	return f(ctx, set, cfg, next)
}

// CreateLogsToProfilesFunc is the equivalent of Factory.CreateLogsToProfiles().
type CreateLogsToProfilesFunc func(context.Context, connector.Settings, component.Config, consumerprofiles.Profiles) (connector.Logs, error)

// CreateLogsToProfiles implements Factory.CreateLogsToProfiles().
func (f CreateLogsToProfilesFunc) CreateLogsToProfiles(ctx context.Context, set connector.Settings, cfg component.Config, next consumerprofiles.Profiles) (connector.Logs, error) {
	if f == nil {
		return nil, internal.ErrDataTypes(set.ID, pipeline.SignalLogs, componentprofiles.SignalProfiles)
	}
	return f(ctx, set, cfg, next)
}

// CreateProfilesToProfilesFunc is the equivalent of Factory.CreateProfilesToProfiles().
type CreateProfilesToProfilesFunc func(context.Context, connector.Settings, component.Config, consumerprofiles.Profiles) (Profiles, error)

// CreateProfilesToProfiles implements Factory.CreateProfilesToProfiles().
func (f CreateProfilesToProfilesFunc) CreateProfilesToProfiles(ctx context.Context, set connector.Settings, cfg component.Config, next consumerprofiles.Profiles) (Profiles, error) {
	if f == nil {
		return nil, internal.ErrDataTypes(set.ID, componentprofiles.SignalProfiles, componentprofiles.SignalProfiles)
	}
	return f(ctx, set, cfg, next)
}

// CreateProfilesToTracesFunc is the equivalent of Factory.CreateProfilesToTraces().
type CreateProfilesToTracesFunc func(context.Context, connector.Settings, component.Config, consumer.Traces) (Profiles, error)

// CreateProfilesToTraces implements Factory.CreateProfilesToTraces().
func (f CreateProfilesToTracesFunc) CreateProfilesToTraces(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Traces) (Profiles, error) {
	if f == nil {
		return nil, internal.ErrDataTypes(set.ID, componentprofiles.SignalProfiles, pipeline.SignalTraces)
	}
	return f(ctx, set, cfg, next)
}

// CreateProfilesToMetricsFunc is the equivalent of Factory.CreateProfilesToMetrics().
type CreateProfilesToMetricsFunc func(context.Context, connector.Settings, component.Config, consumer.Metrics) (Profiles, error)

// CreateProfilesToMetrics implements Factory.CreateProfilesToMetrics().
func (f CreateProfilesToMetricsFunc) CreateProfilesToMetrics(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Metrics) (Profiles, error) {
	if f == nil {
		return nil, internal.ErrDataTypes(set.ID, componentprofiles.SignalProfiles, pipeline.SignalMetrics)
	}
	return f(ctx, set, cfg, next)
}

// CreateProfilesToLogsFunc is the equivalent of Factory.CreateProfilesToLogs().
type CreateProfilesToLogsFunc func(context.Context, connector.Settings, component.Config, consumer.Logs) (Profiles, error)

// CreateProfilesToLogs implements Factory.CreateProfilesToLogs().
func (f CreateProfilesToLogsFunc) CreateProfilesToLogs(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Logs) (Profiles, error) {
	if f == nil {
		return nil, internal.ErrDataTypes(set.ID, componentprofiles.SignalProfiles, pipeline.SignalLogs)
	}
	return f(ctx, set, cfg, next)
}

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
		o.CreateTracesToProfilesFunc = createTracesToProfiles
	})
}

// WithMetricsToProfiles overrides the default "error not supported" implementation for WithMetricsToProfiles and the default "undefined" stability level.
func WithMetricsToProfiles(createMetricsToProfiles CreateMetricsToProfilesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.metricsToProfilesStabilityLevel = sl
		o.CreateMetricsToProfilesFunc = createMetricsToProfiles
	})
}

// WithLogsToProfiles overrides the default "error not supported" implementation for WithLogsToProfiles and the default "undefined" stability level.
func WithLogsToProfiles(createLogsToProfiles CreateLogsToProfilesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.logsToProfilesStabilityLevel = sl
		o.CreateLogsToProfilesFunc = createLogsToProfiles
	})
}

// WithProfilesToProfiles overrides the default "error not supported" implementation for WithProfilesToProfiles and the default "undefined" stability level.
func WithProfilesToProfiles(createProfilesToProfiles CreateProfilesToProfilesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.profilesToProfilesStabilityLevel = sl
		o.CreateProfilesToProfilesFunc = createProfilesToProfiles
	})
}

// WithProfilesToTraces overrides the default "error not supported" implementation for WithProfilesToTraces and the default "undefined" stability level.
func WithProfilesToTraces(createProfilesToTraces CreateProfilesToTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.profilesToTracesStabilityLevel = sl
		o.CreateProfilesToTracesFunc = createProfilesToTraces
	})
}

// WithProfilesToMetrics overrides the default "error not supported" implementation for WithProfilesToMetrics and the default "undefined" stability level.
func WithProfilesToMetrics(createProfilesToMetrics CreateProfilesToMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.profilesToMetricsStabilityLevel = sl
		o.CreateProfilesToMetricsFunc = createProfilesToMetrics
	})
}

// WithProfilesToLogs overrides the default "error not supported" implementation for WithProfilesToLogs and the default "undefined" stability level.
func WithProfilesToLogs(createProfilesToLogs CreateProfilesToLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.profilesToLogsStabilityLevel = sl
		o.CreateProfilesToLogsFunc = createProfilesToLogs
	})
}

// factory implements the Factory interface.
type factory struct {
	connector.Factory

	CreateTracesToProfilesFunc
	CreateMetricsToProfilesFunc
	CreateLogsToProfilesFunc

	CreateProfilesToProfilesFunc
	CreateProfilesToTracesFunc
	CreateProfilesToMetricsFunc
	CreateProfilesToLogsFunc

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

// NewFactory returns a Factory.
func NewFactory(cfgType component.Type, createDefaultConfig component.CreateDefaultConfigFunc, options ...FactoryOption) Factory {
	opts := factoryOpts{factory: &factory{}}
	for _, opt := range options {
		opt.applyOption(&opts)
	}
	opts.Factory = connector.NewFactory(cfgType, createDefaultConfig, opts.opts...)
	return opts.factory
}
