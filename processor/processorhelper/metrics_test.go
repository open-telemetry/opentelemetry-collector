// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorhelper

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor/processorhelper/internal/metadatatest"
	"go.opentelemetry.io/collector/processor/processortest"
)

var testMetricsCfg = struct{}{}

func TestNewMetrics(t *testing.T) {
	mp, err := NewMetrics(context.Background(), processortest.NewNopSettings(), &testMetricsCfg, consumertest.NewNop(), newTestMProcessor(nil))
	require.NoError(t, err)

	assert.True(t, mp.Capabilities().MutatesData)
	assert.NoError(t, mp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, mp.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
	assert.NoError(t, mp.Shutdown(context.Background()))
}

func TestNewMetrics_WithOptions(t *testing.T) {
	want := errors.New("my_error")
	mp, err := NewMetrics(context.Background(), processortest.NewNopSettings(), &testMetricsCfg, consumertest.NewNop(), newTestMProcessor(nil),
		WithStart(func(context.Context, component.Host) error { return want }),
		WithShutdown(func(context.Context) error { return want }),
		WithCapabilities(consumer.Capabilities{MutatesData: false}))
	require.NoError(t, err)

	assert.Equal(t, want, mp.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, want, mp.Shutdown(context.Background()))
	assert.False(t, mp.Capabilities().MutatesData)
}

func TestNewMetrics_NilRequiredFields(t *testing.T) {
	_, err := NewMetrics(context.Background(), processortest.NewNopSettings(), &testMetricsCfg, consumertest.NewNop(), nil)
	assert.Error(t, err)
}

func TestNewMetrics_ProcessMetricsError(t *testing.T) {
	want := errors.New("my_error")
	mp, err := NewMetrics(context.Background(), processortest.NewNopSettings(), &testMetricsCfg, consumertest.NewNop(), newTestMProcessor(want))
	require.NoError(t, err)
	assert.Equal(t, want, mp.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
}

func TestNewMetrics_ProcessMetricsErrSkipProcessingData(t *testing.T) {
	mp, err := NewMetrics(context.Background(), processortest.NewNopSettings(), &testMetricsCfg, consumertest.NewNop(), newTestMProcessor(ErrSkipProcessingData))
	require.NoError(t, err)
	assert.NoError(t, mp.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
}

func newTestMProcessor(retError error) ProcessMetricsFunc {
	return func(_ context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
		return md, retError
	}
}

func TestMetricsConcurrency(t *testing.T) {
	metricsFunc := func(_ context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
		return md, nil
	}

	incomingMetrics := pmetric.NewMetrics()
	dps := incomingMetrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptySum().DataPoints()

	// Add 2 data points to the incoming
	dps.AppendEmpty()
	dps.AppendEmpty()

	mp, err := NewMetrics(context.Background(), processortest.NewNopSettings(), &testLogsCfg, consumertest.NewNop(), metricsFunc)
	require.NoError(t, err)
	assert.NoError(t, mp.Start(context.Background(), componenttest.NewNopHost()))

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10000; j++ {
				assert.NoError(t, mp.ConsumeMetrics(context.Background(), incomingMetrics))
			}
		}()
	}
	wg.Wait()
	assert.NoError(t, mp.Shutdown(context.Background()))
}

func TestMetrics_RecordInOut(t *testing.T) {
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

	testTelemetry := metadatatest.SetupTelemetry()
	mp, err := NewMetrics(context.Background(), testTelemetry.NewSettings(), &testMetricsCfg, consumertest.NewNop(), mockAggregate)
	require.NoError(t, err)

	assert.NoError(t, mp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, mp.ConsumeMetrics(context.Background(), incomingMetrics))
	assert.NoError(t, mp.Shutdown(context.Background()))

	testTelemetry.AssertMetrics(t, []metricdata.Metrics{
		{
			Name:        "otelcol_processor_incoming_items",
			Description: "Number of items passed to the processor. [alpha]",
			Unit:        "{items}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Value:      2,
						Attributes: attribute.NewSet(attribute.String("processor", "processorhelper"), attribute.String("otel.signal", "metrics")),
					},
				},
			},
		},
		{
			Name:        "otelcol_processor_outgoing_items",
			Description: "Number of items emitted from the processor. [alpha]",
			Unit:        "{items}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Value:      3,
						Attributes: attribute.NewSet(attribute.String("processor", "processorhelper"), attribute.String("otel.signal", "metrics")),
					},
				},
			},
		},
	}, metricdatatest.IgnoreTimestamp())
}

func TestMetrics_RecordIn_ErrorOut(t *testing.T) {
	/// Regardless of input, return error
	mockErr := func(_ context.Context, _ pmetric.Metrics) (pmetric.Metrics, error) {
		return pmetric.NewMetrics(), errors.New("fake")
	}

	incomingMetrics := pmetric.NewMetrics()
	dps := incomingMetrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptySum().DataPoints()

	// Add 2 data points to the incoming
	dps.AppendEmpty()
	dps.AppendEmpty()

	testTelemetry := metadatatest.SetupTelemetry()
	mp, err := NewMetrics(context.Background(), testTelemetry.NewSettings(), &testMetricsCfg, consumertest.NewNop(), mockErr)
	require.NoError(t, err)

	require.NoError(t, mp.Start(context.Background(), componenttest.NewNopHost()))
	require.Error(t, mp.ConsumeMetrics(context.Background(), incomingMetrics))
	require.NoError(t, mp.Shutdown(context.Background()))

	testTelemetry.AssertMetrics(t, []metricdata.Metrics{
		{
			Name:        "otelcol_processor_incoming_items",
			Description: "Number of items passed to the processor. [alpha]",
			Unit:        "{items}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Value:      2,
						Attributes: attribute.NewSet(attribute.String("processor", "processorhelper"), attribute.String("otel.signal", "metrics")),
					},
				},
			},
		},
		{
			Name:        "otelcol_processor_outgoing_items",
			Description: "Number of items emitted from the processor. [alpha]",
			Unit:        "{items}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Value:      0,
						Attributes: attribute.NewSet(attribute.String("processor", "processorhelper"), attribute.String("otel.signal", "metrics")),
					},
				},
			},
		},
	}, metricdatatest.IgnoreTimestamp())
}
