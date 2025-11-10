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
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor/processorhelper/internal/metadatatest"
	"go.opentelemetry.io/collector/processor/processortest"
)

var testTracesCfg = struct{}{}

func TestNewTraces(t *testing.T) {
	tp, err := NewTraces(context.Background(), processortest.NewNopSettings(processortest.NopType), &testTracesCfg, consumertest.NewNop(), newTestTProcessor(nil))
	require.NoError(t, err)

	assert.True(t, tp.Capabilities().MutatesData)
	assert.NoError(t, tp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, tp.ConsumeTraces(context.Background(), ptrace.NewTraces()))
	assert.NoError(t, tp.Shutdown(context.Background()))
}

func TestNewTraces_WithOptions(t *testing.T) {
	want := errors.New("my_error")
	tp, err := NewTraces(context.Background(), processortest.NewNopSettings(processortest.NopType), &testTracesCfg, consumertest.NewNop(), newTestTProcessor(nil),
		WithStart(func(context.Context, component.Host) error { return want }),
		WithShutdown(func(context.Context) error { return want }),
		WithCapabilities(consumer.Capabilities{MutatesData: false}))
	require.NoError(t, err)

	assert.Equal(t, want, tp.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, want, tp.Shutdown(context.Background()))
	assert.False(t, tp.Capabilities().MutatesData)
}

func TestNewTraces_NilRequiredFields(t *testing.T) {
	_, err := NewTraces(context.Background(), processortest.NewNopSettings(processortest.NopType), &testTracesCfg, consumertest.NewNop(), nil)
	assert.Error(t, err)
}

func TestNewTraces_ProcessTraceError(t *testing.T) {
	want := errors.New("my_error")
	tp, err := NewTraces(context.Background(), processortest.NewNopSettings(processortest.NopType), &testTracesCfg, consumertest.NewNop(), newTestTProcessor(want))
	require.NoError(t, err)
	assert.Equal(t, want, tp.ConsumeTraces(context.Background(), ptrace.NewTraces()))
}

func TestNewTraces_ProcessTracesErrSkipProcessingData(t *testing.T) {
	tp, err := NewTraces(context.Background(), processortest.NewNopSettings(processortest.NopType), &testTracesCfg, consumertest.NewNop(), newTestTProcessor(ErrSkipProcessingData))
	require.NoError(t, err)
	assert.NoError(t, tp.ConsumeTraces(context.Background(), ptrace.NewTraces()))
}

func newTestTProcessor(retError error) ProcessTracesFunc {
	return func(_ context.Context, td ptrace.Traces) (ptrace.Traces, error) {
		return td, retError
	}
}

func TestTracesConcurrency(t *testing.T) {
	tracesFunc := func(_ context.Context, td ptrace.Traces) (ptrace.Traces, error) {
		return td, nil
	}

	incomingTraces := ptrace.NewTraces()
	incomingSpans := incomingTraces.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans()

	// Add 4 records to the incoming
	incomingSpans.AppendEmpty()
	incomingSpans.AppendEmpty()
	incomingSpans.AppendEmpty()
	incomingSpans.AppendEmpty()

	mp, err := NewTraces(context.Background(), processortest.NewNopSettings(processortest.NopType), &testLogsCfg, consumertest.NewNop(), tracesFunc)
	require.NoError(t, err)
	assert.NoError(t, mp.Start(context.Background(), componenttest.NewNopHost()))

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 10000 {
				assert.NoError(t, mp.ConsumeTraces(context.Background(), incomingTraces))
			}
		}()
	}
	wg.Wait()
	assert.NoError(t, mp.Shutdown(context.Background()))
}

func TestTraces_RecordInOut(t *testing.T) {
	// Regardless of how many spans are ingested, emit just one
	mockAggregate := func(_ context.Context, _ ptrace.Traces) (ptrace.Traces, error) {
		td := ptrace.NewTraces()
		td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
		return td, nil
	}

	incomingTraces := ptrace.NewTraces()
	incomingSpans := incomingTraces.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans()

	// Add 4 records to the incoming
	incomingSpans.AppendEmpty()
	incomingSpans.AppendEmpty()
	incomingSpans.AppendEmpty()
	incomingSpans.AppendEmpty()

	tel := componenttest.NewTelemetry()
	tp, err := NewTraces(context.Background(), newSettings(tel), &testLogsCfg, consumertest.NewNop(), mockAggregate)
	require.NoError(t, err)

	assert.NoError(t, tp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, tp.ConsumeTraces(context.Background(), incomingTraces))
	assert.NoError(t, tp.Shutdown(context.Background()))

	metadatatest.AssertEqualProcessorIncomingItems(t, tel,
		[]metricdata.DataPoint[int64]{
			{
				Value:      4,
				Attributes: attribute.NewSet(attribute.String("processor", "processorhelper"), attribute.String("otel.signal", "traces")),
			},
		}, metricdatatest.IgnoreTimestamp())
	metadatatest.AssertEqualProcessorOutgoingItems(t, tel,
		[]metricdata.DataPoint[int64]{
			{
				Value:      1,
				Attributes: attribute.NewSet(attribute.String("processor", "processorhelper"), attribute.String("otel.signal", "traces")),
			},
		}, metricdatatest.IgnoreTimestamp())
}

func TestTraces_RecordIn_ErrorOut(t *testing.T) {
	// Regardless of input, return error
	mockErr := func(_ context.Context, _ ptrace.Traces) (ptrace.Traces, error) {
		return ptrace.NewTraces(), errors.New("fake")
	}

	incomingTraces := ptrace.NewTraces()
	incomingSpans := incomingTraces.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans()

	// Add 4 records to the incoming
	incomingSpans.AppendEmpty()
	incomingSpans.AppendEmpty()
	incomingSpans.AppendEmpty()
	incomingSpans.AppendEmpty()

	tel := componenttest.NewTelemetry()
	tp, err := NewTraces(context.Background(), newSettings(tel), &testLogsCfg, consumertest.NewNop(), mockErr)
	require.NoError(t, err)

	require.NoError(t, tp.Start(context.Background(), componenttest.NewNopHost()))
	require.Error(t, tp.ConsumeTraces(context.Background(), incomingTraces))
	require.NoError(t, tp.Shutdown(context.Background()))

	metadatatest.AssertEqualProcessorIncomingItems(t, tel,
		[]metricdata.DataPoint[int64]{
			{
				Value:      4,
				Attributes: attribute.NewSet(attribute.String("processor", "processorhelper"), attribute.String("otel.signal", "traces")),
			},
		}, metricdatatest.IgnoreTimestamp())
	metadatatest.AssertEqualProcessorOutgoingItems(t, tel,
		[]metricdata.DataPoint[int64]{
			{
				Value:      0,
				Attributes: attribute.NewSet(attribute.String("processor", "processorhelper"), attribute.String("otel.signal", "traces")),
			},
		}, metricdatatest.IgnoreTimestamp())
}

func TestTraces_ProcessInternalDuration(t *testing.T) {
	mockAggregate := func(_ context.Context, _ ptrace.Traces) (ptrace.Traces, error) {
		td := ptrace.NewTraces()
		td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
		return td, nil
	}

	incomingTraces := ptrace.NewTraces()

	tel := componenttest.NewTelemetry()
	tp, err := NewTraces(context.Background(), newSettings(tel), &testLogsCfg, consumertest.NewNop(), mockAggregate)
	require.NoError(t, err)

	assert.NoError(t, tp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, tp.ConsumeTraces(context.Background(), incomingTraces))
	assert.NoError(t, tp.Shutdown(context.Background()))

	metadatatest.AssertEqualProcessorInternalDuration(t, tel,
		[]metricdata.HistogramDataPoint[float64]{
			{
				Count:        1,
				BucketCounts: []uint64{1},
				Attributes:   attribute.NewSet(attribute.String("processor", "processorhelper"), attribute.String("otel.signal", "traces")),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())
}
