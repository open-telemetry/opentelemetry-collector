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
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/service/internal/obsconsumer"
)

type mockTracesConsumer struct {
	err          error
	capabilities consumer.Capabilities
}

func (m *mockTracesConsumer) ConsumeTraces(_ context.Context, _ ptrace.Traces) error {
	return m.err
}

func (m *mockTracesConsumer) Capabilities() consumer.Capabilities {
	return m.capabilities
}

func TestTracesConsumeSuccess(t *testing.T) {
	ctx := context.Background()
	mockConsumer := &mockTracesConsumer{}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")
	counter, err := meter.Int64Counter("test_counter")
	require.NoError(t, err)

	consumer := obsconsumer.NewTraces(mockConsumer, counter)

	td := ptrace.NewTraces()
	r := td.ResourceSpans().AppendEmpty()
	ss := r.ScopeSpans().AppendEmpty()
	ss.Spans().AppendEmpty()

	err = consumer.ConsumeTraces(ctx, td)
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

func TestTracesConsumeFailure(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("test error")
	mockConsumer := &mockTracesConsumer{err: expectedErr}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")
	counter, err := meter.Int64Counter("test_counter")
	require.NoError(t, err)

	consumer := obsconsumer.NewTraces(mockConsumer, counter)

	td := ptrace.NewTraces()
	r := td.ResourceSpans().AppendEmpty()
	ss := r.ScopeSpans().AppendEmpty()
	ss.Spans().AppendEmpty()

	err = consumer.ConsumeTraces(ctx, td)
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

func TestTracesWithStaticAttributes(t *testing.T) {
	ctx := context.Background()
	mockConsumer := &mockTracesConsumer{}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")
	counter, err := meter.Int64Counter("test_counter")
	require.NoError(t, err)

	staticAttr := attribute.String("test", "value")
	consumer := obsconsumer.NewTraces(mockConsumer, counter, obsconsumer.WithStaticDataPointAttribute(staticAttr))

	td := ptrace.NewTraces()
	r := td.ResourceSpans().AppendEmpty()
	ss := r.ScopeSpans().AppendEmpty()
	ss.Spans().AppendEmpty()

	err = consumer.ConsumeTraces(ctx, td)
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

func TestTracesMultipleItemsMixedOutcomes(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("test error")
	mockConsumer := &mockTracesConsumer{}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")
	counter, err := meter.Int64Counter("test_counter")
	require.NoError(t, err)

	consumer := obsconsumer.NewTraces(mockConsumer, counter)

	// First batch: 2 successful items
	td1 := ptrace.NewTraces()
	for range 2 {
		r := td1.ResourceSpans().AppendEmpty()
		ss := r.ScopeSpans().AppendEmpty()
		ss.Spans().AppendEmpty()
	}
	err = consumer.ConsumeTraces(ctx, td1)
	require.NoError(t, err)

	// Second batch: 1 failed item
	mockConsumer.err = expectedErr
	td2 := ptrace.NewTraces()
	r := td2.ResourceSpans().AppendEmpty()
	ss := r.ScopeSpans().AppendEmpty()
	ss.Spans().AppendEmpty()
	err = consumer.ConsumeTraces(ctx, td2)
	assert.Equal(t, expectedErr, err)

	// Third batch: 2 successful items
	mockConsumer.err = nil
	td3 := ptrace.NewTraces()
	for range 2 {
		r = td3.ResourceSpans().AppendEmpty()
		ss = r.ScopeSpans().AppendEmpty()
		ss.Spans().AppendEmpty()
	}
	err = consumer.ConsumeTraces(ctx, td3)
	require.NoError(t, err)

	// Fourth batch: 1 failed item
	mockConsumer.err = expectedErr
	td4 := ptrace.NewTraces()
	r = td4.ResourceSpans().AppendEmpty()
	ss = r.ScopeSpans().AppendEmpty()
	ss.Spans().AppendEmpty()
	err = consumer.ConsumeTraces(ctx, td4)
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

func TestTracesCapabilities(t *testing.T) {
	mockConsumer := &mockTracesConsumer{
		capabilities: consumer.Capabilities{MutatesData: true},
	}
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")
	counter, err := meter.Int64Counter("test_counter")
	require.NoError(t, err)

	consumer := obsconsumer.NewTraces(mockConsumer, counter)
	require.Equal(t, consumer.Capabilities(), mockConsumer.capabilities)
}
