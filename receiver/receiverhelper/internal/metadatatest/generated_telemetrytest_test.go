// Code generated by mdatagen. DO NOT EDIT.

package metadatatest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/receiver/receiverhelper/internal/metadata"
)

func TestSetupTelemetry(t *testing.T) {
	testTel := componenttest.NewTelemetry()
	tb, err := metadata.NewTelemetryBuilder(testTel.NewTelemetrySettings())
	require.NoError(t, err)
	defer tb.Shutdown()
	tb.ReceiverAcceptedLogRecords.Add(context.Background(), 1)
	tb.ReceiverAcceptedMetricPoints.Add(context.Background(), 1)
	tb.ReceiverAcceptedSpans.Add(context.Background(), 1)
	tb.ReceiverInternalErrorsLogRecords.Add(context.Background(), 1)
	tb.ReceiverInternalErrorsMetricPoints.Add(context.Background(), 1)
	tb.ReceiverInternalErrorsSpans.Add(context.Background(), 1)
	tb.ReceiverRefusedLogRecords.Add(context.Background(), 1)
	tb.ReceiverRefusedMetricPoints.Add(context.Background(), 1)
	tb.ReceiverRefusedSpans.Add(context.Background(), 1)
	AssertEqualReceiverAcceptedLogRecords(t, testTel,
		[]metricdata.DataPoint[int64]{{Value: 1}},
		metricdatatest.IgnoreTimestamp())
	AssertEqualReceiverAcceptedMetricPoints(t, testTel,
		[]metricdata.DataPoint[int64]{{Value: 1}},
		metricdatatest.IgnoreTimestamp())
	AssertEqualReceiverAcceptedSpans(t, testTel,
		[]metricdata.DataPoint[int64]{{Value: 1}},
		metricdatatest.IgnoreTimestamp())
	AssertEqualReceiverInternalErrorsLogRecords(t, testTel,
		[]metricdata.DataPoint[int64]{{Value: 1}},
		metricdatatest.IgnoreTimestamp())
	AssertEqualReceiverInternalErrorsMetricPoints(t, testTel,
		[]metricdata.DataPoint[int64]{{Value: 1}},
		metricdatatest.IgnoreTimestamp())
	AssertEqualReceiverInternalErrorsSpans(t, testTel,
		[]metricdata.DataPoint[int64]{{Value: 1}},
		metricdatatest.IgnoreTimestamp())
	AssertEqualReceiverRefusedLogRecords(t, testTel,
		[]metricdata.DataPoint[int64]{{Value: 1}},
		metricdatatest.IgnoreTimestamp())
	AssertEqualReceiverRefusedMetricPoints(t, testTel,
		[]metricdata.DataPoint[int64]{{Value: 1}},
		metricdatatest.IgnoreTimestamp())
	AssertEqualReceiverRefusedSpans(t, testTel,
		[]metricdata.DataPoint[int64]{{Value: 1}},
		metricdatatest.IgnoreTimestamp())

	require.NoError(t, testTel.Shutdown(context.Background()))
}
