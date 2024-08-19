// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorhelper

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configtelemetry"
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
	assert.Equal(t, nil, mp.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
}

func newTestMProcessor(retError error) ProcessMetricsFunc {
	return func(_ context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
		return md, retError
	}
}

func TestMetricsProcessor_RecordInOut(t *testing.T) {
	// Regardless of how many logs are ingested, emit just one
	mockAggregate := func(_ context.Context, _ pmetric.Metrics) (pmetric.Metrics, error) {
		md := pmetric.NewMetrics()
		md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptySum().DataPoints().AppendEmpty()
		return md, nil
	}

	incomingMetrics := pmetric.NewMetrics()
	dps := incomingMetrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptySum().DataPoints()

	// Add 3 data points to the incoming
	dps.AppendEmpty()
	dps.AppendEmpty()
	dps.AppendEmpty()

	metricReader := sdkmetric.NewManualReader()
	set := processortest.NewNopSettings()
	set.TelemetrySettings.MetricsLevel = configtelemetry.LevelNormal
	set.TelemetrySettings.MetricsLevel = configtelemetry.LevelBasic
	set.TelemetrySettings.LeveledMeterProvider = func(level configtelemetry.Level) metric.MeterProvider {
		if level >= configtelemetry.LevelBasic {
			return sdkmetric.NewMeterProvider(sdkmetric.WithReader(metricReader))
		}
		return nil
	}

	mp, err := NewMetricsProcessor(context.Background(), set, &testMetricsCfg, consumertest.NewNop(), mockAggregate)
	require.NoError(t, err)

	assert.NoError(t, mp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, mp.ConsumeMetrics(context.Background(), incomingMetrics))
	assert.NoError(t, mp.Shutdown(context.Background()))

	ownMetrics := new(metricdata.ResourceMetrics)
	require.NoError(t, metricReader.Collect(context.Background(), ownMetrics))

	require.Len(t, ownMetrics.ScopeMetrics, 1)
	require.Len(t, ownMetrics.ScopeMetrics[0].Metrics, 2)

	inMetric := ownMetrics.ScopeMetrics[0].Metrics[0]
	outMetric := ownMetrics.ScopeMetrics[0].Metrics[1]
	if strings.Contains(inMetric.Name, "outgoing") {
		inMetric, outMetric = outMetric, inMetric
	}

	metricdatatest.AssertAggregationsEqual(t, metricdata.Sum[int64]{
		Temporality: metricdata.CumulativeTemporality,
		IsMonotonic: true,
		DataPoints: []metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(attribute.KeyValue{
					Key:   attribute.Key("processor"),
					Value: attribute.StringValue(set.ID.String()),
				}),
				Value: 3,
			},
		},
	}, inMetric.Data, metricdatatest.IgnoreTimestamp())

	metricdatatest.AssertAggregationsEqual(t, metricdata.Sum[int64]{
		Temporality: metricdata.CumulativeTemporality,
		IsMonotonic: true,
		DataPoints: []metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(attribute.KeyValue{
					Key:   attribute.Key("processor"),
					Value: attribute.StringValue(set.ID.String()),
				}),
				Value: 1,
			},
		},
	}, outMetric.Data, metricdatatest.IgnoreTimestamp())
}
