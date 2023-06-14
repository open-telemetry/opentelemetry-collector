// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
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
	fakeProfilesParentSpanName = "fake_logs_parent_span_name"
)

var (
	fakeProfilesExporterName   = component.NewIDWithName("fake_logs_exporter", "with_name")
	fakeProfilesExporterConfig = struct{}{}
)

func TestProfilesRequest(t *testing.T) {
	lr := newProfilesRequest(context.Background(), testdata.GenerateProfiles(1), nil)

	logErr := consumererror.NewProfiles(errors.New("some error"), plog.NewProfiles())
	assert.EqualValues(
		t,
		newProfilesRequest(context.Background(), plog.NewProfiles(), nil),
		lr.OnError(logErr),
	)
}

func TestProfilesExporter_InvalidName(t *testing.T) {
	le, err := NewProfilesExporter(context.Background(), exportertest.NewNopCreateSettings(), nil, newPushProfilesData(nil))
	require.Nil(t, le)
	require.Equal(t, errNilConfig, err)
}

func TestProfilesExporter_NilProfileger(t *testing.T) {
	le, err := NewProfilesExporter(context.Background(), exporter.CreateSettings{}, &fakeProfilesExporterConfig, newPushProfilesData(nil))
	require.Nil(t, le)
	require.Equal(t, errNilProfileger, err)
}

func TestProfilesExporter_NilPushProfilesData(t *testing.T) {
	le, err := NewProfilesExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeProfilesExporterConfig, nil)
	require.Nil(t, le)
	require.Equal(t, errNilPushProfilesData, err)
}

func TestProfilesExporter_Default(t *testing.T) {
	ld := plog.NewProfiles()
	le, err := NewProfilesExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeProfilesExporterConfig, newPushProfilesData(nil))
	assert.NotNil(t, le)
	assert.NoError(t, err)

	assert.Equal(t, consumer.Capabilities{MutatesData: false}, le.Capabilities())
	assert.NoError(t, le.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, le.ConsumeProfiles(context.Background(), ld))
	assert.NoError(t, le.Shutdown(context.Background()))
}

func TestProfilesExporter_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	le, err := NewProfilesExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeProfilesExporterConfig, newPushProfilesData(nil), WithCapabilities(capabilities))
	require.NoError(t, err)
	require.NotNil(t, le)

	assert.Equal(t, capabilities, le.Capabilities())
}

func TestProfilesExporter_Default_ReturnError(t *testing.T) {
	ld := plog.NewProfiles()
	want := errors.New("my_error")
	le, err := NewProfilesExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeProfilesExporterConfig, newPushProfilesData(want))
	require.NoError(t, err)
	require.NotNil(t, le)
	require.Equal(t, want, le.ConsumeProfiles(context.Background(), ld))
}

func TestProfilesExporter_WithRecordProfiles(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(fakeProfilesExporterName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	le, err := NewProfilesExporter(context.Background(), tt.ToExporterCreateSettings(), &fakeProfilesExporterConfig, newPushProfilesData(nil))
	require.NoError(t, err)
	require.NotNil(t, le)

	checkRecordedMetricsForProfilesExporter(t, tt, le, nil)
}

func TestProfilesExporter_WithRecordProfiles_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	tt, err := obsreporttest.SetupTelemetry(fakeProfilesExporterName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	le, err := NewProfilesExporter(context.Background(), tt.ToExporterCreateSettings(), &fakeProfilesExporterConfig, newPushProfilesData(want))
	require.Nil(t, err)
	require.NotNil(t, le)

	checkRecordedMetricsForProfilesExporter(t, tt, le, want)
}

func TestProfilesExporter_WithRecordEnqueueFailedMetrics(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(fakeProfilesExporterName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rCfg := NewDefaultRetrySettings()
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 1
	qCfg.QueueSize = 2
	wantErr := errors.New("some-error")
	te, err := NewProfilesExporter(context.Background(), tt.ToExporterCreateSettings(), &fakeProfilesExporterConfig, newPushProfilesData(wantErr), WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	require.NotNil(t, te)

	md := testdata.GenerateProfiles(3)
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		// errors are checked in the checkExporterEnqueueFailedProfilesStats function below.
		_ = te.ConsumeProfiles(context.Background(), md)
	}

	// 2 batched must be in queue, and 5 batches (15 log records) rejected due to queue overflow
	checkExporterEnqueueFailedProfilesStats(t, globalInstruments, fakeProfilesExporterName, int64(15))
}

func TestProfilesExporter_WithSpan(t *testing.T) {
	set := exportertest.NewNopCreateSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(trace.NewNoopTracerProvider())

	le, err := NewProfilesExporter(context.Background(), set, &fakeProfilesExporterConfig, newPushProfilesData(nil))
	require.Nil(t, err)
	require.NotNil(t, le)
	checkWrapSpanForProfilesExporter(t, sr, set.TracerProvider.Tracer("test"), le, nil, 1)
}

func TestProfilesExporter_WithSpan_ReturnError(t *testing.T) {
	set := exportertest.NewNopCreateSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(trace.NewNoopTracerProvider())

	want := errors.New("my_error")
	le, err := NewProfilesExporter(context.Background(), set, &fakeProfilesExporterConfig, newPushProfilesData(want))
	require.Nil(t, err)
	require.NotNil(t, le)
	checkWrapSpanForProfilesExporter(t, sr, set.TracerProvider.Tracer("test"), le, want, 1)
}

func TestProfilesExporter_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	le, err := NewProfilesExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeProfilesExporterConfig, newPushProfilesData(nil), WithShutdown(shutdown))
	assert.NotNil(t, le)
	assert.NoError(t, err)

	assert.Nil(t, le.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestProfilesExporter_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	le, err := NewProfilesExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeProfilesExporterConfig, newPushProfilesData(nil), WithShutdown(shutdownErr))
	assert.NotNil(t, le)
	assert.NoError(t, err)

	assert.Equal(t, le.Shutdown(context.Background()), want)
}

func newPushProfilesData(retError error) consumer.ConsumeProfilesFunc {
	return func(ctx context.Context, td plog.Profiles) error {
		return retError
	}
}

func checkRecordedMetricsForProfilesExporter(t *testing.T, tt obsreporttest.TestTelemetry, le exporter.Profiles, wantError error) {
	ld := testdata.GenerateProfiles(2)
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		require.Equal(t, wantError, le.ConsumeProfiles(context.Background(), ld))
	}

	// TODO: When the new metrics correctly count partial dropped fix this.
	if wantError != nil {
		require.NoError(t, tt.CheckExporterProfiles(0, int64(numBatches*ld.ProfileRecordCount())))
	} else {
		require.NoError(t, tt.CheckExporterProfiles(int64(numBatches*ld.ProfileRecordCount()), 0))
	}
}

func generateProfilesTraffic(t *testing.T, tracer trace.Tracer, le exporter.Profiles, numRequests int, wantError error) {
	ld := testdata.GenerateProfiles(1)
	ctx, span := tracer.Start(context.Background(), fakeProfilesParentSpanName)
	defer span.End()
	for i := 0; i < numRequests; i++ {
		require.Equal(t, wantError, le.ConsumeProfiles(ctx, ld))
	}
}

func checkWrapSpanForProfilesExporter(t *testing.T, sr *tracetest.SpanRecorder, tracer trace.Tracer, le exporter.Profiles, wantError error, numProfileRecords int64) {
	const numRequests = 5
	generateProfilesTraffic(t, tracer, le, numRequests, wantError)

	// Inspection time!
	gotSpanData := sr.Ended()
	require.Equal(t, numRequests+1, len(gotSpanData))

	parentSpan := gotSpanData[numRequests]
	require.Equalf(t, fakeProfilesParentSpanName, parentSpan.Name(), "SpanData %v", parentSpan)
	for _, sd := range gotSpanData[:numRequests] {
		require.Equalf(t, parentSpan.SpanContext(), sd.Parent(), "Exporter span not a child\nSpanData %v", sd)
		checkStatus(t, sd, wantError)

		sentProfileRecords := numProfileRecords
		var failedToSendProfileRecords int64
		if wantError != nil {
			sentProfileRecords = 0
			failedToSendProfileRecords = numProfileRecords
		}
		require.Containsf(t, sd.Attributes(), attribute.KeyValue{Key: obsmetrics.SentProfileRecordsKey, Value: attribute.Int64Value(sentProfileRecords)}, "SpanData %v", sd)
		require.Containsf(t, sd.Attributes(), attribute.KeyValue{Key: obsmetrics.FailedToSendProfileRecordsKey, Value: attribute.Int64Value(failedToSendProfileRecords)}, "SpanData %v", sd)
	}
}
