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
	"errors"
	"sync"
	"testing"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/data/testdata"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
)

const (
	fakeTraceExporterType   = "fake_trace_exporter"
	fakeTraceExporterName   = "fake_trace_exporter/with_name"
	fakeTraceParentSpanName = "fake_trace_parent_span_name"
)

var (
	fakeTraceExporterConfig = &configmodels.ExporterSettings{
		TypeVal: fakeTraceExporterType,
		NameVal: fakeTraceExporterName,
	}
)

func TestTracesRequest(t *testing.T) {
	mr := newTracesRequest(context.Background(), testdata.GenerateTraceDataOneSpan(), nil)

	partialErr := consumererror.PartialTracesError(errors.New("some error"), testdata.GenerateTraceDataEmpty())
	assert.EqualValues(t, newTracesRequest(context.Background(), testdata.GenerateTraceDataEmpty(), nil), mr.onPartialError(partialErr.(consumererror.PartialError)))
}

// TODO https://go.opentelemetry.io/collector/issues/266
// Migrate tests to use testify/assert instead of t.Fatal pattern.
func TestTraceExporterOld_InvalidName(t *testing.T) {
	te, err := NewTraceExporterOld(nil, newTraceDataPusherOld(0, nil))
	require.Nil(t, te)
	require.Equal(t, errNilConfig, err)
}

func TestTraceExporterOld_NilPushTraceData(t *testing.T) {
	te, err := NewTraceExporterOld(fakeTraceExporterConfig, nil)
	require.Nil(t, te)
	require.Equal(t, errNilPushTraceData, err)
}

func TestTraceExporterOld_Default(t *testing.T) {
	td := consumerdata.TraceData{}
	te, err := NewTraceExporterOld(fakeTraceExporterConfig, newTraceDataPusherOld(0, nil))
	assert.NotNil(t, te)
	assert.NoError(t, err)

	assert.Nil(t, te.ConsumeTraceData(context.Background(), td))
	assert.Nil(t, te.Shutdown(context.Background()))
}

func TestTraceExporterOld_Default_ReturnError(t *testing.T) {
	td := consumerdata.TraceData{}
	want := errors.New("my_error")
	te, err := NewTraceExporterOld(fakeTraceExporterConfig, newTraceDataPusherOld(0, want))
	require.Nil(t, err)
	require.NotNil(t, te)

	err = te.ConsumeTraceData(context.Background(), td)
	require.Equalf(t, want, err, "ConsumeTraceData returns: Want %v Got %v", want, err)
}

func TestTraceExporterOld_WithRecordMetrics(t *testing.T) {
	te, err := NewTraceExporterOld(fakeTraceExporterConfig, newTraceDataPusherOld(0, nil))
	require.Nil(t, err)
	require.NotNil(t, te)

	checkRecordedMetricsForTraceExporterOld(t, te, nil, 0)
}

func TestTraceExporterOld_WithRecordMetrics_NonZeroDropped(t *testing.T) {
	te, err := NewTraceExporterOld(fakeTraceExporterConfig, newTraceDataPusherOld(1, nil))
	require.Nil(t, err)
	require.NotNil(t, te)

	checkRecordedMetricsForTraceExporterOld(t, te, nil, 1)
}

func TestTraceExporterOld_WithRecordMetrics_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	te, err := NewTraceExporterOld(fakeTraceExporterConfig, newTraceDataPusherOld(0, want))
	require.Nil(t, err)
	require.NotNil(t, te)

	checkRecordedMetricsForTraceExporterOld(t, te, want, 0)
}

func TestTraceExporterOld_WithSpan(t *testing.T) {
	te, err := NewTraceExporterOld(fakeTraceExporterConfig, newTraceDataPusherOld(0, nil))
	require.Nil(t, err)
	require.NotNil(t, te)

	checkWrapSpanForTraceExporterOld(t, te, nil, 1)
}

func TestTraceExporterOld_WithSpan_NonZeroDropped(t *testing.T) {
	te, err := NewTraceExporterOld(fakeTraceExporterConfig, newTraceDataPusherOld(1, nil))
	require.Nil(t, err)
	require.NotNil(t, te)

	checkWrapSpanForTraceExporterOld(t, te, nil, 1)
}

func TestTraceExporterOld_WithSpan_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	te, err := NewTraceExporterOld(fakeTraceExporterConfig, newTraceDataPusherOld(0, want))
	require.Nil(t, err)
	require.NotNil(t, te)

	checkWrapSpanForTraceExporterOld(t, te, want, 1)
}

func TestTraceExporterOld_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	te, err := NewTraceExporterOld(fakeTraceExporterConfig, newTraceDataPusherOld(0, nil), WithShutdown(shutdown))
	assert.NotNil(t, te)
	assert.NoError(t, err)

	assert.Nil(t, te.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestTraceExporterOld_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	te, err := NewTraceExporterOld(fakeTraceExporterConfig, newTraceDataPusherOld(0, nil), WithShutdown(shutdownErr))
	assert.NotNil(t, te)
	assert.NoError(t, err)

	assert.Equal(t, te.Shutdown(context.Background()), want)
}

func newTraceDataPusherOld(droppedSpans int, retError error) traceDataPusherOld {
	return func(ctx context.Context, td consumerdata.TraceData) (int, error) {
		return droppedSpans, retError
	}
}

func checkRecordedMetricsForTraceExporterOld(t *testing.T, te component.TraceExporterOld, wantError error, droppedSpans int) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	spans := make([]*tracepb.Span, 2)
	td := consumerdata.TraceData{Spans: spans}
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		require.Equal(t, wantError, te.ConsumeTraceData(context.Background(), td))
	}

	// TODO: When the new metrics correctly count partial dropped fix this.
	if wantError != nil {
		obsreporttest.CheckExporterTracesViews(t, fakeTraceExporterName, 0, int64(numBatches*len(spans)))
	} else {
		obsreporttest.CheckExporterTracesViews(t, fakeTraceExporterName, int64(numBatches*len(spans)), 0)
	}
}

func generateTraceTrafficOld(t *testing.T, te component.TraceExporterOld, numRequests int, wantError error) {
	td := consumerdata.TraceData{Spans: make([]*tracepb.Span, 1)}
	ctx, span := trace.StartSpan(context.Background(), fakeTraceParentSpanName, trace.WithSampler(trace.AlwaysSample()))
	defer span.End()
	for i := 0; i < numRequests; i++ {
		require.Equal(t, wantError, te.ConsumeTraceData(ctx, td))
	}
}

func checkWrapSpanForTraceExporterOld(t *testing.T, te component.TraceExporterOld, wantError error, numSpans int64) {
	ocSpansSaver := new(testOCTraceExporter)
	trace.RegisterExporter(ocSpansSaver)
	defer trace.UnregisterExporter(ocSpansSaver)

	const numRequests = 5
	generateTraceTrafficOld(t, te, numRequests, wantError)

	// Inspection time!
	ocSpansSaver.mu.Lock()
	defer ocSpansSaver.mu.Unlock()

	require.NotEqual(t, 0, len(ocSpansSaver.spanData), "No exported span data.")

	gotSpanData := ocSpansSaver.spanData
	require.Equal(t, numRequests+1, len(gotSpanData))

	parentSpan := gotSpanData[numRequests]
	require.Equalf(t, fakeTraceParentSpanName, parentSpan.Name, "SpanData %v", parentSpan)

	for _, sd := range gotSpanData[:numRequests] {
		require.Equalf(t, parentSpan.SpanContext.SpanID, sd.ParentSpanID, "Exporter span not a child\nSpanData %v", sd)
		require.Equalf(t, errToStatus(wantError), sd.Status, "SpanData %v", sd)

		sentSpans := numSpans
		var failedToSendSpans int64
		if wantError != nil {
			sentSpans = 0
			failedToSendSpans = numSpans
		}
		require.Equalf(t, sentSpans, sd.Attributes[obsreport.SentSpansKey], "SpanData %v", sd)
		require.Equalf(t, failedToSendSpans, sd.Attributes[obsreport.FailedToSendSpansKey], "SpanData %v", sd)
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

func TestTraceExporter_InvalidName(t *testing.T) {
	te, err := NewTraceExporter(nil, newTraceDataPusher(0, nil))
	require.Nil(t, te)
	require.Equal(t, errNilConfig, err)
}

func TestTraceExporter_NilPushTraceData(t *testing.T) {
	te, err := NewTraceExporter(fakeTraceExporterConfig, nil)
	require.Nil(t, te)
	require.Equal(t, errNilPushTraceData, err)
}

func TestTraceExporter_Default(t *testing.T) {
	td := pdata.NewTraces()
	te, err := NewTraceExporter(fakeTraceExporterConfig, newTraceDataPusher(0, nil))
	assert.NotNil(t, te)
	assert.NoError(t, err)

	assert.Nil(t, te.ConsumeTraces(context.Background(), td))
	assert.Nil(t, te.Shutdown(context.Background()))
}

func TestTraceExporter_Default_ReturnError(t *testing.T) {
	td := pdata.NewTraces()
	want := errors.New("my_error")
	te, err := NewTraceExporter(fakeTraceExporterConfig, newTraceDataPusher(0, want))
	require.Nil(t, err)
	require.NotNil(t, te)

	err = te.ConsumeTraces(context.Background(), td)
	require.Equalf(t, want, err, "ConsumeTraceData returns: Want %v Got %v", want, err)
}

func TestTraceExporter_WithRecordMetrics(t *testing.T) {
	te, err := NewTraceExporter(fakeTraceExporterConfig, newTraceDataPusher(0, nil))
	require.Nil(t, err)
	require.NotNil(t, te)

	checkRecordedMetricsForTraceExporter(t, te, nil, 0)
}

func TestTraceExporter_WithRecordMetrics_NonZeroDropped(t *testing.T) {
	te, err := NewTraceExporter(fakeTraceExporterConfig, newTraceDataPusher(1, nil))
	require.Nil(t, err)
	require.NotNil(t, te)

	checkRecordedMetricsForTraceExporter(t, te, nil, 1)
}

func TestTraceExporter_WithRecordMetrics_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	te, err := NewTraceExporter(fakeTraceExporterConfig, newTraceDataPusher(0, want))
	require.Nil(t, err)
	require.NotNil(t, te)

	checkRecordedMetricsForTraceExporter(t, te, want, 0)
}

func TestTraceExporter_WithSpan(t *testing.T) {
	te, err := NewTraceExporter(fakeTraceExporterConfig, newTraceDataPusher(0, nil))
	require.Nil(t, err)
	require.NotNil(t, te)

	checkWrapSpanForTraceExporter(t, te, nil, 1)
}

func TestTraceExporter_WithSpan_NonZeroDropped(t *testing.T) {
	te, err := NewTraceExporter(fakeTraceExporterConfig, newTraceDataPusher(1, nil))
	require.Nil(t, err)
	require.NotNil(t, te)

	checkWrapSpanForTraceExporter(t, te, nil, 1)
}

func TestTraceExporter_WithSpan_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	te, err := NewTraceExporter(fakeTraceExporterConfig, newTraceDataPusher(0, want))
	require.Nil(t, err)
	require.NotNil(t, te)

	checkWrapSpanForTraceExporter(t, te, want, 1)
}

func TestTraceExporter_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	te, err := NewTraceExporter(fakeTraceExporterConfig, newTraceDataPusher(0, nil), WithShutdown(shutdown))
	assert.NotNil(t, te)
	assert.NoError(t, err)

	assert.Nil(t, te.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestTraceExporter_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	te, err := NewTraceExporter(fakeTraceExporterConfig, newTraceDataPusher(0, nil), WithShutdown(shutdownErr))
	assert.NotNil(t, te)
	assert.NoError(t, err)

	assert.Equal(t, te.Shutdown(context.Background()), want)
}

func newTraceDataPusher(droppedSpans int, retError error) traceDataPusher {
	return func(ctx context.Context, td pdata.Traces) (int, error) {
		return droppedSpans, retError
	}
}

func checkRecordedMetricsForTraceExporter(t *testing.T, te component.TraceExporter, wantError error, droppedSpans int) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	td := testdata.GenerateTraceDataTwoSpansSameResource()
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		require.Equal(t, wantError, te.ConsumeTraces(context.Background(), td))
	}

	// TODO: When the new metrics correctly count partial dropped fix this.
	if wantError != nil {
		obsreporttest.CheckExporterTracesViews(t, fakeTraceExporterName, 0, int64(numBatches*td.SpanCount()))
	} else {
		obsreporttest.CheckExporterTracesViews(t, fakeTraceExporterName, int64(numBatches*td.SpanCount()), 0)
	}
}

func generateTraceTraffic(t *testing.T, te component.TraceExporter, numRequests int, wantError error) {
	td := pdata.NewTraces()
	rs := td.ResourceSpans()
	rs.Resize(1)
	rs.At(0).InstrumentationLibrarySpans().Resize(1)
	rs.At(0).InstrumentationLibrarySpans().At(0).Spans().Resize(1)
	ctx, span := trace.StartSpan(context.Background(), fakeTraceParentSpanName, trace.WithSampler(trace.AlwaysSample()))
	defer span.End()
	for i := 0; i < numRequests; i++ {
		require.Equal(t, wantError, te.ConsumeTraces(ctx, td))
	}
}

func checkWrapSpanForTraceExporter(t *testing.T, te component.TraceExporter, wantError error, numSpans int64) {
	ocSpansSaver := new(testOCTraceExporter)
	trace.RegisterExporter(ocSpansSaver)
	defer trace.UnregisterExporter(ocSpansSaver)

	const numRequests = 5
	generateTraceTraffic(t, te, numRequests, wantError)

	// Inspection time!
	ocSpansSaver.mu.Lock()
	defer ocSpansSaver.mu.Unlock()

	require.NotEqual(t, 0, len(ocSpansSaver.spanData), "No exported span data.")

	gotSpanData := ocSpansSaver.spanData
	require.Equal(t, numRequests+1, len(gotSpanData))

	parentSpan := gotSpanData[numRequests]
	require.Equalf(t, fakeTraceParentSpanName, parentSpan.Name, "SpanData %v", parentSpan)

	for _, sd := range gotSpanData[:numRequests] {
		require.Equalf(t, parentSpan.SpanContext.SpanID, sd.ParentSpanID, "Exporter span not a child\nSpanData %v", sd)
		require.Equalf(t, errToStatus(wantError), sd.Status, "SpanData %v", sd)

		sentSpans := numSpans
		var failedToSendSpans int64
		if wantError != nil {
			sentSpans = 0
			failedToSendSpans = numSpans
		}

		require.Equalf(t, sentSpans, sd.Attributes[obsreport.SentSpansKey], "SpanData %v", sd)
		require.Equalf(t, failedToSendSpans, sd.Attributes[obsreport.FailedToSendSpansKey], "SpanData %v", sd)
	}
}
