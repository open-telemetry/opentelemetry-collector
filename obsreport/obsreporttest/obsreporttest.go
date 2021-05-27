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

package obsreporttest

import (
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
)

var (
	// Names used by the metrics and labels are hard coded here in order to avoid
	// inadvertent changes: at this point changing metric names and labels should
	// be treated as a breaking changing and requires a good justification.
	// Changes to metric names or labels can break alerting, dashboards, etc
	// that are used to monitor the Collector in production deployments.
	// DO NOT SWITCH THE VARIABLES BELOW TO SIMILAR ONES DEFINED ON THE PACKAGE.
	receiverTag, _  = tag.NewKey("receiver")
	scraperTag, _   = tag.NewKey("scraper")
	transportTag, _ = tag.NewKey("transport")
	exporterTag, _  = tag.NewKey("exporter")
	processorTag, _ = tag.NewKey("processor")
)

// SetupRecordedMetricsTest does setup the testing environment to check the metrics recorded by receivers, producers or exporters.
// The returned function should be deferred.
func SetupRecordedMetricsTest() (func(), error) {
	obsMetrics := obsreportconfig.Configure(configtelemetry.LevelNormal)
	views := obsMetrics.Views
	err := view.Register(views...)
	if err != nil {
		return nil, err
	}

	return func() {
		view.Unregister(views...)
	}, err
}

// CheckExporterTraces checks that for the current exported values for trace exporter metrics match given values.
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckExporterTraces(t *testing.T, exporter config.ComponentID, acceptedSpans, droppedSpans int64) {
	exporterTags := tagsForExporterView(exporter)
	checkValueForView(t, exporterTags, acceptedSpans, "exporter/sent_spans")
	checkValueForView(t, exporterTags, droppedSpans, "exporter/send_failed_spans")
}

// CheckExporterMetrics checks that for the current exported values for metrics exporter metrics match given values.
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckExporterMetrics(t *testing.T, exporter config.ComponentID, acceptedMetricsPoints, droppedMetricsPoints int64) {
	exporterTags := tagsForExporterView(exporter)
	checkValueForView(t, exporterTags, acceptedMetricsPoints, "exporter/sent_metric_points")
	checkValueForView(t, exporterTags, droppedMetricsPoints, "exporter/send_failed_metric_points")
}

// CheckExporterLogs checks that for the current exported values for logs exporter metrics match given values.
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckExporterLogs(t *testing.T, exporter config.ComponentID, acceptedLogRecords, droppedLogRecords int64) {
	exporterTags := tagsForExporterView(exporter)
	checkValueForView(t, exporterTags, acceptedLogRecords, "exporter/sent_log_records")
	checkValueForView(t, exporterTags, droppedLogRecords, "exporter/send_failed_log_records")
}

// CheckProcessorTraces checks that for the current exported values for trace exporter metrics match given values.
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckProcessorTraces(t *testing.T, processor config.ComponentID, acceptedSpans, refusedSpans, droppedSpans int64) {
	processorTags := tagsForProcessorView(processor)
	checkValueForView(t, processorTags, acceptedSpans, "processor/accepted_spans")
	checkValueForView(t, processorTags, refusedSpans, "processor/refused_spans")
	checkValueForView(t, processorTags, droppedSpans, "processor/dropped_spans")
}

// CheckProcessorMetrics checks that for the current exported values for metrics exporter metrics match given values.
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckProcessorMetrics(t *testing.T, processor config.ComponentID, acceptedMetricPoints, refusedMetricPoints, droppedMetricPoints int64) {
	processorTags := tagsForProcessorView(processor)
	checkValueForView(t, processorTags, acceptedMetricPoints, "processor/accepted_metric_points")
	checkValueForView(t, processorTags, refusedMetricPoints, "processor/refused_metric_points")
	checkValueForView(t, processorTags, droppedMetricPoints, "processor/dropped_metric_points")
}

// CheckProcessorLogs checks that for the current exported values for logs exporter metrics match given values.
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckProcessorLogs(t *testing.T, processor config.ComponentID, acceptedLogRecords, refusedLogRecords, droppedLogRecords int64) {
	processorTags := tagsForProcessorView(processor)
	checkValueForView(t, processorTags, acceptedLogRecords, "processor/accepted_log_records")
	checkValueForView(t, processorTags, refusedLogRecords, "processor/refused_log_records")
	checkValueForView(t, processorTags, droppedLogRecords, "processor/dropped_log_records")
}

// CheckReceiverTraces checks that for the current exported values for trace receiver metrics match given values.
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckReceiverTraces(t *testing.T, receiver config.ComponentID, protocol string, acceptedSpans, droppedSpans int64) {
	receiverTags := tagsForReceiverView(receiver, protocol)
	checkValueForView(t, receiverTags, acceptedSpans, "receiver/accepted_spans")
	checkValueForView(t, receiverTags, droppedSpans, "receiver/refused_spans")
}

// CheckReceiverLogs checks that for the current exported values for logs receiver metrics match given values.
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckReceiverLogs(t *testing.T, receiver config.ComponentID, protocol string, acceptedLogRecords, droppedLogRecords int64) {
	receiverTags := tagsForReceiverView(receiver, protocol)
	checkValueForView(t, receiverTags, acceptedLogRecords, "receiver/accepted_log_records")
	checkValueForView(t, receiverTags, droppedLogRecords, "receiver/refused_log_records")
}

// CheckReceiverMetrics checks that for the current exported values for metrics receiver metrics match given values.
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckReceiverMetrics(t *testing.T, receiver config.ComponentID, protocol string, acceptedMetricPoints, droppedMetricPoints int64) {
	receiverTags := tagsForReceiverView(receiver, protocol)
	checkValueForView(t, receiverTags, acceptedMetricPoints, "receiver/accepted_metric_points")
	checkValueForView(t, receiverTags, droppedMetricPoints, "receiver/refused_metric_points")
}

// CheckScraperMetrics checks that for the current exported values for metrics scraper metrics match given values.
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckScraperMetrics(t *testing.T, receiver config.ComponentID, scraper config.ComponentID, scrapedMetricPoints, erroredMetricPoints int64) {
	scraperTags := tagsForScraperView(receiver, scraper)
	checkValueForView(t, scraperTags, scrapedMetricPoints, "scraper/scraped_metric_points")
	checkValueForView(t, scraperTags, erroredMetricPoints, "scraper/errored_metric_points")
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

// tagsForReceiverView returns the tags that are needed for the receiver views.
func tagsForReceiverView(receiver config.ComponentID, transport string) []tag.Tag {
	tags := make([]tag.Tag, 0, 2)

	tags = append(tags, tag.Tag{Key: receiverTag, Value: receiver.String()})
	if transport != "" {
		tags = append(tags, tag.Tag{Key: transportTag, Value: transport})
	}

	return tags
}

// tagsForScraperView returns the tags that are needed for the scraper views.
func tagsForScraperView(receiver config.ComponentID, scraper config.ComponentID) []tag.Tag {
	return []tag.Tag{
		{Key: receiverTag, Value: receiver.String()},
		{Key: scraperTag, Value: scraper.String()},
	}
}

// tagsForProcessorView returns the tags that are needed for the processor views.
func tagsForProcessorView(processor config.ComponentID) []tag.Tag {
	return []tag.Tag{
		{Key: processorTag, Value: processor.String()},
	}
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
