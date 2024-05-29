package internal

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
)

// Traces receiver receives traces.
// Its purpose is to translate data from any format to the collector's internal trace format.
// TracesReceiver feeds a consumer.Traces with data.
//
// For example, it could be Zipkin data source which translates Zipkin spans into ptrace.Traces.
type Traces interface {
	component.Component
}

// Metrics receiver receives metrics.
// Its purpose is to translate data from any format to the collector's internal metrics format.
// MetricsReceiver feeds a consumer.Metrics with data.
//
// For example, it could be Prometheus data source which translates Prometheus metrics into pmetric.Metrics.
type Metrics interface {
	component.Component
}

// Logs receiver receives logs.
// Its purpose is to translate data from any format to the collector's internal logs data format.
// LogsReceiver feeds a consumer.Logs with data.
//
// For example, it could be a receiver that reads syslogs and convert them into plog.Logs.
type Logs interface {
	component.Component
}

// CreateSettings configures Receiver creators.
type CreateSettings struct {
	// ID returns the ID of the component that will be created.
	ID component.ID

	component.TelemetrySettings

	// BuildInfo can be used by components for informational purposes.
	BuildInfo component.BuildInfo
}

// FactoryOption apply changes to ReceiverOptions.
type FactoryOption interface {
	// applyOption applies the option.
	applyOption(o *Factory)
}

// factoryOptionFunc is an ReceiverFactoryOption created through a function.
type factoryOptionFunc func(*Factory)

func (f factoryOptionFunc) applyOption(o *Factory) {
	f(o)
}

// CreateTracesFunc is the equivalent of Factory.CreateTraces.
type CreateTracesFunc func(context.Context, CreateSettings, component.Config, consumer.Traces) (Traces, error)

// CreateTracesReceiver implements Factory.CreateTracesReceiver().
func (f CreateTracesFunc) CreateTracesReceiver(
	ctx context.Context,
	set CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Traces) (Traces, error) {
	if f == nil {
		return nil, component.ErrDataTypeIsNotSupported
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateMetricsFunc is the equivalent of Factory.CreateMetrics.
type CreateMetricsFunc func(context.Context, CreateSettings, component.Config, consumer.Metrics) (Metrics, error)

// CreateMetricsReceiver implements Factory.CreateMetricsReceiver().
func (f CreateMetricsFunc) CreateMetricsReceiver(
	ctx context.Context,
	set CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (Metrics, error) {
	if f == nil {
		return nil, component.ErrDataTypeIsNotSupported
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateLogsFunc is the equivalent of ReceiverFactory.CreateLogsReceiver().
type CreateLogsFunc func(context.Context, CreateSettings, component.Config, consumer.Logs) (Logs, error)

// CreateLogsReceiver implements Factory.CreateLogsReceiver().
func (f CreateLogsFunc) CreateLogsReceiver(
	ctx context.Context,
	set CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (Logs, error) {
	if f == nil {
		return nil, component.ErrDataTypeIsNotSupported
	}
	return f(ctx, set, cfg, nextConsumer)
}

type Internal interface {
	unexportedFactoryFunc()
}

type Factory struct {
	cfgType component.Type
	component.CreateDefaultConfigFunc
	CreateTracesFunc
	tracesStabilityLevel component.StabilityLevel
	CreateMetricsFunc
	metricsStabilityLevel component.StabilityLevel
	CreateLogsFunc
	logsStabilityLevel component.StabilityLevel
}

func (f *Factory) Type() component.Type {
	return f.cfgType
}

func (f *Factory) unexportedFactoryFunc() {}

func (f *Factory) TracesReceiverStability() component.StabilityLevel {
	return f.tracesStabilityLevel
}

func (f *Factory) MetricsReceiverStability() component.StabilityLevel {
	return f.metricsStabilityLevel
}

func (f *Factory) LogsReceiverStability() component.StabilityLevel {
	return f.logsStabilityLevel
}

// WithTraces overrides the default "error not supported" implementation for CreateTracesReceiver and the default "undefined" stability level.
func WithTraces(createTracesReceiver CreateTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *Factory) {
		o.tracesStabilityLevel = sl
		o.CreateTracesFunc = createTracesReceiver
	})
}

// WithMetrics overrides the default "error not supported" implementation for CreateMetricsReceiver and the default "undefined" stability level.
func WithMetrics(createMetricsReceiver CreateMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *Factory) {
		o.metricsStabilityLevel = sl
		o.CreateMetricsFunc = createMetricsReceiver
	})
}

// WithLogs overrides the default "error not supported" implementation for CreateLogsReceiver and the default "undefined" stability level.
func WithLogs(createLogsReceiver CreateLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *Factory) {
		o.logsStabilityLevel = sl
		o.CreateLogsFunc = createLogsReceiver
	})
}

// NewFactory returns a Factory.
func NewFactory(cfgType component.Type, createDefaultConfig component.CreateDefaultConfigFunc, options ...FactoryOption) *Factory {
	f := &Factory{
		cfgType:                 cfgType,
		CreateDefaultConfigFunc: createDefaultConfig,
	}
	for _, opt := range options {
		opt.applyOption(f)
	}
	return f
}
