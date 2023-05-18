// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsmetrics // import "go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

const (
	// ProcessorKey is the key used to identify processors in metrics and traces.
	ProcessorKey = "processor"

	// DroppedSpansKey is the key used to identify spans dropped by the Collector.
	DroppedSpansKey = "dropped_spans"

	// DroppedMetricPointsKey is the key used to identify metric points dropped by the Collector.
	DroppedMetricPointsKey = "dropped_metric_points"

	// DroppedLogRecordsKey is the key used to identify log records dropped by the Collector.
	DroppedLogRecordsKey = "dropped_log_records"
)

var (
	TagKeyProcessor, _ = tag.NewKey(ProcessorKey)

	ProcessorPrefix = ProcessorKey + NameSep

	// Processor metrics. Any count of data items below is in the internal format
	// of the collector since processors only deal with internal format.
	ProcessorAcceptedSpans = stats.Int64(
		ProcessorPrefix+AcceptedSpansKey,
		"Number of spans successfully pushed into the next component in the pipeline.",
		stats.UnitDimensionless)
	ProcessorRefusedSpans = stats.Int64(
		ProcessorPrefix+RefusedSpansKey,
		"Number of spans that were rejected by the next component in the pipeline.",
		stats.UnitDimensionless)
	ProcessorDroppedSpans = stats.Int64(
		ProcessorPrefix+DroppedSpansKey,
		"Number of spans that were dropped.",
		stats.UnitDimensionless)
	ProcessorAcceptedMetricPoints = stats.Int64(
		ProcessorPrefix+AcceptedMetricPointsKey,
		"Number of metric points successfully pushed into the next component in the pipeline.",
		stats.UnitDimensionless)
	ProcessorRefusedMetricPoints = stats.Int64(
		ProcessorPrefix+RefusedMetricPointsKey,
		"Number of metric points that were rejected by the next component in the pipeline.",
		stats.UnitDimensionless)
	ProcessorDroppedMetricPoints = stats.Int64(
		ProcessorPrefix+DroppedMetricPointsKey,
		"Number of metric points that were dropped.",
		stats.UnitDimensionless)
	ProcessorAcceptedLogRecords = stats.Int64(
		ProcessorPrefix+AcceptedLogRecordsKey,
		"Number of log records successfully pushed into the next component in the pipeline.",
		stats.UnitDimensionless)
	ProcessorRefusedLogRecords = stats.Int64(
		ProcessorPrefix+RefusedLogRecordsKey,
		"Number of log records that were rejected by the next component in the pipeline.",
		stats.UnitDimensionless)
	ProcessorDroppedLogRecords = stats.Int64(
		ProcessorPrefix+DroppedLogRecordsKey,
		"Number of log records that were dropped.",
		stats.UnitDimensionless)
)
