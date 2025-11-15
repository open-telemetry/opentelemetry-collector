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
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/service/internal/obsconsumer"
)

type mockProfilesConsumer struct {
	err          error
	capabilities consumer.Capabilities
}

func (m *mockProfilesConsumer) ConsumeProfiles(_ context.Context, _ pprofile.Profiles) error {
	return m.err
}

func (m *mockProfilesConsumer) Capabilities() consumer.Capabilities {
	return m.capabilities
}

func TestProfilesNopWhenGateDisabled(t *testing.T) {
	setGateForTest(t, false)

	mp := sdkmetric.NewMeterProvider()
	meter := mp.Meter("test")
	itemCounter, err := meter.Int64Counter("item_counter")
	require.NoError(t, err)
	sizeCounter, err := meter.Int64Counter("size_counter")
	require.NoError(t, err)

	cons := consumertest.NewNop()
	require.Equal(t, cons, obsconsumer.NewProfiles(cons, obsconsumer.Settings{ItemCounter: itemCounter, SizeCounter: sizeCounter, Logger: zap.NewNop()}))
}

func TestProfilesItemsOnly(t *testing.T) {
	setGateForTest(t, true)

	ctx := context.Background()
	mockConsumer := &mockProfilesConsumer{}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	itemCounter, err := meter.Int64Counter("item_counter")
	require.NoError(t, err)
	sizeCounter, err := meter.Int64Counter("size_counter")
	require.NoError(t, err)
	sizeCounterDisabled := newDisabledCounter(sizeCounter)

	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	consumer := obsconsumer.NewProfiles(mockConsumer, obsconsumer.Settings{ItemCounter: itemCounter, SizeCounter: sizeCounterDisabled, Logger: logger})

	pd := pprofile.NewProfiles()
	r := pd.ResourceProfiles().AppendEmpty()
	sp := r.ScopeProfiles().AppendEmpty()
	sp.Profiles().AppendEmpty().Samples().AppendEmpty()

	err = consumer.ConsumeProfiles(ctx, pd)
	require.NoError(t, err)

	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 1)

	metric := rm.ScopeMetrics[0].Metrics[0]
	require.Equal(t, "item_counter", metric.Name)

	data := metric.Data.(metricdata.Sum[int64])
	require.Len(t, data.DataPoints, 1)
	require.Equal(t, int64(1), data.DataPoints[0].Value)

	attrs := data.DataPoints[0].Attributes
	require.Equal(t, 1, attrs.Len())
	val, ok := attrs.Value(attribute.Key(obsconsumer.ComponentOutcome))
	require.True(t, ok)
	require.Equal(t, "success", val.Emit())

	// Check that the logger was not called
	assert.Empty(t, logs.All())
}

func TestProfilesConsumeSuccess(t *testing.T) {
	setGateForTest(t, true)

	ctx := context.Background()
	mockConsumer := &mockProfilesConsumer{}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	itemCounter, err := meter.Int64Counter("item_counter")
	require.NoError(t, err)
	sizeCounter, err := meter.Int64Counter("size_counter")
	require.NoError(t, err)

	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	consumer := obsconsumer.NewProfiles(mockConsumer, obsconsumer.Settings{ItemCounter: itemCounter, SizeCounter: sizeCounter, Logger: logger})

	pd := pprofile.NewProfiles()
	r := pd.ResourceProfiles().AppendEmpty()
	sp := r.ScopeProfiles().AppendEmpty()
	sp.Profiles().AppendEmpty().Samples().AppendEmpty()

	err = consumer.ConsumeProfiles(ctx, pd)
	require.NoError(t, err)

	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 2)

	var itemMetric, sizeMetric metricdata.Metrics
	for _, m := range rm.ScopeMetrics[0].Metrics {
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

	sizeAttrs := sizeData.DataPoints[0].Attributes
	require.Equal(t, 1, sizeAttrs.Len())
	val, ok = sizeAttrs.Value(attribute.Key(obsconsumer.ComponentOutcome))
	require.True(t, ok)
	require.Equal(t, "success", val.Emit())

	// Check that the logger was not called
	assert.Empty(t, logs.All())
}

func TestProfilesConsumeFailure(t *testing.T) {
	setGateForTest(t, true)

	ctx := context.Background()
	expectedErr := errors.New("test error")
	downstreamErr := consumererror.NewDownstream(expectedErr)
	mockConsumer := &mockProfilesConsumer{err: expectedErr}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	itemCounter, err := meter.Int64Counter("item_counter")
	require.NoError(t, err)
	sizeCounter, err := meter.Int64Counter("size_counter")
	require.NoError(t, err)

	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	consumer := obsconsumer.NewProfiles(mockConsumer, obsconsumer.Settings{ItemCounter: itemCounter, SizeCounter: sizeCounter, Logger: logger})

	pd := pprofile.NewProfiles()
	r := pd.ResourceProfiles().AppendEmpty()
	sp := r.ScopeProfiles().AppendEmpty()
	sp.Profiles().AppendEmpty().Samples().AppendEmpty()

	err = consumer.ConsumeProfiles(ctx, pd)
	assert.Equal(t, downstreamErr, err)

	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 2)

	var itemMetric, sizeMetric metricdata.Metrics
	for _, m := range rm.ScopeMetrics[0].Metrics {
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

	// Check that the logger was called with an error
	require.Len(t, logs.All(), 1)
	assert.Contains(t, logs.All()[0].Message, "Profiles pipeline component had an error")
}

func TestProfilesWithStaticAttributes(t *testing.T) {
	setGateForTest(t, true)

	ctx := context.Background()
	mockConsumer := &mockProfilesConsumer{}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	itemCounter, err := meter.Int64Counter("item_counter")
	require.NoError(t, err)
	sizeCounter, err := meter.Int64Counter("size_counter")
	require.NoError(t, err)

	staticAttr := attribute.String("test", "value")
	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	consumer := obsconsumer.NewProfiles(mockConsumer, obsconsumer.Settings{ItemCounter: itemCounter, SizeCounter: sizeCounter, Logger: logger},
		obsconsumer.WithStaticDataPointAttribute(staticAttr))

	pd := pprofile.NewProfiles()
	r := pd.ResourceProfiles().AppendEmpty()
	sp := r.ScopeProfiles().AppendEmpty()
	sp.Profiles().AppendEmpty().Samples().AppendEmpty()

	err = consumer.ConsumeProfiles(ctx, pd)
	require.NoError(t, err)

	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 2)

	var itemMetric, sizeMetric metricdata.Metrics
	for _, m := range rm.ScopeMetrics[0].Metrics {
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

	// Check that the logger was not called
	assert.Empty(t, logs.All())
}

func TestProfilesMultipleItemsMixedOutcomes(t *testing.T) {
	setGateForTest(t, true)

	ctx := context.Background()
	expectedErr := errors.New("test error")
	downstreamErr := consumererror.NewDownstream(expectedErr)
	mockConsumer := &mockProfilesConsumer{}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	itemCounter, err := meter.Int64Counter("item_counter")
	require.NoError(t, err)
	sizeCounter, err := meter.Int64Counter("size_counter")
	require.NoError(t, err)

	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	consumer := obsconsumer.NewProfiles(mockConsumer, obsconsumer.Settings{ItemCounter: itemCounter, SizeCounter: sizeCounter, Logger: logger})

	// First batch: 2 successful items
	pd1 := pprofile.NewProfiles()
	for range 2 {
		r := pd1.ResourceProfiles().AppendEmpty()
		sp := r.ScopeProfiles().AppendEmpty()
		sp.Profiles().AppendEmpty().Samples().AppendEmpty()
	}
	err = consumer.ConsumeProfiles(ctx, pd1)
	require.NoError(t, err)

	// Second batch: 1 failed item
	mockConsumer.err = expectedErr
	pd2 := pprofile.NewProfiles()
	r := pd2.ResourceProfiles().AppendEmpty()
	sp := r.ScopeProfiles().AppendEmpty()
	sp.Profiles().AppendEmpty().Samples().AppendEmpty()
	err = consumer.ConsumeProfiles(ctx, pd2)
	assert.Equal(t, downstreamErr, err)

	// Third batch: 2 successful items
	mockConsumer.err = nil
	pd3 := pprofile.NewProfiles()
	for range 2 {
		r = pd3.ResourceProfiles().AppendEmpty()
		sp = r.ScopeProfiles().AppendEmpty()
		sp.Profiles().AppendEmpty().Samples().AppendEmpty()
	}
	err = consumer.ConsumeProfiles(ctx, pd3)
	require.NoError(t, err)

	// Fourth batch: 1 failed item
	mockConsumer.err = expectedErr
	pd4 := pprofile.NewProfiles()
	r = pd4.ResourceProfiles().AppendEmpty()
	sp = r.ScopeProfiles().AppendEmpty()
	sp.Profiles().AppendEmpty().Samples().AppendEmpty()
	err = consumer.ConsumeProfiles(ctx, pd4)
	assert.Equal(t, downstreamErr, err)

	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 2)

	var itemMetric, sizeMetric metricdata.Metrics
	for _, m := range rm.ScopeMetrics[0].Metrics {
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

	var successSizeDP, failureSizeDP metricdata.DataPoint[int64]
	for _, dp := range sizeData.DataPoints {
		val, ok := dp.Attributes.Value(attribute.Key(obsconsumer.ComponentOutcome))
		if ok && val.Emit() == "success" {
			successSizeDP = dp
		} else {
			failureSizeDP = dp
		}
	}
	require.Equal(t, int64(76), successSizeDP.Value)
	require.Equal(t, int64(40), failureSizeDP.Value)

	// Check that the logger was called for errors
	require.Len(t, logs.All(), 2)
	for _, log := range logs.All() {
		assert.Contains(t, log.Message, "Profiles pipeline component had an error")
	}
}

func TestProfilesCapabilities(t *testing.T) {
	setGateForTest(t, true)

	mockConsumer := &mockProfilesConsumer{
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
	consumer := obsconsumer.NewProfiles(mockConsumer, obsconsumer.Settings{ItemCounter: itemCounter, SizeCounter: sizeCounterDisabled, Logger: zap.NewNop()})
	require.Equal(t, consumer.Capabilities(), mockConsumer.capabilities)

	// Test with both counters
	consumer = obsconsumer.NewProfiles(mockConsumer, obsconsumer.Settings{ItemCounter: itemCounter, SizeCounter: sizeCounter, Logger: zap.NewNop()})
	require.Equal(t, consumer.Capabilities(), mockConsumer.capabilities)
}
