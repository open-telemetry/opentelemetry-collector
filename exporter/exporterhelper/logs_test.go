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
	lr := newLogsRequest(context.Background(), testdata.GenerateLogs(1), nil)

	logErr := consumererror.NewLogs(errors.New("some error"), plog.NewLogs())
	assert.EqualValues(
		t,
		newLogsRequest(context.Background(), plog.NewLogs(), nil),
		lr.OnError(logErr),
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

func TestLogsExporter_NilPushLogsData(t *testing.T) {
	le, err := NewLogsExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeLogsExporterConfig, nil)
	require.Nil(t, le)
	require.Equal(t, errNilPushLogsData, err)
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

func TestLogsExporter_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	le, err := NewLogsExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeLogsExporterConfig, newPushLogsData(nil), WithCapabilities(capabilities))
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

func TestLogsExporter_WithRecordLogs(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(fakeLogsExporterName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	le, err := NewLogsExporter(context.Background(), tt.ToExporterCreateSettings(), &fakeLogsExporterConfig, newPushLogsData(nil))
	require.NoError(t, err)
	require.NotNil(t, le)

	checkRecordedMetricsForLogsExporter(t, tt, le, nil)
}

func TestLogsExporter_WithRecordLogs_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	tt, err := obsreporttest.SetupTelemetry(fakeLogsExporterName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	le, err := NewLogsExporter(context.Background(), tt.ToExporterCreateSettings(), &fakeLogsExporterConfig, newPushLogsData(want))
	require.Nil(t, err)
	require.NotNil(t, le)

	checkRecordedMetricsForLogsExporter(t, tt, le, want)
}

func TestLogsExporter_WithRecordEnqueueFailedMetrics(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(fakeLogsExporterName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rCfg := NewDefaultRetrySettings()
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 1
	qCfg.QueueSize = 2
	wantErr := errors.New("some-error")
	te, err := NewLogsExporter(context.Background(), tt.ToExporterCreateSettings(), &fakeLogsExporterConfig, newPushLogsData(wantErr), WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	require.NotNil(t, te)

	md := testdata.GenerateLogs(3)
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		// errors are checked in the checkExporterEnqueueFailedLogsStats function below.
		_ = te.ConsumeLogs(context.Background(), md)
	}

	// 2 batched must be in queue, and 5 batches (15 log records) rejected due to queue overflow
	checkExporterEnqueueFailedLogsStats(t, globalInstruments, fakeLogsExporterName, int64(15))
}

func TestLogsExporter_WithSpan(t *testing.T) {
	set := exportertest.NewNopCreateSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(trace.NewNoopTracerProvider())

	le, err := NewLogsExporter(context.Background(), set, &fakeLogsExporterConfig, newPushLogsData(nil))
	require.Nil(t, err)
	require.NotNil(t, le)
	checkWrapSpanForLogsExporter(t, sr, set.TracerProvider.Tracer("test"), le, nil, 1)
}

func TestLogsExporter_WithSpan_ReturnError(t *testing.T) {
	set := exportertest.NewNopCreateSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(trace.NewNoopTracerProvider())

	want := errors.New("my_error")
	le, err := NewLogsExporter(context.Background(), set, &fakeLogsExporterConfig, newPushLogsData(want))
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

func TestLogsExporter_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	le, err := NewLogsExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeLogsExporterConfig, newPushLogsData(nil), WithShutdown(shutdownErr))
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

func checkWrapSpanForLogsExporter(t *testing.T, sr *tracetest.SpanRecorder, tracer trace.Tracer, le exporter.Logs, wantError error, numLogRecords int64) {
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
