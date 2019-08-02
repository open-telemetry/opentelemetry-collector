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
package exporterhelper

import (
	"context"
	"errors"
	"sync"
	"testing"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-service/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-service/exporter"
	"github.com/open-telemetry/opentelemetry-service/observability"
	"github.com/open-telemetry/opentelemetry-service/observability/observabilitytest"
)

const (
	fakeReceiverName   = "fake_receiver_trace"
	fakeExporterName   = "fake_exporter_trace"
	fakeSpanName       = "fake_span_name"
	fakeParentSpanName = "fake_parent_span_name"
)

func TestTraceExporter_InvalidName(t *testing.T) {
	if _, err := NewTraceExporter("", newPushTraceData(0, nil)); err != errEmptyExporterName {
		t.Fatalf("NewTraceExporter returns: Want %v Got %v", errEmptyExporterName, err)
	}
}

func TestTraceExporter_NilPushTraceData(t *testing.T) {
	if _, err := NewTraceExporter(fakeExporterName, nil); err != errNilPushTraceData {
		t.Fatalf("NewTraceExporter returns: Want %v Got %v", errNilPushTraceData, err)
	}
}
func TestTraceExporter_Default(t *testing.T) {
	td := consumerdata.TraceData{}
	te, err := NewTraceExporter(fakeExporterName, newPushTraceData(0, nil))
	if err != nil {
		t.Fatalf("NewTraceExporter returns: Want nil Got %v", err)
	}
	if err := te.ConsumeTraceData(context.Background(), td); err != nil {
		t.Fatalf("ConsumeTraceData returns: Want nil Got %v", err)
	}
	if g, w := te.TraceExportFormat(), fakeExporterName; g != w {
		t.Fatalf("TraceExportFormat returns: Want %s Got %s", w, g)
	}
}

func TestTraceExporter_Default_ReturnError(t *testing.T) {
	td := consumerdata.TraceData{}
	want := errors.New("my_error")
	te, err := NewTraceExporter(fakeExporterName, newPushTraceData(0, want))
	if err != nil {
		t.Fatalf("NewTraceExporter returns: Want nil Got %v", err)
	}
	if err := te.ConsumeTraceData(context.Background(), td); err != want {
		t.Fatalf("ConsumeTraceData returns: Want %v Got %v", want, err)
	}
}

func TestTraceExporter_WithRecordMetrics(t *testing.T) {
	te, err := NewTraceExporter(fakeExporterName, newPushTraceData(0, nil), WithRecordMetrics(true))
	if err != nil {
		t.Fatalf("NewTraceExporter returns: Want nil Got %v", err)
	}
	checkRecordedMetricsForTraceExporter(t, te, nil, 0)
}

func TestTraceExporter_WithRecordMetrics_NonZeroDropped(t *testing.T) {
	te, err := NewTraceExporter(fakeExporterName, newPushTraceData(1, nil), WithRecordMetrics(true))
	if err != nil {
		t.Fatalf("NewTraceExporter returns: Want nil Got %v", err)
	}
	checkRecordedMetricsForTraceExporter(t, te, nil, 1)
}

func TestTraceExporter_WithRecordMetrics_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	te, err := NewTraceExporter(fakeExporterName, newPushTraceData(0, want), WithRecordMetrics(true))
	if err != nil {
		t.Fatalf("NewTraceExporter returns: Want nil Got %v", err)
	}
	checkRecordedMetricsForTraceExporter(t, te, want, 0)
}

func TestTraceExporter_WithSpan(t *testing.T) {
	te, err := NewTraceExporter(fakeExporterName, newPushTraceData(0, nil), WithSpanName(fakeSpanName))
	if err != nil {
		t.Fatalf("NewTraceExporter returns: Want nil Got %v", err)
	}
	checkWrapSpanForTraceExporter(t, te, nil, 0)
}

func TestTraceExporter_WithSpan_NonZeroDropped(t *testing.T) {
	te, err := NewTraceExporter(fakeExporterName, newPushTraceData(1, nil), WithSpanName(fakeSpanName))
	if err != nil {
		t.Fatalf("NewTraceExporter returns: Want nil Got %v", err)
	}
	checkWrapSpanForTraceExporter(t, te, nil, 1)
}

func TestTraceExporter_WithSpan_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	te, err := NewTraceExporter(fakeExporterName, newPushTraceData(0, want), WithSpanName(fakeSpanName))
	if err != nil {
		t.Fatalf("NewTraceExporter returns: Want nil Got %v", err)
	}
	checkWrapSpanForTraceExporter(t, te, want, 0)
}

func newPushTraceData(droppedSpans int, retError error) PushTraceData {
	return func(ctx context.Context, td consumerdata.TraceData) (int, error) {
		return droppedSpans, retError
	}
}

func checkRecordedMetricsForTraceExporter(t *testing.T, te exporter.TraceExporter, wantError error, droppedSpans int) {
	doneFn := observabilitytest.SetupRecordedMetricsTest()
	defer doneFn()

	spans := make([]*tracepb.Span, 2)
	td := consumerdata.TraceData{Spans: spans}
	ctx := observability.ContextWithReceiverName(context.Background(), fakeReceiverName)
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		if err := te.ConsumeTraceData(ctx, td); err != wantError {
			t.Fatalf("Want %v Got %v", wantError, err)
		}
	}

	if err := observabilitytest.CheckValueViewExporterReceivedSpans(fakeReceiverName, te.TraceExportFormat(), numBatches*len(spans)); err != nil {
		t.Fatalf("CheckValueViewExporterReceivedSpans: Want nil Got %v", err)
	}
	if err := observabilitytest.CheckValueViewExporterDroppedSpans(fakeReceiverName, te.TraceExportFormat(), numBatches*droppedSpans); err != nil {
		t.Fatalf("CheckValueViewExporterDroppedSpans: Want nil Got %v", err)
	}
}

func generateTraceTraffic(t *testing.T, te exporter.TraceExporter, numRequests int, wantError error) {
	td := consumerdata.TraceData{Spans: make([]*tracepb.Span, 1)}
	ctx, span := trace.StartSpan(context.Background(), fakeParentSpanName, trace.WithSampler(trace.AlwaysSample()))
	defer span.End()
	for i := 0; i < numRequests; i++ {
		if err := te.ConsumeTraceData(ctx, td); err != wantError {
			t.Fatalf("Want %v Got %v", wantError, err)
		}
	}
}

func checkWrapSpanForTraceExporter(t *testing.T, te exporter.TraceExporter, wantError error, droppedSpans int) {
	ocSpansSaver := new(testOCTraceExporter)
	trace.RegisterExporter(ocSpansSaver)
	defer trace.UnregisterExporter(ocSpansSaver)

	const numRequests = 5
	generateTraceTraffic(t, te, numRequests, wantError)

	// Inspection time!
	ocSpansSaver.mu.Lock()
	defer ocSpansSaver.mu.Unlock()

	if len(ocSpansSaver.spanData) == 0 {
		t.Fatal("No exported span data.")
	}

	gotSpanData := ocSpansSaver.spanData[:]
	if g, w := len(gotSpanData), numRequests+1; g != w {
		t.Fatalf("Spandata count: Want %d Got %d", w, g)
	}

	parentSpan := gotSpanData[numRequests]
	if g, w := parentSpan.Name, fakeParentSpanName; g != w {
		t.Fatalf("Parent span name: Want %s Got %s\nSpanData %v", w, g, parentSpan)
	}

	for _, sd := range gotSpanData[:numRequests] {
		if g, w := sd.ParentSpanID, parentSpan.SpanContext.SpanID; g != w {
			t.Fatalf("Exporter span not a child: Want %d Got %d\nSpanData %v", w, g, sd)
		}
		if g, w := sd.Status, errToStatus(wantError); g != w {
			t.Fatalf("Status: Want %v Got %v\nSpanData %v", w, g, sd)
		}
		if g, w := sd.Attributes[numReceivedSpansAttribute], int64(1); g != w {
			t.Fatalf("Number of received spans attribute: Want %d Got %d\nSpanData %v", w, g, sd)
		}
		if g, w := sd.Attributes[numDroppedSpansAttribute], int64(droppedSpans); g != w {
			t.Fatalf("Number of dropped spans attribute: Want %d Got %d\nSpanData %v", w, g, sd)
		}
	}
}

type testOCTraceExporter struct {
	mu       sync.Mutex
	spanData []*trace.SpanData
}

func (tote *testOCTraceExporter) ExportSpan(sd *trace.SpanData) {
	tote.mu.Lock()
	defer tote.mu.Unlock()

	tote.spanData = append(tote.spanData, sd)
}
