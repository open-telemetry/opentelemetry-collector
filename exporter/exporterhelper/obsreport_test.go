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
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
)

func TestExportEnqueueFailure(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	exporter := config.NewID("fakeExporter")

	obsrep := newObsExporter(obsreport.ExporterSettings{
		Level:                  configtelemetry.LevelNormal,
		ExporterID:             exporter,
		ExporterCreateSettings: componenttest.NewNopExporterCreateSettings()})

	logRecords := 7
	obsrep.recordLogsEnqueueFailure(context.Background(), logRecords)
	checkExporterEnqueueFailedLogsStats(t, exporter, int64(logRecords))

	spans := 12
	obsrep.recordTracesEnqueueFailure(context.Background(), spans)
	checkExporterEnqueueFailedTracesStats(t, exporter, int64(spans))

	metricPoints := 21
	obsrep.recordMetricsEnqueueFailure(context.Background(), metricPoints)
	checkExporterEnqueueFailedMetricsStats(t, exporter, int64(metricPoints))
}

// checkExporterEnqueueFailedTracesStats checks that reported number of spans failed to enqueue match given values.
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func checkExporterEnqueueFailedTracesStats(t *testing.T, exporter config.ComponentID, spans int64) {
	exporterTags := tagsForExporterView(exporter)
	checkValueForView(t, exporterTags, spans, "exporter/enqueue_failed_spans")
}

// checkExporterEnqueueFailedMetricsStats checks that reported number of metric points failed to enqueue match given values.
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func checkExporterEnqueueFailedMetricsStats(t *testing.T, exporter config.ComponentID, metricPoints int64) {
	exporterTags := tagsForExporterView(exporter)
	checkValueForView(t, exporterTags, metricPoints, "exporter/enqueue_failed_metric_points")
}

// checkExporterEnqueueFailedLogsStats checks that reported number of log records failed to enqueue match given values.
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func checkExporterEnqueueFailedLogsStats(t *testing.T, exporter config.ComponentID, logRecords int64) {
	exporterTags := tagsForExporterView(exporter)
	checkValueForView(t, exporterTags, logRecords, "exporter/enqueue_failed_log_records")
}

// checkValueForView checks that for the current exported value in the view with the given name
// for {LegacyTagKeyReceiver: receiverName} is equal to "value".
func checkValueForView(t *testing.T, wantTags []tag.Tag, value int64, vName string) {
	// Make sure the tags slice is sorted by tag keys.
	sortTags(wantTags)

	rows, err := view.RetrieveData(vName)
	require.NoError(t, err)

	for _, row := range rows {
		// Make sure the tags slice is sorted by tag keys.
		sortTags(row.Tags)
		if reflect.DeepEqual(wantTags, row.Tags) {
			sum := row.Data.(*view.SumData)
			require.Equal(t, float64(value), sum.Value)
			return
		}
	}

	require.Failf(t, "could not find tags", "wantTags: %s in rows %v", wantTags, rows)
}

// tagsForExporterView returns the tags that are needed for the exporter views.
func tagsForExporterView(exporter config.ComponentID) []tag.Tag {
	return []tag.Tag{
		{Key: exporterTag, Value: exporter.String()},
	}
}

func sortTags(tags []tag.Tag) {
	sort.SliceStable(tags, func(i, j int) bool {
		return tags[i].Key.Name() < tags[j].Key.Name()
	})
}
