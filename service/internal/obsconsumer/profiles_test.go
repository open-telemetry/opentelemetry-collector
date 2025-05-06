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

func TestProfilesConsumeSuccess(t *testing.T) {
	ctx := context.Background()
	mockConsumer := &mockProfilesConsumer{}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")
	counter, err := meter.Int64Counter("test_counter")
	require.NoError(t, err)

	consumer := obsconsumer.NewProfiles(mockConsumer, counter)

	pd := pprofile.NewProfiles()
	r := pd.ResourceProfiles().AppendEmpty()
	sp := r.ScopeProfiles().AppendEmpty()
	sp.Profiles().AppendEmpty().Sample().AppendEmpty()

	err = consumer.ConsumeProfiles(ctx, pd)
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

func TestProfilesConsumeFailure(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("test error")
	mockConsumer := &mockProfilesConsumer{err: expectedErr}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")
	counter, err := meter.Int64Counter("test_counter")
	require.NoError(t, err)

	consumer := obsconsumer.NewProfiles(mockConsumer, counter)

	pd := pprofile.NewProfiles()
	r := pd.ResourceProfiles().AppendEmpty()
	sp := r.ScopeProfiles().AppendEmpty()
	sp.Profiles().AppendEmpty().Sample().AppendEmpty()

	err = consumer.ConsumeProfiles(ctx, pd)
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

func TestProfilesWithStaticAttributes(t *testing.T) {
	ctx := context.Background()
	mockConsumer := &mockProfilesConsumer{}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")
	counter, err := meter.Int64Counter("test_counter")
	require.NoError(t, err)

	staticAttr := attribute.String("test", "value")
	consumer := obsconsumer.NewProfiles(mockConsumer, counter, obsconsumer.WithStaticDataPointAttribute(staticAttr))

	pd := pprofile.NewProfiles()
	r := pd.ResourceProfiles().AppendEmpty()
	sp := r.ScopeProfiles().AppendEmpty()
	sp.Profiles().AppendEmpty().Sample().AppendEmpty()

	err = consumer.ConsumeProfiles(ctx, pd)
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

func TestProfilesMultipleItemsMixedOutcomes(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("test error")
	mockConsumer := &mockProfilesConsumer{}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")
	counter, err := meter.Int64Counter("test_counter")
	require.NoError(t, err)

	consumer := obsconsumer.NewProfiles(mockConsumer, counter)

	// First batch: 2 successful items
	pd1 := pprofile.NewProfiles()
	for range 2 {
		r := pd1.ResourceProfiles().AppendEmpty()
		sp := r.ScopeProfiles().AppendEmpty()
		sp.Profiles().AppendEmpty().Sample().AppendEmpty()
	}
	err = consumer.ConsumeProfiles(ctx, pd1)
	require.NoError(t, err)

	// Second batch: 1 failed item
	mockConsumer.err = expectedErr
	pd2 := pprofile.NewProfiles()
	r := pd2.ResourceProfiles().AppendEmpty()
	sp := r.ScopeProfiles().AppendEmpty()
	sp.Profiles().AppendEmpty().Sample().AppendEmpty()
	err = consumer.ConsumeProfiles(ctx, pd2)
	assert.Equal(t, expectedErr, err)

	// Third batch: 2 successful items
	mockConsumer.err = nil
	pd3 := pprofile.NewProfiles()
	for range 2 {
		r = pd3.ResourceProfiles().AppendEmpty()
		sp = r.ScopeProfiles().AppendEmpty()
		sp.Profiles().AppendEmpty().Sample().AppendEmpty()
	}
	err = consumer.ConsumeProfiles(ctx, pd3)
	require.NoError(t, err)

	// Fourth batch: 1 failed item
	mockConsumer.err = expectedErr
	pd4 := pprofile.NewProfiles()
	r = pd4.ResourceProfiles().AppendEmpty()
	sp = r.ScopeProfiles().AppendEmpty()
	sp.Profiles().AppendEmpty().Sample().AppendEmpty()
	err = consumer.ConsumeProfiles(ctx, pd4)
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

func TestProfilesCapabilities(t *testing.T) {
	mockConsumer := &mockProfilesConsumer{
		capabilities: consumer.Capabilities{MutatesData: true},
	}
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")
	counter, err := meter.Int64Counter("test_counter")
	require.NoError(t, err)

	consumer := obsconsumer.NewProfiles(mockConsumer, counter)
	require.Equal(t, consumer.Capabilities(), mockConsumer.capabilities)
}
