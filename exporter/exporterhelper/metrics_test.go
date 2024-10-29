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
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/testdata"
)

const (
	fakeMetricsParentSpanName = "fake_metrics_parent_span_name"
)

var (
	fakeMetricsName   = component.MustNewIDWithName("fake_metrics_exporter", "with_name")
	fakeMetricsConfig = struct{}{}
)

func TestMetricsRequest(t *testing.T) {
	mr := newMetricsRequest(testdata.GenerateMetrics(1), nil)

	metricsErr := consumererror.NewMetrics(errors.New("some error"), pmetric.NewMetrics())
	assert.EqualValues(
		t,
		newMetricsRequest(pmetric.NewMetrics(), nil),
		mr.(RequestErrorHandler).OnError(metricsErr),
	)
}

func TestMetrics_NilConfig(t *testing.T) {
	me, err := NewMetrics(context.Background(), exportertest.NewNopSettings(), nil, newPushMetricsData(nil))
	require.Nil(t, me)
	require.Equal(t, errNilConfig, err)
}

func TestMetrics_NilLogger(t *testing.T) {
	me, err := NewMetrics(context.Background(), exporter.Settings{}, &fakeMetricsConfig, newPushMetricsData(nil))
	require.Nil(t, me)
	require.Equal(t, errNilLogger, err)
}

func TestMetricsRequest_NilLogger(t *testing.T) {
	me, err := NewMetricsRequest(context.Background(), exporter.Settings{},
		(&internal.FakeRequestConverter{}).RequestFromMetricsFunc)
	require.Nil(t, me)
	require.Equal(t, errNilLogger, err)
}

func TestMetrics_NilPushMetricsData(t *testing.T) {
	me, err := NewMetrics(context.Background(), exportertest.NewNopSettings(), &fakeMetricsConfig, nil)
	require.Nil(t, me)
	require.Equal(t, errNilPushMetricsData, err)
}

func TestMetricsRequest_NilMetricsConverter(t *testing.T) {
	me, err := NewMetricsRequest(context.Background(), exportertest.NewNopSettings(), nil)
	require.Nil(t, me)
	require.Equal(t, errNilMetricsConverter, err)
}

func TestMetrics_Default(t *testing.T) {
	md := pmetric.NewMetrics()
	me, err := NewMetrics(context.Background(), exportertest.NewNopSettings(), &fakeMetricsConfig, newPushMetricsData(nil))
	require.NoError(t, err)
	assert.NotNil(t, me)

	assert.Equal(t, consumer.Capabilities{MutatesData: false}, me.Capabilities())
	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, me.ConsumeMetrics(context.Background(), md))
	assert.NoError(t, me.Shutdown(context.Background()))
}

func TestMetricsRequest_Default(t *testing.T) {
	md := pmetric.NewMetrics()
	me, err := NewMetricsRequest(context.Background(), exportertest.NewNopSettings(),
		(&internal.FakeRequestConverter{}).RequestFromMetricsFunc)
	require.NoError(t, err)
	assert.NotNil(t, me)

	assert.Equal(t, consumer.Capabilities{MutatesData: false}, me.Capabilities())
	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, me.ConsumeMetrics(context.Background(), md))
	assert.NoError(t, me.Shutdown(context.Background()))
}

func TestMetrics_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	me, err := NewMetrics(context.Background(), exportertest.NewNopSettings(), &fakeMetricsConfig, newPushMetricsData(nil), WithCapabilities(capabilities))
	require.NoError(t, err)
	assert.NotNil(t, me)

	assert.Equal(t, capabilities, me.Capabilities())
}

func TestMetricsRequest_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	me, err := NewMetricsRequest(context.Background(), exportertest.NewNopSettings(),
		(&internal.FakeRequestConverter{}).RequestFromMetricsFunc, WithCapabilities(capabilities))
	require.NoError(t, err)
	assert.NotNil(t, me)

	assert.Equal(t, capabilities, me.Capabilities())
}

func TestMetrics_Default_ReturnError(t *testing.T) {
	md := pmetric.NewMetrics()
	want := errors.New("my_error")
	me, err := NewMetrics(context.Background(), exportertest.NewNopSettings(), &fakeMetricsConfig, newPushMetricsData(want))
	require.NoError(t, err)
	require.NotNil(t, me)
	require.Equal(t, want, me.ConsumeMetrics(context.Background(), md))
}

func TestMetricsRequest_Default_ConvertError(t *testing.T) {
	md := pmetric.NewMetrics()
	want := errors.New("convert_error")
	me, err := NewMetricsRequest(context.Background(), exportertest.NewNopSettings(),
		(&internal.FakeRequestConverter{MetricsError: want}).RequestFromMetricsFunc)
	require.NoError(t, err)
	require.NotNil(t, me)
	require.Equal(t, consumererror.NewPermanent(want), me.ConsumeMetrics(context.Background(), md))
}

func TestMetricsRequest_Default_ExportError(t *testing.T) {
	md := pmetric.NewMetrics()
	want := errors.New("export_error")
	me, err := NewMetricsRequest(context.Background(), exportertest.NewNopSettings(),
		(&internal.FakeRequestConverter{RequestError: want}).RequestFromMetricsFunc)
	require.NoError(t, err)
	require.NotNil(t, me)
	require.Equal(t, want, me.ConsumeMetrics(context.Background(), md))
}

func TestMetrics_WithPersistentQueue(t *testing.T) {
	qCfg := NewDefaultQueueConfig()
	storageID := component.MustNewIDWithName("file_storage", "storage")
	qCfg.StorageID = &storageID
	rCfg := configretry.NewDefaultBackOffConfig()
	ms := consumertest.MetricsSink{}
	set := exportertest.NewNopSettings()
	set.ID = component.MustNewIDWithName("test_metrics", "with_persistent_queue")
	te, err := NewMetrics(context.Background(), set, &fakeMetricsConfig, ms.ConsumeMetrics, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)

	host := &internal.MockHost{Ext: map[component.ID]component.Component{
		storageID: queue.NewMockStorageExtension(nil),
	}}
	require.NoError(t, te.Start(context.Background(), host))
	t.Cleanup(func() { require.NoError(t, te.Shutdown(context.Background())) })

	metrics := testdata.GenerateMetrics(2)
	require.NoError(t, te.ConsumeMetrics(context.Background(), metrics))
	require.Eventually(t, func() bool {
		return len(ms.AllMetrics()) == 1 && ms.DataPointCount() == 4
	}, 500*time.Millisecond, 10*time.Millisecond)
}

func TestMetrics_WithRecordMetrics(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(fakeMetricsName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	me, err := NewMetrics(context.Background(), exporter.Settings{ID: fakeMetricsName, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, &fakeMetricsConfig, newPushMetricsData(nil))
	require.NoError(t, err)
	require.NotNil(t, me)

	checkRecordedMetricsForMetrics(t, tt, me, nil)
}

func TestMetrics_pMetricModifiedDownStream_WithRecordMetrics(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(fakeMetricsName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	me, err := NewMetrics(context.Background(), exporter.Settings{ID: fakeMetricsName, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, &fakeMetricsConfig, newPushMetricsDataModifiedDownstream(nil), WithCapabilities(consumer.Capabilities{MutatesData: true}))
	require.NoError(t, err)
	require.NotNil(t, me)
	md := testdata.GenerateMetrics(2)

	require.NoError(t, me.ConsumeMetrics(context.Background(), md))
	assert.Equal(t, 0, md.MetricCount())
	require.NoError(t, tt.CheckExporterMetrics(int64(4), 0))
}

func TestMetricsRequest_WithRecordMetrics(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(fakeMetricsName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	me, err := NewMetricsRequest(context.Background(),
		exporter.Settings{ID: fakeMetricsName, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		(&internal.FakeRequestConverter{}).RequestFromMetricsFunc)
	require.NoError(t, err)
	require.NotNil(t, me)

	checkRecordedMetricsForMetrics(t, tt, me, nil)
}

func TestMetrics_WithRecordMetrics_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	tt, err := componenttest.SetupTelemetry(fakeMetricsName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	me, err := NewMetrics(context.Background(), exporter.Settings{ID: fakeMetricsName, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, &fakeMetricsConfig, newPushMetricsData(want))
	require.NoError(t, err)
	require.NotNil(t, me)

	checkRecordedMetricsForMetrics(t, tt, me, want)
}

func TestMetricsRequest_WithRecordMetrics_ExportError(t *testing.T) {
	want := errors.New("my_error")
	tt, err := componenttest.SetupTelemetry(fakeMetricsName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	me, err := NewMetricsRequest(context.Background(),
		exporter.Settings{ID: fakeMetricsName, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		(&internal.FakeRequestConverter{RequestError: want}).RequestFromMetricsFunc)
	require.NoError(t, err)
	require.NotNil(t, me)

	checkRecordedMetricsForMetrics(t, tt, me, want)
}

func TestMetrics_WithRecordEnqueueFailedMetrics(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(fakeMetricsName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rCfg := configretry.NewDefaultBackOffConfig()
	qCfg := NewDefaultQueueConfig()
	qCfg.NumConsumers = 1
	qCfg.QueueSize = 2
	wantErr := errors.New("some-error")
	te, err := NewMetrics(context.Background(), exporter.Settings{ID: fakeMetricsName, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, &fakeMetricsConfig, newPushMetricsData(wantErr), WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	require.NotNil(t, te)

	md := testdata.GenerateMetrics(1)
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		// errors are checked in the checkExporterEnqueueFailedMetricsStats function below.
		_ = te.ConsumeMetrics(context.Background(), md)
	}

	// 2 batched must be in queue, and 10 metric points rejected due to queue overflow
	require.NoError(t, tt.CheckExporterEnqueueFailedMetrics(int64(10)))
}

func TestMetrics_WithSpan(t *testing.T) {
	set := exportertest.NewNopSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	me, err := NewMetrics(context.Background(), set, &fakeMetricsConfig, newPushMetricsData(nil))
	require.NoError(t, err)
	require.NotNil(t, me)
	checkWrapSpanForMetrics(t, sr, set.TracerProvider.Tracer("test"), me, nil, 2)
}

func TestMetricsRequest_WithSpan(t *testing.T) {
	set := exportertest.NewNopSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	me, err := NewMetricsRequest(context.Background(), set, (&internal.FakeRequestConverter{}).RequestFromMetricsFunc)
	require.NoError(t, err)
	require.NotNil(t, me)
	checkWrapSpanForMetrics(t, sr, set.TracerProvider.Tracer("test"), me, nil, 2)
}

func TestMetrics_WithSpan_ReturnError(t *testing.T) {
	set := exportertest.NewNopSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	want := errors.New("my_error")
	me, err := NewMetrics(context.Background(), set, &fakeMetricsConfig, newPushMetricsData(want))
	require.NoError(t, err)
	require.NotNil(t, me)
	checkWrapSpanForMetrics(t, sr, set.TracerProvider.Tracer("test"), me, want, 2)
}

func TestMetricsRequest_WithSpan_ExportError(t *testing.T) {
	set := exportertest.NewNopSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	want := errors.New("my_error")
	me, err := NewMetricsRequest(context.Background(), set, (&internal.FakeRequestConverter{RequestError: want}).RequestFromMetricsFunc)
	require.NoError(t, err)
	require.NotNil(t, me)
	checkWrapSpanForMetrics(t, sr, set.TracerProvider.Tracer("test"), me, want, 2)
}

func TestMetrics_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	me, err := NewMetrics(context.Background(), exportertest.NewNopSettings(), &fakeMetricsConfig, newPushMetricsData(nil), WithShutdown(shutdown))
	assert.NotNil(t, me)
	assert.NoError(t, err)

	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, me.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestMetricsRequest_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	me, err := NewMetricsRequest(context.Background(), exportertest.NewNopSettings(),
		(&internal.FakeRequestConverter{}).RequestFromMetricsFunc, WithShutdown(shutdown))
	assert.NotNil(t, me)
	assert.NoError(t, err)

	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, me.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestMetrics_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	me, err := NewMetrics(context.Background(), exportertest.NewNopSettings(), &fakeMetricsConfig, newPushMetricsData(nil), WithShutdown(shutdownErr))
	assert.NotNil(t, me)
	assert.NoError(t, err)

	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, want, me.Shutdown(context.Background()))
}

func TestMetricsRequest_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	me, err := NewMetricsRequest(context.Background(), exportertest.NewNopSettings(),
		(&internal.FakeRequestConverter{}).RequestFromMetricsFunc, WithShutdown(shutdownErr))
	assert.NotNil(t, me)
	assert.NoError(t, err)

	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, want, me.Shutdown(context.Background()))
}

func newPushMetricsData(retError error) consumer.ConsumeMetricsFunc {
	return func(_ context.Context, _ pmetric.Metrics) error {
		return retError
	}
}
func newPushMetricsDataModifiedDownstream(retError error) consumer.ConsumeMetricsFunc {
	return func(_ context.Context, metric pmetric.Metrics) error {
		metric.ResourceMetrics().MoveAndAppendTo(pmetric.NewResourceMetricsSlice())
		return retError
	}
}

func checkRecordedMetricsForMetrics(t *testing.T, tt componenttest.TestTelemetry, me exporter.Metrics, wantError error) {
	md := testdata.GenerateMetrics(2)
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		require.Equal(t, wantError, me.ConsumeMetrics(context.Background(), md))
	}

	// TODO: When the new metrics correctly count partial dropped fix this.
	numPoints := int64(numBatches * md.MetricCount() * 2) /* 2 points per metric*/
	if wantError != nil {
		require.NoError(t, tt.CheckExporterMetrics(0, numPoints))
	} else {
		require.NoError(t, tt.CheckExporterMetrics(numPoints, 0))
	}
}

func generateMetricsTraffic(t *testing.T, tracer trace.Tracer, me exporter.Metrics, numRequests int, wantError error) {
	md := testdata.GenerateMetrics(1)
	ctx, span := tracer.Start(context.Background(), fakeMetricsParentSpanName)
	defer span.End()
	for i := 0; i < numRequests; i++ {
		require.Equal(t, wantError, me.ConsumeMetrics(ctx, md))
	}
}

func checkWrapSpanForMetrics(t *testing.T, sr *tracetest.SpanRecorder, tracer trace.Tracer,
	me exporter.Metrics, wantError error, numMetricPoints int64) { // nolint: unparam
	const numRequests = 5
	generateMetricsTraffic(t, tracer, me, numRequests, wantError)

	// Inspection time!
	gotSpanData := sr.Ended()
	require.Len(t, gotSpanData, numRequests+1)

	parentSpan := gotSpanData[numRequests]
	require.Equalf(t, fakeMetricsParentSpanName, parentSpan.Name(), "SpanData %v", parentSpan)
	for _, sd := range gotSpanData[:numRequests] {
		require.Equalf(t, parentSpan.SpanContext(), sd.Parent(), "Exporter span not a child\nSpanData %v", sd)
		internal.CheckStatus(t, sd, wantError)

		sentMetricPoints := numMetricPoints
		var failedToSendMetricPoints int64
		if wantError != nil {
			sentMetricPoints = 0
			failedToSendMetricPoints = numMetricPoints
		}
		require.Containsf(t, sd.Attributes(), attribute.KeyValue{Key: internal.SentMetricPointsKey, Value: attribute.Int64Value(sentMetricPoints)}, "SpanData %v", sd)
		require.Containsf(t, sd.Attributes(), attribute.KeyValue{Key: internal.FailedToSendMetricPointsKey, Value: attribute.Int64Value(failedToSendMetricPoints)}, "SpanData %v", sd)
	}
}
