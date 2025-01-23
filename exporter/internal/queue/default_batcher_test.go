// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/internal"
	"go.opentelemetry.io/collector/exporter/internal/requesttest"
	"go.opentelemetry.io/collector/pipeline"
)

func TestDefaultBatcher_NoSplit_MinThresholdZero_TimeoutDisabled(t *testing.T) {
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

			q := exporterqueue.NewMemoryQueueFactory[internal.Request]()(
				context.Background(),
				exporterqueue.Settings{
					Signal:           pipeline.SignalTraces,
					ExporterSettings: exportertest.NewNopSettings(),
				},
				exporterqueue.NewDefaultConfig())

			ba, err := NewBatcher(cfg, q,
				func(ctx context.Context, req internal.Request) error { return req.Export(ctx) },
				tt.maxWorkers)
			require.NoError(t, err)

			require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
			require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, q.Shutdown(context.Background()))
				require.NoError(t, ba.Shutdown(context.Background()))
			})

			sink := requesttest.NewSink()

			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 8, Sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 8, ExportErr: errors.New("transient error"), Sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 17, Sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 13, Sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 35, Sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 2, Sink: sink}))
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 5 && sink.ItemsCount() == 75
			}, 30*time.Millisecond, 10*time.Millisecond)
		})
	}
}

func TestDefaultBatcher_NoSplit_TimeoutDisabled(t *testing.T) {
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

			q := exporterqueue.NewMemoryQueueFactory[internal.Request]()(
				context.Background(),
				exporterqueue.Settings{
					Signal:           pipeline.SignalTraces,
					ExporterSettings: exportertest.NewNopSettings(),
				},
				exporterqueue.NewDefaultConfig())

			ba, err := NewBatcher(cfg, q,
				func(ctx context.Context, req internal.Request) error { return req.Export(ctx) },
				tt.maxWorkers)
			require.NoError(t, err)

			require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
			require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, q.Shutdown(context.Background()))
				require.NoError(t, ba.Shutdown(context.Background()))
			})

			sink := requesttest.NewSink()

			// These two requests will be dropped because of export error.
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 8, Sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 8, ExportErr: errors.New("transient error"), Sink: sink}))

			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 7, Sink: sink}))

			// This request will be dropped because of merge error
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 8, MergeErr: errors.New("transient error"), Sink: sink}))

			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 13, Sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 35, Sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 2, Sink: sink}))
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 2 && sink.ItemsCount() == 55
			}, 30*time.Millisecond, 10*time.Millisecond)
		})
	}
}

func TestDefaultBatcher_NoSplit_WithTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows, see https://github.com/open-telemetry/opentelemetry-collector/issues/11869")
	}

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

			q := exporterqueue.NewMemoryQueueFactory[internal.Request]()(
				context.Background(),
				exporterqueue.Settings{
					Signal:           pipeline.SignalTraces,
					ExporterSettings: exportertest.NewNopSettings(),
				},
				exporterqueue.NewDefaultConfig())

			ba, err := NewBatcher(cfg, q,
				func(ctx context.Context, req internal.Request) error { return req.Export(ctx) },
				tt.maxWorkers)
			require.NoError(t, err)

			require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
			require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, q.Shutdown(context.Background()))
				require.NoError(t, ba.Shutdown(context.Background()))
			})

			sink := requesttest.NewSink()

			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 8, Sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 17, Sink: sink}))

			// This request will be dropped because of merge error
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 8, MergeErr: errors.New("transient error"), Sink: sink}))

			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 13, Sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 35, Sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 2, Sink: sink}))
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 1 && sink.ItemsCount() == 75
			}, 100*time.Millisecond, 10*time.Millisecond)
		})
	}
}

func TestDefaultBatcher_Split_TimeoutDisabled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows, see https://github.com/open-telemetry/opentelemetry-collector/issues/11847")
	}

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
				MinSizeItems: 100,
			}
			cfg.MaxSizeConfig = exporterbatcher.MaxSizeConfig{
				MaxSizeItems: 100,
			}

			q := exporterqueue.NewMemoryQueueFactory[internal.Request]()(
				context.Background(),
				exporterqueue.Settings{
					Signal:           pipeline.SignalTraces,
					ExporterSettings: exportertest.NewNopSettings(),
				},
				exporterqueue.NewDefaultConfig())

			ba, err := NewBatcher(cfg, q,
				func(ctx context.Context, req internal.Request) error { return req.Export(ctx) },
				tt.maxWorkers)
			require.NoError(t, err)

			require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
			require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, q.Shutdown(context.Background()))
				require.NoError(t, ba.Shutdown(context.Background()))
			})

			sink := requesttest.NewSink()

			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 8, Sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 17, Sink: sink}))

			// This request will be dropped because of merge error
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 8, MergeErr: errors.New("transient error"), Sink: sink}))

			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 13, Sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 35, Sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 2, Sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 30, Sink: sink}))
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 2 && sink.ItemsCount() == 105
			}, 100*time.Millisecond, 10*time.Millisecond)

			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 900, Sink: sink}))
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 11 && sink.ItemsCount() == 1005
			}, 100*time.Millisecond, 10*time.Millisecond)
		})
	}
}

func TestDefaultBatcher_Shutdown(t *testing.T) {
	batchCfg := exporterbatcher.NewDefaultConfig()
	batchCfg.MinSizeItems = 10
	batchCfg.FlushTimeout = 100 * time.Second

	q := exporterqueue.NewMemoryQueueFactory[internal.Request]()(
		context.Background(),
		exporterqueue.Settings{
			Signal:           pipeline.SignalTraces,
			ExporterSettings: exportertest.NewNopSettings(),
		},
		exporterqueue.NewDefaultConfig())

	ba, err := NewBatcher(batchCfg, q,
		func(ctx context.Context, req internal.Request) error { return req.Export(ctx) },
		2)
	require.NoError(t, err)

	require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))

	sink := requesttest.NewSink()

	require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 1, Sink: sink}))
	require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 2, Sink: sink}))

	// Give the batcher some time to read from queue
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int64(0), sink.RequestsCount())
	assert.Equal(t, int64(0), sink.ItemsCount())

	require.NoError(t, q.Shutdown(context.Background()))
	require.NoError(t, ba.Shutdown(context.Background()))

	assert.Equal(t, int64(1), sink.RequestsCount())
	assert.Equal(t, int64(3), sink.ItemsCount())
}
