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
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/label"

	"go.opentelemetry.io/collector/config/configmodels"
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
	defaultExportTracer = global.Tracer("")
	tagKeyExporter, _   = tag.NewKey(ExporterKey)

	exporterPrefix                 = ExporterKey + nameSep
	exportTraceDataOperationSuffix = nameSep + "TraceDataExported"
	exportMetricsOperationSuffix   = nameSep + "MetricsExported"
	exportLogsOperationSuffix      = nameSep + "LogRecordsExported"

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

// startReceiveOptions has the options related to starting a receive operation.
type startExportOptions struct {
	tracer trace.Tracer
}

// StartExporterOption function applies options to Start*ExportOp.
type StartExportOption func(*startExportOptions)

// WithExportTracer assign a non-default Tracer to be used when doing tracing.
func WithExportTracer(tracer trace.Tracer) StartExportOption {
	return func(opts *startExportOptions) {
		opts.tracer = tracer
	}
}

func toExportOptions(opts []StartExportOption) startExportOptions {
	seo := startExportOptions{
		tracer: defaultExportTracer,
	}
	for _, o := range opts {
		o(&seo)
	}
	return seo
}

// StartTraceDataExportOp is called at the start of an Export operation.
// The returned context should be used in other calls to the obsreport functions
// dealing with the same export operation.
func StartTraceDataExportOp(
	operationCtx context.Context,
	exporter string,
	opts ...StartExportOption,
) context.Context {
	return traceExportDataOp(operationCtx, exporter, exportTraceDataOperationSuffix, opts)
}

// EndTraceDataExportOp completes the export operation that was started with
// StartTraceDataExportOp.
func EndTraceDataExportOp(
	exporterCtx context.Context,
	numExportedSpans int,
	numDroppedSpans int, // TODO: For legacy measurements, to be removed in the future.
	err error,
) {
	if useLegacy {
		stats.Record(exporterCtx, mExporterReceivedSpans.M(int64(numExportedSpans)), mExporterDroppedSpans.M(int64(numDroppedSpans)))
	}

	endExportOp(
		exporterCtx,
		numExportedSpans,
		err,
		configmodels.TracesDataType,
	)
}

// StartMetricsExportOp is called at the start of an Export operation.
// The returned context should be used in other calls to the obsreport functions
// dealing with the same export operation.
func StartMetricsExportOp(
	operationCtx context.Context,
	exporter string,
	opts ...StartExportOption,
) context.Context {
	return traceExportDataOp(operationCtx, exporter, exportMetricsOperationSuffix, opts)
}

// EndMetricsExportOp completes the export operation that was started with
// StartMetricsExportOp.
func EndMetricsExportOp(
	exporterCtx context.Context,
	numExportedPoints int,
	numExportedTimeSeries int, // TODO: For legacy measurements, to be removed in the future.
	numDroppedTimeSeries int, // TODO: For legacy measurements, to be removed in the future.
	err error,
) {
	if useLegacy {
		stats.Record(exporterCtx, mExporterReceivedTimeSeries.M(int64(numExportedTimeSeries)), mExporterDroppedTimeSeries.M(int64(numDroppedTimeSeries)))
	}

	endExportOp(
		exporterCtx,
		numExportedPoints,
		err,
		configmodels.MetricsDataType,
	)
}

// StartLogsExportOp is called at the start of an Export operation.
// The returned context should be used in other calls to the obsreport functions
// dealing with the same export operation.
func StartLogsExportOp(
	operationCtx context.Context,
	exporter string,
	opts ...StartExportOption,
) context.Context {
	return traceExportDataOp(operationCtx, exporter, exportLogsOperationSuffix, opts)
}

// EndLogsExportOp completes the export operation that was started with
// StartLogsExportOp.
func EndLogsExportOp(
	exporterCtx context.Context,
	numExportedLogs int,
	numDroppedLogs int, // TODO: For legacy measurements, to be removed in the future.
	err error,
) {
	if useLegacy {
		stats.Record(exporterCtx, mExporterReceivedLogRecords.M(int64(numExportedLogs)), mExporterDroppedLogRecords.M(int64(numDroppedLogs)))
	}

	endExportOp(
		exporterCtx,
		numExportedLogs,
		err,
		configmodels.LogsDataType,
	)
}

// ExporterContext adds the keys used when recording observability metrics to
// the given context returning the newly created context. This context should
// be used in related calls to the obsreport functions so metrics are properly
// recorded.
func ExporterContext(
	ctx context.Context,
	exporter string,
) context.Context {
	if useLegacy {
		ctx, _ = tag.New(ctx, tag.Upsert(LegacyTagKeyExporter, exporter, tag.WithTTL(tag.TTLNoPropagation)))
	}

	ctx, _ = tag.New(ctx, tag.Upsert(tagKeyExporter, exporter, tag.WithTTL(tag.TTLNoPropagation)))

	return ctx
}

// traceExportDataOp creates the span used to trace the operation. Returning
// the updated context and the created span.
func traceExportDataOp(
	exporterCtx context.Context,
	exporterName string,
	operationSuffix string,
	opts []StartExportOption,
) context.Context {
	seo := toExportOptions(opts)
	spanName := exporterPrefix + exporterName + operationSuffix
	ctx, _ := seo.tracer.Start(exporterCtx, spanName)
	return ctx
}

// endExportOp records the observability signals at the end of an operation.
func endExportOp(
	exporterCtx context.Context,
	numExportedItems int,
	err error,
	dataType configmodels.DataType,
) {
	numSent := numExportedItems
	numFailedToSend := 0
	if err != nil {
		numSent = 0
		numFailedToSend = numExportedItems
	}

	if useNew {
		var sentMeasure, failedToSendMeasure *stats.Int64Measure
		switch dataType {
		case configmodels.TracesDataType:
			sentMeasure = mExporterSentSpans
			failedToSendMeasure = mExporterFailedToSendSpans
		case configmodels.MetricsDataType:
			sentMeasure = mExporterSentMetricPoints
			failedToSendMeasure = mExporterFailedToSendMetricPoints
		case configmodels.LogsDataType:
			sentMeasure = mExporterSentLogRecords
			failedToSendMeasure = mExporterFailedToSendLogRecords
		default:
			panic("unknown data type for internal metrics")
		}

		stats.Record(
			exporterCtx,
			sentMeasure.M(int64(numSent)),
			failedToSendMeasure.M(int64(numFailedToSend)))
	}

	span := trace.SpanFromContext(exporterCtx)
	// End span according to errors.
	if span.IsRecording() {
		var sentItemsKey, failedToSendItemsKey string
		switch dataType {
		case configmodels.TracesDataType:
			sentItemsKey = SentSpansKey
			failedToSendItemsKey = FailedToSendSpansKey
		case configmodels.MetricsDataType:
			sentItemsKey = SentMetricPointsKey
			failedToSendItemsKey = FailedToSendMetricPointsKey
		case configmodels.LogsDataType:
			sentItemsKey = SentLogRecordsKey
			failedToSendItemsKey = FailedToSendLogRecordsKey
		}

		span.SetAttributes(
			label.Int64(sentItemsKey, int64(numSent)),
			label.Int64(failedToSendItemsKey, int64(numFailedToSend)),
		)
		if err != nil {
			span.SetStatus(codes.Unknown, err.Error())
		}
	}
	span.End()
}
