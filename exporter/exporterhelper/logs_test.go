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
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
	"go.opentelemetry.io/collector/pdata/plog"
)

const (
	fakeLogsParentSpanName = "fake_logs_parent_span_name"
)

var (
	fakeLogsExporterName   = component.NewIDWithName("fake_logs_exporter", "with_name")
	fakeLogsExporterConfig = struct{}{}
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

func TestLogsExporter_InvalidName(t *testing.T) {
	le, err := NewLogsExporter(context.Background(), exportertest.NewNopCreateSettings(), nil, newPushLogsData(nil))
	require.Nil(t, le)
	require.Equal(t, errNilConfig, err)
}

func TestLogsExporter_NilLogger(t *testing.T) {
	le, err := NewLogsExporter(context.Background(), exporter.CreateSettings{}, &fakeLogsExporterConfig, newPushLogsData(nil))
	require.Nil(t, le)
	require.Equal(t, errNilLogger, err)
}

func TestLogsRequestExporter_NilLogger(t *testing.T) {
	le, err := NewLogsRequestExporter(context.Background(), exporter.CreateSettings{}, (&fakeRequestConverter{}).requestFromLogsFunc)
	require.Nil(t, le)
	require.Equal(t, errNilLogger, err)
}

func TestLogsExporter_NilPushLogsData(t *testing.T) {
	le, err := NewLogsExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeLogsExporterConfig, nil)
	require.Nil(t, le)
	require.Equal(t, errNilPushLogsData, err)
}

func TestLogsRequestExporter_NilLogsConverter(t *testing.T) {
	le, err := NewLogsRequestExporter(context.Background(), exportertest.NewNopCreateSettings(), nil)
	require.Nil(t, le)
	require.Equal(t, errNilLogsConverter, err)
}

func TestLogsExporter_Default(t *testing.T) {
	ld := plog.NewLogs()
	le, err := NewLogsExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeLogsExporterConfig, newPushLogsData(nil))
	assert.NotNil(t, le)
	assert.NoError(t, err)

	assert.Equal(t, consumer.Capabilities{MutatesData: false}, le.Capabilities())
	assert.NoError(t, le.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, le.ConsumeLogs(context.Background(), ld))
	assert.NoError(t, le.Shutdown(context.Background()))
}

func TestLogsRequestExporter_Default(t *testing.T) {
	ld := plog.NewLogs()
	le, err := NewLogsRequestExporter(context.Background(), exportertest.NewNopCreateSettings(),
		(&fakeRequestConverter{}).requestFromLogsFunc)
	assert.NotNil(t, le)
	assert.NoError(t, err)

	assert.Equal(t, consumer.Capabilities{MutatesData: false}, le.Capabilities())
	assert.NoError(t, le.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, le.ConsumeLogs(context.Background(), ld))
	assert.NoError(t, le.Shutdown(context.Background()))
}

func TestLogsExporter_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	le, err := NewLogsExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeLogsExporterConfig, newPushLogsData(nil), WithCapabilities(capabilities))
	require.NoError(t, err)
	require.NotNil(t, le)

	assert.Equal(t, capabilities, le.Capabilities())
}

func TestLogsRequestExporter_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	le, err := NewLogsRequestExporter(context.Background(), exportertest.NewNopCreateSettings(),
		(&fakeRequestConverter{}).requestFromLogsFunc, WithCapabilities(capabilities))
	require.NoError(t, err)
	require.NotNil(t, le)

	assert.Equal(t, capabilities, le.Capabilities())
}

func TestLogsExporter_Default_ReturnError(t *testing.T) {
	ld := plog.NewLogs()
	want := errors.New("my_error")
	le, err := NewLogsExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeLogsExporterConfig, newPushLogsData(want))
	require.NoError(t, err)
	require.NotNil(t, le)
	require.Equal(t, want, le.ConsumeLogs(context.Background(), ld))
}

func TestLogsRequestExporter_Default_ConvertError(t *testing.T) {
	ld := plog.NewLogs()
	want := errors.New("convert_error")
	le, err := NewLogsRequestExporter(context.Background(), exportertest.NewNopCreateSettings(),
		(&fakeRequestConverter{logsError: want}).requestFromLogsFunc)
	require.NoError(t, err)
	require.NotNil(t, le)
	require.Equal(t, consumererror.NewPermanent(want), le.ConsumeLogs(context.Background(), ld))
}

func TestLogsRequestExporter_Default_ExportError(t *testing.T) {
	ld := plog.NewLogs()
	want := errors.New("export_error")
	le, err := NewLogsRequestExporter(context.Background(), exportertest.NewNopCreateSettings(),
		(&fakeRequestConverter{requestError: want}).requestFromLogsFunc)
	require.NoError(t, err)
	require.NotNil(t, le)
	require.Equal(t, want, le.ConsumeLogs(context.Background(), ld))
}

func TestLogsExporter_WithPersistentQueue(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	storageID := component.NewIDWithName("file_storage", "storage")
	qCfg.StorageID = &storageID
	rCfg := configretry.NewDefaultBackOffConfig()
	ts := consumertest.LogsSink{}
	set := exportertest.NewNopCreateSettings()
	set.ID = component.NewIDWithName("test_logs", "with_persistent_queue")
	te, err := NewLogsExporter(context.Background(), set, &fakeLogsExporterConfig, ts.ConsumeLogs, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)

	host := &mockHost{ext: map[component.ID]component.Component{
		storageID: internal.NewMockStorageExtension(nil),
	}}
	require.NoError(t, te.Start(context.Background(), host))
	t.Cleanup(func() { require.NoError(t, te.Shutdown(context.Background())) })

	traces := testdata.GenerateLogs(2)
	require.NoError(t, te.ConsumeLogs(context.Background(), traces))
	require.Eventually(t, func() bool {
		return len(ts.AllLogs()) == 1 && ts.LogRecordCount() == 2
	}, 500*time.Millisecond, 10*time.Millisecond)
}

func TestLogsExporter_WithRecordMetrics(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(fakeLogsExporterName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	le, err := NewLogsExporter(context.Background(), exporter.CreateSettings{ID: fakeLogsExporterName, TelemetrySettings: tt.TelemetrySettings, BuildInfo: component.NewDefaultBuildInfo()}, &fakeLogsExporterConfig, newPushLogsData(nil))
	require.NoError(t, err)
	require.NotNil(t, le)

	checkRecordedMetricsForLogsExporter(t, tt, le, nil)
}

func TestLogsRequestExporter_WithRecordMetrics(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(fakeLogsExporterName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	le, err := NewLogsRequestExporter(context.Background(),
		exporter.CreateSettings{ID: fakeLogsExporterName, TelemetrySettings: tt.TelemetrySettings, BuildInfo: component.NewDefaultBuildInfo()},
		(&fakeRequestConverter{}).requestFromLogsFunc)
	require.NoError(t, err)
	require.NotNil(t, le)

	checkRecordedMetricsForLogsExporter(t, tt, le, nil)
}

func TestLogsExporter_WithRecordMetrics_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	tt, err := obsreporttest.SetupTelemetry(fakeLogsExporterName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	le, err := NewLogsExporter(context.Background(), exporter.CreateSettings{ID: fakeLogsExporterName, TelemetrySettings: tt.TelemetrySettings, BuildInfo: component.NewDefaultBuildInfo()}, &fakeLogsExporterConfig, newPushLogsData(want))
	require.Nil(t, err)
	require.NotNil(t, le)

	checkRecordedMetricsForLogsExporter(t, tt, le, want)
}

func TestLogsRequestExporter_WithRecordMetrics_ExportError(t *testing.T) {
	want := errors.New("export_error")
	tt, err := obsreporttest.SetupTelemetry(fakeLogsExporterName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	le, err := NewLogsRequestExporter(context.Background(), exporter.CreateSettings{ID: fakeLogsExporterName, TelemetrySettings: tt.TelemetrySettings, BuildInfo: component.NewDefaultBuildInfo()},
		(&fakeRequestConverter{requestError: want}).requestFromLogsFunc)
	require.Nil(t, err)
	require.NotNil(t, le)

	checkRecordedMetricsForLogsExporter(t, tt, le, want)
}

func TestLogsExporter_WithRecordEnqueueFailedMetrics(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(fakeLogsExporterName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rCfg := configretry.NewDefaultBackOffConfig()
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 1
	qCfg.QueueSize = 2
	wantErr := errors.New("some-error")
	te, err := NewLogsExporter(context.Background(), exporter.CreateSettings{ID: fakeLogsExporterName, TelemetrySettings: tt.TelemetrySettings, BuildInfo: component.NewDefaultBuildInfo()}, &fakeLogsExporterConfig, newPushLogsData(wantErr), WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	require.NotNil(t, te)

	md := testdata.GenerateLogs(3)
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		// errors are checked in the checkExporterEnqueueFailedLogsStats function below.
		_ = te.ConsumeLogs(context.Background(), md)
	}

	// 2 batched must be in queue, and 5 batches (15 log records) rejected due to queue overflow
	require.NoError(t, tt.CheckExporterEnqueueFailedLogs(int64(15)))
}

func TestLogsExporter_WithSpan(t *testing.T) {
	set := exportertest.NewNopCreateSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	le, err := NewLogsExporter(context.Background(), set, &fakeLogsExporterConfig, newPushLogsData(nil))
	require.Nil(t, err)
	require.NotNil(t, le)
	checkWrapSpanForLogsExporter(t, sr, set.TracerProvider.Tracer("test"), le, nil, 1)
}

func TestLogsRequestExporter_WithSpan(t *testing.T) {
	set := exportertest.NewNopCreateSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	le, err := NewLogsRequestExporter(context.Background(), set, (&fakeRequestConverter{}).requestFromLogsFunc)
	require.Nil(t, err)
	require.NotNil(t, le)
	checkWrapSpanForLogsExporter(t, sr, set.TracerProvider.Tracer("test"), le, nil, 1)
}

func TestLogsExporter_WithSpan_ReturnError(t *testing.T) {
	set := exportertest.NewNopCreateSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	want := errors.New("my_error")
	le, err := NewLogsExporter(context.Background(), set, &fakeLogsExporterConfig, newPushLogsData(want))
	require.Nil(t, err)
	require.NotNil(t, le)
	checkWrapSpanForLogsExporter(t, sr, set.TracerProvider.Tracer("test"), le, want, 1)
}

func TestLogsRequestExporter_WithSpan_ReturnError(t *testing.T) {
	set := exportertest.NewNopCreateSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	want := errors.New("my_error")
	le, err := NewLogsRequestExporter(context.Background(), set, (&fakeRequestConverter{requestError: want}).requestFromLogsFunc)
	require.Nil(t, err)
	require.NotNil(t, le)
	checkWrapSpanForLogsExporter(t, sr, set.TracerProvider.Tracer("test"), le, want, 1)
}

func TestLogsExporter_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	le, err := NewLogsExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeLogsExporterConfig, newPushLogsData(nil), WithShutdown(shutdown))
	assert.NotNil(t, le)
	assert.NoError(t, err)

	assert.Nil(t, le.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestLogsRequestExporter_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	le, err := NewLogsRequestExporter(context.Background(), exportertest.NewNopCreateSettings(),
		(&fakeRequestConverter{}).requestFromLogsFunc, WithShutdown(shutdown))
	assert.NotNil(t, le)
	assert.NoError(t, err)

	assert.Nil(t, le.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestLogsExporter_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	le, err := NewLogsExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeLogsExporterConfig, newPushLogsData(nil), WithShutdown(shutdownErr))
	assert.NotNil(t, le)
	assert.NoError(t, err)

	assert.Equal(t, le.Shutdown(context.Background()), want)
}

func TestLogsRequestExporter_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	le, err := NewLogsRequestExporter(context.Background(), exportertest.NewNopCreateSettings(),
		(&fakeRequestConverter{}).requestFromLogsFunc, WithShutdown(shutdownErr))
	assert.NotNil(t, le)
	assert.NoError(t, err)

	assert.Equal(t, le.Shutdown(context.Background()), want)
}

func newPushLogsData(retError error) consumer.ConsumeLogsFunc {
	return func(ctx context.Context, td plog.Logs) error {
		return retError
	}
}

func checkRecordedMetricsForLogsExporter(t *testing.T, tt obsreporttest.TestTelemetry, le exporter.Logs, wantError error) {
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

func checkWrapSpanForLogsExporter(t *testing.T, sr *tracetest.SpanRecorder, tracer trace.Tracer, le exporter.Logs,
	wantError error, numLogRecords int64) { // nolint: unparam
	const numRequests = 5
	generateLogsTraffic(t, tracer, le, numRequests, wantError)

	// Inspection time!
	gotSpanData := sr.Ended()
	require.Equal(t, numRequests+1, len(gotSpanData))

	parentSpan := gotSpanData[numRequests]
	require.Equalf(t, fakeLogsParentSpanName, parentSpan.Name(), "SpanData %v", parentSpan)
	for _, sd := range gotSpanData[:numRequests] {
		require.Equalf(t, parentSpan.SpanContext(), sd.Parent(), "Exporter span not a child\nSpanData %v", sd)
		checkStatus(t, sd, wantError)

		sentLogRecords := numLogRecords
		var failedToSendLogRecords int64
		if wantError != nil {
			sentLogRecords = 0
			failedToSendLogRecords = numLogRecords
		}
		require.Containsf(t, sd.Attributes(), attribute.KeyValue{Key: obsmetrics.SentLogRecordsKey, Value: attribute.Int64Value(sentLogRecords)}, "SpanData %v", sd)
		require.Containsf(t, sd.Attributes(), attribute.KeyValue{Key: obsmetrics.FailedToSendLogRecordsKey, Value: attribute.Int64Value(failedToSendLogRecords)}, "SpanData %v", sd)
	}
}
