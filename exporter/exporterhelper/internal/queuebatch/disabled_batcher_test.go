// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
)

func TestDisabledBatcher(t *testing.T) {
	tests := []struct {
		name       string
		maxWorkers int
	}{
		{
			name:       "one_worker",
			maxWorkers: 1,
		},
		{
			name:       "three_workers",
			maxWorkers: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := requesttest.NewSink()
			ba := newDisabledBatcher(sink.Export)

			q, err := queue.NewQueue(queue.Settings[request.Request]{
				Capacity:        1000,
				BlockOnOverflow: true,
				NumConsumers:    tt.maxWorkers,
				Telemetry:       componenttest.NewNopTelemetrySettings(),
			}, ba.Consume)
			require.NoError(t, err)

			require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
			require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, q.Shutdown(context.Background()))
				require.NoError(t, ba.Shutdown(context.Background()))
			})

			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 8}))
			sink.SetExportErr(errors.New("transient error"))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 8}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 17}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 13}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 35}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 2}))
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 5 && sink.ItemsCount() == 75
			}, 1*time.Second, 10*time.Millisecond)
		})
	}
}

// TestDisabledBatcher_ContextCancelled verifies that the disabled batcher
// checks for context cancellation before processing and returns early.
// This prevents exporting data when the caller already received a timeout error,
// which would cause confusing observability results and potential duplicate data
// if the caller retries.
// This is a regression test for the context check fix in PR #14473.
func TestDisabledBatcher_ContextCancelled(t *testing.T) {
	exported := atomic.Int64{}

	exportFunc := func(_ context.Context, req request.Request) error {
		exported.Add(int64(req.(*requesttest.FakeRequest).Items))
		return nil
	}

	ba := newDisabledBatcher(exportFunc)
	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, ba.Shutdown(context.Background()))
	})

	// Create already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := newFakeDone()
	ba.Consume(ctx, &requesttest.FakeRequest{Items: 10}, done)

	// Should have error from context cancellation
	assert.Eventually(t, func() bool {
		return done.errors.Load() == 1
	}, time.Second, 10*time.Millisecond)

	// Data should NOT have been exported
	assert.EqualValues(t, 0, exported.Load(),
		"Request with cancelled context should not be exported")
	assert.EqualValues(t, 0, done.success.Load(),
		"Should not have success callback for cancelled context")
}

// TestDisabledBatcher_ContextValid verifies normal operation when context is valid.
func TestDisabledBatcher_ContextValid(t *testing.T) {
	exported := atomic.Int64{}

	exportFunc := func(_ context.Context, req request.Request) error {
		exported.Add(int64(req.(*requesttest.FakeRequest).Items))
		return nil
	}

	ba := newDisabledBatcher(exportFunc)
	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, ba.Shutdown(context.Background()))
	})

	done := newFakeDone()
	ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 10}, done)

	// Should have success
	assert.Eventually(t, func() bool {
		return done.success.Load() == 1
	}, time.Second, 10*time.Millisecond)

	// Data should have been exported
	assert.EqualValues(t, 10, exported.Load(),
		"Request with valid context should be exported")
	assert.EqualValues(t, 0, done.errors.Load(),
		"Should not have error callback for valid context")
}
