// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package samplereceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"go.opentelemetry.io/collector/cmd/mdatagen/internal/samplereceiver/internal/metadata"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

// TestGeneratedMetrics verifies that the internal/metadata API is generated correctly.
func TestGeneratedMetrics(t *testing.T) {
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopCreateSettings())
	m := mb.Emit()
	require.Equal(t, 0, m.ResourceMetrics().Len())
}

func TestComponentTelemetry(t *testing.T) {
	tt := setupTestTelemetry()
	factory := NewFactory()
	_, err := factory.CreateMetricsReceiver(context.Background(), tt.NewCreateSettings(), componenttest.NewNopHost(), new(consumertest.MetricsSink))
	require.NoError(t, err)
	tt.assertMetrics(t, []metricdata.Metrics{
		{
			Name:        "batch_size_trigger_send",
			Description: "Number of times the batch was sent due to a size trigger",
			Unit:        "1",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Value: 1,
					},
				},
			},
		},
		{
			Name:        "process_runtime_total_alloc_bytes",
			Description: "Cumulative bytes allocated for heap objects (see 'go doc runtime.MemStats.TotalAlloc')",
			Unit:        "By",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Value: 2,
					},
				},
			},
		},
	})
	require.NoError(t, tt.Shutdown(context.Background()))
}
