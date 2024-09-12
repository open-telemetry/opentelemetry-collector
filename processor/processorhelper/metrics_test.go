// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorhelper

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor/processortest"
)

var testMetricsCfg = struct{}{}

func TestNewMetricsProcessor(t *testing.T) {
	mp, err := NewMetricsProcessor(context.Background(), processortest.NewNopSettings(), &testMetricsCfg, consumertest.NewNop(), newTestMProcessor(nil))
	require.NoError(t, err)

	assert.True(t, mp.Capabilities().MutatesData)
	assert.NoError(t, mp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, mp.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
	assert.NoError(t, mp.Shutdown(context.Background()))
}

func TestNewMetricsProcessor_WithOptions(t *testing.T) {
	want := errors.New("my_error")
	mp, err := NewMetricsProcessor(context.Background(), processortest.NewNopSettings(), &testMetricsCfg, consumertest.NewNop(), newTestMProcessor(nil),
		WithStart(func(context.Context, component.Host) error { return want }),
		WithShutdown(func(context.Context) error { return want }),
		WithCapabilities(consumer.Capabilities{MutatesData: false}))
	assert.NoError(t, err)

	assert.Equal(t, want, mp.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, want, mp.Shutdown(context.Background()))
	assert.False(t, mp.Capabilities().MutatesData)
}

func TestNewMetricsProcessor_NilRequiredFields(t *testing.T) {
	_, err := NewMetricsProcessor(context.Background(), processortest.NewNopSettings(), &testMetricsCfg, consumertest.NewNop(), nil)
	assert.Error(t, err)
}

func TestNewMetricsProcessor_ProcessMetricsError(t *testing.T) {
	want := errors.New("my_error")
	mp, err := NewMetricsProcessor(context.Background(), processortest.NewNopSettings(), &testMetricsCfg, consumertest.NewNop(), newTestMProcessor(want))
	require.NoError(t, err)
	assert.Equal(t, want, mp.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
}

func TestNewMetricsProcessor_ProcessMetricsErrSkipProcessingData(t *testing.T) {
	mp, err := NewMetricsProcessor(context.Background(), processortest.NewNopSettings(), &testMetricsCfg, consumertest.NewNop(), newTestMProcessor(ErrSkipProcessingData))
	require.NoError(t, err)
	assert.NoError(t, mp.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
}

func newTestMProcessor(retError error) ProcessMetricsFunc {
	return func(_ context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
		return md, retError
	}
}

func TestMetricsProcessor_RecordInOut(t *testing.T) {
	// Regardless of how many data points are ingested, emit 3
	mockAggregate := func(_ context.Context, _ pmetric.Metrics) (pmetric.Metrics, error) {
		md := pmetric.NewMetrics()
		md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptySum().DataPoints().AppendEmpty()
		md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptySum().DataPoints().AppendEmpty()
		md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptySum().DataPoints().AppendEmpty()
		return md, nil
	}

	incomingMetrics := pmetric.NewMetrics()
	dps := incomingMetrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptySum().DataPoints()

	// Add 2 data points to the incoming
	dps.AppendEmpty()
	dps.AppendEmpty()

	testTelemetry := setupTestTelemetry()
	mp, err := NewMetricsProcessor(context.Background(), testTelemetry.NewSettings(), &testMetricsCfg, consumertest.NewNop(), mockAggregate)
	require.NoError(t, err)

	assert.NoError(t, mp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, mp.ConsumeMetrics(context.Background(), incomingMetrics))
	assert.NoError(t, mp.Shutdown(context.Background()))

	testTelemetry.assertMetrics(t, []metricdata.Metrics{
		{
			Name:        "otelcol_processor_incoming_metric_points",
			Description: "Number of metric points passed to the processor.",
			Unit:        "{datapoints}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Value:      2,
						Attributes: attribute.NewSet(attribute.String("processor", "processorhelper")),
					},
				},
			},
		},
		{
			Name:        "otelcol_processor_outgoing_metric_points",
			Description: "Number of metric points emitted from the processor.",
			Unit:        "{datapoints}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Value:      3,
						Attributes: attribute.NewSet(attribute.String("processor", "processorhelper")),
					},
				},
			},
		},
	})
}
