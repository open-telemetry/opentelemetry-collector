// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package samplereceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"go.opentelemetry.io/collector/cmd/mdatagen/internal/samplereceiver/internal/metadata"
	"go.opentelemetry.io/collector/cmd/mdatagen/internal/samplereceiver/internal/metadatatest"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

// TestGeneratedMetrics verifies that the internal/metadata API is generated correctly.
func TestGeneratedMetrics(t *testing.T) {
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type))
	m := mb.Emit()
	require.Equal(t, 0, m.ResourceMetrics().Len())
}

func TestComponentTelemetry(t *testing.T) {
	tt := componenttest.NewTelemetry()
	factory := NewFactory()
	receiver, err := factory.CreateMetrics(context.Background(), metadatatest.NewSettings(tt), newMdatagenNopHost(), new(consumertest.MetricsSink))
	require.NoError(t, err)
	metadatatest.AssertEqualBatchSizeTriggerSend(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Value: 1,
			},
		}, metricdatatest.IgnoreTimestamp())
	metadatatest.AssertEqualProcessRuntimeTotalAllocBytes(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Value: 2,
			},
		}, metricdatatest.IgnoreTimestamp())
	rcv, ok := receiver.(nopReceiver)
	require.True(t, ok)
	rcv.initOptionalMetric()
	metadatatest.AssertEqualQueueLength(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Value: 3,
			},
		}, metricdatatest.IgnoreTimestamp())
	require.NoError(t, tt.Shutdown(context.Background()))
}
