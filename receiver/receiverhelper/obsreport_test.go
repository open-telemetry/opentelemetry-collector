// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receiverhelper

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
	"go.opentelemetry.io/collector/receiver"
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
	testTelemetry(t, receiverID, func(t *testing.T, tt componenttest.TestTelemetry) {
		parentCtx, parentSpan := tt.TelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
		defer parentSpan.End()

		params := []testParams{
			{items: 13, err: errFake},
			{items: 42, err: nil},
		}
		for i, param := range params {
			rec, err := newReceiver(ObsReportSettings{
				ReceiverID:             receiverID,
				Transport:              transport,
				ReceiverCreateSettings: receiver.CreateSettings{ID: receiverID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
			})
			require.NoError(t, err)
			ctx := rec.StartTracesOp(parentCtx)
			assert.NotNil(t, ctx)
			rec.EndTracesOp(ctx, format, params[i].items, param.err)
		}

		spans := tt.SpanRecorder.Ended()
		require.Equal(t, len(params), len(spans))

		var acceptedSpans, refusedSpans int
		for i, span := range spans {
			assert.Equal(t, "receiver/"+receiverID.String()+"/TraceDataReceived", span.Name())
			switch {
			case params[i].err == nil:
				acceptedSpans += params[i].items
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.AcceptedSpansKey, Value: attribute.Int64Value(int64(params[i].items))})
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.RefusedSpansKey, Value: attribute.Int64Value(0)})
				assert.Equal(t, codes.Unset, span.Status().Code)
			case errors.Is(params[i].err, errFake):
				refusedSpans += params[i].items
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.AcceptedSpansKey, Value: attribute.Int64Value(0)})
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.RefusedSpansKey, Value: attribute.Int64Value(int64(params[i].items))})
				assert.Equal(t, codes.Error, span.Status().Code)
				assert.Equal(t, params[i].err.Error(), span.Status().Description)
			default:
				t.Fatalf("unexpected param: %v", params[i])
			}
		}
		require.NoError(t, tt.CheckReceiverTraces(transport, int64(acceptedSpans), int64(refusedSpans)))
	})
}

func TestReceiveLogsOp(t *testing.T) {
	testTelemetry(t, receiverID, func(t *testing.T, tt componenttest.TestTelemetry) {
		parentCtx, parentSpan := tt.TelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
		defer parentSpan.End()

		params := []testParams{
			{items: 13, err: errFake},
			{items: 42, err: nil},
		}
		for i, param := range params {
			rec, err := newReceiver(ObsReportSettings{
				ReceiverID:             receiverID,
				Transport:              transport,
				ReceiverCreateSettings: receiver.CreateSettings{ID: receiverID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
			})
			require.NoError(t, err)

			ctx := rec.StartLogsOp(parentCtx)
			assert.NotNil(t, ctx)
			rec.EndLogsOp(ctx, format, params[i].items, param.err)
		}

		spans := tt.SpanRecorder.Ended()
		require.Equal(t, len(params), len(spans))

		var acceptedLogRecords, refusedLogRecords int
		for i, span := range spans {
			assert.Equal(t, "receiver/"+receiverID.String()+"/LogsReceived", span.Name())
			switch {
			case params[i].err == nil:
				acceptedLogRecords += params[i].items
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.AcceptedLogRecordsKey, Value: attribute.Int64Value(int64(params[i].items))})
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.RefusedLogRecordsKey, Value: attribute.Int64Value(0)})
				assert.Equal(t, codes.Unset, span.Status().Code)
			case errors.Is(params[i].err, errFake):
				refusedLogRecords += params[i].items
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.AcceptedLogRecordsKey, Value: attribute.Int64Value(0)})
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.RefusedLogRecordsKey, Value: attribute.Int64Value(int64(params[i].items))})
				assert.Equal(t, codes.Error, span.Status().Code)
				assert.Equal(t, params[i].err.Error(), span.Status().Description)
			default:
				t.Fatalf("unexpected param: %v", params[i])
			}
		}
		require.NoError(t, tt.CheckReceiverLogs(transport, int64(acceptedLogRecords), int64(refusedLogRecords)))
	})
}

func TestReceiveMetricsOp(t *testing.T) {
	testTelemetry(t, receiverID, func(t *testing.T, tt componenttest.TestTelemetry) {
		parentCtx, parentSpan := tt.TelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
		defer parentSpan.End()

		params := []testParams{
			{items: 23, err: errFake},
			{items: 29, err: nil},
		}
		for i, param := range params {
			rec, err := newReceiver(ObsReportSettings{
				ReceiverID:             receiverID,
				Transport:              transport,
				ReceiverCreateSettings: receiver.CreateSettings{ID: receiverID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
			})
			require.NoError(t, err)

			ctx := rec.StartMetricsOp(parentCtx)
			assert.NotNil(t, ctx)
			rec.EndMetricsOp(ctx, format, params[i].items, param.err)
		}

		spans := tt.SpanRecorder.Ended()
		require.Equal(t, len(params), len(spans))

		var acceptedMetricPoints, refusedMetricPoints int
		for i, span := range spans {
			assert.Equal(t, "receiver/"+receiverID.String()+"/MetricsReceived", span.Name())
			switch {
			case params[i].err == nil:
				acceptedMetricPoints += params[i].items
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.AcceptedMetricPointsKey, Value: attribute.Int64Value(int64(params[i].items))})
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.RefusedMetricPointsKey, Value: attribute.Int64Value(0)})
				assert.Equal(t, codes.Unset, span.Status().Code)
			case errors.Is(params[i].err, errFake):
				refusedMetricPoints += params[i].items
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.AcceptedMetricPointsKey, Value: attribute.Int64Value(0)})
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.RefusedMetricPointsKey, Value: attribute.Int64Value(int64(params[i].items))})
				assert.Equal(t, codes.Error, span.Status().Code)
				assert.Equal(t, params[i].err.Error(), span.Status().Description)
			default:
				t.Fatalf("unexpected param: %v", params[i])
			}
		}

		require.NoError(t, tt.CheckReceiverMetrics(transport, int64(acceptedMetricPoints), int64(refusedMetricPoints)))
	})
}

func TestReceiveWithLongLivedCtx(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(receiverID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	longLivedCtx, parentSpan := tt.TelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
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
			ReceiverCreateSettings: receiver.CreateSettings{ID: receiverID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		})
		require.NoError(t, rerr)
		ctx := rec.StartTracesOp(longLivedCtx)
		assert.NotNil(t, ctx)
		rec.EndTracesOp(ctx, format, params[i].items, params[i].err)
	}

	spans := tt.SpanRecorder.Ended()
	require.Equal(t, len(params), len(spans))

	for i, span := range spans {
		assert.False(t, span.Parent().IsValid())
		require.Equal(t, 1, len(span.Links()))
		link := span.Links()[0]
		assert.Equal(t, parentSpan.SpanContext().TraceID(), link.SpanContext.TraceID())
		assert.Equal(t, parentSpan.SpanContext().SpanID(), link.SpanContext.SpanID())
		assert.Equal(t, "receiver/"+receiverID.String()+"/TraceDataReceived", span.Name())
		require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.TransportKey, Value: attribute.StringValue(transport)})
		switch {
		case params[i].err == nil:
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.AcceptedSpansKey, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.RefusedSpansKey, Value: attribute.Int64Value(0)})
			assert.Equal(t, codes.Unset, span.Status().Code)
		case errors.Is(params[i].err, errFake):
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.AcceptedSpansKey, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.RefusedSpansKey, Value: attribute.Int64Value(int64(params[i].items))})
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		default:
			t.Fatalf("unexpected error: %v", params[i].err)
		}
	}
}

func TestCheckReceiverTracesViews(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(receiverID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rec, err := NewObsReport(ObsReportSettings{
		ReceiverID:             receiverID,
		Transport:              transport,
		ReceiverCreateSettings: receiver.CreateSettings{ID: receiverID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(t, err)
	ctx := rec.StartTracesOp(context.Background())
	require.NotNil(t, ctx)
	rec.EndTracesOp(ctx, format, 7, nil)

	assert.NoError(t, tt.CheckReceiverTraces(transport, 7, 0))
	assert.Error(t, tt.CheckReceiverTraces(transport, 7, 7))
	assert.Error(t, tt.CheckReceiverTraces(transport, 0, 0))
	assert.Error(t, tt.CheckReceiverTraces(transport, 0, 7))
}

func TestCheckReceiverMetricsViews(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(receiverID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rec, err := NewObsReport(ObsReportSettings{
		ReceiverID:             receiverID,
		Transport:              transport,
		ReceiverCreateSettings: receiver.CreateSettings{ID: receiverID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(t, err)
	ctx := rec.StartMetricsOp(context.Background())
	require.NotNil(t, ctx)
	rec.EndMetricsOp(ctx, format, 7, nil)

	assert.NoError(t, tt.CheckReceiverMetrics(transport, 7, 0))
	assert.Error(t, tt.CheckReceiverMetrics(transport, 7, 7))
	assert.Error(t, tt.CheckReceiverMetrics(transport, 0, 0))
	assert.Error(t, tt.CheckReceiverMetrics(transport, 0, 7))
}

func TestCheckReceiverLogsViews(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(receiverID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rec, err := NewObsReport(ObsReportSettings{
		ReceiverID:             receiverID,
		Transport:              transport,
		ReceiverCreateSettings: receiver.CreateSettings{ID: receiverID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(t, err)
	ctx := rec.StartLogsOp(context.Background())
	require.NotNil(t, ctx)
	rec.EndLogsOp(ctx, format, 7, nil)

	assert.NoError(t, tt.CheckReceiverLogs(transport, 7, 0))
	assert.Error(t, tt.CheckReceiverLogs(transport, 7, 7))
	assert.Error(t, tt.CheckReceiverLogs(transport, 0, 0))
	assert.Error(t, tt.CheckReceiverLogs(transport, 0, 7))
}

func testTelemetry(t *testing.T, id component.ID, testFunc func(t *testing.T, tt componenttest.TestTelemetry)) {
	t.Run("WithOTel", func(t *testing.T) {
		tt, err := componenttest.SetupTelemetry(id)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

		testFunc(t, tt)
	})
}
