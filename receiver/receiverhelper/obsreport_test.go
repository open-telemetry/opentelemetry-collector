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
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/featuregate"
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
	originalState := NewReceiverMetricsGate.IsEnabled()
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(NewReceiverMetricsGate.ID(), originalState))
	})

	for _, tc := range []struct {
		name    string
		enabled bool
	}{{"gate_enabled", true}, {"gate_disabled", false}} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, featuregate.GlobalRegistry().Set(NewReceiverMetricsGate.ID(), tc.enabled))
			testTelemetry(t, func(t *testing.T, tt *componenttest.Telemetry) {
				parentCtx, parentSpan := tt.NewTelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
				defer parentSpan.End()

				params := []testParams{
					{items: 13, err: consumererror.NewDownstream(errFake)},
					{items: 42, err: nil},
					{items: 7, err: errors.New("non-downstream error")}, // Regular error to test numFailedErrors path
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

				var acceptedSpans, refusedSpans, failedSpans int
				for i, span := range spans {
					assert.Equal(t, "receiver/"+receiverID.String()+"/TraceDataReceived", span.Name())
					err := params[i].err
					if err == nil {
						acceptedSpans += params[i].items
						require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.AcceptedSpansKey, Value: attribute.Int64Value(int64(params[i].items))})
						require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.RefusedSpansKey, Value: attribute.Int64Value(0)})
						require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.FailedSpansKey, Value: attribute.Int64Value(0)})
						assert.Equal(t, codes.Unset, span.Status().Code)
					} else {
						isDownstream := consumererror.IsDownstream(err)
						if !tc.enabled || (tc.enabled && isDownstream) {
							refusedSpans += params[i].items
							require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.RefusedSpansKey, Value: attribute.Int64Value(int64(params[i].items))})
							require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.FailedSpansKey, Value: attribute.Int64Value(0)})
						} else {
							failedSpans += params[i].items
							require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.RefusedSpansKey, Value: attribute.Int64Value(0)})
							require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.FailedSpansKey, Value: attribute.Int64Value(int64(params[i].items))})
						}
						require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.AcceptedSpansKey, Value: attribute.Int64Value(0)})
						assert.Equal(t, codes.Error, span.Status().Code)
						assert.Equal(t, err.Error(), span.Status().Description)
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
				metadatatest.AssertEqualReceiverFailedSpans(t, tt,
					[]metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.String(internal.ReceiverKey, receiverID.String()),
								attribute.String(internal.TransportKey, transport)),
							Value: int64(failedSpans),
						},
					}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

				// Assert otelcol_receiver_requests metric with outcome attribute
				if tc.enabled {
					outcomes := make(map[string]int64)
					for _, param := range params {
						var outcome string
						switch {
						case param.err == nil:
							outcome = "success"
						case consumererror.IsDownstream(param.err):
							outcome = "refused"
						default:
							outcome = "failure"
						}
						outcomes[outcome]++
					}
					var expectedRequests []metricdata.DataPoint[int64]
					for outcome, count := range outcomes {
						expectedRequests = append(expectedRequests, metricdata.DataPoint[int64]{
							Attributes: attribute.NewSet(
								attribute.String(internal.ReceiverKey, receiverID.String()),
								attribute.String(internal.TransportKey, transport),
								attribute.String("outcome", outcome)),
							Value: count,
						})
					}
					metadatatest.AssertEqualReceiverRequests(t, tt, expectedRequests, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
				}
			})
		})
	}
}

func TestReceiveLogsOp(t *testing.T) {
	originalState := NewReceiverMetricsGate.IsEnabled()
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(NewReceiverMetricsGate.ID(), originalState))
	})

	for _, tc := range []struct {
		name    string
		enabled bool
	}{{"gate_enabled", true}, {"gate_disabled", false}} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, featuregate.GlobalRegistry().Set(NewReceiverMetricsGate.ID(), tc.enabled))
			testTelemetry(t, func(t *testing.T, tt *componenttest.Telemetry) {
				parentCtx, parentSpan := tt.NewTelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
				defer parentSpan.End()

				params := []testParams{
					{items: 13, err: consumererror.NewDownstream(errFake)},
					{items: 42, err: nil},
					{items: 7, err: errors.New("non-downstream error")},
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

				var acceptedLogRecords, refusedLogRecords, failedLogRecords int
				for i, span := range spans {
					assert.Equal(t, "receiver/"+receiverID.String()+"/LogsReceived", span.Name())
					err := params[i].err
					if err == nil {
						acceptedLogRecords += params[i].items
						require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.AcceptedLogRecordsKey, Value: attribute.Int64Value(int64(params[i].items))})
						require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.RefusedLogRecordsKey, Value: attribute.Int64Value(0)})
						require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.FailedLogRecordsKey, Value: attribute.Int64Value(0)})
						assert.Equal(t, codes.Unset, span.Status().Code)
					} else {
						isDownstream := consumererror.IsDownstream(err)
						if !tc.enabled || (tc.enabled && isDownstream) {
							refusedLogRecords += params[i].items
							require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.RefusedLogRecordsKey, Value: attribute.Int64Value(int64(params[i].items))})
							require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.FailedLogRecordsKey, Value: attribute.Int64Value(0)})
						} else {
							failedLogRecords += params[i].items
							require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.RefusedLogRecordsKey, Value: attribute.Int64Value(0)})
							require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.FailedLogRecordsKey, Value: attribute.Int64Value(int64(params[i].items))})
						}
						require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.AcceptedLogRecordsKey, Value: attribute.Int64Value(0)})
						assert.Equal(t, codes.Error, span.Status().Code)
						assert.Equal(t, err.Error(), span.Status().Description)
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
				metadatatest.AssertEqualReceiverFailedLogRecords(t, tt,
					[]metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.String(internal.ReceiverKey, receiverID.String()),
								attribute.String(internal.TransportKey, transport)),
							Value: int64(failedLogRecords),
						},
					}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

				// Assert otelcol_receiver_requests metric with outcome attribute
				if tc.enabled {
					outcomes := make(map[string]int64)
					for _, param := range params {
						var outcome string
						switch {
						case param.err == nil:
							outcome = "success"
						case consumererror.IsDownstream(param.err):
							outcome = "refused"
						default:
							outcome = "failure"
						}
						outcomes[outcome]++
					}
					var expectedRequests []metricdata.DataPoint[int64]
					for outcome, count := range outcomes {
						expectedRequests = append(expectedRequests, metricdata.DataPoint[int64]{
							Attributes: attribute.NewSet(
								attribute.String(internal.ReceiverKey, receiverID.String()),
								attribute.String(internal.TransportKey, transport),
								attribute.String("outcome", outcome)),
							Value: count,
						})
					}
					metadatatest.AssertEqualReceiverRequests(t, tt, expectedRequests, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
				}
			})
		})
	}
}

func TestReceiveMetricsOp(t *testing.T) {
	originalState := NewReceiverMetricsGate.IsEnabled()
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(NewReceiverMetricsGate.ID(), originalState))
	})

	for _, tc := range []struct {
		name    string
		enabled bool
	}{{"gate_enabled", true}, {"gate_disabled", false}} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, featuregate.GlobalRegistry().Set(NewReceiverMetricsGate.ID(), tc.enabled))
			testTelemetry(t, func(t *testing.T, tt *componenttest.Telemetry) {
				parentCtx, parentSpan := tt.NewTelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
				defer parentSpan.End()

				params := []testParams{
					{items: 13, err: consumererror.NewDownstream(errFake)},
					{items: 42, err: nil},
					{items: 7, err: errors.New("non-downstream error")},
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

				var acceptedMetricPoints, refusedMetricPoints, failedMetricPoints int
				for i, span := range spans {
					assert.Equal(t, "receiver/"+receiverID.String()+"/MetricsReceived", span.Name())
					err := params[i].err
					if err == nil {
						acceptedMetricPoints += params[i].items
						require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.AcceptedMetricPointsKey, Value: attribute.Int64Value(int64(params[i].items))})
						require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.RefusedMetricPointsKey, Value: attribute.Int64Value(0)})
						require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.FailedMetricPointsKey, Value: attribute.Int64Value(0)})
						assert.Equal(t, codes.Unset, span.Status().Code)
					} else {
						isDownstream := consumererror.IsDownstream(err)
						if !tc.enabled || (tc.enabled && isDownstream) {
							refusedMetricPoints += params[i].items
							require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.RefusedMetricPointsKey, Value: attribute.Int64Value(int64(params[i].items))})
							require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.FailedMetricPointsKey, Value: attribute.Int64Value(0)})
						} else {
							failedMetricPoints += params[i].items
							require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.RefusedMetricPointsKey, Value: attribute.Int64Value(0)})
							require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.FailedMetricPointsKey, Value: attribute.Int64Value(int64(params[i].items))})
						}
						require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.AcceptedMetricPointsKey, Value: attribute.Int64Value(0)})
						assert.Equal(t, codes.Error, span.Status().Code)
						assert.Equal(t, err.Error(), span.Status().Description)
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
				metadatatest.AssertEqualReceiverFailedMetricPoints(t, tt,
					[]metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.String(internal.ReceiverKey, receiverID.String()),
								attribute.String(internal.TransportKey, transport)),
							Value: int64(failedMetricPoints),
						},
					}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

				// Assert otelcol_receiver_requests metric with outcome attribute
				if tc.enabled {
					outcomes := make(map[string]int64)
					for _, param := range params {
						var outcome string
						switch {
						case param.err == nil:
							outcome = "success"
						case consumererror.IsDownstream(param.err):
							outcome = "refused"
						default:
							outcome = "failure"
						}
						outcomes[outcome]++
					}
					var expectedRequests []metricdata.DataPoint[int64]
					for outcome, count := range outcomes {
						expectedRequests = append(expectedRequests, metricdata.DataPoint[int64]{
							Attributes: attribute.NewSet(
								attribute.String(internal.ReceiverKey, receiverID.String()),
								attribute.String(internal.TransportKey, transport),
								attribute.String("outcome", outcome)),
							Value: count,
						})
					}
					metadatatest.AssertEqualReceiverRequests(t, tt, expectedRequests, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
				}
			})
		})
	}
}

func TestReceiveWithLongLivedCtx(t *testing.T) {
	originalState := NewReceiverMetricsGate.IsEnabled()
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(NewReceiverMetricsGate.ID(), originalState))
	})

	for _, tc := range []struct {
		name    string
		enabled bool
	}{{"gate_enabled", true}, {"gate_disabled", false}} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, featuregate.GlobalRegistry().Set(NewReceiverMetricsGate.ID(), tc.enabled))
			tt := componenttest.NewTelemetry()
			t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

			longLivedCtx, parentSpan := tt.NewTelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
			defer parentSpan.End()

			params := []testParams{
				{items: 17, err: nil},
				{items: 23, err: consumererror.NewDownstream(errFake)},
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
					require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.FailedSpansKey, Value: attribute.Int64Value(0)})
					assert.Equal(t, codes.Unset, span.Status().Code)
				case consumererror.IsDownstream(params[i].err):
					require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.AcceptedSpansKey, Value: attribute.Int64Value(0)})
					// For downstream errors
					require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.RefusedSpansKey, Value: attribute.Int64Value(int64(params[i].items))})
					require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.FailedSpansKey, Value: attribute.Int64Value(0)})
					assert.Equal(t, codes.Error, span.Status().Code)
					assert.Equal(t, params[i].err.Error(), span.Status().Description)
				default:
					t.Fatalf("unexpected error: %v", params[i].err)
				}
			}
		})
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
	metadatatest.AssertEqualReceiverFailedSpans(t, tt,
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
	metadatatest.AssertEqualReceiverFailedMetricPoints(t, tt,
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
	metadatatest.AssertEqualReceiverFailedLogRecords(t, tt,
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
