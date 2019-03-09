// Copyright 2018, OpenCensus Authors
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

package observabilitytest

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"time"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/exporter"
	"github.com/census-instrumentation/opencensus-service/internal"
	"github.com/census-instrumentation/opencensus-service/observability"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

const (
	fakeReceiverName = "fake_receiver_trace"
	fakeExporterName = "fake_exporter_trace"
)

type nopMetricsExporter int

var _ view.Exporter = (*nopMetricsExporter)(nil)

func (cme *nopMetricsExporter) ExportView(vd *view.Data) {}

// SetupRecordedMetricsTest does setup the testing environment to check the metrics recorded by receivers, producers or exporters.
// The returned function should be deferred "defer SetupRecordedMetricsTest()()".
func SetupRecordedMetricsTest() func() {
	// Register a nop metrics exporter for the OC library.
	nmp := new(nopMetricsExporter)
	view.RegisterExporter(nmp)

	// Now for the stats exporter
	view.Register(observability.AllViews...)

	return func() {
		view.UnregisterExporter(nmp)
		view.Unregister(observability.AllViews...)
	}
}

// CheckRecordedMetricsForTraceExporter checks that the given TraceExporter records the correct set of metrics with the correct
// set of tags by sending few TraceData to the exporter. The exporter should be able to handle the requests correctly without
// dropping.
func CheckRecordedMetricsForTraceExporter(te exporter.TraceExporter) error {
	defer SetupRecordedMetricsTest()()

	now := time.Now().UTC()
	spans := []*tracepb.Span{
		{
			TraceId:      []byte{0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2A, 0x2B, 0x2C, 0x2D, 0x2E},
			SpanId:       []byte{0xF0, 0xF1, 0xF2, 0xF3, 0xF4, 0xF5, 0xF6, 0xF7},
			ParentSpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			Name:         &tracepb.TruncatableString{Value: "ServerSpan"},
			Kind:         tracepb.Span_SERVER,
			StartTime:    internal.TimeToTimestamp(now.Add(-15 * time.Millisecond)),
			EndTime:      internal.TimeToTimestamp(now),
			Status:       &tracepb.Status{Code: int32(0), Message: "OK"},
			Tracestate:   &tracepb.Span_Tracestate{},
			Links: &tracepb.Span_Links{
				Link: []*tracepb.Span_Link{
					{
						TraceId: []byte{0x4F, 0x4E, 0x4D, 0x4C, 0x4B, 0x4A, 0x49, 0x48, 0x47, 0x46, 0x45, 0x44, 0x43, 0x42, 0x41, 0x40},
						SpanId:  []byte{0x7F, 0x7E, 0x7D, 0x7C, 0x7B, 0x7A, 0x79, 0x78},
						Type:    tracepb.Span_Link_PARENT_LINKED_SPAN,
					},
				},
			},
		},
	}
	td := data.TraceData{Spans: spans}
	ctx := observability.ContextWithReceiverName(context.Background(), fakeReceiverName)
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		if err := te.ConsumeTraceData(ctx, td); err != nil {
			return fmt.Errorf("Want nil got %v", err)
		}
	}

	if err := CheckValueViewExporterReceivedSpans(fakeReceiverName, te.TraceExportFormat(), numBatches*len(spans)); err != nil {
		return err
	}
	if err := CheckValueViewExporterDroppedSpans(fakeReceiverName, te.TraceExportFormat(), 0); err != nil {
		return err
	}
	return nil
}

// CheckValueViewExporterReceivedSpans checks that for the current exported value in the ViewExporterReceivedSpans
// for {TagKeyReceiver: receiverName, TagKeyExporter: exporterTagName} is equal to "value".
// When this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckValueViewExporterReceivedSpans(receiverName string, exporterTagName string, value int) error {
	return checkValueForView(observability.ViewExporterReceivedSpans.Name,
		wantsTagsForExporterView(receiverName, exporterTagName), int64(value))
}

// CheckValueViewExporterDroppedSpans checks that for the current exported value in the ViewExporterDroppedSpans
// for {TagKeyReceiver: receiverName} is equal to "value".
// In tests that this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckValueViewExporterDroppedSpans(receiverName string, exporterTagName string, value int) error {
	return checkValueForView(observability.ViewExporterDroppedSpans.Name,
		wantsTagsForExporterView(receiverName, exporterTagName), int64(value))
}

// CheckValueViewReceiverReceivedSpans checks that for the current exported value in the ViewReceiverReceivedSpans
// for {TagKeyReceiver: receiverName, TagKeyExporter: exporterTagName} is equal to "value".
// In tests that this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckValueViewReceiverReceivedSpans(receiverName string, value int) error {
	return checkValueForView(observability.ViewReceiverReceivedSpans.Name,
		wantsTagsForReceiverView(receiverName), int64(value))
}

// CheckValueViewReceiverDroppedSpans checks that for the current exported value in the ViewReceiverDroppedSpans
// for {TagKeyReceiver: receiverName} is equal to "value".
// In tests that this function is called it is required to also call SetupRecordedMetricsTest as first thing.
func CheckValueViewReceiverDroppedSpans(receiverName string, value int) error {
	return checkValueForView(observability.ViewReceiverDroppedSpans.Name,
		wantsTagsForReceiverView(receiverName), int64(value))
}

func checkValueForView(vName string, wantTags []tag.Tag, value int64) error {
	// Make sure the tags slice is sorted by tag keys.
	sortTags(wantTags)

	rows, err := view.RetrieveData(vName)
	if err != nil {
		return fmt.Errorf("Error retrieving view data for view Name %s", vName)
	}

	for _, row := range rows {
		// Make sure the tags slice is sorted by tag keys.
		sortTags(row.Tags)
		if reflect.DeepEqual(wantTags, row.Tags) {
			sum := row.Data.(*view.SumData)
			if float64(value) != sum.Value {
				return fmt.Errorf("Different recorded value: want %v got %v", float64(value), sum.Value)
			}
			// We found the result
			return nil
		}
	}
	return fmt.Errorf("Could not find wantTags: %s in rows %v", wantTags, rows)
}

func wantsTagsForExporterView(receiverName string, exporterTagName string) []tag.Tag {
	return []tag.Tag{
		{Key: observability.TagKeyReceiver, Value: receiverName},
		{Key: observability.TagKeyExporter, Value: exporterTagName},
	}
}

func wantsTagsForReceiverView(receiverName string) []tag.Tag {
	return []tag.Tag{
		{Key: observability.TagKeyReceiver, Value: receiverName},
	}
}

func sortTags(tags []tag.Tag) {
	sort.SliceStable(tags, func(i, j int) bool {
		return tags[i].Key.Name() < tags[j].Key.Name()
	})
}
