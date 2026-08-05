// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
)

// ObsMetrics reports the metrics produced by exporterhelper for one signal.
// Components that reuse exporterhelper can provide their own implementation to
// report metrics using names and attributes appropriate for that component.
// Exporterhelper owns the instance after WithObsMetrics applies successfully
// and calls Shutdown when construction fails or the component shuts down.
type ObsMetrics interface {
	queue.Metrics
	RecordInFlight(context.Context, int64)
	RecordSent(context.Context, int64)
	RecordSendFailure(context.Context, int64, ...metric.AddOption)
	Shutdown()
}

type RecordInFlightFunc func(context.Context, int64)

func (f RecordInFlightFunc) RecordInFlight(ctx context.Context, delta int64) {
	if f != nil {
		f(ctx, delta)
	}
}

type RecordSentFunc func(context.Context, int64)

func (f RecordSentFunc) RecordSent(ctx context.Context, items int64) {
	if f != nil {
		f(ctx, items)
	}
}

type RecordSendFailureFunc func(context.Context, int64, ...metric.AddOption)

func (f RecordSendFailureFunc) RecordSendFailure(ctx context.Context, items int64, options ...metric.AddOption) {
	if f != nil {
		f(ctx, items, options...)
	}
}

type ShutdownObsMetricsFunc func()

func (f ShutdownObsMetricsFunc) Shutdown() {
	if f != nil {
		f()
	}
}

type exporterObsMetrics struct {
	queue.Metrics
	RecordInFlightFunc
	RecordSentFunc
	RecordSendFailureFunc
	ShutdownObsMetricsFunc
}

func newExporterObsMetrics(
	tel component.TelemetrySettings,
	id component.ID,
	signal pipeline.Signal,
	extraAttrs []attribute.KeyValue,
) (ObsMetrics, error) {
	tb, err := metadata.NewTelemetryBuilder(tel)
	if err != nil {
		return nil, err
	}

	exporterAttr := attribute.String(ExporterKey, id.String())
	metricAttr := metric.WithAttributeSet(attribute.NewSet(append(extraAttrs, exporterAttr)...))
	asyncAttr := metric.WithAttributeSet(attribute.NewSet(exporterAttr, attribute.String(DataTypeKey, signal.String())))
	var itemsSentInst, itemsFailedInst metric.Int64Counter
	switch signal {
	case pipeline.SignalTraces:
		itemsSentInst = tb.ExporterSentSpans
		itemsFailedInst = tb.ExporterSendFailedSpans
	case pipeline.SignalMetrics:
		itemsSentInst = tb.ExporterSentMetricPoints
		itemsFailedInst = tb.ExporterSendFailedMetricPoints
	case pipeline.SignalLogs:
		itemsSentInst = tb.ExporterSentLogRecords
		itemsFailedInst = tb.ExporterSendFailedLogRecords
	case xpipeline.SignalProfiles:
		itemsSentInst = tb.ExporterSentProfileSamples
		itemsFailedInst = tb.ExporterSendFailedProfileSamples
	}
	om := &exporterObsMetrics{
		Metrics: queue.NewExporterMetrics(tb, id, signal),
		RecordInFlightFunc: func(ctx context.Context, delta int64) {
			tb.ExporterInFlightRequests.Add(ctx, delta, asyncAttr)
		},
		ShutdownObsMetricsFunc: tb.Shutdown,
	}
	if itemsSentInst != nil {
		om.RecordSentFunc = func(ctx context.Context, items int64) {
			itemsSentInst.Add(ctx, items, metricAttr)
		}
	}
	if itemsFailedInst != nil {
		om.RecordSendFailureFunc = func(ctx context.Context, items int64, options ...metric.AddOption) {
			itemsFailedInst.Add(ctx, items, append([]metric.AddOption{metricAttr}, options...)...)
		}
	}

	return om, nil
}
