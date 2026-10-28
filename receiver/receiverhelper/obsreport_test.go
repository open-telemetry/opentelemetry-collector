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
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper/internal"
	"go.opentelemetry.io/collector/receiver/receiverhelper/internal/metadata"
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
	originalState := metadata.ReceiverhelperNewReceiverMetricsFeatureGate.IsEnabled()
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.ReceiverhelperNewReceiverMetricsFeatureGate.ID(), originalState))
	})

	for _, tc := range []struct {
		name    string
		enabled bool
	}{{"gate_enabled", true}, {"gate_disabled", false}} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, featuregate.GlobalRegistry().Set(metadata.ReceiverhelperNewReceiverMetricsFeatureGate.ID(), tc.enabled))
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
								attribute.String(internal.TransportKey, transport),
							),
							Value: int64(acceptedSpans),
						},
					}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
				metadatatest.AssertEqualReceiverRefusedSpans(t, tt,
					[]metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.String(internal.ReceiverKey, receiverID.String()),
								attribute.String(internal.TransportKey, transport),
							),
							Value: int64(refusedSpans),
						},
					}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
				metadatatest.AssertEqualReceiverFailedSpans(t, tt,
					[]metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.String(internal.ReceiverKey, receiverID.String()),
								attribute.String(internal.TransportKey, transport),
							),
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
								attribute.String("outcome", outcome),
							),
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
	originalState := metadata.ReceiverhelperNewReceiverMetricsFeatureGate.IsEnabled()
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.ReceiverhelperNewReceiverMetricsFeatureGate.ID(), originalState))
	})

	for _, tc := range []struct {
		name    string
		enabled bool
	}{{"gate_enabled", true}, {"gate_disabled", false}} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, featuregate.GlobalRegistry().Set(metadata.ReceiverhelperNewReceiverMetricsFeatureGate.ID(), tc.enabled))
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
								attribute.String(internal.TransportKey, transport),
							),
							Value: int64(acceptedLogRecords),
						},
					}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
				metadatatest.AssertEqualReceiverRefusedLogRecords(t, tt,
					[]metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.String(internal.ReceiverKey, receiverID.String()),
								attribute.String(internal.TransportKey, transport),
							),
							Value: int64(refusedLogRecords),
						},
					}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
				metadatatest.AssertEqualReceiverFailedLogRecords(t, tt,
					[]metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.String(internal.ReceiverKey, receiverID.String()),
								attribute.String(internal.TransportKey, transport),
							),
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
								attribute.String("outcome", outcome),
							),
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
	originalState := metadata.ReceiverhelperNewReceiverMetricsFeatureGate.IsEnabled()
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.ReceiverhelperNewReceiverMetricsFeatureGate.ID(), originalState))
	})

	for _, tc := range []struct {
		name    string
		enabled bool
	}{{"gate_enabled", true}, {"gate_disabled", false}} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, featuregate.GlobalRegistry().Set(metadata.ReceiverhelperNewReceiverMetricsFeatureGate.ID(), tc.enabled))
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
								attribute.String(internal.TransportKey, transport),
							),
							Value: int64(acceptedMetricPoints),
						},
					}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
				metadatatest.AssertEqualReceiverRefusedMetricPoints(t, tt,
					[]metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.String(internal.ReceiverKey, receiverID.String()),
								attribute.String(internal.TransportKey, transport),
							),
							Value: int64(refusedMetricPoints),
						},
					}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
				metadatatest.AssertEqualReceiverFailedMetricPoints(t, tt,
					[]metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.String(internal.ReceiverKey, receiverID.String()),
								attribute.String(internal.TransportKey, transport),
							),
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
								attribute.String("outcome", outcome),
							),
							Value: count,
						})
					}
					metadatatest.AssertEqualReceiverRequests(t, tt, expectedRequests, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
				}
			})
		})
	}
}

func TestReceiveProfilesOp(t *testing.T) {
	originalState := metadata.ReceiverhelperNewReceiverMetricsFeatureGate.IsEnabled()
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.ReceiverhelperNewReceiverMetricsFeatureGate.ID(), originalState))
	})

	for _, tc := range []struct {
		name    string
		enabled bool
	}{{"gate_enabled", true}, {"gate_disabled", false}} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, featuregate.GlobalRegistry().Set(metadata.ReceiverhelperNewReceiverMetricsFeatureGate.ID(), tc.enabled))
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
					ctx := rec.StartProfilesOp(parentCtx)
					assert.NotNil(t, ctx)
					rec.EndProfilesOp(ctx, format, params[i].items, param.err)
				}

				spans := tt.SpanRecorder.Ended()
				require.Len(t, spans, len(params))

				var acceptedSamples, refusedSamples, failedSamples int
				for i, span := range spans {
					assert.Equal(t, "receiver/"+receiverID.String()+"/ProfilesReceived", span.Name())
					err := params[i].err
					if err == nil {
						acceptedSamples += params[i].items
						require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.AcceptedProfileSamplesKey, Value: attribute.Int64Value(int64(params[i].items))})
						require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.RefusedProfileSamplesKey, Value: attribute.Int64Value(0)})
						require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.FailedProfileSamplesKey, Value: attribute.Int64Value(0)})
						assert.Equal(t, codes.Unset, span.Status().Code)
					} else {
						isDownstream := consumererror.IsDownstream(err)
						if !tc.enabled || (tc.enabled && isDownstream) {
							refusedSamples += params[i].items
							require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.RefusedProfileSamplesKey, Value: attribute.Int64Value(int64(params[i].items))})
							require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.FailedProfileSamplesKey, Value: attribute.Int64Value(0)})
						} else {
							failedSamples += params[i].items
							require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.RefusedProfileSamplesKey, Value: attribute.Int64Value(0)})
							require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.FailedProfileSamplesKey, Value: attribute.Int64Value(int64(params[i].items))})
						}
						require.Contains(t, span.Attributes(), attribute.KeyValue{Key: internal.AcceptedProfileSamplesKey, Value: attribute.Int64Value(0)})
						assert.Equal(t, codes.Error, span.Status().Code)
						assert.Equal(t, err.Error(), span.Status().Description)
					}
				}
				metadatatest.AssertEqualReceiverAcceptedProfileSamples(t, tt,
					[]metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.String(internal.ReceiverKey, receiverID.String()),
								attribute.String(internal.TransportKey, transport),
							),
							Value: int64(acceptedSamples),
						},
					}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
				metadatatest.AssertEqualReceiverRefusedProfileSamples(t, tt,
					[]metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.String(internal.ReceiverKey, receiverID.String()),
								attribute.String(internal.TransportKey, transport),
							),
							Value: int64(refusedSamples),
						},
					}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
				metadatatest.AssertEqualReceiverFailedProfileSamples(t, tt,
					[]metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.String(internal.ReceiverKey, receiverID.String()),
								attribute.String(internal.TransportKey, transport),
							),
							Value: int64(failedSamples),
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
								attribute.String("outcome", outcome),
							),
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
	originalState := metadata.ReceiverhelperNewReceiverMetricsFeatureGate.IsEnabled()
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.ReceiverhelperNewReceiverMetricsFeatureGate.ID(), originalState))
	})

	for _, tc := range []struct {
		name    string
		enabled bool
	}{{"gate_enabled", true}, {"gate_disabled", false}} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, featuregate.GlobalRegistry().Set(metadata.ReceiverhelperNewReceiverMetricsFeatureGate.ID(), tc.enabled))
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

func TestReceiveWithLongLivedCtxNoParent(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rec, err := NewObsReport(ObsReportSettings{
		ReceiverID:             receiverID,
		Transport:              transport,
		LongLivedCtx:           true,
		ReceiverCreateSettings: receiver.Settings{ID: receiverID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(t, err)

	ctx := rec.StartTracesOp(context.Background())
	rec.EndTracesOp(ctx, format, 7, nil)

	spans := tt.SpanRecorder.Ended()
	require.Len(t, spans, 1)
	assert.False(t, spans[0].Parent().IsValid())
	assert.Empty(t, spans[0].Links())
}

func BenchmarkStartTracesOpLongLivedCtx(b *testing.B) {
	rec, err := NewObsReport(ObsReportSettings{
		ReceiverID:             receiverID,
		Transport:              transport,
		LongLivedCtx:           true,
		ReceiverCreateSettings: receiver.Settings{ID: receiverID, TelemetrySettings: componenttest.NewNopTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(b, err)

	parentCtx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SpanID:     trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	}))

	for _, bc := range []struct {
		name string
		ctx  context.Context
	}{
		{"no_parent", context.Background()},
		{"with_parent", parentCtx},
	} {
		b.Run(bc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				rec.StartTracesOp(bc.ctx)
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
					attribute.String(internal.TransportKey, transport),
				),
				Value: int64(7),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	metadatatest.AssertEqualReceiverRefusedSpans(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(internal.ReceiverKey, receiverID.String()),
					attribute.String(internal.TransportKey, transport),
				),
				Value: int64(0),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	metadatatest.AssertEqualReceiverFailedSpans(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(internal.ReceiverKey, receiverID.String()),
					attribute.String(internal.TransportKey, transport),
				),
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
					attribute.String(internal.TransportKey, transport),
				),
				Value: int64(7),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	metadatatest.AssertEqualReceiverRefusedMetricPoints(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(internal.ReceiverKey, receiverID.String()),
					attribute.String(internal.TransportKey, transport),
				),
				Value: int64(0),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	metadatatest.AssertEqualReceiverFailedMetricPoints(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(internal.ReceiverKey, receiverID.String()),
					attribute.String(internal.TransportKey, transport),
				),
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
					attribute.String(internal.TransportKey, transport),
				),
				Value: int64(7),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	metadatatest.AssertEqualReceiverRefusedLogRecords(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(internal.ReceiverKey, receiverID.String()),
					attribute.String(internal.TransportKey, transport),
				),
				Value: int64(0),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	metadatatest.AssertEqualReceiverFailedLogRecords(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(internal.ReceiverKey, receiverID.String()),
					attribute.String(internal.TransportKey, transport),
				),
				Value: int64(0),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

func TestCheckReceiverProfilesViews(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rec, err := NewObsReport(ObsReportSettings{
		ReceiverID:             receiverID,
		Transport:              transport,
		ReceiverCreateSettings: receiver.Settings{ID: receiverID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(t, err)
	ctx := rec.StartProfilesOp(context.Background())
	require.NotNil(t, ctx)
	rec.EndProfilesOp(ctx, format, 7, nil)

	metadatatest.AssertEqualReceiverAcceptedProfileSamples(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(internal.ReceiverKey, receiverID.String()),
					attribute.String(internal.TransportKey, transport),
				),
				Value: int64(7),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	metadatatest.AssertEqualReceiverRefusedProfileSamples(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(internal.ReceiverKey, receiverID.String()),
					attribute.String(internal.TransportKey, transport),
				),
				Value: int64(0),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	metadatatest.AssertEqualReceiverFailedProfileSamples(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(internal.ReceiverKey, receiverID.String()),
					attribute.String(internal.TransportKey, transport),
				),
				Value: int64(0),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

func TestReceiveProfilesOpBySampleType(t *testing.T) {
	originalState := metadata.ReceiverhelperNewReceiverMetricsFeatureGate.IsEnabled()
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.ReceiverhelperNewReceiverMetricsFeatureGate.ID(), originalState))
	})

	samplesByType := map[string]int{
		"samples:count":       7,
		"off_cpu:nanoseconds": 2,
	}

	for _, tc := range []struct {
		name        string
		gateEnabled bool
		err         error
		wantMetric  string
	}{
		{name: "accepted", wantMetric: "accepted"},
		{name: "accepted_gate_enabled", gateEnabled: true, wantMetric: "accepted"},
		{name: "downstream_error", err: consumererror.NewDownstream(errFake), wantMetric: "refused"},
		{name: "downstream_error_gate_enabled", gateEnabled: true, err: consumererror.NewDownstream(errFake), wantMetric: "refused"},
		{name: "internal_error", err: errFake, wantMetric: "refused"},
		{name: "internal_error_gate_enabled", gateEnabled: true, err: errFake, wantMetric: "failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, featuregate.GlobalRegistry().Set(metadata.ReceiverhelperNewReceiverMetricsFeatureGate.ID(), tc.gateEnabled))
			testTelemetry(t, func(t *testing.T, tt *componenttest.Telemetry) {
				rec, err := NewObsReport(ObsReportSettings{
					ReceiverID:             receiverID,
					Transport:              transport,
					ReceiverCreateSettings: receiver.Settings{ID: receiverID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
				})
				require.NoError(t, err)
				ctx := rec.StartProfilesOp(context.Background())
				require.NotNil(t, ctx)
				rec.EndProfilesOpBySampleType(ctx, format, samplesByType, tc.err)

				spans := tt.SpanRecorder.Ended()
				require.Len(t, spans, 1)
				var wantAccepted, wantRefused, wantFailed int64
				switch tc.wantMetric {
				case "accepted":
					wantAccepted = 9
				case "refused":
					wantRefused = 9
				case "failed":
					wantFailed = 9
				}
				require.Contains(t, spans[0].Attributes(), attribute.KeyValue{Key: internal.AcceptedProfileSamplesKey, Value: attribute.Int64Value(wantAccepted)})
				require.Contains(t, spans[0].Attributes(), attribute.KeyValue{Key: internal.RefusedProfileSamplesKey, Value: attribute.Int64Value(wantRefused)})
				require.Contains(t, spans[0].Attributes(), attribute.KeyValue{Key: internal.FailedProfileSamplesKey, Value: attribute.Int64Value(wantFailed)})

				dataPoints := func(metric string) []metricdata.DataPoint[int64] {
					var dps []metricdata.DataPoint[int64]
					for sampleType, n := range samplesByType {
						value := int64(0)
						if metric == tc.wantMetric {
							value = int64(n)
						}
						dps = append(dps, metricdata.DataPoint[int64]{
							Attributes: attribute.NewSet(
								attribute.String(internal.ReceiverKey, receiverID.String()),
								attribute.String(internal.TransportKey, transport),
								attribute.String(internal.SampleTypeKey, sampleType),
							),
							Value: value,
						})
					}
					return dps
				}
				metadatatest.AssertEqualReceiverAcceptedProfileSamples(t, tt, dataPoints("accepted"), metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
				metadatatest.AssertEqualReceiverRefusedProfileSamples(t, tt, dataPoints("refused"), metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
				metadatatest.AssertEqualReceiverFailedProfileSamples(t, tt, dataPoints("failed"), metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
			})
		})
	}
}

func TestProfileSampleCountsByType(t *testing.T) {
	pd := pprofile.NewProfiles()
	strTable := pd.Dictionary().StringTable()
	strTable.Append("", "samples", "count", "off_cpu", "nanoseconds", "events")

	ps := pd.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles()
	for _, p := range []struct {
		typeIdx, unitIdx int32
		samples          int
	}{
		{typeIdx: 1, unitIdx: 2, samples: 3},
		{typeIdx: 3, unitIdx: 4, samples: 5},
		{typeIdx: 1, unitIdx: 2, samples: 4},
		{typeIdx: 5, unitIdx: 2, samples: 0},
		// Out of range sample type indices.
		{typeIdx: 42, unitIdx: -1, samples: 1},
		// Out of range unit index only.
		{typeIdx: 1, unitIdx: 42, samples: 1},
	} {
		profile := ps.AppendEmpty()
		profile.SampleType().SetTypeStrindex(p.typeIdx)
		profile.SampleType().SetUnitStrindex(p.unitIdx)
		for range p.samples {
			profile.Samples().AppendEmpty()
		}
	}

	assert.Equal(t, map[string]int{
		"samples:count":       7,
		"off_cpu:nanoseconds": 5,
		"events:count":        0,
		":":                   1,
		"samples:":            1,
	}, ProfileSampleCountsByType(pd))
	assert.Empty(t, ProfileSampleCountsByType(pprofile.NewProfiles()))
}

func testTelemetry(t *testing.T, testFunc func(t *testing.T, tt *componenttest.Telemetry)) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	testFunc(t, tt)
}
