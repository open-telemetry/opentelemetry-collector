// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/internal"
)

func TestDefaultBatcher_MinThresholdZero_TimeoutDisabled(t *testing.T) {
	tests := []struct {
		name       string
		maxWorkers int
	}{
		{
			name:       "infinate_workers",
			maxWorkers: 0,
		},
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
			cfg := exporterbatcher.NewDefaultConfig()
			cfg.Enabled = true
			cfg.FlushTimeout = 0
			cfg.MinSizeConfig = exporterbatcher.MinSizeConfig{
				MinSizeItems: 0,
			}

			q := NewBoundedMemoryQueue[internal.Request](
				MemoryQueueSettings[internal.Request]{
					Sizer:    &RequestSizer[internal.Request]{},
					Capacity: 10,
				})

			ba, err := NewBatcher(cfg, q, tt.maxWorkers)
			require.NoError(t, err)

			require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
			require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, q.Shutdown(context.Background()))
				require.NoError(t, ba.Shutdown(context.Background()))
			})

			sink := newFakeRequestSink()

			require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 8, sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 8, exportErr: errors.New("transient error"), sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 17, sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 13, sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 35, sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 2, sink: sink}))
			assert.Eventually(t, func() bool {
				return sink.requestsCount.Load() == 5 && sink.itemsCount.Load() == 75
			}, 30*time.Millisecond, 10*time.Millisecond)
		})
	}
}

func TestDefaultBatcher_TimeoutDisabled(t *testing.T) {
	tests := []struct {
		name       string
		maxWorkers int
	}{
		{
			name:       "infinate_workers",
			maxWorkers: 0,
		},
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
			cfg := exporterbatcher.NewDefaultConfig()
			cfg.Enabled = true
			cfg.FlushTimeout = 0
			cfg.MinSizeConfig = exporterbatcher.MinSizeConfig{
				MinSizeItems: 10,
			}

			q := NewBoundedMemoryQueue[internal.Request](
				MemoryQueueSettings[internal.Request]{
					Sizer:    &RequestSizer[internal.Request]{},
					Capacity: 10,
				})

			ba, err := NewBatcher(cfg, q, tt.maxWorkers)
			require.NoError(t, err)

			require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
			require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, q.Shutdown(context.Background()))
				require.NoError(t, ba.Shutdown(context.Background()))
			})

			sink := newFakeRequestSink()

			// These two requests will be dropped because of export error.
			require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 8, sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 8, exportErr: errors.New("transient error"), sink: sink}))

			require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 7, sink: sink}))

			// This request will be dropped because of merge error
			require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 8, mergeErr: errors.New("transient error"), sink: sink}))

			require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 13, sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 35, sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 2, sink: sink}))
			assert.Eventually(t, func() bool {
				return sink.requestsCount.Load() == 2 && sink.itemsCount.Load() == 55
			}, 30*time.Millisecond, 10*time.Millisecond)
		})
	}
}

func TestDefaultBatcher_WithTimeout(t *testing.T) {
	tests := []struct {
		name       string
		maxWorkers int
	}{
		{
			name:       "infinate_workers",
			maxWorkers: 0,
		},
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
			cfg := exporterbatcher.NewDefaultConfig()
			cfg.Enabled = true
			cfg.FlushTimeout = 50 * time.Millisecond
			cfg.MinSizeConfig = exporterbatcher.MinSizeConfig{
				MinSizeItems: 100,
			}

			q := NewBoundedMemoryQueue[internal.Request](
				MemoryQueueSettings[internal.Request]{
					Sizer:    &RequestSizer[internal.Request]{},
					Capacity: 10,
				})

			ba, err := NewBatcher(cfg, q, tt.maxWorkers)
			require.NoError(t, err)

			require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
			require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, q.Shutdown(context.Background()))
				require.NoError(t, ba.Shutdown(context.Background()))
			})

			sink := newFakeRequestSink()

			require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 8, sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 17, sink: sink}))

			// This request will be dropped because of merge error
			require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 8, mergeErr: errors.New("transient error"), sink: sink}))

			require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 13, sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 35, sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 2, sink: sink}))
			assert.Eventually(t, func() bool {
				return sink.requestsCount.Load() == 1 && sink.itemsCount.Load() == 75
			}, 100*time.Millisecond, 10*time.Millisecond)
		})
	}
}

func TestDisabledBatcher_SplitNotImplemented(t *testing.T) {
	cfg := exporterbatcher.NewDefaultConfig()
	cfg.Enabled = true
	maxWorkers := 0
	cfg.MaxSizeConfig.MaxSizeItems = 1

	q := NewBoundedMemoryQueue[internal.Request](
		MemoryQueueSettings[internal.Request]{
			Sizer:    &RequestSizer[internal.Request]{},
			Capacity: 10,
		})

	_, err := NewBatcher(cfg, q, maxWorkers)
	require.Error(t, err)
}
