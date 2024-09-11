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
	// FailedToSendSpansKey used to track spans that failed to be sent by exporters.
	FailedToSendSpansKey = "send_failed_spans"

	// SentMetricPointsKey used to track metric points sent by exporters.
	SentMetricPointsKey = "sent_metric_points"
	// FailedToSendMetricPointsKey used to track metric points that failed to be sent by exporters.
	FailedToSendMetricPointsKey = "send_failed_metric_points"

	// SentLogRecordsKey used to track logs sent by exporters.
	SentLogRecordsKey = "sent_log_records"
	// FailedToSendLogRecordsKey used to track logs that failed to be sent by exporters.
	FailedToSendLogRecordsKey = "send_failed_log_records"

	ExporterPrefix                 = ExporterKey + spanNameSep
	ExportTraceDataOperationSuffix = spanNameSep + "traces"
	ExportMetricsOperationSuffix   = spanNameSep + "metrics"
	ExportLogsOperationSuffix      = spanNameSep + "logs"
)
