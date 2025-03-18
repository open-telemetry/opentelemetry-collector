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
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/oteltest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/internal/storagetest"
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
	mr := newMetricsRequest(testdata.GenerateMetrics(1))

	metricsErr := consumererror.NewMetrics(errors.New("some error"), pmetric.NewMetrics())
	assert.EqualValues(
		t,
		newMetricsRequest(pmetric.NewMetrics()),
		mr.(RequestErrorHandler).OnError(metricsErr),
	)
}

func TestMetrics_NilConfig(t *testing.T) {
	me, err := NewMetrics(context.Background(), exportertest.NewNopSettings(exportertest.NopType), nil, newPushMetricsData(nil))
	require.Nil(t, me)
	require.Equal(t, errNilConfig, err)
}

func TestMetrics_NilLogger(t *testing.T) {
	me, err := NewMetrics(context.Background(), exporter.Settings{}, &fakeMetricsConfig, newPushMetricsData(nil))
	require.Nil(t, me)
	require.Equal(t, errNilLogger, err)
}

func TestMetricsRequest_NilLogger(t *testing.T) {
	me, err := NewMetricsRequest(context.Background(), exporter.Settings{}, requesttest.RequestFromMetricsFunc(nil), requesttest.NoopPusherFunc)
	require.Nil(t, me)
	require.Equal(t, errNilLogger, err)
}

func TestMetrics_NilPushMetricsData(t *testing.T) {
	me, err := NewMetrics(context.Background(), exportertest.NewNopSettings(exportertest.NopType), &fakeMetricsConfig, nil)
	require.Nil(t, me)
	require.Equal(t, errNilPushMetrics, err)
}

func TestMetricsRequest_NilMetricsConverter(t *testing.T) {
	me, err := NewMetricsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType), nil, requesttest.NoopPusherFunc)
	require.Nil(t, me)
	require.Equal(t, errNilMetricsConverter, err)
}

func TestMetricsRequest_NilPushMetricsData(t *testing.T) {
	me, err := NewMetricsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType), requesttest.RequestFromMetricsFunc(nil), nil)
	require.Nil(t, me)
	require.Equal(t, errNilConsumeRequest, err)
}

func TestMetrics_Default(t *testing.T) {
	md := pmetric.NewMetrics()
	me, err := NewMetrics(context.Background(), exportertest.NewNopSettings(exportertest.NopType), &fakeMetricsConfig, newPushMetricsData(nil))
	require.NoError(t, err)
	assert.NotNil(t, me)

	assert.Equal(t, consumer.Capabilities{MutatesData: false}, me.Capabilities())
	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, me.ConsumeMetrics(context.Background(), md))
	assert.NoError(t, me.Shutdown(context.Background()))
}

func TestMetricsRequest_Default(t *testing.T) {
	md := pmetric.NewMetrics()
	me, err := NewMetricsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromMetricsFunc(nil), requesttest.NoopPusherFunc)
	require.NoError(t, err)
	assert.NotNil(t, me)

	assert.Equal(t, consumer.Capabilities{MutatesData: false}, me.Capabilities())
	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, me.ConsumeMetrics(context.Background(), md))
	assert.NoError(t, me.Shutdown(context.Background()))
}

func TestMetrics_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	me, err := NewMetrics(context.Background(), exportertest.NewNopSettings(exportertest.NopType), &fakeMetricsConfig, newPushMetricsData(nil), WithCapabilities(capabilities))
	require.NoError(t, err)
	assert.NotNil(t, me)

	assert.Equal(t, capabilities, me.Capabilities())
}

func TestMetricsRequest_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	me, err := NewMetricsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromMetricsFunc(nil), requesttest.NoopPusherFunc, WithCapabilities(capabilities))
	require.NoError(t, err)
	assert.NotNil(t, me)

	assert.Equal(t, capabilities, me.Capabilities())
}

func TestMetrics_Default_ReturnError(t *testing.T) {
	md := pmetric.NewMetrics()
	want := errors.New("my_error")
	me, err := NewMetrics(context.Background(), exportertest.NewNopSettings(exportertest.NopType), &fakeMetricsConfig, newPushMetricsData(want))
	require.NoError(t, err)
	require.NotNil(t, me)
	require.Equal(t, want, me.ConsumeMetrics(context.Background(), md))
}

func TestMetricsRequest_Default_ConvertError(t *testing.T) {
	md := pmetric.NewMetrics()
	want := errors.New("convert_error")
	me, err := NewMetricsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromMetricsFunc(want), requesttest.NoopPusherFunc)
	require.NoError(t, err)
	require.NotNil(t, me)
	require.Equal(t, consumererror.NewPermanent(want), me.ConsumeMetrics(context.Background(), md))
}

func TestMetricsRequest_Default_ExportError(t *testing.T) {
	md := pmetric.NewMetrics()
	want := errors.New("export_error")
	me, err := NewMetricsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromMetricsFunc(nil), requesttest.NewErrPusherFunc(want))
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
	set := exportertest.NewNopSettings(exportertest.NopType)
	set.ID = component.MustNewIDWithName("test_metrics", "with_persistent_queue")
	te, err := NewMetrics(context.Background(), set, &fakeMetricsConfig, ms.ConsumeMetrics, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)

	host := &internal.MockHost{Ext: map[component.ID]component.Component{
		storageID: storagetest.NewMockStorageExtension(nil),
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
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	me, err := NewMetrics(context.Background(), exporter.Settings{ID: fakeMetricsName, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, &fakeMetricsConfig, newPushMetricsData(nil))
	require.NoError(t, err)
	require.NotNil(t, me)

	checkRecordedMetricsForMetrics(t, tt, fakeMetricsName, me, nil)
}

func TestMetrics_pMetricModifiedDownStream_WithRecordMetrics(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	me, err := NewMetrics(context.Background(), exporter.Settings{ID: fakeMetricsName, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, &fakeMetricsConfig, newPushMetricsDataModifiedDownstream(nil), WithCapabilities(consumer.Capabilities{MutatesData: true}))
	require.NoError(t, err)
	require.NotNil(t, me)
	md := testdata.GenerateMetrics(2)

	require.NoError(t, me.ConsumeMetrics(context.Background(), md))
	assert.Equal(t, 0, md.MetricCount())

	metadatatest.AssertEqualExporterSentMetricPoints(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(internal.ExporterKey, fakeMetricsName.String())),
				Value: int64(4),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

func TestMetricsRequest_WithRecordMetrics(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	me, err := NewMetricsRequest(context.Background(),
		exporter.Settings{ID: fakeMetricsName, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		requesttest.RequestFromMetricsFunc(nil), requesttest.NoopPusherFunc)
	require.NoError(t, err)
	require.NotNil(t, me)

	checkRecordedMetricsForMetrics(t, tt, fakeMetricsName, me, nil)
}

func TestMetrics_WithRecordMetrics_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	me, err := NewMetrics(context.Background(), exporter.Settings{ID: fakeMetricsName, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, &fakeMetricsConfig, newPushMetricsData(want))
	require.NoError(t, err)
	require.NotNil(t, me)

	checkRecordedMetricsForMetrics(t, tt, fakeMetricsName, me, want)
}

func TestMetricsRequest_WithRecordMetrics_ExportError(t *testing.T) {
	want := errors.New("my_error")
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	me, err := NewMetricsRequest(context.Background(),
		exporter.Settings{ID: fakeMetricsName, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		requesttest.RequestFromMetricsFunc(nil), requesttest.NewErrPusherFunc(want))
	require.NoError(t, err)
	require.NotNil(t, me)

	checkRecordedMetricsForMetrics(t, tt, fakeMetricsName, me, want)
}

func TestMetrics_WithSpan(t *testing.T) {
	set := exportertest.NewNopSettings(exportertest.NopType)
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	me, err := NewMetrics(context.Background(), set, &fakeMetricsConfig, newPushMetricsData(nil))
	require.NoError(t, err)
	require.NotNil(t, me)
	checkWrapSpanForMetrics(t, sr, set.TracerProvider.Tracer("test"), me, nil)
}

func TestMetricsRequest_WithSpan(t *testing.T) {
	set := exportertest.NewNopSettings(exportertest.NopType)
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	me, err := NewMetricsRequest(context.Background(), set, requesttest.RequestFromMetricsFunc(nil), requesttest.NoopPusherFunc)
	require.NoError(t, err)
	require.NotNil(t, me)
	checkWrapSpanForMetrics(t, sr, set.TracerProvider.Tracer("test"), me, nil)
}

func TestMetrics_WithSpan_ReturnError(t *testing.T) {
	set := exportertest.NewNopSettings(exportertest.NopType)
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	want := errors.New("my_error")
	me, err := NewMetrics(context.Background(), set, &fakeMetricsConfig, newPushMetricsData(want))
	require.NoError(t, err)
	require.NotNil(t, me)
	checkWrapSpanForMetrics(t, sr, set.TracerProvider.Tracer("test"), me, want)
}

func TestMetricsRequest_WithSpan_ExportError(t *testing.T) {
	set := exportertest.NewNopSettings(exportertest.NopType)
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	want := errors.New("my_error")
	me, err := NewMetricsRequest(context.Background(), set, requesttest.RequestFromMetricsFunc(nil), requesttest.NewErrPusherFunc(want))
	require.NoError(t, err)
	require.NotNil(t, me)
	checkWrapSpanForMetrics(t, sr, set.TracerProvider.Tracer("test"), me, want)
}

func TestMetrics_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	me, err := NewMetrics(context.Background(), exportertest.NewNopSettings(exportertest.NopType), &fakeMetricsConfig, newPushMetricsData(nil), WithShutdown(shutdown))
	assert.NotNil(t, me)
	assert.NoError(t, err)

	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, me.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestMetricsRequest_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	me, err := NewMetricsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromMetricsFunc(nil), requesttest.NoopPusherFunc, WithShutdown(shutdown))
	assert.NotNil(t, me)
	assert.NoError(t, err)

	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, me.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestMetrics_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	me, err := NewMetrics(context.Background(), exportertest.NewNopSettings(exportertest.NopType), &fakeMetricsConfig, newPushMetricsData(nil), WithShutdown(shutdownErr))
	assert.NotNil(t, me)
	assert.NoError(t, err)

	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, want, me.Shutdown(context.Background()))
}

func TestMetricsRequest_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	me, err := NewMetricsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromMetricsFunc(nil), requesttest.NoopPusherFunc, WithShutdown(shutdownErr))
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

func checkRecordedMetricsForMetrics(t *testing.T, tt *componenttest.Telemetry, id component.ID, me exporter.Metrics, wantError error) {
	md := testdata.GenerateMetrics(2)
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		require.Equal(t, wantError, me.ConsumeMetrics(context.Background(), md))
	}

	// TODO: When the new metrics correctly count partial dropped fix this.
	numPoints := int64(numBatches * md.MetricCount() * 2) /* 2 points per metric*/
	if wantError != nil {
		metadatatest.AssertEqualExporterSendFailedMetricPoints(t, tt,
			[]metricdata.DataPoint[int64]{
				{
					Attributes: attribute.NewSet(
						attribute.String(internal.ExporterKey, id.String())),
					Value: numPoints,
				},
			}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	} else {
		metadatatest.AssertEqualExporterSentMetricPoints(t, tt,
			[]metricdata.DataPoint[int64]{
				{
					Attributes: attribute.NewSet(
						attribute.String(internal.ExporterKey, id.String())),
					Value: numPoints,
				},
			}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
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

func checkWrapSpanForMetrics(t *testing.T, sr *tracetest.SpanRecorder, tracer trace.Tracer, me exporter.Metrics, wantError error) {
	const numRequests = 5
	generateMetricsTraffic(t, tracer, me, numRequests, wantError)

	// Inspection time!
	gotSpanData := sr.Ended()
	require.Len(t, gotSpanData, numRequests+1)

	parentSpan := gotSpanData[numRequests]
	require.Equalf(t, fakeMetricsParentSpanName, parentSpan.Name(), "SpanData %v", parentSpan)
	for _, sd := range gotSpanData[:numRequests] {
		require.Equalf(t, parentSpan.SpanContext(), sd.Parent(), "Exporter span not a child\nSpanData %v", sd)
		oteltest.CheckStatus(t, sd, wantError)

		sentMetricPoints := int64(2)
		failedToSendMetricPoints := int64(0)
		if wantError != nil {
			sentMetricPoints = 0
			failedToSendMetricPoints = 2
		}
		require.Containsf(t, sd.Attributes(), attribute.KeyValue{Key: internal.ItemsSent, Value: attribute.Int64Value(sentMetricPoints)}, "SpanData %v", sd)
		require.Containsf(t, sd.Attributes(), attribute.KeyValue{Key: internal.ItemsFailed, Value: attribute.Int64Value(failedToSendMetricPoints)}, "SpanData %v", sd)
	}
}
