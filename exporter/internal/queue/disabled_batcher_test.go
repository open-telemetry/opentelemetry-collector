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

func TestDisabledBatcher_InfiniteWorkerPool(t *testing.T) {
	cfg := exporterbatcher.NewDefaultConfig()
	cfg.Enabled = false

	q := NewBoundedMemoryQueue[internal.Request](
		MemoryQueueSettings[internal.Request]{
			Sizer:    &RequestSizer[internal.Request]{},
			Capacity: 10,
		})

	maxWorkers := 0
	ba := NewBatcher(cfg, q, maxWorkers)

	require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, q.Shutdown(context.Background()))
		require.NoError(t, ba.Shutdown(context.Background()))
	})

	sink := newFakeRequestSink()

	require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 8, sink: sink}))
	require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 8, exportErr: errors.New("transient error"), sink: sink}))
	assert.Eventually(t, func() bool {
		return sink.requestsCount.Load() == 1 && sink.itemsCount.Load() == 8
	}, 20*time.Millisecond, 10*time.Millisecond)

	require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 17, sink: sink}))
	assert.Eventually(t, func() bool {
		return sink.requestsCount.Load() == 2 && sink.itemsCount.Load() == 25
	}, 20*time.Millisecond, 10*time.Millisecond)

	require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 13, sink: sink}))

	assert.Eventually(t, func() bool {
		return sink.requestsCount.Load() == 3 && sink.itemsCount.Load() == 38
	}, 20*time.Millisecond, 10*time.Millisecond)
}
