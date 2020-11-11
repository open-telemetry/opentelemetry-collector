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

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/obsreport"
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
	views := obsreport.Configure(configtelemetry.LevelNormal)
	err := view.Register(views...)
	if err != nil {
		return nil, err
	}

	return func() {
		view.Unregister(views...)
	}, err
}

// CheckExporterTracesViews checks that for the current exported values for trace exporter views match given values.
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckExporterTracesViews(t *testing.T, exporter string, acceptedSpans, droppedSpans int64) {
	exporterTags := tagsForExporterView(exporter)
	CheckValueForView(t, exporterTags, acceptedSpans, "exporter/sent_spans")
	CheckValueForView(t, exporterTags, droppedSpans, "exporter/send_failed_spans")
}

// CheckExporterMetricsViews checks that for the current exported values for metrics exporter views match given values.
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckExporterMetricsViews(t *testing.T, exporter string, acceptedMetricsPoints, droppedMetricsPoints int64) {
	exporterTags := tagsForExporterView(exporter)
	CheckValueForView(t, exporterTags, acceptedMetricsPoints, "exporter/sent_metric_points")
	CheckValueForView(t, exporterTags, droppedMetricsPoints, "exporter/send_failed_metric_points")
}

// CheckExporterLogsViews checks that for the current exported values for logs exporter views match given values.
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckExporterLogsViews(t *testing.T, exporter string, acceptedLogRecords, droppedLogRecords int64) {
	exporterTags := tagsForExporterView(exporter)
	CheckValueForView(t, exporterTags, acceptedLogRecords, "exporter/sent_log_records")
	CheckValueForView(t, exporterTags, droppedLogRecords, "exporter/send_failed_log_records")
}

// CheckProcessorTracesViews checks that for the current exported values for trace exporter views match given values.
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckProcessorTracesViews(t *testing.T, processor string, acceptedSpans, refusedSpans, droppedSpans int64) {
	processorTags := tagsForProcessorView(processor)
	CheckValueForView(t, processorTags, acceptedSpans, "processor/accepted_spans")
	CheckValueForView(t, processorTags, refusedSpans, "processor/refused_spans")
	CheckValueForView(t, processorTags, droppedSpans, "processor/dropped_spans")
}

// CheckProcessorMetricsViews checks that for the current exported values for metrics exporter views match given values.
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckProcessorMetricsViews(t *testing.T, processor string, acceptedMetricPoints, refusedMetricPoints, droppedMetricPoints int64) {
	processorTags := tagsForProcessorView(processor)
	CheckValueForView(t, processorTags, acceptedMetricPoints, "processor/accepted_metric_points")
	CheckValueForView(t, processorTags, refusedMetricPoints, "processor/refused_metric_points")
	CheckValueForView(t, processorTags, droppedMetricPoints, "processor/dropped_metric_points")
}

// CheckProcessorLogsViews checks that for the current exported values for logs exporter views match given values.
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckProcessorLogsViews(t *testing.T, processor string, acceptedLogRecords, refusedLogRecords, droppedLogRecords int64) {
	processorTags := tagsForProcessorView(processor)
	CheckValueForView(t, processorTags, acceptedLogRecords, "processor/accepted_log_records")
	CheckValueForView(t, processorTags, refusedLogRecords, "processor/refused_log_records")
	CheckValueForView(t, processorTags, droppedLogRecords, "processor/dropped_log_records")
}

// CheckReceiverTracesViews checks that for the current exported values for trace receiver views match given values.
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckReceiverTracesViews(t *testing.T, receiver, protocol string, acceptedSpans, droppedSpans int64) {
	receiverTags := tagsForReceiverView(receiver, protocol)
	CheckValueForView(t, receiverTags, acceptedSpans, "receiver/accepted_spans")
	CheckValueForView(t, receiverTags, droppedSpans, "receiver/refused_spans")
}

// CheckReceiverLogsViews checks that for the current exported values for logs receiver views match given values.
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckReceiverLogsViews(t *testing.T, receiver, protocol string, acceptedLogRecords, droppedLogRecords int64) {
	receiverTags := tagsForReceiverView(receiver, protocol)
	CheckValueForView(t, receiverTags, acceptedLogRecords, "receiver/accepted_log_records")
	CheckValueForView(t, receiverTags, droppedLogRecords, "receiver/refused_log_records")
}

// CheckReceiverMetricsViews checks that for the current exported values for metrics receiver views match given values.
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckReceiverMetricsViews(t *testing.T, receiver, protocol string, acceptedMetricPoints, droppedMetricPoints int64) {
	receiverTags := tagsForReceiverView(receiver, protocol)
	CheckValueForView(t, receiverTags, acceptedMetricPoints, "receiver/accepted_metric_points")
	CheckValueForView(t, receiverTags, droppedMetricPoints, "receiver/refused_metric_points")
}

// CheckScraperMetricsViews checks that for the current exported values for metrics scraper views match given values.
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckScraperMetricsViews(t *testing.T, receiver, scraper string, scrapedMetricPoints, erroredMetricPoints int64) {
	scraperTags := tagsForScraperView(receiver, scraper)
	CheckValueForView(t, scraperTags, scrapedMetricPoints, "scraper/scraped_metric_points")
	CheckValueForView(t, scraperTags, erroredMetricPoints, "scraper/errored_metric_points")
}

// CheckValueForView checks that for the current exported value in the view with the given name
// for {LegacyTagKeyReceiver: receiverName} is equal to "value".
func CheckValueForView(t *testing.T, wantTags []tag.Tag, value int64, vName string) {
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
func tagsForReceiverView(receiver, transport string) []tag.Tag {
	tags := make([]tag.Tag, 0, 2)

	tags = append(tags, tag.Tag{Key: receiverTag, Value: receiver})
	if transport != "" {
		tags = append(tags, tag.Tag{Key: transportTag, Value: transport})
	}

	return tags
}

// tagsForScraperView returns the tags that are needed for the scraper views.
func tagsForScraperView(receiver, scraper string) []tag.Tag {
	tags := make([]tag.Tag, 0, 2)

	tags = append(tags, tag.Tag{Key: receiverTag, Value: receiver})
	if scraper != "" {
		tags = append(tags, tag.Tag{Key: scraperTag, Value: scraper})
	}

	return tags
}

// tagsForProcessorView returns the tags that are needed for the processor views.
func tagsForProcessorView(processor string) []tag.Tag {
	return []tag.Tag{
		{Key: processorTag, Value: processor},
	}
}

// tagsForExporterView returns the tags that are needed for the exporter views.
func tagsForExporterView(exporter string) []tag.Tag {
	return []tag.Tag{
		{Key: exporterTag, Value: exporter},
	}
}

func sortTags(tags []tag.Tag) {
	sort.SliceStable(tags, func(i, j int) bool {
		return tags[i].Key.Name() < tags[j].Key.Name()
	})
}
