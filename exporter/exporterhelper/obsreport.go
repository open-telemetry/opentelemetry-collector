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
	droppedTraceSpans           *metric.Int64Cumulative
	droppedMetricPoints         *metric.Int64Cumulative
	droppedLogRecords           *metric.Int64Cumulative
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

	insts.droppedTraceSpans, _ = registry.AddInt64Cumulative(
		obsmetrics.ExporterKey+"/dropped_spans",
		metric.WithDescription("Number of spans dropped from the sending queue."),
		metric.WithLabelKeys(obsmetrics.ExporterKey),
		metric.WithUnit(metricdata.UnitDimensionless))

	insts.droppedMetricPoints, _ = registry.AddInt64Cumulative(
		obsmetrics.ExporterKey+"/dropped_metric_points",
		metric.WithDescription("Number of metric points dropped from the sending queue."),
		metric.WithLabelKeys(obsmetrics.ExporterKey),
		metric.WithUnit(metricdata.UnitDimensionless))

	insts.droppedLogRecords, _ = registry.AddInt64Cumulative(
		obsmetrics.ExporterKey+"/dropped_log_records",
		metric.WithDescription("Number of log records dropped from the sending queue."),
		metric.WithLabelKeys(obsmetrics.ExporterKey),
		metric.WithUnit(metricdata.UnitDimensionless))

	return insts
}

// obsExporter is a helper to add observability to a component.Exporter.
type obsExporter struct {
	*obsreport.Exporter
	failedToEnqueueTraceSpansEntry   *metric.Int64CumulativeEntry
	failedToEnqueueMetricPointsEntry *metric.Int64CumulativeEntry
	failedToEnqueueLogRecordsEntry   *metric.Int64CumulativeEntry
	droppedTraceSpansEntry           *metric.Int64CumulativeEntry
	droppedMetricPointsEntry         *metric.Int64CumulativeEntry
	droppedLogRecordsEntry           *metric.Int64CumulativeEntry
}

// newObsExporter creates a new observability exporter.
func newObsExporter(cfg obsreport.ExporterSettings, insts *instruments) (*obsExporter, error) {
	labelValue := metricdata.NewLabelValue(cfg.ExporterID.String())
	failedToEnqueueTraceSpansEntry, _ := insts.failedToEnqueueTraceSpans.GetEntry(labelValue)
	failedToEnqueueMetricPointsEntry, _ := insts.failedToEnqueueMetricPoints.GetEntry(labelValue)
	failedToEnqueueLogRecordsEntry, _ := insts.failedToEnqueueLogRecords.GetEntry(labelValue)
	droppedTraceSpansEntry, _ := insts.droppedTraceSpans.GetEntry(labelValue)
	droppedMetricPointsEntry, _ := insts.droppedMetricPoints.GetEntry(labelValue)
	droppedLogRecordsEntry, _ := insts.droppedLogRecords.GetEntry(labelValue)

	exp, err := obsreport.NewExporter(cfg)
	if err != nil {
		return nil, err
	}

	return &obsExporter{
		Exporter:                         exp,
		failedToEnqueueTraceSpansEntry:   failedToEnqueueTraceSpansEntry,
		failedToEnqueueMetricPointsEntry: failedToEnqueueMetricPointsEntry,
		failedToEnqueueLogRecordsEntry:   failedToEnqueueLogRecordsEntry,
		droppedTraceSpansEntry:           droppedTraceSpansEntry,
		droppedMetricPointsEntry:         droppedMetricPointsEntry,
		droppedLogRecordsEntry:           droppedLogRecordsEntry,
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

// recordTracesDropped records number of spans that were dropped by the sending queue.
func (eor *obsExporter) recordTracesDropped(_ context.Context, numSpans int64) {
	eor.droppedTraceSpansEntry.Inc(numSpans)
}

// recordMetricsDropped records number of metric points that were dropped by the sending queue.
func (eor *obsExporter) recordMetricsDropped(_ context.Context, numMetricPoints int64) {
	eor.droppedMetricPointsEntry.Inc(numMetricPoints)
}

// recordLogsDropped records number of log records that were dropped by the sending queue.
func (eor *obsExporter) recordLogsDropped(_ context.Context, numLogRecords int64) {
	eor.droppedLogRecordsEntry.Inc(numLogRecords)
}
