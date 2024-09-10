// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/connector/internal"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentprofiles"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
)

// Factory is a factory interface for connectors.
//
// This interface cannot be directly implemented. Implementations must
// use the NewFactory to implement it.
type Factory interface {
	component.Factory

	// CreateDefaultConfig creates the default configuration for the Connector.
	// This method can be called multiple times depending on the pipeline
	// configuration and should not cause side-effects that prevent the creation
	// of multiple instances of the Connector.
	// The object returned by this method needs to pass the checks implemented by
	// 'configtest.CheckConfigStruct'. It is recommended to have these checks in the
	// tests of any implementation of the Factory interface.
	CreateDefaultConfig() component.Config

	CreateTracesToTraces(ctx context.Context, set Settings, cfg component.Config, nextConsumer consumer.Traces) (Traces, error)
	CreateTracesToMetrics(ctx context.Context, set Settings, cfg component.Config, nextConsumer consumer.Metrics) (Traces, error)
	CreateTracesToLogs(ctx context.Context, set Settings, cfg component.Config, nextConsumer consumer.Logs) (Traces, error)
	CreateTracesToProfiles(ctx context.Context, set Settings, cfg component.Config, nextConsumer consumerprofiles.Profiles) (Traces, error)

	CreateMetricsToTraces(ctx context.Context, set Settings, cfg component.Config, nextConsumer consumer.Traces) (Metrics, error)
	CreateMetricsToMetrics(ctx context.Context, set Settings, cfg component.Config, nextConsumer consumer.Metrics) (Metrics, error)
	CreateMetricsToLogs(ctx context.Context, set Settings, cfg component.Config, nextConsumer consumer.Logs) (Metrics, error)
	CreateMetricsToProfiles(ctx context.Context, set Settings, cfg component.Config, nextConsumer consumerprofiles.Profiles) (Metrics, error)

	CreateLogsToTraces(ctx context.Context, set Settings, cfg component.Config, nextConsumer consumer.Traces) (Logs, error)
	CreateLogsToMetrics(ctx context.Context, set Settings, cfg component.Config, nextConsumer consumer.Metrics) (Logs, error)
	CreateLogsToLogs(ctx context.Context, set Settings, cfg component.Config, nextConsumer consumer.Logs) (Logs, error)
	CreateLogsToProfiles(ctx context.Context, set Settings, cfg component.Config, nextConsumer consumerprofiles.Profiles) (Logs, error)

	CreateProfilesToProfiles(ctx context.Context, set Settings, cfg component.Config, nextConsumer consumerprofiles.Profiles) (Profiles, error)
	CreateProfilesToTraces(ctx context.Context, set Settings, cfg component.Config, nextConsumer consumer.Traces) (Profiles, error)
	CreateProfilesToMetrics(ctx context.Context, set Settings, cfg component.Config, nextConsumer consumer.Metrics) (Profiles, error)
	CreateProfilesToLogs(ctx context.Context, set Settings, cfg component.Config, nextConsumer consumer.Logs) (Profiles, error)

	TracesToTracesStability() component.StabilityLevel
	TracesToMetricsStability() component.StabilityLevel
	TracesToLogsStability() component.StabilityLevel
	TracesToProfilesStability() component.StabilityLevel

	MetricsToTracesStability() component.StabilityLevel
	MetricsToMetricsStability() component.StabilityLevel
	MetricsToLogsStability() component.StabilityLevel
	MetricsToProfilesStability() component.StabilityLevel

	LogsToTracesStability() component.StabilityLevel
	LogsToMetricsStability() component.StabilityLevel
	LogsToLogsStability() component.StabilityLevel
	LogsToProfilesStability() component.StabilityLevel

	ProfilesToProfilesStability() component.StabilityLevel
	ProfilesToTracesStability() component.StabilityLevel
	ProfilesToMetricsStability() component.StabilityLevel
	ProfilesToLogsStability() component.StabilityLevel

	unexportedFactoryFunc()
}

// FactoryOption applies changes to Factory.
type FactoryOption interface {
	// apply applies the option.
	apply(o *factory)
}

var _ FactoryOption = (*factoryOptionFunc)(nil)

// factoryOptionFunc is an FactoryOption created through a function.
type factoryOptionFunc func(*factory)

func (f factoryOptionFunc) apply(o *factory) {
	f(o)
}

// factory implements the Factory interface.
type factory struct {
	cfgType component.Type
	component.CreateDefaultConfigFunc

	CreateTracesToTracesFunc
	CreateTracesToMetricsFunc
	CreateTracesToLogsFunc
	CreateTracesToProfilesFunc

	CreateMetricsToTracesFunc
	CreateMetricsToMetricsFunc
	CreateMetricsToLogsFunc
	CreateMetricsToProfilesFunc

	CreateLogsToTracesFunc
	CreateLogsToMetricsFunc
	CreateLogsToLogsFunc
	CreateLogsToProfilesFunc

	CreateProfilesToProfilesFunc
	CreateProfilesToTracesFunc
	CreateProfilesToMetricsFunc
	CreateProfilesToLogsFunc

	tracesToTracesStabilityLevel   component.StabilityLevel
	tracesToMetricsStabilityLevel  component.StabilityLevel
	tracesToLogsStabilityLevel     component.StabilityLevel
	tracesToProfilesStabilityLevel component.StabilityLevel

	metricsToTracesStabilityLevel   component.StabilityLevel
	metricsToMetricsStabilityLevel  component.StabilityLevel
	metricsToLogsStabilityLevel     component.StabilityLevel
	metricsToProfilesStabilityLevel component.StabilityLevel

	logsToTracesStabilityLevel   component.StabilityLevel
	logsToMetricsStabilityLevel  component.StabilityLevel
	logsToLogsStabilityLevel     component.StabilityLevel
	logsToProfilesStabilityLevel component.StabilityLevel

	profilesToProfilesStabilityLevel component.StabilityLevel
	profilesToTracesStabilityLevel   component.StabilityLevel
	profilesToMetricsStabilityLevel  component.StabilityLevel
	profilesToLogsStabilityLevel     component.StabilityLevel
}

// CreateTracesToTracesFunc is the equivalent of Factory.CreateTracesToTraces().
type CreateTracesToTracesFunc func(context.Context, Settings, component.Config, consumer.Traces) (Traces, error)

// CreateTracesToTraces implements Factory.CreateTracesToTraces().
func (f CreateTracesToTracesFunc) CreateTracesToTraces(
	ctx context.Context,
	set Settings,
	cfg component.Config,
	nextConsumer consumer.Traces) (Traces, error) {
	if f == nil {
		return nil, ErrDataTypes(set.ID, component.DataTypeTraces, component.DataTypeTraces)
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateTracesToMetricsFunc is the equivalent of Factory.CreateTracesToMetrics().
type CreateTracesToMetricsFunc func(context.Context, Settings, component.Config, consumer.Metrics) (Traces, error)

// CreateTracesToMetrics implements Factory.CreateTracesToMetrics().
func (f CreateTracesToMetricsFunc) CreateTracesToMetrics(
	ctx context.Context,
	set Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (Traces, error) {
	if f == nil {
		return nil, ErrDataTypes(set.ID, component.DataTypeTraces, component.DataTypeMetrics)
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateTracesToLogsFunc is the equivalent of Factory.CreateTracesToLogs().
type CreateTracesToLogsFunc func(context.Context, Settings, component.Config, consumer.Logs) (Traces, error)

// CreateTracesToLogs implements Factory.CreateTracesToLogs().
func (f CreateTracesToLogsFunc) CreateTracesToLogs(
	ctx context.Context,
	set Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (Traces, error) {
	if f == nil {
		return nil, ErrDataTypes(set.ID, component.DataTypeTraces, component.DataTypeLogs)
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateTracesToProfilesFunc is the equivalent of Factory.CreateTracesToProfiles().
type CreateTracesToProfilesFunc func(context.Context, Settings, component.Config, consumerprofiles.Profiles) (Traces, error)

// CreateTracesToProfiles implements Factory.CreateTracesToProfiles().
func (f CreateTracesToProfilesFunc) CreateTracesToProfiles(
	ctx context.Context,
	set Settings,
	cfg component.Config,
	nextConsumer consumerprofiles.Profiles,
) (Traces, error) {
	if f == nil {
		return nil, ErrDataTypes(set.ID, component.DataTypeTraces, componentprofiles.DataTypeProfiles)
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateMetricsToTracesFunc is the equivalent of Factory.CreateMetricsToTraces().
type CreateMetricsToTracesFunc func(context.Context, Settings, component.Config, consumer.Traces) (Metrics, error)

// CreateMetricsToTraces implements Factory.CreateMetricsToTraces().
func (f CreateMetricsToTracesFunc) CreateMetricsToTraces(
	ctx context.Context,
	set Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (Metrics, error) {
	if f == nil {
		return nil, ErrDataTypes(set.ID, component.DataTypeMetrics, component.DataTypeTraces)
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateMetricsToMetricsFunc is the equivalent of Factory.CreateMetricsToTraces().
type CreateMetricsToMetricsFunc func(context.Context, Settings, component.Config, consumer.Metrics) (Metrics, error)

// CreateMetricsToMetrics implements Factory.CreateMetricsToTraces().
func (f CreateMetricsToMetricsFunc) CreateMetricsToMetrics(
	ctx context.Context,
	set Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (Metrics, error) {
	if f == nil {
		return nil, ErrDataTypes(set.ID, component.DataTypeMetrics, component.DataTypeMetrics)
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateMetricsToLogsFunc is the equivalent of Factory.CreateMetricsToLogs().
type CreateMetricsToLogsFunc func(context.Context, Settings, component.Config, consumer.Logs) (Metrics, error)

// CreateMetricsToLogs implements Factory.CreateMetricsToLogs().
func (f CreateMetricsToLogsFunc) CreateMetricsToLogs(
	ctx context.Context,
	set Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (Metrics, error) {
	if f == nil {
		return nil, ErrDataTypes(set.ID, component.DataTypeMetrics, component.DataTypeLogs)
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateMetricsToProfilesFunc is the equivalent of Factory.CreateMetricsToProfiles().
type CreateMetricsToProfilesFunc func(context.Context, Settings, component.Config, consumerprofiles.Profiles) (Metrics, error)

// CreateMetricsToProfiles implements Factory.CreateMetricsToProfiles().
func (f CreateMetricsToProfilesFunc) CreateMetricsToProfiles(
	ctx context.Context,
	set Settings,
	cfg component.Config,
	nextConsumer consumerprofiles.Profiles,
) (Metrics, error) {
	if f == nil {
		return nil, ErrDataTypes(set.ID, component.DataTypeMetrics, componentprofiles.DataTypeProfiles)
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateLogsToTracesFunc is the equivalent of Factory.CreateLogsToTraces().
type CreateLogsToTracesFunc func(context.Context, Settings, component.Config, consumer.Traces) (Logs, error)

// CreateLogsToTraces implements Factory.CreateLogsToTraces().
func (f CreateLogsToTracesFunc) CreateLogsToTraces(
	ctx context.Context,
	set Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (Logs, error) {
	if f == nil {
		return nil, ErrDataTypes(set.ID, component.DataTypeLogs, component.DataTypeTraces)
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateLogsToMetricsFunc is the equivalent of Factory.CreateLogsToMetrics().
type CreateLogsToMetricsFunc func(context.Context, Settings, component.Config, consumer.Metrics) (Logs, error)

// CreateLogsToMetrics implements Factory.CreateLogsToMetrics().
func (f CreateLogsToMetricsFunc) CreateLogsToMetrics(
	ctx context.Context,
	set Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (Logs, error) {
	if f == nil {
		return nil, ErrDataTypes(set.ID, component.DataTypeLogs, component.DataTypeMetrics)
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateLogsToLogsFunc is the equivalent of Factory.CreateLogsToLogs().
type CreateLogsToLogsFunc func(context.Context, Settings, component.Config, consumer.Logs) (Logs, error)

// CreateLogsToLogs implements Factory.CreateLogsToLogs().
func (f CreateLogsToLogsFunc) CreateLogsToLogs(
	ctx context.Context,
	set Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (Logs, error) {
	if f == nil {
		return nil, ErrDataTypes(set.ID, component.DataTypeLogs, component.DataTypeLogs)
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateLogsToProfilesFunc is the equivalent of Factory.CreateLogsToProfiles().
type CreateLogsToProfilesFunc func(context.Context, Settings, component.Config, consumerprofiles.Profiles) (Logs, error)

// CreateLogsToProfiles implements Factory.CreateLogsToProfiles().
func (f CreateLogsToProfilesFunc) CreateLogsToProfiles(
	ctx context.Context,
	set Settings,
	cfg component.Config,
	nextConsumer consumerprofiles.Profiles,
) (Logs, error) {
	if f == nil {
		return nil, ErrDataTypes(set.ID, component.DataTypeLogs, componentprofiles.DataTypeProfiles)
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateProfilesToProfilesFunc is the equivalent of Factory.CreateProfilesToProfiles().
type CreateProfilesToProfilesFunc func(context.Context, Settings, component.Config, consumerprofiles.Profiles) (Profiles, error)

// CreateProfilesToProfiles implements Factory.CreateProfilesToProfiles().
func (f CreateProfilesToProfilesFunc) CreateProfilesToProfiles(
	ctx context.Context,
	set Settings,
	cfg component.Config,
	nextConsumer consumerprofiles.Profiles,
) (Profiles, error) {
	if f == nil {
		return nil, ErrDataTypes(set.ID, componentprofiles.DataTypeProfiles, componentprofiles.DataTypeProfiles)
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateProfilesToTracesFunc is the equivalent of Factory.CreateProfilesToTraces().
type CreateProfilesToTracesFunc func(context.Context, Settings, component.Config, consumer.Traces) (Profiles, error)

// CreateProfilesToTraces implements Factory.CreateProfilesToTraces().
func (f CreateProfilesToTracesFunc) CreateProfilesToTraces(
	ctx context.Context,
	set Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (Profiles, error) {
	if f == nil {
		return nil, ErrDataTypes(set.ID, componentprofiles.DataTypeProfiles, component.DataTypeTraces)
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateProfilesToMetricsFunc is the equivalent of Factory.CreateProfilesToMetrics().
type CreateProfilesToMetricsFunc func(context.Context, Settings, component.Config, consumer.Metrics) (Profiles, error)

// CreateProfilesToMetrics implements Factory.CreateProfilesToMetrics().
func (f CreateProfilesToMetricsFunc) CreateProfilesToMetrics(
	ctx context.Context,
	set Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (Profiles, error) {
	if f == nil {
		return nil, ErrDataTypes(set.ID, componentprofiles.DataTypeProfiles, component.DataTypeMetrics)
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateProfilesToLogsFunc is the equivalent of Factory.CreateProfilesToLogs().
type CreateProfilesToLogsFunc func(context.Context, Settings, component.Config, consumer.Logs) (Profiles, error)

// CreateProfilesToLogs implements Factory.CreateProfilesToLogs().
func (f CreateProfilesToLogsFunc) CreateProfilesToLogs(
	ctx context.Context,
	set Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (Profiles, error) {
	if f == nil {
		return nil, ErrDataTypes(set.ID, componentprofiles.DataTypeProfiles, component.DataTypeLogs)
	}
	return f(ctx, set, cfg, nextConsumer)
}

// Type returns the type of component.
func (f factory) Type() component.Type {
	return f.cfgType
}

func (f factory) unexportedFactoryFunc() {}

func (f factory) TracesToTracesStability() component.StabilityLevel {
	return f.tracesToTracesStabilityLevel
}

func (f factory) TracesToMetricsStability() component.StabilityLevel {
	return f.tracesToMetricsStabilityLevel
}

func (f factory) TracesToLogsStability() component.StabilityLevel {
	return f.tracesToLogsStabilityLevel
}

func (f factory) TracesToProfilesStability() component.StabilityLevel {
	return f.tracesToProfilesStabilityLevel
}

func (f factory) MetricsToTracesStability() component.StabilityLevel {
	return f.metricsToTracesStabilityLevel
}

func (f factory) MetricsToMetricsStability() component.StabilityLevel {
	return f.metricsToMetricsStabilityLevel
}

func (f factory) MetricsToLogsStability() component.StabilityLevel {
	return f.metricsToLogsStabilityLevel
}

func (f factory) MetricsToProfilesStability() component.StabilityLevel {
	return f.metricsToProfilesStabilityLevel
}

func (f factory) LogsToTracesStability() component.StabilityLevel {
	return f.logsToTracesStabilityLevel
}

func (f factory) LogsToMetricsStability() component.StabilityLevel {
	return f.logsToMetricsStabilityLevel
}

func (f factory) LogsToLogsStability() component.StabilityLevel {
	return f.logsToLogsStabilityLevel
}

func (f factory) LogsToProfilesStability() component.StabilityLevel {
	return f.logsToProfilesStabilityLevel
}

func (f factory) ProfilesToProfilesStability() component.StabilityLevel {
	return f.profilesToProfilesStabilityLevel
}

func (f factory) ProfilesToTracesStability() component.StabilityLevel {
	return f.profilesToTracesStabilityLevel
}

func (f factory) ProfilesToMetricsStability() component.StabilityLevel {
	return f.profilesToMetricsStabilityLevel
}

func (f factory) ProfilesToLogsStability() component.StabilityLevel {
	return f.profilesToLogsStabilityLevel
}

// NewFactory returns a Factory.
func NewFactory(cfgType component.Type, createDefaultConfig component.CreateDefaultConfigFunc, options ...FactoryOption) Factory {
	f := &factory{
		cfgType:                 cfgType,
		CreateDefaultConfigFunc: createDefaultConfig,
	}
	for _, opt := range options {
		opt.apply(f)
	}
	return f
}

// WithTracesToTraces overrides the default "error not supported" implementation for WithTracesToTraces and the default "undefined" stability level.
func WithTracesToTraces(createTracesToTraces CreateTracesToTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.tracesToTracesStabilityLevel = sl
		o.CreateTracesToTracesFunc = createTracesToTraces
	})
}

// WithTracesToMetrics overrides the default "error not supported" implementation for WithTracesToMetrics and the default "undefined" stability level.
func WithTracesToMetrics(createTracesToMetrics CreateTracesToMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.tracesToMetricsStabilityLevel = sl
		o.CreateTracesToMetricsFunc = createTracesToMetrics
	})
}

// WithTracesToLogs overrides the default "error not supported" implementation for WithTracesToLogs and the default "undefined" stability level.
func WithTracesToLogs(createTracesToLogs CreateTracesToLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.tracesToLogsStabilityLevel = sl
		o.CreateTracesToLogsFunc = createTracesToLogs
	})
}

// WithTracesToProfiles overrides the default "error not supported" implementation for WithTracesToProfiles and the default "undefined" stability level.
func WithTracesToProfiles(createTracesToProfiles CreateTracesToProfilesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.tracesToProfilesStabilityLevel = sl
		o.CreateTracesToProfilesFunc = createTracesToProfiles
	})
}

// WithMetricsToTraces overrides the default "error not supported" implementation for WithMetricsToTraces and the default "undefined" stability level.
func WithMetricsToTraces(createMetricsToTraces CreateMetricsToTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.metricsToTracesStabilityLevel = sl
		o.CreateMetricsToTracesFunc = createMetricsToTraces
	})
}

// WithMetricsToMetrics overrides the default "error not supported" implementation for WithMetricsToMetrics and the default "undefined" stability level.
func WithMetricsToMetrics(createMetricsToMetrics CreateMetricsToMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.metricsToMetricsStabilityLevel = sl
		o.CreateMetricsToMetricsFunc = createMetricsToMetrics
	})
}

// WithMetricsToLogs overrides the default "error not supported" implementation for WithMetricsToLogs and the default "undefined" stability level.
func WithMetricsToLogs(createMetricsToLogs CreateMetricsToLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.metricsToLogsStabilityLevel = sl
		o.CreateMetricsToLogsFunc = createMetricsToLogs
	})
}

// WithMetricsToProfiles overrides the default "error not supported" implementation for WithMetricsToProfiles and the default "undefined" stability level.
func WithMetricsToProfiles(createMetricsToProfiles CreateMetricsToProfilesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.metricsToProfilesStabilityLevel = sl
		o.CreateMetricsToProfilesFunc = createMetricsToProfiles
	})
}

// WithLogsToTraces overrides the default "error not supported" implementation for WithLogsToTraces and the default "undefined" stability level.
func WithLogsToTraces(createLogsToTraces CreateLogsToTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.logsToTracesStabilityLevel = sl
		o.CreateLogsToTracesFunc = createLogsToTraces
	})
}

// WithLogsToMetrics overrides the default "error not supported" implementation for WithLogsToMetrics and the default "undefined" stability level.
func WithLogsToMetrics(createLogsToMetrics CreateLogsToMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.logsToMetricsStabilityLevel = sl
		o.CreateLogsToMetricsFunc = createLogsToMetrics
	})
}

// WithLogsToLogs overrides the default "error not supported" implementation for WithLogsToLogs and the default "undefined" stability level.
func WithLogsToLogs(createLogsToLogs CreateLogsToLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.logsToLogsStabilityLevel = sl
		o.CreateLogsToLogsFunc = createLogsToLogs
	})
}

// WithLogsToProfiles overrides the default "error not supported" implementation for WithLogsToProfiles and the default "undefined" stability level.
func WithLogsToProfiles(createLogsToProfiles CreateLogsToProfilesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.logsToProfilesStabilityLevel = sl
		o.CreateLogsToProfilesFunc = createLogsToProfiles
	})
}

// WithProfilesToProfiles overrides the default "error not supported" implementation for WithProfilesToProfiles and the default "undefined" stability level.
func WithProfilesToProfiles(createProfilesToProfiles CreateProfilesToProfilesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.profilesToProfilesStabilityLevel = sl
		o.CreateProfilesToProfilesFunc = createProfilesToProfiles
	})
}

// WithProfilesToTraces overrides the default "error not supported" implementation for WithProfilesToTraces and the default "undefined" stability level.
func WithProfilesToTraces(createProfilesToTraces CreateProfilesToTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.profilesToTracesStabilityLevel = sl
		o.CreateProfilesToTracesFunc = createProfilesToTraces
	})
}

// WithProfilesToMetrics overrides the default "error not supported" implementation for WithProfilesToMetrics and the default "undefined" stability level.
func WithProfilesToMetrics(createProfilesToMetrics CreateProfilesToMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.profilesToMetricsStabilityLevel = sl
		o.CreateProfilesToMetricsFunc = createProfilesToMetrics
	})
}

// WithProfilesToLogs overrides the default "error not supported" implementation for WithProfilesToLogs and the default "undefined" stability level.
func WithProfilesToLogs(createProfilesToLogs CreateProfilesToLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.profilesToLogsStabilityLevel = sl
		o.CreateProfilesToLogsFunc = createProfilesToLogs
	})
}

func ErrDataTypes(id component.ID, from, to component.DataType) error {
	return fmt.Errorf("connector %q cannot connect from %s to %s: %w", id, from, to, component.ErrDataTypeIsNotSupported)
}
