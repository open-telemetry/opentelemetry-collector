// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pipeline"
)

func TestNewQueueSenderFailedRequestDropped(t *testing.T) {
	qSet := queuebatch.AllSettings[request.Request]{
		Signal:    pipeline.SignalMetrics,
		ID:        component.NewID(exportertest.NopType),
		Telemetry: componenttest.NewNopTelemetrySettings(),
	}
	logger, observed := observer.New(zap.ErrorLevel)
	qSet.Telemetry.Logger = zap.New(logger)
	be, err := NewQueueSender(
		qSet, NewDefaultQueueConfig(), "", sender.NewSender(func(context.Context, request.Request) error { return errors.New("some error") }))
	require.NoError(t, err)

	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 2}))
	require.NoError(t, be.Shutdown(context.Background()))
	assert.Len(t, observed.All(), 1)
	assert.Equal(t, "Exporting failed. Dropping data.", observed.All()[0].Message)
}

func TestQueueSender_ArcAcquireWaitMetric_Failure(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	qSet := queuebatch.AllSettings[request.Request]{
		Signal:    pipeline.SignalMetrics,
		ID:        component.NewID(exportertest.NopType),
		Telemetry: tt.NewTelemetrySettings(),
	}
	qCfg := NewDefaultQueueConfig()
	qCfg.Arc.Enabled = true
	qCfg.Arc.InitialLimit = 1
	qCfg.Batch = configoptional.Optional[queuebatch.BatchConfig]{} // This disables batching
	qCfg.WaitForResult = true

	// Create a sender that blocks
	blocker := make(chan struct{})
	mockSender := sender.NewSender(func(context.Context, request.Request) error {
		<-blocker // Block the first request
		return nil
	})

	be, err := NewQueueSender(qSet, qCfg, "", mockSender)
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	// Send first request (will block in sender)
	var sendWg sync.WaitGroup
	sendWg.Add(1)
	go func() {
		defer sendWg.Done()
		assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 1}))
	}()

	// Wait for the first request to acquire the permit
	time.Sleep(50 * time.Millisecond)

	// Send second request with a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	sendWg.Add(1)
	go func() {
		defer sendWg.Done()
		// This send will block in arcCtl.Acquire() and then fail
		sendErr := be.Send(ctx, &requesttest.FakeRequest{Items: 1})
		assert.Error(t, sendErr)
		// The error should be context.Canceled, as the context passed to Send is canceled.
		assert.ErrorIs(t, sendErr, context.Canceled)
	}()

	// Wait for the second request to block in Acquire
	time.Sleep(50 * time.Millisecond)

	// Cancel the context
	cancel()

	// Unblock the first sender
	close(blocker)

	// Wait for both sends to complete
	sendWg.Wait()

	// Shutdown to ensure all telemetry is flushed
	require.NoError(t, be.Shutdown(context.Background()))

	// 4. Verify the acquire wait metric
	metrics, err := tt.GetMetric("otelcol_exporter_arc_acquire_wait_ms")
	require.NoError(t, err)
	points := metrics.Data.(metricdata.Histogram[int64]).DataPoints
	require.Len(t, points, 1)

	// Both requests should have recorded a wait time (1 success, 1 failure)
	assert.Equal(t, uint64(2), points[0].Count)
	maxVal, ok := points[0].Max.Value()
	require.True(t, ok, "Max value should be valid")
	assert.Greater(t, maxVal, int64(40), "Max wait time (from failed acquire) should be significant")

	// Verify the arc failures metric
	metrics, err = tt.GetMetric("otelcol_exporter_arc_failures")
	require.NoError(t, err)
	pointsF := metrics.Data.(metricdata.Sum[int64]).DataPoints
	require.Len(t, pointsF, 1)
	assert.Equal(t, int64(1), pointsF[0].Value)
}

func TestQueueConfig_Validate(t *testing.T) {
	qCfg := NewDefaultQueueConfig()
	require.NoError(t, qCfg.Validate())

	qCfg.NumConsumers = 0
	require.EqualError(t, qCfg.Validate(), "`num_consumers` must be positive")

	qCfg = NewDefaultQueueConfig()
	qCfg.QueueSize = 0
	require.EqualError(t, qCfg.Validate(), "`queue_size` must be positive")

	// Confirm Validate doesn't return error with invalid config when feature is disabled
	qCfg.Enabled = false
	assert.NoError(t, qCfg.Validate())
}

func TestArc_BackpressureTriggersShrinkAndMetrics(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	qSet := queuebatch.AllSettings[request.Request]{
		Signal:    pipeline.SignalTraces,
		ID:        component.NewID(exportertest.NopType),
		Telemetry: tt.NewTelemetrySettings(),
	}

	qCfg := NewDefaultQueueConfig()
	qCfg.WaitForResult = true
	qCfg.Batch = configoptional.Optional[queuebatch.BatchConfig]{} // disable batching
	qCfg.Arc.Enabled = true
	qCfg.Arc.InitialLimit = 3
	qCfg.Arc.MaxConcurrency = 10

	// First call: explicit backpressure (gRPC ResourceExhausted). Second call: success.
	var call int32
	be, err := NewQueueSender(
		qSet,
		qCfg,
		qSet.ID.String(),
		sender.NewSender(func(context.Context, request.Request) error {
			c := atomic.AddInt32(&call, 1)
			if c == 1 {
				// 1. Simulates a backpressure error
				return status.Error(codes.ResourceExhausted, "overload")
			}
			return nil
		}),
	)
	require.NoError(t, err)

	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	// First send (backpressure)
	require.Error(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 1}))
	// Ensure control step boundary is crossed (~300ms default period)
	time.Sleep(350 * time.Millisecond)
	// Second send (success) triggers control step and metrics flush
	require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 1}))
	require.NoError(t, be.Shutdown(context.Background()))

	// 2. Asserts otelcol_exporter_arc_backoff_events increments
	m, err := tt.GetMetric("otelcol_exporter_arc_backoff_events")
	require.NoError(t, err)
	backoffPoints := m.Data.(metricdata.Sum[int64]).DataPoints
	var backoffSum int64
	for _, dp := range backoffPoints {
		backoffSum += dp.Value
	}
	assert.GreaterOrEqual(t, backoffSum, int64(1), "expected at least one backoff event")

	// 3. Asserts otelcol_exporter_arc_limit_changes{direction="down"} increments
	m, err = tt.GetMetric("otelcol_exporter_arc_limit_changes")
	require.NoError(t, err)
	lcPoints := m.Data.(metricdata.Sum[int64]).DataPoints
	var down int64
	for _, dp := range lcPoints {
		if v, ok := dp.Attributes.Value("direction"); ok && v.AsString() == "down" {
			down += dp.Value
		}
	}
	assert.GreaterOrEqual(t, down, int64(1), "expected at least one limit-down change")
}
