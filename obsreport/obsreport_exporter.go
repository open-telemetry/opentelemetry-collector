// Copyright 2020 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/observability"
)

const (
	ExporterKey = "exporter"

	SentSpansKey         = "sent_spans"
	FailedToSendSpansKey = "send_failed_spans"

	SentMetricPointsKey         = "sent_metric_points"
	FailedToSendMetricPointsKey = "send_failed_metric_points"
)

var (
	tagKeyExporter, _ = tag.NewKey(ExporterKey)

	exporterPrefix                 = ExporterKey + nameSep
	exportTraceDataOperationSuffix = nameSep + "TraceDataExported"
	exportMetricsOperationSuffix   = nameSep + "MetricsExported"

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
		exporterPrefix+RefusedMetricPointsKey,
		"Number of metric points in failed attempts to send to destination.",
		stats.UnitDimensionless)
)

// StartTraceDataExportOp is called at the start of an Export operation.
// The returned context should be used in other calls to the obsreport functions
// dealing with the same export operation.
func StartTraceDataExportOp(ctx context.Context, exporter string) (context.Context, *trace.Span) {
	return traceExportDataOp(
		exporterContext(ctx, exporter),
		exporter,
		exportTraceDataOperationSuffix)
}

// EndTraceDataExportOp completes the export operation that was started with
// StartTraceDataExportOp.
func EndTraceDataExportOp(
	ctx context.Context,
	span *trace.Span,
	numExportedSpans int,
	numDroppedSpans int, // TODO: For legacy measurements, to be removed in the future.
	err error,
) {
	if useLegacy {
		observability.RecordMetricsForTraceExporter(
			ctx, numExportedSpans, numDroppedSpans)
	}

	endExportOp(
		ctx,
		span,
		numExportedSpans,
		err,
		configmodels.TracesDataType,
	)
}

// StartMetricsExportOp is called at the start of an Export operation.
// The returned context should be used in other calls to the obsreport functions
// dealing with the same export operation.
func StartMetricsExportOp(
	ctx context.Context,
	exporter string,
) (context.Context, *trace.Span) {
	return traceExportDataOp(
		exporterContext(ctx, exporter),
		exporter,
		exportMetricsOperationSuffix)
}

// EndMetricsExportOp completes the export operation that was started with
// StartMetricsExportOp.
func EndMetricsExportOp(
	ctx context.Context,
	span *trace.Span,
	numExportedPoints int,
	numExportedTimeSeries int, // TODO: For legacy measurements, to be removed in the future.
	numDroppedTimeSeries int, // TODO: For legacy measurements, to be removed in the future.
	err error,
) {
	if useLegacy {
		observability.RecordMetricsForMetricsExporter(
			ctx, numExportedTimeSeries, numDroppedTimeSeries)
	}

	endExportOp(
		ctx,
		span,
		numExportedPoints,
		err,
		configmodels.MetricsDataType,
	)
}

// exporterContext adds the keys used when recording observability metrics to
// the given context returning the newly created context. This context should
// be used in related calls to the obsreport functions so metrics are properly
// recorded.
func exporterContext(
	ctx context.Context,
	exporter string,
) context.Context {
	if useLegacy {
		ctx = observability.ContextWithExporterName(ctx, exporter)
	}

	if useNew {
		ctx, _ = tag.New(ctx, tag.Upsert(
			tagKeyExporter, exporter, tag.WithTTL(tag.TTLNoPropagation)))
	}

	return ctx
}

// traceExportDataOp creates the span used to trace the operation. Returning
// the updated context and the created span.
func traceExportDataOp(
	exporterCtx context.Context,
	exporterName string,
	operationSuffix string,
) (context.Context, *trace.Span) {
	spanName := exporterPrefix + exporterName + operationSuffix
	return trace.StartSpan(exporterCtx, spanName)
}

// endExportOp records the observability signals at the end of an operation.
func endExportOp(
	exporterCtx context.Context,
	span *trace.Span,
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

	var sentMeasure, failedToSendMeasure *stats.Int64Measure
	var sentItemsKey, failedToSendItemsKey string
	switch dataType {
	case configmodels.TracesDataType:
		sentMeasure = mExporterSentSpans
		failedToSendMeasure = mExporterFailedToSendSpans
		sentItemsKey = SentSpansKey
		failedToSendItemsKey = FailedToSendSpansKey
	case configmodels.MetricsDataType:
		sentMeasure = mExporterSentMetricPoints
		failedToSendMeasure = mExporterFailedToSendMetricPoints
		sentItemsKey = SentMetricPointsKey
		failedToSendItemsKey = FailedToSendMetricPointsKey
	}

	if useNew {
		stats.Record(
			exporterCtx,
			sentMeasure.M(int64(numSent)),
			failedToSendMeasure.M(int64(numFailedToSend)))
	}

	// End span according to errors.
	if span.IsRecordingEvents() {
		span.AddAttributes(
			trace.Int64Attribute(
				sentItemsKey, int64(numSent)),
			trace.Int64Attribute(
				failedToSendItemsKey, int64(numFailedToSend)),
		)
		span.SetStatus(errToStatus(err))
	}
	span.End()
}
