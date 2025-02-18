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
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
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
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadatatest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/internal/storagetest"
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
	te, err := NewTraces(context.Background(), exportertest.NewNopSettingsWithType(exportertest.NopType), nil, newTraceDataPusher(nil))
	require.Nil(t, te)
	require.Equal(t, errNilConfig, err)
}

func TestTraces_NilLogger(t *testing.T) {
	te, err := NewTraces(context.Background(), exporter.Settings{}, &fakeTracesConfig, newTraceDataPusher(nil))
	require.Nil(t, te)
	require.Equal(t, errNilLogger, err)
}

func TestTracesRequest_NilLogger(t *testing.T) {
	te, err := NewTracesRequest(context.Background(), exporter.Settings{}, requesttest.RequestFromTracesFunc(nil))
	require.Nil(t, te)
	require.Equal(t, errNilLogger, err)
}

func TestTraces_NilPushTraceData(t *testing.T) {
	te, err := NewTraces(context.Background(), exportertest.NewNopSettingsWithType(exportertest.NopType), &fakeTracesConfig, nil)
	require.Nil(t, te)
	require.Equal(t, errNilPushTraceData, err)
}

func TestTracesRequest_NilTracesConverter(t *testing.T) {
	te, err := NewTracesRequest(context.Background(), exportertest.NewNopSettingsWithType(exportertest.NopType), nil)
	require.Nil(t, te)
	require.Equal(t, errNilTracesConverter, err)
}

func TestTraces_Default(t *testing.T) {
	td := ptrace.NewTraces()
	te, err := NewTraces(context.Background(), exportertest.NewNopSettingsWithType(exportertest.NopType), &fakeTracesConfig, newTraceDataPusher(nil))
	assert.NotNil(t, te)
	require.NoError(t, err)

	assert.Equal(t, consumer.Capabilities{MutatesData: false}, te.Capabilities())
	assert.NoError(t, te.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, te.ConsumeTraces(context.Background(), td))
	assert.NoError(t, te.Shutdown(context.Background()))
}

func TestTracesRequest_Default(t *testing.T) {
	td := ptrace.NewTraces()
	te, err := NewTracesRequest(context.Background(), exportertest.NewNopSettingsWithType(exportertest.NopType),
		requesttest.RequestFromTracesFunc(nil))
	assert.NotNil(t, te)
	require.NoError(t, err)

	assert.Equal(t, consumer.Capabilities{MutatesData: false}, te.Capabilities())
	assert.NoError(t, te.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, te.ConsumeTraces(context.Background(), td))
	assert.NoError(t, te.Shutdown(context.Background()))
}

func TestTraces_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	te, err := NewTraces(context.Background(), exportertest.NewNopSettingsWithType(exportertest.NopType), &fakeTracesConfig, newTraceDataPusher(nil), WithCapabilities(capabilities))
	assert.NotNil(t, te)
	require.NoError(t, err)

	assert.Equal(t, capabilities, te.Capabilities())
}

func TestTracesRequest_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	te, err := NewTracesRequest(context.Background(), exportertest.NewNopSettingsWithType(exportertest.NopType),
		requesttest.RequestFromTracesFunc(nil), WithCapabilities(capabilities))
	assert.NotNil(t, te)
	require.NoError(t, err)

	assert.Equal(t, capabilities, te.Capabilities())
}

func TestTraces_Default_ReturnError(t *testing.T) {
	td := ptrace.NewTraces()
	want := errors.New("my_error")
	te, err := NewTraces(context.Background(), exportertest.NewNopSettingsWithType(exportertest.NopType), &fakeTracesConfig, newTraceDataPusher(want))
	require.NoError(t, err)
	require.NotNil(t, te)

	err = te.ConsumeTraces(context.Background(), td)
	require.Equal(t, want, err)
}

func TestTracesRequest_Default_ConvertError(t *testing.T) {
	td := ptrace.NewTraces()
	want := errors.New("convert_error")
	te, err := NewTracesRequest(context.Background(), exportertest.NewNopSettingsWithType(exportertest.NopType),
		func(context.Context, ptrace.Traces) (Request, error) {
			return nil, want
		})
	require.NoError(t, err)
	require.NotNil(t, te)
	require.Equal(t, consumererror.NewPermanent(want), te.ConsumeTraces(context.Background(), td))
}

func TestTracesRequest_Default_ExportError(t *testing.T) {
	td := ptrace.NewTraces()
	want := errors.New("export_error")
	te, err := NewTracesRequest(context.Background(), exportertest.NewNopSettingsWithType(exportertest.NopType),
		requesttest.RequestFromTracesFunc(want))
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
	set := exportertest.NewNopSettingsWithType(exportertest.NopType)
	set.ID = component.MustNewIDWithName("test_traces", "with_persistent_queue")
	te, err := NewTraces(context.Background(), set, &fakeTracesConfig, ts.ConsumeTraces, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)

	host := &internal.MockHost{Ext: map[component.ID]component.Component{
		storageID: storagetest.NewMockStorageExtension(nil),
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
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := NewTraces(context.Background(), exporter.Settings{ID: fakeTracesName, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, &fakeTracesConfig, newTraceDataPusher(nil))
	require.NoError(t, err)
	require.NotNil(t, te)

	checkRecordedMetricsForTraces(t, tt, fakeTracesName, te, nil)
}

func TestTraces_pLogModifiedDownStream_WithRecordMetrics(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := NewTraces(context.Background(), exporter.Settings{ID: fakeTracesName, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, &fakeTracesConfig, newTraceDataPusherModifiedDownstream(nil), WithCapabilities(consumer.Capabilities{MutatesData: true}))
	assert.NotNil(t, te)
	require.NoError(t, err)
	td := testdata.GenerateTraces(2)

	require.NoError(t, te.ConsumeTraces(context.Background(), td))
	assert.Equal(t, 0, td.SpanCount())

	metadatatest.AssertEqualExporterSentSpans(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(internal.ExporterKey, fakeTracesName.String())),
				Value: int64(2),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

func TestTracesRequest_WithRecordMetrics(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := NewTracesRequest(context.Background(),
		exporter.Settings{ID: fakeTracesName, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		requesttest.RequestFromTracesFunc(nil))
	require.NoError(t, err)
	require.NotNil(t, te)

	checkRecordedMetricsForTraces(t, tt, fakeTracesName, te, nil)
}

func TestTraces_WithRecordMetrics_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := NewTraces(context.Background(), exporter.Settings{ID: fakeTracesName, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, &fakeTracesConfig, newTraceDataPusher(want))
	require.NoError(t, err)
	require.NotNil(t, te)

	checkRecordedMetricsForTraces(t, tt, fakeTracesName, te, want)
}

func TestTracesRequest_WithRecordMetrics_RequestSenderError(t *testing.T) {
	want := errors.New("export_error")
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := NewTracesRequest(context.Background(),
		exporter.Settings{ID: fakeTracesName, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		requesttest.RequestFromTracesFunc(want))
	require.NoError(t, err)
	require.NotNil(t, te)

	checkRecordedMetricsForTraces(t, tt, fakeTracesName, te, want)
}

func TestTraces_WithSpan(t *testing.T) {
	set := exportertest.NewNopSettingsWithType(exportertest.NopType)
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	te, err := NewTraces(context.Background(), set, &fakeTracesConfig, newTraceDataPusher(nil))
	require.NoError(t, err)
	require.NotNil(t, te)

	checkWrapSpanForTraces(t, sr, set.TracerProvider.Tracer("test"), te, nil)
}

func TestTracesRequest_WithSpan(t *testing.T) {
	set := exportertest.NewNopSettingsWithType(exportertest.NopType)
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	te, err := NewTracesRequest(context.Background(), set, requesttest.RequestFromTracesFunc(nil))
	require.NoError(t, err)
	require.NotNil(t, te)

	checkWrapSpanForTraces(t, sr, set.TracerProvider.Tracer("test"), te, nil)
}

func TestTraces_WithSpan_ReturnError(t *testing.T) {
	set := exportertest.NewNopSettingsWithType(exportertest.NopType)
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	want := errors.New("my_error")
	te, err := NewTraces(context.Background(), set, &fakeTracesConfig, newTraceDataPusher(want))
	require.NoError(t, err)
	require.NotNil(t, te)

	checkWrapSpanForTraces(t, sr, set.TracerProvider.Tracer("test"), te, want)
}

func TestTracesRequest_WithSpan_ExportError(t *testing.T) {
	set := exportertest.NewNopSettingsWithType(exportertest.NopType)
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	want := errors.New("export_error")
	te, err := NewTracesRequest(context.Background(), set, requesttest.RequestFromTracesFunc(want))
	require.NoError(t, err)
	require.NotNil(t, te)

	checkWrapSpanForTraces(t, sr, set.TracerProvider.Tracer("test"), te, want)
}

func TestTraces_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	te, err := NewTraces(context.Background(), exportertest.NewNopSettingsWithType(exportertest.NopType), &fakeTracesConfig, newTraceDataPusher(nil), WithShutdown(shutdown))
	assert.NotNil(t, te)
	assert.NoError(t, err)

	assert.NoError(t, te.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, te.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestTracesRequest_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	te, err := NewTracesRequest(context.Background(), exportertest.NewNopSettingsWithType(exportertest.NopType),
		requesttest.RequestFromTracesFunc(nil), WithShutdown(shutdown))
	assert.NotNil(t, te)
	assert.NoError(t, err)

	assert.NoError(t, te.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, te.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestTraces_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	te, err := NewTraces(context.Background(), exportertest.NewNopSettingsWithType(exportertest.NopType), &fakeTracesConfig, newTraceDataPusher(nil), WithShutdown(shutdownErr))
	assert.NotNil(t, te)
	assert.NoError(t, err)

	assert.NoError(t, te.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, want, te.Shutdown(context.Background()))
}

func TestTracesRequest_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	te, err := NewTracesRequest(context.Background(), exportertest.NewNopSettingsWithType(exportertest.NopType),
		requesttest.RequestFromTracesFunc(nil), WithShutdown(shutdownErr))
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

func checkRecordedMetricsForTraces(t *testing.T, tt *componenttest.Telemetry, id component.ID, te exporter.Traces, wantError error) {
	td := testdata.GenerateTraces(2)
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		require.Equal(t, wantError, te.ConsumeTraces(context.Background(), td))
	}

	// TODO: When the new metrics correctly count partial dropped fix this.
	if wantError != nil {
		metadatatest.AssertEqualExporterSendFailedSpans(t, tt,
			[]metricdata.DataPoint[int64]{
				{
					Attributes: attribute.NewSet(
						attribute.String(internal.ExporterKey, id.String())),
					Value: int64(numBatches * td.SpanCount()),
				},
			}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	} else {
		metadatatest.AssertEqualExporterSentSpans(t, tt,
			[]metricdata.DataPoint[int64]{
				{
					Attributes: attribute.NewSet(
						attribute.String(internal.ExporterKey, id.String())),
					Value: int64(numBatches * td.SpanCount()),
				},
			}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
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

func checkWrapSpanForTraces(t *testing.T, sr *tracetest.SpanRecorder, tracer trace.Tracer, te exporter.Traces, wantError error) {
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

		sentSpans := int64(1)
		failedToSendSpans := int64(0)
		if wantError != nil {
			sentSpans = 0
			failedToSendSpans = 1
		}
		require.Containsf(t, sd.Attributes(), attribute.KeyValue{Key: internal.ItemsSent, Value: attribute.Int64Value(sentSpans)}, "SpanData %v", sd)
		require.Containsf(t, sd.Attributes(), attribute.KeyValue{Key: internal.ItemsFailed, Value: attribute.Int64Value(failedToSendSpans)}, "SpanData %v", sd)
	}
}
