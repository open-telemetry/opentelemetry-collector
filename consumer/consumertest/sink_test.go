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

type (
	ctxKey  string
	testKey int
)

func TestTracesSink(t *testing.T) {
	sink := new(TracesSink)
	td := testdata.GenerateTraces(1)
	want := make([]ptrace.Traces, 0, 7)
	for range 7 {
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
	for range 7 {
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
	for range 7 {
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
	for range 7 {
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

	for i := range 7 {
		ctx := context.WithValue(context.Background(), testKey(i), fmt.Sprintf("value-%d", i))
		require.NoError(t, sink.ConsumeTraces(ctx, td))
		want = append(want, td)
		wantCtx = append(wantCtx, ctx)
	}

	assert.Equal(t, want, sink.AllTraces())
	assert.Equal(t, len(want), sink.SpanCount())

	// Verify contexts
	gotCtx := sink.Contexts()
	assert.Len(t, gotCtx, len(wantCtx))
	for i, ctx := range gotCtx {
		assert.Equal(t, fmt.Sprintf("value-%d", i), ctx.Value(testKey(i)))
	}

	sink.Reset()
	assert.Empty(t, sink.AllTraces())
	assert.Empty(t, sink.Contexts())
	assert.Equal(t, 0, sink.SpanCount())
}

func TestMetricsSinkWithContext(t *testing.T) {
	sink := new(MetricsSink)
	md := testdata.GenerateMetrics(1)
	want := make([]pmetric.Metrics, 0, 7)
	wantCtx := make([]context.Context, 0, 7)

	for i := range 7 {
		ctx := context.WithValue(context.Background(), testKey(i), fmt.Sprintf("value-%d", i))
		require.NoError(t, sink.ConsumeMetrics(ctx, md))
		want = append(want, md)
		wantCtx = append(wantCtx, ctx)
	}

	assert.Equal(t, want, sink.AllMetrics())
	assert.Equal(t, 2*len(want), sink.DataPointCount())

	// Verify contexts
	gotCtx := sink.Contexts()
	assert.Len(t, gotCtx, len(wantCtx))
	for i, ctx := range gotCtx {
		assert.Equal(t, fmt.Sprintf("value-%d", i), ctx.Value(testKey(i)))
	}

	sink.Reset()
	assert.Empty(t, sink.AllMetrics())
	assert.Empty(t, sink.Contexts())
	assert.Equal(t, 0, sink.DataPointCount())
}

func TestLogsSinkWithContext(t *testing.T) {
	sink := new(LogsSink)
	md := testdata.GenerateLogs(1)
	want := make([]plog.Logs, 0, 7)
	wantCtx := make([]context.Context, 0, 7)

	for i := range 7 {
		ctx := context.WithValue(context.Background(), testKey(i), fmt.Sprintf("value-%d", i))
		require.NoError(t, sink.ConsumeLogs(ctx, md))
		want = append(want, md)
		wantCtx = append(wantCtx, ctx)
	}

	assert.Equal(t, want, sink.AllLogs())
	assert.Equal(t, len(want), sink.LogRecordCount())

	// Verify contexts
	gotCtx := sink.Contexts()
	assert.Len(t, gotCtx, len(wantCtx))
	for i, ctx := range gotCtx {
		assert.Equal(t, fmt.Sprintf("value-%d", i), ctx.Value(testKey(i)))
	}

	sink.Reset()
	assert.Empty(t, sink.AllLogs())
	assert.Empty(t, sink.Contexts())
	assert.Equal(t, 0, sink.LogRecordCount())
}

func TestProfilesSinkWithContext(t *testing.T) {
	sink := new(ProfilesSink)
	td := testdata.GenerateProfiles(1)
	want := make([]pprofile.Profiles, 0, 7)
	wantCtx := make([]context.Context, 0, 7)

	for i := range 7 {
		ctx := context.WithValue(context.Background(), testKey(i), fmt.Sprintf("value-%d", i))
		require.NoError(t, sink.ConsumeProfiles(ctx, td))
		want = append(want, td)
		wantCtx = append(wantCtx, ctx)
	}

	assert.Equal(t, want, sink.AllProfiles())
	assert.Equal(t, len(want), sink.SampleCount())

	// Verify contexts
	gotCtx := sink.Contexts()
	assert.Len(t, gotCtx, len(wantCtx))
	for i, ctx := range gotCtx {
		assert.Equal(t, fmt.Sprintf("value-%d", i), ctx.Value(testKey(i)))
	}

	sink.Reset()
	assert.Empty(t, sink.AllProfiles())
	assert.Empty(t, sink.Contexts())
	assert.Equal(t, 0, sink.SampleCount())
}

// TestSinkContextTransformation verifies that the context is stored and transformed correctly
func TestSinkContextTransformation(t *testing.T) {
	testCases := []struct {
		name string
		sink interface {
			Contexts() []context.Context
		}
		consumeFunc func(any, context.Context) error
		testData    any
	}{
		{
			name: "TracesSink",
			sink: new(TracesSink),
			consumeFunc: func(sink any, ctx context.Context) error {
				return sink.(*TracesSink).ConsumeTraces(ctx, testdata.GenerateTraces(1))
			},
		},
		{
			name: "MetricsSink",
			sink: new(MetricsSink),
			consumeFunc: func(sink any, ctx context.Context) error {
				return sink.(*MetricsSink).ConsumeMetrics(ctx, testdata.GenerateMetrics(1))
			},
		},
		{
			name: "LogsSink",
			sink: new(LogsSink),
			consumeFunc: func(sink any, ctx context.Context) error {
				return sink.(*LogsSink).ConsumeLogs(ctx, testdata.GenerateLogs(1))
			},
		},
		{
			name: "ProfilesSink",
			sink: new(ProfilesSink),
			consumeFunc: func(sink any, ctx context.Context) error {
				return sink.(*ProfilesSink).ConsumeProfiles(ctx, testdata.GenerateProfiles(1))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a context with initial values
			initialCtx := context.WithValue(context.Background(), ctxKey("initial-key"), "initial-value")

			// Create a context chain to simulate transformation
			transformedCtx := context.WithValue(initialCtx, ctxKey("transformed-key"), "transformed-value")

			// Consume data with the transformed context
			err := tc.consumeFunc(tc.sink, transformedCtx)
			require.NoError(t, err)

			// Verify context storage and transformation
			storedContexts := tc.sink.Contexts()
			assert.Len(t, storedContexts, 1, "Should have stored exactly one context")

			storedCtx := storedContexts[0]
			// Verify both initial and transformed values are preserved
			assert.Equal(t, "initial-value", storedCtx.Value(ctxKey("initial-key")),
				"Initial context value should be preserved")
			assert.Equal(t, "transformed-value", storedCtx.Value(ctxKey("transformed-key")),
				"Transformed context value should be stored")
		})
	}
}

// TestContextTransformationChain verifies that the context is stored and transformed correctly in a chain of transformations
func TestContextTransformationChain(t *testing.T) {
	sink := new(TracesSink)

	// Create a context transformation chain
	baseCtx := context.Background()
	ctx1 := context.WithValue(baseCtx, ctxKey("step1"), "value1")
	ctx2 := context.WithValue(ctx1, ctxKey("step2"), "value2")
	ctx3 := context.WithValue(ctx2, ctxKey("step3"), "value3")

	// Consume traces with the transformed context
	td := testdata.GenerateTraces(1)
	err := sink.ConsumeTraces(ctx3, td)
	require.NoError(t, err)

	// Verify the complete transformation chain
	storedContexts := sink.Contexts()
	require.Len(t, storedContexts, 1)

	finalCtx := storedContexts[0]
	// Verify each transformation step
	assert.Equal(t, "value1", finalCtx.Value(ctxKey("step1")), "First transformation should be preserved")
	assert.Equal(t, "value2", finalCtx.Value(ctxKey("step2")), "Second transformation should be preserved")
	assert.Equal(t, "value3", finalCtx.Value(ctxKey("step3")), "Third transformation should be preserved")
}

// TestConcurrentContextTransformations verifies context handling under concurrent operations
func TestConcurrentContextTransformations(t *testing.T) {
	sink := new(TracesSink)
	const numGoroutines = 10
	errChan := make(chan error, numGoroutines)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(idx int) {
			defer wg.Done()
			key := ctxKey(fmt.Sprintf("goroutine-%d", idx))
			value := fmt.Sprintf("value-%d", idx)
			ctx := context.WithValue(context.Background(), key, value)

			td := testdata.GenerateTraces(1)
			if err := sink.ConsumeTraces(ctx, td); err != nil {
				errChan <- err
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Check for any errors that occurred in goroutines
	for err := range errChan {
		t.Errorf("Error in goroutine: %v", err)
	}

	// Verify all contexts were stored correctly
	storedContexts := sink.Contexts()
	assert.Len(t, storedContexts, numGoroutines)

	// Create a map to verify all expected values are present
	contextValues := make(map[string]bool)
	for _, ctx := range storedContexts {
		for i := range numGoroutines {
			key := ctxKey(fmt.Sprintf("goroutine-%d", i))
			expectedValue := fmt.Sprintf("value-%d", i)
			if val := ctx.Value(key); val == expectedValue {
				contextValues[fmt.Sprintf("goroutine-%d", i)] = true
			}
		}
	}

	// Verify all goroutines' contexts were preserved
	assert.Len(t, contextValues, numGoroutines,
		"Should have stored contexts from all goroutines")
}
