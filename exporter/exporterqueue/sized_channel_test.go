// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterqueue

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sizerInt struct{}

func (s sizerInt) Sizeof(el int) int64 {
	return int64(el)
}

func TestSizedCapacityChannel(t *testing.T) {
	q := newSizedChannel[int](7, sizerInt{})
	require.NoError(t, q.push(1))
	assert.Equal(t, 1, q.Size())
	assert.Equal(t, 7, q.Capacity())

	require.NoError(t, q.push(3))
	assert.Equal(t, 4, q.Size())

	// should not be able to send to the full queue
	require.Error(t, q.push(4))
	assert.Equal(t, 4, q.Size())

	el, ok := q.pop()
	assert.Equal(t, 1, el)
	assert.True(t, ok)
	assert.Equal(t, 3, q.Size())

	el, ok = q.pop()
	assert.Equal(t, 3, el)
	assert.True(t, ok)
	assert.Equal(t, 0, q.Size())

	q.shutdown()
	el, ok = q.pop()
	assert.False(t, ok)
	assert.Equal(t, 0, el)
}

func TestSizedCapacityChannel_Offer_sizedNotFullButChannelFull(t *testing.T) {
	q := newSizedChannel[int](1, sizerInt{})
	require.NoError(t, q.push(1))

	q.used.Store(0)
	err := q.push(1)
	require.Error(t, err)
	assert.Equal(t, ErrQueueIsFull, err)
}
