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

package obsreport

import (
	"context"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
)

// Exporter is a helper to add observability to a component.Exporter.
type Exporter struct {
	level          configtelemetry.Level
	spanNamePrefix string
	mutators       []tag.Mutator
}

// ExporterSettings are settings for creating an Exporter.
type ExporterSettings struct {
	Level      configtelemetry.Level
	ExporterID config.ComponentID
}

// NewExporter creates a new Exporter.
func NewExporter(cfg ExporterSettings) *Exporter {
	return &Exporter{
		level:          cfg.Level,
		spanNamePrefix: obsmetrics.ExporterPrefix + cfg.ExporterID.String(),
		mutators:       []tag.Mutator{tag.Upsert(obsmetrics.TagKeyExporter, cfg.ExporterID.String(), tag.WithTTL(tag.TTLNoPropagation))},
	}
}

// StartTracesOp is called at the start of an Export operation.
// The returned context should be used in other calls to the Exporter functions
// dealing with the same export operation.
func (eor *Exporter) StartTracesOp(ctx context.Context) context.Context {
	return eor.startSpan(ctx, obsmetrics.ExportTraceDataOperationSuffix)
}

// EndTracesOp completes the export operation that was started with StartTracesOp.
func (eor *Exporter) EndTracesOp(ctx context.Context, numSpans int, err error) {
	numSent, numFailedToSend := toNumItems(numSpans, err)
	eor.recordMetrics(ctx, numSent, numFailedToSend, obsmetrics.ExporterSentSpans, obsmetrics.ExporterFailedToSendSpans)
	endSpan(ctx, err, numSent, numFailedToSend, obsmetrics.SentSpansKey, obsmetrics.FailedToSendSpansKey)
}

// RecordTracesEnqueueFailure records number of spans that failed to be added to the sending queue.
func (eor *Exporter) RecordTracesEnqueueFailure(ctx context.Context, numSpans int) {
	_ = stats.RecordWithTags(ctx, eor.mutators, obsmetrics.ExporterFailedToEnqueueSpans.M(int64(numSpans)))
}

// StartMetricsOp is called at the start of an Export operation.
// The returned context should be used in other calls to the Exporter functions
// dealing with the same export operation.
func (eor *Exporter) StartMetricsOp(ctx context.Context) context.Context {
	return eor.startSpan(ctx, obsmetrics.ExportMetricsOperationSuffix)
}

// EndMetricsOp completes the export operation that was started with
// StartMetricsOp.
func (eor *Exporter) EndMetricsOp(ctx context.Context, numMetricPoints int, err error) {
	numSent, numFailedToSend := toNumItems(numMetricPoints, err)
	eor.recordMetrics(ctx, numSent, numFailedToSend, obsmetrics.ExporterSentMetricPoints, obsmetrics.ExporterFailedToSendMetricPoints)
	endSpan(ctx, err, numSent, numFailedToSend, obsmetrics.SentMetricPointsKey, obsmetrics.FailedToSendMetricPointsKey)
}

// RecordMetricsEnqueueFailure records number of metric points that failed to be added to the sending queue.
func (eor *Exporter) RecordMetricsEnqueueFailure(ctx context.Context, numMetricPoints int) {
	_ = stats.RecordWithTags(ctx, eor.mutators, obsmetrics.ExporterFailedToEnqueueMetricPoints.M(int64(numMetricPoints)))
}

// StartLogsOp is called at the start of an Export operation.
// The returned context should be used in other calls to the Exporter functions
// dealing with the same export operation.
func (eor *Exporter) StartLogsOp(ctx context.Context) context.Context {
	return eor.startSpan(ctx, obsmetrics.ExportLogsOperationSuffix)
}

// EndLogsOp completes the export operation that was started with StartLogsOp.
func (eor *Exporter) EndLogsOp(ctx context.Context, numLogRecords int, err error) {
	numSent, numFailedToSend := toNumItems(numLogRecords, err)
	eor.recordMetrics(ctx, numSent, numFailedToSend, obsmetrics.ExporterSentLogRecords, obsmetrics.ExporterFailedToSendLogRecords)
	endSpan(ctx, err, numSent, numFailedToSend, obsmetrics.SentLogRecordsKey, obsmetrics.FailedToSendLogRecordsKey)
}

// RecordLogsEnqueueFailure records number of log records that failed to be added to the sending queue.
func (eor *Exporter) RecordLogsEnqueueFailure(ctx context.Context, numLogRecords int) {
	_ = stats.RecordWithTags(ctx, eor.mutators, obsmetrics.ExporterFailedToEnqueueLogRecords.M(int64(numLogRecords)))
}

// startSpan creates the span used to trace the operation. Returning
// the updated context and the created span.
func (eor *Exporter) startSpan(ctx context.Context, operationSuffix string) context.Context {
	spanName := eor.spanNamePrefix + operationSuffix
	ctx, _ = trace.StartSpan(ctx, spanName)
	return ctx
}

func (eor *Exporter) recordMetrics(ctx context.Context, numSent, numFailedToSend int64, sentMeasure, failedToSendMeasure *stats.Int64Measure) {
	if obsreportconfig.Level == configtelemetry.LevelNone {
		return
	}
	// Ignore the error for now. This should not happen.
	_ = stats.RecordWithTags(
		ctx,
		eor.mutators,
		sentMeasure.M(numSent),
		failedToSendMeasure.M(numFailedToSend))
}

func endSpan(ctx context.Context, err error, numSent, numFailedToSend int64, sentItemsKey, failedToSendItemsKey string) {
	span := trace.FromContext(ctx)
	// End span according to errors.
	if span.IsRecordingEvents() {
		span.AddAttributes(
			trace.Int64Attribute(
				sentItemsKey, numSent),
			trace.Int64Attribute(
				failedToSendItemsKey, numFailedToSend),
		)
		span.SetStatus(errToStatus(err))
	}
	span.End()
}

func toNumItems(numExportedItems int, err error) (int64, int64) {
	if err != nil {
		return 0, int64(numExportedItems)
	}
	return int64(numExportedItems), 0
}
