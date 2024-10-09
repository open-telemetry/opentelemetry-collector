// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/internal/queue"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/testdata"
)

const (
	fakeTraceParentSpanName = "fake_trace_parent_span_name"
)

var (
	fakeTracesName   = component.MustNewIDWithName("fake_traces_exporter", "with_name")
	fakeTracesConfig = struct{}{}
)

func TestTracesRequest(t *testing.T) {
	mr := newTracesRequest(testdata.GenerateTraces(1), nil)

	traceErr := consumererror.NewTraces(errors.New("some error"), ptrace.NewTraces())
	assert.EqualValues(t, newTracesRequest(ptrace.NewTraces(), nil), mr.(RequestErrorHandler).OnError(traceErr))
}

func TestTraces_InvalidName(t *testing.T) {
	te, err := NewTraces(context.Background(), exportertest.NewNopSettings(), nil, newTraceDataPusher(nil))
	require.Nil(t, te)
	require.Equal(t, errNilConfig, err)
}

func TestTraces_NilLogger(t *testing.T) {
	te, err := NewTraces(context.Background(), exporter.Settings{}, &fakeTracesConfig, newTraceDataPusher(nil))
	require.Nil(t, te)
	require.Equal(t, errNilLogger, err)
}

func TestTracesRequest_NilLogger(t *testing.T) {
	te, err := NewTracesRequest(context.Background(), exporter.Settings{}, (&internal.FakeRequestConverter{}).RequestFromTracesFunc)
	require.Nil(t, te)
	require.Equal(t, errNilLogger, err)
}

func TestTraces_NilPushTraceData(t *testing.T) {
	te, err := NewTraces(context.Background(), exportertest.NewNopSettings(), &fakeTracesConfig, nil)
	require.Nil(t, te)
	require.Equal(t, errNilPushTraceData, err)
}

func TestTracesRequest_NilTracesConverter(t *testing.T) {
	te, err := NewTracesRequest(context.Background(), exportertest.NewNopSettings(), nil)
	require.Nil(t, te)
	require.Equal(t, errNilTracesConverter, err)
}

func TestTraces_Default(t *testing.T) {
	td := ptrace.NewTraces()
	te, err := NewTraces(context.Background(), exportertest.NewNopSettings(), &fakeTracesConfig, newTraceDataPusher(nil))
	assert.NotNil(t, te)
	require.NoError(t, err)

	assert.Equal(t, consumer.Capabilities{MutatesData: false}, te.Capabilities())
	assert.NoError(t, te.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, te.ConsumeTraces(context.Background(), td))
	assert.NoError(t, te.Shutdown(context.Background()))
}

func TestTracesRequest_Default(t *testing.T) {
	td := ptrace.NewTraces()
	te, err := NewTracesRequest(context.Background(), exportertest.NewNopSettings(),
		(&internal.FakeRequestConverter{}).RequestFromTracesFunc)
	assert.NotNil(t, te)
	require.NoError(t, err)

	assert.Equal(t, consumer.Capabilities{MutatesData: false}, te.Capabilities())
	assert.NoError(t, te.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, te.ConsumeTraces(context.Background(), td))
	assert.NoError(t, te.Shutdown(context.Background()))
}

func TestTraces_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	te, err := NewTraces(context.Background(), exportertest.NewNopSettings(), &fakeTracesConfig, newTraceDataPusher(nil), WithCapabilities(capabilities))
	assert.NotNil(t, te)
	require.NoError(t, err)

	assert.Equal(t, capabilities, te.Capabilities())
}

func TestTracesRequest_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	te, err := NewTracesRequest(context.Background(), exportertest.NewNopSettings(),
		(&internal.FakeRequestConverter{}).RequestFromTracesFunc, WithCapabilities(capabilities))
	assert.NotNil(t, te)
	require.NoError(t, err)

	assert.Equal(t, capabilities, te.Capabilities())
}

func TestTraces_Default_ReturnError(t *testing.T) {
	td := ptrace.NewTraces()
	want := errors.New("my_error")
	te, err := NewTraces(context.Background(), exportertest.NewNopSettings(), &fakeTracesConfig, newTraceDataPusher(want))
	require.NoError(t, err)
	require.NotNil(t, te)

	err = te.ConsumeTraces(context.Background(), td)
	require.Equal(t, want, err)
}

func TestTracesRequest_Default_ConvertError(t *testing.T) {
	td := ptrace.NewTraces()
	want := errors.New("convert_error")
	te, err := NewTracesRequest(context.Background(), exportertest.NewNopSettings(),
		(&internal.FakeRequestConverter{TracesError: want}).RequestFromTracesFunc)
	require.NoError(t, err)
	require.NotNil(t, te)
	require.Equal(t, consumererror.NewPermanent(want), te.ConsumeTraces(context.Background(), td))
}

func TestTracesRequest_Default_ExportError(t *testing.T) {
	td := ptrace.NewTraces()
	want := errors.New("export_error")
	te, err := NewTracesRequest(context.Background(), exportertest.NewNopSettings(),
		(&internal.FakeRequestConverter{RequestError: want}).RequestFromTracesFunc)
	require.NoError(t, err)
	require.NotNil(t, te)
	require.Equal(t, want, te.ConsumeTraces(context.Background(), td))
}

func TestTraces_WithPersistentQueue(t *testing.T) {
	qCfg := NewDefaultQueueConfig()
	storageID := component.MustNewIDWithName("file_storage", "storage")
	qCfg.StorageID = &storageID
	rCfg := configretry.NewDefaultBackOffConfig()
	ts := consumertest.TracesSink{}
	set := exportertest.NewNopSettings()
	set.ID = component.MustNewIDWithName("test_traces", "with_persistent_queue")
	te, err := NewTraces(context.Background(), set, &fakeTracesConfig, ts.ConsumeTraces, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)

	host := &internal.MockHost{Ext: map[component.ID]component.Component{
		storageID: queue.NewMockStorageExtension(nil),
	}}
	require.NoError(t, te.Start(context.Background(), host))
	t.Cleanup(func() { require.NoError(t, te.Shutdown(context.Background())) })

	traces := testdata.GenerateTraces(2)
	require.NoError(t, te.ConsumeTraces(context.Background(), traces))
	require.Eventually(t, func() bool {
		return len(ts.AllTraces()) == 1 && ts.SpanCount() == 2
	}, 500*time.Millisecond, 10*time.Millisecond)
}

func TestTraces_WithRecordMetrics(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(fakeTracesName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := NewTraces(context.Background(), exporter.Settings{ID: fakeTracesName, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, &fakeTracesConfig, newTraceDataPusher(nil))
	require.NoError(t, err)
	require.NotNil(t, te)

	checkRecordedMetricsForTraces(t, tt, te, nil)
}

func TestTraces_pLogModifiedDownStream_WithRecordMetrics(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(fakeTracesName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := NewTraces(context.Background(), exporter.Settings{ID: fakeTracesName, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, &fakeTracesConfig, newTraceDataPusherModifiedDownstream(nil), WithCapabilities(consumer.Capabilities{MutatesData: true}))
	assert.NotNil(t, te)
	require.NoError(t, err)
	td := testdata.GenerateTraces(2)

	require.NoError(t, te.ConsumeTraces(context.Background(), td))
	assert.Equal(t, 0, td.SpanCount())
	require.NoError(t, tt.CheckExporterTraces(int64(2), 0))
}

func TestTracesRequest_WithRecordMetrics(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(fakeTracesName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := NewTracesRequest(context.Background(),
		exporter.Settings{ID: fakeTracesName, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		(&internal.FakeRequestConverter{}).RequestFromTracesFunc)
	require.NoError(t, err)
	require.NotNil(t, te)

	checkRecordedMetricsForTraces(t, tt, te, nil)
}

func TestTraces_WithRecordMetrics_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	tt, err := componenttest.SetupTelemetry(fakeTracesName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := NewTraces(context.Background(), exporter.Settings{ID: fakeTracesName, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, &fakeTracesConfig, newTraceDataPusher(want))
	require.NoError(t, err)
	require.NotNil(t, te)

	checkRecordedMetricsForTraces(t, tt, te, want)
}

func TestTracesRequest_WithRecordMetrics_RequestSenderError(t *testing.T) {
	want := errors.New("export_error")
	tt, err := componenttest.SetupTelemetry(fakeTracesName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := NewTracesRequest(context.Background(),
		exporter.Settings{ID: fakeTracesName, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		(&internal.FakeRequestConverter{RequestError: want}).RequestFromTracesFunc)
	require.NoError(t, err)
	require.NotNil(t, te)

	checkRecordedMetricsForTraces(t, tt, te, want)
}

func TestTraces_WithRecordEnqueueFailedMetrics(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(fakeTracesName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rCfg := configretry.NewDefaultBackOffConfig()
	qCfg := NewDefaultQueueConfig()
	qCfg.NumConsumers = 1
	qCfg.QueueSize = 2
	wantErr := errors.New("some-error")
	te, err := NewTraces(context.Background(), exporter.Settings{ID: fakeTracesName, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, &fakeTracesConfig, newTraceDataPusher(wantErr), WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	require.NotNil(t, te)

	td := testdata.GenerateTraces(2)
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		// errors are checked in the checkEnqueueFailedTracesStats function below.
		_ = te.ConsumeTraces(context.Background(), td)
	}

	// 2 batched must be in queue, and 5 batches (10 spans) rejected due to queue overflow
	require.NoError(t, tt.CheckExporterEnqueueFailedTraces(int64(10)))
}

func TestTraces_WithSpan(t *testing.T) {
	set := exportertest.NewNopSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	te, err := NewTraces(context.Background(), set, &fakeTracesConfig, newTraceDataPusher(nil))
	require.NoError(t, err)
	require.NotNil(t, te)

	checkWrapSpanForTraces(t, sr, set.TracerProvider.Tracer("test"), te, nil, 1)
}

func TestTracesRequest_WithSpan(t *testing.T) {
	set := exportertest.NewNopSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	te, err := NewTracesRequest(context.Background(), set, (&internal.FakeRequestConverter{}).RequestFromTracesFunc)
	require.NoError(t, err)
	require.NotNil(t, te)

	checkWrapSpanForTraces(t, sr, set.TracerProvider.Tracer("test"), te, nil, 1)
}

func TestTraces_WithSpan_ReturnError(t *testing.T) {
	set := exportertest.NewNopSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	want := errors.New("my_error")
	te, err := NewTraces(context.Background(), set, &fakeTracesConfig, newTraceDataPusher(want))
	require.NoError(t, err)
	require.NotNil(t, te)

	checkWrapSpanForTraces(t, sr, set.TracerProvider.Tracer("test"), te, want, 1)
}

func TestTracesRequest_WithSpan_ExportError(t *testing.T) {
	set := exportertest.NewNopSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	want := errors.New("export_error")
	te, err := NewTracesRequest(context.Background(), set, (&internal.FakeRequestConverter{RequestError: want}).RequestFromTracesFunc)
	require.NoError(t, err)
	require.NotNil(t, te)

	checkWrapSpanForTraces(t, sr, set.TracerProvider.Tracer("test"), te, want, 1)
}

func TestTraces_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	te, err := NewTraces(context.Background(), exportertest.NewNopSettings(), &fakeTracesConfig, newTraceDataPusher(nil), WithShutdown(shutdown))
	assert.NotNil(t, te)
	assert.NoError(t, err)

	assert.NoError(t, te.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, te.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestTracesRequest_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	te, err := NewTracesRequest(context.Background(), exportertest.NewNopSettings(),
		(&internal.FakeRequestConverter{}).RequestFromTracesFunc, WithShutdown(shutdown))
	assert.NotNil(t, te)
	assert.NoError(t, err)

	assert.NoError(t, te.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, te.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestTraces_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	te, err := NewTraces(context.Background(), exportertest.NewNopSettings(), &fakeTracesConfig, newTraceDataPusher(nil), WithShutdown(shutdownErr))
	assert.NotNil(t, te)
	assert.NoError(t, err)

	assert.NoError(t, te.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, want, te.Shutdown(context.Background()))
}

func TestTracesRequest_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	te, err := NewTracesRequest(context.Background(), exportertest.NewNopSettings(),
		(&internal.FakeRequestConverter{}).RequestFromTracesFunc, WithShutdown(shutdownErr))
	assert.NotNil(t, te)
	assert.NoError(t, err)

	assert.NoError(t, te.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, want, te.Shutdown(context.Background()))
}

func newTraceDataPusher(retError error) consumer.ConsumeTracesFunc {
	return func(context.Context, ptrace.Traces) error {
		return retError
	}
}

func newTraceDataPusherModifiedDownstream(retError error) consumer.ConsumeTracesFunc {
	return func(_ context.Context, trace ptrace.Traces) error {
		trace.ResourceSpans().MoveAndAppendTo(ptrace.NewResourceSpansSlice())
		return retError
	}
}

func checkRecordedMetricsForTraces(t *testing.T, tt componenttest.TestTelemetry, te exporter.Traces, wantError error) {
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

func checkWrapSpanForTraces(t *testing.T, sr *tracetest.SpanRecorder, tracer trace.Tracer,
	te exporter.Traces, wantError error, numSpans int64) { // nolint: unparam
	const numRequests = 5
	generateTraceTraffic(t, tracer, te, numRequests, wantError)

	// Inspection time!
	gotSpanData := sr.Ended()
	require.Len(t, gotSpanData, numRequests+1)

	parentSpan := gotSpanData[numRequests]
	require.Equalf(t, fakeTraceParentSpanName, parentSpan.Name(), "SpanData %v", parentSpan)

	for _, sd := range gotSpanData[:numRequests] {
		require.Equalf(t, parentSpan.SpanContext(), sd.Parent(), "Exporter span not a child\nSpanData %v", sd)
		internal.CheckStatus(t, sd, wantError)

		sentSpans := numSpans
		var failedToSendSpans int64
		if wantError != nil {
			sentSpans = 0
			failedToSendSpans = numSpans
		}
		require.Containsf(t, sd.Attributes(), attribute.KeyValue{Key: internal.SentSpansKey, Value: attribute.Int64Value(sentSpans)}, "SpanData %v", sd)
		require.Containsf(t, sd.Attributes(), attribute.KeyValue{Key: internal.FailedToSendSpansKey, Value: attribute.Int64Value(failedToSendSpans)}, "SpanData %v", sd)
	}
}
