// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const (
	fakeTraceParentSpanName = "fake_trace_parent_span_name"
)

var (
	fakeTracesExporterName   = component.NewIDWithName("fake_traces_exporter", "with_name")
	fakeTracesExporterConfig = struct{}{}
)

func TestTracesRequest(t *testing.T) {
	mr := newTracesRequest(context.Background(), testdata.GenerateTraces(1), nil)

	traceErr := consumererror.NewTraces(errors.New("some error"), ptrace.NewTraces())
	assert.EqualValues(t, newTracesRequest(context.Background(), ptrace.NewTraces(), nil), mr.OnError(traceErr))
}

func TestTracesExporter_InvalidName(t *testing.T) {
	te, err := NewTracesExporter(context.Background(), exportertest.NewNopCreateSettings(), nil, newTraceDataPusher(nil))
	require.Nil(t, te)
	require.Equal(t, errNilConfig, err)
}

func TestTracesExporter_NilLogger(t *testing.T) {
	te, err := NewTracesExporter(context.Background(), exporter.CreateSettings{}, &fakeTracesExporterConfig, newTraceDataPusher(nil))
	require.Nil(t, te)
	require.Equal(t, errNilLogger, err)
}

func TestTracesExporter_NilPushTraceData(t *testing.T) {
	te, err := NewTracesExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeTracesExporterConfig, nil)
	require.Nil(t, te)
	require.Equal(t, errNilPushTraceData, err)
}

func TestTracesExporter_Default(t *testing.T) {
	td := ptrace.NewTraces()
	te, err := NewTracesExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeTracesExporterConfig, newTraceDataPusher(nil))
	assert.NotNil(t, te)
	assert.NoError(t, err)

	assert.Equal(t, consumer.Capabilities{MutatesData: false}, te.Capabilities())
	assert.NoError(t, te.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, te.ConsumeTraces(context.Background(), td))
	assert.NoError(t, te.Shutdown(context.Background()))
}

func TestTracesExporter_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	te, err := NewTracesExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeTracesExporterConfig, newTraceDataPusher(nil), WithCapabilities(capabilities))
	assert.NotNil(t, te)
	assert.NoError(t, err)

	assert.Equal(t, capabilities, te.Capabilities())
}

func TestTracesExporter_Default_ReturnError(t *testing.T) {
	td := ptrace.NewTraces()
	want := errors.New("my_error")
	te, err := NewTracesExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeTracesExporterConfig, newTraceDataPusher(want))
	require.NoError(t, err)
	require.NotNil(t, te)

	err = te.ConsumeTraces(context.Background(), td)
	require.Equal(t, want, err)
}

func TestTracesExporter_WithRecordMetrics(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(fakeTracesExporterName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := NewTracesExporter(context.Background(), tt.ToExporterCreateSettings(), &fakeTracesExporterConfig, newTraceDataPusher(nil))
	require.NoError(t, err)
	require.NotNil(t, te)

	checkRecordedMetricsForTracesExporter(t, tt, te, nil)
}

func TestTracesExporter_WithRecordMetrics_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	tt, err := obsreporttest.SetupTelemetry(fakeTracesExporterName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := NewTracesExporter(context.Background(), tt.ToExporterCreateSettings(), &fakeTracesExporterConfig, newTraceDataPusher(want))
	require.NoError(t, err)
	require.NotNil(t, te)

	checkRecordedMetricsForTracesExporter(t, tt, te, want)
}

func TestTracesExporter_WithRecordEnqueueFailedMetrics(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(fakeTracesExporterName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rCfg := NewDefaultRetrySettings()
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 1
	qCfg.QueueSize = 2
	wantErr := errors.New("some-error")
	te, err := NewTracesExporter(context.Background(), tt.ToExporterCreateSettings(), &fakeTracesExporterConfig, newTraceDataPusher(wantErr), WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	require.NotNil(t, te)

	td := testdata.GenerateTraces(2)
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		// errors are checked in the checkExporterEnqueueFailedTracesStats function below.
		_ = te.ConsumeTraces(context.Background(), td)
	}

	// 2 batched must be in queue, and 5 batches (10 spans) rejected due to queue overflow
	checkExporterEnqueueFailedTracesStats(t, globalInstruments, fakeTracesExporterName, int64(10))
}

func TestTracesExporter_WithSpan(t *testing.T) {
	set := exportertest.NewNopCreateSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(trace.NewNoopTracerProvider())

	te, err := NewTracesExporter(context.Background(), set, &fakeTracesExporterConfig, newTraceDataPusher(nil))
	require.NoError(t, err)
	require.NotNil(t, te)

	checkWrapSpanForTracesExporter(t, sr, set.TracerProvider.Tracer("test"), te, nil, 1)
}

func TestTracesExporter_WithSpan_ReturnError(t *testing.T) {
	set := exportertest.NewNopCreateSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(trace.NewNoopTracerProvider())

	want := errors.New("my_error")
	te, err := NewTracesExporter(context.Background(), set, &fakeTracesExporterConfig, newTraceDataPusher(want))
	require.NoError(t, err)
	require.NotNil(t, te)

	checkWrapSpanForTracesExporter(t, sr, set.TracerProvider.Tracer("test"), te, want, 1)
}

func TestTracesExporter_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	te, err := NewTracesExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeTracesExporterConfig, newTraceDataPusher(nil), WithShutdown(shutdown))
	assert.NotNil(t, te)
	assert.NoError(t, err)

	assert.NoError(t, te.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, te.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestTracesExporter_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	te, err := NewTracesExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeTracesExporterConfig, newTraceDataPusher(nil), WithShutdown(shutdownErr))
	assert.NotNil(t, te)
	assert.NoError(t, err)

	assert.NoError(t, te.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, te.Shutdown(context.Background()), want)
}

func newTraceDataPusher(retError error) consumer.ConsumeTracesFunc {
	return func(ctx context.Context, td ptrace.Traces) error {
		return retError
	}
}

func checkRecordedMetricsForTracesExporter(t *testing.T, tt obsreporttest.TestTelemetry, te exporter.Traces, wantError error) {
	td := testdata.GenerateTraces(2)
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		require.Equal(t, wantError, te.ConsumeTraces(context.Background(), td))
	}

	// TODO: When the new metrics correctly count partial dropped fix this.
	if wantError != nil {
		require.NoError(t, tt.CheckExporterTraces(0, int64(numBatches*td.SpanCount())))
	} else {
		require.NoError(t, tt.CheckExporterTraces(int64(numBatches*td.SpanCount()), 0))
	}
}

func generateTraceTraffic(t *testing.T, tracer trace.Tracer, te exporter.Traces, numRequests int, wantError error) {
	td := ptrace.NewTraces()
	td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	ctx, span := tracer.Start(context.Background(), fakeTraceParentSpanName)
	defer span.End()
	for i := 0; i < numRequests; i++ {
		require.Equal(t, wantError, te.ConsumeTraces(ctx, td))
	}
}

func checkWrapSpanForTracesExporter(t *testing.T, sr *tracetest.SpanRecorder, tracer trace.Tracer, te exporter.Traces, wantError error, numSpans int64) {
	const numRequests = 5
	generateTraceTraffic(t, tracer, te, numRequests, wantError)

	// Inspection time!
	gotSpanData := sr.Ended()
	require.Equal(t, numRequests+1, len(gotSpanData))

	parentSpan := gotSpanData[numRequests]
	require.Equalf(t, fakeTraceParentSpanName, parentSpan.Name(), "SpanData %v", parentSpan)

	for _, sd := range gotSpanData[:numRequests] {
		require.Equalf(t, parentSpan.SpanContext(), sd.Parent(), "Exporter span not a child\nSpanData %v", sd)
		checkStatus(t, sd, wantError)

		sentSpans := numSpans
		var failedToSendSpans int64
		if wantError != nil {
			sentSpans = 0
			failedToSendSpans = numSpans
		}
		require.Containsf(t, sd.Attributes(), attribute.KeyValue{Key: obsmetrics.SentSpansKey, Value: attribute.Int64Value(sentSpans)}, "SpanData %v", sd)
		require.Containsf(t, sd.Attributes(), attribute.KeyValue{Key: obsmetrics.FailedToSendSpansKey, Value: attribute.Int64Value(failedToSendSpans)}, "SpanData %v", sd)
	}
}
