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

package obsreport // import "go.opentelemetry.io/collector/obsreport"

import (
	"context"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
)

const (
	exporterName = "exporter"

	exporterScope = scopeName + nameSep + exporterName
)

// Exporter is a helper to add observability to a component.Exporter.
type Exporter struct {
	level          configtelemetry.Level
	spanNamePrefix string
	mutators       []tag.Mutator
	tracer         trace.Tracer
	logger         *zap.Logger

	useOtelForMetrics        bool
	otelAttrs                []attribute.KeyValue
	sentSpans                instrument.Int64Counter
	failedToSendSpans        instrument.Int64Counter
	sentMetricPoints         instrument.Int64Counter
	failedToSendMetricPoints instrument.Int64Counter
	sentLogRecords           instrument.Int64Counter
	failedToSendLogRecords   instrument.Int64Counter
}

// ExporterSettings are settings for creating an Exporter.
type ExporterSettings struct {
	ExporterID             component.ID
	ExporterCreateSettings exporter.CreateSettings
}

// NewExporter creates a new Exporter.
func NewExporter(cfg ExporterSettings) (*Exporter, error) {
	return newExporter(cfg, obsreportconfig.UseOtelForInternalMetricsfeatureGate.IsEnabled())
}

func newExporter(cfg ExporterSettings, useOtel bool) (*Exporter, error) {
	exp := &Exporter{
		level:          cfg.ExporterCreateSettings.TelemetrySettings.MetricsLevel,
		spanNamePrefix: obsmetrics.ExporterPrefix + cfg.ExporterID.String(),
		mutators:       []tag.Mutator{tag.Upsert(obsmetrics.TagKeyExporter, cfg.ExporterID.String(), tag.WithTTL(tag.TTLNoPropagation))},
		tracer:         cfg.ExporterCreateSettings.TracerProvider.Tracer(cfg.ExporterID.String()),
		logger:         cfg.ExporterCreateSettings.Logger,

		useOtelForMetrics: useOtel,
		otelAttrs: []attribute.KeyValue{
			attribute.String(obsmetrics.ExporterKey, cfg.ExporterID.String()),
		},
	}

	if err := exp.createOtelMetrics(cfg); err != nil {
		return nil, err
	}

	return exp, nil
}

func (exp *Exporter) createOtelMetrics(cfg ExporterSettings) error {
	if !exp.useOtelForMetrics {
		return nil
	}
	meter := cfg.ExporterCreateSettings.MeterProvider.Meter(exporterScope)

	var errors, err error

	exp.sentSpans, err = meter.Int64Counter(
		obsmetrics.ExporterPrefix+obsmetrics.SentSpansKey,
		instrument.WithDescription("Number of spans successfully sent to destination."),
		instrument.WithUnit("1"))
	errors = multierr.Append(errors, err)

	exp.failedToSendSpans, err = meter.Int64Counter(
		obsmetrics.ExporterPrefix+obsmetrics.FailedToSendSpansKey,
		instrument.WithDescription("Number of spans in failed attempts to send to destination."),
		instrument.WithUnit("1"))
	errors = multierr.Append(errors, err)

	exp.sentMetricPoints, err = meter.Int64Counter(
		obsmetrics.ExporterPrefix+obsmetrics.SentMetricPointsKey,
		instrument.WithDescription("Number of metric points successfully sent to destination."),
		instrument.WithUnit("1"))
	errors = multierr.Append(errors, err)

	exp.failedToSendMetricPoints, err = meter.Int64Counter(
		obsmetrics.ExporterPrefix+obsmetrics.FailedToSendMetricPointsKey,
		instrument.WithDescription("Number of metric points in failed attempts to send to destination."),
		instrument.WithUnit("1"))
	errors = multierr.Append(errors, err)

	exp.sentLogRecords, err = meter.Int64Counter(
		obsmetrics.ExporterPrefix+obsmetrics.SentLogRecordsKey,
		instrument.WithDescription("Number of log record successfully sent to destination."),
		instrument.WithUnit("1"))
	errors = multierr.Append(errors, err)

	exp.failedToSendLogRecords, err = meter.Int64Counter(
		obsmetrics.ExporterPrefix+obsmetrics.FailedToSendLogRecordsKey,
		instrument.WithDescription("Number of log records in failed attempts to send to destination."),
		instrument.WithUnit("1"))
	errors = multierr.Append(errors, err)

	return errors
}

// StartTracesOp is called at the start of an Export operation.
// The returned context should be used in other calls to the Exporter functions
// dealing with the same export operation.
func (exp *Exporter) StartTracesOp(ctx context.Context) context.Context {
	return exp.startOp(ctx, obsmetrics.ExportTraceDataOperationSuffix)
}

// EndTracesOp completes the export operation that was started with StartTracesOp.
func (exp *Exporter) EndTracesOp(ctx context.Context, numSpans int, err error) {
	numSent, numFailedToSend := toNumItems(numSpans, err)
	exp.recordMetrics(ctx, component.DataTypeTraces, numSent, numFailedToSend)
	endSpan(ctx, err, numSent, numFailedToSend, obsmetrics.SentSpansKey, obsmetrics.FailedToSendSpansKey)
}

// StartMetricsOp is called at the start of an Export operation.
// The returned context should be used in other calls to the Exporter functions
// dealing with the same export operation.
func (exp *Exporter) StartMetricsOp(ctx context.Context) context.Context {
	return exp.startOp(ctx, obsmetrics.ExportMetricsOperationSuffix)
}

// EndMetricsOp completes the export operation that was started with
// StartMetricsOp.
func (exp *Exporter) EndMetricsOp(ctx context.Context, numMetricPoints int, err error) {
	numSent, numFailedToSend := toNumItems(numMetricPoints, err)
	exp.recordMetrics(ctx, component.DataTypeMetrics, numSent, numFailedToSend)
	endSpan(ctx, err, numSent, numFailedToSend, obsmetrics.SentMetricPointsKey, obsmetrics.FailedToSendMetricPointsKey)
}

// StartLogsOp is called at the start of an Export operation.
// The returned context should be used in other calls to the Exporter functions
// dealing with the same export operation.
func (exp *Exporter) StartLogsOp(ctx context.Context) context.Context {
	return exp.startOp(ctx, obsmetrics.ExportLogsOperationSuffix)
}

// EndLogsOp completes the export operation that was started with StartLogsOp.
func (exp *Exporter) EndLogsOp(ctx context.Context, numLogRecords int, err error) {
	numSent, numFailedToSend := toNumItems(numLogRecords, err)
	exp.recordMetrics(ctx, component.DataTypeLogs, numSent, numFailedToSend)
	endSpan(ctx, err, numSent, numFailedToSend, obsmetrics.SentLogRecordsKey, obsmetrics.FailedToSendLogRecordsKey)
}

// startOp creates the span used to trace the operation. Returning
// the updated context and the created span.
func (exp *Exporter) startOp(ctx context.Context, operationSuffix string) context.Context {
	spanName := exp.spanNamePrefix + operationSuffix
	ctx, _ = exp.tracer.Start(ctx, spanName)
	return ctx
}

func (exp *Exporter) recordMetrics(ctx context.Context, dataType component.DataType, numSent, numFailed int64) {
	if exp.level == configtelemetry.LevelNone {
		return
	}
	if exp.useOtelForMetrics {
		exp.recordWithOtel(ctx, dataType, numSent, numFailed)
	} else {
		exp.recordWithOC(ctx, dataType, numSent, numFailed)
	}
}

func (exp *Exporter) recordWithOtel(ctx context.Context, dataType component.DataType, sent int64, failed int64) {
	var sentMeasure, failedMeasure instrument.Int64Counter
	switch dataType {
	case component.DataTypeTraces:
		sentMeasure = exp.sentSpans
		failedMeasure = exp.failedToSendSpans
	case component.DataTypeMetrics:
		sentMeasure = exp.sentMetricPoints
		failedMeasure = exp.failedToSendMetricPoints
	case component.DataTypeLogs:
		sentMeasure = exp.sentLogRecords
		failedMeasure = exp.failedToSendLogRecords
	}

	sentMeasure.Add(ctx, sent, exp.otelAttrs...)
	failedMeasure.Add(ctx, failed, exp.otelAttrs...)
}

func (exp *Exporter) recordWithOC(ctx context.Context, dataType component.DataType, sent int64, failed int64) {
	var sentMeasure, failedMeasure *stats.Int64Measure
	switch dataType {
	case component.DataTypeTraces:
		sentMeasure = obsmetrics.ExporterSentSpans
		failedMeasure = obsmetrics.ExporterFailedToSendSpans
	case component.DataTypeMetrics:
		sentMeasure = obsmetrics.ExporterSentMetricPoints
		failedMeasure = obsmetrics.ExporterFailedToSendMetricPoints
	case component.DataTypeLogs:
		sentMeasure = obsmetrics.ExporterSentLogRecords
		failedMeasure = obsmetrics.ExporterFailedToSendLogRecords
	}

	if failed > 0 {
		_ = stats.RecordWithTags(
			ctx,
			exp.mutators,
			sentMeasure.M(sent),
			failedMeasure.M(failed))
	} else {
		_ = stats.RecordWithTags(
			ctx,
			exp.mutators,
			sentMeasure.M(sent))
	}
}

func endSpan(ctx context.Context, err error, numSent, numFailedToSend int64, sentItemsKey, failedToSendItemsKey string) {
	span := trace.SpanFromContext(ctx)
	// End the span according to errors.
	if span.IsRecording() {
		span.SetAttributes(
			attribute.Int64(sentItemsKey, numSent),
			attribute.Int64(failedToSendItemsKey, numFailedToSend),
		)
		recordError(span, err)
	}
	span.End()
}

func toNumItems(numExportedItems int, err error) (int64, int64) {
	if err != nil {
		return 0, int64(numExportedItems)
	}
	return int64(numExportedItems), 0
}
