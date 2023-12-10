// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
)

func TestBatchSender_MergeBatcherConfig_Validate(t *testing.T) {
	cfg := NewDefaultMergeBatcherConfig()
	assert.NoError(t, cfg.Validate())

	cfg.MinSizeItems = -1
	assert.EqualError(t, cfg.Validate(), "min_size_items must be greater than or equal to zero")

	cfg = NewDefaultMergeBatcherConfig()
	cfg.Timeout = 0
	assert.EqualError(t, cfg.Validate(), "timeout must be greater than zero")
}

func TestBatchSender_SplitBatcherConfig_Validate(t *testing.T) {
	cfg := SplitBatcherConfig{}
	assert.NoError(t, cfg.Validate())

	cfg.MaxSizeItems = -1
	assert.EqualError(t, cfg.Validate(), "max_size_items must be greater than or equal to zero")
}

func TestBatchSender_MergeSplitBatcherConfig_Validate(t *testing.T) {
	cfg := MergeSplitBatcherConfig{
		MergeBatcherConfig: NewDefaultMergeBatcherConfig(),
		SplitBatcherConfig: SplitBatcherConfig{
			MaxSizeItems: 20000,
		},
	}
	assert.NoError(t, cfg.Validate())

	cfg.MinSizeItems = 20001
	assert.EqualError(t, cfg.Validate(), "max_size_items must be greater than or equal to min_size_items")
}

func TestBatchSender_Merge(t *testing.T) {
	mergeCfg := NewDefaultMergeBatcherConfig()
	mergeCfg.MinSizeItems = 10
	mergeCfg.Timeout = 100 * time.Millisecond

	tests := []struct {
		name          string
		batcherOption Option
	}{
		{
			name:          "merge_only",
			batcherOption: WithBatcher(mergeCfg, fakeBatchMergeFunc),
		},
		{
			name: "split_disabled",
			batcherOption: WithBatcher(mergeCfg, fakeBatchMergeFunc,
				WithSplitBatcher(SplitBatcherConfig{MaxSizeItems: 0}, fakeBatchMergeSplitFunc)),
		},
		{
			name: "split_high_limit",
			batcherOption: WithBatcher(mergeCfg, fakeBatchMergeFunc,
				WithSplitBatcher(SplitBatcherConfig{MaxSizeItems: 1000}, fakeBatchMergeSplitFunc)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			be := queueBatchExporter(t, tt.batcherOption)

			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, be.Shutdown(context.Background()))
			})

			sink := newFakeRequestSink()

			require.NoError(t, be.send(context.Background(), &fakeRequest{items: 8, sink: sink}))
			require.NoError(t, be.send(context.Background(), &fakeRequest{items: 3, sink: sink}))

			// the first two requests should be merged into one and sent by reaching the minimum items size
			assert.Eventually(t, func() bool {
				return sink.requestsCount.Load() == 1 && sink.itemsCount.Load() == 11
			}, 50*time.Millisecond, 10*time.Millisecond)

			require.NoError(t, be.send(context.Background(), &fakeRequest{items: 3, sink: sink}))
			require.NoError(t, be.send(context.Background(), &fakeRequest{items: 1, sink: sink}))

			// the third and fifth requests should be sent by reaching the timeout
			// the fourth request should be ignored because of the merge error.
			time.Sleep(50 * time.Millisecond)

			// should be ignored because of the merge error.
			require.NoError(t, be.send(context.Background(), &fakeRequest{items: 3, sink: sink,
				mergeErr: errors.New("merge error")}))

			assert.Equal(t, uint64(1), sink.requestsCount.Load())
			assert.Eventually(t, func() bool {
				return sink.requestsCount.Load() == 2 && sink.itemsCount.Load() == 15
			}, 100*time.Millisecond, 10*time.Millisecond)
		})
	}
}

func TestBatchSender_BatchExportError(t *testing.T) {
	mergeCfg := NewDefaultMergeBatcherConfig()
	mergeCfg.MinSizeItems = 10
	tests := []struct {
		name             string
		batcherOption    Option
		expectedRequests uint64
		expectedItems    uint64
	}{
		{
			name:          "merge_only",
			batcherOption: WithBatcher(mergeCfg, fakeBatchMergeFunc),
		},
		{
			name: "merge_with_split_triggered",
			batcherOption: WithBatcher(mergeCfg, fakeBatchMergeFunc,
				WithSplitBatcher(SplitBatcherConfig{MaxSizeItems: 200}, fakeBatchMergeSplitFunc)),
		},
		{
			name: "merge_with_split_triggered",
			batcherOption: WithBatcher(mergeCfg, fakeBatchMergeFunc,
				WithSplitBatcher(SplitBatcherConfig{MaxSizeItems: 20}, fakeBatchMergeSplitFunc)),
			expectedRequests: 1,
			expectedItems:    20,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			be := queueBatchExporter(t, tt.batcherOption)

			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, be.Shutdown(context.Background()))
			})

			sink := newFakeRequestSink()

			require.NoError(t, be.send(context.Background(), &fakeRequest{items: 4, sink: sink}))
			require.NoError(t, be.send(context.Background(), &fakeRequest{items: 4, sink: sink}))

			// the first two requests should be blocked by the batchSender.
			time.Sleep(50 * time.Millisecond)
			assert.Equal(t, uint64(0), sink.requestsCount.Load())

			// the third request should trigger the export and cause an error.
			errReq := &fakeRequest{items: 20, exportErr: errors.New("transient error"), sink: sink}
			require.NoError(t, be.send(context.Background(), errReq))

			// the batch should be dropped since the queue doesn't have requeuing enabled.
			assert.Eventually(t, func() bool {
				return sink.requestsCount.Load() == tt.expectedRequests &&
					sink.itemsCount.Load() == tt.expectedItems &&
					be.batchSender.(*batchSender).activeRequests.Load() == uint64(0) &&
					be.queueSender.(*queueSender).queueController.Size() == 0
			}, 100*time.Millisecond, 10*time.Millisecond)
		})
	}
}

func TestBatchSender_MergeOrSplit(t *testing.T) {
	mergeCfg := NewDefaultMergeBatcherConfig()
	mergeCfg.MinSizeItems = 5
	mergeCfg.Timeout = 100 * time.Millisecond
	be := queueBatchExporter(t, WithBatcher(mergeCfg, fakeBatchMergeFunc,
		WithSplitBatcher(SplitBatcherConfig{MaxSizeItems: 10}, fakeBatchMergeSplitFunc)))

	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, be.Shutdown(context.Background()))
	})

	sink := newFakeRequestSink()

	// should be sent right away by reaching the minimum items size.
	require.NoError(t, be.send(context.Background(), &fakeRequest{items: 8, sink: sink}))
	assert.Eventually(t, func() bool {
		return sink.requestsCount.Load() == 1 && sink.itemsCount.Load() == 8
	}, 50*time.Millisecond, 10*time.Millisecond)

	// big request should be broken down into two requests, both are sent right away.
	require.NoError(t, be.send(context.Background(), &fakeRequest{items: 17, sink: sink}))

	assert.Eventually(t, func() bool {
		return sink.requestsCount.Load() == 3 && sink.itemsCount.Load() == 25
	}, 50*time.Millisecond, 10*time.Millisecond)

	// request that cannot be split should be dropped.
	require.NoError(t, be.send(context.Background(), &fakeRequest{items: 11, sink: sink,
		mergeErr: errors.New("split error")}))

	// big request should be broken down into two requests, both are sent right away.
	require.NoError(t, be.send(context.Background(), &fakeRequest{items: 13, sink: sink}))

	assert.Eventually(t, func() bool {
		return sink.requestsCount.Load() == 5 && sink.itemsCount.Load() == 38
	}, 50*time.Millisecond, 10*time.Millisecond)

	fmt.Println("TestBatchSender_MergeOrSplit")
}

func TestBatchSender_Shutdown(t *testing.T) {
	batchCfg := NewDefaultMergeBatcherConfig()
	batchCfg.MinSizeItems = 10
	be := queueBatchExporter(t, WithBatcher(batchCfg, fakeBatchMergeFunc))

	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	sink := newFakeRequestSink()
	require.NoError(t, be.send(context.Background(), &fakeRequest{items: 3, sink: sink}))

	// To make the request reached the batchSender before shutdown.
	time.Sleep(50 * time.Millisecond)

	require.NoError(t, be.Shutdown(context.Background()))

	// shutdown should force sending the batch
	assert.Equal(t, uint64(1), sink.requestsCount.Load())
	assert.Equal(t, uint64(3), sink.itemsCount.Load())
}

func TestBatchSender_Disabled(t *testing.T) {
	mergeCfg := NewDefaultMergeBatcherConfig()
	mergeCfg.Enabled = false
	be, err := newBaseExporter(defaultSettings, "", true, nil, nil, newNoopObsrepSender,
		WithBatcher(mergeCfg, fakeBatchMergeFunc, WithSplitBatcher(SplitBatcherConfig{MaxSizeItems: 10}, fakeBatchMergeSplitFunc)))
	require.NotNil(t, be)
	require.NoError(t, err)

	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, be.Shutdown(context.Background()))
	})

	sink := newFakeRequestSink()
	// should be sent right away because batching is disabled.
	require.NoError(t, be.send(context.Background(), &fakeRequest{items: 8, sink: sink}))
	assert.Equal(t, uint64(1), sink.requestsCount.Load())
	assert.Equal(t, uint64(8), sink.itemsCount.Load())
}

func TestBatchSender_InvalidMergeSplitFunc(t *testing.T) {
	invalidMergeSplitFunc := func(_ context.Context, _ Request, req2 Request, _ int) ([]Request, error) {
		// reply with invalid 0 length slice if req2 is more than 20 items
		if req2.(*fakeRequest).items > 20 {
			return []Request{}, nil
		}
		// otherwise reply with a single request.
		return []Request{req2}, nil
	}
	mergeCfg := NewDefaultMergeBatcherConfig()
	mergeCfg.Timeout = 50 * time.Millisecond
	be := queueBatchExporter(t, WithBatcher(mergeCfg, fakeBatchMergeFunc,
		WithSplitBatcher(SplitBatcherConfig{MaxSizeItems: 20}, invalidMergeSplitFunc)))

	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, be.Shutdown(context.Background()))
	})

	sink := newFakeRequestSink()
	// first request should be ignored due to invalid merge/split function.
	require.NoError(t, be.send(context.Background(), &fakeRequest{items: 30, sink: sink}))
	// second request should be sent after reaching the timeout.
	require.NoError(t, be.send(context.Background(), &fakeRequest{items: 15, sink: sink}))
	assert.Eventually(t, func() bool {
		return sink.requestsCount.Load() == 1 && sink.itemsCount.Load() == 15
	}, 100*time.Millisecond, 10*time.Millisecond)
}

func TestBatchSender_PostShutdown(t *testing.T) {
	be, err := newBaseExporter(defaultSettings, "", true, nil, nil,
		newNoopObsrepSender, WithBatcher(NewDefaultMergeBatcherConfig(), fakeBatchMergeFunc))
	require.NotNil(t, be)
	require.NoError(t, err)
	assert.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, be.Shutdown(context.Background()))

	// Closed batch sender should act as a pass-through to not block queue draining.
	sink := newFakeRequestSink()
	require.NoError(t, be.send(context.Background(), &fakeRequest{items: 8, sink: sink}))
	assert.Equal(t, uint64(1), sink.requestsCount.Load())
	assert.Equal(t, uint64(8), sink.itemsCount.Load())
}

func TestBatchSender_ConcurrencyLimitReached(t *testing.T) {
	qCfg := NewDefaultQueueConfig()
	qCfg.NumConsumers = 2
	be, err := newBaseExporter(defaultSettings, "", true, nil, nil,
		newNoopObsrepSender, WithBatcher(NewDefaultMergeBatcherConfig(), fakeBatchMergeFunc), WithRequestQueue(qCfg, NewMemoryQueueFactory()))
	require.NotNil(t, be)
	require.NoError(t, err)
	assert.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	sink := newFakeRequestSink()
	assert.NoError(t, be.send(context.Background(), &fakeRequest{items: 8, sink: sink}))

	time.Sleep(50 * time.Millisecond)
	// the first request should be still in-flight.
	assert.Equal(t, uint64(0), sink.requestsCount.Load())

	// the second request should be sent by reaching max concurrency limit.
	assert.NoError(t, be.send(context.Background(), &fakeRequest{items: 8, sink: sink}))

	assert.Eventually(t, func() bool {
		return sink.requestsCount.Load() == 1 && sink.itemsCount.Load() == 16
	}, 100*time.Millisecond, 10*time.Millisecond)
}

func queueBatchExporter(t *testing.T, batchOption Option) *baseExporter {
	be, err := newBaseExporter(defaultSettings, "", true, nil, nil,
		newNoopObsrepSender, batchOption, WithRequestQueue(NewDefaultQueueConfig(), NewMemoryQueueFactory()))
	require.NotNil(t, be)
	require.NoError(t, err)
	return be
}
