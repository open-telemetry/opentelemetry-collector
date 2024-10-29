// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

const (
	// spanNameSep is duplicate between receiver and exporter.
	spanNameSep = "/"

	// ExporterKey used to identify exporters in metrics and traces.
	ExporterKey = "exporter"

	// DataTypeKey used to identify the data type in the queue size metric.
	DataTypeKey = "data_type"

	// SentSpansKey used to track spans sent by exporters.
	SentSpansKey = "sent_spans"
	// SentSpansBytesKey used to track spans bytes sent by exporters.
	SentSpansBytesKey = "sent_spans_bytes"
	// FailedToSendSpansKey used to track spans that failed to be sent by exporters.
	FailedToSendSpansKey = "send_failed_spans"
	// SentLogRecordsBytesKey used to track logs bytes sent by exporters.
	SentLogRecordsBytesKey = "sent_log_records_bytes"
	// SentMetricPointsKey used to track metric points sent by exporters.
	SentMetricPointsKey = "sent_metric_points"
	// SentMetricPointsBytesKey used to track metric points bytes sent by exporters.
	SentMetricPointsBytesKey = "sent_metric_points_bytes"
	// FailedToSendMetricPointsKey used to track metric points that failed to be sent by exporters.
	FailedToSendMetricPointsKey = "send_failed_metric_points"

	// SentLogRecordsKey used to track logs sent by exporters.
	SentLogRecordsKey = "sent_log_records"
	// FailedToSendLogRecordsKey used to track logs that failed to be sent by exporters.
	FailedToSendLogRecordsKey = "send_failed_log_records"

	// SentSamplesKey used to track profiles samples sent by exporters.
	SentSamplesKey = "sent_samples"
	// SentSamplesBytesKey used to track profiles samples bytes sent by exporters.
	SentSamplesBytesKey = "sent_samples_bytes"
	// FailedToSendSamplesKey used to track samples that failed to be sent by exporters.
	FailedToSendSamplesKey = "send_failed_samples"

	ExporterPrefix                 = ExporterKey + spanNameSep
	ExportTraceDataOperationSuffix = spanNameSep + "traces"
	ExportMetricsOperationSuffix   = spanNameSep + "metrics"
	ExportLogsOperationSuffix      = spanNameSep + "logs"
)
