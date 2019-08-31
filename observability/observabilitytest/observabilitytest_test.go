// Copyright 2019, OpenTelemetry Authors
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
	"testing"

	"github.com/open-telemetry/opentelemetry-service/observability"
	"github.com/open-telemetry/opentelemetry-service/observability/observabilitytest"
)

const (
	receiverName = "fake_receiver"
	exporterName = "fake_exporter"
)

func TestCheckValueViewTraceReceiverViews(t *testing.T) {
	doneFn := observabilitytest.SetupRecordedMetricsTest()
	defer doneFn()

	receiverCtx := observability.ContextWithReceiverName(context.Background(), receiverName)
	observability.RecordMetricsForTraceReceiver(receiverCtx, 17, 13)
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

func TestCheckValueViewMetricsReceiverViews(t *testing.T) {
	doneFn := observabilitytest.SetupRecordedMetricsTest()
	defer doneFn()

	receiverCtx := observability.ContextWithReceiverName(context.Background(), receiverName)
	observability.RecordMetricsForMetricsReceiver(receiverCtx, 17, 13)
	// Test expected values.
	if err := observabilitytest.CheckValueViewReceiverReceivedTimeSeries(receiverName, 17); err != nil {
		t.Fatalf("When check recorded values: want nil got %v", err)
	}
	if err := observabilitytest.CheckValueViewReceiverDroppedTimeSeries(receiverName, 13); err != nil {
		t.Fatalf("When check recorded values: want nil got %v", err)
	}
	// Test unexpected tag values
	if err := observabilitytest.CheckValueViewReceiverReceivedTimeSeries(exporterName, 17); err == nil {
		t.Fatalf("When check recorded values: want not-nil got nil")
	}
	if err := observabilitytest.CheckValueViewReceiverDroppedTimeSeries(exporterName, 13); err == nil {
		t.Fatalf("When check recorded values: want not-nil got nil")
	}
	// Test unexpected recorded values
	if err := observabilitytest.CheckValueViewReceiverReceivedTimeSeries(receiverName, 13); err == nil {
		t.Fatalf("When check recorded values: want not-nil got nil")
	}
	if err := observabilitytest.CheckValueViewReceiverDroppedTimeSeries(receiverName, 17); err == nil {
		t.Fatalf("When check recorded values: want not-nil got nil")
	}
}

func TestCheckValueViewTraceExporterViews(t *testing.T) {
	doneFn := observabilitytest.SetupRecordedMetricsTest()
	defer doneFn()

	receiverCtx := observability.ContextWithReceiverName(context.Background(), receiverName)
	exporterCtx := observability.ContextWithExporterName(receiverCtx, exporterName)
	observability.RecordMetricsForTraceExporter(exporterCtx, 17, 13)
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

func TestCheckValueViewMetricsExporterViews(t *testing.T) {
	doneFn := observabilitytest.SetupRecordedMetricsTest()
	defer doneFn()

	receiverCtx := observability.ContextWithReceiverName(context.Background(), receiverName)
	exporterCtx := observability.ContextWithExporterName(receiverCtx, exporterName)
	observability.RecordMetricsForMetricsExporter(exporterCtx, 17, 13)
	// Test expected values.
	if err := observabilitytest.CheckValueViewExporterReceivedTimeSeries(receiverName, exporterName, 17); err != nil {
		t.Fatalf("When check recorded values: want nil got %v", err)
	}
	if err := observabilitytest.CheckValueViewExporterDroppedTimeSeries(receiverName, exporterName, 13); err != nil {
		t.Fatalf("When check recorded values: want nil got %v", err)
	}
	// Test unexpected tag values
	if err := observabilitytest.CheckValueViewExporterReceivedTimeSeries(receiverName, receiverName, 17); err == nil {
		t.Fatalf("When check recorded values: want not-nil got nil")
	}
	if err := observabilitytest.CheckValueViewExporterDroppedTimeSeries(receiverName, receiverName, 13); err == nil {
		t.Fatalf("When check recorded values: want not-nil got nil")
	}
	// Test unexpected recorded values
	if err := observabilitytest.CheckValueViewExporterReceivedTimeSeries(receiverName, exporterName, 13); err == nil {
		t.Fatalf("When check recorded values: want not-nil got nil")
	}
	if err := observabilitytest.CheckValueViewExporterDroppedTimeSeries(receiverName, exporterName, 17); err == nil {
		t.Fatalf("When check recorded values: want not-nil got nil")
	}
}

func TestNoSetupCalled(t *testing.T) {
	receiverCtx := observability.ContextWithReceiverName(context.Background(), receiverName)
	observability.RecordMetricsForTraceReceiver(receiverCtx, 17, 13)
	// Failed to check because views are not registered.
	if err := observabilitytest.CheckValueViewReceiverReceivedSpans(receiverName, 17); err == nil {
		t.Fatalf("When check recorded values: want not-nil got nil")
	}
}
