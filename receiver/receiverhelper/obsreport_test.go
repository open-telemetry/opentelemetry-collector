// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receiverhelper

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper/internal"
	"go.opentelemetry.io/collector/receiver/receiverhelper/internal/metadatatest"
)

const (
	transport = "fakeTransport"
	format    = "fakeFormat"
)

var (
	receiverID = component.MustNewID("fakeReceiver")

	errFake = errors.New("errFake")
)

type testParams struct {
	items int
	err   error
}

func TestReceiveTraceDataOp(t *testing.T) {
	testTelemetry(t, func(t *testing.T, tt *componenttest.Telemetry) {
		parentCtx, parentSpan := tt.NewTelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
		defer parentSpan.End()

		params := []testParams{
			{items: 13, err: errFake},
			{items: 42, err: nil},
		}
		for i, param := range params {
			rec, err := newReceiver(ObsReportSettings{
				ReceiverID:             receiverID,
				Transport:              transport,
				ReceiverCreateSettings: receiver.Settings{ID: receiverID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
			})
			require.NoError(t, err)
			ctx := rec.StartTracesOp(parentCtx)
			assert.NotNil(t, ctx)
			rec.EndTracesOp(ctx, format, params[i].items, param.err)
		}

		spans := tt.SpanRecorder.Ended()
		require.Len(t, spans, len(params))

		var acceptedSpans, refusedSpans int
		for i, span := range spans {
			assert.Equal(t, "receiver/"+receiverID.String()+"/TraceDataReceived", span.Name())
			switch {
			case params[i].err == nil:
				acceptedSpans += params[i].items
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.AcceptedSpansKey, Value: attribute.Int64Value(int64(params[i].items))})
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.RefusedSpansKey, Value: attribute.Int64Value(0)})
				assert.Equal(t, codes.Unset, span.Status().Code)
			case errors.Is(params[i].err, errFake):
				refusedSpans += params[i].items
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.AcceptedSpansKey, Value: attribute.Int64Value(0)})
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.RefusedSpansKey, Value: attribute.Int64Value(int64(params[i].items))})
				assert.Equal(t, codes.Error, span.Status().Code)
				assert.Equal(t, params[i].err.Error(), span.Status().Description)
			default:
				t.Fatalf("unexpected param: %v", params[i])
			}
		}

		metadatatest.AssertEqualReceiverAcceptedSpans(t, tt,
			[]metricdata.DataPoint[int64]{
				{
					Attributes: attribute.NewSet(
						attribute.String(internal.ReceiverKey, receiverID.String()),
						attribute.String(internal.TransportKey, transport)),
					Value: int64(acceptedSpans),
				},
			}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
		metadatatest.AssertEqualReceiverRefusedSpans(t, tt,
			[]metricdata.DataPoint[int64]{
				{
					Attributes: attribute.NewSet(
						attribute.String(internal.ReceiverKey, receiverID.String()),
						attribute.String(internal.TransportKey, transport)),
					Value: int64(refusedSpans),
				},
			}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	})
}

func TestReceiveLogsOp(t *testing.T) {
	testTelemetry(t, func(t *testing.T, tt *componenttest.Telemetry) {
		parentCtx, parentSpan := tt.NewTelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
		defer parentSpan.End()

		params := []testParams{
			{items: 13, err: errFake},
			{items: 42, err: nil},
		}
		for i, param := range params {
			rec, err := newReceiver(ObsReportSettings{
				ReceiverID:             receiverID,
				Transport:              transport,
				ReceiverCreateSettings: receiver.Settings{ID: receiverID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
			})
			require.NoError(t, err)

			ctx := rec.StartLogsOp(parentCtx)
			assert.NotNil(t, ctx)
			rec.EndLogsOp(ctx, format, params[i].items, param.err)
		}

		spans := tt.SpanRecorder.Ended()
		require.Len(t, spans, len(params))

		var acceptedLogRecords, refusedLogRecords int
		for i, span := range spans {
			assert.Equal(t, "receiver/"+receiverID.String()+"/LogsReceived", span.Name())
			switch {
			case params[i].err == nil:
				acceptedLogRecords += params[i].items
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.AcceptedLogRecordsKey, Value: attribute.Int64Value(int64(params[i].items))})
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.RefusedLogRecordsKey, Value: attribute.Int64Value(0)})
				assert.Equal(t, codes.Unset, span.Status().Code)
			case errors.Is(params[i].err, errFake):
				refusedLogRecords += params[i].items
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.AcceptedLogRecordsKey, Value: attribute.Int64Value(0)})
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.RefusedLogRecordsKey, Value: attribute.Int64Value(int64(params[i].items))})
				assert.Equal(t, codes.Error, span.Status().Code)
				assert.Equal(t, params[i].err.Error(), span.Status().Description)
			default:
				t.Fatalf("unexpected param: %v", params[i])
			}
		}
		metadatatest.AssertEqualReceiverAcceptedLogRecords(t, tt,
			[]metricdata.DataPoint[int64]{
				{
					Attributes: attribute.NewSet(
						attribute.String(internal.ReceiverKey, receiverID.String()),
						attribute.String(internal.TransportKey, transport)),
					Value: int64(acceptedLogRecords),
				},
			}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
		metadatatest.AssertEqualReceiverRefusedLogRecords(t, tt,
			[]metricdata.DataPoint[int64]{
				{
					Attributes: attribute.NewSet(
						attribute.String(internal.ReceiverKey, receiverID.String()),
						attribute.String(internal.TransportKey, transport)),
					Value: int64(refusedLogRecords),
				},
			}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	})
}

func TestReceiveMetricsOp(t *testing.T) {
	testTelemetry(t, func(t *testing.T, tt *componenttest.Telemetry) {
		parentCtx, parentSpan := tt.NewTelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
		defer parentSpan.End()

		params := []testParams{
			{items: 23, err: errFake},
			{items: 29, err: nil},
		}
		for i, param := range params {
			rec, err := newReceiver(ObsReportSettings{
				ReceiverID:             receiverID,
				Transport:              transport,
				ReceiverCreateSettings: receiver.Settings{ID: receiverID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
			})
			require.NoError(t, err)

			ctx := rec.StartMetricsOp(parentCtx)
			assert.NotNil(t, ctx)
			rec.EndMetricsOp(ctx, format, params[i].items, param.err)
		}

		spans := tt.SpanRecorder.Ended()
		require.Len(t, spans, len(params))

		var acceptedMetricPoints, refusedMetricPoints int
		for i, span := range spans {
			assert.Equal(t, "receiver/"+receiverID.String()+"/MetricsReceived", span.Name())
			switch {
			case params[i].err == nil:
				acceptedMetricPoints += params[i].items
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.AcceptedMetricPointsKey, Value: attribute.Int64Value(int64(params[i].items))})
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.RefusedMetricPointsKey, Value: attribute.Int64Value(0)})
				assert.Equal(t, codes.Unset, span.Status().Code)
			case errors.Is(params[i].err, errFake):
				refusedMetricPoints += params[i].items
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.AcceptedMetricPointsKey, Value: attribute.Int64Value(0)})
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.RefusedMetricPointsKey, Value: attribute.Int64Value(int64(params[i].items))})
				assert.Equal(t, codes.Error, span.Status().Code)
				assert.Equal(t, params[i].err.Error(), span.Status().Description)
			default:
				t.Fatalf("unexpected param: %v", params[i])
			}
		}

		metadatatest.AssertEqualReceiverAcceptedMetricPoints(t, tt,
			[]metricdata.DataPoint[int64]{
				{
					Attributes: attribute.NewSet(
						attribute.String(internal.ReceiverKey, receiverID.String()),
						attribute.String(internal.TransportKey, transport)),
					Value: int64(acceptedMetricPoints),
				},
			}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
		metadatatest.AssertEqualReceiverRefusedMetricPoints(t, tt,
			[]metricdata.DataPoint[int64]{
				{
					Attributes: attribute.NewSet(
						attribute.String(internal.ReceiverKey, receiverID.String()),
						attribute.String(internal.TransportKey, transport)),
					Value: int64(refusedMetricPoints),
				},
			}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	})
}

func TestReceiveWithLongLivedCtx(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	longLivedCtx, parentSpan := tt.NewTelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
	defer parentSpan.End()

	params := []testParams{
		{items: 17, err: nil},
		{items: 23, err: errFake},
	}
	for i := range params {
		// Use a new context on each operation to simulate distinct operations
		// under the same long lived context.
		rec, rerr := NewObsReport(ObsReportSettings{
			ReceiverID:             receiverID,
			Transport:              transport,
			LongLivedCtx:           true,
			ReceiverCreateSettings: receiver.Settings{ID: receiverID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		})
		require.NoError(t, rerr)
		ctx := rec.StartTracesOp(longLivedCtx)
		assert.NotNil(t, ctx)
		rec.EndTracesOp(ctx, format, params[i].items, params[i].err)
	}

	spans := tt.SpanRecorder.Ended()
	require.Len(t, spans, len(params))

	for i, span := range spans {
		assert.False(t, span.Parent().IsValid())
		require.Len(t, span.Links(), 1)
		link := span.Links()[0]
		assert.Equal(t, parentSpan.SpanContext().TraceID(), link.SpanContext.TraceID())
		assert.Equal(t, parentSpan.SpanContext().SpanID(), link.SpanContext.SpanID())
		assert.Equal(t, "receiver/"+receiverID.String()+"/TraceDataReceived", span.Name())
		require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.TransportKey, Value: attribute.StringValue(transport)})
		switch {
		case params[i].err == nil:
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.AcceptedSpansKey, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.RefusedSpansKey, Value: attribute.Int64Value(0)})
			assert.Equal(t, codes.Unset, span.Status().Code)
		case errors.Is(params[i].err, errFake):
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.AcceptedSpansKey, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.RefusedSpansKey, Value: attribute.Int64Value(int64(params[i].items))})
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		default:
			t.Fatalf("unexpected error: %v", params[i].err)
		}
	}
}

func TestCheckReceiverTracesViews(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rec, err := NewObsReport(ObsReportSettings{
		ReceiverID:             receiverID,
		Transport:              transport,
		ReceiverCreateSettings: receiver.Settings{ID: receiverID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(t, err)
	ctx := rec.StartTracesOp(context.Background())
	require.NotNil(t, ctx)
	rec.EndTracesOp(ctx, format, 7, nil)

	metadatatest.AssertEqualReceiverAcceptedSpans(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(internal.ReceiverKey, receiverID.String()),
					attribute.String(internal.TransportKey, transport)),
				Value: int64(7),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	metadatatest.AssertEqualReceiverRefusedSpans(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(internal.ReceiverKey, receiverID.String()),
					attribute.String(internal.TransportKey, transport)),
				Value: int64(0),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

func TestCheckReceiverMetricsViews(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rec, err := NewObsReport(ObsReportSettings{
		ReceiverID:             receiverID,
		Transport:              transport,
		ReceiverCreateSettings: receiver.Settings{ID: receiverID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(t, err)
	ctx := rec.StartMetricsOp(context.Background())
	require.NotNil(t, ctx)
	rec.EndMetricsOp(ctx, format, 7, nil)

	metadatatest.AssertEqualReceiverAcceptedMetricPoints(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(internal.ReceiverKey, receiverID.String()),
					attribute.String(internal.TransportKey, transport)),
				Value: int64(7),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	metadatatest.AssertEqualReceiverRefusedMetricPoints(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(internal.ReceiverKey, receiverID.String()),
					attribute.String(internal.TransportKey, transport)),
				Value: int64(0),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

func TestCheckReceiverLogsViews(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rec, err := NewObsReport(ObsReportSettings{
		ReceiverID:             receiverID,
		Transport:              transport,
		ReceiverCreateSettings: receiver.Settings{ID: receiverID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(t, err)
	ctx := rec.StartLogsOp(context.Background())
	require.NotNil(t, ctx)
	rec.EndLogsOp(ctx, format, 7, nil)

	metadatatest.AssertEqualReceiverAcceptedLogRecords(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(internal.ReceiverKey, receiverID.String()),
					attribute.String(internal.TransportKey, transport)),
				Value: int64(7),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	metadatatest.AssertEqualReceiverRefusedLogRecords(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(internal.ReceiverKey, receiverID.String()),
					attribute.String(internal.TransportKey, transport)),
				Value: int64(0),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

func testTelemetry(t *testing.T, testFunc func(t *testing.T, tt *componenttest.Telemetry)) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	testFunc(t, tt)
}

func TestReceiverDurationMetricWithStartTimeForMetrics(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rec, err := NewObsReport(ObsReportSettings{
		ReceiverID:             receiverID,
		Transport:              transport,
		ReceiverCreateSettings: receiver.Settings{ID: receiverID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(t, err)

	// Test that duration metric is recorded when context has start time
	ctx := rec.StartMetricsOp(context.Background())
	require.NotNil(t, ctx)

	// Add a small delay to ensure measurable duration
	time.Sleep(10 * time.Millisecond)

	rec.EndMetricsOp(ctx, format, 7, nil)

	// Verify that duration metric was recorded
	// Use IgnoreValue() to ignore the exact duration value since it can vary
	metadatatest.AssertEqualReceiverDuration(t, tt,
		[]metricdata.HistogramDataPoint[float64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(internal.ReceiverKey, receiverID.String()),
					attribute.String(internal.TransportKey, transport)),
				Count: 1,
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars(), metricdatatest.IgnoreValue())
}

func TestReceiverDurationMetricWithoutStartTimeForMetrics(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rec, err := NewObsReport(ObsReportSettings{
		ReceiverID:             receiverID,
		Transport:              transport,
		ReceiverCreateSettings: receiver.Settings{ID: receiverID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(t, err)

	// Test that duration metric is NOT recorded when context doesn't have start time
	ctx := context.Background() // Context without start time
	rec.EndMetricsOp(ctx, format, 7, nil)

	// Verify that no duration metric was recorded
	// Since the metric is not recorded when there's no start time, we expect an error
	// when trying to get the metric
	_, err = tt.GetMetric("otelcol_receiver_duration")
	require.Error(t, err)
	require.Contains(t, err.Error(), "metric 'otelcol_receiver_duration' not found")
}

func TestReceiverDurationMetricWithStartTimeForTraces(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rec, err := NewObsReport(ObsReportSettings{
		ReceiverID:             receiverID,
		Transport:              transport,
		ReceiverCreateSettings: receiver.Settings{ID: receiverID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(t, err)

	// Test that duration metric is recorded when context has start time for traces
	ctx := rec.StartTracesOp(context.Background())
	require.NotNil(t, ctx)

	// Add a small delay to ensure measurable duration
	time.Sleep(10 * time.Millisecond)

	rec.EndTracesOp(ctx, format, 7, nil)

	// Verify that duration metric was recorded
	metadatatest.AssertEqualReceiverDuration(t, tt,
		[]metricdata.HistogramDataPoint[float64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(internal.ReceiverKey, receiverID.String()),
					attribute.String(internal.TransportKey, transport)),
				Count: 1,
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars(), metricdatatest.IgnoreValue())
}

func TestReceiverDurationMetricWithoutStartTimeForTraces(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rec, err := NewObsReport(ObsReportSettings{
		ReceiverID:             receiverID,
		Transport:              transport,
		ReceiverCreateSettings: receiver.Settings{ID: receiverID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(t, err)

	// Test that duration metric is NOT recorded when context doesn't have start time for traces
	ctx := context.Background() // Context without start time
	rec.EndTracesOp(ctx, format, 7, nil)

	// Verify that no duration metric was recorded
	_, err = tt.GetMetric("otelcol_receiver_duration")
	require.Error(t, err)
	require.Contains(t, err.Error(), "metric 'otelcol_receiver_duration' not found")
}
func TestReceiverDurationMetricWithStartTimeForLogs(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rec, err := NewObsReport(ObsReportSettings{
		ReceiverID:             receiverID,
		Transport:              transport,
		ReceiverCreateSettings: receiver.Settings{ID: receiverID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(t, err)

	// Test that duration metric is recorded when context has start time for logs
	ctx := rec.StartLogsOp(context.Background())
	require.NotNil(t, ctx)

	// Add a small delay to ensure measurable duration
	time.Sleep(10 * time.Millisecond)

	rec.EndLogsOp(ctx, format, 7, nil)

	// Verify that duration metric was recorded
	metadatatest.AssertEqualReceiverDuration(t, tt,
		[]metricdata.HistogramDataPoint[float64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(internal.ReceiverKey, receiverID.String()),
					attribute.String(internal.TransportKey, transport)),
				Count: 1,
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars(), metricdatatest.IgnoreValue())
}

func TestReceiverDurationMetricWithoutStartTimeForLogs(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rec, err := NewObsReport(ObsReportSettings{
		ReceiverID:             receiverID,
		Transport:              transport,
		ReceiverCreateSettings: receiver.Settings{ID: receiverID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(t, err)

	// Test that duration metric is NOT recorded when context doesn't have start time for logs
	ctx := context.Background() // Context without start time
	rec.EndLogsOp(ctx, format, 7, nil)

	// Verify that no duration metric was recorded
	_, err = tt.GetMetric("otelcol_receiver_duration")
	require.Error(t, err)
	require.Contains(t, err.Error(), "metric 'otelcol_receiver_duration' not found")
}
