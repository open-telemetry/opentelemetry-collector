// Code generated by mdatagen. DO NOT EDIT.

package metadatatest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/processor/batchprocessor/internal/metadata"
)

func TestSetupTelemetry(t *testing.T) {
	testTel := componenttest.NewTelemetry()
	tb, err := metadata.NewTelemetryBuilder(testTel.NewTelemetrySettings())
	require.NoError(t, err)
	defer tb.Shutdown()
	require.NoError(t, tb.RegisterProcessorBatchMetadataCardinalityCallback(func(_ context.Context, observer metric.Int64Observer) error {
		observer.Observe(1)
		return nil
	}))
	tb.ProcessorBatchBatchSendSize.Record(context.Background(), 1)
	tb.ProcessorBatchBatchSendSizeBytes.Record(context.Background(), 1)
	tb.ProcessorBatchBatchSizeTriggerSend.Add(context.Background(), 1)
	tb.ProcessorBatchTimeoutTriggerSend.Add(context.Background(), 1)
	AssertEqualProcessorBatchBatchSendSize(t, testTel,
		[]metricdata.HistogramDataPoint[int64]{{}}, metricdatatest.IgnoreValue(),
		metricdatatest.IgnoreTimestamp())
	AssertEqualProcessorBatchBatchSendSizeBytes(t, testTel,
		[]metricdata.HistogramDataPoint[int64]{{}}, metricdatatest.IgnoreValue(),
		metricdatatest.IgnoreTimestamp())
	AssertEqualProcessorBatchBatchSizeTriggerSend(t, testTel,
		[]metricdata.DataPoint[int64]{{Value: 1}},
		metricdatatest.IgnoreTimestamp())
	AssertEqualProcessorBatchMetadataCardinality(t, testTel,
		[]metricdata.DataPoint[int64]{{Value: 1}},
		metricdatatest.IgnoreTimestamp())
	AssertEqualProcessorBatchTimeoutTriggerSend(t, testTel,
		[]metricdata.DataPoint[int64]{{Value: 1}},
		metricdatatest.IgnoreTimestamp())

	require.NoError(t, testTel.Shutdown(context.Background()))
}
