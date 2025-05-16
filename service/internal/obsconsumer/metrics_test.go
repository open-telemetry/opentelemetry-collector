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
	"go.opentelemetry.io/collector/consumer/consumertest"
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

func TestMetricsNopWhenGateDisabled(t *testing.T) {
	setGateForTest(t, false)
	mp := sdkmetric.NewMeterProvider()
	meter := mp.Meter("test")
	itemCounter, err := meter.Int64Counter("item_counter")
	require.NoError(t, err)
	sizeCounter, err := meter.Int64Counter("size_counter")
	require.NoError(t, err)

	cons := consumertest.NewNop()
	require.Equal(t, cons, obsconsumer.NewMetrics(cons, itemCounter, sizeCounter))
}

func TestMetricsItemsOnly(t *testing.T) {
	setGateForTest(t, true)
	ctx := context.Background()
	mockConsumer := &mockMetricsConsumer{}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	itemCounter, err := meter.Int64Counter("item_counter")
	require.NoError(t, err)
	sizeCounter, err := meter.Int64Counter("size_counter")
	require.NoError(t, err)
	sizeCounterDisabled := newDisabledCounter(sizeCounter)

	consumer := obsconsumer.NewMetrics(mockConsumer, itemCounter, sizeCounterDisabled)

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
	require.Equal(t, "item_counter", metric.Name)

	data := metric.Data.(metricdata.Sum[int64])
	require.Len(t, data.DataPoints, 1)
	require.Equal(t, int64(1), data.DataPoints[0].Value)

	attrs := data.DataPoints[0].Attributes
	require.Equal(t, 1, attrs.Len())
	val, ok := attrs.Value(attribute.Key(obsconsumer.ComponentOutcome))
	require.True(t, ok)
	require.Equal(t, "success", val.Emit())
}

func TestMetricsConsumeSuccess(t *testing.T) {
	setGateForTest(t, true)

	ctx := context.Background()
	mockConsumer := &mockMetricsConsumer{}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	itemCounter, err := meter.Int64Counter("item_counter")
	require.NoError(t, err)
	sizeCounter, err := meter.Int64Counter("size_counter")
	require.NoError(t, err)

	consumer := obsconsumer.NewMetrics(mockConsumer, itemCounter, sizeCounter)

	md := pmetric.NewMetrics()
	r := md.ResourceMetrics().AppendEmpty()
	sm := r.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()
	m.SetEmptyGauge().DataPoints().AppendEmpty()

	err = consumer.ConsumeMetrics(ctx, md)
	require.NoError(t, err)

	var metrics metricdata.ResourceMetrics
	err = reader.Collect(ctx, &metrics)
	require.NoError(t, err)
	require.Len(t, metrics.ScopeMetrics, 1)
	require.Len(t, metrics.ScopeMetrics[0].Metrics, 2)

	var itemMetric, sizeMetric metricdata.Metrics
	for _, m := range metrics.ScopeMetrics[0].Metrics {
		switch m.Name {
		case "item_counter":
			itemMetric = m
		case "size_counter":
			sizeMetric = m
		}
	}
	require.NotNil(t, itemMetric)
	require.NotNil(t, sizeMetric)

	itemData := itemMetric.Data.(metricdata.Sum[int64])
	require.Len(t, itemData.DataPoints, 1)
	require.Equal(t, int64(1), itemData.DataPoints[0].Value)

	itemAttrs := itemData.DataPoints[0].Attributes
	require.Equal(t, 1, itemAttrs.Len())
	val, ok := itemAttrs.Value(attribute.Key(obsconsumer.ComponentOutcome))
	require.True(t, ok)
	require.Equal(t, "success", val.Emit())

	sizeData := sizeMetric.Data.(metricdata.Sum[int64])
	require.Len(t, sizeData.DataPoints, 1)
	require.Positive(t, sizeData.DataPoints[0].Value)

	attrs := sizeData.DataPoints[0].Attributes
	require.Equal(t, 1, attrs.Len())
	val, ok = attrs.Value(attribute.Key(obsconsumer.ComponentOutcome))
	require.True(t, ok)
	require.Equal(t, "success", val.Emit())
}

func TestMetricsConsumeFailure(t *testing.T) {
	setGateForTest(t, true)

	ctx := context.Background()
	expectedErr := errors.New("test error")
	mockConsumer := &mockMetricsConsumer{err: expectedErr}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	itemCounter, err := meter.Int64Counter("item_counter")
	require.NoError(t, err)
	sizeCounter, err := meter.Int64Counter("size_counter")
	require.NoError(t, err)

	consumer := obsconsumer.NewMetrics(mockConsumer, itemCounter, sizeCounter)

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
	require.Len(t, metrics.ScopeMetrics[0].Metrics, 2)

	// Find the item counter and size counter metrics
	var itemMetric, sizeMetric metricdata.Metrics
	for _, m := range metrics.ScopeMetrics[0].Metrics {
		switch m.Name {
		case "item_counter":
			itemMetric = m
		case "size_counter":
			sizeMetric = m
		}
	}
	require.NotNil(t, itemMetric)
	require.NotNil(t, sizeMetric)

	itemData := itemMetric.Data.(metricdata.Sum[int64])
	require.Len(t, itemData.DataPoints, 1)
	require.Equal(t, int64(1), itemData.DataPoints[0].Value)

	itemAttrs := itemData.DataPoints[0].Attributes
	require.Equal(t, 1, itemAttrs.Len())
	val, ok := itemAttrs.Value(attribute.Key(obsconsumer.ComponentOutcome))
	require.True(t, ok)
	require.Equal(t, "failure", val.Emit())

	sizeData := sizeMetric.Data.(metricdata.Sum[int64])
	require.Len(t, sizeData.DataPoints, 1)
	require.Positive(t, sizeData.DataPoints[0].Value)

	sizeAttrs := sizeData.DataPoints[0].Attributes
	require.Equal(t, 1, sizeAttrs.Len())
	val, ok = sizeAttrs.Value(attribute.Key(obsconsumer.ComponentOutcome))
	require.True(t, ok)
	require.Equal(t, "failure", val.Emit())
}

func TestMetricsWithStaticAttributes(t *testing.T) {
	setGateForTest(t, true)

	ctx := context.Background()
	mockConsumer := &mockMetricsConsumer{}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	itemCounter, err := meter.Int64Counter("item_counter")
	require.NoError(t, err)
	sizeCounter, err := meter.Int64Counter("size_counter")
	require.NoError(t, err)

	staticAttr := attribute.String("test", "value")
	consumer := obsconsumer.NewMetrics(mockConsumer, itemCounter, sizeCounter,
		obsconsumer.WithStaticDataPointAttribute(staticAttr))

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
	require.Len(t, metrics.ScopeMetrics[0].Metrics, 2)

	var itemMetric, sizeMetric metricdata.Metrics
	for _, m := range metrics.ScopeMetrics[0].Metrics {
		switch m.Name {
		case "item_counter":
			itemMetric = m
		case "size_counter":
			sizeMetric = m
		}
	}
	require.NotNil(t, itemMetric)
	require.NotNil(t, sizeMetric)

	itemData := itemMetric.Data.(metricdata.Sum[int64])
	require.Len(t, itemData.DataPoints, 1)
	require.Equal(t, int64(1), itemData.DataPoints[0].Value)

	itemAttrs := itemData.DataPoints[0].Attributes
	require.Equal(t, 2, itemAttrs.Len())
	val, ok := itemAttrs.Value(attribute.Key("test"))
	require.True(t, ok)
	require.Equal(t, "value", val.Emit())
	val, ok = itemAttrs.Value(attribute.Key(obsconsumer.ComponentOutcome))
	require.True(t, ok)
	require.Equal(t, "success", val.Emit())

	sizeData := sizeMetric.Data.(metricdata.Sum[int64])
	require.Len(t, sizeData.DataPoints, 1)
	require.Positive(t, sizeData.DataPoints[0].Value)

	sizeAttrs := sizeData.DataPoints[0].Attributes
	require.Equal(t, 2, sizeAttrs.Len())
	val, ok = sizeAttrs.Value(attribute.Key("test"))
	require.True(t, ok)
	require.Equal(t, "value", val.Emit())
	val, ok = sizeAttrs.Value(attribute.Key(obsconsumer.ComponentOutcome))
	require.True(t, ok)
	require.Equal(t, "success", val.Emit())
}

func TestMetricsMultipleItemsMixedOutcomes(t *testing.T) {
	setGateForTest(t, true)

	ctx := context.Background()
	expectedErr := errors.New("test error")
	mockConsumer := &mockMetricsConsumer{}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	itemCounter, err := meter.Int64Counter("item_counter")
	require.NoError(t, err)
	sizeCounter, err := meter.Int64Counter("size_counter")
	require.NoError(t, err)

	consumer := obsconsumer.NewMetrics(mockConsumer, itemCounter, sizeCounter)

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
	require.Len(t, metrics.ScopeMetrics[0].Metrics, 2)

	var itemMetric, sizeMetric metricdata.Metrics
	for _, m := range metrics.ScopeMetrics[0].Metrics {
		switch m.Name {
		case "item_counter":
			itemMetric = m
		case "size_counter":
			sizeMetric = m
		}
	}
	require.NotNil(t, itemMetric)
	require.NotNil(t, sizeMetric)

	itemData := itemMetric.Data.(metricdata.Sum[int64])
	require.Len(t, itemData.DataPoints, 2)
	sizeData := sizeMetric.Data.(metricdata.Sum[int64])
	require.Len(t, sizeData.DataPoints, 2)

	// Find success and failure data points
	var successDP, failureDP metricdata.DataPoint[int64]
	for _, dp := range itemData.DataPoints {
		val, ok := dp.Attributes.Value(attribute.Key(obsconsumer.ComponentOutcome))
		if ok && val.Emit() == "success" {
			successDP = dp
		} else {
			failureDP = dp
		}
	}
	require.Equal(t, int64(4), successDP.Value)
	require.Equal(t, int64(2), failureDP.Value)

	for _, dp := range sizeData.DataPoints {
		val, ok := dp.Attributes.Value(attribute.Key(obsconsumer.ComponentOutcome))
		if ok && val.Emit() == "success" {
			successDP = dp
		} else {
			failureDP = dp
		}
	}
	require.Equal(t, int64(56), successDP.Value)
	require.Equal(t, int64(28), failureDP.Value)
}

func TestMetricsCapabilities(t *testing.T) {
	setGateForTest(t, true)
	mockConsumer := &mockMetricsConsumer{
		capabilities: consumer.Capabilities{MutatesData: true},
	}
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	itemCounter, err := meter.Int64Counter("item_counter")
	require.NoError(t, err)
	sizeCounter, err := meter.Int64Counter("size_counter")
	require.NoError(t, err)
	sizeCounterDisabled := newDisabledCounter(sizeCounter)

	// Test with item counter only
	consumer := obsconsumer.NewMetrics(mockConsumer, itemCounter, sizeCounterDisabled)
	require.Equal(t, consumer.Capabilities(), mockConsumer.capabilities)

	// Test with both counters
	consumer = obsconsumer.NewMetrics(mockConsumer, itemCounter, sizeCounter)
	require.Equal(t, consumer.Capabilities(), mockConsumer.capabilities)
}
