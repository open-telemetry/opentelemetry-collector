// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/receiver/internal"

const (
	// SpanNameSep is duplicate between receiver and exporter.
	SpanNameSep = "/"

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

	// ScraperKey used to identify scrapers in metrics and traces.
	ScraperKey = "scraper"

	// ScrapedMetricPointsKey used to identify metric points scraped by the
	// Collector.
	ScrapedMetricPointsKey = "scraped_metric_points"
	// ErroredMetricPointsKey used to identify metric points errored (i.e.
	// unable to be scraped) by the Collector.
	ErroredMetricPointsKey        = "errored_metric_points"
	ScraperPrefix                 = ScraperKey + SpanNameSep
	ScraperMetricsOperationSuffix = SpanNameSep + "MetricsScraped"

	ReceiverPrefix                  = ReceiverKey + SpanNameSep
	ReceiveTraceDataOperationSuffix = SpanNameSep + "TraceDataReceived"
	ReceiverMetricsOperationSuffix  = SpanNameSep + "MetricsReceived"
	ReceiverLogsOperationSuffix     = SpanNameSep + "LogsReceived"
)
