// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/receiver/internal"

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
)

// Factory is a factory interface for receivers.
//
// This interface cannot be directly implemented. Implementations must
// use the NewReceiverFactory to implement it.
type Factory interface {
	component.Factory

	// CreateTracesReceiver creates a TracesReceiver based on this config.
	// If the receiver type does not support traces,
	// this function returns the error [component.ErrDataTypeIsNotSupported].
	// Implementers can assume `nextConsumer` is never nil.
	CreateTracesReceiver(ctx context.Context, set Settings, cfg component.Config, nextConsumer consumer.Traces) (Traces, error)

	// TracesReceiverStability gets the stability level of the TracesReceiver.
	TracesReceiverStability() component.StabilityLevel

	// CreateMetricsReceiver creates a MetricsReceiver based on this config.
	// If the receiver type does not support metrics,
	// this function returns the error [component.ErrDataTypeIsNotSupported].
	// Implementers can assume `nextConsumer` is never nil.
	CreateMetricsReceiver(ctx context.Context, set Settings, cfg component.Config, nextConsumer consumer.Metrics) (Metrics, error)

	// MetricsReceiverStability gets the stability level of the MetricsReceiver.
	MetricsReceiverStability() component.StabilityLevel

	// CreateLogsReceiver creates a LogsReceiver based on this config.
	// If the receiver type does not support logs,
	// this function returns the error [component.ErrDataTypeIsNotSupported].
	// Implementers can assume `nextConsumer` is never nil.
	CreateLogsReceiver(ctx context.Context, set Settings, cfg component.Config, nextConsumer consumer.Logs) (Logs, error)

	// LogsReceiverStability gets the stability level of the LogsReceiver.
	LogsReceiverStability() component.StabilityLevel

	// CreateProfilesReceiver creates a ProfilesReceiver based on this config.
	// If the receiver type does not support tracing or if the config is not valid
	// an error will be returned instead. `nextConsumer` is never nil.
	CreateProfilesReceiver(ctx context.Context, set Settings, cfg component.Config, nextConsumer consumerprofiles.Profiles) (Profiles, error)

	// ProfilesReceiverStability gets the stability level of the ProfilesReceiver.
	ProfilesReceiverStability() component.StabilityLevel

	unexportedFactoryFunc()
}

type sharedComponentMap struct {
	sync.RWMutex
	components map[component.Config]component.Component
}

func (scm *sharedComponentMap) set(config component.Config, comp component.Component) {
	scm.RWMutex.Lock()
	defer scm.RWMutex.Unlock()
	scm.components[config] = comp
}

func (scm *sharedComponentMap) get(config component.Config) component.Component {
	scm.RWMutex.RLock()
	defer scm.RWMutex.RUnlock()
	return scm.components[config]
}

// FactoryOption apply changes to ReceiverOptions.
type FactoryOption interface {
	// applyOption applies the option.
	applyOption(o *factory)
}

// factoryOptionFunc is an ReceiverFactoryOption created through a function.
type factoryOptionFunc func(*factory)

func (f factoryOptionFunc) applyOption(o *factory) {
	f(o)
}

// CreateTracesFunc is the equivalent of Factory.CreateTraces.
type CreateTracesFunc func(context.Context, Settings, component.Config, consumer.Traces) (Traces, error)

// SharedTracesFunc sets the trace consumer on the receiver if it is already configured.
// If the function returns nil and no errors, a new receiver will be created.
type SharedTracesFunc func(receiver component.Component, traces consumer.Traces)

// CreateTracesReceiver implements Factory.CreateTracesReceiver().
func (f *factory) CreateTracesReceiver(
	ctx context.Context,
	set Settings,
	cfg component.Config,
	nextConsumer consumer.Traces) (Traces, error) {
	if f == nil || f.CreateTracesFunc == nil {
		return nil, component.ErrDataTypeIsNotSupported
	}
	if f.SharedTracesFunc != nil {
		if r := f.shared.get(cfg); r != nil {
			f.SharedTracesFunc(r, nextConsumer)
			return r, nil
		}
	}
	r, err := f.CreateTracesFunc(ctx, set, cfg, nextConsumer)
	if f.SharedTracesFunc != nil {
		f.shared.set(cfg, r)
	}

	return r, err
}

// CreateMetricsFunc is the equivalent of Factory.CreateMetrics.
type CreateMetricsFunc func(context.Context, Settings, component.Config, consumer.Metrics) (Metrics, error)

// SharedMetricsFunc sets the metrics consumer on the receiver.
type SharedMetricsFunc func(receiver component.Component, metrics consumer.Metrics)

// CreateMetricsReceiver implements Factory.CreateMetricsReceiver().
func (f *factory) CreateMetricsReceiver(
	ctx context.Context,
	set Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (Metrics, error) {
	if f == nil || f.CreateMetricsFunc == nil {
		return nil, component.ErrDataTypeIsNotSupported
	}
	if f.SharedMetricsFunc != nil {
		if r := f.shared.get(cfg); r != nil {
			f.SharedMetricsFunc(r, nextConsumer)
			return r, nil
		}
	}
	r, err := f.CreateMetricsFunc(ctx, set, cfg, nextConsumer)
	if f.SharedMetricsFunc != nil {
		f.shared.set(cfg, r)
	}

	return r, err
}

// CreateLogsFunc is the equivalent of ReceiverFactory.CreateLogsReceiver().
type CreateLogsFunc func(context.Context, Settings, component.Config, consumer.Logs) (Logs, error)

// SharedLogsFunc sets the logs consumer on the receiver.
type SharedLogsFunc func(receiver component.Component, logs consumer.Logs)

// CreateLogsReceiver implements Factory.CreateLogsReceiver().
func (f *factory) CreateLogsReceiver(
	ctx context.Context,
	set Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (Logs, error) {
	if f == nil || f.CreateLogsFunc == nil {
		return nil, component.ErrDataTypeIsNotSupported
	}
	if f.SharedLogsFunc != nil {
		if r := f.shared.get(cfg); r != nil {
			f.SharedLogsFunc(r, nextConsumer)
			return r, nil
		}
	}
	r, err := f.CreateLogsFunc(ctx, set, cfg, nextConsumer)
	if f.SharedLogsFunc != nil {
		f.shared.set(cfg, r)
	}

	return r, err
}

// CreateProfilesFunc is the equivalent of Factory.CreateProfiles.
type CreateProfilesFunc func(context.Context, Settings, component.Config, consumerprofiles.Profiles) (Profiles, error)

// CreateProfilesReceiver implements Factory.CreateProfilesReceiver().
func (f CreateProfilesFunc) CreateProfilesReceiver(
	ctx context.Context,
	set Settings,
	cfg component.Config,
	nextConsumer consumerprofiles.Profiles) (Profiles, error) {
	if f == nil {
		return nil, component.ErrDataTypeIsNotSupported
	}
	return f(ctx, set, cfg, nextConsumer)
}

type factory struct {
	cfgType component.Type
	component.CreateDefaultConfigFunc
	CreateTracesFunc
	tracesStabilityLevel component.StabilityLevel
	CreateMetricsFunc
	metricsStabilityLevel component.StabilityLevel
	CreateLogsFunc
	logsStabilityLevel component.StabilityLevel
	CreateProfilesFunc
	profilesStabilityLevel component.StabilityLevel
	shared                 *sharedComponentMap
	SharedLogsFunc
	SharedMetricsFunc
	SharedTracesFunc
}

func (f *factory) Type() component.Type {
	return f.cfgType
}

func (f *factory) unexportedFactoryFunc() {}

func (f *factory) TracesReceiverStability() component.StabilityLevel {
	return f.tracesStabilityLevel
}

func (f *factory) MetricsReceiverStability() component.StabilityLevel {
	return f.metricsStabilityLevel
}

func (f *factory) LogsReceiverStability() component.StabilityLevel {
	return f.logsStabilityLevel
}

func (f *factory) ProfilesReceiverStability() component.StabilityLevel {
	return f.profilesStabilityLevel
}

// WithTraces overrides the default "error not supported" implementation for CreateTracesReceiver and the default "undefined" stability level.
func WithTraces(createTracesReceiver CreateTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.tracesStabilityLevel = sl
		o.CreateTracesFunc = createTracesReceiver
	})
}

// WithSharedTraces sets the trace consumer on the receiver if it is already configured.
func WithSharedTraces(sharedTracesFunc SharedTracesFunc) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.SharedTracesFunc = sharedTracesFunc
	})
}

// WithMetrics overrides the default "error not supported" implementation for CreateMetricsReceiver and the default "undefined" stability level.
func WithMetrics(createMetricsReceiver CreateMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.metricsStabilityLevel = sl
		o.CreateMetricsFunc = createMetricsReceiver
	})
}

// WithSharedMetrics sets the metrics consumer on the receiver if it is already configured.
func WithSharedMetrics(sharedMetricsFunc SharedMetricsFunc) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.SharedMetricsFunc = sharedMetricsFunc
	})
}

// WithLogs overrides the default "error not supported" implementation for CreateLogsReceiver and the default "undefined" stability level.
func WithLogs(createLogsReceiver CreateLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.logsStabilityLevel = sl
		o.CreateLogsFunc = createLogsReceiver
	})
}

// WithSharedLogs sets the logs consumer on the receiver if it is already configured.
func WithSharedLogs(sharedLogsFunc SharedLogsFunc) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.SharedLogsFunc = sharedLogsFunc
	})
}

// WithProfiles overrides the default "error not supported" implementation for CreateProfilesReceiver and the default "undefined" stability level.
func WithProfiles(createProfilesReceiver CreateProfilesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.profilesStabilityLevel = sl
		o.CreateProfilesFunc = createProfilesReceiver
	})
}

// NewFactory returns a Factory.
func NewFactory(cfgType component.Type, createDefaultConfig component.CreateDefaultConfigFunc, options ...FactoryOption) Factory {
	f := &factory{
		cfgType:                 cfgType,
		CreateDefaultConfigFunc: createDefaultConfig,
		shared: &sharedComponentMap{
			components: map[component.Config]component.Component{},
		},
	}
	for _, opt := range options {
		opt.applyOption(f)
	}
	return f
}
