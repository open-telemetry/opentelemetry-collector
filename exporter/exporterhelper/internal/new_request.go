// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pipeline"
)

type logsExporter struct {
	*BaseExporter
	consumer.Logs
}

// NewLogsRequest creates new logs exporter based on custom LogsConverter and Sender.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewLogsRequest(
	_ context.Context,
	set exporter.Settings,
	converter request.RequestConverterFunc[plog.Logs],
	pusher request.RequestConsumeFunc,
	options ...Option,
) (exporter.Logs, error) {
	if set.Logger == nil {
		return nil, errNilLogger
	}

	if converter == nil {
		return nil, errNilLogsConverter
	}

	if pusher == nil {
		return nil, errNilConsumeRequest
	}

	be, err := NewBaseExporter(set, pipeline.SignalLogs, pusher, options...)
	if err != nil {
		return nil, err
	}

	lc, err := consumer.NewLogs(newConsumeLogs(converter, be, set.Logger), be.ConsumerOptions...)
	if err != nil {
		return nil, err
	}

	return &logsExporter{BaseExporter: be, Logs: lc}, nil
}

func newConsumeLogs(converter request.RequestConverterFunc[plog.Logs], be *BaseExporter, logger *zap.Logger) consumer.ConsumeLogsFunc {
	return func(ctx context.Context, ld plog.Logs) error {
		req, err := converter(ctx, ld)
		if err != nil {
			logger.Error("Failed to convert logs. Dropping data.",
				zap.Int("dropped_log_records", ld.LogRecordCount()),
				zap.Error(err))
			return consumererror.NewPermanent(err)
		}
		return be.Send(ctx, req)
	}
}

type tracesExporter struct {
	*BaseExporter
	consumer.Traces
}

// NewTracesRequest creates a new traces exporter based on a custom TracesConverter and Sender.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewTracesRequest(
	_ context.Context,
	set exporter.Settings,
	converter request.RequestConverterFunc[ptrace.Traces],
	pusher request.RequestConsumeFunc,
	options ...Option,
) (exporter.Traces, error) {
	if set.Logger == nil {
		return nil, errNilLogger
	}

	if converter == nil {
		return nil, errNilTracesConverter
	}

	if pusher == nil {
		return nil, errNilConsumeRequest
	}

	be, err := NewBaseExporter(set, pipeline.SignalTraces, pusher, options...)
	if err != nil {
		return nil, err
	}

	tc, err := consumer.NewTraces(newConsumeTraces(converter, be, set.Logger), be.ConsumerOptions...)
	if err != nil {
		return nil, err
	}

	return &tracesExporter{BaseExporter: be, Traces: tc}, nil
}

func newConsumeTraces(converter request.RequestConverterFunc[ptrace.Traces], be *BaseExporter, logger *zap.Logger) consumer.ConsumeTracesFunc {
	return func(ctx context.Context, td ptrace.Traces) error {
		req, err := converter(ctx, td)
		if err != nil {
			logger.Error("Failed to convert traces. Dropping data.",
				zap.Int("dropped_spans", td.SpanCount()),
				zap.Error(err))
			return consumererror.NewPermanent(err)
		}
		return be.Send(ctx, req)
	}
}

type metricsExporter struct {
	*BaseExporter
	consumer.Metrics
}

// NewMetricsRequest creates a new metrics exporter based on a custom MetricsConverter and Sender.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewMetricsRequest(
	_ context.Context,
	set exporter.Settings,
	converter request.RequestConverterFunc[pmetric.Metrics],
	pusher request.RequestConsumeFunc,
	options ...Option,
) (exporter.Metrics, error) {
	if set.Logger == nil {
		return nil, errNilLogger
	}

	if converter == nil {
		return nil, errNilMetricsConverter
	}

	if pusher == nil {
		return nil, errNilConsumeRequest
	}

	be, err := NewBaseExporter(set, pipeline.SignalMetrics, pusher, options...)
	if err != nil {
		return nil, err
	}

	mc, err := consumer.NewMetrics(newConsumeMetrics(converter, be, set.Logger), be.ConsumerOptions...)
	if err != nil {
		return nil, err
	}

	return &metricsExporter{BaseExporter: be, Metrics: mc}, nil
}

func newConsumeMetrics(converter request.RequestConverterFunc[pmetric.Metrics], be *BaseExporter, logger *zap.Logger) consumer.ConsumeMetricsFunc {
	return func(ctx context.Context, md pmetric.Metrics) error {
		req, err := converter(ctx, md)
		if err != nil {
			logger.Error("Failed to convert metrics. Dropping data.",
				zap.Int("dropped_data_points", md.DataPointCount()),
				zap.Error(err))
			return consumererror.NewPermanent(err)
		}
		return be.Send(ctx, req)
	}
}
