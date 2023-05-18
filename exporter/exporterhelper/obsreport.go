// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"

	"go.opencensus.io/metric"
	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/metric/metricproducer"

	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
	"go.opentelemetry.io/collector/obsreport"
)

// TODO: Incorporate this functionality along with tests from obsreport_test.go
//       into existing `obsreport` package once its functionally is not exposed
//       as public API. For now this part is kept private.

var (
	globalInstruments = newInstruments(metric.NewRegistry())
)

func init() {
	metricproducer.GlobalManager().AddProducer(globalInstruments.registry)
}

type instruments struct {
	registry                    *metric.Registry
	queueSize                   *metric.Int64DerivedGauge
	queueCapacity               *metric.Int64DerivedGauge
	failedToEnqueueTraceSpans   *metric.Int64Cumulative
	failedToEnqueueMetricPoints *metric.Int64Cumulative
	failedToEnqueueLogRecords   *metric.Int64Cumulative
}

func newInstruments(registry *metric.Registry) *instruments {
	insts := &instruments{
		registry: registry,
	}
	insts.queueSize, _ = registry.AddInt64DerivedGauge(
		obsmetrics.ExporterKey+"/queue_size",
		metric.WithDescription("Current size of the retry queue (in batches)"),
		metric.WithLabelKeys(obsmetrics.ExporterKey),
		metric.WithUnit(metricdata.UnitDimensionless))

	insts.queueCapacity, _ = registry.AddInt64DerivedGauge(
		obsmetrics.ExporterKey+"/queue_capacity",
		metric.WithDescription("Fixed capacity of the retry queue (in batches)"),
		metric.WithLabelKeys(obsmetrics.ExporterKey),
		metric.WithUnit(metricdata.UnitDimensionless))

	insts.failedToEnqueueTraceSpans, _ = registry.AddInt64Cumulative(
		obsmetrics.ExporterKey+"/enqueue_failed_spans",
		metric.WithDescription("Number of spans failed to be added to the sending queue."),
		metric.WithLabelKeys(obsmetrics.ExporterKey),
		metric.WithUnit(metricdata.UnitDimensionless))

	insts.failedToEnqueueMetricPoints, _ = registry.AddInt64Cumulative(
		obsmetrics.ExporterKey+"/enqueue_failed_metric_points",
		metric.WithDescription("Number of metric points failed to be added to the sending queue."),
		metric.WithLabelKeys(obsmetrics.ExporterKey),
		metric.WithUnit(metricdata.UnitDimensionless))

	insts.failedToEnqueueLogRecords, _ = registry.AddInt64Cumulative(
		obsmetrics.ExporterKey+"/enqueue_failed_log_records",
		metric.WithDescription("Number of log records failed to be added to the sending queue."),
		metric.WithLabelKeys(obsmetrics.ExporterKey),
		metric.WithUnit(metricdata.UnitDimensionless))

	return insts
}

// obsExporter is a helper to add observability to an exporter.
type obsExporter struct {
	*obsreport.Exporter
	failedToEnqueueTraceSpansEntry   *metric.Int64CumulativeEntry
	failedToEnqueueMetricPointsEntry *metric.Int64CumulativeEntry
	failedToEnqueueLogRecordsEntry   *metric.Int64CumulativeEntry
}

// newObsExporter creates a new observability exporter.
func newObsExporter(cfg obsreport.ExporterSettings, insts *instruments) (*obsExporter, error) {
	labelValue := metricdata.NewLabelValue(cfg.ExporterID.String())
	failedToEnqueueTraceSpansEntry, _ := insts.failedToEnqueueTraceSpans.GetEntry(labelValue)
	failedToEnqueueMetricPointsEntry, _ := insts.failedToEnqueueMetricPoints.GetEntry(labelValue)
	failedToEnqueueLogRecordsEntry, _ := insts.failedToEnqueueLogRecords.GetEntry(labelValue)

	exp, err := obsreport.NewExporter(cfg)
	if err != nil {
		return nil, err
	}

	return &obsExporter{
		Exporter:                         exp,
		failedToEnqueueTraceSpansEntry:   failedToEnqueueTraceSpansEntry,
		failedToEnqueueMetricPointsEntry: failedToEnqueueMetricPointsEntry,
		failedToEnqueueLogRecordsEntry:   failedToEnqueueLogRecordsEntry,
	}, nil
}

// recordTracesEnqueueFailure records number of spans that failed to be added to the sending queue.
func (eor *obsExporter) recordTracesEnqueueFailure(_ context.Context, numSpans int64) {
	eor.failedToEnqueueTraceSpansEntry.Inc(numSpans)
}

// recordMetricsEnqueueFailure records number of metric points that failed to be added to the sending queue.
func (eor *obsExporter) recordMetricsEnqueueFailure(_ context.Context, numMetricPoints int64) {
	eor.failedToEnqueueMetricPointsEntry.Inc(numMetricPoints)
}

// recordLogsEnqueueFailure records number of log records that failed to be added to the sending queue.
func (eor *obsExporter) recordLogsEnqueueFailure(_ context.Context, numLogRecords int64) {
	eor.failedToEnqueueLogRecordsEntry.Inc(numLogRecords)
}
