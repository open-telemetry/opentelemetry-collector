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
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor/processortest"
)

var testTracesCfg = struct{}{}

func TestNewTracesProcessor(t *testing.T) {
	tp, err := NewTracesProcessor(context.Background(), processortest.NewNopSettings(), &testTracesCfg, consumertest.NewNop(), newTestTProcessor(nil))
	require.NoError(t, err)

	assert.True(t, tp.Capabilities().MutatesData)
	assert.NoError(t, tp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, tp.ConsumeTraces(context.Background(), ptrace.NewTraces()))
	assert.NoError(t, tp.Shutdown(context.Background()))
}

func TestNewTracesProcessor_WithOptions(t *testing.T) {
	want := errors.New("my_error")
	tp, err := NewTracesProcessor(context.Background(), processortest.NewNopSettings(), &testTracesCfg, consumertest.NewNop(), newTestTProcessor(nil),
		WithStart(func(context.Context, component.Host) error { return want }),
		WithShutdown(func(context.Context) error { return want }),
		WithCapabilities(consumer.Capabilities{MutatesData: false}))
	assert.NoError(t, err)

	assert.Equal(t, want, tp.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, want, tp.Shutdown(context.Background()))
	assert.False(t, tp.Capabilities().MutatesData)
}

func TestNewTracesProcessor_NilRequiredFields(t *testing.T) {
	_, err := NewTracesProcessor(context.Background(), processortest.NewNopSettings(), &testTracesCfg, consumertest.NewNop(), nil)
	assert.Error(t, err)
}

func TestNewTracesProcessor_ProcessTraceError(t *testing.T) {
	want := errors.New("my_error")
	tp, err := NewTracesProcessor(context.Background(), processortest.NewNopSettings(), &testTracesCfg, consumertest.NewNop(), newTestTProcessor(want))
	require.NoError(t, err)
	assert.Equal(t, want, tp.ConsumeTraces(context.Background(), ptrace.NewTraces()))
}

func TestNewTracesProcessor_ProcessTracesErrSkipProcessingData(t *testing.T) {
	tp, err := NewTracesProcessor(context.Background(), processortest.NewNopSettings(), &testTracesCfg, consumertest.NewNop(), newTestTProcessor(ErrSkipProcessingData))
	require.NoError(t, err)
	assert.Equal(t, nil, tp.ConsumeTraces(context.Background(), ptrace.NewTraces()))
}

func newTestTProcessor(retError error) ProcessTracesFunc {
	return func(_ context.Context, td ptrace.Traces) (ptrace.Traces, error) {
		return td, retError
	}
}

func TestTracesProcessor_RecordInOut(t *testing.T) {
	// Regardless of how many spans are ingested, emit just one
	mockAggregate := func(_ context.Context, _ ptrace.Traces) (ptrace.Traces, error) {
		td := ptrace.NewTraces()
		td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
		return td, nil
	}

	incomingTraces := ptrace.NewTraces()
	incomingSpans := incomingTraces.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans()

	// Add 3 records to the incoming
	incomingSpans.AppendEmpty()
	incomingSpans.AppendEmpty()
	incomingSpans.AppendEmpty()

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

	tp, err := NewTracesProcessor(context.Background(), set, &testLogsCfg, consumertest.NewNop(), mockAggregate)
	require.NoError(t, err)

	assert.NoError(t, tp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, tp.ConsumeTraces(context.Background(), incomingTraces))
	assert.NoError(t, tp.Shutdown(context.Background()))

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
