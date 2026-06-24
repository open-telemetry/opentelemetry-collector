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
	"go.opentelemetry.io/collector/processor"
)

// exporterSettings derives exporter.Settings from processor.Settings while
// disabling the exporterhelper's own telemetry. Substituting a no-op
// MeterProvider silences every otelcol_exporter_* series (queue and sender)
// that the exporter helper would otherwise emit; this processor reports the
// batch metrics itself under otelcol_processor_batch_* names instead.
func exporterSettings(set processor.Settings) exporter.Settings {
	tel := set.TelemetrySettings
	tel.MeterProvider = noopmetric.NewMeterProvider()
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

func newTracesProcessor(ctx context.Context, set processor.Settings, cfg *Config, next consumer.Traces) (processor.Traces, error) {
	bt, err := newBatchTelemetry(set.TelemetrySettings, set.ID)
	if err != nil {
		return nil, err
	}
	sizer := &ptrace.ProtoMarshaler{}
	pusher := func(ctx context.Context, td ptrace.Traces) error {
		bt.recordBatch(ctx, int64(td.SpanCount()), func() int64 { return int64(sizer.TracesSize(td)) })
		return next.ConsumeTraces(ctx, td)
	}
	return exporterhelper.NewTraces(ctx, exporterSettings(set), cfg, pusher, queueOptions(cfg)...)
}

func newMetricsProcessor(ctx context.Context, set processor.Settings, cfg *Config, next consumer.Metrics) (processor.Metrics, error) {
	bt, err := newBatchTelemetry(set.TelemetrySettings, set.ID)
	if err != nil {
		return nil, err
	}
	sizer := &pmetric.ProtoMarshaler{}
	pusher := func(ctx context.Context, md pmetric.Metrics) error {
		bt.recordBatch(ctx, int64(md.DataPointCount()), func() int64 { return int64(sizer.MetricsSize(md)) })
		return next.ConsumeMetrics(ctx, md)
	}
	return exporterhelper.NewMetrics(ctx, exporterSettings(set), cfg, pusher, queueOptions(cfg)...)
}

func newLogsProcessor(ctx context.Context, set processor.Settings, cfg *Config, next consumer.Logs) (processor.Logs, error) {
	bt, err := newBatchTelemetry(set.TelemetrySettings, set.ID)
	if err != nil {
		return nil, err
	}
	sizer := &plog.ProtoMarshaler{}
	pusher := func(ctx context.Context, ld plog.Logs) error {
		bt.recordBatch(ctx, int64(ld.LogRecordCount()), func() int64 { return int64(sizer.LogsSize(ld)) })
		return next.ConsumeLogs(ctx, ld)
	}
	return exporterhelper.NewLogs(ctx, exporterSettings(set), cfg, pusher, queueOptions(cfg)...)
}
