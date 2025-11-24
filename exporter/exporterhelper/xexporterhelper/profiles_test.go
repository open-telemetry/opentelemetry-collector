// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xexporterhelper

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
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/consumererror/xconsumererror"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/hosttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/oteltest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sendertest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/storagetest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/xexporter"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/testdata"
)

const (
	fakeProfilesParentSpanName = "fake_profiles_parent_span_name"
)

var fakeProfilesExporterConfig = struct{}{}

func TestProfilesRequest(t *testing.T) {
	lr := newProfilesRequest(testdata.GenerateProfiles(1))

	profileErr := xconsumererror.NewProfiles(errors.New("some error"), pprofile.NewProfiles())
	assert.Equal(
		t,
		newProfilesRequest(pprofile.NewProfiles()),
		lr.(RequestErrorHandler).OnError(profileErr),
	)
}

func TestProfilesExporter_InvalidName(t *testing.T) {
	le, err := NewProfiles(context.Background(), exportertest.NewNopSettings(exportertest.NopType), nil, newPushProfilesData(nil))
	require.Nil(t, le)
	require.Equal(t, errNilConfig, err)
}

func TestProfilesExporter_NilLogger(t *testing.T) {
	le, err := NewProfiles(context.Background(), exporter.Settings{}, &fakeProfilesExporterConfig, newPushProfilesData(nil))
	require.Nil(t, le)
	require.Equal(t, errNilLogger, err)
}

func TestProfilesRequestExporter_NilLogger(t *testing.T) {
	le, err := NewProfilesRequest(context.Background(), exporter.Settings{}, requestFromProfilesFunc(nil), sendertest.NewNopSenderFunc[Request]())
	require.Nil(t, le)
	require.Equal(t, errNilLogger, err)
}

func TestProfilesExporter_NilPushProfilesData(t *testing.T) {
	le, err := NewProfiles(context.Background(), exportertest.NewNopSettings(exportertest.NopType), &fakeProfilesExporterConfig, nil)
	require.Nil(t, le)
	require.Equal(t, errNilPushProfileData, err)
}

func TestProfilesExporter_NilProfilesConverter(t *testing.T) {
	te, err := NewProfilesRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType), nil, sendertest.NewNopSenderFunc[Request]())
	require.Nil(t, te)
	require.Equal(t, errNilProfilesConverter, err)
}

func TestProfilesRequestExporter_NilProfilesConverter(t *testing.T) {
	le, err := NewProfilesRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType), requestFromProfilesFunc(nil), nil)
	require.Nil(t, le)
	require.Equal(t, errNilConsumeRequest, err)
}

func TestProfilesExporter_Default(t *testing.T) {
	ld := pprofile.NewProfiles()
	le, err := NewProfiles(context.Background(), exportertest.NewNopSettings(exportertest.NopType), &fakeProfilesExporterConfig, newPushProfilesData(nil))
	assert.NotNil(t, le)
	require.NoError(t, err)

	assert.Equal(t, consumer.Capabilities{MutatesData: false}, le.Capabilities())
	require.NoError(t, le.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, le.ConsumeProfiles(context.Background(), ld))
	require.NoError(t, le.Shutdown(context.Background()))
}

func TestProfilesRequestExporter_Default(t *testing.T) {
	ld := pprofile.NewProfiles()
	le, err := NewProfilesRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requestFromProfilesFunc(nil), sendertest.NewNopSenderFunc[Request]())
	assert.NotNil(t, le)
	require.NoError(t, err)

	assert.Equal(t, consumer.Capabilities{MutatesData: false}, le.Capabilities())
	require.NoError(t, le.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, le.ConsumeProfiles(context.Background(), ld))
	require.NoError(t, le.Shutdown(context.Background()))
}

func TestProfilesExporter_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	le, err := NewProfiles(context.Background(), exportertest.NewNopSettings(exportertest.NopType), &fakeProfilesExporterConfig, newPushProfilesData(nil), exporterhelper.WithCapabilities(capabilities))
	require.NoError(t, err)
	require.NotNil(t, le)

	assert.Equal(t, capabilities, le.Capabilities())
}

func TestProfilesRequestExporter_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	le, err := NewProfilesRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requestFromProfilesFunc(nil), sendertest.NewNopSenderFunc[Request](), exporterhelper.WithCapabilities(capabilities))
	require.NoError(t, err)
	require.NotNil(t, le)

	assert.Equal(t, capabilities, le.Capabilities())
}

func TestProfilesExporter_Default_ReturnError(t *testing.T) {
	ld := pprofile.NewProfiles()
	want := errors.New("my_error")
	le, err := NewProfiles(context.Background(), exportertest.NewNopSettings(exportertest.NopType), &fakeProfilesExporterConfig, newPushProfilesData(want))
	require.NoError(t, err)
	require.NotNil(t, le)
	require.Equal(t, want, le.ConsumeProfiles(context.Background(), ld))
}

func TestProfilesRequestExporter_Default_ConvertError(t *testing.T) {
	ld := pprofile.NewProfiles()
	want := errors.New("convert_error")
	le, err := NewProfilesRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requestFromProfilesFunc(want), sendertest.NewNopSenderFunc[Request]())
	require.NoError(t, err)
	require.NotNil(t, le)
	require.Equal(t, consumererror.NewPermanent(want), le.ConsumeProfiles(context.Background(), ld))
}

func TestProfilesRequestExporter_Default_ExportError(t *testing.T) {
	ld := pprofile.NewProfiles()
	want := errors.New("export_error")
	le, err := NewProfilesRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requestFromProfilesFunc(nil), sendertest.NewErrSenderFunc[Request](want))
	require.NoError(t, err)
	require.NotNil(t, le)
	require.Equal(t, want, le.ConsumeProfiles(context.Background(), ld))
}

func TestProfiles_WithPersistentQueue(t *testing.T) {
	fgOrigReadState := queue.PersistRequestContextOnRead
	fgOrigWriteState := queue.PersistRequestContextOnWrite
	qCfg := exporterhelper.NewDefaultQueueConfig()
	storageID := component.MustNewIDWithName("file_storage", "storage")
	qCfg.StorageID = &storageID
	set := exportertest.NewNopSettings(exportertest.NopType)
	set.ID = component.MustNewIDWithName("test_logs", "with_persistent_queue")
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

			ts := consumertest.ProfilesSink{}
			te, err := NewProfiles(context.Background(), set, &fakeProfilesExporterConfig, ts.ConsumeProfiles, exporterhelper.WithQueue(configoptional.Some(qCfg)))
			require.NoError(t, err)
			require.NoError(t, te.Start(context.Background(), host))
			t.Cleanup(func() { require.NoError(t, te.Shutdown(context.Background())) })

			profiles := testdata.GenerateProfiles(2)
			require.NoError(t, te.ConsumeProfiles(trace.ContextWithSpanContext(context.Background(), spanCtx), profiles))
			if tt.wantData {
				require.Eventually(t, func() bool {
					return len(ts.AllProfiles()) == 1 && ts.SampleCount() == 2
				}, 500*time.Millisecond, 10*time.Millisecond)
			}

			// check that the span context is persisted if the feature gate is enabled
			if tt.wantSpanCtx {
				assert.Len(t, ts.Contexts(), 1)
				assert.Equal(t, spanCtx, trace.SpanContextFromContext(ts.Contexts()[0]))
			}
		})
	}
}

func TestProfilesExporter_WithSpan(t *testing.T) {
	set := exportertest.NewNopSettings(exportertest.NopType)
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	le, err := NewProfiles(context.Background(), set, &fakeProfilesExporterConfig, newPushProfilesData(nil))
	require.NoError(t, err)
	require.NotNil(t, le)
	checkWrapSpanForProfilesExporter(t, sr, set.TracerProvider.Tracer("test"), le, nil)
}

func TestProfilesRequestExporter_WithSpan(t *testing.T) {
	set := exportertest.NewNopSettings(exportertest.NopType)
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	le, err := NewProfilesRequest(context.Background(), set, requestFromProfilesFunc(nil), sendertest.NewNopSenderFunc[Request]())
	require.NoError(t, err)
	require.NotNil(t, le)
	checkWrapSpanForProfilesExporter(t, sr, set.TracerProvider.Tracer("test"), le, nil)
}

func TestProfilesExporter_WithSpan_ReturnError(t *testing.T) {
	set := exportertest.NewNopSettings(exportertest.NopType)
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	want := errors.New("my_error")
	le, err := NewProfiles(context.Background(), set, &fakeProfilesExporterConfig, newPushProfilesData(want))
	require.NoError(t, err)
	require.NotNil(t, le)
	checkWrapSpanForProfilesExporter(t, sr, set.TracerProvider.Tracer("test"), le, want)
}

func TestProfilesRequestExporter_WithSpan_ReturnError(t *testing.T) {
	set := exportertest.NewNopSettings(exportertest.NopType)
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	want := errors.New("my_error")
	le, err := NewProfilesRequest(context.Background(), set, requestFromProfilesFunc(nil), sendertest.NewErrSenderFunc[Request](want))
	require.NoError(t, err)
	require.NotNil(t, le)
	checkWrapSpanForProfilesExporter(t, sr, set.TracerProvider.Tracer("test"), le, want)
}

func TestProfilesExporter_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	le, err := NewProfiles(context.Background(), exportertest.NewNopSettings(exportertest.NopType), &fakeProfilesExporterConfig, newPushProfilesData(nil), exporterhelper.WithShutdown(shutdown))
	assert.NotNil(t, le)
	require.NoError(t, err)

	require.NoError(t, le.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestProfilesRequestExporter_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	le, err := NewProfilesRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requestFromProfilesFunc(nil), sendertest.NewNopSenderFunc[Request](), exporterhelper.WithShutdown(shutdown))
	assert.NotNil(t, le)
	require.NoError(t, err)

	require.NoError(t, le.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestProfilesExporter_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	le, err := NewProfiles(context.Background(), exportertest.NewNopSettings(exportertest.NopType), &fakeProfilesExporterConfig, newPushProfilesData(nil), exporterhelper.WithShutdown(shutdownErr))
	assert.NotNil(t, le)
	require.NoError(t, err)

	assert.Equal(t, want, le.Shutdown(context.Background()))
}

func TestProfilesRequestExporter_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	le, err := NewProfilesRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requestFromProfilesFunc(nil), sendertest.NewNopSenderFunc[Request](), exporterhelper.WithShutdown(shutdownErr))
	assert.NotNil(t, le)
	require.NoError(t, err)

	assert.Equal(t, want, le.Shutdown(context.Background()))
}

func newPushProfilesData(retError error) xconsumer.ConsumeProfilesFunc {
	return func(_ context.Context, _ pprofile.Profiles) error {
		return retError
	}
}

func generateProfilesTraffic(t *testing.T, tracer trace.Tracer, le xexporter.Profiles, numRequests int, wantError error) {
	ld := testdata.GenerateProfiles(1)
	ctx, span := tracer.Start(context.Background(), fakeProfilesParentSpanName)
	defer span.End()
	for range numRequests {
		require.Equal(t, wantError, le.ConsumeProfiles(ctx, ld))
	}
}

func checkWrapSpanForProfilesExporter(t *testing.T, sr *tracetest.SpanRecorder, tracer trace.Tracer, le xexporter.Profiles, wantError error) {
	const numRequests = 5
	generateProfilesTraffic(t, tracer, le, numRequests, wantError)

	// Inspection time!
	gotSpanData := sr.Ended()
	require.Len(t, gotSpanData, numRequests+1)

	parentSpan := gotSpanData[numRequests]
	require.Equalf(t, fakeProfilesParentSpanName, parentSpan.Name(), "SpanData %v", parentSpan)
	for _, sd := range gotSpanData[:numRequests] {
		require.Equalf(t, parentSpan.SpanContext(), sd.Parent(), "Exporter span not a child\nSpanData %v", sd)
		oteltest.CheckStatus(t, sd, wantError)

		sentSampleRecords := int64(1)
		failedToSendSampleRecords := int64(0)
		if wantError != nil {
			sentSampleRecords = 0
			failedToSendSampleRecords = 1
		}
		require.Containsf(t, sd.Attributes(), attribute.KeyValue{Key: internal.ItemsSent, Value: attribute.Int64Value(sentSampleRecords)}, "SpanData %v", sd)
		require.Containsf(t, sd.Attributes(), attribute.KeyValue{Key: internal.ItemsFailed, Value: attribute.Int64Value(failedToSendSampleRecords)}, "SpanData %v", sd)
	}
}

func requestFromProfilesFunc(err error) func(context.Context, pprofile.Profiles) (Request, error) {
	return func(_ context.Context, pd pprofile.Profiles) (Request, error) {
		return &requesttest.FakeRequest{Items: pd.SampleCount()}, err
	}
}
