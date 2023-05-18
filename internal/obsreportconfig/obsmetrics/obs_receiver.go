// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsmetrics // import "go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

const (
	// ReceiverKey used to identify receivers in metrics and traces.
	ReceiverKey = "receiver"
	// TransportKey used to identify the transport used to received the data.
	TransportKey = "transport"
	// FormatKey used to identify the format of the data received.
	FormatKey = "format"

	// AcceptedSpansKey used to identify spans accepted by the Collector.
	AcceptedSpansKey = "accepted_spans"
	// RefusedSpansKey used to identify spans refused (ie.: not ingested) by the Collector.
	RefusedSpansKey = "refused_spans"

	// AcceptedMetricPointsKey used to identify metric points accepted by the Collector.
	AcceptedMetricPointsKey = "accepted_metric_points"
	// RefusedMetricPointsKey used to identify metric points refused (ie.: not ingested) by the
	// Collector.
	RefusedMetricPointsKey = "refused_metric_points"

	// AcceptedLogRecordsKey used to identify log records accepted by the Collector.
	AcceptedLogRecordsKey = "accepted_log_records"
	// RefusedLogRecordsKey used to identify log records refused (ie.: not ingested) by the
	// Collector.
	RefusedLogRecordsKey = "refused_log_records"
)

var (
	TagKeyReceiver, _  = tag.NewKey(ReceiverKey)
	TagKeyTransport, _ = tag.NewKey(TransportKey)

	ReceiverPrefix                  = ReceiverKey + NameSep
	ReceiveTraceDataOperationSuffix = NameSep + "TraceDataReceived"
	ReceiverMetricsOperationSuffix  = NameSep + "MetricsReceived"
	ReceiverLogsOperationSuffix     = NameSep + "LogsReceived"

	// Receiver metrics. Any count of data items below is in the original format
	// that they were received, reasoning: reconciliation is easier if measurement
	// on clients and receiver are expected to be the same. Translation issues
	// that result in a different number of elements should be reported in a
	// separate way.
	ReceiverAcceptedSpans = stats.Int64(
		ReceiverPrefix+AcceptedSpansKey,
		"Number of spans successfully pushed into the pipeline.",
		stats.UnitDimensionless)
	ReceiverRefusedSpans = stats.Int64(
		ReceiverPrefix+RefusedSpansKey,
		"Number of spans that could not be pushed into the pipeline.",
		stats.UnitDimensionless)
	ReceiverAcceptedMetricPoints = stats.Int64(
		ReceiverPrefix+AcceptedMetricPointsKey,
		"Number of metric points successfully pushed into the pipeline.",
		stats.UnitDimensionless)
	ReceiverRefusedMetricPoints = stats.Int64(
		ReceiverPrefix+RefusedMetricPointsKey,
		"Number of metric points that could not be pushed into the pipeline.",
		stats.UnitDimensionless)
	ReceiverAcceptedLogRecords = stats.Int64(
		ReceiverPrefix+AcceptedLogRecordsKey,
		"Number of log records successfully pushed into the pipeline.",
		stats.UnitDimensionless)
	ReceiverRefusedLogRecords = stats.Int64(
		ReceiverPrefix+RefusedLogRecordsKey,
		"Number of log records that could not be pushed into the pipeline.",
		stats.UnitDimensionless)
)
