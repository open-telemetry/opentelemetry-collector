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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
)

const (
	fakeTracesExporterName  = "fake_traces_exporter/with_name"
	fakeTraceParentSpanName = "fake_trace_parent_span_name"
)

var (
	fakeTracesExporterConfig = config.NewExporterSettings(config.MustIDFromString(fakeTracesExporterName))
)

func TestTracesRequest(t *testing.T) {
	mr := newTracesRequest(context.Background(), testdata.GenerateTraceDataOneSpan(), nil)

	traceErr := consumererror.NewTraces(errors.New("some error"), testdata.GenerateTraceDataEmpty())
	assert.EqualValues(t, newTracesRequest(context.Background(), testdata.GenerateTraceDataEmpty(), nil), mr.onError(traceErr))
}

type testOCTracesExporter struct {
	mu       sync.Mutex
	spanData []*trace.SpanData
}

func (tote *testOCTracesExporter) ExportSpan(sd *trace.SpanData) {
	tote.mu.Lock()
	defer tote.mu.Unlock()

	tote.spanData = append(tote.spanData, sd)
}

func TestTracesExporter_InvalidName(t *testing.T) {
	te, err := NewTracesExporter(nil, zap.NewNop(), newTraceDataPusher(nil))
	require.Nil(t, te)
	require.Equal(t, errNilConfig, err)
}

func TestTracesExporter_NilLogger(t *testing.T) {
	te, err := NewTracesExporter(&fakeTracesExporterConfig, nil, newTraceDataPusher(nil))
	require.Nil(t, te)
	require.Equal(t, errNilLogger, err)
}

func TestTracesExporter_NilPushTraceData(t *testing.T) {
	te, err := NewTracesExporter(&fakeTracesExporterConfig, zap.NewNop(), nil)
	require.Nil(t, te)
	require.Equal(t, errNilPushTraceData, err)
}

func TestTracesExporter_Default(t *testing.T) {
	td := pdata.NewTraces()
	te, err := NewTracesExporter(&fakeTracesExporterConfig, zap.NewNop(), newTraceDataPusher(nil))
	assert.NotNil(t, te)
	assert.NoError(t, err)

	assert.Nil(t, te.ConsumeTraces(context.Background(), td))
	assert.Nil(t, te.Shutdown(context.Background()))
}

func TestTracesExporter_Default_ReturnError(t *testing.T) {
	td := pdata.NewTraces()
	want := errors.New("my_error")
	te, err := NewTracesExporter(&fakeTracesExporterConfig, zap.NewNop(), newTraceDataPusher(want))
	require.Nil(t, err)
	require.NotNil(t, te)

	err = te.ConsumeTraces(context.Background(), td)
	require.Equal(t, want, err)
}

func TestTracesExporter_WithRecordMetrics(t *testing.T) {
	te, err := NewTracesExporter(&fakeTracesExporterConfig, zap.NewNop(), newTraceDataPusher(nil))
	require.Nil(t, err)
	require.NotNil(t, te)

	checkRecordedMetricsForTracesExporter(t, te, nil)
}

func TestTracesExporter_WithRecordMetrics_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	te, err := NewTracesExporter(&fakeTracesExporterConfig, zap.NewNop(), newTraceDataPusher(want))
	require.Nil(t, err)
	require.NotNil(t, te)

	checkRecordedMetricsForTracesExporter(t, te, want)
}

func TestTracesExporter_WithSpan(t *testing.T) {
	te, err := NewTracesExporter(&fakeTracesExporterConfig, zap.NewNop(), newTraceDataPusher(nil))
	require.Nil(t, err)
	require.NotNil(t, te)

	checkWrapSpanForTracesExporter(t, te, nil, 1)
}

func TestTracesExporter_WithSpan_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	te, err := NewTracesExporter(&fakeTracesExporterConfig, zap.NewNop(), newTraceDataPusher(want))
	require.Nil(t, err)
	require.NotNil(t, te)

	checkWrapSpanForTracesExporter(t, te, want, 1)
}

func TestTracesExporter_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	te, err := NewTracesExporter(&fakeTracesExporterConfig, zap.NewNop(), newTraceDataPusher(nil), WithShutdown(shutdown))
	assert.NotNil(t, te)
	assert.NoError(t, err)

	assert.Nil(t, te.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestTracesExporter_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	te, err := NewTracesExporter(&fakeTracesExporterConfig, zap.NewNop(), newTraceDataPusher(nil), WithShutdown(shutdownErr))
	assert.NotNil(t, te)
	assert.NoError(t, err)

	assert.Equal(t, te.Shutdown(context.Background()), want)
}

func newTraceDataPusher(retError error) PushTraces {
	return func(ctx context.Context, td pdata.Traces) error {
		return retError
	}
}

func checkRecordedMetricsForTracesExporter(t *testing.T, te component.TracesExporter, wantError error) {
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
		obsreporttest.CheckExporterTraces(t, fakeTracesExporterName, 0, int64(numBatches*td.SpanCount()))
	} else {
		obsreporttest.CheckExporterTraces(t, fakeTracesExporterName, int64(numBatches*td.SpanCount()), 0)
	}
}

func generateTraceTraffic(t *testing.T, te component.TracesExporter, numRequests int, wantError error) {
	td := pdata.NewTraces()
	td.ResourceSpans().AppendEmpty().InstrumentationLibrarySpans().AppendEmpty().Spans().AppendEmpty()
	ctx, span := trace.StartSpan(context.Background(), fakeTraceParentSpanName, trace.WithSampler(trace.AlwaysSample()))
	defer span.End()
	for i := 0; i < numRequests; i++ {
		require.Equal(t, wantError, te.ConsumeTraces(ctx, td))
	}
}

func checkWrapSpanForTracesExporter(t *testing.T, te component.TracesExporter, wantError error, numSpans int64) {
	ocSpansSaver := new(testOCTracesExporter)
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
