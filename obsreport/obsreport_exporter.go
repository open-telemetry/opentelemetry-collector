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

	"go.opentelemetry.io/collector/config/configtelemetry"
)

const (
	// Key used to identify exporters in metrics and traces.
	ExporterKey = "exporter"

	// Key used to track spans sent by exporters.
	SentSpansKey = "sent_spans"
	// Key used to track spans that failed to be sent by exporters.
	FailedToSendSpansKey = "send_failed_spans"

	// Key used to track metric points sent by exporters.
	SentMetricPointsKey = "sent_metric_points"
	// Key used to track metric points that failed to be sent by exporters.
	FailedToSendMetricPointsKey = "send_failed_metric_points"

	// Key used to track logs sent by exporters.
	SentLogRecordsKey = "sent_log_records"
	// Key used to track logs that failed to be sent by exporters.
	FailedToSendLogRecordsKey = "send_failed_log_records"
)

var (
	tagKeyExporter, _ = tag.NewKey(ExporterKey)

	exporterPrefix                 = ExporterKey + nameSep
	exportTraceDataOperationSuffix = nameSep + "traces"
	exportMetricsOperationSuffix   = nameSep + "metrics"
	exportLogsOperationSuffix      = nameSep + "logs"

	// Exporter metrics. Any count of data items below is in the final format
	// that they were sent, reasoning: reconciliation is easier if measurements
	// on backend and exporter are expected to be the same. Translation issues
	// that result in a different number of elements should be reported in a
	// separate way.
	mExporterSentSpans = stats.Int64(
		exporterPrefix+SentSpansKey,
		"Number of spans successfully sent to destination.",
		stats.UnitDimensionless)
	mExporterFailedToSendSpans = stats.Int64(
		exporterPrefix+FailedToSendSpansKey,
		"Number of spans in failed attempts to send to destination.",
		stats.UnitDimensionless)
	mExporterSentMetricPoints = stats.Int64(
		exporterPrefix+SentMetricPointsKey,
		"Number of metric points successfully sent to destination.",
		stats.UnitDimensionless)
	mExporterFailedToSendMetricPoints = stats.Int64(
		exporterPrefix+FailedToSendMetricPointsKey,
		"Number of metric points in failed attempts to send to destination.",
		stats.UnitDimensionless)
	mExporterSentLogRecords = stats.Int64(
		exporterPrefix+SentLogRecordsKey,
		"Number of log record successfully sent to destination.",
		stats.UnitDimensionless)
	mExporterFailedToSendLogRecords = stats.Int64(
		exporterPrefix+FailedToSendLogRecordsKey,
		"Number of log records in failed attempts to send to destination.",
		stats.UnitDimensionless)
)

type Exporter struct {
	level        configtelemetry.Level
	exporterName string
	mutators     []tag.Mutator
}

func NewExporter(level configtelemetry.Level, exporterName string) *Exporter {
	return &Exporter{
		level:        level,
		exporterName: exporterName,
		mutators:     []tag.Mutator{tag.Upsert(tagKeyExporter, exporterName, tag.WithTTL(tag.TTLNoPropagation))},
	}
}

// StartTracesExportOp is called at the start of an Export operation.
// The returned context should be used in other calls to the Exporter functions
// dealing with the same export operation.
func (eor *Exporter) StartTracesExportOp(ctx context.Context) context.Context {
	return eor.startSpan(ctx, exportTraceDataOperationSuffix)
}

// EndTracesExportOp completes the export operation that was started with StartTracesExportOp.
func (eor *Exporter) EndTracesExportOp(ctx context.Context, numSpans int, err error) {
	numSent, numFailedToSend := toNumItems(numSpans, err)
	eor.recordMetrics(ctx, numSent, numFailedToSend, mExporterSentSpans, mExporterFailedToSendSpans)
	endSpan(ctx, err, numSent, numFailedToSend, SentSpansKey, FailedToSendSpansKey)
}

// StartMetricsExportOp is called at the start of an Export operation.
// The returned context should be used in other calls to the Exporter functions
// dealing with the same export operation.
func (eor *Exporter) StartMetricsExportOp(ctx context.Context) context.Context {
	return eor.startSpan(ctx, exportMetricsOperationSuffix)
}

// EndMetricsExportOp completes the export operation that was started with
// StartMetricsExportOp.
func (eor *Exporter) EndMetricsExportOp(ctx context.Context, numMetricPoints int, err error) {
	numSent, numFailedToSend := toNumItems(numMetricPoints, err)
	eor.recordMetrics(ctx, numSent, numFailedToSend, mExporterSentMetricPoints, mExporterFailedToSendMetricPoints)
	endSpan(ctx, err, numSent, numFailedToSend, SentMetricPointsKey, FailedToSendMetricPointsKey)
}

// StartLogsExportOp is called at the start of an Export operation.
// The returned context should be used in other calls to the Exporter functions
// dealing with the same export operation.
func (eor *Exporter) StartLogsExportOp(ctx context.Context) context.Context {
	return eor.startSpan(ctx, exportLogsOperationSuffix)
}

// EndLogsExportOp completes the export operation that was started with StartLogsExportOp.
func (eor *Exporter) EndLogsExportOp(ctx context.Context, numLogRecords int, err error) {
	numSent, numFailedToSend := toNumItems(numLogRecords, err)
	eor.recordMetrics(ctx, numSent, numFailedToSend, mExporterSentLogRecords, mExporterFailedToSendLogRecords)
	endSpan(ctx, err, numSent, numFailedToSend, SentLogRecordsKey, FailedToSendLogRecordsKey)
}

// startSpan creates the span used to trace the operation. Returning
// the updated context and the created span.
func (eor *Exporter) startSpan(ctx context.Context, operationSuffix string) context.Context {
	spanName := exporterPrefix + eor.exporterName + operationSuffix
	ctx, _ = trace.StartSpan(ctx, spanName)
	return ctx
}

func (eor *Exporter) recordMetrics(ctx context.Context, numSent, numFailedToSend int64, sentMeasure, failedToSendMeasure *stats.Int64Measure) {
	if gLevel == configtelemetry.LevelNone {
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
