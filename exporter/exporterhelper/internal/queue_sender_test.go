// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

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

func TestQueueSender_ArcAcquireWaitMetric(t *testing.T) {
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
	// Disable batching and wait for results to test ARC correctly.
	qCfg.Batch = configoptional.Optional[queuebatch.BatchConfig]{} // This disables batching
	qCfg.WaitForResult = true

	// Create a sender that blocks until the channel is closed
	blocker := make(chan struct{})
	mockSender := sender.NewSender(func(context.Context, request.Request) error {
		<-blocker
		return nil
	})

	be, err := NewQueueSender(qSet, qCfg, "", mockSender)
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	var sendWg sync.WaitGroup
	sendWg.Add(2)

	// Send first request
	go func() {
		defer sendWg.Done()
		// This send will acquire the only permit and block in the sender
		assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 1}))
	}()

	// Wait for the first request to acquire the permit
	time.Sleep(20 * time.Millisecond)

	// Send second request
	go func() {
		defer sendWg.Done()
		// This send will block in arcCtl.Acquire()
		assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 1}))
	}()

	// Wait for the second request to block in Acquire
	time.Sleep(50 * time.Millisecond)

	// Unblock the sender, which will release the permit, unblocking the second request
	close(blocker)

	// Wait for both sends to complete
	sendWg.Wait()

	// Shutdown to ensure all telemetry is flushed
	require.NoError(t, be.Shutdown(context.Background()))

	// Verify the metric
	metrics, err := tt.GetMetric("otelcol_exporter_arc_acquire_wait_ms")
	require.NoError(t, err)
	points := metrics.Data.(metricdata.Histogram[int64]).DataPoints
	require.Len(t, points, 1)

	// Both requests should have recorded a wait time
	assert.Equal(t, uint64(2), points[0].Count)

	// The max wait time should be > 40ms (accounting for test scheduler variance)
	// The first request had minimal wait, the second had ~50ms wait.
	maxVal, ok := points[0].Max.Value()
	require.True(t, ok, "Max value should be valid")
	assert.Greater(t, maxVal, int64(40), "Max wait time should be significant")
	assert.Greater(t, points[0].Sum, int64(40), "Sum of wait times should be significant")
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
