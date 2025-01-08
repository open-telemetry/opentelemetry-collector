// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterqueue

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockingMemoryQueue(t *testing.T) {
	var wg sync.WaitGroup
	q := NewBlockingMemoryQueue[string](blockingMemoryQueueSettings[string]{Sizer: &requestSizer[string]{}, Capacity: 1})

	err := errors.New("This is an error")
	wg.Add(1)
	go func() {
		assert.EqualError(t, q.Offer(context.Background(), "a"), err.Error()) // Blocks until OnProcessingFinished is called
		wg.Done()
	}()

	index, ctx, req, ok := q.Read(context.Background())
	for !ok {
		index, ctx, req, ok = q.Read(context.Background())
	}

	require.Equal(t, uint64(0), index)
	require.Equal(t, context.Background(), ctx)
	require.Equal(t, "a", req)
	q.OnProcessingFinished(index, err)
	wg.Wait()
}
