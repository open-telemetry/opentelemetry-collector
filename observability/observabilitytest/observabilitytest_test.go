// Copyright 2019, OpenCensus Authors
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

// observabilitytest_test instead of just observabilitytest to avoid dependency cycle between
// observabilitytest and other test packages.
package observabilitytest_test

import (
	"context"
	"errors"
	"testing"

	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/exporter"
	"github.com/census-instrumentation/opencensus-service/exporter/exportertest"
	"github.com/census-instrumentation/opencensus-service/observability"
	"github.com/census-instrumentation/opencensus-service/observability/observabilitytest"
)

const (
	receiverName = "fake_receiver"
	exporterName = "fake_exporter"
)

func TestCheckValueViewReceiverViews(t *testing.T) {
	defer observabilitytest.SetupRecordedMetricsTest()()

	receiverCtx := observability.ContextWithReceiverName(context.Background(), receiverName)
	observability.RecordTraceReceiverMetrics(receiverCtx, 17, 13)
	// Test expected values.
	if err := observabilitytest.CheckValueViewReceiverReceivedSpans(receiverName, 17); err != nil {
		t.Fatalf("When check recorded values: want nil got %v", err)
	}
	if err := observabilitytest.CheckValueViewReceiverDroppedSpans(receiverName, 13); err != nil {
		t.Fatalf("When check recorded values: want nil got %v", err)
	}
	// Test unexpected tag values
	if err := observabilitytest.CheckValueViewReceiverReceivedSpans(exporterName, 17); err == nil {
		t.Fatalf("When check recorded values: want not-nil got nil")
	}
	if err := observabilitytest.CheckValueViewReceiverDroppedSpans(exporterName, 13); err == nil {
		t.Fatalf("When check recorded values: want not-nil got nil")
	}
	// Test unexpected recorded values
	if err := observabilitytest.CheckValueViewReceiverReceivedSpans(receiverName, 13); err == nil {
		t.Fatalf("When check recorded values: want not-nil got nil")
	}
	if err := observabilitytest.CheckValueViewReceiverDroppedSpans(receiverName, 17); err == nil {
		t.Fatalf("When check recorded values: want not-nil got nil")
	}
}

func TestCheckValueViewExporterViews(t *testing.T) {
	defer observabilitytest.SetupRecordedMetricsTest()()

	receiverCtx := observability.ContextWithReceiverName(context.Background(), receiverName)
	exporterCtx := observability.ContextWithExporterName(receiverCtx, exporterName)
	observability.RecordTraceExporterMetrics(exporterCtx, 17, 13)
	// Test expected values.
	if err := observabilitytest.CheckValueViewExporterReceivedSpans(receiverName, exporterName, 17); err != nil {
		t.Fatalf("When check recorded values: want nil got %v", err)
	}
	if err := observabilitytest.CheckValueViewExporterDroppedSpans(receiverName, exporterName, 13); err != nil {
		t.Fatalf("When check recorded values: want nil got %v", err)
	}
	// Test unexpected tag values
	if err := observabilitytest.CheckValueViewExporterReceivedSpans(receiverName, receiverName, 17); err == nil {
		t.Fatalf("When check recorded values: want not-nil got nil")
	}
	if err := observabilitytest.CheckValueViewExporterDroppedSpans(receiverName, receiverName, 13); err == nil {
		t.Fatalf("When check recorded values: want not-nil got nil")
	}
	// Test unexpected recorded values
	if err := observabilitytest.CheckValueViewExporterReceivedSpans(receiverName, exporterName, 13); err == nil {
		t.Fatalf("When check recorded values: want not-nil got nil")
	}
	if err := observabilitytest.CheckValueViewExporterDroppedSpans(receiverName, exporterName, 17); err == nil {
		t.Fatalf("When check recorded values: want not-nil got nil")
	}
}

func TestNoSetupCalled(t *testing.T) {
	receiverCtx := observability.ContextWithReceiverName(context.Background(), receiverName)
	observability.RecordTraceReceiverMetrics(receiverCtx, 17, 13)
	// Failed to check because views are not registered.
	if err := observabilitytest.CheckValueViewReceiverReceivedSpans(receiverName, 17); err == nil {
		t.Fatalf("When check recorded values: want not-nil got nil")
	}
}

func TestCheckRecordedMetricsForTraceExporter_WithNoMetricsRecorded(t *testing.T) {
	ne := exportertest.NewNopTraceExporter()
	if err := observabilitytest.CheckRecordedMetricsForTraceExporter(ne); err == nil {
		t.Fatalf("When check recorded values: want not-nil got nil")
	}
}

func TestCheckRecordedMetricsForTraceExporter_WithReturnError(t *testing.T) {
	newe := exportertest.NewNopTraceExporter(exportertest.WithReturnError(errors.New("my_error")))
	if err := observabilitytest.CheckRecordedMetricsForTraceExporter(newe); err == nil {
		t.Fatalf("When check recorded values: want not-nil got nil")
	}
}

func TestCheckRecordedMetricsForTraceExporter_WrongReceived(t *testing.T) {
	if err := observabilitytest.CheckRecordedMetricsForTraceExporter(&obsTestExporter{receivedSpans: 2, droppedSpans: 0}); err == nil {
		t.Fatalf("When check recorded values: want not-nil got nil")
	}
}
func TestCheckRecordedMetricsForTraceExporter_WrongDropped(t *testing.T) {
	if err := observabilitytest.CheckRecordedMetricsForTraceExporter(&obsTestExporter{receivedSpans: 1, droppedSpans: 1}); err == nil {
		t.Fatalf("When check recorded values: want not-nil got nil")
	}
}

func TestCheckRecordedMetricsForTraceExporter(t *testing.T) {
	if err := observabilitytest.CheckRecordedMetricsForTraceExporter(&obsTestExporter{receivedSpans: 1, droppedSpans: 0}); err != nil {
		t.Fatalf("When check recorded values: want nil got %v", err)
	}
}

type obsTestExporter struct{ receivedSpans, droppedSpans int }

var _ exporter.TraceExporter = (*obsTestExporter)(nil)

func (ote *obsTestExporter) ConsumeTraceData(ctx context.Context, td data.TraceData) error {
	exporterCtx := observability.ContextWithExporterName(ctx, ote.TraceExportFormat())
	observability.RecordTraceExporterMetrics(exporterCtx, ote.receivedSpans, ote.droppedSpans)
	return nil
}

func (ote *obsTestExporter) TraceExportFormat() string {
	return exporterName
}
