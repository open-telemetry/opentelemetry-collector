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
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/hosttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadatatest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/oteltest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sendertest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/storagetest"
	"go.opentelemetry.io/collector/exporter/exportertest"
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

func TestMetrics_NilPushMetricsData(t *testing.T) {
	me, err := NewMetrics(context.Background(), exportertest.NewNopSettings(exportertest.NopType), &fakeMetricsConfig, nil)
	require.Nil(t, me)
	require.Equal(t, errNilPushMetrics, err)
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

func TestMetrics_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	me, err := NewMetrics(context.Background(), exportertest.NewNopSettings(exportertest.NopType), &fakeMetricsConfig, newPushMetricsData(nil), WithCapabilities(capabilities))
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

func TestMetrics_WithPersistentQueue(t *testing.T) {
	fgOrigReadState := queue.PersistRequestContextOnRead
	fgOrigWriteState := queue.PersistRequestContextOnWrite
	qCfg := NewDefaultQueueConfig()
	storageID := component.MustNewIDWithName("file_storage", "storage")
	qCfg.StorageID = &storageID
	set := exportertest.NewNopSettings(exportertest.NopType)
	set.ID = component.MustNewIDWithName("test_metrics", "with_persistent_queue")
	host := hosttest.NewHost(map[component.ID]component.Component{
		storageID: storagetest.NewMockStorageExtension(nil),
	})
	spanCtx := oteltest.FakeSpanContext(t)

	tests := []struct {
		name             string
		fgEnabledOnWrite bool
		fgEnabledOnRead  bool
		wantData         bool
		wantSpanCtx      bool
	}{
		{
			name:     "feature_gate_disabled_on_write_and_read",
			wantData: true,
		},
		{
			name:             "feature_gate_enabled_on_write_and_read",
			fgEnabledOnWrite: true,
			fgEnabledOnRead:  true,
			wantData:         true,
			wantSpanCtx:      true,
		},
		{
			name:            "feature_gate_disabled_on_write_enabled_on_read",
			wantData:        true,
			fgEnabledOnRead: true,
		},
		{
			name:             "feature_gate_enabled_on_write_disabled_on_read",
			fgEnabledOnWrite: true,
			wantData:         false, // going back from enabled to disabled feature gate isn't supported
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queue.PersistRequestContextOnRead = func() bool { return tt.fgEnabledOnRead }
			queue.PersistRequestContextOnWrite = func() bool { return tt.fgEnabledOnWrite }
			t.Cleanup(func() {
				queue.PersistRequestContextOnRead = fgOrigReadState
				queue.PersistRequestContextOnWrite = fgOrigWriteState
			})

			ms := consumertest.MetricsSink{}
			te, err := NewMetrics(context.Background(), set, &fakeMetricsConfig, ms.ConsumeMetrics, WithQueue(configoptional.Some(qCfg)))
			require.NoError(t, err)
			require.NoError(t, te.Start(context.Background(), host))
			t.Cleanup(func() { require.NoError(t, te.Shutdown(context.Background())) })

			traces := testdata.GenerateMetrics(2)
			require.NoError(t, te.ConsumeMetrics(trace.ContextWithSpanContext(context.Background(), spanCtx), traces))
			if tt.wantData {
				require.Eventually(t, func() bool {
					return len(ms.AllMetrics()) == 1 && ms.DataPointCount() == 4
				}, 500*time.Millisecond, 10*time.Millisecond)
			}

			// check that the span context is persisted if the feature gate is enabled
			if tt.wantSpanCtx {
				assert.Len(t, ms.Contexts(), 1)
				assert.Equal(t, spanCtx, trace.SpanContextFromContext(ms.Contexts()[0]))
			}
		})
	}
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

	me, err := internal.NewMetricsRequest(context.Background(),
		exporter.Settings{ID: fakeMetricsName, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		requesttest.RequestFromMetricsFunc(nil), sendertest.NewNopSenderFunc[request.Request]())
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

	me, err := internal.NewMetricsRequest(context.Background(),
		exporter.Settings{ID: fakeMetricsName, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		requesttest.RequestFromMetricsFunc(nil), sendertest.NewErrSenderFunc[request.Request](want))
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

	me, err := internal.NewMetricsRequest(context.Background(), set, requesttest.RequestFromMetricsFunc(nil), sendertest.NewNopSenderFunc[request.Request]())
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
	me, err := internal.NewMetricsRequest(context.Background(), set, requesttest.RequestFromMetricsFunc(nil), sendertest.NewErrSenderFunc[request.Request](want))
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

func TestMetrics_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	me, err := NewMetrics(context.Background(), exportertest.NewNopSettings(exportertest.NopType), &fakeMetricsConfig, newPushMetricsData(nil), WithShutdown(shutdownErr))
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
	for range numBatches {
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
	for range numRequests {
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
