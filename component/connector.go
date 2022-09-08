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
	"fmt"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var (
	errDataTypesNotSupported = "connection from %s to %s is not supported"
	ErrTracesToTraces        = fmt.Errorf(errDataTypesNotSupported, DataTypeTraces, DataTypeTraces)
	ErrTracesToMetrics       = fmt.Errorf(errDataTypesNotSupported, DataTypeTraces, DataTypeMetrics)
	ErrTracesToLogs          = fmt.Errorf(errDataTypesNotSupported, DataTypeTraces, DataTypeLogs)
	ErrMetricsToTraces       = fmt.Errorf(errDataTypesNotSupported, DataTypeMetrics, DataTypeTraces)
	ErrMetricsToMetrics      = fmt.Errorf(errDataTypesNotSupported, DataTypeMetrics, DataTypeMetrics)
	ErrMetricsToLogs         = fmt.Errorf(errDataTypesNotSupported, DataTypeMetrics, DataTypeLogs)
	ErrLogsToTraces          = fmt.Errorf(errDataTypesNotSupported, DataTypeLogs, DataTypeTraces)
	ErrLogsToMetrics         = fmt.Errorf(errDataTypesNotSupported, DataTypeLogs, DataTypeMetrics)
	ErrLogsToLogs            = fmt.Errorf(errDataTypesNotSupported, DataTypeLogs, DataTypeLogs)
)

// Connector sends telemetry data from one pipeline to another. A connector
// is both an exporter and receiver, working together to connect pipelines.
// The purpose is to allow for differentiated processing of telemetry data.
//
// Connectors can be used to replicate or route data, merge data streams,
// derive signals from other signals, etc.
type Connector interface {
	Component
}

// A TracesToTracesConnector sends traces from one pipeline to another.
// Its purpose is to allow for differentiated processing of traces.
// TracesToTracesConnector feeds a consumer.Traces with data.
//
// For example traces could be collected in one pipeline and routed to another traces pipeline
// based on criteria such as attributes or other content of the trace. The second pipeline can
// then process and export the trace to the appropriate backend.
type TracesToTracesConnector interface {
	Connector
	ConsumeTracesToTraces(ctx context.Context, td ptrace.Traces) error
}

// A TracesToMetricsConnector acts as an exporter from a traces pipeline and a receiver to a metrics pipeline.
// Its purpose is to derive metrics from a traces pipeline.
// TracesToMetricsConnector feeds a consumer.Metrics with data.
//
// For example traces could be summarized by a metrics connector that emits statistics describing the traces observed.
type TracesToMetricsConnector interface {
	Connector
	ConsumeTracesToMetrics(ctx context.Context, td ptrace.Traces) error
}

// A TracesToLogsConnector acts as an exporter from a traces pipeline and a receiver to a logs pipeline.
// Its purpose is to derive logs from a traces pipeline.
// TracesToLogsConnector feeds a consumer.Logs with data.
//
// For example traces could be analyzed by a logs connector that emits events when particular criteria are met.
type TracesToLogsConnector interface {
	Connector
	ConsumeTracesToLogs(ctx context.Context, td ptrace.Traces) error
}

// A MetricsToTracesConnector acts as an exporter from a metrics pipeline and a receiver to a traces pipeline.
// Its purpose is to derive traces from a metrics pipeline.
// MetricsToTracesConnector feeds a consumer.Traces with data.
//
// For example latency between related data points could be modeled and emitted as traces.
type MetricsToTracesConnector interface {
	Connector
	ConsumeMetricsToTraces(ctx context.Context, tm pmetric.Metrics) error
}

// A MetricsToMetricsConnector sends metrics from one pipeline to another.
// Its purpose is to allow for differentiated processing of metrics.
// MetricsToMetricsConnector feeds a consumer.Metrics with data.
//
// For example metrics could be collected in one pipeline and routed to another metrics pipeline
// based on criteria such as attributes or other content of the metric. The second pipeline can
// then process and export the metric to the appropriate backend.
type MetricsToMetricsConnector interface {
	Connector
	ConsumeMetricsToMetrics(ctx context.Context, tm pmetric.Metrics) error
}

// A MetricsToLogsConnector acts as an exporter from a metrics pipeline and a receiver to a logs pipeline.
// Its purpose is to derive logs from a metrics pipeline.
// MetricsToLogsConnector feeds a consumer.Logs with data.
//
// For example metrics could be analyzed by a logs connector that emits events when particular criteria are met.
type MetricsToLogsConnector interface {
	Connector
	ConsumeMetricsToLogs(ctx context.Context, tm pmetric.Metrics) error
}

// A LogsToTracesConnector acts as an exporter from a logs pipeline and a receiver to a traces pipeline.
// Its purpose is to derive traces from a logs pipeline.
// LogsToTracesConnector feeds a consumer.Traces with data.
//
// For example structured logs containing span information could be consumed and emitted as traces.
type LogsToTracesConnector interface {
	Connector
	ConsumeLogsToTraces(ctx context.Context, tm plog.Logs) error
}

// A LogsToMetricsConnector acts as an exporter from a logs pipeline and a receiver to a metrics pipeline.
// Its purpose is to derive metrics from a logs pipeline.
// LogsToMetricsConnector feeds a consumer.Metrics with data.
//
// For example metrics could be extracted from structured logs that contain numeric data.
type LogsToMetricsConnector interface {
	Connector
	ConsumeLogsToMetrics(ctx context.Context, tm plog.Logs) error
}

// A LogsToLogsConnector sends logs from one pipeline to another.
// Its purpose is to allow for differentiated processing of logs.
// LogsToLogsConnector feeds a consumer.Logs with data.
//
// For example logs could be collected in one pipeline and routed to another logs pipeline
// based on criteria such as attributes or other content of the log. The second pipeline can
// then process and export the log to the appropriate backend.
type LogsToLogsConnector interface {
	Connector
	ConsumeLogsToLogs(ctx context.Context, tm plog.Logs) error
}

// ConnectorCreateSettings configures Connector creators.
type ConnectorCreateSettings struct {
	TelemetrySettings

	// BuildInfo can be used by components for informational purposes.
	BuildInfo BuildInfo
}

// ConnectorFactory is factory interface for connectors.
//
// This interface cannot be directly implemented. Implementations must
// use the NewConnectorFactory to implement it.
type ConnectorFactory interface {
	Factory

	// CreateDefaultConfig creates the default configuration for the Connector.
	// This method can be called multiple times depending on the pipeline
	// configuration and should not cause side-effects that prevent the creation
	// of multiple instances of the Connector.
	// The object returned by this method needs to pass the checks implemented by
	// 'configtest.CheckConfigStruct'. It is recommended to have these checks in the
	// tests of any implementation of the Factory interface.
	CreateDefaultConfig() Config

	CreateTracesToTracesConnector(ctx context.Context, set ConnectorCreateSettings, cfg Config, nextConsumer consumer.Traces) (TracesToTracesConnector, error)
	CreateTracesToMetricsConnector(ctx context.Context, set ConnectorCreateSettings, cfg Config, nextConsumer consumer.Metrics) (TracesToMetricsConnector, error)
	CreateTracesToLogsConnector(ctx context.Context, set ConnectorCreateSettings, cfg Config, nextConsumer consumer.Logs) (TracesToLogsConnector, error)

	CreateMetricsToTracesConnector(ctx context.Context, set ConnectorCreateSettings, cfg Config, nextConsumer consumer.Traces) (MetricsToTracesConnector, error)
	CreateMetricsToMetricsConnector(ctx context.Context, set ConnectorCreateSettings, cfg Config, nextConsumer consumer.Metrics) (MetricsToMetricsConnector, error)
	CreateMetricsToLogsConnector(ctx context.Context, set ConnectorCreateSettings, cfg Config, nextConsumer consumer.Logs) (MetricsToLogsConnector, error)

	CreateLogsToTracesConnector(ctx context.Context, set ConnectorCreateSettings, cfg Config, nextConsumer consumer.Traces) (LogsToTracesConnector, error)
	CreateLogsToMetricsConnector(ctx context.Context, set ConnectorCreateSettings, cfg Config, nextConsumer consumer.Metrics) (LogsToMetricsConnector, error)
	CreateLogsToLogsConnector(ctx context.Context, set ConnectorCreateSettings, cfg Config, nextConsumer consumer.Logs) (LogsToLogsConnector, error)

	TracesToTracesConnectorStability() StabilityLevel
	TracesToMetricsConnectorStability() StabilityLevel
	TracesToLogsConnectorStability() StabilityLevel

	MetricsToTracesConnectorStability() StabilityLevel
	MetricsToMetricsConnectorStability() StabilityLevel
	MetricsToLogsConnectorStability() StabilityLevel

	LogsToTracesConnectorStability() StabilityLevel
	LogsToMetricsConnectorStability() StabilityLevel
	LogsToLogsConnectorStability() StabilityLevel
}

// ConnectorFactoryOption apply changes to ConnectorOptions.
type ConnectorFactoryOption interface {
	// applyConnectorFactoryOption applies the option.
	applyConnectorFactoryOption(o *connectorFactory)
}

var _ ConnectorFactoryOption = (*connectorFactoryOptionFunc)(nil)

// connectorFactoryOptionFunc is an ConnectorFactoryOption created through a function.
type connectorFactoryOptionFunc func(*connectorFactory)

func (f connectorFactoryOptionFunc) applyConnectorFactoryOption(o *connectorFactory) {
	f(o)
}

// ConnectorCreateDefaultConfigFunc is the equivalent of ConnectorFactory.CreateDefaultConfig().
type ConnectorCreateDefaultConfigFunc func() Config

// CreateDefaultConfig implements ConnectorFactory.CreateDefaultConfig().
func (f ConnectorCreateDefaultConfigFunc) CreateDefaultConfig() Config {
	return f()
}

// CreateTracesToTracesConnectorFunc is the equivalent of ConnectorFactory.CreateTracesToTracesConnector().
type CreateTracesToTracesConnectorFunc func(context.Context, ConnectorCreateSettings, Config, consumer.Traces) (TracesToTracesConnector, error)

// CreateTracesToTracesConnector implements ConnectorFactory.CreateTracesToTracesConnector().
func (f CreateTracesToTracesConnectorFunc) CreateTracesToTracesConnector(
	ctx context.Context,
	set ConnectorCreateSettings,
	cfg Config,
	nextConsumer consumer.Traces) (TracesToTracesConnector, error) {
	if f == nil {
		return nil, ErrTracesToTraces
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateTracesToMetricsConnectorFunc is the equivalent of ConnectorFactory.CreateTracesToMetricsConnector().
type CreateTracesToMetricsConnectorFunc func(context.Context, ConnectorCreateSettings, Config, consumer.Metrics) (TracesToMetricsConnector, error)

// CreateTracesToMetricsConnector implements ConnectorFactory.CreateTracesToMetricsConnector().
func (f CreateTracesToMetricsConnectorFunc) CreateTracesToMetricsConnector(
	ctx context.Context,
	set ConnectorCreateSettings,
	cfg Config,
	nextConsumer consumer.Metrics,
) (TracesToMetricsConnector, error) {
	if f == nil {
		return nil, ErrTracesToMetrics
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateTracesToLogsConnectorFunc is the equivalent of ConnectorFactory.CreateTracesToLogsConnector().
type CreateTracesToLogsConnectorFunc func(context.Context, ConnectorCreateSettings, Config, consumer.Logs) (TracesToLogsConnector, error)

// CreateTracesToLogsConnector implements ConnectorFactory.CreateTracesToLogsConnector().
func (f CreateTracesToLogsConnectorFunc) CreateTracesToLogsConnector(
	ctx context.Context,
	set ConnectorCreateSettings,
	cfg Config,
	nextConsumer consumer.Logs,
) (TracesToLogsConnector, error) {
	if f == nil {
		return nil, ErrTracesToLogs
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateMetricsToTracesConnectorFunc is the equivalent of ConnectorFactory.CreateMetricsToTracesConnector().
type CreateMetricsToTracesConnectorFunc func(context.Context, ConnectorCreateSettings, Config, consumer.Traces) (MetricsToTracesConnector, error)

// CreateMetricsToTracesConnector implements ConnectorFactory.CreateMetricsToTracesConnector().
func (f CreateMetricsToTracesConnectorFunc) CreateMetricsToTracesConnector(
	ctx context.Context,
	set ConnectorCreateSettings,
	cfg Config,
	nextConsumer consumer.Traces,
) (MetricsToTracesConnector, error) {
	if f == nil {
		return nil, ErrMetricsToTraces
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateMetricsToMetricsConnectorFunc is the equivalent of ConnectorFactory.CreateMetricsToTracesConnector().
type CreateMetricsToMetricsConnectorFunc func(context.Context, ConnectorCreateSettings, Config, consumer.Metrics) (MetricsToMetricsConnector, error)

// CreateMetricsToTracesConnector implements ConnectorFactory.CreateMetricsToTracesConnector().
func (f CreateMetricsToMetricsConnectorFunc) CreateMetricsToMetricsConnector(
	ctx context.Context,
	set ConnectorCreateSettings,
	cfg Config,
	nextConsumer consumer.Metrics,
) (MetricsToMetricsConnector, error) {
	if f == nil {
		return nil, ErrMetricsToMetrics
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateMetricsToLogsConnectorFunc is the equivalent of ConnectorFactory.CreateMetricsToLogsConnector().
type CreateMetricsToLogsConnectorFunc func(context.Context, ConnectorCreateSettings, Config, consumer.Logs) (MetricsToLogsConnector, error)

// CreateMetricsToLogsConnector implements ConnectorFactory.CreateMetricsToLogsConnector().
func (f CreateMetricsToLogsConnectorFunc) CreateMetricsToLogsConnector(
	ctx context.Context,
	set ConnectorCreateSettings,
	cfg Config,
	nextConsumer consumer.Logs,
) (MetricsToLogsConnector, error) {
	if f == nil {
		return nil, ErrMetricsToLogs
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateLogsToTracesConnectorFunc is the equivalent of ConnectorFactory.CreateLogsToTracesConnector().
type CreateLogsToTracesConnectorFunc func(context.Context, ConnectorCreateSettings, Config, consumer.Traces) (LogsToTracesConnector, error)

// CreateLogsToTracesConnector implements ConnectorFactory.CreateLogsToTracesConnector().
func (f CreateLogsToTracesConnectorFunc) CreateLogsToTracesConnector(
	ctx context.Context,
	set ConnectorCreateSettings,
	cfg Config,
	nextConsumer consumer.Traces,
) (LogsToTracesConnector, error) {
	if f == nil {
		return nil, ErrLogsToTraces
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateLogsToMetricssConnectorFunc is the equivalent of ConnectorFactory.CreateLogsToMetricsConnector().
type CreateLogsToMetricsConnectorFunc func(context.Context, ConnectorCreateSettings, Config, consumer.Metrics) (LogsToMetricsConnector, error)

// CreateLogsToMetricsConnector implements ConnectorFactory.CreateLogsToMetricsConnector().
func (f CreateLogsToMetricsConnectorFunc) CreateLogsToMetricsConnector(
	ctx context.Context,
	set ConnectorCreateSettings,
	cfg Config,
	nextConsumer consumer.Metrics,
) (LogsToMetricsConnector, error) {
	if f == nil {
		return nil, ErrLogsToMetrics
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateLogsToLogsConnectorFunc is the equivalent of ConnectorFactory.CreateLogsToLogsConnector().
type CreateLogsToLogsConnectorFunc func(context.Context, ConnectorCreateSettings, Config, consumer.Logs) (LogsToLogsConnector, error)

// CreateLogsToLogsConnector implements ConnectorFactory.CreateLogsToLogsConnector().
func (f CreateLogsToLogsConnectorFunc) CreateLogsToLogsConnector(
	ctx context.Context,
	set ConnectorCreateSettings,
	cfg Config,
	nextConsumer consumer.Logs,
) (LogsToLogsConnector, error) {
	if f == nil {
		return nil, ErrLogsToLogs
	}
	return f(ctx, set, cfg, nextConsumer)
}

type connectorFactory struct {
	baseFactory
	ConnectorCreateDefaultConfigFunc

	CreateTracesToTracesConnectorFunc
	CreateTracesToMetricsConnectorFunc
	CreateTracesToLogsConnectorFunc

	CreateMetricsToTracesConnectorFunc
	CreateMetricsToMetricsConnectorFunc
	CreateMetricsToLogsConnectorFunc

	CreateLogsToTracesConnectorFunc
	CreateLogsToMetricsConnectorFunc
	CreateLogsToLogsConnectorFunc

	tracesToTracesStabilityLevel  StabilityLevel
	tracesToMetricsStabilityLevel StabilityLevel
	tracesToLogsStabilityLevel    StabilityLevel

	metricsToTracesStabilityLevel  StabilityLevel
	metricsToMetricsStabilityLevel StabilityLevel
	metricsToLogsStabilityLevel    StabilityLevel

	logsToTracesStabilityLevel  StabilityLevel
	logsToMetricsStabilityLevel StabilityLevel
	logsToLogsStabilityLevel    StabilityLevel
}

// WithTracesToTracesConnector overrides the default "error not supported" implementation for WithTracesToTracesConnector and the default "undefined" stability level.
func WithTracesToTracesConnector(createTracesToTracesConnector CreateTracesToTracesConnectorFunc, sl StabilityLevel) ConnectorFactoryOption {
	return connectorFactoryOptionFunc(func(o *connectorFactory) {
		o.tracesToTracesStabilityLevel = sl
		o.CreateTracesToTracesConnectorFunc = createTracesToTracesConnector
	})
}

// WithTracesToMetricsConnector overrides the default "error not supported" implementation for WithTracesToMetricsConnector and the default "undefined" stability level.
func WithTracesToMetricsConnector(createTracesToMetricsConnector CreateTracesToMetricsConnectorFunc, sl StabilityLevel) ConnectorFactoryOption {
	return connectorFactoryOptionFunc(func(o *connectorFactory) {
		o.tracesToMetricsStabilityLevel = sl
		o.CreateTracesToMetricsConnectorFunc = createTracesToMetricsConnector
	})
}

// WithTracesToLogsConnector overrides the default "error not supported" implementation for WithTracesToLogsConnector and the default "undefined" stability level.
func WithTracesToLogsConnector(createTracesToLogsConnector CreateTracesToLogsConnectorFunc, sl StabilityLevel) ConnectorFactoryOption {
	return connectorFactoryOptionFunc(func(o *connectorFactory) {
		o.tracesToLogsStabilityLevel = sl
		o.CreateTracesToLogsConnectorFunc = createTracesToLogsConnector
	})
}

// WithMetricsToTracesConnector overrides the default "error not supported" implementation for WithMetricsToTracesConnector and the default "undefined" stability level.
func WithMetricsToTracesConnector(createMetricsToTracesConnector CreateMetricsToTracesConnectorFunc, sl StabilityLevel) ConnectorFactoryOption {
	return connectorFactoryOptionFunc(func(o *connectorFactory) {
		o.metricsToTracesStabilityLevel = sl
		o.CreateMetricsToTracesConnectorFunc = createMetricsToTracesConnector
	})
}

// WithMetricsToMetricsConnector overrides the default "error not supported" implementation for WithMetricsToMetricsConnector and the default "undefined" stability level.
func WithMetricsToMetricsConnector(createMetricsToMetricsConnector CreateMetricsToMetricsConnectorFunc, sl StabilityLevel) ConnectorFactoryOption {
	return connectorFactoryOptionFunc(func(o *connectorFactory) {
		o.metricsToMetricsStabilityLevel = sl
		o.CreateMetricsToMetricsConnectorFunc = createMetricsToMetricsConnector
	})
}

// WithMetricsToLogsConnector overrides the default "error not supported" implementation for WithMetricsToLogsConnector and the default "undefined" stability level.
func WithMetricsToLogsConnector(createMetricsToLogsConnector CreateMetricsToLogsConnectorFunc, sl StabilityLevel) ConnectorFactoryOption {
	return connectorFactoryOptionFunc(func(o *connectorFactory) {
		o.metricsToLogsStabilityLevel = sl
		o.CreateMetricsToLogsConnectorFunc = createMetricsToLogsConnector
	})
}

// WithLogsToTracesConnector overrides the default "error not supported" implementation for WithLogsToTracesConnector and the default "undefined" stability level.
func WithLogsToTracesConnector(createLogsToTracesConnector CreateLogsToTracesConnectorFunc, sl StabilityLevel) ConnectorFactoryOption {
	return connectorFactoryOptionFunc(func(o *connectorFactory) {
		o.logsToTracesStabilityLevel = sl
		o.CreateLogsToTracesConnectorFunc = createLogsToTracesConnector
	})
}

// WithLogsToMetricsConnector overrides the default "error not supported" implementation for WithLogsToMetricsConnector and the default "undefined" stability level.
func WithLogsToMetricsConnector(createLogsToMetricsConnector CreateLogsToMetricsConnectorFunc, sl StabilityLevel) ConnectorFactoryOption {
	return connectorFactoryOptionFunc(func(o *connectorFactory) {
		o.logsToMetricsStabilityLevel = sl
		o.CreateLogsToMetricsConnectorFunc = createLogsToMetricsConnector
	})
}

// WithLogsToLogsConnector overrides the default "error not supported" implementation for WithLogsToLogsConnector and the default "undefined" stability level.
func WithLogsToLogsConnector(createLogsToLogsConnector CreateLogsToLogsConnectorFunc, sl StabilityLevel) ConnectorFactoryOption {
	return connectorFactoryOptionFunc(func(o *connectorFactory) {
		o.logsToLogsStabilityLevel = sl
		o.CreateLogsToLogsConnectorFunc = createLogsToLogsConnector
	})
}

func (p connectorFactory) TracesToTracesConnectorStability() StabilityLevel {
	return p.tracesToTracesStabilityLevel
}
func (p connectorFactory) TracesToMetricsConnectorStability() StabilityLevel {
	return p.tracesToMetricsStabilityLevel
}
func (p connectorFactory) TracesToLogsConnectorStability() StabilityLevel {
	return p.tracesToLogsStabilityLevel
}

func (p connectorFactory) MetricsToTracesConnectorStability() StabilityLevel {
	return p.metricsToTracesStabilityLevel
}
func (p connectorFactory) MetricsToMetricsConnectorStability() StabilityLevel {
	return p.metricsToMetricsStabilityLevel
}
func (p connectorFactory) MetricsToLogsConnectorStability() StabilityLevel {
	return p.metricsToLogsStabilityLevel
}

func (p connectorFactory) LogsToTracesConnectorStability() StabilityLevel {
	return p.logsToTracesStabilityLevel
}
func (p connectorFactory) LogsToMetricsConnectorStability() StabilityLevel {
	return p.logsToMetricsStabilityLevel
}
func (p connectorFactory) LogsToLogsConnectorStability() StabilityLevel {
	return p.logsToLogsStabilityLevel
}

// NewConnectorFactory returns a ConnectorFactory.
func NewConnectorFactory(cfgType Type, createDefaultConfig ConnectorCreateDefaultConfigFunc, options ...ConnectorFactoryOption) ConnectorFactory {
	f := &connectorFactory{
		baseFactory:                      baseFactory{cfgType: cfgType},
		ConnectorCreateDefaultConfigFunc: createDefaultConfig,
	}
	for _, opt := range options {
		opt.applyConnectorFactoryOption(f)
	}
	return f
}
