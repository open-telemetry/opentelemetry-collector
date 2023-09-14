// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
)

func TestBatcherSingleShardMerge(t *testing.T) {
	batchCfg := NewDefaultBaseBatchConfig()
	batchCfg.MinSizeItems = 10
	batchCfg.Timeout = 100 * time.Millisecond

	tests := []struct {
		name string
		b    *Batcher
	}{
		{
			name: "merge_only_batcher",
			b:    NewMergeBatcher(batchCfg, fakeBatchMergeFunc),
		},
		{
			name: "merge_or_split_batcher",
			b:    NewMergeOrSplitBatcher(batchCfg, BatchConfigMaxItems{MaxSizeItems: 0}, fakeBatchMergeOrSplitFunc),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			be, err := newBaseExporter(defaultSettings, "", true, nil, nil, newNoopObsrepSender, WithBatcher(tt.b))
			require.NotNil(t, be)
			require.NoError(t, err)

			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, be.Shutdown(context.Background()))
			})

			sink := newFakeRequestSink()

			require.NoError(t, be.send(newRequest(context.Background(), &fakeRequest{items: 8, sink: sink})))
			require.NoError(t, be.send(newRequest(context.Background(), &fakeRequest{items: 3, sink: sink})))
			require.NoError(t, be.send(newRequest(context.Background(), &fakeRequest{items: 3, sink: sink})))
			require.NoError(t, be.send(newRequest(context.Background(), &fakeRequest{items: 3, sink: sink,
				mergeErr: errors.New("merge error")})))
			require.NoError(t, be.send(newRequest(context.Background(), &fakeRequest{items: 1, sink: sink})))

			// the first two requests should be merged into one and sent by reaching the minimum items size
			assert.Eventually(t, func() bool {
				return sink.requestsCount.Load() == 1 && sink.itemsCount.Load() == 11
			}, 50*time.Millisecond, 10*time.Millisecond)

			// the third and fifth requests should be sent by reaching the timeout
			// the fourth request should be ignored because of the merge error
			assert.Eventually(t, func() bool {
				return sink.requestsCount.Load() == 2 && sink.itemsCount.Load() == 15
			}, 150*time.Millisecond, 10*time.Millisecond)
		})
	}
}

func TestBatcherMultiShardMerge(t *testing.T) {
	batchCfg := NewDefaultBaseBatchConfig()
	batchCfg.MinSizeItems = 10
	batchCfg.Timeout = 100 * time.Millisecond
	batchIdentifierOpt := WithRequestBatchIdentifier(func(_ context.Context, req Request) string {
		return req.(*fakeRequest).batchID
	}, BatchConfigBatchersLimit{BatchersLimit: 2})

	tests := []struct {
		name string
		b    *Batcher
	}{
		{
			name: "merge_only_batcher",
			b:    NewMergeBatcher(batchCfg, fakeBatchMergeFunc, batchIdentifierOpt),
		},
		{
			name: "merge_or_split_batcher",
			b:    NewMergeOrSplitBatcher(batchCfg, BatchConfigMaxItems{MaxSizeItems: 0}, fakeBatchMergeOrSplitFunc, batchIdentifierOpt),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			be, err := newBaseExporter(defaultSettings, "", true, nil, nil, newNoopObsrepSender, WithBatcher(tt.b))
			require.NotNil(t, be)
			require.NoError(t, err)

			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, be.Shutdown(context.Background()))
			})

			sink := newFakeRequestSink()

			require.NoError(t, be.send(newRequest(context.Background(), &fakeRequest{items: 8, sink: sink,
				batchID: "1"}))) // batch 1
			require.NoError(t, be.send(newRequest(context.Background(), &fakeRequest{items: 3, sink: sink,
				batchID: "2"}))) // batch 2
			require.NoError(t, be.send(newRequest(context.Background(), &fakeRequest{items: 3, sink: sink,
				batchID: "1"}))) // batch 1

			// batch 1 should be sent by reaching the minimum items size
			assert.Eventually(t, func() bool {
				return sink.requestsCount.Load() == 1 && sink.itemsCount.Load() == 11
			}, 50*time.Millisecond, 10*time.Millisecond)

			require.NoError(t, be.send(newRequest(context.Background(), &fakeRequest{items: 9, sink: sink,
				batchID: "2"}))) // batch 2
			require.Error(t, be.send(newRequest(context.Background(), &fakeRequest{items: 3, sink: sink,
				batchID: "3"}))) // batchers limit reached
			require.NoError(t, be.send(newRequest(context.Background(), &fakeRequest{items: 2, sink: sink,
				batchID: "2"}))) // batch 3

			// batch 2 should be sent by reaching the minimum items size
			assert.Eventually(t, func() bool {
				return sink.requestsCount.Load() == 2 && sink.itemsCount.Load() == 23
			}, 50*time.Millisecond, 10*time.Millisecond)

			// batch 3 should be sent by reaching the timeout
			assert.Eventually(t, func() bool {
				return sink.requestsCount.Load() == 3 && sink.itemsCount.Load() == 25
			}, 150*time.Millisecond, 10*time.Millisecond)
		})
	}
}

func TestBatchSenderShutdown(t *testing.T) {
	batchCfg := NewDefaultBaseBatchConfig()
	batchCfg.MinSizeItems = 10
	b := NewMergeBatcher(batchCfg, fakeBatchMergeFunc)
	be, err := newBaseExporter(defaultSettings, "", true, nil, nil, newNoopObsrepSender, WithBatcher(b))
	require.NotNil(t, be)
	require.NoError(t, err)

	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	sink := newFakeRequestSink()
	require.NoError(t, be.send(newRequest(context.Background(), &fakeRequest{items: 3, sink: sink})))

	require.NoError(t, be.Shutdown(context.Background()))

	// shutdown should force sending the batch
	assert.Equal(t, int64(1), sink.requestsCount.Load())
	assert.Equal(t, int64(3), sink.itemsCount.Load())
}

func TestBatcherSingleShardMergeOrSplit(t *testing.T) {
	batchCfg := NewDefaultBaseBatchConfig()
	batchCfg.MinSizeItems = 5
	batchCfg.Timeout = 100 * time.Millisecond
	b := NewMergeOrSplitBatcher(batchCfg, BatchConfigMaxItems{MaxSizeItems: 10}, fakeBatchMergeOrSplitFunc)
	be, err := newBaseExporter(defaultSettings, "", true, nil, nil, newNoopObsrepSender, WithBatcher(b))
	require.NotNil(t, be)
	require.NoError(t, err)

	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, be.Shutdown(context.Background()))
	})

	sink := newFakeRequestSink()

	// should be sent right away by reaching the minimum items size.
	require.NoError(t, be.send(newRequest(context.Background(), &fakeRequest{items: 8, sink: sink})))

	// should be left in the queue because it doesn't reach the minimum items size.
	require.NoError(t, be.send(newRequest(context.Background(), &fakeRequest{items: 3, sink: sink})))

	assert.Eventually(t, func() bool {
		return sink.requestsCount.Load() == 1 && sink.itemsCount.Load() == 8
	}, 50*time.Millisecond, 10*time.Millisecond)

	// big request should be merged with the second request and broken down into two requests
	require.NoError(t, be.send(newRequest(context.Background(), &fakeRequest{items: 17, sink: sink})))

	assert.Eventually(t, func() bool {
		return sink.requestsCount.Load() == 3 && sink.itemsCount.Load() == 28
	}, 50*time.Millisecond, 10*time.Millisecond)

	// big request should be broken down into two requests, both are sent right away by reaching the minimum items size.
	require.NoError(t, be.send(newRequest(context.Background(), &fakeRequest{items: 17, sink: sink})))

	assert.Eventually(t, func() bool {
		return sink.requestsCount.Load() == 5 && sink.itemsCount.Load() == 45
	}, 50*time.Millisecond, 10*time.Millisecond)

	// invalid merge error should be ignored
	require.NoError(t, be.send(newRequest(context.Background(), &fakeRequest{items: 9, sink: sink,
		mergeErr: errors.New("merge error")})))

	// big request should be broken down into two requests, only the first one is sent right away by reaching the minimum items size.
	// the second one should be sent by reaching the timeout.
	require.NoError(t, be.send(newRequest(context.Background(), &fakeRequest{items: 13, sink: sink})))

	assert.Eventually(t, func() bool {
		return sink.requestsCount.Load() == 6 && sink.itemsCount.Load() == 55
	}, 50*time.Millisecond, 10*time.Millisecond)

	assert.Eventually(t, func() bool {
		return sink.requestsCount.Load() == 7 && sink.itemsCount.Load() == 58
	}, 150*time.Millisecond, 10*time.Millisecond)
}
