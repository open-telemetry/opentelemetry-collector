// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/service/internal/obsconsumer"
)

type mockMetricsConsumer struct {
	err          error
	capabilities consumer.Capabilities
}

func (m *mockMetricsConsumer) ConsumeMetrics(_ context.Context, _ pmetric.Metrics) error {
	return m.err
}

func (m *mockMetricsConsumer) Capabilities() consumer.Capabilities {
	return m.capabilities
}

func TestMetricsConsumeSuccess(t *testing.T) {
	ctx := context.Background()
	mockConsumer := &mockMetricsConsumer{}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")
	counter, err := meter.Int64Counter("test_counter")
	require.NoError(t, err)

	consumer := obsconsumer.NewMetrics(mockConsumer, counter)

	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()
	m.SetEmptyGauge().DataPoints().AppendEmpty()

	err = consumer.ConsumeMetrics(ctx, md)
	require.NoError(t, err)

	var metrics metricdata.ResourceMetrics
	err = reader.Collect(ctx, &metrics)
	require.NoError(t, err)
	require.Len(t, metrics.ScopeMetrics, 1)
	require.Len(t, metrics.ScopeMetrics[0].Metrics, 1)

	metric := metrics.ScopeMetrics[0].Metrics[0]
	require.Equal(t, "test_counter", metric.Name)

	data := metric.Data.(metricdata.Sum[int64])
	require.Len(t, data.DataPoints, 1)
	require.Equal(t, int64(1), data.DataPoints[0].Value)

	attrs := data.DataPoints[0].Attributes
	require.Equal(t, 1, attrs.Len())
	val, ok := attrs.Value(attribute.Key(obsconsumer.ComponentOutcome))
	require.True(t, ok)
	require.Equal(t, "success", val.Emit())
}

func TestMetricsConsumeFailure(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("test error")
	mockConsumer := &mockMetricsConsumer{err: expectedErr}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")
	counter, err := meter.Int64Counter("test_counter")
	require.NoError(t, err)

	consumer := obsconsumer.NewMetrics(mockConsumer, counter)

	md := pmetric.NewMetrics()
	r := md.ResourceMetrics().AppendEmpty()
	sm := r.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()
	m.SetEmptyGauge().DataPoints().AppendEmpty()

	err = consumer.ConsumeMetrics(ctx, md)
	assert.Equal(t, expectedErr, err)

	var metrics metricdata.ResourceMetrics
	err = reader.Collect(ctx, &metrics)
	require.NoError(t, err)
	require.Len(t, metrics.ScopeMetrics, 1)
	require.Len(t, metrics.ScopeMetrics[0].Metrics, 1)

	metric := metrics.ScopeMetrics[0].Metrics[0]
	require.Equal(t, "test_counter", metric.Name)

	data := metric.Data.(metricdata.Sum[int64])
	require.Len(t, data.DataPoints, 1)
	require.Equal(t, int64(1), data.DataPoints[0].Value)

	attrs := data.DataPoints[0].Attributes
	require.Equal(t, 1, attrs.Len())
	val, ok := attrs.Value(attribute.Key(obsconsumer.ComponentOutcome))
	require.True(t, ok)
	require.Equal(t, "failure", val.Emit())
}

func TestMetricsWithStaticAttributes(t *testing.T) {
	ctx := context.Background()
	mockConsumer := &mockMetricsConsumer{}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")
	counter, err := meter.Int64Counter("test_counter")
	require.NoError(t, err)

	staticAttr := attribute.String("test", "value")
	consumer := obsconsumer.NewMetrics(mockConsumer, counter, obsconsumer.WithStaticDataPointAttribute(staticAttr))

	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	rm.ScopeMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()
	m.SetEmptyGauge().DataPoints().AppendEmpty()

	err = consumer.ConsumeMetrics(ctx, md)
	require.NoError(t, err)

	var metrics metricdata.ResourceMetrics
	err = reader.Collect(ctx, &metrics)
	require.NoError(t, err)
	require.Len(t, metrics.ScopeMetrics, 1)
	require.Len(t, metrics.ScopeMetrics[0].Metrics, 1)

	metric := metrics.ScopeMetrics[0].Metrics[0]
	require.Equal(t, "test_counter", metric.Name)

	data := metric.Data.(metricdata.Sum[int64])
	require.Len(t, data.DataPoints, 1)
	require.Equal(t, int64(1), data.DataPoints[0].Value)

	attrs := data.DataPoints[0].Attributes
	require.Equal(t, 2, attrs.Len())
	val, ok := attrs.Value(attribute.Key("test"))
	require.True(t, ok)
	require.Equal(t, "value", val.Emit())
	val, ok = attrs.Value(attribute.Key(obsconsumer.ComponentOutcome))
	require.True(t, ok)
	require.Equal(t, "success", val.Emit())
}

func TestMetricsMultipleItemsMixedOutcomes(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("test error")
	mockConsumer := &mockMetricsConsumer{}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")
	counter, err := meter.Int64Counter("test_counter")
	require.NoError(t, err)

	consumer := obsconsumer.NewMetrics(mockConsumer, counter)

	// First batch: 2 successful items
	md1 := pmetric.NewMetrics()
	for range 2 {
		r := md1.ResourceMetrics().AppendEmpty()
		sm := r.ScopeMetrics().AppendEmpty()
		m := sm.Metrics().AppendEmpty()
		m.SetEmptyGauge().DataPoints().AppendEmpty()
	}
	err = consumer.ConsumeMetrics(ctx, md1)
	require.NoError(t, err)

	// Second batch: 1 failed item
	mockConsumer.err = expectedErr
	md2 := pmetric.NewMetrics()
	r := md2.ResourceMetrics().AppendEmpty()
	sm := r.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()
	m.SetEmptyGauge().DataPoints().AppendEmpty()
	err = consumer.ConsumeMetrics(ctx, md2)
	assert.Equal(t, expectedErr, err)

	// Third batch: 2 successful items
	mockConsumer.err = nil
	md3 := pmetric.NewMetrics()
	for range 2 {
		r = md3.ResourceMetrics().AppendEmpty()
		sm = r.ScopeMetrics().AppendEmpty()
		m = sm.Metrics().AppendEmpty()
		m.SetEmptyGauge().DataPoints().AppendEmpty()
	}
	err = consumer.ConsumeMetrics(ctx, md3)
	require.NoError(t, err)

	// Fourth batch: 1 failed item
	mockConsumer.err = expectedErr
	md4 := pmetric.NewMetrics()
	r = md4.ResourceMetrics().AppendEmpty()
	sm = r.ScopeMetrics().AppendEmpty()
	m = sm.Metrics().AppendEmpty()
	m.SetEmptyGauge().DataPoints().AppendEmpty()
	err = consumer.ConsumeMetrics(ctx, md4)
	assert.Equal(t, expectedErr, err)

	var metrics metricdata.ResourceMetrics
	err = reader.Collect(ctx, &metrics)
	require.NoError(t, err)
	require.Len(t, metrics.ScopeMetrics, 1)
	require.Len(t, metrics.ScopeMetrics[0].Metrics, 1)

	metric := metrics.ScopeMetrics[0].Metrics[0]
	require.Equal(t, "test_counter", metric.Name)

	data := metric.Data.(metricdata.Sum[int64])
	require.Len(t, data.DataPoints, 2)

	// Find success and failure data points
	var successDP, failureDP metricdata.DataPoint[int64]
	for _, dp := range data.DataPoints {
		val, ok := dp.Attributes.Value(attribute.Key(obsconsumer.ComponentOutcome))
		if ok && val.Emit() == "success" {
			successDP = dp
		} else {
			failureDP = dp
		}
	}

	require.Equal(t, int64(4), successDP.Value)
	require.Equal(t, int64(2), failureDP.Value)
}

func TestMetricsCapabilities(t *testing.T) {
	mockConsumer := &mockMetricsConsumer{
		capabilities: consumer.Capabilities{MutatesData: true},
	}
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")
	counter, err := meter.Int64Counter("test_counter")
	require.NoError(t, err)

	consumer := obsconsumer.NewMetrics(mockConsumer, counter)
	require.Equal(t, consumer.Capabilities(), mockConsumer.capabilities)
}
