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
	le, err := NewLogs(context.Background(), exportertest.NewNopSettings(), nil, newPushLogsData(nil))
	require.Nil(t, le)
	require.Equal(t, errNilConfig, err)
}

func TestLogs_NilLogger(t *testing.T) {
	le, err := NewLogs(context.Background(), exporter.Settings{}, &fakeLogsConfig, newPushLogsData(nil))
	require.Nil(t, le)
	require.Equal(t, errNilLogger, err)
}

func TestLogsRequest_NilLogger(t *testing.T) {
	le, err := NewLogsRequest(context.Background(), exporter.Settings{}, (&internal.FakeRequestConverter{}).RequestFromLogsFunc)
	require.Nil(t, le)
	require.Equal(t, errNilLogger, err)
}

func TestLogs_NilPushLogsData(t *testing.T) {
	le, err := NewLogs(context.Background(), exportertest.NewNopSettings(), &fakeLogsConfig, nil)
	require.Nil(t, le)
	require.Equal(t, errNilPushLogsData, err)
}

func TestLogsRequest_NilLogsConverter(t *testing.T) {
	le, err := NewLogsRequest(context.Background(), exportertest.NewNopSettings(), nil)
	require.Nil(t, le)
	require.Equal(t, errNilLogsConverter, err)
}

func TestLogs_Default(t *testing.T) {
	ld := plog.NewLogs()
	le, err := NewLogs(context.Background(), exportertest.NewNopSettings(), &fakeLogsConfig, newPushLogsData(nil))
	assert.NotNil(t, le)
	require.NoError(t, err)

	assert.Equal(t, consumer.Capabilities{MutatesData: false}, le.Capabilities())
	assert.NoError(t, le.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, le.ConsumeLogs(context.Background(), ld))
	assert.NoError(t, le.Shutdown(context.Background()))
}

func TestLogsRequest_Default(t *testing.T) {
	ld := plog.NewLogs()
	le, err := NewLogsRequest(context.Background(), exportertest.NewNopSettings(),
		(&internal.FakeRequestConverter{}).RequestFromLogsFunc)
	assert.NotNil(t, le)
	require.NoError(t, err)

	assert.Equal(t, consumer.Capabilities{MutatesData: false}, le.Capabilities())
	assert.NoError(t, le.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, le.ConsumeLogs(context.Background(), ld))
	assert.NoError(t, le.Shutdown(context.Background()))
}

func TestLogs_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	le, err := NewLogs(context.Background(), exportertest.NewNopSettings(), &fakeLogsConfig, newPushLogsData(nil), WithCapabilities(capabilities))
	require.NoError(t, err)
	require.NotNil(t, le)

	assert.Equal(t, capabilities, le.Capabilities())
}

func TestLogsRequest_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	le, err := NewLogsRequest(context.Background(), exportertest.NewNopSettings(),
		(&internal.FakeRequestConverter{}).RequestFromLogsFunc, WithCapabilities(capabilities))
	require.NoError(t, err)
	require.NotNil(t, le)

	assert.Equal(t, capabilities, le.Capabilities())
}

func TestLogs_Default_ReturnError(t *testing.T) {
	ld := plog.NewLogs()
	want := errors.New("my_error")
	le, err := NewLogs(context.Background(), exportertest.NewNopSettings(), &fakeLogsConfig, newPushLogsData(want))
	require.NoError(t, err)
	require.NotNil(t, le)
	require.Equal(t, want, le.ConsumeLogs(context.Background(), ld))
}

func TestLogsRequest_Default_ConvertError(t *testing.T) {
	ld := plog.NewLogs()
	want := errors.New("convert_error")
	le, err := NewLogsRequest(context.Background(), exportertest.NewNopSettings(),
		(&internal.FakeRequestConverter{LogsError: want}).RequestFromLogsFunc)
	require.NoError(t, err)
	require.NotNil(t, le)
	require.Equal(t, consumererror.NewPermanent(want), le.ConsumeLogs(context.Background(), ld))
}

func TestLogsRequest_Default_ExportError(t *testing.T) {
	ld := plog.NewLogs()
	want := errors.New("export_error")
	le, err := NewLogsRequest(context.Background(), exportertest.NewNopSettings(),
		(&internal.FakeRequestConverter{RequestError: want}).RequestFromLogsFunc)
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
	set := exportertest.NewNopSettings()
	set.ID = component.MustNewIDWithName("test_logs", "with_persistent_queue")
	te, err := NewLogs(context.Background(), set, &fakeLogsConfig, ts.ConsumeLogs, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)

	host := &internal.MockHost{Ext: map[component.ID]component.Component{
		storageID: queue.NewMockStorageExtension(nil),
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
	tt, err := componenttest.SetupTelemetry(fakeLogsName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	le, err := NewLogs(context.Background(), exporter.Settings{ID: fakeLogsName, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, &fakeLogsConfig, newPushLogsData(nil))
	require.NoError(t, err)
	require.NotNil(t, le)

	checkRecordedMetricsForLogs(t, tt, le, nil)
}

func TestLogs_pLogModifiedDownStream_WithRecordMetrics(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(fakeLogsName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	le, err := NewLogs(context.Background(), exporter.Settings{ID: fakeLogsName, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, &fakeLogsConfig, newPushLogsDataModifiedDownstream(nil), WithCapabilities(consumer.Capabilities{MutatesData: true}))
	assert.NotNil(t, le)
	require.NoError(t, err)
	ld := testdata.GenerateLogs(2)

	require.NoError(t, le.ConsumeLogs(context.Background(), ld))
	assert.Equal(t, 0, ld.LogRecordCount())
	require.NoError(t, tt.CheckExporterLogs(int64(2), 0))
}

func TestLogsRequest_WithRecordMetrics(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(fakeLogsName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	le, err := NewLogsRequest(context.Background(),
		exporter.Settings{ID: fakeLogsName, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		(&internal.FakeRequestConverter{}).RequestFromLogsFunc)
	require.NoError(t, err)
	require.NotNil(t, le)

	checkRecordedMetricsForLogs(t, tt, le, nil)
}

func TestLogs_WithRecordMetrics_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	tt, err := componenttest.SetupTelemetry(fakeLogsName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	le, err := NewLogs(context.Background(), exporter.Settings{ID: fakeLogsName, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, &fakeLogsConfig, newPushLogsData(want))
	require.NoError(t, err)
	require.NotNil(t, le)

	checkRecordedMetricsForLogs(t, tt, le, want)
}

func TestLogsRequest_WithRecordMetrics_ExportError(t *testing.T) {
	want := errors.New("export_error")
	tt, err := componenttest.SetupTelemetry(fakeLogsName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	le, err := NewLogsRequest(context.Background(), exporter.Settings{ID: fakeLogsName, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		(&internal.FakeRequestConverter{RequestError: want}).RequestFromLogsFunc)
	require.NoError(t, err)
	require.NotNil(t, le)

	checkRecordedMetricsForLogs(t, tt, le, want)
}

func TestLogs_WithRecordEnqueueFailedMetrics(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(fakeLogsName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rCfg := configretry.NewDefaultBackOffConfig()
	qCfg := NewDefaultQueueConfig()
	qCfg.NumConsumers = 1
	qCfg.QueueSize = 2
	wantErr := errors.New("some-error")
	te, err := NewLogs(context.Background(), exporter.Settings{ID: fakeLogsName, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, &fakeLogsConfig, newPushLogsData(wantErr), WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	require.NotNil(t, te)

	md := testdata.GenerateLogs(3)
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		// errors are checked in the checkEnqueueFailedLogsStats function below.
		_ = te.ConsumeLogs(context.Background(), md)
	}

	// 2 batched must be in queue, and 5 batches (15 log records) rejected due to queue overflow
	require.NoError(t, tt.CheckExporterEnqueueFailedLogs(int64(15)))
}

func TestLogs_WithSpan(t *testing.T) {
	set := exportertest.NewNopSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	le, err := NewLogs(context.Background(), set, &fakeLogsConfig, newPushLogsData(nil))
	require.NoError(t, err)
	require.NotNil(t, le)
	checkWrapSpanForLogs(t, sr, set.TracerProvider.Tracer("test"), le, nil, 1)
}

func TestLogsRequest_WithSpan(t *testing.T) {
	set := exportertest.NewNopSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	le, err := NewLogsRequest(context.Background(), set, (&internal.FakeRequestConverter{}).RequestFromLogsFunc)
	require.NoError(t, err)
	require.NotNil(t, le)
	checkWrapSpanForLogs(t, sr, set.TracerProvider.Tracer("test"), le, nil, 1)
}

func TestLogs_WithSpan_ReturnError(t *testing.T) {
	set := exportertest.NewNopSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	want := errors.New("my_error")
	le, err := NewLogs(context.Background(), set, &fakeLogsConfig, newPushLogsData(want))
	require.NoError(t, err)
	require.NotNil(t, le)
	checkWrapSpanForLogs(t, sr, set.TracerProvider.Tracer("test"), le, want, 1)
}

func TestLogsRequest_WithSpan_ReturnError(t *testing.T) {
	set := exportertest.NewNopSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	want := errors.New("my_error")
	le, err := NewLogsRequest(context.Background(), set, (&internal.FakeRequestConverter{RequestError: want}).RequestFromLogsFunc)
	require.NoError(t, err)
	require.NotNil(t, le)
	checkWrapSpanForLogs(t, sr, set.TracerProvider.Tracer("test"), le, want, 1)
}

func TestLogs_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	le, err := NewLogs(context.Background(), exportertest.NewNopSettings(), &fakeLogsConfig, newPushLogsData(nil), WithShutdown(shutdown))
	assert.NotNil(t, le)
	assert.NoError(t, err)

	assert.NoError(t, le.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestLogsRequest_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	le, err := NewLogsRequest(context.Background(), exportertest.NewNopSettings(),
		(&internal.FakeRequestConverter{}).RequestFromLogsFunc, WithShutdown(shutdown))
	assert.NotNil(t, le)
	assert.NoError(t, err)

	assert.NoError(t, le.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestLogs_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	le, err := NewLogs(context.Background(), exportertest.NewNopSettings(), &fakeLogsConfig, newPushLogsData(nil), WithShutdown(shutdownErr))
	assert.NotNil(t, le)
	require.NoError(t, err)

	assert.Equal(t, want, le.Shutdown(context.Background()))
}

func TestLogsRequest_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	le, err := NewLogsRequest(context.Background(), exportertest.NewNopSettings(),
		(&internal.FakeRequestConverter{}).RequestFromLogsFunc, WithShutdown(shutdownErr))
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

func checkRecordedMetricsForLogs(t *testing.T, tt componenttest.TestTelemetry, le exporter.Logs, wantError error) {
	ld := testdata.GenerateLogs(2)
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		require.Equal(t, wantError, le.ConsumeLogs(context.Background(), ld))
	}

	// TODO: When the new metrics correctly count partial dropped fix this.
	if wantError != nil {
		require.NoError(t, tt.CheckExporterLogs(0, int64(numBatches*ld.LogRecordCount())))
	} else {
		require.NoError(t, tt.CheckExporterLogs(int64(numBatches*ld.LogRecordCount()), 0))
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

func checkWrapSpanForLogs(t *testing.T, sr *tracetest.SpanRecorder, tracer trace.Tracer, le exporter.Logs,
	wantError error, numLogRecords int64) { // nolint: unparam
	const numRequests = 5
	generateLogsTraffic(t, tracer, le, numRequests, wantError)

	// Inspection time!
	gotSpanData := sr.Ended()
	require.Len(t, gotSpanData, numRequests+1)

	parentSpan := gotSpanData[numRequests]
	require.Equalf(t, fakeLogsParentSpanName, parentSpan.Name(), "SpanData %v", parentSpan)
	for _, sd := range gotSpanData[:numRequests] {
		require.Equalf(t, parentSpan.SpanContext(), sd.Parent(), "Exporter span not a child\nSpanData %v", sd)
		internal.CheckStatus(t, sd, wantError)

		sentLogRecords := numLogRecords
		var failedToSendLogRecords int64
		if wantError != nil {
			sentLogRecords = 0
			failedToSendLogRecords = numLogRecords
		}
		require.Containsf(t, sd.Attributes(), attribute.KeyValue{Key: internal.SentLogRecordsKey, Value: attribute.Int64Value(sentLogRecords)}, "SpanData %v", sd)
		require.Containsf(t, sd.Attributes(), attribute.KeyValue{Key: internal.FailedToSendLogRecordsKey, Value: attribute.Int64Value(failedToSendLogRecords)}, "SpanData %v", sd)
	}
}
