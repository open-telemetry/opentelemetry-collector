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

package exporterhelper

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/metric"
	"go.opencensus.io/tag"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
)

func TestExportEnqueueFailure(t *testing.T) {
	exporter := component.NewID("fakeExporter")
	tt, err := obsreporttest.SetupTelemetry(exporter)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	insts := newInstruments(metric.NewRegistry())
	obsrep, err := newObsExporter(obsreport.ExporterSettings{
		ExporterID:             exporter,
		ExporterCreateSettings: tt.ToExporterCreateSettings(),
	}, insts)
	require.NoError(t, err)

	logRecords := int64(7)
	obsrep.recordLogsEnqueueFailure(context.Background(), logRecords)
	checkExporterEnqueueFailedLogsStats(t, insts, exporter, logRecords)

	spans := int64(12)
	obsrep.recordTracesEnqueueFailure(context.Background(), spans)
	checkExporterEnqueueFailedTracesStats(t, insts, exporter, spans)

	metricPoints := int64(21)
	obsrep.recordMetricsEnqueueFailure(context.Background(), metricPoints)
	checkExporterEnqueueFailedMetricsStats(t, insts, exporter, metricPoints)

	logRecords = int64(1)
	obsrep.recordLogsDropped(context.Background(), logRecords)
	checkExporterDroppedLogsStats(t, insts, exporter, logRecords)

	spans = int64(2)
	obsrep.recordTracesDropped(context.Background(), spans)
	checkExporterDroppedTracesStats(t, insts, exporter, spans)

	metricPoints = int64(3)
	obsrep.recordMetricsDropped(context.Background(), metricPoints)
	checkExporterDroppedMetricsStats(t, insts, exporter, metricPoints)
}

// checkExporterEnqueueFailedTracesStats checks that reported number of spans failed to enqueue match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func checkExporterEnqueueFailedTracesStats(t *testing.T, insts *instruments, exporter component.ID, spans int64) {
	checkStats(t, insts, exporter, spans, "exporter/enqueue_failed_spans")
}

// checkExporterEnqueueFailedMetricsStats checks that reported number of metric points failed to enqueue match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func checkExporterEnqueueFailedMetricsStats(t *testing.T, insts *instruments, exporter component.ID, metricPoints int64) {
	checkStats(t, insts, exporter, metricPoints, "exporter/enqueue_failed_metric_points")
}

// checkExporterEnqueueFailedLogsStats checks that reported number of log records failed to enqueue match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func checkExporterEnqueueFailedLogsStats(t *testing.T, insts *instruments, exporter component.ID, logRecords int64) {
	checkStats(t, insts, exporter, logRecords, "exporter/enqueue_failed_log_records")
}

// checkExporterDroppedTracesStats checks that reported number of spans dropped match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func checkExporterDroppedTracesStats(t *testing.T, insts *instruments, exporter component.ID, spans int64) {
	checkStats(t, insts, exporter, spans, "exporter/dropped_spans")
}

// checkExporterEnqueueFailedMetricsStats checks that reported number of metric points dropped match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func checkExporterDroppedMetricsStats(t *testing.T, insts *instruments, exporter component.ID, metricPoints int64) {
	checkStats(t, insts, exporter, metricPoints, "exporter/dropped_metric_points")
}

// checkExporterEnqueueFailedLogsStats checks that reported number of log records dropped match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func checkExporterDroppedLogsStats(t *testing.T, insts *instruments, exporter component.ID, logRecords int64) {
	checkStats(t, insts, exporter, logRecords, "exporter/dropped_log_records")
}

func checkStats(t *testing.T, insts *instruments, exporter component.ID, value int64, vName string) {
	assert.Eventually(t, func() bool {
		return checkValueForProducer(t, insts.registry, tagsForExporterView(exporter), value, vName)
	}, time.Second, time.Millisecond)
}

// tagsForExporterView returns the tags that are needed for the exporter views.
func tagsForExporterView(exporter component.ID) []tag.Tag {
	return []tag.Tag{
		{Key: exporterTag, Value: exporter.String()},
	}
}
