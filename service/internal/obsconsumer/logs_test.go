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
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/service/internal/obsconsumer"
)

type mockLogsConsumer struct {
	err          error
	capabilities consumer.Capabilities
}

func (m *mockLogsConsumer) ConsumeLogs(_ context.Context, _ plog.Logs) error {
	return m.err
}

func (m *mockLogsConsumer) Capabilities() consumer.Capabilities {
	return m.capabilities
}

func TestLogsConsumeSuccess(t *testing.T) {
	ctx := context.Background()
	mockConsumer := &mockLogsConsumer{}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")
	counter, err := meter.Int64Counter("test_counter")
	require.NoError(t, err)

	consumer := obsconsumer.NewLogs(mockConsumer, counter)

	ld := plog.NewLogs()
	r := ld.ResourceLogs().AppendEmpty()
	sl := r.ScopeLogs().AppendEmpty()
	sl.LogRecords().AppendEmpty()

	err = consumer.ConsumeLogs(ctx, ld)
	require.NoError(t, err)

	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 1)

	metric := rm.ScopeMetrics[0].Metrics[0]
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

func TestLogsConsumeFailure(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("test error")
	mockConsumer := &mockLogsConsumer{err: expectedErr}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")
	counter, err := meter.Int64Counter("test_counter")
	require.NoError(t, err)

	consumer := obsconsumer.NewLogs(mockConsumer, counter)

	ld := plog.NewLogs()
	r := ld.ResourceLogs().AppendEmpty()
	sl := r.ScopeLogs().AppendEmpty()
	sl.LogRecords().AppendEmpty()

	err = consumer.ConsumeLogs(ctx, ld)
	assert.Equal(t, expectedErr, err)

	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 1)

	metric := rm.ScopeMetrics[0].Metrics[0]
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

func TestLogsWithStaticAttributes(t *testing.T) {
	ctx := context.Background()
	mockConsumer := &mockLogsConsumer{}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")
	counter, err := meter.Int64Counter("test_counter")
	require.NoError(t, err)

	staticAttr := attribute.String("test", "value")
	consumer := obsconsumer.NewLogs(mockConsumer, counter, obsconsumer.WithStaticDataPointAttribute(staticAttr))

	ld := plog.NewLogs()
	r := ld.ResourceLogs().AppendEmpty()
	sl := r.ScopeLogs().AppendEmpty()
	sl.LogRecords().AppendEmpty()

	err = consumer.ConsumeLogs(ctx, ld)
	require.NoError(t, err)

	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 1)

	metric := rm.ScopeMetrics[0].Metrics[0]
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

func TestLogsMultipleItemsMixedOutcomes(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("test error")
	mockConsumer := &mockLogsConsumer{}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")
	counter, err := meter.Int64Counter("test_counter")
	require.NoError(t, err)

	consumer := obsconsumer.NewLogs(mockConsumer, counter)

	// First batch: 2 successful items
	ld1 := plog.NewLogs()
	for range 2 {
		r := ld1.ResourceLogs().AppendEmpty()
		sl := r.ScopeLogs().AppendEmpty()
		sl.LogRecords().AppendEmpty()
	}
	err = consumer.ConsumeLogs(ctx, ld1)
	require.NoError(t, err)

	// Second batch: 1 failed item
	mockConsumer.err = expectedErr
	ld2 := plog.NewLogs()
	r := ld2.ResourceLogs().AppendEmpty()
	sl := r.ScopeLogs().AppendEmpty()
	sl.LogRecords().AppendEmpty()
	err = consumer.ConsumeLogs(ctx, ld2)
	assert.Equal(t, expectedErr, err)

	// Third batch: 2 successful items
	mockConsumer.err = nil
	ld3 := plog.NewLogs()
	for range 2 {
		r = ld3.ResourceLogs().AppendEmpty()
		sl = r.ScopeLogs().AppendEmpty()
		sl.LogRecords().AppendEmpty()
	}
	err = consumer.ConsumeLogs(ctx, ld3)
	require.NoError(t, err)

	// Fourth batch: 1 failed item
	mockConsumer.err = expectedErr
	ld4 := plog.NewLogs()
	r = ld4.ResourceLogs().AppendEmpty()
	sl = r.ScopeLogs().AppendEmpty()
	sl.LogRecords().AppendEmpty()
	err = consumer.ConsumeLogs(ctx, ld4)
	assert.Equal(t, expectedErr, err)

	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 1)

	metric := rm.ScopeMetrics[0].Metrics[0]
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

func TestLogsCapabilities(t *testing.T) {
	mockConsumer := &mockLogsConsumer{
		capabilities: consumer.Capabilities{MutatesData: true},
	}
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")
	counter, err := meter.Int64Counter("test_counter")
	require.NoError(t, err)

	consumer := obsconsumer.NewLogs(mockConsumer, counter)
	require.Equal(t, consumer.Capabilities(), mockConsumer.capabilities)
}
