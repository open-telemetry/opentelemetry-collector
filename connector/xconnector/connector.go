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

	CreateTracesToEntities(ctx context.Context, set connector.Settings, cfg component.Config, next xconsumer.Entities) (connector.Traces, error)
	CreateMetricsToEntities(ctx context.Context, set connector.Settings, cfg component.Config, next xconsumer.Entities) (connector.Metrics, error)
	CreateLogsToEntities(ctx context.Context, set connector.Settings, cfg component.Config, next xconsumer.Entities) (connector.Logs, error)

	TracesToEntitiesStability() component.StabilityLevel
	MetricsToEntitiesStability() component.StabilityLevel
	LogsToEntitiesStability() component.StabilityLevel

	CreateEntitiesToEntities(ctx context.Context, set connector.Settings, cfg component.Config, next xconsumer.Entities) (Entities, error)
	CreateEntitiesToTraces(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Traces) (Entities, error)
	CreateEntitiesToMetrics(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Metrics) (Entities, error)
	CreateEntitiesToLogs(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Logs) (Entities, error)

	EntitiesToEntitiesStability() component.StabilityLevel
	EntitiesToTracesStability() component.StabilityLevel
	EntitiesToMetricsStability() component.StabilityLevel
	EntitiesToLogsStability() component.StabilityLevel
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

// CreateTracesToProfiles implements Factory.CreateTracesToProfiles().
func (f CreateTracesToProfilesFunc) CreateTracesToProfiles(ctx context.Context, set connector.Settings, cfg component.Config, next xconsumer.Profiles) (connector.Traces, error) {
	if f == nil {
		return nil, internal.ErrDataTypes(set.ID, pipeline.SignalTraces, xpipeline.SignalProfiles)
	}
	return f(ctx, set, cfg, next)
}

// CreateMetricsToProfilesFunc is the equivalent of Factory.CreateMetricsToProfiles().
type CreateMetricsToProfilesFunc func(context.Context, connector.Settings, component.Config, xconsumer.Profiles) (connector.Metrics, error)

// CreateMetricsToProfiles implements Factory.CreateMetricsToProfiles().
func (f CreateMetricsToProfilesFunc) CreateMetricsToProfiles(ctx context.Context, set connector.Settings, cfg component.Config, next xconsumer.Profiles) (connector.Metrics, error) {
	if f == nil {
		return nil, internal.ErrDataTypes(set.ID, pipeline.SignalMetrics, xpipeline.SignalProfiles)
	}
	return f(ctx, set, cfg, next)
}

// CreateLogsToProfilesFunc is the equivalent of Factory.CreateLogsToProfiles().
type CreateLogsToProfilesFunc func(context.Context, connector.Settings, component.Config, xconsumer.Profiles) (connector.Logs, error)

// CreateLogsToProfiles implements Factory.CreateLogsToProfiles().
func (f CreateLogsToProfilesFunc) CreateLogsToProfiles(ctx context.Context, set connector.Settings, cfg component.Config, next xconsumer.Profiles) (connector.Logs, error) {
	if f == nil {
		return nil, internal.ErrDataTypes(set.ID, pipeline.SignalLogs, xpipeline.SignalProfiles)
	}
	return f(ctx, set, cfg, next)
}

// CreateProfilesToProfilesFunc is the equivalent of Factory.CreateProfilesToProfiles().
type CreateProfilesToProfilesFunc func(context.Context, connector.Settings, component.Config, xconsumer.Profiles) (Profiles, error)

// CreateProfilesToProfiles implements Factory.CreateProfilesToProfiles().
func (f CreateProfilesToProfilesFunc) CreateProfilesToProfiles(ctx context.Context, set connector.Settings, cfg component.Config, next xconsumer.Profiles) (Profiles, error) {
	if f == nil {
		return nil, internal.ErrDataTypes(set.ID, xpipeline.SignalProfiles, xpipeline.SignalProfiles)
	}
	return f(ctx, set, cfg, next)
}

// CreateProfilesToTracesFunc is the equivalent of Factory.CreateProfilesToTraces().
type CreateProfilesToTracesFunc func(context.Context, connector.Settings, component.Config, consumer.Traces) (Profiles, error)

// CreateProfilesToTraces implements Factory.CreateProfilesToTraces().
func (f CreateProfilesToTracesFunc) CreateProfilesToTraces(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Traces) (Profiles, error) {
	if f == nil {
		return nil, internal.ErrDataTypes(set.ID, xpipeline.SignalProfiles, pipeline.SignalTraces)
	}
	return f(ctx, set, cfg, next)
}

// CreateProfilesToMetricsFunc is the equivalent of Factory.CreateProfilesToMetrics().
type CreateProfilesToMetricsFunc func(context.Context, connector.Settings, component.Config, consumer.Metrics) (Profiles, error)

// CreateProfilesToMetrics implements Factory.CreateProfilesToMetrics().
func (f CreateProfilesToMetricsFunc) CreateProfilesToMetrics(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Metrics) (Profiles, error) {
	if f == nil {
		return nil, internal.ErrDataTypes(set.ID, xpipeline.SignalProfiles, pipeline.SignalMetrics)
	}
	return f(ctx, set, cfg, next)
}

// CreateProfilesToLogsFunc is the equivalent of Factory.CreateProfilesToLogs().
type CreateProfilesToLogsFunc func(context.Context, connector.Settings, component.Config, consumer.Logs) (Profiles, error)

// CreateProfilesToLogs implements Factory.CreateProfilesToLogs().
func (f CreateProfilesToLogsFunc) CreateProfilesToLogs(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Logs) (Profiles, error) {
	if f == nil {
		return nil, internal.ErrDataTypes(set.ID, xpipeline.SignalProfiles, pipeline.SignalLogs)
	}
	return f(ctx, set, cfg, next)
}

// A Entities connector acts as an exporter from a profiles pipeline and a receiver
// to one or more traces, metrics, logs, or profiles pipelines.
// Entities feeds a consumer.Traces, consumer.Metrics, consumer.Logs, or xconsumer.Entities with data.
//
// Examples:
//   - Entities could be collected in one pipeline and routed to another profiles pipeline
//     based on criteria such as attributes or other content of the profile. The second
//     pipeline can then process and export the profile to the appropriate backend.
//   - Entities could be summarized by a metrics connector that emits statistics describing
//     the number of profiles observed.
//   - Entities could be analyzed by a logs connector that emits events when particular
//     criteria are met.
type Entities interface {
	component.Component
	xconsumer.Entities
}

// CreateTracesToEntitiesFunc is the equivalent of Factory.CreateTracesToEntities().
type CreateTracesToEntitiesFunc func(context.Context, connector.Settings, component.Config,
	xconsumer.Entities) (connector.Traces, error)

// CreateTracesToEntities implements Factory.CreateTracesToEntities().
func (f CreateTracesToEntitiesFunc) CreateTracesToEntities(ctx context.Context, set connector.Settings,
	cfg component.Config, next xconsumer.Entities,
) (connector.Traces, error) {
	if f == nil {
		return nil, internal.ErrDataTypes(set.ID, pipeline.SignalTraces, xpipeline.SignalEntities)
	}
	return f(ctx, set, cfg, next)
}

// CreateMetricsToEntitiesFunc is the equivalent of Factory.CreateMetricsToEntities().
type CreateMetricsToEntitiesFunc func(context.Context, connector.Settings, component.Config,
	xconsumer.Entities) (connector.Metrics, error)

// CreateMetricsToEntities implements Factory.CreateMetricsToEntities().
func (f CreateMetricsToEntitiesFunc) CreateMetricsToEntities(ctx context.Context, set connector.Settings,
	cfg component.Config, next xconsumer.Entities,
) (connector.Metrics, error) {
	if f == nil {
		return nil, internal.ErrDataTypes(set.ID, pipeline.SignalMetrics, xpipeline.SignalEntities)
	}
	return f(ctx, set, cfg, next)
}

// CreateLogsToEntitiesFunc is the equivalent of Factory.CreateLogsToEntities().
type CreateLogsToEntitiesFunc func(context.Context, connector.Settings, component.Config,
	xconsumer.Entities) (connector.Logs, error)

// CreateLogsToEntities implements Factory.CreateLogsToEntities().
func (f CreateLogsToEntitiesFunc) CreateLogsToEntities(ctx context.Context, set connector.Settings, cfg component.Config, next xconsumer.Entities) (connector.Logs, error) {
	if f == nil {
		return nil, internal.ErrDataTypes(set.ID, pipeline.SignalLogs, xpipeline.SignalEntities)
	}
	return f(ctx, set, cfg, next)
}

// CreateEntitiesToEntitiesFunc is the equivalent of Factory.CreateEntitiesToEntities().
type CreateEntitiesToEntitiesFunc func(context.Context, connector.Settings, component.Config, xconsumer.Entities) (Entities,
	error)

// CreateEntitiesToEntities implements Factory.CreateEntitiesToEntities().
func (f CreateEntitiesToEntitiesFunc) CreateEntitiesToEntities(ctx context.Context, set connector.Settings, cfg component.Config, next xconsumer.Entities) (Entities, error) {
	if f == nil {
		return nil, internal.ErrDataTypes(set.ID, xpipeline.SignalEntities, xpipeline.SignalEntities)
	}
	return f(ctx, set, cfg, next)
}

// CreateEntitiesToTracesFunc is the equivalent of Factory.CreateEntitiesToTraces().
type CreateEntitiesToTracesFunc func(context.Context, connector.Settings, component.Config, consumer.Traces) (Entities, error)

// CreateEntitiesToTraces implements Factory.CreateEntitiesToTraces().
func (f CreateEntitiesToTracesFunc) CreateEntitiesToTraces(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Traces) (Entities, error) {
	if f == nil {
		return nil, internal.ErrDataTypes(set.ID, xpipeline.SignalEntities, pipeline.SignalTraces)
	}
	return f(ctx, set, cfg, next)
}

// CreateEntitiesToMetricsFunc is the equivalent of Factory.CreateEntitiesToMetrics().
type CreateEntitiesToMetricsFunc func(context.Context, connector.Settings, component.Config, consumer.Metrics) (Entities, error)

// CreateEntitiesToMetrics implements Factory.CreateEntitiesToMetrics().
func (f CreateEntitiesToMetricsFunc) CreateEntitiesToMetrics(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Metrics) (Entities, error) {
	if f == nil {
		return nil, internal.ErrDataTypes(set.ID, xpipeline.SignalEntities, pipeline.SignalMetrics)
	}
	return f(ctx, set, cfg, next)
}

// CreateEntitiesToLogsFunc is the equivalent of Factory.CreateEntitiesToLogs().
type CreateEntitiesToLogsFunc func(context.Context, connector.Settings, component.Config, consumer.Logs) (Entities, error)

// CreateEntitiesToLogs implements Factory.CreateEntitiesToLogs().
func (f CreateEntitiesToLogsFunc) CreateEntitiesToLogs(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Logs) (Entities, error) {
	if f == nil {
		return nil, internal.ErrDataTypes(set.ID, xpipeline.SignalEntities, pipeline.SignalLogs)
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

// WithTracesToEntities overrides the default "error not supported" implementation for WithTracesToEntities and the
// default
// "undefined" stability level.
func WithTracesToEntities(createTracesToEntities CreateTracesToEntitiesFunc,
	sl component.StabilityLevel,
) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.tracesToEntitiesStabilityLevel = sl
		o.CreateTracesToEntitiesFunc = createTracesToEntities
	})
}

// WithMetricsToEntities overrides the default "error not supported" implementation for WithMetricsToEntities and the
// default "undefined" stability level.
func WithMetricsToEntities(createMetricsToEntities CreateMetricsToEntitiesFunc,
	sl component.StabilityLevel,
) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.metricsToEntitiesStabilityLevel = sl
		o.CreateMetricsToEntitiesFunc = createMetricsToEntities
	})
}

// WithLogsToEntities overrides the default "error not supported" implementation for WithLogsToEntities and the default
// "undefined" stability level.
func WithLogsToEntities(createLogsToEntities CreateLogsToEntitiesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.logsToEntitiesStabilityLevel = sl
		o.CreateLogsToEntitiesFunc = createLogsToEntities
	})
}

// WithEntitiesToEntities overrides the default "error not supported" implementation for WithEntitiesToEntities and the
// default "undefined" stability level.
func WithEntitiesToEntities(createEntitiesToEntities CreateEntitiesToEntitiesFunc,
	sl component.StabilityLevel,
) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.entitiesToEntitiesStabilityLevel = sl
		o.CreateEntitiesToEntitiesFunc = createEntitiesToEntities
	})
}

// WithEntitiesToTraces overrides the default "error not supported" implementation for WithEntitiesToTraces and the
// default
// "undefined" stability level.
func WithEntitiesToTraces(createEntitiesToTraces CreateEntitiesToTracesFunc,
	sl component.StabilityLevel,
) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.entitiesToTracesStabilityLevel = sl
		o.CreateEntitiesToTracesFunc = createEntitiesToTraces
	})
}

// WithEntitiesToMetrics overrides the default "error not supported" implementation for WithEntitiesToMetrics and the
// default "undefined" stability level.
func WithEntitiesToMetrics(createEntitiesToMetrics CreateEntitiesToMetricsFunc,
	sl component.StabilityLevel,
) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.entitiesToMetricsStabilityLevel = sl
		o.CreateEntitiesToMetricsFunc = createEntitiesToMetrics
	})
}

// WithEntitiesToLogs overrides the default "error not supported" implementation for WithEntitiesToLogs and the default
// "undefined" stability level.
func WithEntitiesToLogs(createProfilesToLogs CreateProfilesToLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.entitiesToLogsStabilityLevel = sl
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

	CreateTracesToEntitiesFunc
	CreateMetricsToEntitiesFunc
	CreateLogsToEntitiesFunc

	CreateEntitiesToEntitiesFunc
	CreateEntitiesToTracesFunc
	CreateEntitiesToMetricsFunc
	CreateEntitiesToLogsFunc

	tracesToProfilesStabilityLevel  component.StabilityLevel
	metricsToProfilesStabilityLevel component.StabilityLevel
	logsToProfilesStabilityLevel    component.StabilityLevel

	profilesToProfilesStabilityLevel component.StabilityLevel
	profilesToTracesStabilityLevel   component.StabilityLevel
	profilesToMetricsStabilityLevel  component.StabilityLevel
	profilesToLogsStabilityLevel     component.StabilityLevel

	tracesToEntitiesStabilityLevel  component.StabilityLevel
	metricsToEntitiesStabilityLevel component.StabilityLevel
	logsToEntitiesStabilityLevel    component.StabilityLevel

	entitiesToEntitiesStabilityLevel component.StabilityLevel
	entitiesToTracesStabilityLevel   component.StabilityLevel
	entitiesToMetricsStabilityLevel  component.StabilityLevel
	entitiesToLogsStabilityLevel     component.StabilityLevel
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

func (f *factory) TracesToEntitiesStability() component.StabilityLevel {
	return f.tracesToEntitiesStabilityLevel
}

func (f *factory) MetricsToEntitiesStability() component.StabilityLevel {
	return f.metricsToEntitiesStabilityLevel
}

func (f *factory) LogsToEntitiesStability() component.StabilityLevel {
	return f.logsToEntitiesStabilityLevel
}

func (f *factory) EntitiesToEntitiesStability() component.StabilityLevel {
	return f.entitiesToEntitiesStabilityLevel
}

func (f *factory) EntitiesToTracesStability() component.StabilityLevel {
	return f.entitiesToTracesStabilityLevel
}

func (f *factory) EntitiesToMetricsStability() component.StabilityLevel {
	return f.entitiesToMetricsStabilityLevel
}

func (f *factory) EntitiesToLogsStability() component.StabilityLevel {
	return f.entitiesToLogsStabilityLevel
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
