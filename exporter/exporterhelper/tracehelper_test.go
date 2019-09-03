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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-service/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-service/exporter"
	"github.com/open-telemetry/opentelemetry-service/observability"
	"github.com/open-telemetry/opentelemetry-service/observability/observabilitytest"
)

const (
	fakeTraceReceiverName = "fake_receiver"
	fakeTraceExporterName = "fake_exporter"
	fakeSpanName          = "fake_span_name"
	fakeParentSpanName    = "fake_parent_span_name"
)

// TODO https://github.com/open-telemetry/opentelemetry-service/issues/266
// Migrate tests to use testify/assert instead of t.Fatal pattern.
func TestTraceExporter_InvalidName(t *testing.T) {
	te, err := NewTraceExporter("", newPushTraceData(0, nil))
	require.Nil(t, te)
	require.Equal(t, errEmptyExporterName, err)
}

func TestTraceExporter_NilPushTraceData(t *testing.T) {
	te, err := NewTraceExporter(fakeTraceExporterName, nil)
	require.Nil(t, te)
	require.Equal(t, errNilPushTraceData, err)
}

func TestTraceExporter_Default(t *testing.T) {
	td := consumerdata.TraceData{}
	te, err := NewTraceExporter(fakeTraceExporterName, newPushTraceData(0, nil))
	assert.NotNil(t, te)
	assert.Nil(t, err)

	assert.Nil(t, te.ConsumeTraceData(context.Background(), td))
	assert.Equal(t, te.Name(), fakeTraceExporterName)
	assert.Nil(t, te.Shutdown())
}

func TestTraceExporter_Default_ReturnError(t *testing.T) {
	td := consumerdata.TraceData{}
	want := errors.New("my_error")
	te, err := NewTraceExporter(fakeTraceExporterName, newPushTraceData(0, want))
	require.Nil(t, err)
	require.NotNil(t, te)

	err = te.ConsumeTraceData(context.Background(), td)
	require.Equalf(t, want, err, "ConsumeTraceData returns: Want %v Got %v", want, err)
}

func TestTraceExporter_WithRecordMetrics(t *testing.T) {
	te, err := NewTraceExporter(fakeTraceExporterName, newPushTraceData(0, nil), WithRecordMetrics(true))
	require.Nil(t, err)
	require.NotNil(t, te)

	checkRecordedMetricsForTraceExporter(t, te, nil, 0)
}

func TestTraceExporter_WithRecordMetrics_NonZeroDropped(t *testing.T) {
	te, err := NewTraceExporter(fakeTraceExporterName, newPushTraceData(1, nil), WithRecordMetrics(true))
	require.Nil(t, err)
	require.NotNil(t, te)

	checkRecordedMetricsForTraceExporter(t, te, nil, 1)
}

func TestTraceExporter_WithRecordMetrics_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	te, err := NewTraceExporter(fakeTraceExporterName, newPushTraceData(0, want), WithRecordMetrics(true))
	require.Nil(t, err)
	require.NotNil(t, te)

	checkRecordedMetricsForTraceExporter(t, te, want, 0)
}

func TestTraceExporter_WithSpan(t *testing.T) {
	te, err := NewTraceExporter(fakeTraceExporterName, newPushTraceData(0, nil), WithSpanName(fakeSpanName))
	require.Nil(t, err)
	require.NotNil(t, te)

	checkWrapSpanForTraceExporter(t, te, nil, 0)
}

func TestTraceExporter_WithSpan_NonZeroDropped(t *testing.T) {
	te, err := NewTraceExporter(fakeTraceExporterName, newPushTraceData(1, nil), WithSpanName(fakeSpanName))
	require.Nil(t, err)
	require.NotNil(t, te)

	checkWrapSpanForTraceExporter(t, te, nil, 1)
}

func TestTraceExporter_WithSpan_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	te, err := NewTraceExporter(fakeTraceExporterName, newPushTraceData(0, want), WithSpanName(fakeSpanName))
	require.Nil(t, err)
	require.NotNil(t, te)

	checkWrapSpanForTraceExporter(t, te, want, 0)
}

func TestTraceExporter_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func() error { shutdownCalled = true; return nil }

	te, err := NewTraceExporter(fakeTraceExporterName, newPushTraceData(0, nil), WithShutdown(shutdown))
	assert.NotNil(t, te)
	assert.Nil(t, err)

	assert.Nil(t, te.Shutdown())
	assert.True(t, shutdownCalled)
}

func TestTraceExporter_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func() error { return want }

	te, err := NewTraceExporter(fakeTraceExporterName, newPushTraceData(0, nil), WithShutdown(shutdownErr))
	assert.NotNil(t, te)
	assert.Nil(t, err)

	assert.Equal(t, te.Shutdown(), want)
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
	ctx := observability.ContextWithReceiverName(context.Background(), fakeTraceReceiverName)
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		require.Equal(t, wantError, te.ConsumeTraceData(ctx, td))
	}

	err := observabilitytest.CheckValueViewExporterReceivedSpans(fakeTraceReceiverName, te.Name(), numBatches*len(spans))
	require.Nilf(t, err, "CheckValueViewExporterReceivedSpans: Want nil Got %v", err)

	err = observabilitytest.CheckValueViewExporterDroppedSpans(fakeTraceReceiverName, te.Name(), numBatches*droppedSpans)
	require.Nilf(t, err, "CheckValueViewExporterDroppedSpans: Want nil Got %v", err)
}

func generateTraceTraffic(t *testing.T, te exporter.TraceExporter, numRequests int, wantError error) {
	td := consumerdata.TraceData{Spans: make([]*tracepb.Span, 1)}
	ctx, span := trace.StartSpan(context.Background(), fakeParentSpanName, trace.WithSampler(trace.AlwaysSample()))
	defer span.End()
	for i := 0; i < numRequests; i++ {
		require.Equal(t, wantError, te.ConsumeTraceData(ctx, td))
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

	require.NotEqual(t, 0, len(ocSpansSaver.spanData), "No exported span data.")

	gotSpanData := ocSpansSaver.spanData[:]
	require.Equal(t, numRequests+1, len(gotSpanData))

	parentSpan := gotSpanData[numRequests]
	require.Equalf(t, fakeParentSpanName, parentSpan.Name, "SpanData %v", parentSpan)

	for _, sd := range gotSpanData[:numRequests] {
		require.Equalf(t, parentSpan.SpanContext.SpanID, sd.ParentSpanID, "Exporter span not a child\nSpanData %v", sd)
		require.Equalf(t, errToStatus(wantError), sd.Status, "SpanData %v", sd)
		require.Equalf(t, int64(1), sd.Attributes[numReceivedSpansAttribute], "SpanData %v", sd)
		require.Equalf(t, int64(droppedSpans), sd.Attributes[numDroppedSpansAttribute], "SpanData %v", sd)
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
