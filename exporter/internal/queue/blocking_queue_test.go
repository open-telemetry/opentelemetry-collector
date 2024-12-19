// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBlockingMemoryQueue(t *testing.T) {
	q := NewBlockingMemoryQueue[string](BlockingMemoryQueueSettings[string]{Sizer: &RequestSizer[string]{}, Capacity: 1})

	done := false
	err := errors.New("This is an error")
	go func() {
		require.EqualError(t, q.Offer(context.Background(), "a"), err.Error())
		done = true
	}()

	require.False(t, done)
	index, ctx, req, ok := q.Read(context.Background())
	require.Equal(t, index, uint64(0))
	require.Equal(t, ctx, context.Background())
	require.Equal(t, req, "a")
	require.True(t, ok)

	require.False(t, done)
	q.OnProcessingFinished(index, err)

	time.Sleep(100 * time.Millisecond)
	require.True(t, done)
}
