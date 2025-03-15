// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batcher

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
)

func TestDefaultBatcher_NoSplit_MinThresholdZero_TimeoutDisabled(t *testing.T) {
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
			cfg := exporterbatcher.NewDefaultConfig()
			cfg.Enabled = true
			cfg.FlushTimeout = 0
			cfg.SizeConfig = exporterbatcher.SizeConfig{
				Sizer:   exporterbatcher.SizerTypeItems,
				MinSize: 0,
			}

			sink := requesttest.NewSink()
			ba, err := NewBatcher(cfg, sink.Export, tt.maxWorkers)
			require.NoError(t, err)
			require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, ba.Shutdown(context.Background()))
			})

			done := newFakeDone()
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 8}, done)
			sink.SetExportErr(errors.New("transient error"))
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 8}, done)
			<-time.After(10 * time.Millisecond)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 17}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 13}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 35}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 2}, done)
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 5 && sink.ItemsCount() == 75
			}, 1*time.Second, 10*time.Millisecond)
			// Check that done callback is called for the right amount of times.
			assert.EqualValues(t, 1, done.errors.Load())
			assert.EqualValues(t, 5, done.success.Load())
		})
	}
}

func TestDefaultBatcher_NoSplit_TimeoutDisabled(t *testing.T) {
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
			cfg := exporterbatcher.NewDefaultConfig()
			cfg.Enabled = true
			cfg.FlushTimeout = 0
			cfg.SizeConfig = exporterbatcher.SizeConfig{
				Sizer:   exporterbatcher.SizerTypeItems,
				MinSize: 10,
			}

			sink := requesttest.NewSink()
			ba, err := NewBatcher(cfg, sink.Export, tt.maxWorkers)
			require.NoError(t, err)
			require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))

			done := newFakeDone()
			// These two requests will be dropped because of export error.
			sink.SetExportErr(errors.New("transient error"))
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 8}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 8}, done)
			<-time.After(10 * time.Millisecond)

			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 7}, done)
			// This requests will be dropped because of merge error.
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 8, MergeErr: errors.New("transient error")}, done)

			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 13}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 35}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 2}, done)

			// Only the requests with 7+13 and 35 will be flushed.
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 2 && sink.ItemsCount() == 55
			}, 1*time.Second, 10*time.Millisecond)

			require.NoError(t, ba.Shutdown(context.Background()))

			// After shutdown the pending "current batch" is also flushed.
			assert.EqualValues(t, 3, sink.RequestsCount())
			assert.EqualValues(t, 57, sink.ItemsCount())

			// Check that done callback is called for the right amount of times.
			assert.EqualValues(t, 3, done.errors.Load())
			assert.EqualValues(t, 4, done.success.Load())
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
			cfg.SizeConfig = exporterbatcher.SizeConfig{
				Sizer:   exporterbatcher.SizerTypeItems,
				MinSize: 100,
			}

			sink := requesttest.NewSink()
			ba, err := NewBatcher(cfg, sink.Export, tt.maxWorkers)
			require.NoError(t, err)
			require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, ba.Shutdown(context.Background()))
			})

			done := newFakeDone()
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 8}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 17}, done)
			// This requests will be dropped because of merge error.
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 8, MergeErr: errors.New("transient error")}, done)

			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 13}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 35}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 2}, done)
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 1 && sink.ItemsCount() == 75
			}, 1*time.Second, 10*time.Millisecond)

			// Check that done callback is called for the right amount of times.
			assert.EqualValues(t, 1, done.errors.Load())
			assert.EqualValues(t, 5, done.success.Load())
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
			cfg.SizeConfig = exporterbatcher.SizeConfig{
				Sizer:   exporterbatcher.SizerTypeItems,
				MinSize: 100,
				MaxSize: 100,
			}

			sink := requesttest.NewSink()
			ba, err := NewBatcher(cfg, sink.Export, tt.maxWorkers)
			require.NoError(t, err)
			require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))

			done := newFakeDone()
			// This requests will be dropped because of merge error.
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 8, MergeErr: errors.New("transient error")}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 8}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 17}, done)
			// This requests will be dropped because of merge error.
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 8, MergeErr: errors.New("transient error")}, done)

			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 13}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 35}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 2}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 30}, done)
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 1 && sink.ItemsCount() == 100
			}, 1*time.Second, 10*time.Millisecond)

			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 900}, done)
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 10 && sink.ItemsCount() == 1000
			}, 1*time.Second, 10*time.Millisecond)

			// At this point the 7th not failing request is still pending.
			assert.EqualValues(t, 6, done.success.Load())

			require.NoError(t, ba.Shutdown(context.Background()))

			// After shutdown the pending "current batch" is also flushed.
			assert.EqualValues(t, 11, sink.RequestsCount())
			assert.EqualValues(t, 1005, sink.ItemsCount())

			// Check that done callback is called for the right amount of times.
			assert.EqualValues(t, 2, done.errors.Load())
			assert.EqualValues(t, 7, done.success.Load())
		})
	}
}

func TestDefaultBatcher_Shutdown(t *testing.T) {
	batchCfg := exporterbatcher.NewDefaultConfig()
	batchCfg.MinSize = 10
	batchCfg.FlushTimeout = 100 * time.Second

	sink := requesttest.NewSink()
	ba, err := NewBatcher(batchCfg, sink.Export, 2)
	require.NoError(t, err)
	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))

	done := newFakeDone()
	ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 1}, done)
	ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 2}, done)

	assert.EqualValues(t, 0, sink.RequestsCount())
	assert.EqualValues(t, 0, sink.ItemsCount())

	require.NoError(t, ba.Shutdown(context.Background()))

	assert.EqualValues(t, 1, sink.RequestsCount())
	assert.EqualValues(t, 3, sink.ItemsCount())

	// Check that done callback is called for the right amount of times.
	assert.EqualValues(t, 0, done.errors.Load())
	assert.EqualValues(t, 2, done.success.Load())
}

func TestDefaultBatcher_MergeError(t *testing.T) {
	batchCfg := exporterbatcher.NewDefaultConfig()
	batchCfg.MinSize = 5
	batchCfg.MaxSize = 7

	sink := requesttest.NewSink()
	ba, err := NewBatcher(batchCfg, sink.Export, 2)
	require.NoError(t, err)

	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, ba.Shutdown(context.Background()))
	})

	done := newFakeDone()
	ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 9}, done)
	assert.Eventually(t, func() bool {
		return sink.RequestsCount() == 1 && sink.ItemsCount() == 7
	}, 1*time.Second, 10*time.Millisecond)

	sink.SetExportErr(errors.New("transient error"))
	ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 4}, done)
	assert.Eventually(t, func() bool {
		return 2 == done.errors.Load()
	}, 1*time.Second, 10*time.Millisecond)

	// Check that done callback is called for the right amount of times.
	assert.EqualValues(t, 2, done.errors.Load())
	assert.EqualValues(t, 0, done.success.Load())
}

type fakeDone struct {
	errors  *atomic.Int64
	success *atomic.Int64
}

func newFakeDone() fakeDone {
	return fakeDone{
		errors:  &atomic.Int64{},
		success: &atomic.Int64{},
	}
}

func (fd fakeDone) OnDone(err error) {
	if err != nil {
		fd.errors.Add(1)
	} else {
		fd.success.Add(1)
	}
}
