// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadatatest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
)

var (
	exporterID = component.MustNewID("fakeExporter")

	errFake = errors.New("errFake")
)

func TestExportTraceDataOp(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	parentCtx, parentSpan := tt.NewTelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
	defer parentSpan.End()

	var exporterErr error
	obsrep, err := newObsReportSender(
		exporter.Settings{ID: exporterID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		pipeline.SignalTraces,
		sender.NewSender(func(context.Context, request.Request) error { return exporterErr }),
	)
	require.NoError(t, err)

	params := []testParams{
		{items: 22, err: nil},
		{items: 14, err: errFake},
	}
	for i := range params {
		exporterErr = params[i].err
		require.ErrorIs(t, obsrep.Send(parentCtx, &requesttest.FakeRequest{Items: params[i].items}), params[i].err)
	}

	spans := tt.SpanRecorder.Ended()
	require.Len(t, spans, len(params))

	var sentSpans, failedToSendSpans int
	for i, span := range spans {
		assert.Equal(t, "exporter/"+exporterID.String()+"/traces", span.Name())
		switch {
		case params[i].err == nil:
			sentSpans += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(0)})
			assert.Equal(t, codes.Unset, span.Status().Code)
		case errors.Is(params[i].err, errFake):
			failedToSendSpans += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(int64(params[i].items))})
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		default:
			t.Fatalf("unexpected error: %v", params[i].err)
		}
	}

	metadatatest.AssertEqualExporterSentSpans(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String("exporter", exporterID.String())),
				Value: int64(sentSpans),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

	if failedToSendSpans > 0 {
		metadatatest.AssertEqualExporterSendFailedSpans(t, tt,
			[]metricdata.DataPoint[int64]{
				{
					Attributes: attribute.NewSet(
						attribute.String("exporter", exporterID.String())),
					Value: int64(failedToSendSpans),
				},
			}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	}
}

func TestExportMetricsOp(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	parentCtx, parentSpan := tt.NewTelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
	defer parentSpan.End()

	var exporterErr error
	obsrep, err := newObsReportSender(
		exporter.Settings{ID: exporterID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		pipeline.SignalMetrics,
		sender.NewSender(func(context.Context, request.Request) error { return exporterErr }),
	)
	require.NoError(t, err)

	params := []testParams{
		{items: 17, err: nil},
		{items: 23, err: errFake},
	}
	for i := range params {
		exporterErr = params[i].err
		require.ErrorIs(t, obsrep.Send(parentCtx, &requesttest.FakeRequest{Items: params[i].items}), params[i].err)
	}

	spans := tt.SpanRecorder.Ended()
	require.Len(t, spans, len(params))

	var sentMetricPoints, failedToSendMetricPoints int
	for i, span := range spans {
		assert.Equal(t, "exporter/"+exporterID.String()+"/metrics", span.Name())
		switch {
		case params[i].err == nil:
			sentMetricPoints += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(0)})
			assert.Equal(t, codes.Unset, span.Status().Code)
		case errors.Is(params[i].err, errFake):
			failedToSendMetricPoints += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(int64(params[i].items))})
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		default:
			t.Fatalf("unexpected error: %v", params[i].err)
		}
	}

	metadatatest.AssertEqualExporterSentMetricPoints(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String("exporter", exporterID.String())),
				Value: int64(sentMetricPoints),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

	if failedToSendMetricPoints > 0 {
		metadatatest.AssertEqualExporterSendFailedMetricPoints(t, tt,
			[]metricdata.DataPoint[int64]{
				{
					Attributes: attribute.NewSet(
						attribute.String("exporter", exporterID.String())),
					Value: int64(failedToSendMetricPoints),
				},
			}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	}
}

func TestExportLogsOp(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	parentCtx, parentSpan := tt.NewTelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
	defer parentSpan.End()

	var exporterErr error
	obsrep, err := newObsReportSender(
		exporter.Settings{ID: exporterID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		pipeline.SignalLogs,
		sender.NewSender(func(context.Context, request.Request) error { return exporterErr }),
	)
	require.NoError(t, err)

	params := []testParams{
		{items: 17, err: nil},
		{items: 23, err: errFake},
	}
	for i := range params {
		exporterErr = params[i].err
		require.ErrorIs(t, obsrep.Send(parentCtx, &requesttest.FakeRequest{Items: params[i].items}), params[i].err)
	}

	spans := tt.SpanRecorder.Ended()
	require.Len(t, spans, len(params))

	var sentLogRecords, failedToSendLogRecords int
	for i, span := range spans {
		assert.Equal(t, "exporter/"+exporterID.String()+"/logs", span.Name())
		switch {
		case params[i].err == nil:
			sentLogRecords += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(0)})
			assert.Equal(t, codes.Unset, span.Status().Code)
		case errors.Is(params[i].err, errFake):
			failedToSendLogRecords += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(int64(params[i].items))})
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		default:
			t.Fatalf("unexpected error: %v", params[i].err)
		}
	}

	metadatatest.AssertEqualExporterSentLogRecords(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String("exporter", exporterID.String())),
				Value: int64(sentLogRecords),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

	if failedToSendLogRecords > 0 {
		metadatatest.AssertEqualExporterSendFailedLogRecords(t, tt,
			[]metricdata.DataPoint[int64]{
				{
					Attributes: attribute.NewSet(
						attribute.String("exporter", exporterID.String())),
					Value: int64(failedToSendLogRecords),
				},
			}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	}
}

func TestExportProfilesOp(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	parentCtx, parentSpan := tt.NewTelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
	defer parentSpan.End()

	var exporterErr error
	obsrep, err := newObsReportSender(
		exporter.Settings{ID: exporterID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		xpipeline.SignalProfiles,
		sender.NewSender(func(context.Context, request.Request) error { return exporterErr }),
	)
	require.NoError(t, err)

	params := []testParams{
		{items: 17, err: nil},
		{items: 23, err: errFake},
	}
	for i := range params {
		exporterErr = params[i].err
		require.ErrorIs(t, obsrep.Send(parentCtx, &requesttest.FakeRequest{Items: params[i].items}), params[i].err)
	}

	spans := tt.SpanRecorder.Ended()
	require.Len(t, spans, len(params))

	var sentProfileRecords, failedToSendProfileRecords int
	for i, span := range spans {
		assert.Equal(t, "exporter/"+exporterID.String()+"/profiles", span.Name())
		switch {
		case params[i].err == nil:
			sentProfileRecords += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(0)})
			assert.Equal(t, codes.Unset, span.Status().Code)
		case errors.Is(params[i].err, errFake):
			failedToSendProfileRecords += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(int64(params[i].items))})
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		default:
			t.Fatalf("unexpected error: %v", params[i].err)
		}
	}

	metadatatest.AssertEqualExporterSentProfileSamples(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String("exporter", exporterID.String())),
				Value: int64(sentProfileRecords),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

	if failedToSendProfileRecords > 0 {
		metadatatest.AssertEqualExporterSendFailedProfileSamples(t, tt,
			[]metricdata.DataPoint[int64]{
				{
					Attributes: attribute.NewSet(
						attribute.String("exporter", exporterID.String())),
					Value: int64(failedToSendProfileRecords),
				},
			}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	}
}

type testParams struct {
	items int
	err   error
}

func TestExportMetricsOpWithPartialError(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	parentCtx, parentSpan := tt.NewTelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
	defer parentSpan.End()

	var exporterErr error
	obsrep, err := newObsReportSender(
		exporter.Settings{ID: exporterID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		pipeline.SignalMetrics,
		sender.NewSender(func(context.Context, request.Request) error { return exporterErr }),
	)
	require.NoError(t, err)

	// Test partial error: 20 total items, 5 dropped
	totalItems := 20
	droppedItems := 5
	exporterErr = consumererror.NewPartialError(errors.New("some items dropped"), droppedItems)
	require.Error(t, obsrep.Send(parentCtx, &requesttest.FakeRequest{Items: totalItems}))

	spans := tt.SpanRecorder.Ended()
	require.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, "exporter/"+exporterID.String()+"/metrics", span.Name())
	// Verify that sent count is (total - dropped) and failed count is dropped
	require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(int64(totalItems - droppedItems))})
	require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(int64(droppedItems))})
	assert.Equal(t, codes.Error, span.Status().Code)

	metadatatest.AssertEqualExporterSentMetricPoints(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String("exporter", exporterID.String())),
				Value: int64(totalItems - droppedItems),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

	metadatatest.AssertEqualExporterSendFailedMetricPoints(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String("exporter", exporterID.String())),
				Value: int64(droppedItems),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

func TestToNumItems(t *testing.T) {
	tests := []struct {
		name       string
		items      int
		err        error
		wantSent   int64
		wantFailed int64
	}{
		{
			name:       "no error - all sent",
			items:      10,
			err:        nil,
			wantSent:   10,
			wantFailed: 0,
		},
		{
			name:       "error - all failed",
			items:      10,
			err:        errors.New("export failed"),
			wantSent:   0,
			wantFailed: 10,
		},
		{
			name:       "partial error - some sent, some failed",
			items:      20,
			err:        consumererror.NewPartialError(errors.New("partial failure"), 5),
			wantSent:   15,
			wantFailed: 5,
		},
		{
			name:       "partial error - all dropped",
			items:      10,
			err:        consumererror.NewPartialError(errors.New("all dropped"), 10),
			wantSent:   0,
			wantFailed: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sent, failed := toNumItems(tt.items, tt.err)
			assert.Equal(t, tt.wantSent, sent)
			assert.Equal(t, tt.wantFailed, failed)
		})
	}
}
