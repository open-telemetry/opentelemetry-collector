// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsmetrics // import "go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

const (
	// ExporterKey used to identify exporters in metrics and traces.
	ExporterKey = "exporter"

	// SentSpansKey used to track spans sent by exporters.
	SentSpansKey = "sent_spans"
	// FailedToSendSpansKey used to track spans that failed to be sent by exporters.
	FailedToSendSpansKey = "send_failed_spans"
	// FailedToEnqueueSpansKey used to track spans that failed to be enqueued by exporters.
	FailedToEnqueueSpansKey = "enqueue_failed_spans"

	// SentMetricPointsKey used to track metric points sent by exporters.
	SentMetricPointsKey = "sent_metric_points"
	// FailedToSendMetricPointsKey used to track metric points that failed to be sent by exporters.
	FailedToSendMetricPointsKey = "send_failed_metric_points"
	// FailedToEnqueueMetricPointsKey used to track metric points that failed to be enqueued by exporters.
	FailedToEnqueueMetricPointsKey = "enqueue_failed_metric_points"

	// SentLogRecordsKey used to track logs sent by exporters.
	SentLogRecordsKey = "sent_log_records"
	// FailedToSendLogRecordsKey used to track logs that failed to be sent by exporters.
	FailedToSendLogRecordsKey = "send_failed_log_records"
	// FailedToEnqueueLogRecordsKey used to track logs that failed to be enqueued by exporters.
	FailedToEnqueueLogRecordsKey = "enqueue_failed_log_records"
)

var (
	TagKeyExporter, _ = tag.NewKey(ExporterKey)

	ExporterPrefix                 = ExporterKey + NameSep
	ExportTraceDataOperationSuffix = NameSep + "traces"
	ExportMetricsOperationSuffix   = NameSep + "metrics"
	ExportLogsOperationSuffix      = NameSep + "logs"

	// Exporter metrics. Any count of data items below is in the final format
	// that they were sent, reasoning: reconciliation is easier if measurements
	// on backend and exporter are expected to be the same. Translation issues
	// that result in a different number of elements should be reported in a
	// separate way.
	ExporterSentSpans = stats.Int64(
		ExporterPrefix+SentSpansKey,
		"Number of spans successfully sent to destination.",
		stats.UnitDimensionless)
	ExporterFailedToSendSpans = stats.Int64(
		ExporterPrefix+FailedToSendSpansKey,
		"Number of spans in failed attempts to send to destination.",
		stats.UnitDimensionless)
	ExporterFailedToEnqueueSpans = stats.Int64(
		ExporterPrefix+FailedToEnqueueSpansKey,
		"Number of spans failed to be added to the sending queue.",
		stats.UnitDimensionless)
	ExporterSentMetricPoints = stats.Int64(
		ExporterPrefix+SentMetricPointsKey,
		"Number of metric points successfully sent to destination.",
		stats.UnitDimensionless)
	ExporterFailedToSendMetricPoints = stats.Int64(
		ExporterPrefix+FailedToSendMetricPointsKey,
		"Number of metric points in failed attempts to send to destination.",
		stats.UnitDimensionless)
	ExporterFailedToEnqueueMetricPoints = stats.Int64(
		ExporterPrefix+FailedToEnqueueMetricPointsKey,
		"Number of metric points failed to be added to the sending queue.",
		stats.UnitDimensionless)
	ExporterSentLogRecords = stats.Int64(
		ExporterPrefix+SentLogRecordsKey,
		"Number of log record successfully sent to destination.",
		stats.UnitDimensionless)
	ExporterFailedToSendLogRecords = stats.Int64(
		ExporterPrefix+FailedToSendLogRecordsKey,
		"Number of log records in failed attempts to send to destination.",
		stats.UnitDimensionless)
	ExporterFailedToEnqueueLogRecords = stats.Int64(
		ExporterPrefix+FailedToEnqueueLogRecordsKey,
		"Number of log records failed to be added to the sending queue.",
		stats.UnitDimensionless)
)
