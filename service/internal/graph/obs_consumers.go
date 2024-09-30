// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"context"

	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/otel/metric"
)

type obsTracesConsumer struct {
	consumer.Traces
	itemCounter metric.Int64Counter
}

func (t obsTracesConsumer) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	t.itemCounter.Add(ctx, int64(td.SpanCount()))
	return t.Traces.ConsumeTraces(ctx, td)
}

type obsMetricsConsumer struct {
	consumer.Metrics
	itemCounter metric.Int64Counter
}

func (t obsMetricsConsumer) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	t.itemCounter.Add(ctx, int64(md.DataPointCount()))
	return t.Metrics.ConsumeMetrics(ctx, md)
}

type obsLogsConsumer struct {
	consumer.Logs
	itemCounter metric.Int64Counter
}

func (t obsLogsConsumer) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	t.itemCounter.Add(ctx, int64(ld.LogRecordCount()))
	return t.Logs.ConsumeLogs(ctx, ld)
}

type obsTracesProcessor struct {
	processor.Traces
	itemCounter metric.Int64Counter
}

func (o obsTracesProcessor) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	o.itemCounter.Add(ctx, int64(td.SpanCount()))
	return o.Traces.ConsumeTraces(ctx, td)
}

type obsMetricsProcessor struct {
	processor.Metrics
	itemCounter metric.Int64Counter
}

func (o obsMetricsProcessor) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	o.itemCounter.Add(ctx, int64(md.DataPointCount()))
	return o.Metrics.ConsumeMetrics(ctx, md)
}

type obsLogsProcessor struct {
	processor.Logs
	itemCounter metric.Int64Counter
}

func (o obsLogsProcessor) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	o.itemCounter.Add(ctx, int64(ld.LogRecordCount()))
	return o.Logs.ConsumeLogs(ctx, ld)
}

type obsTracesExporter struct {
	exporter.Traces
	itemCounter metric.Int64Counter
}

func (o obsTracesExporter) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	o.itemCounter.Add(ctx, int64(td.SpanCount()))
	return o.Traces.ConsumeTraces(ctx, td)
}

type obsMetricsExporter struct {
	exporter.Metrics
	itemCounter metric.Int64Counter
}

func (o obsMetricsExporter) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	o.itemCounter.Add(ctx, int64(md.DataPointCount()))
	return o.Metrics.ConsumeMetrics(ctx, md)
}

type obsLogsExporter struct {
	exporter.Logs
	itemCounter metric.Int64Counter
}

func (o obsLogsExporter) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	o.itemCounter.Add(ctx, int64(ld.LogRecordCount()))
	return o.Logs.ConsumeLogs(ctx, ld)
}

type obsTracesConnector struct {
	connector.Traces
	itemCounter metric.Int64Counter
}

func (o obsTracesConnector) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	o.itemCounter.Add(ctx, int64(td.SpanCount()))
	return o.Traces.ConsumeTraces(ctx, td)
}

type obsMetricsConnector struct {
	connector.Metrics
	itemCounter metric.Int64Counter
}

func (o obsMetricsConnector) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	o.itemCounter.Add(ctx, int64(md.DataPointCount()))
	return o.Metrics.ConsumeMetrics(ctx, md)
}

type obsLogsConnector struct {
	connector.Logs
	itemCounter metric.Int64Counter
}

func (o obsLogsConnector) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	o.itemCounter.Add(ctx, int64(ld.LogRecordCount()))
	return o.Logs.ConsumeLogs(ctx, ld)
}
