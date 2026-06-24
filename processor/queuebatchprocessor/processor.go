// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatchprocessor // import "go.opentelemetry.io/collector/processor/queuebatchprocessor"

import (
	"context"

	noopmetric "go.opentelemetry.io/otel/metric/noop"

	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor"
)

// exporterSettings derives exporter.Settings from processor.Settings while
// reframing the exporterhelper's telemetry as the processor's own. The
// MeterProvider is replaced with a no-op so the exporterhelper emits no
// otelcol_exporter_* metrics (this processor reports its own), and the
// TracerProvider is wrapped so the exporterhelper's spans are reported under
// this processor's scope with processor/-namespaced names and attributes.
func exporterSettings(set processor.Settings) exporter.Settings {
	tel := set.TelemetrySettings
	tel.MeterProvider = noopmetric.NewMeterProvider()
	tel.TracerProvider = newRenamingTracerProvider(tel.TracerProvider)
	return exporter.Settings{
		ID:                set.ID,
		TelemetrySettings: tel,
		BuildInfo:         set.BuildInfo,
	}
}

// queueOptions are the exporterhelper options shared by all signals: the
// queue/batch configuration and a disabled per-export timeout. The batch
// processor never imposed an export timeout, so neither does this processor;
// callers that want one can wrap the next consumer.
func queueOptions(cfg *Config) []exporterhelper.Option {
	return []exporterhelper.Option{
		exporterhelper.WithQueue(configoptional.Some(*cfg)),
		exporterhelper.WithTimeout(exporterhelper.TimeoutConfig{Timeout: 0}),
	}
}

// tracesProcessor wraps the exporterhelper result to count incoming items on
// arrival, before they are queued and batched.
type tracesProcessor struct {
	exporter.Traces
	bt *batchTelemetry
}

func (p *tracesProcessor) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	p.bt.recordIncoming(ctx, int64(td.SpanCount()))
	return p.Traces.ConsumeTraces(ctx, td)
}

func newTracesProcessor(ctx context.Context, set processor.Settings, cfg *Config, next consumer.Traces) (processor.Traces, error) {
	bt, err := newBatchTelemetry(set.TelemetrySettings, set.ID, pipeline.SignalTraces)
	if err != nil {
		return nil, err
	}
	sizer := &ptrace.ProtoMarshaler{}
	pusher := func(ctx context.Context, td ptrace.Traces) error {
		bt.recordOutgoing(ctx, int64(td.SpanCount()), func() int64 { return int64(sizer.TracesSize(td)) })
		return next.ConsumeTraces(ctx, td)
	}
	inner, err := exporterhelper.NewTraces(ctx, exporterSettings(set), cfg, pusher, queueOptions(cfg)...)
	if err != nil {
		return nil, err
	}
	return &tracesProcessor{Traces: inner, bt: bt}, nil
}

// metricsProcessor wraps the exporterhelper result to count incoming items.
type metricsProcessor struct {
	exporter.Metrics
	bt *batchTelemetry
}

func (p *metricsProcessor) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	p.bt.recordIncoming(ctx, int64(md.DataPointCount()))
	return p.Metrics.ConsumeMetrics(ctx, md)
}

func newMetricsProcessor(ctx context.Context, set processor.Settings, cfg *Config, next consumer.Metrics) (processor.Metrics, error) {
	bt, err := newBatchTelemetry(set.TelemetrySettings, set.ID, pipeline.SignalMetrics)
	if err != nil {
		return nil, err
	}
	sizer := &pmetric.ProtoMarshaler{}
	pusher := func(ctx context.Context, md pmetric.Metrics) error {
		bt.recordOutgoing(ctx, int64(md.DataPointCount()), func() int64 { return int64(sizer.MetricsSize(md)) })
		return next.ConsumeMetrics(ctx, md)
	}
	inner, err := exporterhelper.NewMetrics(ctx, exporterSettings(set), cfg, pusher, queueOptions(cfg)...)
	if err != nil {
		return nil, err
	}
	return &metricsProcessor{Metrics: inner, bt: bt}, nil
}

// logsProcessor wraps the exporterhelper result to count incoming items.
type logsProcessor struct {
	exporter.Logs
	bt *batchTelemetry
}

func (p *logsProcessor) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	p.bt.recordIncoming(ctx, int64(ld.LogRecordCount()))
	return p.Logs.ConsumeLogs(ctx, ld)
}

func newLogsProcessor(ctx context.Context, set processor.Settings, cfg *Config, next consumer.Logs) (processor.Logs, error) {
	bt, err := newBatchTelemetry(set.TelemetrySettings, set.ID, pipeline.SignalLogs)
	if err != nil {
		return nil, err
	}
	sizer := &plog.ProtoMarshaler{}
	pusher := func(ctx context.Context, ld plog.Logs) error {
		bt.recordOutgoing(ctx, int64(ld.LogRecordCount()), func() int64 { return int64(sizer.LogsSize(ld)) })
		return next.ConsumeLogs(ctx, ld)
	}
	inner, err := exporterhelper.NewLogs(ctx, exporterSettings(set), cfg, pusher, queueOptions(cfg)...)
	if err != nil {
		return nil, err
	}
	return &logsProcessor{Logs: inner, bt: bt}, nil
}
