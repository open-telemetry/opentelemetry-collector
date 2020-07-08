// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package obsreporttest

import (
	"fmt"
	"reflect"
	"sort"

	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"go.opentelemetry.io/collector/obsreport"
)

// SetupRecordedMetricsTest does setup the testing environment to check the metrics recorded by receivers, producers or exporters.
// The returned function should be deferred.
func SetupRecordedMetricsTest() (doneFn func()) {
	// Register views
	view.Register(obsreport.LegacyAllViews...)

	return func() {
		view.Unregister(obsreport.LegacyAllViews...)
	}
}

// CheckValueViewExporterReceivedSpans checks that for the current exported value in the LegacyViewExporterReceivedSpans
// for {TagKeyReceiver: receiverName, TagKeyExporter: exporterTagName} is equal to "value".
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckValueViewExporterReceivedSpans(receiverName string, exporterTagName string, value int) error {
	return checkValueForView(obsreport.LegacyViewExporterReceivedSpans.Name,
		wantsTagsForExporterView(receiverName, exporterTagName), int64(value))
}

// CheckValueViewExporterDroppedSpans checks that for the current exported value in the LegacyViewExporterDroppedSpans
// for {TagKeyReceiver: receiverName} is equal to "value".
// In tests that this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckValueViewExporterDroppedSpans(receiverName string, exporterTagName string, value int) error {
	return checkValueForView(obsreport.LegacyViewExporterDroppedSpans.Name,
		wantsTagsForExporterView(receiverName, exporterTagName), int64(value))
}

// CheckValueViewExporterReceivedTimeSeries checks that for the current exported value in the LegacyViewExporterReceivedTimeSeries
// for {TagKeyReceiver: receiverName, TagKeyExporter: exporterTagName} is equal to "value".
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckValueViewExporterReceivedTimeSeries(receiverName string, exporterTagName string, value int) error {
	return checkValueForView(obsreport.LegacyViewExporterReceivedTimeSeries.Name,
		wantsTagsForExporterView(receiverName, exporterTagName), int64(value))
}

// CheckValueViewExporterDroppedTimeSeries checks that for the current exported value in the LegacyViewExporterDroppedTimeSeries
// for {TagKeyReceiver: receiverName} is equal to "value".
// In tests that this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckValueViewExporterDroppedTimeSeries(receiverName string, exporterTagName string, value int) error {
	return checkValueForView(obsreport.LegacyViewExporterDroppedTimeSeries.Name,
		wantsTagsForExporterView(receiverName, exporterTagName), int64(value))
}

// CheckValueViewExporterReceivedLogRecords checks that for the current exported value in the LegacyViewExporterReceivedLogRecords
// for {TagKeyReceiver: receiverName, TagKeyExporter: exporterTagName} is equal to "value".
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckValueViewExporterReceivedLogRecords(receiverName string, exporterTagName string, value int) error {
	return checkValueForView(obsreport.LegacyViewExporterReceivedLogRecords.Name,
		wantsTagsForExporterView(receiverName, exporterTagName), int64(value))
}

// CheckValueViewExporterDroppedLogRecords checks that for the current exported value in the LegacyViewExporterDroppedLogRecords
// for {TagKeyReceiver: receiverName} is equal to "value".
// In tests that this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckValueViewExporterDroppedLogRecords(receiverName string, exporterTagName string, value int) error {
	return checkValueForView(obsreport.LegacyViewExporterDroppedLogRecords.Name,
		wantsTagsForExporterView(receiverName, exporterTagName), int64(value))
}

// CheckValueViewReceiverReceivedSpans checks that for the current exported value in the LegacyViewReceiverReceivedSpans
// for {TagKeyReceiver: receiverName, TagKeyExporter: exporterTagName} is equal to "value".
// In tests that this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckValueViewReceiverReceivedSpans(receiverName string, value int) error {
	return checkValueForView(obsreport.LegacyViewReceiverReceivedSpans.Name,
		wantsTagsForReceiverView(receiverName), int64(value))
}

// CheckValueViewReceiverDroppedSpans checks that for the current exported value in the LegacyViewReceiverDroppedSpans
// for {TagKeyReceiver: receiverName} is equal to "value".
// In tests that this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckValueViewReceiverDroppedSpans(receiverName string, value int) error {
	return checkValueForView(obsreport.LegacyViewReceiverDroppedSpans.Name,
		wantsTagsForReceiverView(receiverName), int64(value))
}

// CheckValueViewReceiverReceivedTimeSeries checks that for the current exported value in the LegacyViewReceiverReceivedTimeSeries
// for {TagKeyReceiver: receiverName, TagKeyExporter: exporterTagName} is equal to "value".
// In tests that this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckValueViewReceiverReceivedTimeSeries(receiverName string, value int) error {
	return checkValueForView(obsreport.LegacyViewReceiverReceivedTimeSeries.Name,
		wantsTagsForReceiverView(receiverName), int64(value))
}

// CheckValueViewReceiverDroppedTimeSeries checks that for the current exported value in the LegacyViewReceiverDroppedTimeSeries
// for {TagKeyReceiver: receiverName} is equal to "value".
// In tests that this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckValueViewReceiverDroppedTimeSeries(receiverName string, value int) error {
	return checkValueForView(obsreport.LegacyViewReceiverDroppedTimeSeries.Name,
		wantsTagsForReceiverView(receiverName), int64(value))
}

func checkValueForView(vName string, wantTags []tag.Tag, value int64) error {
	// Make sure the tags slice is sorted by tag keys.
	sortTags(wantTags)

	rows, err := view.RetrieveData(vName)
	if err != nil {
		return fmt.Errorf("error retrieving view data for view Name %s", vName)
	}

	for _, row := range rows {
		// Make sure the tags slice is sorted by tag keys.
		sortTags(row.Tags)
		if reflect.DeepEqual(wantTags, row.Tags) {
			sum := row.Data.(*view.SumData)
			if float64(value) != sum.Value {
				return fmt.Errorf("different recorded value: want %v got %v", float64(value), sum.Value)
			}
			// We found the result
			return nil
		}
	}
	return fmt.Errorf("could not find wantTags: %s in rows %v", wantTags, rows)
}

func wantsTagsForExporterView(receiverName string, exporterTagName string) []tag.Tag {
	return []tag.Tag{
		{Key: obsreport.TagKeyReceiver, Value: receiverName},
		{Key: obsreport.TagKeyExporter, Value: exporterTagName},
	}
}

func wantsTagsForReceiverView(receiverName string) []tag.Tag {
	return []tag.Tag{
		{Key: obsreport.TagKeyReceiver, Value: receiverName},
	}
}

func sortTags(tags []tag.Tag) {
	sort.SliceStable(tags, func(i, j int) bool {
		return tags[i].Key.Name() < tags[j].Key.Name()
	})
}
