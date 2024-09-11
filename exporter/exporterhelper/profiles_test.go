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
	"go.opentelemetry.io/collector/consumer/consumererror/consumererrorprofiles"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterprofiles"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/internal/queue"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/testdata"
)

const (
	fakeProfilesParentSpanName = "fake_profiles_parent_span_name"
)

var (
	fakeProfilesExporterName   = component.MustNewIDWithName("fake_profiles_exporter", "with_name")
	fakeProfilesExporterConfig = struct{}{}
)

func TestProfilesRequest(t *testing.T) {
	lr := newProfilesRequest(testdata.GenerateProfiles(1), nil)

	profileErr := consumererrorprofiles.NewProfiles(errors.New("some error"), pprofile.NewProfiles())
	assert.EqualValues(
		t,
		newProfilesRequest(pprofile.NewProfiles(), nil),
		lr.(RequestErrorHandler).OnError(profileErr),
	)
}

func TestProfilesExporter_InvalidName(t *testing.T) {
	le, err := NewProfilesExporter(context.Background(), exportertest.NewNopSettings(), nil, newPushProfilesData(nil))
	require.Nil(t, le)
	require.Equal(t, errNilConfig, err)
}

func TestProfilesExporter_NilLogger(t *testing.T) {
	le, err := NewProfilesExporter(context.Background(), exporter.Settings{}, &fakeProfilesExporterConfig, newPushProfilesData(nil))
	require.Nil(t, le)
	require.Equal(t, errNilLogger, err)
}

func TestProfilesRequestExporter_NilLogger(t *testing.T) {
	le, err := NewProfilesRequestExporter(context.Background(), exporter.Settings{}, (&fakeRequestConverter{}).requestFromProfilesFunc)
	require.Nil(t, le)
	require.Equal(t, errNilLogger, err)
}

func TestProfilesExporter_NilPushProfilesData(t *testing.T) {
	le, err := NewProfilesExporter(context.Background(), exportertest.NewNopSettings(), &fakeProfilesExporterConfig, nil)
	require.Nil(t, le)
	require.Equal(t, errNilPushProfileData, err)
}

func TestProfilesRequestExporter_NilProfilesConverter(t *testing.T) {
	le, err := NewProfilesRequestExporter(context.Background(), exportertest.NewNopSettings(), nil)
	require.Nil(t, le)
	require.Equal(t, errNilProfilesConverter, err)
}

func TestProfilesExporter_Default(t *testing.T) {
	ld := pprofile.NewProfiles()
	le, err := NewProfilesExporter(context.Background(), exportertest.NewNopSettings(), &fakeProfilesExporterConfig, newPushProfilesData(nil))
	assert.NotNil(t, le)
	assert.NoError(t, err)

	assert.Equal(t, consumer.Capabilities{MutatesData: false}, le.Capabilities())
	assert.NoError(t, le.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, le.ConsumeProfiles(context.Background(), ld))
	assert.NoError(t, le.Shutdown(context.Background()))
}

func TestProfilesRequestExporter_Default(t *testing.T) {
	ld := pprofile.NewProfiles()
	le, err := NewProfilesRequestExporter(context.Background(), exportertest.NewNopSettings(),
		(&fakeRequestConverter{}).requestFromProfilesFunc)
	assert.NotNil(t, le)
	assert.NoError(t, err)

	assert.Equal(t, consumer.Capabilities{MutatesData: false}, le.Capabilities())
	assert.NoError(t, le.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, le.ConsumeProfiles(context.Background(), ld))
	assert.NoError(t, le.Shutdown(context.Background()))
}

func TestProfilesExporter_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	le, err := NewProfilesExporter(context.Background(), exportertest.NewNopSettings(), &fakeProfilesExporterConfig, newPushProfilesData(nil), WithCapabilities(capabilities))
	require.NoError(t, err)
	require.NotNil(t, le)

	assert.Equal(t, capabilities, le.Capabilities())
}

func TestProfilesRequestExporter_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	le, err := NewProfilesRequestExporter(context.Background(), exportertest.NewNopSettings(),
		(&fakeRequestConverter{}).requestFromProfilesFunc, WithCapabilities(capabilities))
	require.NoError(t, err)
	require.NotNil(t, le)

	assert.Equal(t, capabilities, le.Capabilities())
}

func TestProfilesExporter_Default_ReturnError(t *testing.T) {
	ld := pprofile.NewProfiles()
	want := errors.New("my_error")
	le, err := NewProfilesExporter(context.Background(), exportertest.NewNopSettings(), &fakeProfilesExporterConfig, newPushProfilesData(want))
	require.NoError(t, err)
	require.NotNil(t, le)
	require.Equal(t, want, le.ConsumeProfiles(context.Background(), ld))
}

func TestProfilesRequestExporter_Default_ConvertError(t *testing.T) {
	ld := pprofile.NewProfiles()
	want := errors.New("convert_error")
	le, err := NewProfilesRequestExporter(context.Background(), exportertest.NewNopSettings(),
		(&fakeRequestConverter{profilesError: want}).requestFromProfilesFunc)
	require.NoError(t, err)
	require.NotNil(t, le)
	require.Equal(t, consumererror.NewPermanent(want), le.ConsumeProfiles(context.Background(), ld))
}

func TestProfilesRequestExporter_Default_ExportError(t *testing.T) {
	ld := pprofile.NewProfiles()
	want := errors.New("export_error")
	le, err := NewProfilesRequestExporter(context.Background(), exportertest.NewNopSettings(),
		(&fakeRequestConverter{requestError: want}).requestFromProfilesFunc)
	require.NoError(t, err)
	require.NotNil(t, le)
	require.Equal(t, want, le.ConsumeProfiles(context.Background(), ld))
}

func TestProfilesExporter_WithPersistentQueue(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	storageID := component.MustNewIDWithName("file_storage", "storage")
	qCfg.StorageID = &storageID
	rCfg := configretry.NewDefaultBackOffConfig()
	ts := consumertest.ProfilesSink{}
	set := exportertest.NewNopSettings()
	set.ID = component.MustNewIDWithName("test_profiles", "with_persistent_queue")
	te, err := NewProfilesExporter(context.Background(), set, &fakeProfilesExporterConfig, ts.ConsumeProfiles, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)

	host := &mockHost{ext: map[component.ID]component.Component{
		storageID: queue.NewMockStorageExtension(nil),
	}}
	require.NoError(t, te.Start(context.Background(), host))
	t.Cleanup(func() { require.NoError(t, te.Shutdown(context.Background())) })

	traces := testdata.GenerateProfiles(2)
	require.NoError(t, te.ConsumeProfiles(context.Background(), traces))
	require.Eventually(t, func() bool {
		return len(ts.AllProfiles()) == 1 && ts.SampleCount() == 2
	}, 500*time.Millisecond, 10*time.Millisecond)
}

func TestProfilesExporter_WithRecordMetrics(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(fakeProfilesExporterName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	le, err := NewProfilesExporter(context.Background(), exporter.Settings{ID: fakeProfilesExporterName, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, &fakeProfilesExporterConfig, newPushProfilesData(nil))
	require.NoError(t, err)
	require.NotNil(t, le)

	checkRecordedMetricsForProfilesExporter(t, tt, le, nil)
}

func TestProfilesExporter_pProfileModifiedDownStream_WithRecordMetrics(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(fakeProfilesExporterName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	le, err := NewProfilesExporter(context.Background(), exporter.Settings{ID: fakeProfilesExporterName, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, &fakeProfilesExporterConfig, newPushProfilesDataModifiedDownstream(nil), WithCapabilities(consumer.Capabilities{MutatesData: true}))
	assert.NotNil(t, le)
	assert.NoError(t, err)
	ld := testdata.GenerateProfiles(2)

	assert.NoError(t, le.ConsumeProfiles(context.Background(), ld))
	assert.Equal(t, 0, ld.SampleCount())
	require.NoError(t, tt.CheckExporterProfiles(int64(2), 0))
}

func TestProfilesRequestExporter_WithRecordMetrics(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(fakeProfilesExporterName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	le, err := NewProfilesRequestExporter(context.Background(),
		exporter.Settings{ID: fakeProfilesExporterName, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		(&fakeRequestConverter{}).requestFromProfilesFunc)
	require.NoError(t, err)
	require.NotNil(t, le)

	checkRecordedMetricsForProfilesExporter(t, tt, le, nil)
}

func TestProfilesExporter_WithRecordMetrics_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	tt, err := componenttest.SetupTelemetry(fakeProfilesExporterName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	le, err := NewProfilesExporter(context.Background(), exporter.Settings{ID: fakeProfilesExporterName, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, &fakeProfilesExporterConfig, newPushProfilesData(want))
	require.Nil(t, err)
	require.NotNil(t, le)

	checkRecordedMetricsForProfilesExporter(t, tt, le, want)
}

func TestProfilesRequestExporter_WithRecordMetrics_ExportError(t *testing.T) {
	want := errors.New("export_error")
	tt, err := componenttest.SetupTelemetry(fakeProfilesExporterName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	le, err := NewProfilesRequestExporter(context.Background(), exporter.Settings{ID: fakeProfilesExporterName, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		(&fakeRequestConverter{requestError: want}).requestFromProfilesFunc)
	require.Nil(t, err)
	require.NotNil(t, le)

	checkRecordedMetricsForProfilesExporter(t, tt, le, want)
}

func TestProfilesExporter_WithRecordEnqueueFailedMetrics(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(fakeProfilesExporterName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rCfg := configretry.NewDefaultBackOffConfig()
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 1
	qCfg.QueueSize = 2
	wantErr := errors.New("some-error")
	te, err := NewProfilesExporter(context.Background(), exporter.Settings{ID: fakeProfilesExporterName, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, &fakeProfilesExporterConfig, newPushProfilesData(wantErr), WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	require.NotNil(t, te)

	md := testdata.GenerateProfiles(3)
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		// errors are checked in the checkExporterEnqueueFailedProfilesStats function below.
		_ = te.ConsumeProfiles(context.Background(), md)
	}

	// 2 batched must be in queue, and 5 batches (15 profile records) rejected due to queue overflow
	require.NoError(t, tt.CheckExporterEnqueueFailedProfiles(int64(15)))
}

func TestProfilesExporter_WithSpan(t *testing.T) {
	set := exportertest.NewNopSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	le, err := NewProfilesExporter(context.Background(), set, &fakeProfilesExporterConfig, newPushProfilesData(nil))
	require.Nil(t, err)
	require.NotNil(t, le)
	checkWrapSpanForProfilesExporter(t, sr, set.TracerProvider.Tracer("test"), le, nil, 1)
}

func TestProfilesRequestExporter_WithSpan(t *testing.T) {
	set := exportertest.NewNopSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	le, err := NewProfilesRequestExporter(context.Background(), set, (&fakeRequestConverter{}).requestFromProfilesFunc)
	require.Nil(t, err)
	require.NotNil(t, le)
	checkWrapSpanForProfilesExporter(t, sr, set.TracerProvider.Tracer("test"), le, nil, 1)
}

func TestProfilesExporter_WithSpan_ReturnError(t *testing.T) {
	set := exportertest.NewNopSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	want := errors.New("my_error")
	le, err := NewProfilesExporter(context.Background(), set, &fakeProfilesExporterConfig, newPushProfilesData(want))
	require.Nil(t, err)
	require.NotNil(t, le)
	checkWrapSpanForProfilesExporter(t, sr, set.TracerProvider.Tracer("test"), le, want, 1)
}

func TestProfilesRequestExporter_WithSpan_ReturnError(t *testing.T) {
	set := exportertest.NewNopSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	want := errors.New("my_error")
	le, err := NewProfilesRequestExporter(context.Background(), set, (&fakeRequestConverter{requestError: want}).requestFromProfilesFunc)
	require.Nil(t, err)
	require.NotNil(t, le)
	checkWrapSpanForProfilesExporter(t, sr, set.TracerProvider.Tracer("test"), le, want, 1)
}

func TestProfilesExporter_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	le, err := NewProfilesExporter(context.Background(), exportertest.NewNopSettings(), &fakeProfilesExporterConfig, newPushProfilesData(nil), WithShutdown(shutdown))
	assert.NotNil(t, le)
	assert.NoError(t, err)

	assert.Nil(t, le.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestProfilesRequestExporter_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	le, err := NewProfilesRequestExporter(context.Background(), exportertest.NewNopSettings(),
		(&fakeRequestConverter{}).requestFromProfilesFunc, WithShutdown(shutdown))
	assert.NotNil(t, le)
	assert.NoError(t, err)

	assert.Nil(t, le.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestProfilesExporter_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	le, err := NewProfilesExporter(context.Background(), exportertest.NewNopSettings(), &fakeProfilesExporterConfig, newPushProfilesData(nil), WithShutdown(shutdownErr))
	assert.NotNil(t, le)
	assert.NoError(t, err)

	assert.Equal(t, le.Shutdown(context.Background()), want)
}

func TestProfilesRequestExporter_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	le, err := NewProfilesRequestExporter(context.Background(), exportertest.NewNopSettings(),
		(&fakeRequestConverter{}).requestFromProfilesFunc, WithShutdown(shutdownErr))
	assert.NotNil(t, le)
	assert.NoError(t, err)

	assert.Equal(t, le.Shutdown(context.Background()), want)
}

func newPushProfilesDataModifiedDownstream(retError error) consumerprofiles.ConsumeProfilesFunc {
	return func(_ context.Context, profile pprofile.Profiles) error {
		profile.ResourceProfiles().MoveAndAppendTo(pprofile.NewResourceProfilesSlice())
		return retError
	}
}

func newPushProfilesData(retError error) consumerprofiles.ConsumeProfilesFunc {
	return func(_ context.Context, _ pprofile.Profiles) error {
		return retError
	}
}

func checkRecordedMetricsForProfilesExporter(t *testing.T, tt componenttest.TestTelemetry, le exporterprofiles.Profiles, wantError error) {
	ld := testdata.GenerateProfiles(2)
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		require.Equal(t, wantError, le.ConsumeProfiles(context.Background(), ld))
	}

	// TODO: When the new metrics correctly count partial dropped fix this.
	if wantError != nil {
		require.NoError(t, tt.CheckExporterProfiles(0, int64(numBatches*ld.SampleCount())))
	} else {
		require.NoError(t, tt.CheckExporterProfiles(int64(numBatches*ld.SampleCount()), 0))
	}
}

func generateProfilesTraffic(t *testing.T, tracer trace.Tracer, le exporterprofiles.Profiles, numRequests int, wantError error) {
	ld := testdata.GenerateProfiles(1)
	ctx, span := tracer.Start(context.Background(), fakeProfilesParentSpanName)
	defer span.End()
	for i := 0; i < numRequests; i++ {
		require.Equal(t, wantError, le.ConsumeProfiles(ctx, ld))
	}
}

func checkWrapSpanForProfilesExporter(t *testing.T, sr *tracetest.SpanRecorder, tracer trace.Tracer, le exporterprofiles.Profiles,
	wantError error, numProfileRecords int64) { // nolint: unparam
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
		require.Containsf(t, sd.Attributes(), attribute.KeyValue{Key: obsmetrics.SentSamplesKey, Value: attribute.Int64Value(sentProfileRecords)}, "SpanData %v", sd)
		require.Containsf(t, sd.Attributes(), attribute.KeyValue{Key: obsmetrics.FailedToSendSamplesKey, Value: attribute.Int64Value(failedToSendProfileRecords)}, "SpanData %v", sd)
	}
}
