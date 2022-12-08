// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package component // import "go.opentelemetry.io/collector/component"

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
)

// Deprecated: [v0.67.0] use Config.
type ReceiverConfig = Config

// Deprecated: [v0.67.0] use UnmarshalConfig.
var UnmarshalReceiverConfig = UnmarshalConfig

// Deprecated: [v0.67.0] use receiver.Traces.
type TracesReceiver interface {
	Component
}

// Deprecated: [v0.67.0] use receiver.Metrics.
type MetricsReceiver interface {
	Component
}

// Deprecated: [v0.67.0] use receiver.Logs.
type LogsReceiver interface {
	Component
}

// Deprecated: [v0.67.0] use receiver.CreateSettings.
type ReceiverCreateSettings struct {
	// ID returns the ID of the component that will be created.
	ID ID

	TelemetrySettings

	// BuildInfo can be used by components for informational purposes.
	BuildInfo BuildInfo
}

// Deprecated: [v0.67.0] use receivrer.Factory.
type ReceiverFactory interface {
	Factory

	// CreateTracesReceiver creates a TracesReceiver based on this config.
	// If the receiver type does not support tracing or if the config is not valid
	// an error will be returned instead.
	CreateTracesReceiver(ctx context.Context, set ReceiverCreateSettings, cfg Config, nextConsumer consumer.Traces) (TracesReceiver, error)

	// TracesReceiverStability gets the stability level of the TracesReceiver.
	TracesReceiverStability() StabilityLevel

	// CreateMetricsReceiver creates a MetricsReceiver based on this config.
	// If the receiver type does not support metrics or if the config is not valid
	// an error will be returned instead.
	CreateMetricsReceiver(ctx context.Context, set ReceiverCreateSettings, cfg Config, nextConsumer consumer.Metrics) (MetricsReceiver, error)

	// MetricsReceiverStability gets the stability level of the MetricsReceiver.
	MetricsReceiverStability() StabilityLevel

	// CreateLogsReceiver creates a LogsReceiver based on this config.
	// If the receiver type does not support the data type or if the config is not valid
	// an error will be returned instead.
	CreateLogsReceiver(ctx context.Context, set ReceiverCreateSettings, cfg Config, nextConsumer consumer.Logs) (LogsReceiver, error)

	// LogsReceiverStability gets the stability level of the LogsReceiver.
	LogsReceiverStability() StabilityLevel
}

// Deprecated: [v0.67.0] use receiver.FactoryOption.
type ReceiverFactoryOption interface {
	// applyReceiverFactoryOption applies the option.
	applyReceiverFactoryOption(o *receiverFactory)
}

var _ ReceiverFactoryOption = (*receiverFactoryOptionFunc)(nil)

// receiverFactoryOptionFunc is an ReceiverFactoryOption created through a function.
type receiverFactoryOptionFunc func(*receiverFactory)

func (f receiverFactoryOptionFunc) applyReceiverFactoryOption(o *receiverFactory) {
	f(o)
}

// Deprecated: [v0.67.0] use CreateDefaultConfigFunc.
type ReceiverCreateDefaultConfigFunc = CreateDefaultConfigFunc

// Deprecated: [v0.67.0] use receiver.CreateTracesFunc.
type CreateTracesReceiverFunc func(context.Context, ReceiverCreateSettings, Config, consumer.Traces) (TracesReceiver, error)

// CreateTracesReceiver implements ReceiverFactory.CreateTracesReceiver().
func (f CreateTracesReceiverFunc) CreateTracesReceiver(
	ctx context.Context,
	set ReceiverCreateSettings,
	cfg Config,
	nextConsumer consumer.Traces) (TracesReceiver, error) {
	if f == nil {
		return nil, ErrDataTypeIsNotSupported
	}
	return f(ctx, set, cfg, nextConsumer)
}

// Deprecated: [v0.67.0] use receiver.CreateMetricsFunc.
type CreateMetricsReceiverFunc func(context.Context, ReceiverCreateSettings, Config, consumer.Metrics) (MetricsReceiver, error)

// CreateMetricsReceiver implements ReceiverFactory.CreateMetricsReceiver().
func (f CreateMetricsReceiverFunc) CreateMetricsReceiver(
	ctx context.Context,
	set ReceiverCreateSettings,
	cfg Config,
	nextConsumer consumer.Metrics,
) (MetricsReceiver, error) {
	if f == nil {
		return nil, ErrDataTypeIsNotSupported
	}
	return f(ctx, set, cfg, nextConsumer)
}

// Deprecated: [v0.67.0] use receiver.CreateLogsFunc.
type CreateLogsReceiverFunc func(context.Context, ReceiverCreateSettings, Config, consumer.Logs) (LogsReceiver, error)

// CreateLogsReceiver implements ReceiverFactory.CreateLogsReceiver().
func (f CreateLogsReceiverFunc) CreateLogsReceiver(
	ctx context.Context,
	set ReceiverCreateSettings,
	cfg Config,
	nextConsumer consumer.Logs,
) (LogsReceiver, error) {
	if f == nil {
		return nil, ErrDataTypeIsNotSupported
	}
	return f(ctx, set, cfg, nextConsumer)
}

type receiverFactory struct {
	baseFactory
	CreateTracesReceiverFunc
	tracesStabilityLevel StabilityLevel
	CreateMetricsReceiverFunc
	metricsStabilityLevel StabilityLevel
	CreateLogsReceiverFunc
	logsStabilityLevel StabilityLevel
}

func (r receiverFactory) TracesReceiverStability() StabilityLevel {
	return r.tracesStabilityLevel
}

func (r receiverFactory) MetricsReceiverStability() StabilityLevel {
	return r.metricsStabilityLevel
}

func (r receiverFactory) LogsReceiverStability() StabilityLevel {
	return r.logsStabilityLevel
}

// Deprecated: [v0.67.0] use receiver.WithTraces.
func WithTracesReceiver(createTracesReceiver CreateTracesReceiverFunc, sl StabilityLevel) ReceiverFactoryOption {
	return receiverFactoryOptionFunc(func(o *receiverFactory) {
		o.tracesStabilityLevel = sl
		o.CreateTracesReceiverFunc = createTracesReceiver
	})
}

// Deprecated: [v0.67.0] use receiver.WithMetrics.
func WithMetricsReceiver(createMetricsReceiver CreateMetricsReceiverFunc, sl StabilityLevel) ReceiverFactoryOption {
	return receiverFactoryOptionFunc(func(o *receiverFactory) {
		o.metricsStabilityLevel = sl
		o.CreateMetricsReceiverFunc = createMetricsReceiver
	})
}

// Deprecated: [v0.67.0] use receiver.WithLogs.
func WithLogsReceiver(createLogsReceiver CreateLogsReceiverFunc, sl StabilityLevel) ReceiverFactoryOption {
	return receiverFactoryOptionFunc(func(o *receiverFactory) {
		o.logsStabilityLevel = sl
		o.CreateLogsReceiverFunc = createLogsReceiver
	})
}

// Deprecated: [v0.67.0] use receiver.NewFactory.
func NewReceiverFactory(cfgType Type, createDefaultConfig CreateDefaultConfigFunc, options ...ReceiverFactoryOption) ReceiverFactory {
	f := &receiverFactory{
		baseFactory: baseFactory{
			cfgType:                 cfgType,
			CreateDefaultConfigFunc: createDefaultConfig,
		},
	}
	for _, opt := range options {
		opt.applyReceiverFactoryOption(f)
	}
	return f
}
