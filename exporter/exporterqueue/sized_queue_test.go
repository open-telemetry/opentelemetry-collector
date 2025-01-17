// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterqueue

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sizerInt struct{}

func (s sizerInt) Sizeof(el int) int64 {
	return int64(el)
}

func TestSizedQueue(t *testing.T) {
	q := newSizedQueue[int](7, sizerInt{}, false)
	require.NoError(t, q.Offer(context.Background(), 1))
	assert.Equal(t, int64(1), q.Size())
	assert.Equal(t, int64(7), q.Capacity())

	require.NoError(t, q.Offer(context.Background(), 3))
	assert.Equal(t, int64(4), q.Size())

	// should not be able to send to the full queue
	require.Error(t, q.Offer(context.Background(), 4))
	assert.Equal(t, int64(4), q.Size())

	_, el, ok := q.pop()
	assert.Equal(t, 1, el)
	assert.True(t, ok)
	assert.Equal(t, int64(3), q.Size())

	_, el, ok = q.pop()
	assert.Equal(t, 3, el)
	assert.True(t, ok)
	assert.Equal(t, int64(0), q.Size())

	require.NoError(t, q.Shutdown(context.Background()))
	_, el, ok = q.pop()
	assert.False(t, ok)
	assert.Equal(t, 0, el)
}

func TestSizedQueue_DrainAllElements(t *testing.T) {
	q := newSizedQueue[int](7, sizerInt{}, false)
	require.NoError(t, q.Offer(context.Background(), 1))
	require.NoError(t, q.Offer(context.Background(), 3))

	_, el, ok := q.pop()
	assert.Equal(t, 1, el)
	assert.True(t, ok)
	assert.Equal(t, int64(3), q.Size())

	require.NoError(t, q.Shutdown(context.Background()))
	_, el, ok = q.pop()
	assert.Equal(t, 3, el)
	assert.True(t, ok)
	assert.Equal(t, int64(0), q.Size())

	_, el, ok = q.pop()
	assert.False(t, ok)
	assert.Equal(t, 0, el)
}

func TestSizedChannel_OfferInvalidSize(t *testing.T) {
	q := newSizedQueue[int](1, sizerInt{}, false)
	require.ErrorIs(t, q.Offer(context.Background(), -1), errInvalidSize)
}

func TestSizedChannel_OfferZeroSize(t *testing.T) {
	q := newSizedQueue[int](1, sizerInt{}, false)
	require.NoError(t, q.Offer(context.Background(), 0))
	require.NoError(t, q.Shutdown(context.Background()))
	// Because the size 0 is ignored, nothing to drain.
	_, el, ok := q.pop()
	assert.False(t, ok)
	assert.Equal(t, 0, el)
}
