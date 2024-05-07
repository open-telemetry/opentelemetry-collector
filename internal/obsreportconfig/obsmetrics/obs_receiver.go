// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsmetrics // import "go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"

const (
	// ReceiverKey used to identify receivers in metrics and traces.
	ReceiverKey = "receiver"
	// TransportKey used to identify the transport used to received the data.
	TransportKey = "transport"
	// FormatKey used to identify the format of the data received.
	FormatKey = "format"

	// AcceptedSpansKey used to identify spans accepted by the Collector.
	AcceptedSpansKey = "accepted_spans"
	// AcceptedSpansBytesKey used to calculate spans in bytes accepted by the Collector.
	AcceptedSpansBytesKey = "accepted_spans_bytes"
	// RefusedSpansKey used to identify spans refused (ie.: not ingested) by the Collector.
	RefusedSpansKey = "refused_spans"

	// AcceptedMetricPointsKey used to identify metric points accepted by the Collector.
	AcceptedMetricPointsKey = "accepted_metric_points"
	// AcceptedMetricPointsBytesKey used to calculate metric points in bytes accepted by the Collector.
	AcceptedMetricPointsBytesKey = "accepted_metric_points_bytes"
	// RefusedMetricPointsKey used to identify metric points refused (ie.: not ingested) by the
	// Collector.
	RefusedMetricPointsKey = "refused_metric_points"

	// AcceptedLogRecordsKey used to identify log records accepted by the Collector.
	AcceptedLogRecordsKey = "accepted_log_records"
	// AcceptedLogRecordsBytesKey used to calculate log records in bytes accepted by the Collector.
	AcceptedLogRecordsBytesKey = "accepted_log_records_bytes"
	// RefusedLogRecordsKey used to identify log records refused (ie.: not ingested) by the
	// Collector.
	RefusedLogRecordsKey = "refused_log_records"
)

var (
	ReceiverPrefix                  = ReceiverKey + SpanNameSep
	ReceiverMetricPrefix            = ReceiverKey + MetricNameSep
	ReceiveTraceDataOperationSuffix = SpanNameSep + "TraceDataReceived"
	ReceiverMetricsOperationSuffix  = SpanNameSep + "MetricsReceived"
	ReceiverLogsOperationSuffix     = SpanNameSep + "LogsReceived"
)
