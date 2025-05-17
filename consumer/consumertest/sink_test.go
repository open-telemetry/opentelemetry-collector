// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumertest

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestTracesSink(t *testing.T) {
	sink := new(TracesSink)
	td := testdata.GenerateTraces(1)
	want := make([]ptrace.Traces, 0, 7)
	for i := 0; i < 7; i++ {
		require.NoError(t, sink.ConsumeTraces(context.Background(), td))
		want = append(want, td)
	}
	assert.Equal(t, want, sink.AllTraces())
	assert.Equal(t, len(want), sink.SpanCount())
	sink.Reset()
	assert.Empty(t, sink.AllTraces())
	assert.Equal(t, 0, sink.SpanCount())
}

func TestMetricsSink(t *testing.T) {
	sink := new(MetricsSink)
	md := testdata.GenerateMetrics(1)
	want := make([]pmetric.Metrics, 0, 7)
	for i := 0; i < 7; i++ {
		require.NoError(t, sink.ConsumeMetrics(context.Background(), md))
		want = append(want, md)
	}
	assert.Equal(t, want, sink.AllMetrics())
	assert.Equal(t, 2*len(want), sink.DataPointCount())
	sink.Reset()
	assert.Empty(t, sink.AllMetrics())
	assert.Equal(t, 0, sink.DataPointCount())
}

func TestLogsSink(t *testing.T) {
	sink := new(LogsSink)
	md := testdata.GenerateLogs(1)
	want := make([]plog.Logs, 0, 7)
	for i := 0; i < 7; i++ {
		require.NoError(t, sink.ConsumeLogs(context.Background(), md))
		want = append(want, md)
	}
	assert.Equal(t, want, sink.AllLogs())
	assert.Equal(t, len(want), sink.LogRecordCount())
	sink.Reset()
	assert.Empty(t, sink.AllLogs())
	assert.Equal(t, 0, sink.LogRecordCount())
}

func TestProfilesSink(t *testing.T) {
	sink := new(ProfilesSink)
	td := testdata.GenerateProfiles(1)
	want := make([]pprofile.Profiles, 0, 7)
	for i := 0; i < 7; i++ {
		require.NoError(t, sink.ConsumeProfiles(context.Background(), td))
		want = append(want, td)
	}
	assert.Equal(t, want, sink.AllProfiles())
	assert.Equal(t, len(want), sink.SampleCount())
	sink.Reset()
	assert.Empty(t, sink.AllProfiles())
	assert.Empty(t, sink.SampleCount())
}

func TestTracesSinkWithContext(t *testing.T) {
	sink := new(TracesSink)
	td := testdata.GenerateTraces(1)
	want := make([]ptrace.Traces, 0, 7)
	wantCtx := make([]context.Context, 0, 7)

	for i := 0; i < 7; i++ {
		ctx := context.WithValue(context.Background(), fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
		require.NoError(t, sink.ConsumeTraces(ctx, td))
		want = append(want, td)
		wantCtx = append(wantCtx, ctx)
	}

	assert.Equal(t, want, sink.AllTraces())
	assert.Equal(t, len(want), sink.SpanCount())

	// Verify contexts
	gotCtx := sink.AllContexts()
	assert.Equal(t, len(wantCtx), len(gotCtx))
	for i, ctx := range gotCtx {
		assert.Equal(t, fmt.Sprintf("value-%d", i), ctx.Value(fmt.Sprintf("key-%d", i)))
	}

	sink.Reset()
	assert.Empty(t, sink.AllTraces())
	assert.Empty(t, sink.AllContexts())
	assert.Equal(t, 0, sink.SpanCount())
}

func TestMetricsSinkWithContext(t *testing.T) {
	sink := new(MetricsSink)
	md := testdata.GenerateMetrics(1)
	want := make([]pmetric.Metrics, 0, 7)
	wantCtx := make([]context.Context, 0, 7)

	for i := 0; i < 7; i++ {
		ctx := context.WithValue(context.Background(), fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
		require.NoError(t, sink.ConsumeMetrics(ctx, md))
		want = append(want, md)
		wantCtx = append(wantCtx, ctx)
	}

	assert.Equal(t, want, sink.AllMetrics())
	assert.Equal(t, 2*len(want), sink.DataPointCount())

	// Verify contexts
	gotCtx := sink.AllContexts()
	assert.Equal(t, len(wantCtx), len(gotCtx))
	for i, ctx := range gotCtx {
		assert.Equal(t, fmt.Sprintf("value-%d", i), ctx.Value(fmt.Sprintf("key-%d", i)))
	}

	sink.Reset()
	assert.Empty(t, sink.AllMetrics())
	assert.Empty(t, sink.AllContexts())
	assert.Equal(t, 0, sink.DataPointCount())
}

func TestLogsSinkWithContext(t *testing.T) {
	sink := new(LogsSink)
	md := testdata.GenerateLogs(1)
	want := make([]plog.Logs, 0, 7)
	wantCtx := make([]context.Context, 0, 7)

	for i := 0; i < 7; i++ {
		ctx := context.WithValue(context.Background(), fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
		require.NoError(t, sink.ConsumeLogs(ctx, md))
		want = append(want, md)
		wantCtx = append(wantCtx, ctx)
	}

	assert.Equal(t, want, sink.AllLogs())
	assert.Equal(t, len(want), sink.LogRecordCount())

	// Verify contexts
	gotCtx := sink.AllContexts()
	assert.Equal(t, len(wantCtx), len(gotCtx))
	for i, ctx := range gotCtx {
		assert.Equal(t, fmt.Sprintf("value-%d", i), ctx.Value(fmt.Sprintf("key-%d", i)))
	}

	sink.Reset()
	assert.Empty(t, sink.AllLogs())
	assert.Empty(t, sink.AllContexts())
	assert.Equal(t, 0, sink.LogRecordCount())
}

func TestProfilesSinkWithContext(t *testing.T) {
	sink := new(ProfilesSink)
	td := testdata.GenerateProfiles(1)
	want := make([]pprofile.Profiles, 0, 7)
	wantCtx := make([]context.Context, 0, 7)

	for i := 0; i < 7; i++ {
		ctx := context.WithValue(context.Background(), fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
		require.NoError(t, sink.ConsumeProfiles(ctx, td))
		want = append(want, td)
		wantCtx = append(wantCtx, ctx)
	}

	assert.Equal(t, want, sink.AllProfiles())
	assert.Equal(t, len(want), sink.SampleCount())

	// Verify contexts
	gotCtx := sink.AllContexts()
	assert.Equal(t, len(wantCtx), len(gotCtx))
	for i, ctx := range gotCtx {
		assert.Equal(t, fmt.Sprintf("value-%d", i), ctx.Value(fmt.Sprintf("key-%d", i)))
	}

	sink.Reset()
	assert.Empty(t, sink.AllProfiles())
	assert.Empty(t, sink.AllContexts())
	assert.Equal(t, 0, sink.SampleCount())
}

// TestSinkContextTransformation verifies that the context is stored and transformed correctly
func TestSinkContextTransformation(t *testing.T) {
	// Test cases for different sink types
	testCases := []struct {
		name string
		sink interface {
			AllContexts() []context.Context
		}
		consumeFunc func(interface{}, context.Context) error
		testData    interface{}
	}{
		{
			name: "TracesSink",
			sink: new(TracesSink),
			consumeFunc: func(sink interface{}, ctx context.Context) error {
				return sink.(*TracesSink).ConsumeTraces(ctx, testdata.GenerateTraces(1))
			},
		},
		{
			name: "MetricsSink",
			sink: new(MetricsSink),
			consumeFunc: func(sink interface{}, ctx context.Context) error {
				return sink.(*MetricsSink).ConsumeMetrics(ctx, testdata.GenerateMetrics(1))
			},
		},
		{
			name: "LogsSink",
			sink: new(LogsSink),
			consumeFunc: func(sink interface{}, ctx context.Context) error {
				return sink.(*LogsSink).ConsumeLogs(ctx, testdata.GenerateLogs(1))
			},
		},
		{
			name: "ProfilesSink",
			sink: new(ProfilesSink),
			consumeFunc: func(sink interface{}, ctx context.Context) error {
				return sink.(*ProfilesSink).ConsumeProfiles(ctx, testdata.GenerateProfiles(1))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a context with initial values
			initialCtx := context.WithValue(context.Background(), "initial-key", "initial-value")

			// Create a context chain to simulate transformation
			transformedCtx := context.WithValue(initialCtx, "transformed-key", "transformed-value")

			// Consume data with the transformed context
			err := tc.consumeFunc(tc.sink, transformedCtx)
			require.NoError(t, err)

			// Verify context storage and transformation
			storedContexts := tc.sink.AllContexts()
			require.Len(t, storedContexts, 1, "Should have stored exactly one context")

			storedCtx := storedContexts[0]
			// Verify both initial and transformed values are preserved
			assert.Equal(t, "initial-value", storedCtx.Value("initial-key"),
				"Initial context value should be preserved")
			assert.Equal(t, "transformed-value", storedCtx.Value("transformed-key"),
				"Transformed context value should be stored")
		})
	}
}

// TestContextTransformationChain verifies that the context is stored and transformed correctly in a chain of transformations
func TestContextTransformationChain(t *testing.T) {
	sink := new(TracesSink)

	// Create a context transformation chain
	baseCtx := context.Background()
	ctx1 := context.WithValue(baseCtx, "step1", "value1")
	ctx2 := context.WithValue(ctx1, "step2", "value2")
	ctx3 := context.WithValue(ctx2, "step3", "value3")

	// Consume traces with the transformed context
	td := testdata.GenerateTraces(1)
	err := sink.ConsumeTraces(ctx3, td)
	require.NoError(t, err)

	// Verify the complete transformation chain
	storedContexts := sink.AllContexts()
	require.Len(t, storedContexts, 1)

	finalCtx := storedContexts[0]
	// Verify each transformation step
	assert.Equal(t, "value1", finalCtx.Value("step1"), "First transformation should be preserved")
	assert.Equal(t, "value2", finalCtx.Value("step2"), "Second transformation should be preserved")
	assert.Equal(t, "value3", finalCtx.Value("step3"), "Third transformation should be preserved")
}

// TestConcurrentContextTransformations verifies context handling under concurrent operations
func TestConcurrentContextTransformations(t *testing.T) {
	sink := new(TracesSink)
	const numGoroutines = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()

			// Create a unique context for each goroutine
			ctx := context.WithValue(context.Background(),
				fmt.Sprintf("goroutine-%d", idx),
				fmt.Sprintf("value-%d", idx))

			td := testdata.GenerateTraces(1)
			err := sink.ConsumeTraces(ctx, td)
			require.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Verify all contexts were stored correctly
	storedContexts := sink.AllContexts()
	assert.Len(t, storedContexts, numGoroutines)

	// Create a map to verify all expected values are present
	contextValues := make(map[string]bool)
	for _, ctx := range storedContexts {
		for i := 0; i < numGoroutines; i++ {
			key := fmt.Sprintf("goroutine-%d", i)
			expectedValue := fmt.Sprintf("value-%d", i)
			if val := ctx.Value(key); val == expectedValue {
				contextValues[key] = true
			}
		}
	}

	// Verify all goroutines' contexts were preserved
	assert.Len(t, contextValues, numGoroutines,
		"Should have stored contexts from all goroutines")
}
