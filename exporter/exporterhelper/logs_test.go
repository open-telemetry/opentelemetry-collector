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
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/testdata"
)

const (
	fakeLogsParentSpanName = "fake_logs_parent_span_name"
)

var (
	fakeLogsName   = component.MustNewIDWithName("fake_logs_exporter", "with_name")
	fakeLogsConfig = struct{}{}
)

func TestLogsRequest(t *testing.T) {
	lr := newLogsRequest(testdata.GenerateLogs(1), nil)

	logErr := consumererror.NewLogs(errors.New("some error"), plog.NewLogs())
	assert.EqualValues(
		t,
		newLogsRequest(plog.NewLogs(), nil),
		lr.(RequestErrorHandler).OnError(logErr),
	)
}

func TestLogs_InvalidName(t *testing.T) {
	le, err := NewLogs(context.Background(), exportertest.NewNopSettings(exportertest.NopType), nil, newPushLogsData(nil))
	require.Nil(t, le)
	require.Equal(t, errNilConfig, err)
}

func TestLogs_NilLogger(t *testing.T) {
	le, err := NewLogs(context.Background(), exporter.Settings{}, &fakeLogsConfig, newPushLogsData(nil))
	require.Nil(t, le)
	require.Equal(t, errNilLogger, err)
}

func TestLogsRequest_NilLogger(t *testing.T) {
	le, err := NewLogsRequest(context.Background(), exporter.Settings{}, requesttest.RequestFromLogsFunc(nil))
	require.Nil(t, le)
	require.Equal(t, errNilLogger, err)
}

func TestLogs_NilPushLogsData(t *testing.T) {
	le, err := NewLogs(context.Background(), exportertest.NewNopSettings(exportertest.NopType), &fakeLogsConfig, nil)
	require.Nil(t, le)
	require.Equal(t, errNilPushLogsData, err)
}

func TestLogsRequest_NilLogsConverter(t *testing.T) {
	le, err := NewLogsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType), nil)
	require.Nil(t, le)
	require.Equal(t, errNilLogsConverter, err)
}

func TestLogs_Default(t *testing.T) {
	ld := plog.NewLogs()
	le, err := NewLogs(context.Background(), exportertest.NewNopSettings(exportertest.NopType), &fakeLogsConfig, newPushLogsData(nil))
	assert.NotNil(t, le)
	require.NoError(t, err)

	assert.Equal(t, consumer.Capabilities{MutatesData: false}, le.Capabilities())
	assert.NoError(t, le.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, le.ConsumeLogs(context.Background(), ld))
	assert.NoError(t, le.Shutdown(context.Background()))
}

func TestLogsRequest_Default(t *testing.T) {
	ld := plog.NewLogs()
	le, err := NewLogsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromLogsFunc(nil))
	assert.NotNil(t, le)
	require.NoError(t, err)

	assert.Equal(t, consumer.Capabilities{MutatesData: false}, le.Capabilities())
	assert.NoError(t, le.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, le.ConsumeLogs(context.Background(), ld))
	assert.NoError(t, le.Shutdown(context.Background()))
}

func TestLogs_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	le, err := NewLogs(context.Background(), exportertest.NewNopSettings(exportertest.NopType), &fakeLogsConfig, newPushLogsData(nil), WithCapabilities(capabilities))
	require.NoError(t, err)
	require.NotNil(t, le)

	assert.Equal(t, capabilities, le.Capabilities())
}

func TestLogsRequest_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	le, err := NewLogsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromLogsFunc(nil), WithCapabilities(capabilities))
	require.NoError(t, err)
	require.NotNil(t, le)

	assert.Equal(t, capabilities, le.Capabilities())
}

func TestLogs_Default_ReturnError(t *testing.T) {
	ld := plog.NewLogs()
	want := errors.New("my_error")
	le, err := NewLogs(context.Background(), exportertest.NewNopSettings(exportertest.NopType), &fakeLogsConfig, newPushLogsData(want))
	require.NoError(t, err)
	require.NotNil(t, le)
	require.Equal(t, want, le.ConsumeLogs(context.Background(), ld))
}

func TestLogsRequest_Default_ConvertError(t *testing.T) {
	ld := plog.NewLogs()
	want := errors.New("convert_error")
	le, err := NewLogsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		func(context.Context, plog.Logs) (Request, error) {
			return nil, want
		})
	require.NoError(t, err)
	require.NotNil(t, le)
	require.Equal(t, consumererror.NewPermanent(want), le.ConsumeLogs(context.Background(), ld))
}

func TestLogsRequest_Default_ExportError(t *testing.T) {
	ld := plog.NewLogs()
	want := errors.New("export_error")
	le, err := NewLogsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromLogsFunc(want))
	require.NoError(t, err)
	require.NotNil(t, le)
	require.Equal(t, want, le.ConsumeLogs(context.Background(), ld))
}

func TestLogs_WithPersistentQueue(t *testing.T) {
	qCfg := NewDefaultQueueConfig()
	storageID := component.MustNewIDWithName("file_storage", "storage")
	qCfg.StorageID = &storageID
	rCfg := configretry.NewDefaultBackOffConfig()
	ts := consumertest.LogsSink{}
	set := exportertest.NewNopSettings(exportertest.NopType)
	set.ID = component.MustNewIDWithName("test_logs", "with_persistent_queue")
	te, err := NewLogs(context.Background(), set, &fakeLogsConfig, ts.ConsumeLogs, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)

	host := &internal.MockHost{Ext: map[component.ID]component.Component{
		storageID: storagetest.NewMockStorageExtension(nil),
	}}
	require.NoError(t, te.Start(context.Background(), host))
	t.Cleanup(func() { require.NoError(t, te.Shutdown(context.Background())) })

	traces := testdata.GenerateLogs(2)
	require.NoError(t, te.ConsumeLogs(context.Background(), traces))
	require.Eventually(t, func() bool {
		return len(ts.AllLogs()) == 1 && ts.LogRecordCount() == 2
	}, 500*time.Millisecond, 10*time.Millisecond)
}

func TestLogs_WithRecordMetrics(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	le, err := NewLogs(context.Background(), exporter.Settings{ID: fakeLogsName, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, &fakeLogsConfig, newPushLogsData(nil))
	require.NoError(t, err)
	require.NotNil(t, le)

	checkRecordedMetricsForLogs(t, tt, fakeLogsName, le, nil)
}

func TestLogs_pLogModifiedDownStream_WithRecordMetrics(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	le, err := NewLogs(context.Background(), exporter.Settings{ID: fakeLogsName, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, &fakeLogsConfig, newPushLogsDataModifiedDownstream(nil), WithCapabilities(consumer.Capabilities{MutatesData: true}))
	assert.NotNil(t, le)
	require.NoError(t, err)
	ld := testdata.GenerateLogs(2)

	require.NoError(t, le.ConsumeLogs(context.Background(), ld))
	assert.Equal(t, 0, ld.LogRecordCount())

	metadatatest.AssertEqualExporterSentLogRecords(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String("exporter", fakeLogsName.String())),
				Value: int64(2),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

func TestLogsRequest_WithRecordMetrics(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	le, err := NewLogsRequest(context.Background(),
		exporter.Settings{ID: fakeLogsName, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		requesttest.RequestFromLogsFunc(nil))
	require.NoError(t, err)
	require.NotNil(t, le)

	checkRecordedMetricsForLogs(t, tt, fakeLogsName, le, nil)
}

func TestLogs_WithRecordMetrics_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	le, err := NewLogs(context.Background(), exporter.Settings{ID: fakeLogsName, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, &fakeLogsConfig, newPushLogsData(want))
	require.NoError(t, err)
	require.NotNil(t, le)

	checkRecordedMetricsForLogs(t, tt, fakeLogsName, le, want)
}

func TestLogsRequest_WithRecordMetrics_ExportError(t *testing.T) {
	want := errors.New("export_error")
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	le, err := NewLogsRequest(context.Background(), exporter.Settings{ID: fakeLogsName, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		requesttest.RequestFromLogsFunc(want))
	require.NoError(t, err)
	require.NotNil(t, le)

	checkRecordedMetricsForLogs(t, tt, fakeLogsName, le, want)
}

func TestLogs_WithSpan(t *testing.T) {
	set := exportertest.NewNopSettings(exportertest.NopType)
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	le, err := NewLogs(context.Background(), set, &fakeLogsConfig, newPushLogsData(nil))
	require.NoError(t, err)
	require.NotNil(t, le)
	checkWrapSpanForLogs(t, sr, set.TracerProvider.Tracer("test"), le, nil)
}

func TestLogsRequest_WithSpan(t *testing.T) {
	set := exportertest.NewNopSettings(exportertest.NopType)
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	le, err := NewLogsRequest(context.Background(), set, requesttest.RequestFromLogsFunc(nil))
	require.NoError(t, err)
	require.NotNil(t, le)
	checkWrapSpanForLogs(t, sr, set.TracerProvider.Tracer("test"), le, nil)
}

func TestLogs_WithSpan_ReturnError(t *testing.T) {
	set := exportertest.NewNopSettings(exportertest.NopType)
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	want := errors.New("my_error")
	le, err := NewLogs(context.Background(), set, &fakeLogsConfig, newPushLogsData(want))
	require.NoError(t, err)
	require.NotNil(t, le)
	checkWrapSpanForLogs(t, sr, set.TracerProvider.Tracer("test"), le, want)
}

func TestLogsRequest_WithSpan_ReturnError(t *testing.T) {
	set := exportertest.NewNopSettings(exportertest.NopType)
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	want := errors.New("my_error")
	le, err := NewLogsRequest(context.Background(), set, requesttest.RequestFromLogsFunc(want))
	require.NoError(t, err)
	require.NotNil(t, le)
	checkWrapSpanForLogs(t, sr, set.TracerProvider.Tracer("test"), le, want)
}

func TestLogs_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	le, err := NewLogs(context.Background(), exportertest.NewNopSettings(exportertest.NopType), &fakeLogsConfig, newPushLogsData(nil), WithShutdown(shutdown))
	assert.NotNil(t, le)
	assert.NoError(t, err)

	assert.NoError(t, le.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestLogsRequest_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	le, err := NewLogsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromLogsFunc(nil), WithShutdown(shutdown))
	assert.NotNil(t, le)
	assert.NoError(t, err)

	assert.NoError(t, le.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestLogs_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	le, err := NewLogs(context.Background(), exportertest.NewNopSettings(exportertest.NopType), &fakeLogsConfig, newPushLogsData(nil), WithShutdown(shutdownErr))
	assert.NotNil(t, le)
	require.NoError(t, err)

	assert.Equal(t, want, le.Shutdown(context.Background()))
}

func TestLogsRequest_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	le, err := NewLogsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromLogsFunc(nil), WithShutdown(shutdownErr))
	assert.NotNil(t, le)
	require.NoError(t, err)

	assert.Equal(t, want, le.Shutdown(context.Background()))
}

func newPushLogsDataModifiedDownstream(retError error) consumer.ConsumeLogsFunc {
	return func(_ context.Context, log plog.Logs) error {
		log.ResourceLogs().MoveAndAppendTo(plog.NewResourceLogsSlice())
		return retError
	}
}

func newPushLogsData(retError error) consumer.ConsumeLogsFunc {
	return func(_ context.Context, _ plog.Logs) error {
		return retError
	}
}

func checkRecordedMetricsForLogs(t *testing.T, tt *componenttest.Telemetry, id component.ID, le exporter.Logs, wantError error) {
	ld := testdata.GenerateLogs(2)
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		require.Equal(t, wantError, le.ConsumeLogs(context.Background(), ld))
	}

	// TODO: When the new metrics correctly count partial dropped fix this.
	if wantError != nil {
		metadatatest.AssertEqualExporterSendFailedLogRecords(t, tt,
			[]metricdata.DataPoint[int64]{
				{
					Attributes: attribute.NewSet(
						attribute.String("exporter", id.String())),
					Value: int64(numBatches * ld.LogRecordCount()),
				},
			}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	} else {
		metadatatest.AssertEqualExporterSentLogRecords(t, tt,
			[]metricdata.DataPoint[int64]{
				{
					Attributes: attribute.NewSet(
						attribute.String("exporter", id.String())),
					Value: int64(numBatches * ld.LogRecordCount()),
				},
			}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	}
}

func generateLogsTraffic(t *testing.T, tracer trace.Tracer, le exporter.Logs, numRequests int, wantError error) {
	ld := testdata.GenerateLogs(1)
	ctx, span := tracer.Start(context.Background(), fakeLogsParentSpanName)
	defer span.End()
	for i := 0; i < numRequests; i++ {
		require.Equal(t, wantError, le.ConsumeLogs(ctx, ld))
	}
}

func checkWrapSpanForLogs(t *testing.T, sr *tracetest.SpanRecorder, tracer trace.Tracer, le exporter.Logs, wantError error) {
	const numRequests = 5
	generateLogsTraffic(t, tracer, le, numRequests, wantError)

	// Inspection time!
	gotSpanData := sr.Ended()
	require.Len(t, gotSpanData, numRequests+1)

	parentSpan := gotSpanData[numRequests]
	require.Equalf(t, fakeLogsParentSpanName, parentSpan.Name(), "SpanData %v", parentSpan)
	for _, sd := range gotSpanData[:numRequests] {
		require.Equalf(t, parentSpan.SpanContext(), sd.Parent(), "Exporter span not a child\nSpanData %v", sd)
		oteltest.CheckStatus(t, sd, wantError)

		sentLogRecords := int64(1)
		failedToSendLogRecords := int64(0)
		if wantError != nil {
			sentLogRecords = 0
			failedToSendLogRecords = 1
		}
		require.Containsf(t, sd.Attributes(), attribute.KeyValue{Key: internal.ItemsSent, Value: attribute.Int64Value(sentLogRecords)}, "SpanData %v", sd)
		require.Containsf(t, sd.Attributes(), attribute.KeyValue{Key: internal.ItemsFailed, Value: attribute.Int64Value(failedToSendLogRecords)}, "SpanData %v", sd)
	}
}
