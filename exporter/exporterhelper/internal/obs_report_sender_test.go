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
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadatatest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
	"go.opentelemetry.io/collector/pipeline"
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
		{items: 10, err: NewDroppedItems("incompatible format", 5)}, // partial drop
		{items: 8, err: NewDroppedItems("unsupported", 8)},          // full drop
	}
	for i := range params {
		exporterErr = params[i].err
		require.ErrorIs(t, obsrep.Send(parentCtx, &requesttest.FakeRequest{Items: params[i].items}), params[i].err)
	}

	spans := tt.SpanRecorder.Ended()
	require.Len(t, spans, len(params))

	var sentSpans, failedToSendSpans, droppedSpans int
	for i, span := range spans {
		assert.Equal(t, "exporter/"+exporterID.String()+"/traces", span.Name())
		switch {
		case params[i].err == nil:
			sentSpans += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsDropped, Value: attribute.Int64Value(0)})
			assert.Equal(t, codes.Unset, span.Status().Code)
		case IsDroppedItems(params[i].err):
			droppedCount := GetDroppedCount(params[i].err)
			droppedSpans += droppedCount
			sentSpans += params[i].items - droppedCount
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(int64(params[i].items - droppedCount))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsDropped, Value: attribute.Int64Value(int64(droppedCount))})
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		case errors.Is(params[i].err, errFake):
			failedToSendSpans += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsDropped, Value: attribute.Int64Value(0)})
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

	if droppedSpans > 0 {
		metadatatest.AssertEqualExporterDroppedSpans(t, tt,
			[]metricdata.DataPoint[int64]{
				{
					Attributes: attribute.NewSet(
						attribute.String("exporter", exporterID.String())),
					Value: int64(droppedSpans),
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
		{items: 15, err: NewDroppedItems("non-monotonic delta", 7)}, // partial drop
	}
	for i := range params {
		exporterErr = params[i].err
		require.ErrorIs(t, obsrep.Send(parentCtx, &requesttest.FakeRequest{Items: params[i].items}), params[i].err)
	}

	spans := tt.SpanRecorder.Ended()
	require.Len(t, spans, len(params))

	var sentMetricPoints, failedToSendMetricPoints, droppedMetricPoints int
	for i, span := range spans {
		assert.Equal(t, "exporter/"+exporterID.String()+"/metrics", span.Name())
		switch {
		case params[i].err == nil:
			sentMetricPoints += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsDropped, Value: attribute.Int64Value(0)})
			assert.Equal(t, codes.Unset, span.Status().Code)
		case IsDroppedItems(params[i].err):
			droppedCount := GetDroppedCount(params[i].err)
			droppedMetricPoints += droppedCount
			sentMetricPoints += params[i].items - droppedCount
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(int64(params[i].items - droppedCount))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsDropped, Value: attribute.Int64Value(int64(droppedCount))})
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		case errors.Is(params[i].err, errFake):
			failedToSendMetricPoints += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsDropped, Value: attribute.Int64Value(0)})
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

	if droppedMetricPoints > 0 {
		metadatatest.AssertEqualExporterDroppedMetricPoints(t, tt,
			[]metricdata.DataPoint[int64]{
				{
					Attributes: attribute.NewSet(
						attribute.String("exporter", exporterID.String())),
					Value: int64(droppedMetricPoints),
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
		{items: 12, err: NewDroppedItems("incompatible schema", 4)}, // partial drop
	}
	for i := range params {
		exporterErr = params[i].err
		require.ErrorIs(t, obsrep.Send(parentCtx, &requesttest.FakeRequest{Items: params[i].items}), params[i].err)
	}

	spans := tt.SpanRecorder.Ended()
	require.Len(t, spans, len(params))

	var sentLogRecords, failedToSendLogRecords, droppedLogRecords int
	for i, span := range spans {
		assert.Equal(t, "exporter/"+exporterID.String()+"/logs", span.Name())
		switch {
		case params[i].err == nil:
			sentLogRecords += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsDropped, Value: attribute.Int64Value(0)})
			assert.Equal(t, codes.Unset, span.Status().Code)
		case IsDroppedItems(params[i].err):
			droppedCount := GetDroppedCount(params[i].err)
			droppedLogRecords += droppedCount
			sentLogRecords += params[i].items - droppedCount
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(int64(params[i].items - droppedCount))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsDropped, Value: attribute.Int64Value(int64(droppedCount))})
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		case errors.Is(params[i].err, errFake):
			failedToSendLogRecords += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsDropped, Value: attribute.Int64Value(0)})
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

	if droppedLogRecords > 0 {
		metadatatest.AssertEqualExporterDroppedLogRecords(t, tt,
			[]metricdata.DataPoint[int64]{
				{
					Attributes: attribute.NewSet(
						attribute.String("exporter", exporterID.String())),
					Value: int64(droppedLogRecords),
				},
			}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	}
}

type testParams struct {
	items int
	err   error
}

func TestDroppedItemsError(t *testing.T) {
	// Test NewDroppedItems and Error()
	err := NewDroppedItems("test reason", 42)
	require.Error(t, err)
	assert.Equal(t, "items dropped: test reason", err.Error())

	// Test IsDroppedItems
	assert.True(t, IsDroppedItems(err))
	assert.False(t, IsDroppedItems(errFake))
	assert.False(t, IsDroppedItems(nil))

	// Test GetDroppedCount
	assert.Equal(t, 42, GetDroppedCount(err))
	assert.Equal(t, 0, GetDroppedCount(errFake))
	assert.Equal(t, 0, GetDroppedCount(nil))
}

func TestToNumItems(t *testing.T) {
	tests := []struct {
		name        string
		numItems    int
		err         error
		wantSent    int64
		wantFailed  int64
		wantDropped int64
	}{
		{
			name:        "successful export",
			numItems:    100,
			err:         nil,
			wantSent:    100,
			wantFailed:  0,
			wantDropped: 0,
		},
		{
			name:        "failed export with regular error",
			numItems:    50,
			err:         errFake,
			wantSent:    0,
			wantFailed:  50,
			wantDropped: 0,
		},
		{
			name:        "partial drop",
			numItems:    80,
			err:         NewDroppedItems("incompatible", 30),
			wantSent:    50,
			wantFailed:  0,
			wantDropped: 30,
		},
		{
			name:        "full drop",
			numItems:    60,
			err:         NewDroppedItems("unsupported", 60),
			wantSent:    0,
			wantFailed:  0,
			wantDropped: 60,
		},
		{
			name:        "zero count drop",
			numItems:    10,
			err:         NewDroppedItems("reason", 0),
			wantSent:    0,
			wantFailed:  0,
			wantDropped: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sent, failed, dropped := toNumItems(tt.numItems, tt.err)
			assert.Equal(t, tt.wantSent, sent)
			assert.Equal(t, tt.wantFailed, failed)
			assert.Equal(t, tt.wantDropped, dropped)
		})
	}
}
